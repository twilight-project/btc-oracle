package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/spf13/viper"
	comms "github.com/twilight-project/forkoracle-go/comms"
	btcOracleTypes "github.com/twilight-project/forkoracle-go/types"
)

func InitConfigFile() {
	viper.AddConfigPath("./configs")
	viper.SetConfigName("config") // Register config file name (no extension)
	viper.SetConfigType("json")   // Look for specific type
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("Error reading config file: ", err)
	}
}

func SetDelegator(valAddr string, oracleAddr string, btcPublicKey string) {
	accountName := fmt.Sprintf("%v", viper.Get("accountName"))
	command := fmt.Sprintf("nyksd tx nyks set-delegate-addresses %s %s %s %s --from %s --chain-id nyks --keyring-backend test -y", valAddr, oracleAddr, btcPublicKey, oracleAddr, accountName)
	fmt.Println("delegate command : ", command)

	args := strings.Fields(command)
	cmd := exec.Command(args[0], args[1:]...)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		panic(err)
	}

	fmt.Println("Delegate Address Set")
}

func getBitcoinRpcClient(walletName string) *rpcclient.Client {
	connCfg := &rpcclient.ConnConfig{
		Host:         viper.GetString("btc_node_host"),
		User:         viper.GetString("btc_node_user"),
		Pass:         viper.GetString("btc_node_pass"),
		HTTPPostMode: true,
		DisableTLS:   true,
	}

	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		fmt.Println("Failed to connect to the Bitcoin client : ", err)
	}

	return client
}

func LoadBtcWallet(walletName string) {
	client := getBitcoinRpcClient(walletName)
	_, err := client.LoadWallet(walletName)
	if err != nil {
		fmt.Println("Failed to load wallet : ", err)
	}
}

// func GetAddressInfo(addr string) (*btcjson.GetAddressInfoResult, error) {
// 	client := getBitcoinRpcClient()
// 	addressInfo, err := client.GetAddressInfo(addr)
// 	if err != nil {
// 		fmt.Println("Error getting address info : ", err)
// 		return nil, err
// 	}
// 	return addressInfo, nil
// }

// func BroadcastBtcTransaction(tx *wire.MsgTx) {
// 	client := getBitcoinRpcClient()
// 	txHash, err := client.SendRawTransaction(tx, true)
// 	if err != nil {
// 		fmt.Println("Failed to broadcast transaction : ", err)
// 	}

// 	defer client.Shutdown()
// 	fmt.Println("broadcasted btc transaction, txhash : ", txHash)
// }

// func getReserveForAddress(address string) BtcReserve {
// 	btcReserves := getBtcReserves()
// 	for _, reserve := range btcReserves.BtcReserves {
// 		if reserve.ReserveAddress == address {
// 			return reserve
// 		}
// 	}
// 	return BtcReserve{}
// }

func CreateTxFromHex(txHex string) (*wire.MsgTx, error) {
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

func SignTx(tx *wire.MsgTx, script []byte) []string {
	signatures := []string{}
	witnessInputs := make([]btcjson.RawTxWitnessInput, len(tx.TxIn))
	walletName := viper.GetString("wallet_name")
	client := getBitcoinRpcClient(walletName)
	fmt.Println("got client btc")
	for i, input := range tx.TxIn {
		fmt.Println("inside loop")
		t, err := client.GetRawTransactionVerbose(&input.PreviousOutPoint.Hash)
		if err != nil {
			fmt.Println("error in getting raw transaction from btc wallet : ", err)
		}
		fmt.Println("got prev tx btc")
		witnessInputs[i] = btcjson.RawTxWitnessInput{
			Txid:         input.PreviousOutPoint.Hash.String(),
			Vout:         input.PreviousOutPoint.Index,
			ScriptPubKey: t.Vout[input.PreviousOutPoint.Index].ScriptPubKey.Hex,
			Amount:       &t.Vout[input.PreviousOutPoint.Index].Value,
		}
	}
	fmt.Println("actual sign starting")
	signedTx, _, err := client.SignRawTransactionWithWallet3(tx, witnessInputs, rpcclient.SigHashAllAnyoneCanPay)
	fmt.Println("actual sign done")
	if err != nil {
		fmt.Println("Error in signing btc tx:", err)
	}

	for _, input := range signedTx.TxIn {
		signatures = append(signatures, hex.EncodeToString(input.Witness[0]))
	}
	return signatures
}

func RefundsignTx(tx *wire.MsgTx, script []byte) []string {
	signatures := []string{}
	scriptHex := hex.EncodeToString(script)
	tx.LockTime = uint32(849765)
	witnessInputs := make([]btcjson.RawTxWitnessInput, len(tx.TxIn))
	walletName := viper.GetString("wallet_name")
	client := getBitcoinRpcClient(walletName)
	sum := 0
	for _, output := range tx.TxOut {
		sum += int(output.Value)
	}
	sumfloat := float64(sum)
	total := &sumfloat

	// Compute the SHA-256 hash
	hash := sha256.Sum256(script)

	fmt.Println("amount : ", *total)
	fmt.Println("locktime : ", tx.LockTime)
	fmt.Println("scriptpubkey  : ", "0020"+hex.EncodeToString(hash[:]))
	fmt.Println("witnessscript (locking)  : ", hex.EncodeToString(script))
	scriptpubkey := "0020" + hex.EncodeToString(hash[:])

	for i, input := range tx.TxIn {
		fmt.Println("inside loop")
		fmt.Println("Txid : ", input.PreviousOutPoint.Hash.String())
		fmt.Println("Vout : ", input.PreviousOutPoint.Index)
		witnessInputs[i] = btcjson.RawTxWitnessInput{
			Txid:          input.PreviousOutPoint.Hash.String(),
			Vout:          input.PreviousOutPoint.Index,
			Amount:        total,
			ScriptPubKey:  scriptpubkey,
			WitnessScript: &scriptHex,
		}
	}
	fmt.Println("actual sign starting")
	signedTx, status, err := client.SignRawTransactionWithWallet3(tx, witnessInputs, rpcclient.SigHashAllAnyoneCanPay)
	fmt.Println("signed : ", status)
	fmt.Println("actual sign done")
	if err != nil {
		fmt.Println("Error in signing btc tx:", err)
	}

	var UnsignedTx bytes.Buffer
	err = signedTx.Serialize(&UnsignedTx)
	if err != nil {
		fmt.Println("error in serializing sweep tx : ", err)
	}
	hexTx := hex.EncodeToString(UnsignedTx.Bytes())
	fmt.Println(hexTx)

	for _, input := range signedTx.TxIn {
		signatures = append(signatures, hex.EncodeToString(input.Witness[0]))
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

func StringInSlice(str string, slice []string) bool {
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

func CreateTxIn(utxo btcOracleTypes.Utxo) (*wire.TxIn, error) {
	utxoHash, err := chainhash.NewHashFromStr(utxo.Txid)
	if err != nil {
		log.Println("error with UTXO")
		return nil, err
	}
	outPoint := wire.NewOutPoint(utxoHash, utxo.Vout)
	txIn := wire.NewTxIn(outPoint, nil, nil)
	return txIn, nil
}

// func SendSweepSign(hexSignatures []string, address string, accountName string, reserveId uint64, roundId uint64, oracleAddr string) {
// 	cosmos := comms.GetCosmosClient()
// 	msg := &bridgetypes.MsgSignSweep{
// 		ReserveId:        reserveId,
// 		RoundId:          roundId,
// 		SignerPublicKey:  getBtcPublicKey(),
// 		SweepSignature:   hexSignatures,
// 		BtcOracleAddress: oracleAddr,
// 	}

// 	comms.SendTransactionSignSweep(accountName, cosmos, msg)
// }

// func sendRefundSign(hexSignatures string, address string, accountName string, reserveId uint64, roundId uint64, oracleAddr string) {
// 	cosmos := comms.GetCosmosClient()
// 	msg := &bridgetypes.MsgSignRefund{
// 		ReserveId:        reserveId,
// 		RoundId:          roundId,
// 		SignerPublicKey:  getBtcPublicKey(),
// 		RefundSignature:  []string{hexSignatures},
// 		BtcOracleAddress: oracleAddr,
// 	}

// 	comms.SendTransactionSignRefund(accountName, cosmos, msg)
// }

// func RegisterJudge(accountName string, oracleAddr string, valAddr string) {
// 	cosmos := comms.GetCosmosClient()
// 	msg := &bridgetypes.MsgRegisterJudge{
// 		Creator:          oracleAddr,
// 		JudgeAddress:     oracleAddr,
// 		ValidatorAddress: valAddr,
// 	}

// 	comms.SendTransactionRegisterJudge(accountName, cosmos, msg)
// 	fmt.Println("registered Judge")
// }

// check the fragment of the current judge
func GetCurrentFragment(judgeAddr string) (btcOracleTypes.Fragment, error) {
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
		//log.Println("No fragment found with the specified judge address")
		return fragment, fmt.Errorf("No fragment found with the specified judge address")
	}
	return fragment, nil
}

// Check if the current Judge belongs to any registered fragment
// acquire the fragment and its Signers list
// check if the current signer is in the Signers list
// add the signatures in the order the signers appear in the Fragment
// The ordering should be looked at again once we have multiple fragments to reaafirm the ordering algorithm
// Note that currently the checks work on the assumption that each Judge has only one fragment and each Signer has only one address associated with BTC wallet address
// The function should be updated to handle matching public keys used in the BTC sweep script
func FilterAndOrderSignSweep(sweepSignatures btcOracleTypes.MsgSignSweepResp, pubkeys []string, judgeAddr string) []btcOracleTypes.MsgSignSweep {
	fmt.Println(sweepSignatures.SignSweepMsg)
	fmt.Println(pubkeys)
	// FIltering is not required in the new design
	// filtereSignSweep := []btcOracleTypes.MsgSignSweep{}
	// for _, sweepSig := range sweepSignatures.SignSweepMsg {
	// 	if StringInSlice(sweepSig.SignerPublicKey, pubkeys) {
	// 		filtereSignSweep = append(filtereSignSweep, sweepSig)
	// 	}
	// }

	// check if the current Judge belongs to any registered fragment
	fragment, err := GetCurrentFragment(judgeAddr)
	if err != nil {
		fmt.Println("Error getting current fragment : ", err)
		return nil
	}
	// acquire the fragment and its Signers list
	// check if the current signer is in the Signers list
	orderedSignSweep := make([]btcOracleTypes.MsgSignSweep, 0)

	for _, signer := range fragment.Signers {
		for _, sweepSig := range sweepSignatures.SignSweepMsg {
			if signer.SignerAddress == sweepSig.SignerAddress {
				orderedSignSweep = append(orderedSignSweep, sweepSig)
			}
		}
	}

	fmt.Println("Signatures Sweep : ", len(orderedSignSweep))

	return orderedSignSweep
}

func OrderSignRefund(refundSignatures btcOracleTypes.MsgSignRefundResp, address string,
	pubkeys []string, judgeAddr string) []btcOracleTypes.MsgSignRefund {
	fmt.Println("Inside OrderSignRefund*******")
	fmt.Println("script address : ", address)
	fmt.Println("public keys : ", pubkeys)
	fmt.Println("judge address : ", judgeAddr)

	// check if the current Judge belongs to any registered fragment
	fragment, err := GetCurrentFragment(judgeAddr)

	if err != nil {
		fmt.Println("Error getting current fragment : ", err)
		return nil
	}

	// acquire the fragment and its Signers list
	orderedSignRefund := make([]btcOracleTypes.MsgSignRefund, 0)

	for _, signer := range fragment.Signers {
		for _, refundSig := range refundSignatures.SignRefundMsg {
			if signer.SignerAddress == refundSig.SignerAddress {
				orderedSignRefund = append(orderedSignRefund, refundSig)
			}
		}
	}
	fmt.Println("Signatures refund : ", len(orderedSignRefund))

	return orderedSignRefund
}

// func GetCurrentReserveandRound(oracleAddr string) (btcOracleTypes.BtcReserve, uint64, uint64, error) {

// 	empty := btcOracleTypes.BtcReserve{"", "", "", "", "", "", "", "", "", ""}
// 	var currentReservesForThisJudge []btcOracleTypes.BtcReserve
// 	reserves := comms.GetBtcReserves()
// 	for _, reserve := range reserves.BtcReserves {
// 		if reserve.JudgeAddress == oracleAddr {
// 			currentReservesForThisJudge = append(currentReservesForThisJudge, reserve)
// 		}
// 	}

// 	if len(currentReservesForThisJudge) == 0 {
// 		time.Sleep(2 * time.Minute)
// 		fmt.Println("no judge")
// 		return empty, 0, 0, errors.New("No Judge Found")
// 	}

// 	currentJudgeReserve := currentReservesForThisJudge[0]

// 	reserveIdForProposal, _ := strconv.Atoi(currentJudgeReserve.ReserveId)
// 	if reserveIdForProposal == 1 {
// 		reserveIdForProposal = len(reserves.BtcReserves)
// 	} else {
// 		reserveIdForProposal = reserveIdForProposal - 1
// 	}

// 	var reserveToBeUpdated btcOracleTypes.BtcReserve
// 	for _, reserve := range reserves.BtcReserves {
// 		tempId, _ := strconv.Atoi(reserve.ReserveId)
// 		if tempId == reserveIdForProposal {
// 			reserveToBeUpdated = reserve
// 			break
// 		}
// 	}

// 	RoundId, _ := strconv.Atoi(reserveToBeUpdated.RoundId)
// 	return currentJudgeReserve, uint64(reserveIdForProposal), uint64(RoundId), nil
// }

func BtcToSats(btc float64) int64 {
	return int64(btc * 1e8)
}

func SatsToBtc(sats int64) float64 {
	return float64(sats) / 100000000.0
}

func Base64ToHex(base64String string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func HexToBase64(hexString string) (string, error) {
	data, err := hex.DecodeString(hexString)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func DecodeBtcScript(script string) string {
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

func GetFeeRateFromBtcNode(tx *wire.MsgTx) (int64, error) {
	walletName := viper.GetString("wallet_name")
	feeRateAdjustment := viper.GetInt64("fee_rate_adjustment")
	result, err := comms.GetEstimateFee(walletName)
	if err != nil {
		fmt.Println("Error getting fee rate : ", err)
		return 0, err
	}

	feeRateInBtc := result.Result.Feerate

	fmt.Printf("Estimated fee per kilobyte for a transaction to be confirmed within 2 blocks: %f BTC\n", feeRateInBtc)
	feeRate := BtcToSats(feeRateInBtc) + feeRateAdjustment
	fmt.Printf("Estimated fee per kilobyte for a transaction to be confirmed within 2 blocks: %d Sats\n", feeRate)
	return feeRate, nil

}

// func GetBtcFeeRate() btcOracleTypes.FeeRate {
// 	resp, err := http.Get("https://api.blockchain.info/mempool/fees")
// 	if err != nil {
// 		log.Fatalln(err)
// 	}
// 	//We Read the response body on the line below.
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		log.Fatalln(err)
// 	}

// 	a := btcOracleTypes.FeeRate{}
// 	err = json.Unmarshal(body, &a)
// 	if err != nil {
// 		fmt.Println("Error decoding Fee Rate : ", err)
// 	}

// 	return a
// }

func CreateFeeUtxo(fee int64) (string, error) {
	walletName := viper.GetString("judge_btc_wallet_name")
	address, err := comms.GetNewAddress(walletName)
	if err != nil {
		fmt.Println("Failed to get new address for fee utxo : ", err)
		return "", err
	}
	feeInBtc := SatsToBtc(fee)
	fmt.Println("fee in btc : ", feeInBtc)
	result, err := comms.SendToAddress(address, feeInBtc, walletName)
	if err != nil {
		fmt.Println("Failed to send btc to address : ", err)
		return "", err
	}
	fmt.Println("Fee Utxo created : ", result.Result)
	return result.Result, nil
}

func BroadcastBtcTransaction(tx *wire.MsgTx) {
	walletName := viper.GetString("judge_btc_wallet_name")
	client := getBitcoinRpcClient(walletName)
	txHash, err := client.SendRawTransaction(tx, true)
	if err != nil {
		fmt.Println("Failed to broadcast transaction : ", err)
	}

	defer client.Shutdown()
	fmt.Println("broadcasted btc transaction, txhash : ", txHash)
}

func SignFeeUtxo(tx *wire.MsgTx) (wire.TxWitness, error) {
	walletName := viper.GetString("judge_btc_wallet_name")
	client := getBitcoinRpcClient(walletName)
	signedTx, _, err := client.SignRawTransactionWithWallet(tx)
	if err != nil {
		fmt.Println("Failed to sign fee utxo : ", err)
		return nil, err
	}
	total := len(tx.TxIn)
	return signedTx.TxIn[total-1].Witness, nil
}

// func GetUnlockHeightFromScript(script string) int64 {
// 	// Split the decoded script into parts
// 	height := int64(0)
// 	parts := strings.Split(script, " ")
// 	if len(parts) == 0 {
// 		return height
// 	}
// 	// Reverse the byte order
// 	for i, j := 0, len(parts[0])-2; i < j; i, j = i+2, j-2 {
// 		parts[0] = parts[0][:i] + parts[0][j:j+2] + parts[0][i+2:j] + parts[0][i:i+2] + parts[0][j+2:]
// 	}
// 	// Convert the first part from hex to decimal
// 	height, err := strconv.ParseInt(parts[0], 16, 64)
// 	if err != nil {
// 		fmt.Println("Error converting block height from hex to decimal:", err)
// 	}

// 	return height
// }

func GetUnlockHeightFromScript(script string) int64 {
	// Split the decoded script into parts
	height := int64(0)
	part := 25
	parts := strings.Split(script, " ")
	if len(parts) == 0 {
		return height
	}
	// Reverse the byte order
	for i, j := 0, len(parts[part])-2; i < j; i, j = i+2, j-2 {
		parts[part] = parts[part][:i] + parts[part][j:j+2] + parts[part][i+2:j] + parts[part][i:i+2] + parts[part][j+2:]
	}
	// Convert the first part from hex to decimal
	height, err := strconv.ParseInt(parts[part], 16, 64)
	if err != nil {
		fmt.Println("Error converting block height from hex to decimal:", err)
	}

	return height
}

func getUnlockHeightFromMiniscript(s string) ([]string, error) {
	re := regexp.MustCompile(`after\((\d+)\)`)
	matches := re.FindAllStringSubmatch(s, -1)

	var values []string
	for _, match := range matches {
		if len(match) > 1 {
			values = append(values, match[1])
		}
	}
	return values, nil
}

func GetMinSignFromScript(script string) int {
	// Split the decoded script into parts
	var m int
	_, err := fmt.Sscanf(script, "%d", &m)
	if err != nil {
		fmt.Println("Error parsing m value:", err)
		return 0
	}

	fmt.Println(m)
	return m
}

func GetPublicKeysFromScript(script string, limit int) []string {
	// Split the decoded script into parts
	pubkeys := []string{}
	parts := strings.Split(script, " ")
	if len(parts) <= 1+limit {
		return pubkeys
	}
	// // Reverse the byte order
	// for i, j := 0, len(parts[1])-2; i < j; i, j = i+2, j-2 {
	// 	parts[1] = parts[1][:i] + parts[1][j:j+2] + parts[1][i+2:j] + parts[1][i:i+2] + parts[1][j+2:]
	// }
	// Convert the first part from hex to decimal
	pubkeys = append(pubkeys, parts[1:1+limit]...)

	return pubkeys
}
