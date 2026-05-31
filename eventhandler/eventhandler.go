package eventhandler

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"github.com/twilight-project/forkoracle-go/address"
	"github.com/twilight-project/forkoracle-go/judge"
	"github.com/twilight-project/forkoracle-go/transaction_signer"
	btcOracleTypes "github.com/twilight-project/forkoracle-go/types"
)

func NyksEventListener(event string, accountName string, functionCall string, dbconn *sql.DB,
	oracleAddr string, valAddr string, WsHub *btcOracleTypes.Hub, latestRefundTxHash *prometheus.GaugeVec) {

	rpcAddr := viper.GetString("nyksd_rpc_url")
	if rpcAddr == "" {
		rpcAddr = "http://127.0.0.1:26657"
	}
	query := fmt.Sprintf("tm.event='Tx' AND message.action='%s'", event)

	for {
		client, err := rpchttp.New(rpcAddr, "/websocket")
		if err != nil {
			fmt.Println("nyks event listener create client:", err)
			time.Sleep(10 * time.Second)
			continue
		}

		if err := client.Start(); err != nil {
			fmt.Println("nyks event listener start:", err)
			time.Sleep(10 * time.Second)
			continue
		}

		eventCh, err := client.Subscribe(context.Background(), "nyks-oracle", query)
		if err != nil {
			fmt.Println("nyks event listener subscribe:", err)
			client.Stop() //nolint:errcheck
			time.Sleep(10 * time.Second)
			continue
		}

		for range eventCh {
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

		fmt.Println("nyks event listener disconnected, reconnecting...")
		client.Stop() //nolint:errcheck
		time.Sleep(10 * time.Second)
	}
}
