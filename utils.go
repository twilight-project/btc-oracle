package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/spf13/viper"
	bridgetypes "github.com/twilight-project/nyks/x/bridge/types"
)

func initConfigFile() {
	viper.AddConfigPath("/testnet/btc-oracle/configs")
	viper.SetConfigName("config") // Register config file name (no extension)
	viper.SetConfigType("json")   // Look for specific type
	viper.ReadInConfig()
}

func setDelegator(btcPubkey string) {
	accountName := fmt.Sprintf("%v", viper.Get("accountName"))
	command := fmt.Sprintf("nyksd keys show %s --bech val -a --keyring-backend test", accountName)
	args := strings.Fields(command)
	cmd := exec.Command(args[0], args[1:]...)

	valAddr_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	valAddr = string(valAddr_)
	valAddr = strings.ReplaceAll(valAddr, "\n", "")
	fmt.Println("Val Address : ", valAddr)

	command = fmt.Sprintf("nyksd keys show %s -a --keyring-backend test", accountName)
	args = strings.Fields(command)
	cmd = exec.Command(args[0], args[1:]...)

	oracleAddr_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	oracleAddr = string(oracleAddr_)
	oracleAddr = strings.ReplaceAll(oracleAddr, "\n", "")
	fmt.Println("Oracle Address : ", oracleAddr)

	command = fmt.Sprintf("nyksd tx nyks set-delegate-addresses %s %s %s --from %s --chain-id nyks --keyring-backend test -y", valAddr, oracleAddr, btcPubkey, accountName)
	fmt.Println("delegate command : ", command)

	args = strings.Fields(command)
	cmd = exec.Command(args[0], args[1:]...)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		panic(err)
	}

	fmt.Println("Delegate Address Set")
}

func getBitcoinRpcClient() *rpcclient.Client {
	connCfg := &rpcclient.ConnConfig{
		Host:         "143.244.138.170:8332",
		User:         "bitcoin",
		Pass:         "Persario_1",
		HTTPPostMode: true,
		DisableTLS:   true,
	}

	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		fmt.Println("Failed to connect to the Bitcoin client : ", err)
	}

	return client
}

func broadcastBtcTransaction(tx *wire.MsgTx) {
	client := getBitcoinRpcClient()
	txHash, err := client.SendRawTransaction(tx, true)
	if err != nil {
		fmt.Println("Failed to broadcast transaction : ", err)
	}

	defer client.Shutdown()
	fmt.Println("broadcasted btc transaction, txhash : ", txHash)
}

func registerAddressOnValidators() {
	// {add check to see if the address already exists}
	savedAddress := queryAllAddressOnly()
	resp := getReserveddresses()
	if len(resp.Addresses) > 0 {
		for _, address := range resp.Addresses {
			if stringInSlice(address.ReserveAddress, savedAddress) == false {
				registerAddressOnForkscanner(address.ReserveAddress)
				reserveScript, _ := hex.DecodeString(address.ReserveScript)
				insertSweepAddress(address.ReserveAddress, reserveScript, nil, 0, "")
			}
		}
	}
}

func getReserveForAddress(address string) BtcReserve {
	btcReserves := getBtcReserves()
	for _, reserve := range btcReserves.BtcReserves {
		if reserve.ReserveAddress == address {
			return reserve
		}
	}
	return BtcReserve{}
}

func registerReserveAddressOnNyks(accountName string, address string, script []byte) {

	cosmos := getCosmosClient()

	reserveScript := hex.EncodeToString(script)

	msg := &bridgetypes.MsgRegisterReserveAddress{
		ReserveScript:  reserveScript,
		ReserveAddress: address,
		JudgeAddress:   oracleAddr,
	}

	// store response in txResp
	txResp, err := cosmos.BroadcastTx(accountName, msg)
	if err != nil {
		fmt.Println("error in registering reserve address : ", err)
	}

	// print response from broadcasting a transaction
	fmt.Println("MsgRegisterReserveAddress : ")
	fmt.Println(txResp)
}

func registerAddressOnForkscanner(address string) {
	dt := time.Now().UTC()
	dt = dt.AddDate(1, 0, 0)

	request_body := map[string]interface{}{
		"method":  "add_watched_addresses",
		"id":      1,
		"jsonrpc": "2.0",
		"params": map[string]interface{}{
			"add": []interface{}{
				map[string]string{
					"address":     address,
					"watch_until": dt.Format(time.RFC3339),
				},
			},
		},
	}

	data, err := json.Marshal(request_body)
	if err != nil {
		log.Fatalf("Post: %v", err)
	}
	fmt.Println(string(data))

	resp, err := http.Post("http://0.0.0.0:8339", "application/json", strings.NewReader(string(data)))
	if err != nil {
		log.Fatalf("Post: %v", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("ReadAll: %v", err)
	}
	result := make(map[string]interface{})
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	log.Println(result)

}

func createTxFromHex(txHex string) (*wire.MsgTx, error) {
	// Decode the transaction hex string
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string: %v", err)
	}

	// Create a new transaction object
	tx := wire.NewMsgTx(wire.TxVersion)

	// Deserialize the transaction bytes
	err = tx.Deserialize(bytes.NewReader(txBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %v", err)
	}

	return tx, nil
}

func signTx(tx *wire.MsgTx, script []byte) []string {
	signatures := []string{}

	for i, input := range tx.TxIn {

		amount := queryAmount(input.PreviousOutPoint.Index, input.PreviousOutPoint.Hash.String())
		sighashes := txscript.NewTxSigHashes(tx)

		privkeybytes, err := masterPrivateKey.Serialize()
		if err != nil {
			fmt.Println("Error: converting private key to bytes : ", err)
		}

		privkey, _ := btcec.PrivKeyFromBytes(btcec.S256(), privkeybytes)

		signature, err := txscript.RawTxInWitnessSignature(tx, sighashes, i, int64(amount), script, txscript.SigHashAll, privkey)
		if err != nil {
			fmt.Println("Error:", err)
		}

		hexSignature := hex.EncodeToString(signature)

		signatures = append(signatures, hexSignature)
	}

	return signatures
}

// func signTx(tx *wire.MsgTx, address string) []byte {
// 	amount := queryAmount(tx.TxIn[0].PreviousOutPoint.Index, tx.TxIn[0].PreviousOutPoint.Hash.String())
// 	sighashes := txscript.NewTxSigHashes(tx)
// 	script := querySweepAddressScriptByParentAddress(address)

// 	privkeybytes, err := masterPrivateKey.Serialize()
// 	if err != nil {
// 		fmt.Println("Error: converting private key to bytes : ", err)
// 	}

// 	privkey, _ := btcec.PrivKeyFromBytes(btcec.S256(), privkeybytes)

// 	signature, err := txscript.RawTxInWitnessSignature(tx, sighashes, 0, int64(amount), script, txscript.SigHashAll|txscript.SigHashAnyOneCanPay, privkey)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 	}

// 	return signature
// }

func reverseArray(arr []MsgSignSweep) []MsgSignSweep {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}

func stringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func CreateTxOut(addr string, amount int64) (*wire.TxOut, error) {
	// Decode the Bitcoin address.
	address, err := btcutil.DecodeAddress(addr, &chaincfg.MainNetParams)
	if err != nil {
		fmt.Println("Error decoding address:", err)
		return nil, err
	}

	// Generate the pay-to-address script.
	destinationAddrByte, err := txscript.PayToAddrScript(address)
	if err != nil {
		fmt.Println("Error generating pay-to-address script:", err)
		return nil, err
	}
	TxOut := wire.NewTxOut(amount, destinationAddrByte)
	return TxOut, nil

}

func CreateTxIn(utxo Utxo) (*wire.TxIn, error) {
	utxoHash, err := chainhash.NewHashFromStr(utxo.Txid)
	if err != nil {
		log.Println("error with UTXO")
		return nil, err
	}
	outPoint := wire.NewOutPoint(utxoHash, utxo.Vout)
	txIn := wire.NewTxIn(outPoint, nil, nil)
	return txIn, nil
}

func sendUnsignedSweepTx(reserveId uint64, roundId uint64, sweepTx string, sweeptxId string, accountName string) {
	cosmos := getCosmosClient()
	msg := bridgetypes.NewMsgUnsignedTxSweep(sweeptxId, sweepTx, reserveId, roundId, oracleAddr)
	sendTransactionUnsignedSweepTx(accountName, cosmos, msg)
}

func sendUnsignedRefundTx(refundTx string, reserveId uint64, roundId uint64, accountName string) {
	cosmos := getCosmosClient()
	msg := bridgetypes.NewMsgUnsignedTxRefund(reserveId, roundId, refundTx, oracleAddr)
	sendTransactionUnsignedRefundTx(accountName, cosmos, msg)
}

// func sendUnsignedSweepTx(sweepTx string, sweeptxId string, oldReserveAddress string, withdrawals []BtcWithdrawRequest, accountName string, total uint64) {

// 	fmt.Println("inside sending sweep proposal")

// 	twilightIndividualAccounts := make([]*volttypes.IndividualTwilightReserveAccount, 0)
// 	for _, withdrawal := range withdrawals {
// 		amount, _ := strconv.Atoi(withdrawal.WithdrawAmount)
// 		individualAccount := volttypes.IndividualTwilightReserveAccount{
// 			TwilightAddress: withdrawal.WithdrawAddress,
// 			BtcValue:        uint64(amount),
// 		}
// 		twilightIndividualAccounts = append(twilightIndividualAccounts, &individualAccount)
// 	}

// 	cosmos := getCosmosClient()

// 	addrs := querySweepAddressByParentAddress(oldReserveAddress)
// 	if len(addrs) <= 0 {
// 		fmt.Println("New Resevre address not found hence not sending sweep proposal")
// 		return
// 	}

// 	newReserveAddress := addrs[0]

// 	msg := &bridgetypes.MsgSweepProposal{

// 		ReserveId:                        1,
// 		ReserveAddress:                   oldReserveAddress,
// 		JudgeAddress:                     oracleAddr,
// 		BtcRelayCapacityValue:            0,
// 		TotalValue:                       total,
// 		PrivatePoolValue:                 0,
// 		PublicValue:                      0,
// 		FeePool:                          0,
// 		IndividualTwilightReserveAccount: twilightIndividualAccounts,
// 		BtcRefundTx:                      refundTx + "++" + newReserveAddress.Address, // change to refund tx
// 		BtcSweepTx:                       sweepTx,
// 	}

// 	sendTransactionSweepProposal(accountName, cosmos, msg)
// }

func sendSweepSign(hexSignatures []string, address string, accountName string, reserveId uint64, roundId uint64) {
	cosmos := getCosmosClient()
	msg := &bridgetypes.MsgSignSweep{
		ReserveId:        reserveId,
		RoundId:          roundId,
		SignerPublicKey:  getBtcPublicKey(), // no idea what this is
		SweepSignature:   hexSignatures,
		BtcOracleAddress: oracleAddr,
	}

	sendTransactionSignSweep(accountName, cosmos, msg)
}

func sendRefundSign(hexSignatures string, address string, accountName string, reserveId uint64, roundId uint64) {
	cosmos := getCosmosClient()
	msg := &bridgetypes.MsgSignRefund{
		ReserveId:        reserveId,
		RoundId:          roundId,
		SignerPublicKey:  getBtcPublicKey(), // no idea what this is
		RefundSignature:  hexSignatures,
		BtcOracleAddress: oracleAddr,
	}

	sendTransactionSignRefund(accountName, cosmos, msg)
}

func broadcastSweeptxNYKS(sweepTxHex string, accountName string, reserveId uint64, roundId uint64) {
	cosmos := getCosmosClient()
	msg := &bridgetypes.MsgBroadcastTxSweep{
		SignedSweepTx: sweepTxHex,
		JudgeAddress:  oracleAddr,
		ReserveId:     reserveId,
		RoundId:       roundId,
	}

	sendTransactionBroadcastSweeptx(accountName, cosmos, msg)
}

func broadcastRefundtxNYKS(refundTxHex string, accountName string, reserveId uint64, roundId uint64) {
	cosmos := getCosmosClient()
	msg := &bridgetypes.MsgBroadcastTxRefund{
		SignedRefundTx: refundTxHex,
		JudgeAddress:   oracleAddr,
		ReserveId:      reserveId,
		RoundId:        roundId,
	}

	sendTransactionBroadcastRefundtx(accountName, cosmos, msg)
}

func generateAndRegisterNewBtcReserveAddress(accountName string, height int64) string {
	newSweepAddress, reserveScript := generateAddress(height, "")
	registerReserveAddressOnNyks(accountName, newSweepAddress, reserveScript)
	registerAddressOnForkscanner(newSweepAddress)

	BtcReserves := getBtcReserves()
	var currentReserve BtcReserve
	for _, reserve := range BtcReserves.BtcReserves {
		if reserve.JudgeAddress == oracleAddr {
			currentReserve = reserve
		}
	}

	reserveId, _ := strconv.Atoi(currentReserve.ReserveId)

	if reserveId == 1 {
		UpdateAddressUnlockHeight(newSweepAddress, height+int64(144))
	} else if reserveId == 2 {
		UpdateAddressUnlockHeight(newSweepAddress, height+int64(72))
	}

	return newSweepAddress
}

func generateAndRegisterNewProposedAddress(accountName string, height int64, oldReserveAddress string) (string, string) {
	newSweepAddress, script := generateAddress(height, oldReserveAddress)
	registerAddressOnForkscanner(newSweepAddress)
	hexScript := hex.EncodeToString(script)
	return newSweepAddress, hexScript
}

func registerJudge(accountName string) {
	cosmos := getCosmosClient()
	msg := &bridgetypes.MsgRegisterJudge{
		Creator:          oracleAddr,
		JudgeAddress:     oracleAddr,
		ValidatorAddress: valAddr,
	}

	sendTransactionRegisterJudge(accountName, cosmos, msg)
	fmt.Println("registered Judge")
}

func orderSignSweep(sweepSignatures MsgSignSweepResp) []MsgSignSweep {
	delegateAddresses := getDelegateAddresses()
	orderedSignSweep := make([]MsgSignSweep, 0)

	for _, oracleAddr := range delegateAddresses.Addresses {
		for _, sweepSig := range sweepSignatures.SignSweepMsg {
			if oracleAddr.BtcOracleAddress == sweepSig.BtcOracleAddress {
				orderedSignSweep = append(orderedSignSweep, sweepSig)
			}
		}
	}

	fmt.Println("Signatures Sweep : ", len(orderedSignSweep))

	return orderedSignSweep
}

func OrderSignRefund(refundSignatures MsgSignRefundResp, address string) ([]MsgSignRefund, MsgSignRefund) {
	delegateAddresses := getDelegateAddresses()
	//needs to change for multi judge > 2 with staking in place
	registeredJudges := getRegisteredJudges()
	var otherJudgeAddress RegisteredJudge

	for _, judge := range registeredJudges.Judges {
		if judge.JudgeAddress != oracleAddr {
			otherJudgeAddress = judge
		}
	}

	orderedSignRefund := make([]MsgSignRefund, 0)
	var judgeSign MsgSignRefund

	for _, oracleAddr := range delegateAddresses.Addresses {
		for _, refundSig := range refundSignatures.SignRefundMsg {
			if oracleAddr.BtcOracleAddress == refundSig.BtcOracleAddress {
				orderedSignRefund = append(orderedSignRefund, refundSig)
			}
			if otherJudgeAddress.JudgeAddress == refundSig.BtcOracleAddress {
				judgeSign = refundSig
			}
		}
	}
	fmt.Println("Signatures refund : ", len(orderedSignRefund))

	return orderedSignRefund, judgeSign
}

func getBtcWithdrawRequestForAddress(sweepAddress SweepAddress) []BtcWithdrawRequest {
	withdrawals := getBtcWithdrawRequest()

	withdrawRequests := make([]BtcWithdrawRequest, 0)
	for _, withdrawal := range withdrawals.WithdrawRequest {
		if withdrawal.ReserveAddress == sweepAddress.Address {
			withdrawRequests = append(withdrawRequests, withdrawal)
		}
	}

	return withdrawRequests
}

func getBtcFeeRate() FeeRate {
	resp, err := http.Get("https://api.blockchain.info/mempool/fees")
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	a := FeeRate{}
	err = json.Unmarshal(body, &a)

	return a
}
