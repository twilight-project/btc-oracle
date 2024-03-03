package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"github.com/tyler-smith/go-bip32"

	"github.com/gorilla/websocket"
)

var dbconn *sql.DB
var masterPrivateKey *bip32.Key
var judge bool
var oracleAddr string
var valAddr string
var upgrader = websocket.Upgrader{}
var WsHub *Hub

var latestSweepTxHash = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "latest_sweep_tx_hash",
		Help: "Hash of the latest swept transaction.",
	},
	[]string{"hash"},
)

func initialize() {
	initConfigFile()
	btcPubkey := initWallet()
	dbconn = initDB()
	setDelegator(btcPubkey)
}

func main() {

	initialize()

	accountName := fmt.Sprintf("%v", viper.Get("accountName"))
	fmt.Println("account name : ", accountName)
	var forkscanner_host = fmt.Sprintf("%v:%v", viper.Get("forkscanner_host"), viper.Get("forkscanner_ws_port"))
	forkscanner_url := url.URL{Scheme: "ws", Host: forkscanner_host, Path: "/"}
	if accountName == "validator-sfo" || accountName == "validator-ams" || accountName == "validator-stg" {
		judge = true
	}

	time.Sleep(30 * time.Second)

	go orchestrator(accountName, forkscanner_url)

	initJudge(accountName)

	time.Sleep(1 * time.Minute)
	if judge {
		go startJudge(accountName)
	} else {
		time.Sleep(2 * time.Minute)
	}

	time.Sleep(1 * time.Minute)
	go startBridge(accountName, forkscanner_url)
	go pubsubServer()
	go startTransactionSigner(accountName)
	prometheus_server()
	fmt.Println("exiting main")
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.writePump()
}

func pubsubServer() {
	fmt.Println("starting pubsub server")
	WsHub = &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}

	go WsHub.run()

	http.HandleFunc("/tapinscription", func(w http.ResponseWriter, r *http.Request) {
		serveWs(WsHub, w, r)
	})

	err := http.ListenAndServe(":2300", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func prometheus_server() {
	// Create a new instance of a registry
	reg := prometheus.NewRegistry()

	// Optional: Add Go module build info.
	reg.MustRegister(
		latestSweepTxHash,
	)

	// Register the promhttp handler with the registry
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// Simple health check endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Server is running"))
	})

	// Start the server
	log.Println("Starting prometheus server on :2555")
	if err := http.ListenAndServe(":2555", nil); err != nil {
		log.Fatalf("Error starting server: %s", err)
	}
}

// func main() {

// 	initialize()
// 	// accountName := fmt.Sprintf("%v", viper.Get("accountName"))
// 	sweeptx := getUnsignedSweepTx(1, 1)
// 	tx := sweeptx.UnsignedTxSweepMsg.BtcUnsignedSweepTx
// 	sweepTx, _ := createTxFromHex(tx)

// 	signatureSweep := getSignSweep(1, 1)
// 	x := sweepTx.TxIn[0].Witness[0]
// 	hx := hex.EncodeToString(x)
// 	decodedscript := decodeBtcScript(hx)
// 	min := getMinSignFromScript(decodedscript)
// 	pubkeys := getPublicKeysFromScript(decodedscript, int(min))

// 	t := filterAndOrderSignSweep(signatureSweep, pubkeys)

// 	fmt.Println(t)
// }
