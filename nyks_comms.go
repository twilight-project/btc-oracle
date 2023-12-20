package main

// contains all code communicating with nyksd chain (cosmos)
import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	"github.com/spf13/viper"
	bridgetypes "github.com/twilight-project/nyks/x/bridge/types"
	forktypes "github.com/twilight-project/nyks/x/forks/types"
)

func sendTransactionSweepAddressProposal(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgProposeSweepAddress) {

	_, err := cosmos.BroadcastTx(accountName, data)
	if err != nil {
		fmt.Println("error in sending sweep address proposal : ", err)
	} else {
		fmt.Println("Sweep Address propose Transaction sent")
	}
}

func sendTransactionSeenBtcChainTip(accountName string, cosmos cosmosclient.Client, data *forktypes.MsgSeenBtcChainTip) {
	_, err := cosmos.BroadcastTx(accountName, data)
	if err != nil {
		fmt.Println("error in chaintip trnasaction : ", err)
	} else {
		fmt.Println("sent Seen Chaintip transaction")
	}
}

func sendTransactionConfirmBtcdeposit(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgConfirmBtcDeposit) {

	_, err := cosmos.BroadcastTx(accountName, data)
	if err != nil {
		fmt.Println("error in confirm deposit transaction : ", err)
	} else {
		fmt.Println("btc deposit confirmation sent")
	}
}

func sendTransactionSweepProposal(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgSweepProposal) {

	_, err := cosmos.BroadcastTx(accountName, data)
	if err != nil {
		fmt.Println("error in sending sweep transaction proposal : ", err)
	} else {
		fmt.Println("Sweep Transaction sent")
	}
}

func sendTransactionUnsignedSweepTx(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgUnsignedTxSweep) {

	_, err := cosmos.BroadcastTx(accountName, data)
	if err != nil {
		fmt.Println("error in sending unsigned sweep transaction : ", err)
	} else {
		fmt.Println("unsigned Sweep Transaction sent")
	}
}

func sendTransactionUnsignedRefundTx(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgUnsignedTxRefund) {

	_, err := cosmos.BroadcastTx(accountName, data)
	if err != nil {
		fmt.Println("error in sending unsigned Refund transaction : ", err)
	} else {
		fmt.Println("unsigned Refund Transaction sent")
	}
}

func sendTransactionRegisterJudge(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgRegisterJudge) {

	_, err := cosmos.BroadcastTx(accountName, data)
	if err != nil {
		fmt.Println("error in sending register judge transaction : ", err)
	}
}

func sendTransactionSignSweep(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgSignSweep) {

	_, err := cosmos.BroadcastTx(accountName, data)
	if err == nil {
		fmt.Println("Sweep Signature sent")
	} else {
		fmt.Println("Error in sending sweep signature : {}", err)
	}
}

func sendTransactionSignRefund(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgSignRefund) {

	_, err := cosmos.BroadcastTx(accountName, data)
	if err == nil {
		fmt.Println("Refund Signature sent")
	} else {
		fmt.Println("Error in sending refund signature : {}", err)
	}
}

func sendTransactionBroadcastSweeptx(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgBroadcastTxSweep) {

	_, err := cosmos.BroadcastTx(accountName, data)
	if err != nil {
		fmt.Println("error in Boradcasting Sweep Tx transaction : ", err)
	}
}

func sendTransactionBroadcastRefundtx(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgBroadcastTxRefund) {

	_, err := cosmos.BroadcastTx(accountName, data)
	if err != nil {
		fmt.Println("error in Boradcasting Sweep Tx transaction : ", err)
	}
}

func getCosmosClient() cosmosclient.Client {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	homePath := filepath.Join(home, ".nyks")

	cosmosOptions := []cosmosclient.Option{
		cosmosclient.WithHome(homePath),
	}

	config := sdktypes.GetConfig()
	config.SetBech32PrefixForAccount("twilight", "twilight"+"pub")

	// create an instance of cosmosclient
	cosmos, err := cosmosclient.New(context.Background(), cosmosOptions...)
	if err != nil {
		log.Fatal(err)
	}

	return cosmos
}

func getCosmosAddress(accountName string, cosmos cosmosclient.Client) sdktypes.AccAddress {
	address, err := cosmos.Address(accountName)
	if err != nil {
		log.Fatal(err)
	}
	return address
}

func getDepositAddresses() QueryDepositAddressResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/registered_btc_deposit_addresses")
	if err != nil {
		fmt.Println("error getting deposit addresses : ", err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting deposit addresses body : ", err)
	}

	a := QueryDepositAddressResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling deposit addresses : ", err)
	}
	return a
}

func getDepositAddress(address string) DepositAddress {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/registered_btc_deposit_address/" + address)
	if err != nil {
		fmt.Println("error getting deposit addresses : ", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting deposit addresses body : ", err)
	}
	a := DepositAddress{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling deposit addresses : ", err)
	}
	return a
}

func getAllDepositAddress() QueryDepositAddressResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/registered_btc_deposit_addresses")
	if err != nil {
		fmt.Println("error getting deposit addresses : ", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting deposit addresses body : ", err)
	}
	a := QueryDepositAddressResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling deposit addresses : ", err)
	}
	return a
}

func getAttestations(limit string) NyksAttestaionBlock {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	req_url := fmt.Sprintf("%s/twilight-project/nyks/nyks/attestations?limit=%s&order_by=desc", nyksd_url, limit)
	resp, err := http.Get(req_url)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	a := NyksAttestaionBlock{}
	err = json.Unmarshal(body, &a)

	return a
}

// func getAttestationsSweepProposal() AttestaionBlockSweep {
// 	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
// 	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/nyks/attestations?limit=20&order_by=desc&proposal_type=2")
// 	if err != nil {
// 		log.Fatalln(err)
// 	}
// 	//We Read the response body on the line below.
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		log.Fatalln(err)
// 	}

// 	a := AttestaionBlockSweep{}
// 	err = json.Unmarshal(body, &a)

// 	return a
// }

func getUnsignedSweepTx(reserveId uint64, roundId uint64) UnsignedTxSweepResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/unsigned_tx_sweep/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	a := UnsignedTxSweepResp{}
	err = json.Unmarshal(body, &a)

	return a
}

func getAllUnsignedSweepTx() UnsignedTxSweepResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/unsigned_tx_sweep_all?limit=5")
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	a := UnsignedTxSweepResp{}
	err = json.Unmarshal(body, &a)

	return a
}

func getUnsignedRefundTx(reserveId int64, roundId int64) UnsignedTxRefundResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/unsigned_tx_refund/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	a := UnsignedTxRefundResp{}
	err = json.Unmarshal(body, &a)

	return a
}

func getAllUnsignedRefundTx() UnsignedTxRefundResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/unsigned_tx_refund_all?limit=5")
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	a := UnsignedTxRefundResp{}
	err = json.Unmarshal(body, &a)

	return a
}

func getDelegateAddresses() DelegateAddressesResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/forks/delegate_keys_all")
	if err != nil {
		fmt.Println("error getting delegate addresses : ", err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading delegate addresses : ", err)
	}

	a := DelegateAddressesResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling delegate addresses : ", err)
	}
	return a
}

func getSignSweep(reserveId uint64, roundId uint64) MsgSignSweepResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/sign_sweep/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		fmt.Println("error getting sign sweep : ", err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting sign sweep body : ", err)
	}

	a := MsgSignSweepResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshallin sign sweep : ", err)
	}
	return a
}

func getSignRefund(reserveId uint64, roundId uint64) MsgSignRefundResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/sign_refund/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		fmt.Println("error getting refund signature : ", err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting refund signature body : ", err)
	}

	a := MsgSignRefundResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling refund signature : ", err)
	}
	return a
}

func getReserveddresses() ReserveAddressResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/registered_reserve_addresses")
	if err != nil {
		fmt.Println("error getting reserve addresses : ", err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting reserve addresses body : ", err)
	}

	a := ReserveAddressResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling reserve addresses : ", err)
	}
	return a
}

func getRegisteredJudges() RegisteredJudgeResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/registered_judges")
	if err != nil {
		fmt.Println("error getting registered judges : ", err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting registered judges body : ", err)
	}

	a := RegisteredJudgeResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling registered judges : ", err)
	}
	return a
}

func getBtcReserves() BtcReserveResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/volt/btc_reserve")
	if err != nil {
		fmt.Println("error getting btcreserves : ", err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting btcreserves body : ", err)
	}

	a := BtcReserveResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling  btcreserves: ", err)
	}
	return a
}

func getProposedSweepAddresses() ProposedAddressesResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/propose_sweep_addresses_all/24")
	if err != nil {
		fmt.Println("error getting proposed address : ", err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting proposed address body : ", err)
	}

	a := ProposedAddressesResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling proposed address : ", err)
	}
	return a
}

func getProposedSweepAddress(reserveId uint64, roundId uint64) ProposedAddressResp {
	// nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/propose_sweep_address/%d/%d", reserveId, roundId)
	resp, err := http.Get("https://nyks.twilight-explorer.com/api" + path)
	if err != nil {
		fmt.Println("error getting proposed address : ", err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting proposed address body : ", err)
	}

	a := ProposedAddressResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling proposed address : ", err)
	}
	return a
}

func getRefundSnapshot(reserveId uint64, roundId uint64) RefundTxSnapshot {
	// nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/volt/refund_tx_snapshot/%d/%d", reserveId, roundId)
	resp, err := http.Get("https://nyks.twilight-explorer.com/api" + path)
	if err != nil {
		fmt.Println("error getting refund snapshot : ", err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting refund snapshot body : ", err)
	}

	a := RefundTxSnapshotResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling refund snapshot : ", err)
	}
	return a.RefundTxSnapshot
}

func getWithdrawSnapshot(reserveId uint64, roundId uint64) ReserveWithdrawSnapshot {
	// nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/volt/reserve_withdraw_snapshot/%d/%d", reserveId, roundId)
	resp, err := http.Get("https://nyks.twilight-explorer.com/api" + path)
	if err != nil {
		fmt.Println("error getting withdraw snapshot : ", err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting withdraw snapshot body : ", err)
	}

	a := ReserveWithdrawSnapshotResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling withdraw snapshot : ", err)
	}
	return a.ReserveWithdrawSnapshot
}

func getBroadCastedRefundTx(reserveId uint64, roundId uint64) BroadcastRefundMsg {
	// nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/broadcast_tx_refund/%d/%d", reserveId, roundId)
	resp, err := http.Get("https://nyks.twilight-explorer.com/api" + path)
	if err != nil {
		fmt.Println("error getting broadcasted refund : ", err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting broadcasted refund body : ", err)
	}

	a := BroadcastRefundMsgResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling broadcasted refund : ", err)
	}
	return a.BroadcastRefundMsg
}
