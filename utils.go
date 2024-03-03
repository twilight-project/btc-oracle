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
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	bridgetypes "github.com/twilight-project/nyks/x/bridge/types"
)

func initConfigFile() {
	viper.AddConfigPath("./configs")
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
	fmt.Println("registering address on validators")
	savedAddress := queryAllAddressOnly()
	respReserve := getReserveAddresses()
	if len(respReserve.Addresses) > 0 {
		for _, address := range respReserve.Addresses {
			if !stringInSlice(address.ReserveAddress, savedAddress) {
				registerAddressOnForkscanner(address.ReserveAddress)
				decodedScript := decodeBtcScript(address.ReserveScript)
				height := getHeightFromScript(decodedScript)
				reserveScript, _ := hex.DecodeString(address.ReserveScript)
				insertSweepAddress(address.ReserveAddress, reserveScript, nil, height+1, "", false)
			}
		}
	}
	respProposed := getProposedAddresses()
	if len(respProposed.ProposeSweepAddressMsgs) > 0 {
		for _, address := range respProposed.ProposeSweepAddressMsgs {
			if !stringInSlice(address.BtcAddress, savedAddress) {
				registerAddressOnForkscanner(address.BtcAddress)
				decodedScript := decodeBtcScript(address.BtcScript)
				height := getHeightFromScript(decodedScript)
				reserveScript, _ := hex.DecodeString(address.BtcScript)
				insertSweepAddress(address.BtcAddress, reserveScript, nil, height+1, "", false)
			}
		}
	}
}

// func getReserveForAddress(address string) BtcReserve {
// 	btcReserves := getBtcReserves()
// 	for _, reserve := range btcReserves.BtcReserves {
// 		if reserve.ReserveAddress == address {
// 			return reserve
// 		}
// 	}
// 	return BtcReserve{}
// }

func registerReserveAddressOnNyks(accountName string, address string, script []byte) {

	cosmos := getCosmosClient()

	reserveScript := hex.EncodeToString(script)

	msg := &bridgetypes.MsgRegisterReserveAddress{
		ReserveScript:  reserveScript,
		ReserveAddress: address,
		JudgeAddress:   oracleAddr,
	}

	// store response in txResp
	txResp, err := sendTransactionRegisterReserveAddress(accountName, cosmos, msg)
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

	fmt.Println("registered address on forkscanner : ", address)

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

// func reverseArray(arr []MsgSignSweep) []MsgSignSweep {
// 	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
// 		arr[i], arr[j] = arr[j], arr[i]
// 	}
// 	return arr
// }

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

func sendSweepSign(hexSignatures []string, address string, accountName string, reserveId uint64, roundId uint64) {
	cosmos := getCosmosClient()
	msg := &bridgetypes.MsgSignSweep{
		ReserveId:        reserveId,
		RoundId:          roundId,
		SignerPublicKey:  getBtcPublicKey(),
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
		SignerPublicKey:  getBtcPublicKey(),
		RefundSignature:  []string{hexSignatures},
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

	// BtcReserves := getBtcReserves()
	// var currentReserve BtcReserve
	// for _, reserve := range BtcReserves.BtcReserves {
	// 	if reserve.JudgeAddress == oracleAddr {
	// 		currentReserve = reserve
	// 	}
	// }

	// reserveId, _ := strconv.Atoi(currentReserve.ReserveId)

	// if reserveId == 1 {
	// 	UpdateAddressUnlockHeight(newSweepAddress, height+int64(144))
	// } else if reserveId == 2 {
	// 	UpdateAddressUnlockHeight(newSweepAddress, height+int64(72))
	// }

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

func filterAndOrderSignSweep(sweepSignatures MsgSignSweepResp, pubkeys []string) []MsgSignSweep {
	fmt.Println(sweepSignatures.SignSweepMsg)
	fmt.Println(pubkeys)
	filtereSignSweep := []MsgSignSweep{}
	for _, sweepSig := range sweepSignatures.SignSweepMsg {
		if stringInSlice(sweepSig.SignerPublicKey, pubkeys) {
			filtereSignSweep = append(filtereSignSweep, sweepSig)
		}
	}

	delegateAddresses := getDelegateAddresses()
	orderedSignSweep := make([]MsgSignSweep, 0)

	for _, oracleAddr := range delegateAddresses.Addresses {
		for _, sweepSig := range filtereSignSweep {
			if oracleAddr.BtcOracleAddress == sweepSig.BtcOracleAddress {
				orderedSignSweep = append(orderedSignSweep, sweepSig)
			}
		}
	}

	fmt.Println("Signatures Sweep : ", len(orderedSignSweep))

	return orderedSignSweep
}

func OrderSignRefund(refundSignatures MsgSignRefundResp, address string, pubkeys []string) ([]MsgSignRefund, MsgSignRefund) {
	delegateAddresses := getDelegateAddresses()

	//needs to change for multi judge > 2 with staking in place
	registeredJudges := getRegisteredJudges()
	var otherJudgeAddress RegisteredJudge

	if len(registeredJudges.Judges) > 1 {
		for _, judge := range registeredJudges.Judges {
			if judge.JudgeAddress != oracleAddr {
				otherJudgeAddress = judge
			}
		}
	} else {
		otherJudgeAddress = registeredJudges.Judges[0]
	}

	filteresSignRefund := make([]MsgSignRefund, 0)
	for _, refundSig := range refundSignatures.SignRefundMsg {
		if stringInSlice(refundSig.signerPublicKey, pubkeys) {
			filteresSignRefund = append(filteresSignRefund, refundSig)
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
	if err != nil {
		fmt.Println("Error decoding Fee Rate : ", err)
	}

	return a
}

func decodeBtcScript(script string) string {
	decoded, err := hex.DecodeString(script)
	if err != nil {
		fmt.Println("Error decoding script Hex : ", err)
	}
	decodedScript, err := txscript.DisasmString(decoded)
	if err != nil {
		fmt.Println("Error decoding script : ", err)
	}

	return decodedScript
}

func getHeightFromScript(script string) int64 {
	// Split the decoded script into parts
	height := int64(0)
	parts := strings.Split(script, " ")
	if len(parts) == 0 {
		return height
	}
	// Reverse the byte order
	for i, j := 0, len(parts[0])-2; i < j; i, j = i+2, j-2 {
		parts[0] = parts[0][:i] + parts[0][j:j+2] + parts[0][i+2:j] + parts[0][i:i+2] + parts[0][j+2:]
	}
	// Convert the first part from hex to decimal
	height, err := strconv.ParseInt(parts[0], 16, 64)
	if err != nil {
		fmt.Println("Error converting block height from hex to decimal:", err)
	}

	return height
}

func getMinSignFromScript(script string) int64 {
	// Split the decoded script into parts
	minSignRequired := int64(0)
	parts := strings.Split(script, " ")
	if len(parts) < 4 {
		return minSignRequired
	}
	// Reverse the byte order
	for i, j := 0, len(parts[3])-2; i < j; i, j = i+2, j-2 {
		parts[3] = parts[3][:i] + parts[3][j:j+2] + parts[3][i+2:j] + parts[3][i:i+2] + parts[3][j+2:]
	}
	// Convert the first part from hex to decimal
	minSignRequired, err := strconv.ParseInt(parts[3], 16, 64)
	if err != nil {
		fmt.Println("Error converting block height from hex to decimal:", err)
	}

	return minSignRequired
}

func getPublicKeysFromScript(script string, limit int) []string {
	// Split the decoded script into parts
	pubkeys := []string{}
	parts := strings.Split(script, " ")
	if len(parts) <= 4+limit {
		return pubkeys
	}
	// Reverse the byte order
	for i, j := 0, len(parts[3])-2; i < j; i, j = i+2, j-2 {
		parts[3] = parts[3][:i] + parts[3][j:j+2] + parts[3][i+2:j] + parts[3][i:i+2] + parts[3][j+2:]
	}
	// Convert the first part from hex to decimal
	pubkeys = append(pubkeys, parts[4:4+limit]...)

	return pubkeys
}

func nyksEventListener(event string, accountName string, functionCall string) {
	headers := make(map[string][]string)
	headers["Content-Type"] = []string{"application/json"}
	nyksd_url := fmt.Sprintf("%v", viper.Get("nyksd_socket_url"))
	conn, _, err := websocket.DefaultDialer.Dial(nyksd_url, headers)
	if err != nil {
		fmt.Println("nyks event listerner dial:", err)
	}
	defer conn.Close()

	// Set up ping/pong connection health check
	pingPeriod := 30 * time.Second
	pongWait := 60 * time.Second
	stopChan := make(chan struct{}) // Create the stop channel

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
			case <-stopChan: // Listen to the stop channel
				return
			}
		}
	}()

	payload := `{
        "jsonrpc": "2.0",
        "method": "subscribe",
        "id": 0,
        "params": {
            "query": "tm.event='Tx' AND message.action='%s'"
        }
    }`
	payload = fmt.Sprintf(payload, event)

	err = conn.WriteMessage(websocket.TextMessage, []byte(payload))
	if err != nil {
		fmt.Println("error in nyks event handler: ", err)
		stopChan <- struct{}{} // Signal goroutine to stop
		return
	}

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("error in nyks event handler: ", err)
			stopChan <- struct{}{} // Signal goroutine to stop
			return
		}

		// var event Event
		// err = json.Unmarshal(message, &event)
		// if err != nil {
		// 	fmt.Println("error unmarshalling event: ", err)
		// 	continue
		// }

		// fmt.Print("event : ", event)
		// fmt.Print("event : ", message)

		// if event.Method == "subscribe" && event.Params.Query == fmt.Sprintf("tm.event='Tx' AND message.action='%s'", event) {
		// 	continue
		// }

		switch functionCall {
		case "signed_sweep_process":
			go processSignedSweep(accountName)
		case "refund_process":
			go processRefund(accountName)
		case "signed_refund_process":
			go processSignedRefund(accountName)
		case "register_res_addr_validators":
			go registerAddressOnValidators()
		case "signing_sweep":
			go processTxSigningSweep(accountName)
		case "signing_refund":
			go processTxSigningRefund(accountName)
		case "sweep_process":
			go processSweep(accountName)
		default:
			log.Println("Unknown function :", functionCall)
		}
	}
}
