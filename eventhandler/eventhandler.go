package eventhandler

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"github.com/twilight-project/forkoracle-go/address"
	"github.com/twilight-project/forkoracle-go/judge"
	"github.com/twilight-project/forkoracle-go/transaction_signer"
	btcOracleTypes "github.com/twilight-project/forkoracle-go/types"
)

func NyksEventListener(event string, accountName string, functionCall string, dbconn *sql.DB,
	oracleAddr string, valAddr string, WsHub *btcOracleTypes.Hub, latestRefundTxHash *prometheus.GaugeVec) {
	headers := make(map[string][]string)
	headers["Content-Type"] = []string{"application/json"}
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_socket_url"))

	for {
		conn, _, err := websocket.DefaultDialer.Dial(nyksd_url, headers)
		if err != nil {
			fmt.Println("nyks event listener dial:", err)
			time.Sleep(10 * time.Second)
			continue
		}

		// Set up ping/pong connection health check
		pingPeriod := 30 * time.Second
		pongWait := 60 * time.Second
		stopChan := make(chan struct{})

		conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		go func() {
			ticker := time.NewTicker(pingPeriod)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						return
					}
				case <-stopChan:
					return
				}
			}
		}()

		payload := fmt.Sprintf(`{
        "jsonrpc": "2.0",
        "method": "subscribe",
        "id": 0,
        "params": {
            "query": "tm.event='Tx' AND message.action='%s'"
        }
    }`, event)

		if err = conn.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
			fmt.Println("error in nyks event handler: ", err)
			close(stopChan)
			conn.Close()
			time.Sleep(10 * time.Second)
			continue
		}

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("error in nyks event handler: ", err)
				close(stopChan)
				conn.Close()
				break
			}

			switch functionCall {
			case "signed_sweep_process":
				go judge.ProcessSignedSweep(accountName, oracleAddr, dbconn)
			case "refund_process":
				go judge.ProcessRefund(accountName, oracleAddr, dbconn)
			case "signed_refund_process":
				go judge.ProcessSignedRefund(accountName, oracleAddr, dbconn, WsHub, latestRefundTxHash)
			case "register_res_addr_validators":
				go address.RegisterAddressOnValidators(dbconn)
			case "register_res_addr_signers":
				go address.RegisterAddressOnSigners(dbconn)
			case "signing_sweep":
				go transaction_signer.ProcessTxSigningSweep(accountName, dbconn, oracleAddr)
			case "signing_refund":
				go transaction_signer.ProcessTxSigningRefund(accountName, dbconn, oracleAddr)
			case "sweep_process":
				go judge.ProcessSweep(accountName, dbconn, oracleAddr)
			default:
				log.Println("Unknown function :", functionCall)
			}
		}

		time.Sleep(10 * time.Second)
	}
}
