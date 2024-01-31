package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"github.com/twilight-project/nyks/x/bridge/types"
	bridgetypes "github.com/twilight-project/nyks/x/bridge/types"
)

func watchAddress(url url.URL) {
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

		c := WatchtowerResponse{}
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
			insertNotifications(notification)
		}

	}

}

func kDeepService(accountName string) {
	fmt.Println("running k deep service")
	for {
		resp := getAttestations("5")
		if len(resp.Attestations) > 0 {
			fmt.Println("INFO : k deep : latest attestaion is : ", resp.Attestations[0])
			for _, attestation := range resp.Attestations {
				if attestation.Observed {
					fmt.Println("attestaion is Observed : ", attestation.Proposal.Height)
					height, err := strconv.ParseUint(attestation.Proposal.Height, 10, 64)
					if err != nil {
						fmt.Println(err)
					}
					kDeepCheck(accountName, uint64(height))
					break
				}
			}

		}
		time.Sleep(5 * time.Minute)
	}
}

func kDeepCheck(accountName string, height uint64) {
	fmt.Println("running k deep check for height : ", height)
	addresses := queryNotification()
	watchedTx := queryWatchedTransactions()
	number := fmt.Sprintf("%v", viper.Get("confirmation_limit"))
	confirmations, _ := strconv.ParseUint(number, 10, 64)
	for _, a := range addresses {
		if height-a.Height >= confirmations {
			fmt.Println("reached height confirmations: ", height)
			confirmBtcTransactionOnNyks(accountName, a)

			for _, tx := range watchedTx {
				if a.Receiving_txid == tx.Txid {

					var currentReservesForThisJudge []BtcReserve
					reserves := getBtcReserves()
					for _, reserve := range reserves.BtcReserves {
						if reserve.JudgeAddress == oracleAddr {
							currentReservesForThisJudge = append(currentReservesForThisJudge, reserve)
						}
					}

					var reserveTobeProcessed BtcReserve
					minRoundId := 500
					// Iterate through the array and find the minimum roundId
					for _, reserve := range currentReservesForThisJudge {
						tempRoundId, _ := strconv.Atoi(reserve.RoundId)
						if tempRoundId < minRoundId {
							minRoundId = tempRoundId
							reserveTobeProcessed = reserve
						}
					}

					roundId, _ := strconv.Atoi(reserveTobeProcessed.RoundId)
					reserveId, _ := strconv.Atoi(reserveTobeProcessed.ReserveId)

					cosmos := getCosmosClient()
					msg := &bridgetypes.MsgSweepProposal{
						ReserveId:             uint64(reserveId),
						NewReserveAddress:     tx.Address,
						JudgeAddress:          oracleAddr,
						BtcRelayCapacityValue: 0,
						BtcTxHash:             tx.Txid,
						UnlockHeight:          0,
						RoundId:               uint64(roundId),
						BtcBlockNumber:        0,
					}
					fmt.Println("Sending Sweep proposal message")
					sendTransactionSweepProposal(accountName, cosmos, msg)
					markTransactionProcessed(tx.Txid)

				}
			}
		}
	}
	fmt.Println("finishing k deep check for height : ", height)
}

func confirmBtcTransactionOnNyks(accountName string, data WatchtowerNotification) {
	fmt.Println("inside confirm btc transaction")
	cosmos := getCosmosClient()

	depositAddresses := getAllDepositAddress()
	var depositAddress []DepositAddress
	for _, deposit := range depositAddresses.Addresses {
		if deposit.BtcDepositAddress == data.Sending {
			fmt.Println("inside equal address check")
			depositAddress = append(depositAddress, deposit)
			break
		}
	}

	if len(depositAddress) == 0 {
		fmt.Println("zero addresses bridge")
		markProcessedNotifications(data)
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
	sendTransactionConfirmBtcdeposit(accountName, cosmos, msg)
	fmt.Println("deleting notifiction after procesing")
	markProcessedNotifications(data)

}

func startBridge(accountName string, forkscanner_url url.URL) {
	fmt.Println("starting bridge")
	registerAddressOnValidators()
	go nyksEventListener("propose_sweep_address", accountName, "register_res_addr_validators")
	go watchAddress(forkscanner_url)
	kDeepService(accountName)
	fmt.Println("finishing bridge")
}
