package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/twilight-project/nyks/x/bridge/types"
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
		//save in DB
		fmt.Printf("recv watchtower noti: %s", message)

		c := WatchtowerResponse{}
		err = json.Unmarshal(message, &c)
		if err != nil {
			fmt.Println("error in address watcher: ", err)
			continue
		}

		watchtower_notifications := c.Params
		resp := getDepositAddresses()

		for _, address := range resp.Addresses {
			for _, notification := range watchtower_notifications {
				if address.DepositAddress == notification.Sending {
					insertNotifications(notification)
				}
			}
		}

	}

}

func kDeepService(accountName string, url url.URL) {
	for {
		resp := getAttestations()
		if len(resp.Attestations) > 0 {
			attestation := resp.Attestations[0]
			if attestation.Observed == true {
				height, err := strconv.ParseUint(attestation.Proposal.Height, 10, 64)
				if err != nil {
					fmt.Println(err)
				}
				kDeepCheck(accountName, uint64(height))
			}

		}
		time.Sleep(5 * time.Minute)
	}
}

func kDeepCheck(accountName string, height uint64) {
	addresses := queryNotification()
	for _, a := range addresses {
		if height-a.Height > 3 {
			time.Sleep(1 * time.Minute)
			confirmBtcTransactionOnNyks(accountName, a)
		}
	}
}

func confirmBtcTransactionOnNyks(accountName string, data WatchtowerNotification) {
	cosmos := getCosmosClient()
	oracle_address := getCosmosAddress(accountName, cosmos)

	deposit_addresses := getDepositAddresses()
	for _, a := range deposit_addresses.Addresses {
		if a.DepositAddress == data.Sending {
			msg := &types.MsgConfirmBtcDeposit{
				DepositAddress:         data.Receiving,
				DepositAmount:          data.Satoshis,
				Height:                 data.Height,
				Hash:                   data.Receiving_txid,
				TwilightDepositAddress: a.TwilightDepositAddress,
				BtcOracleAddress:       oracle_address.String(),
			}

			sendTransactionConfirmBtcdeposit(accountName, cosmos, msg)
			markProcessedNotifications(data)
			fmt.Println("sent confirm btc transaction")
		}
	}

}

func startBridge(accountName string, forkscanner_url url.URL) {

	go watchAddress(forkscanner_url)
	go kDeepService(accountName, forkscanner_url)

}
