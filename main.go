package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"

	"github.com/twilight-project/forkoracle-go/address"
	"github.com/twilight-project/forkoracle-go/bridge"
	"github.com/twilight-project/forkoracle-go/comms"
	db "github.com/twilight-project/forkoracle-go/db"
	"github.com/twilight-project/forkoracle-go/eventhandler"
	"github.com/twilight-project/forkoracle-go/judge"
	"github.com/twilight-project/forkoracle-go/orchestrator"
	"github.com/twilight-project/forkoracle-go/servers"
	btcOracleTypes "github.com/twilight-project/forkoracle-go/types"
	utils "github.com/twilight-project/forkoracle-go/utils"
)

var wg sync.WaitGroup

func initialize() (string, string, *sql.DB) {
	utils.InitConfigFile()
	// btcPubkey := utils.GetBtcPublicKey()
	comms.GetAllFragments()
	dbconn := db.InitDB()
	valAddr := viper.GetString("own_validator_address")
	oracleAddr := viper.GetString("own_address")
	// validator := viper.GetBool("validator")

	allowed_modes := map[string]bool{
		"judge":  true,
		"signer": true,
		"":       true,
	}
	running_mode := fmt.Sprintf("%v", viper.GetString("running_mode"))
	if !allowed_modes[running_mode] {
		fmt.Println("Invalid running mode, has to be one of judge, signer or leave empty for just validator")
		panic("")
	}

	// btcPublicKey := viper.GetString("btc_xpublic_key")
	// if validator == true || running_mode == "judge" {
	// 	utils.SetDelegator(valAddr, oracleAddr, btcPublicKey)
	// }

	utils.LoadBtcWallet(viper.GetString("wallet_name"))
	utils.LoadBtcWallet(viper.GetString("fee_wallet_name"))

	return valAddr, oracleAddr, dbconn
}

func main() {
	valAddr, oracleAddr, dbconn := initialize()
	validator := viper.GetBool("validator")
	running_mode := viper.GetString("running_mode")
	// var upgrader = websocket.Upgrader{}
	var WsHub *btcOracleTypes.Hub

	var latestSweepTxHash = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "latest_sweep_tx_hash",
			Help: "Hash of the latest swept transaction.",
		},
		[]string{"hash"},
	)

	var latestRefundTxHash = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "latest_refund_tx_hash",
			Help: "Hash of the latest swept transaction.",
		},
		[]string{"hash"},
	)

	fmt.Println("valAddr : ", valAddr)
	fmt.Println("oracleAddr : ", oracleAddr)

	accountName := viper.GetString("accountName")
	fmt.Println("account name : ", accountName)
	var forkscanner_host = fmt.Sprintf("%v:%v", viper.Get("forkscanner_host"), viper.Get("forkscanner_ws_port"))
	forkscanner_url := url.URL{Scheme: "ws", Host: forkscanner_host, Path: "/"}

	time.Sleep(30 * time.Second)

	if running_mode == "judge" || validator == true {
		wg.Add(1)
		go orchestrator.Orchestrator(accountName, forkscanner_url, oracleAddr, &wg)
	}

	time.Sleep(1 * time.Minute)
	if running_mode == "judge" {
		wg.Add(1)
		go startJudge(accountName, dbconn, oracleAddr, valAddr, WsHub, latestRefundTxHash)
		time.Sleep(1 * time.Minute)
		wg.Add(1)
		go startBtcTxBroadcaster(dbconn)
	} else {
		time.Sleep(2 * time.Minute)
	}

	if running_mode == "judge" || validator == true {
		time.Sleep(1 * time.Minute)
		wg.Add(1)
		go startBridge(accountName, forkscanner_url, dbconn, latestSweepTxHash, oracleAddr, valAddr, WsHub)
	}
	// go servers.PubsubServer(WsHub, upgrader)

	if running_mode == "signer" {
		wg.Add(1)
		go startTransactionSigner(accountName, dbconn, oracleAddr, valAddr, WsHub)
	}

	if running_mode == "judge" {
		servers.Prometheus_server(latestSweepTxHash, latestRefundTxHash)
	}

	wg.Wait()
	fmt.Println("exiting main")
}

func startBtcTxBroadcaster(dbconn *sql.DB) {
	for {
		resp := comms.GetAttestations("1")
		if len(resp.Attestations) == 0 {
			time.Sleep(3 * time.Minute)
			continue
		}
		if resp.Attestations[0].Observed == false {
			time.Sleep(3 * time.Minute)
			continue
		}

		height, _ := strconv.ParseUint(resp.Attestations[0].Proposal.Height, 10, 64)
		signedTxs, err := db.QuerySignedSweeptx(dbconn, height)
		if err != nil {
			fmt.Println("error querying signed tx : ", err)
			continue
		}

		for _, t := range signedTxs {
			tx, err := utils.CreateTxFromHex(t.Tx)
			if err != nil {
				fmt.Println("error creating tx from hex : ", err)
				continue
			}
			fmt.Println("about to broadcast transaction")
			utils.BroadcastBtcTransaction(tx)
			db.DeleteSignedSweeptx(dbconn, height)
		}

		time.Sleep(3 * time.Minute)
	}
}

func startTransactionSigner(accountName string, dbconn *sql.DB, signerAddr string, valAddr string, WsHub *btcOracleTypes.Hub) {
	fmt.Println("starting Transaction Signer")
	defer wg.Done()
	judge_address := viper.GetString("judge_address")
	fragments := comms.GetAllFragments()
	var fragment btcOracleTypes.Fragment
	found := false
	for _, f := range fragments.Fragments {
		if f.JudgeAddress == judge_address {
			fragment = f
			found = true
			break
		}
	}
	if !found {
		panic("No fragment found with the specified judge address")
	}
	found = false
	for _, signer := range fragment.Signers {
		if signer.SignerAddress == signerAddr {
			found = true
		}
	}
	if !found {
		panic("Signer is not registered with the provided judge")
	}
	address.RegisterAddressOnSigners(dbconn)
	go eventhandler.NyksEventListener("unsigned_tx_refund", accountName, "signing_refund", dbconn, signerAddr, valAddr, WsHub, nil)
	go eventhandler.NyksEventListener("broadcast_tx_refund", accountName, "signing_sweep", dbconn, signerAddr, valAddr, WsHub, nil)
	validator := viper.GetBool("validator")
	if validator == false {
		eventhandler.NyksEventListener("propose_sweep_address", accountName, "register_res_addr_signers", dbconn, signerAddr, valAddr, WsHub, nil)
	}

	fmt.Println("finishing transaction signer")
}

func startJudge(accountName string, dbconn *sql.DB, judgeAddr string, valAddr string, WsHub *btcOracleTypes.Hub, latestRefundTxHash *prometheus.GaugeVec) {
	fmt.Println("starting judge")
	defer wg.Done()
	fragments := comms.GetAllFragments()
	var fragment btcOracleTypes.Fragment
	found := false
	for _, f := range fragments.Fragments {
		if f.JudgeAddress == judgeAddr {
			fragment = f
			found = true
			break
		}
	}
	if !found {
		panic("Judge has not registered a fragment with the nyks chain. Please ensure that the fragment is registered before running a Judge  Exiting...")
	}

	if len(fragment.ReserveIds) <= 0 {
		judge.InitReserve(accountName, judgeAddr, valAddr, dbconn)
	}

	go address.ProcessProposeAddress(accountName, judgeAddr, dbconn)
	// go judge.BroadcastOnBtc(dbconn)
	go eventhandler.NyksEventListener("propose_sweep_address", accountName, "sweep_process", dbconn, judgeAddr, valAddr, WsHub, nil)
	go eventhandler.NyksEventListener("broadcast_tx_refund", accountName, "signed_sweep_process", dbconn, judgeAddr, valAddr, WsHub, latestRefundTxHash)
	go eventhandler.NyksEventListener("unsigned_tx_sweep", accountName, "refund_process", dbconn, judgeAddr, valAddr, WsHub, nil)
	eventhandler.NyksEventListener("unsigned_tx_refund", accountName, "signed_refund_process", dbconn, judgeAddr, valAddr, WsHub, nil)
}

func startBridge(accountName string, forkscanner_url url.URL, dbconn *sql.DB, latestSweepTxHash *prometheus.GaugeVec, oracleAddr string, valAddr string, WsHub *btcOracleTypes.Hub) {
	fmt.Println("starting bridge")
	defer wg.Done()
	address.RegisterAddressOnValidators(dbconn)
	go eventhandler.NyksEventListener("propose_sweep_address", accountName, "register_res_addr_validators", dbconn, oracleAddr, valAddr, WsHub, nil)
	go bridge.WatchAddress(forkscanner_url, dbconn)
	bridge.KDeepService(accountName, dbconn, latestSweepTxHash, oracleAddr)
	fmt.Println("finishing bridge")
}
