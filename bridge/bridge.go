package bridge

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"

	comms "github.com/twilight-project/forkoracle-go/comms"
	db "github.com/twilight-project/forkoracle-go/db"
	btcOracleTypes "github.com/twilight-project/forkoracle-go/types"
	"github.com/twilight-project/nyks/x/bridge/types"
	bridgetypes "github.com/twilight-project/nyks/x/bridge/types"
)

func WatchAddress(url url.URL, dbconn *sql.DB) {
	conn, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	defer conn.Close()

	payload := `{
		"jsonrpc": "2.0",
		"id": "watched_address_checks",
		"method": "watched_address_checks",
		"params": {
			"watch": [],
			"watch_until": "2999-09-30T00:00:00.0Z"
		}
	}`

	err = conn.WriteMessage(websocket.TextMessage, []byte(payload))
	if err != nil {
		log.Println("error in address watcher: ", err)
		return
	}

	fmt.Println("registered on address watcher")

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("error in address watcher: ", err)
			return
		}

		fmt.Printf("recv watchtower noti: %s", message)

		c := btcOracleTypes.WatchtowerResponse{}
		err = json.Unmarshal(message, &c)
		if err != nil {
			fmt.Println("error in address watcher: ", err)
			continue
		}

		if len(c.Params) <= 0 {
			fmt.Println("first message from forkscanner, confirming subscription")
			continue
		}

		watchtower_notifications := c.Params
		for _, notification := range watchtower_notifications {
			db.InsertNotifications(dbconn, notification)
		}

	}

}

func KDeepService(accountName string, dbconn *sql.DB, latestSweepTxHash *prometheus.GaugeVec, oracleAddr string) {
	fmt.Println("running k deep service")
	for {
		resp := comms.GetAttestations("5")
		if len(resp.Attestations) > 0 {
			fmt.Println("INFO : k deep : latest attestaion is : ", resp.Attestations[0])
			for _, attestation := range resp.Attestations {
				if attestation.Observed {
					fmt.Println("attestaion is Observed : ", attestation.Proposal.Height)
					height, err := strconv.ParseUint(attestation.Proposal.Height, 10, 64)
					if err != nil {
						fmt.Println(err)
					}
					kDeepCheck(accountName, uint64(height), dbconn, latestSweepTxHash, oracleAddr)
					break
				}
			}

		}
		time.Sleep(5 * time.Minute)
	}
}

func kDeepCheck(accountName string, height uint64, dbconn *sql.DB, latestSweepTxHash *prometheus.GaugeVec, oracleAddr string) {
	fmt.Println("running k deep check for height : ", height)
	notifications := db.QueryNotification(dbconn)
	watchedTx := db.QueryWatchedTransactions(dbconn)
	number := fmt.Sprintf("%v", viper.Get("confirmation_limit"))
	confirmations, _ := strconv.ParseUint(number, 10, 64)
	for _, n := range notifications {
		if height-n.Height >= confirmations {
			fmt.Println("reached height confirmations: ", height)
			confirmBtcTransactionOnNyks(accountName, n, dbconn, oracleAddr)
		}

		for _, tx := range watchedTx {
			if height-n.Height <= confirmations {
				continue
			}
			if n.Sending == tx.Address {

				addresses := db.QuerySweepAddress(dbconn, tx.Address)
				if len(addresses) <= 0 {
					fmt.Println("no record of this address")
					return
				}

				reserves := comms.GetBtcReserves()
				var reserve btcOracleTypes.BtcReserve
				for _, res := range reserves.BtcReserves {
					if res.ReserveId == strconv.Itoa(int(tx.Reserve)) {
						reserve = res
						break
					}
				}
				var emptyReserve btcOracleTypes.BtcReserve
				if reserve == emptyReserve {
					fmt.Println("Reserve not found : ", tx.Reserve)
					continue
				}

				cosmos := comms.GetCosmosClient()
				fmt.Println("==============proposal message =============")
				fmt.Println(tx.Reserve)
				fmt.Println(n.Receiving)
				fmt.Println(reserve.JudgeAddress)
				fmt.Println(n.Receiving_txid)
				fmt.Println(n.Height)
				fmt.Println(tx.Round)
				fmt.Println(accountName)
				var uheight uint64
				addrs := db.QuerySweepAddress(dbconn, n.Receiving)
				if len(addrs) <= 0 {
					fmt.Println("address not found in DB")
					uheight = n.Height
				} else {
					uheight = uint64(addrs[0].Unlock_height)
				}

				msg := &bridgetypes.MsgSweepProposal{
					ReserveId:             uint64(tx.Reserve),
					NewReserveAddress:     n.Receiving,
					JudgeAddress:          reserve.JudgeAddress,
					BtcRelayCapacityValue: 0,
					BtcTxHash:             n.Receiving_txid,
					UnlockHeight:          uheight,
					RoundId:               uint64(tx.Round),
					BtcBlockNumber:        uint64(n.Height),
					OracleAddress:         oracleAddr,
				}
				fmt.Println("Sending Sweep proposal message")
				comms.SendTransactionSweepProposal(accountName, cosmos, msg)
				db.MarkTransactionProcessed(dbconn, tx.Txid)
				db.MarkProcessedNotifications(dbconn, n)
				latestSweepTxHash.Reset()
				latestSweepTxHash.WithLabelValues(tx.Txid).Set(float64(tx.Reserve))

			}
		}
	}
	fmt.Println("finishing k deep check for height : ", height)
}

func confirmBtcTransactionOnNyks(accountName string, data btcOracleTypes.WatchtowerNotification, dbconn *sql.DB, oracleAddr string) {
	fmt.Println("inside confirm btc transaction")
	cosmos := comms.GetCosmosClient()

	depositAddresses := comms.GetAllDepositAddress()
	var depositAddress []btcOracleTypes.DepositAddress
	for _, deposit := range depositAddresses.Addresses {
		if deposit.BtcDepositAddress == data.Sending {
			fmt.Println("inside equal address check")
			depositAddress = append(depositAddress, deposit)
			break
		}
	}

	if len(depositAddress) == 0 {
		fmt.Println("zero addresses bridge")
		// db.MarkProcessedNotifications(dbconn, data)
		return
	}

	msg := &types.MsgConfirmBtcDeposit{
		ReserveAddress:         data.Receiving,
		DepositAmount:          data.Satoshis,
		Height:                 data.Height,
		Hash:                   data.Receiving_txid,
		TwilightDepositAddress: depositAddress[0].TwilightAddress,
		OracleAddress:          oracleAddr,
	}
	fmt.Println("confirming btc transaction")
	comms.SendTransactionConfirmBtcdeposit(accountName, cosmos, msg)
	fmt.Println("deleting notifiction after procesing")
	db.MarkProcessedNotifications(dbconn, data)
}

//nyksd tx bridge msg-confirm-btc-deposit 14uEN8abvKA1zgYCpv8MWCUwAMLGBqdZGM 100000000000 789656 000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f $(nyksd keys show validator-stg -a --keyring-backend test) --from validator-stg --chain-id nyks --keyring-backend test
