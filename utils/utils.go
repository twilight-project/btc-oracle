package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
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

func CreateTxFromHex(txHex string) (*wire.MsgTx, error) {
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string: %v", err)
	}

	tx := wire.NewMsgTx(wire.TxVersion)

	err = tx.Deserialize(bytes.NewReader(txBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %v", err)
	}

	return tx, nil
}

func StringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

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
		return fragment, fmt.Errorf("No fragment found with the specified judge address")
	}
	return fragment, nil
}

func FilterAndOrderSignSweep(sweepSignatures btcOracleTypes.MsgSignSweepResp, pubkeys []string, judgeAddr string) []btcOracleTypes.MsgSignSweep {
	fmt.Println("Public Keys: ", pubkeys)

	fragment, err := GetCurrentFragment(judgeAddr)
	if err != nil {
		fmt.Println("Error getting current fragment : ", err)
		return nil
	}

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

	fragment, err := GetCurrentFragment(judgeAddr)

	if err != nil {
		fmt.Println("Error getting current fragment : ", err)
		return nil
	}

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

func GetFeeRateFromBtcNode(tx *wire.MsgTx) (float64, error) {
	walletName := viper.GetString("wallet_name")
	result, err := comms.GetEstimateFee(walletName)
	if err != nil {
		fmt.Println("Error getting fee rate : ", err)
		return 0, err
	}

	feeRateInBtc := result.Result.Feerate

	fmt.Printf("Estimated fee rate: %f BTC\n", feeRateInBtc)
	return feeRateInBtc, nil
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

func GetUnlockHeightFromScript(script string) int64 {
	height := int64(0)
	part := 10
	parts := strings.Split(script, " ")
	if len(parts) == 0 {
		return height
	}

	heightstr := parts[part-1]
	if h, err := strconv.ParseInt(heightstr, 10, 64); err == nil {
		return h
	}

	// Otherwise treat as hex bytes in little-endian ScriptNum (e.g. "fe280e")
	b, err := hex.DecodeString(heightstr)
	if err != nil || len(b) == 0 {
		return 0
	}
	return scriptNumLEToInt64(b)
}

func scriptNumLEToInt64(v []byte) int64 {
	if len(v) == 0 {
		return 0
	}
	neg := (v[len(v)-1] & 0x80) != 0
	v = append([]byte(nil), v...) // copy to avoid mutating caller slice
	v[len(v)-1] &^= byte(0x80)

	var n int64
	for i := 0; i < len(v); i++ {
		n |= int64(v[i]) << (8 * i)
	}
	if neg {
		n = -n
	}
	return n
}

func GetMinSignFromScript(script string) int {
	var m int
	_, err := fmt.Sscanf(script, "%d", &m)
	if err != nil {
		fmt.Println("Error parsing m value:", err)
		return 0
	}

	fmt.Println("Minimum Signature Required : ", m)
	return m
}

func GetPublicKeysFromScript(script string, limit int) []string {
	pubkeys := []string{}
	parts := strings.Split(script, " ")
	if len(parts) <= 1+limit {
		return pubkeys
	}
	pubkeys = append(pubkeys, parts[1:1+limit]...)

	return pubkeys
}
