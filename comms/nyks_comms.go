package comms

// contains all code communicating with nyksd chain (cosmos)
import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite-hq/cli/ignite/pkg/cosmosclient"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	"github.com/spf13/viper"
	btcOracleTypes "github.com/twilight-project/forkoracle-go/types"
	bridgetypes "github.com/twilight-project/nyks/x/bridge/types"
	forktypes "github.com/twilight-project/nyks/x/forks/types"
)

func SendTransactionSweepAddressProposal(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgProposeSweepAddress) {
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

func SendTransactionRegisterReserveAddress(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgRegisterReserveAddress) (cosmosclient.Response, error) {
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

func SendTransactionSeenBtcChainTip(accountName string, cosmos cosmosclient.Client, data *forktypes.MsgSeenBtcChainTip) {
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

func SendTransactionConfirmBtcdeposit(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgConfirmBtcDeposit) {
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

func SendTransactionSweepProposal(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgSweepProposal) {
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

func SendTransactionUnsignedSweepTx(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgUnsignedTxSweep) {
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

func SendTransactionUnsignedRefundTx(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgUnsignedTxRefund) {
	var err error
	for i := 0; i < 5; i++ {
		_, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			fmt.Println("unsigned Refund Transaction sent")
			return
		}
		if strings.Contains(err.Error(), "duplicate") {
			fmt.Println("duplicate error in refund transaction, not retrying... : ", err)
			break
		} else {
			fmt.Println("error in sending unsigned Refund transaction, retrying... : ", err)
			time.Sleep(10 * time.Second)
		}
	}
	fmt.Println("error in sending unsigned Refund transaction after 5 attempts: ", err)
}

// func SendTransactionRegisterJudge(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgRegisterJudge) {
// 	var err error
// 	for i := 0; i < 5; i++ {
// 		_, err = cosmos.BroadcastTx(accountName, data)
// 		if err == nil {
// 			return
// 		}
// 		fmt.Println("error in sending register judge transaction, retrying... : ", err)
// 		time.Sleep(10 * time.Second)
// 	}
// 	fmt.Println("error in sending register judge transaction after 5 attempts: ", err)
// }

func SendTransactionSignSweep(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgSignSweep) {
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

func SendTransactionSignRefund(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgSignRefund) {
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

func SendTransactionBroadcastSweeptx(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgBroadcastTxSweep) {
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

func SendTransactionBroadcastRefundtx(accountName string, cosmos cosmosclient.Client, data *bridgetypes.MsgBroadcastTxRefund) {
	var err error
	for i := 0; i < 5; i++ {
		_, err = cosmos.BroadcastTx(accountName, data)
		if err == nil {
			return
		}
		if strings.Contains(err.Error(), "duplicate") {
			fmt.Println("duplicate error in Broadcasting refund transaction, not retrying... : ", err)
			break
		} else {
			fmt.Println("error in Broadcasting Refund Tx transaction, retrying... : ", err)
			time.Sleep(10 * time.Second)
		}
	}
	fmt.Println("error in Broadcasting Refund Tx transaction after 5 attempts: ", err)
}

func GetCosmosClient() cosmosclient.Client {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	homePath := filepath.Join(home, ".nyks")

	cosmosOptions := []cosmosclient.Option{
		cosmosclient.WithHome(homePath),
		cosmosclient.WithKeyringBackend("file"),
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
// 	body, err := io.ReadAll(resp.Body)
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
// 	body, err := io.ReadAll(resp.Body)
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

func GetAllDepositAddress() btcOracleTypes.QueryDepositAddressResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/registered_btc_deposit_addresses")
	if err != nil {
		fmt.Println("error getting deposit addresses : ", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting deposit addresses body : ", err)
	}
	a := btcOracleTypes.QueryDepositAddressResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling deposit addresses : ", err)
	}
	return a
}

func GetAttestations(limit string) btcOracleTypes.NyksAttestaionBlock {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	req_url := fmt.Sprintf("%s/twilight-project/nyks/nyks/attestations?limit=%s&order_by=desc", nyksd_url, limit)
	resp, err := http.Get(req_url)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	a := btcOracleTypes.NyksAttestaionBlock{}
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
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		log.Fatalln(err)
// 	}

// 	a := AttestaionBlockSweep{}
// 	err = json.Unmarshal(body, &a)

// 	return a
// }

func GetAllFragments() btcOracleTypes.Fragments {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/volt/get_all_fragments")
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	a := btcOracleTypes.Fragments{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error getting Fragments : ", err)
	}

	return a
}

func GetUnsignedSweepTx(reserveId uint64, roundId uint64) btcOracleTypes.UnsignedTxSweepResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/unsigned_tx_sweep/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	a := btcOracleTypes.UnsignedTxSweepResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error getting unsigned sweep tx : ", err)
	}

	return a
}

func GetAllUnsignedSweepTx() btcOracleTypes.UnsignedTxSweepResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := "/twilight-project/nyks/bridge/unsigned_tx_sweep_all?limit=5"
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	a := btcOracleTypes.UnsignedTxSweepResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error getting all unsigned sweep tx : ", err)
	}

	return a
}

func GetUnsignedRefundTx(reserveId int64, roundId int64) btcOracleTypes.UnsignedTxRefundResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/unsigned_tx_refund/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	a := btcOracleTypes.UnsignedTxRefundResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error getting unsigned refund tx : ", err)
	}

	return a
}

func GetAllUnsignedRefundTx() btcOracleTypes.UnsignedTxRefundResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/unsigned_tx_refund_all?limit=5")
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	a := btcOracleTypes.UnsignedTxRefundResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error getting all unsigned refund tx : ", err)
	}

	return a
}

func GetDelegateAddresses() btcOracleTypes.DelegateAddressesResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/forks/delegate_keys_all")
	if err != nil {
		fmt.Println("error getting delegate addresses : ", err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading delegate addresses : ", err)
	}

	a := btcOracleTypes.DelegateAddressesResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling delegate addresses : ", err)
	}
	return a
}

func GetSignSweep(reserveId uint64, roundId uint64) btcOracleTypes.MsgSignSweepResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/sign_sweep/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		fmt.Println("error getting sign sweep : ", err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting sign sweep body : ", err)
	}

	a := btcOracleTypes.MsgSignSweepResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshallin sign sweep : ", err)
	}
	return a
}

func GetSignRefund(reserveId uint64, roundId uint64) btcOracleTypes.MsgSignRefundResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/sign_refund/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		fmt.Println("error getting refund signature : ", err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting refund signature body : ", err)
	}

	a := btcOracleTypes.MsgSignRefundResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling refund signature : ", err)
	}
	return a
}

func GetReserveAddresses() btcOracleTypes.ReserveAddressResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/registered_reserve_addresses")
	if err != nil {
		fmt.Println("error getting reserve addresses : ", err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting reserve addresses body : ", err)
	}

	a := btcOracleTypes.ReserveAddressResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling reserve addresses : ", err)
	}
	return a
}

// func GetRegisteredJudges() btcOracleTypes.RegisteredJudgeResp {
// 	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
// 	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/registered_judges")
// 	if err != nil {
// 		fmt.Println("error getting registered judges : ", err)
// 	}
// 	//We Read the response body on the line below.
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		fmt.Println("error getting registered judges body : ", err)
// 	}

// 	a := btcOracleTypes.RegisteredJudgeResp{}
// 	err = json.Unmarshal(body, &a)
// 	if err != nil {
// 		fmt.Println("error unmarshalling registered judges : ", err)
// 	}
// 	return a
// }

func GetBtcReserves() btcOracleTypes.BtcReserveResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/volt/btc_reserve")
	if err != nil {
		fmt.Println("error getting btcreserves : ", err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting btcreserves body : ", err)
	}

	a := btcOracleTypes.BtcReserveResp{}
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
// 	body, err := io.ReadAll(resp.Body)
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

func GetProposedSweepAddress(reserveId uint64, roundId uint64) btcOracleTypes.ProposedAddressResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/propose_sweep_address/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		fmt.Println("error getting proposed address : ", err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting proposed address body : ", err)
	}

	a := btcOracleTypes.ProposedAddressResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling proposed address : ", err)
	}
	return a
}

func GetRefundSnapshot(reserveId uint64, roundId uint64) btcOracleTypes.RefundTxSnapshot {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/volt/refund_tx_snapshot/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		fmt.Println("error getting refund snapshot : ", err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting refund snapshot body : ", err)
	}

	a := btcOracleTypes.RefundTxSnapshotResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling refund snapshot : ", err)
	}
	return a.RefundTxSnapshot
}

func GetWithdrawSnapshot(reserveId uint64, roundId uint64) btcOracleTypes.ReserveWithdrawSnapshot {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/volt/reserve_withdraw_snapshot/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		fmt.Println("error getting withdraw snapshot : ", err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting withdraw snapshot body : ", err)
	}

	a := btcOracleTypes.ReserveWithdrawSnapshotResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling withdraw snapshot : ", err)
	}
	return a.ReserveWithdrawSnapshot
}

func GetBroadCastedRefundTx(reserveId uint64, roundId uint64) btcOracleTypes.BroadcastRefundMsg {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	path := fmt.Sprintf("/twilight-project/nyks/bridge/broadcast_tx_refund/%d/%d", reserveId, roundId)
	resp, err := http.Get(nyksd_url + path)
	if err != nil {
		fmt.Println("error getting broadcasted refund : ", err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting broadcasted refund body : ", err)
	}

	a := btcOracleTypes.BroadcastRefundMsgResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling broadcasted refund : ", err)
	}
	return a.BroadcastRefundMsg
}

func GetProposedAddresses() btcOracleTypes.ProposeSweepAddressMsgResp {
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_url"))
	resp, err := http.Get(nyksd_url + "/twilight-project/nyks/bridge/propose_sweep_addresses_all/25")
	if err != nil {
		fmt.Println("error getting reserve addresses : ", err)
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error getting reserve addresses body : ", err)
	}

	a := btcOracleTypes.ProposeSweepAddressMsgResp{}
	err = json.Unmarshal(body, &a)
	if err != nil {
		fmt.Println("error unmarshalling reserve addresses : ", err)
	}
	return a
}
