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
	"strings"
	"time"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	"github.com/spf13/viper"
	bridgetypes "github.com/twilight-project/nyks/x/bridge/types"
	forktypes "github.com/twilight-project/nyks/x/forks/types"
)

func sendTransactionSweepAddressProposal(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgProposeSweepAddress) {
	var err error
	for i := 0; i < 5; i++ {
		_, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			fmt.Println("Sweep Address propose Transaction sent")
			return
		}
		fmt.Println("error in sending sweep address proposal, retrying... : ", err)
		time.Sleep(10 * time.Second)
	}
	fmt.Println("error in sending sweep address proposal after 5 attempts: ", err)
}

func sendTransactionRegisterReserveAddress(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgRegisterReserveAddress) (cosmosclient.Response, error) {
	var err error
	var resp cosmosclient.Response
	for i := 0; i < 5; i++ {
		resp, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			fmt.Println("Register Reserve Transaction sent")
			return resp, nil
		}
		fmt.Println("error in sending Register Reserve, retrying... : ", err)
		time.Sleep(10 * time.Second)
	}
	fmt.Println("error in sending Register Reserve after 5 attempts: ", err)
	return resp, err
}

func sendTransactionSeenBtcChainTip(accountName string, cosmos cosmosclient.Client, data *forktypes.MsgSeenBtcChainTip) {
	var err error
	for i := 0; i < 5; i++ {
		_, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			fmt.Println("sent Seen Chaintip transaction")
			return
		}
		if strings.Contains(err.Error(), "Duplicate vote") {
			fmt.Println("duplicate error in chaintip transaction, not retrying... : ", err)
			break
		}
		fmt.Println("error in chaintip transaction, retrying... : ", err)
		time.Sleep(10 * time.Second)
	}
	fmt.Println("error in chaintip transaction after 5 attempts: ", err)
}

func sendTransactionConfirmBtcdeposit(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgConfirmBtcDeposit) {
	var err error
	for i := 0; i < 5; i++ {
		_, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			fmt.Println("btc deposit confirmation sent")
			return
		}
		fmt.Println("error in confirm deposit transaction, retrying... : ", err)
		time.Sleep(10 * time.Second)
	}
	fmt.Println("error in confirm deposit transaction after 5 attempts: ", err)
}

func sendTransactionSweepProposal(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgSweepProposal) {
	var err error
	for i := 0; i < 5; i++ {
		_, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			fmt.Println("Sweep proposal Transaction sent")
			return
		}
		fmt.Println("error in sending sweep transaction proposal, retrying... : ", err)
		time.Sleep(10 * time.Second)
	}
	fmt.Println("error in sending sweep transaction proposal after 5 attempts: ", err)
}

func sendTransactionUnsignedSweepTx(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgUnsignedTxSweep) {
	var err error
	for i := 0; i < 5; i++ {
		_, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			fmt.Println("unsigned Sweep Transaction sent")
			return
		}
		fmt.Println("error in sending unsigned sweep transaction, retrying... : ", err)
		time.Sleep(10 * time.Second)
	}
	fmt.Println("error in sending unsigned sweep transaction after 5 attempts: ", err)
}

func sendTransactionUnsignedRefundTx(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgUnsignedTxRefund) {
	var err error
	for i := 0; i < 5; i++ {
		_, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			fmt.Println("unsigned Refund Transaction sent")
			return
		}
		fmt.Println("error in sending unsigned Refund transaction, retrying... : ", err)
		time.Sleep(10 * time.Second)
	}
	fmt.Println("error in sending unsigned Refund transaction after 5 attempts: ", err)
}

func sendTransactionRegisterJudge(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgRegisterJudge) {
	var err error
	for i := 0; i < 5; i++ {
		_, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			return
		}
		fmt.Println("error in sending register judge transaction, retrying... : ", err)
		time.Sleep(10 * time.Second)
	}
	fmt.Println("error in sending register judge transaction after 5 attempts: ", err)
}

func sendTransactionSignSweep(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgSignSweep) {
	var err error
	for i := 0; i < 5; i++ {
		_, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			fmt.Println("Sweep Signature sent")
			return
		}
		fmt.Println("Error in sending sweep signature, retrying... : ", err)
		time.Sleep(10 * time.Second)
	}
	fmt.Println("Error in sending sweep signature after 5 attempts: ", err)
}

func sendTransactionSignRefund(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgSignRefund) {
	var err error
	for i := 0; i < 5; i++ {
		_, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			fmt.Println("Refund Signature sent")
			return
		}
		fmt.Println("Error in sending refund signature, retrying... : ", err)
		time.Sleep(10 * time.Second)
	}
	fmt.Println("Error in sending refund signature after 5 attempts: ", err)
}

func sendTransactionBroadcastSweeptx(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgBroadcastTxSweep) {
	var err error
	for i := 0; i < 5; i++ {
		_, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			return
		}
		fmt.Println("error in Broadcasting Sweep Tx transaction, retrying... : ", err)
		time.Sleep(10 * time.Second)
	}
	fmt.Println("error in Broadcasting Sweep Tx transaction after 5 attempts: ", err)
}

func sendTransactionBroadcastRefundtx(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgBroadcastTxRefund) {
	var err error
	for i := 0; i < 5; i++ {
		_, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			return
		}
		fmt.Println("error in Broadcasting Refund Tx transaction, retrying... : ", err)
		time.Sleep(10 * time.Second)
	}
	fmt.Println("error in Broadcasting Refund Tx transaction after 5 attempts: ", err)
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

func GetCurrentSequence(accountName string, cosmos cosmosclient.Client) (uint64, error) {
	accAddr := getCosmosAddress(accountName, cosmos)

	accRetriever := cosmos.Context().AccountRetriever
	_, seq, err := accRetriever.GetAccountNumberSequence(cosmos.Context(), accAddr)
	if err != nil {
		return 0, err
	}

	return seq, nil
}

// func getAccountSequence(cosmos cosmosclient.Client, address string) (uint64, error) {
// 	accAddress, err := sdk.AccAddressFromBech32(address)
// 	if err != nil {
// 		return 0, fmt.Errorf("invalid account address: %v", err)
// 	}

// 	// Get the account keeper from the Cosmos client
// 	accKeeper := cosmos.App().AccountKeeper()

// 	// Retrieve the account using the account keeper
// 	account := accKeeper.GetAccount(cosmos.Context(), accAddress)
// 	if account == nil {
// 		return 0, fmt.Errorf("account %s does not exist", accAddress)
// 	}

// 	return account.GetSequence(), nil
// }

func getCosmosAddress(accountName string, cosmos cosmosclient.Client) sdktypes.AccAddress {
	address, err := cosmos.Address(accountName)
	if err != nil {
		log.Fatal(err)
	}
	return address
}

// func getDepositAddresses() QueryDepositAddressResp {
// 	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
// 	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/registered_btc_deposit_addresses")
// 	if err != nil {
// 		fmt.Println("error getting deposit addresses : ", err)
// 	}
// 	//We Read the response body on the line below.
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		fmt.Println("error getting deposit addresses body : ", err)
// 	}

// 	a := QueryDepositAddressResp{}
// 	err = json.Unmarshal(body, &a)
// 	if err != nil {
// 		fmt.Println("error unmarshalling deposit addresses : ", err)
// 	}
// 	return a
// }

// func getDepositAddress(address string) DepositAddress {
// 	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
// 	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/registered_btc_deposit_address/" + address)
// 	if err != nil {
// 		fmt.Println("error getting deposit addresses : ", err)
// 	}
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		fmt.Println("error getting deposit addresses body : ", err)
// 	}
// 	a := DepositAddress{}
// 	err = json.Unmarshal(body, &a)
// 	if err != nil {
// 		fmt.Println("error unmarshalling deposit addresses : ", err)
// 	}
// 	return a
// }

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
	if err != nil {
		fmt.Println("error getting attestations : ", err)
	}

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
	if err != nil {
		fmt.Println("error getting unsigned sweep tx : ", err)
	}

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
	if err != nil {
		fmt.Println("error getting all unsigned sweep tx : ", err)
	}

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
	if err != nil {
		fmt.Println("error getting unsigned refund tx : ", err)
	}

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
	if err != nil {
		fmt.Println("error getting all unsigned refund tx : ", err)
	}

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

func getReserveAddresses() ReserveAddressResp {
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

// func getProposedSweepAddresses() ProposedAddressesResp {
// 	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
// 	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/propose_sweep_addresses_all/24")
// 	if err != nil {
// 		fmt.Println("error getting proposed address : ", err)
// 	}
// 	//We Read the response body on the line below.
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		fmt.Println("error getting proposed address body : ", err)
// 	}

// 	a := ProposedAddressesResp{}
// 	err = json.Unmarshal(body, &a)
// 	if err != nil {
// 		fmt.Println("error unmarshalling proposed address : ", err)
// 	}
// 	return a
// }

func getProposedSweepAddress(reserveId uint64, roundId uint64) ProposedAddressResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/propose_sweep_address/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
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
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/volt/refund_tx_snapshot/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
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
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/volt/reserve_withdraw_snapshot/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
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
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/broadcast_tx_refund/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
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

func getProposedAddresses() ProposeSweepAddressMsgResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/propose_sweep_addresses_all/25")
	if err != nil {
		fmt.Println("error getting reserve addresses : ", err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting reserve addresses body : ", err)
	}

	a := ProposeSweepAddressMsgResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling reserve addresses : ", err)
	}
	return a
}
