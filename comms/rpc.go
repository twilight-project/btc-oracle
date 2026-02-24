package comms

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/psbt"
	"github.com/cosmos/btcutil"
	"github.com/spf13/viper"
)

// generic request
type JSONRPCRequest struct {
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Jsonrpc string      `json:"jsonrpc"`
}

// generic string response
type JSONRPCResponse struct {
	Result string      `json:"result"`
	Error  interface{} `json:"error"`
	ID     int         `json:"id"`
}

type JSONRPCResponseAddressInfo struct {
	Result AddressInfo `json:"result"`
	Error  interface{} `json:"error"`
	ID     int64       `json:"id"`
}

type AddressInfo struct {
	Address        string   `json:"address"`
	ScriptPubKey   string   `json:"scriptPubKey"`
	Ismine         bool     `json:"ismine"`
	Solvable       bool     `json:"solvable"`
	Desc           string   `json:"desc"`
	ParentDesc     string   `json:"parent_desc"`
	IsWatchOnly    bool     `json:"iswatchonly"`
	Isscript       bool     `json:"isscript"`
	IsWitness      bool     `json:"iswitness"`
	WitnessVersion int      `json:"witness_version"`
	WitnessProgram string   `json:"witness_program"`
	Script         string   `json:"script"`
	Hex            string   `json:"hex"`
	IsChange       bool     `json:"ischange"`
	Labels         []string `json:"labels"`
}

// response with desc info
type JSONRPCResponseDesc struct {
	Result DescriptorInfo `json:"result"`
	Error  interface{}    `json:"error"`
	ID     int64          `json:"id"`
}

type DescriptorInfo struct {
	Descriptor     string `json:"descriptor"`
	Checksum       string `json:"checksum"`
	IsRange        bool   `json:"isrange"`
	IsSolvable     bool   `json:"issolvable"`
	HasPrivateKeys bool   `json:"hasprivatekeys"`
}

// //response for importing descriptor
type ImportDescriptorType struct {
	Desc      string `json:"desc"`
	Active    bool   `json:"active"`
	Internal  bool   `json:"internal"`
	Timestamp string `json:"timestamp"`
}

// response for decoding psbt
type JSONRPCResponsePsbt struct {
	Result PSBT        `json:"result"`
	Error  interface{} `json:"error"`
	ID     int64       `json:"id"`
}

type PSBT struct {
	Tx          Tx                `json:"tx"`
	GlobalXPubs []interface{}     `json:"global_xpubs"`
	PSBTVersion int               `json:"psbt_version"`
	Proprietary []interface{}     `json:"proprietary"`
	Unknown     map[string]string `json:"unknown"`
	Inputs      []Input           `json:"inputs"`
	Outputs     []Output          `json:"outputs"`
	Fee         float64           `json:"fee"`
}

type Tx struct {
	TxID     string `json:"txid"`
	Hash     string `json:"hash"`
	Version  int    `json:"version"`
	Size     int    `json:"size"`
	VSize    int    `json:"vsize"`
	Weight   int    `json:"weight"`
	Locktime int    `json:"locktime"`
	Vin      []Vin  `json:"vin"`
	Vout     []Vout `json:"vout"`
}

type Vin struct {
	TxID      string    `json:"txid"`
	Vout      int       `json:"vout"`
	ScriptSig ScriptSig `json:"scriptSig"`
	Sequence  int       `json:"sequence"`
}

type ScriptSig struct {
	Asm string `json:"asm"`
	Hex string `json:"hex"`
}

type Vout struct {
	Value        float64      `json:"value"`
	N            int          `json:"n"`
	ScriptPubKey ScriptPubKey `json:"scriptPubKey"`
}

type ScriptPubKey struct {
	Asm     string `json:"asm"`
	Desc    string `json:"desc"`
	Hex     string `json:"hex"`
	Address string `json:"address"`
	Type    string `json:"type"`
}

type Input struct {
	WitnessUTXO    WitnessUTXO           `json:"witness_utxo"`
	NonWitnessUTXO NonWitnessUTXO        `json:"non_witness_utxo"`
	PartialSigs    map[string]string     `json:"partial_signatures"`
	WitnessScript  Script                `json:"witness_script"`
	BIP32Derivs    []BIP32DerivationPath `json:"bip32_derivs"`
}

type WitnessUTXO struct {
	Amount       float64      `json:"amount"`
	ScriptPubKey ScriptPubKey `json:"scriptPubKey"`
}

type NonWitnessUTXO struct {
	TxID     string `json:"txid"`
	Hash     string `json:"hash"`
	Version  int    `json:"version"`
	Size     int    `json:"size"`
	VSize    int    `json:"vsize"`
	Weight   int    `json:"weight"`
	Locktime int    `json:"locktime"`
	Vin      []Vin  `json:"vin"`
	Vout     []Vout `json:"vout"`
}

type Script struct {
	Asm  string `json:"asm"`
	Hex  string `json:"hex"`
	Type string `json:"type"`
}

type BIP32DerivationPath struct {
	PubKey            string `json:"pubkey"`
	MasterFingerprint string `json:"master_fingerprint"`
	Path              string `json:"path"`
}

type Output struct {
	WitnessScript Script                `json:"witness_script,omitempty"`
	BIP32Derivs   []BIP32DerivationPath `json:"bip32_derivs,omitempty"`
}

// response for Create PSbt
type RPCResponseCreatePsbt struct {
	Result struct {
		Psbt     string `json:"psbt"`
		Complete bool   `json:"complete"`
	} `json:"result"`
	Error interface{} `json:"error"`
	ID    int         `json:"id"`
}

type RPCResponseSignPsbt struct {
	Result struct {
		Psbt      string  `json:"psbt"`
		Fee       float64 `json:"fee"`
		ChangePos int     `json:"changepos"`
	} `json:"result"`
	Error interface{} `json:"error"`
	ID    int         `json:"id"`
}

type UnspentOutput struct {
	TxID          string  `json:"txid"`
	Vout          int     `json:"vout"`
	Address       string  `json:"address"`
	Amount        float64 `json:"amount"`
	Confirmations int     `json:"confirmations"`
	Spendable     bool    `json:"spendable"`
	Solvable      bool    `json:"solvable"`
	Safe          bool    `json:"safe"`
}

type RPCResponseListUnspent struct {
	Result []UnspentOutput `json:"result"`
	Error  interface{}     `json:"error"`
	ID     int             `json:"id"`
}

// structs for creating a transaction/psbt
type TxInput struct {
	Txid     string `json:"txid"`
	Vout     int64  `json:"vout"`
	Sequence int64  `json:"sequence"`
}

type TxOutput map[string]float64

type CreateTx struct {
	Inputs  []TxInput  `json:"inputs"`
	Outputs []TxOutput `json:"outputs"`
}

type EstimateFeeResponse struct {
	Result struct {
		Feerate float64  `json:"feerate,omitempty"`
		Errors  []string `json:"errors,omitempty"`
		Blocks  int      `json:"blocks"`
	}
	Error interface{} `json:"error"`
	ID    int         `json:"id"`
}

type TransactionInfo struct {
	Result string
	Error  interface{} `json:"error"`
	ID     int         `json:"id"`
}

type TransactionError struct {
	TxID      string `json:"txid"`
	Vout      int    `json:"vout"`
	ScriptSig string `json:"scriptSig"`
	Sequence  int    `json:"sequence"`
	Error     string `json:"error"`
}

type SignedTransactionInfo struct {
	Result struct {
		Hex      string             `json:"hex"`
		Complete bool               `json:"complete"`
		Errors   []TransactionError `json:"errors,omitempty"`
	}
	Error interface{} `json:"error"`
	ID    int         `json:"id"`
}

func SendRPC(method string, data []interface{}, wallet string) ([]byte, error) {
	host := viper.GetString("btc_node_host")
	user := viper.GetString("btc_node_user")
	pass := viper.GetString("btc_node_pass")

	request := JSONRPCRequest{
		ID:      1,
		Method:  method,
		Params:  data,
		Jsonrpc: "1.0",
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error creating JSON: ", err)
		return nil, err
	}

	// Create a HTTP client
	client := &http.Client{}

	// Create a HTTP request
	host = "http://" + host + "/wallet/" + wallet
	req, err := http.NewRequest("POST", host, bytes.NewBuffer(requestJSON))
	if err != nil {
		fmt.Println("Error creating request: ", err)
		return nil, err
	}

	// Set the content type to application/json
	req.Header.Set("Content-Type", "application/json")

	// Set the basic authentication header
	req.SetBasicAuth(user, pass)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request: ", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response: ", err)
		return nil, err
	}
	return body, nil
}

func GetDescriptorInfo(dataStr string, wallet string) (DescriptorInfo, error) {

	var response JSONRPCResponseDesc
	data := []interface{}{dataStr}
	result, err := SendRPC("getdescriptorinfo", data, wallet)
	if err != nil {
		fmt.Println("error getting descriptor info : ", err)
		return response.Result, err
	}

	err = json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return response.Result, err
	}
	if response.Error != nil {
		return DescriptorInfo{}, errors.New("error in getting descriptor info")
	}

	return response.Result, nil
}

func ImportDescriptor(desc string, wallet string) error {

	descData := []ImportDescriptorType{
		{
			Desc:      desc,
			Active:    true,
			Internal:  false,
			Timestamp: "now",
		},
	}
	data := []interface{}{descData}

	_, err := SendRPC("importdescriptors", data, wallet)
	if err != nil {
		fmt.Println("error importing descriptor	: ", err)
	}
	return nil
}

func GetNewAddress(wallet string) (string, error) {
	result, _ := SendRPC("getnewaddress", nil, wallet)
	fmt.Println("result New Address: ", string(result))
	var response JSONRPCResponse
	err := json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return "", err
	}
	if response.Error != nil {
		return "", errors.New("error in get new address")
	}
	return response.Result, nil
}

func DecodePsbt(psbt string, wallet string) (PSBT, error) {
	data := []interface{}{psbt}
	result, _ := SendRPC("decodepsbt", data, wallet)
	var response JSONRPCResponsePsbt
	err := json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return PSBT{}, err
	}
	if response.Error != nil {
		return PSBT{}, errors.New("error in Decode")
	}
	return response.Result, nil
}

func CreateRawTx(inputs []TxInput, outputs []TxOutput, locktime uint32, wallet string) (string, error) {
	data := []interface{}{inputs, outputs, locktime}
	result, _ := SendRPC("createrawtransaction", data, wallet)
	fmt.Println("result in CreateRawTx: ", string(result))
	var response JSONRPCResponse
	err := json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return "", err
	}

	if response.Error != nil {
		return "", errors.New("error in CreateRawTx")
	}

	return response.Result, nil
}

func ConvertToPsbt(txHex string, wallet string) (string, error) {
	data := []interface{}{txHex}
	result, err := SendRPC("converttopsbt", data, wallet)
	if err != nil {
		return "", err
	}

	var response JSONRPCResponse
	if err := json.Unmarshal(result, &response); err != nil {
		return "", err
	}

	if response.Error != nil {
		return "", errors.New("error in converttopsbt")
	}

	return response.Result, nil
}

func ListUnspent(wallet string) ([]UnspentOutput, error) {
	data := []interface{}{1} // min_conf = 1
	result, err := SendRPC("listunspent", data, wallet)
	if err != nil {
		return nil, err
	}

	var response RPCResponseListUnspent
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, errors.New("error in listunspent")
	}

	return response.Result, nil
}

func SignPsbt(psbtStr string, wallet string) ([]string, error) {
	data := []interface{}{psbtStr, true, "ALL|ANYONECANPAY"}
	result, _ := SendRPC("walletprocesspsbt", data, wallet)
	fmt.Println("result Sign PSBT: ", string(result))
	var response RPCResponseCreatePsbt
	err := json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return nil, err
	}
	if response.Error != nil {
		return []string{""}, errors.New("error in SignPsbt")
	}
	p := response.Result.Psbt
	psbt, err := DecodePsbt(p, wallet)

	if len(psbt.Inputs) <= 0 {
		return nil, errors.New("no inputs in psbt")
	}
	var signatures []string

	for _, input := range psbt.Inputs {
		for _, v := range input.PartialSigs {
			signatures = append(signatures, v)
			break
		}
	}

	return signatures, nil
}

func UtxoUpdatePsbt(psbtStr string, desc string, wallet string) (string, error) {
	data := []interface{}{
		psbtStr,
		[]map[string]interface{}{
			{
				"desc": desc,
			},
		},
	}
	result, _ := SendRPC("utxoupdatepsbt", data, wallet)
	fmt.Println("result Update Utxo PSBT: ", string(result))
	var response JSONRPCResponse
	err := json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return "", err
	}

	if response.Error != nil {
		return "", errors.New("error in utxoupdatepsbt")
	}

	return response.Result, nil
}

func CreatePsbtV1(utxo TxInput, outputs []TxOutput, unlockHeight uint32, scriptPubKey []byte, amount int64) (*psbt.Packet, error) {
	// Create a new PSBT
	hash, err := chainhash.NewHashFromStr(utxo.Txid)
	if err != nil {
		log.Fatalf("Invalid hash: %v", err)
	}
	TxIn := wire.OutPoint{
		Hash:  *hash,
		Index: uint32(utxo.Vout),
	}
	TxOut := []*wire.TxOut{}
	for _, output := range outputs {
		fmt.Println("output : ", output)
		for addr, amount := range output {
			fmt.Println("amount : ", amount)
			address, err := btcutil.DecodeAddress(addr, &chaincfg.MainNetParams)
			if err != nil {
				return nil, err
			}

			// script, err := txscript.PayToAddrScript(address)
			// if err != nil {
			// 	fmt.Println(err)
			// 	return nil, err
			// }

			TxOut = append(TxOut, wire.NewTxOut(int64(amount), address.ScriptAddress()))
		}
	}

	fmt.Println("TxOut: ", TxOut)
	fmt.Println("TxIn: ", TxIn)

	packet, err := psbt.New([]*wire.OutPoint{&TxIn}, TxOut, 2, 0, []uint32{unlockHeight})
	if err != nil {
		return nil, err
	}

	packet.Inputs[0].WitnessUtxo = &wire.TxOut{
		PkScript: scriptPubKey,
		Value:    amount,
	}

	return packet, nil
}

func GetAddressInfo(address string, wallet string) (AddressInfo, error) {
	data := []interface{}{address}
	var response JSONRPCResponseAddressInfo
	result, err := SendRPC("getaddressinfo", data, wallet)
	if err != nil {
		fmt.Println("error getting Address info : ", err)
		return AddressInfo{}, err
	}

	err = json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return AddressInfo{}, err
	}
	if response.Error != nil {
		return AddressInfo{}, errors.New("error in getting address info")
	}
	return response.Result, nil
}

func GetEstimateFee(wallet string) (EstimateFeeResponse, error) {
	data := []interface{}{3}
	var response EstimateFeeResponse
	result, err := SendRPC("estimatesmartfee", data, wallet)
	if err != nil {
		fmt.Println("error getting  estimate fee info : ", err)
		return EstimateFeeResponse{}, err
	}

	err = json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return EstimateFeeResponse{}, err
	}
	if response.Error != nil {
		return EstimateFeeResponse{}, errors.New("error in GetEstimateFee")
	}
	return response, nil
}

func SendToAddress(address string, fee float64, wallet string) (TransactionInfo, error) {
	var response TransactionInfo
	for i := 0; i < 5; i++ {
		data := []interface{}{address, fee + 0.000001}
		result, err := SendRPC("sendtoaddress", data, wallet)
		fmt.Println("result Send to Address: ", string(result))
		if err != nil {
			fmt.Println("error creating utxo for fee : ", err)
			return TransactionInfo{}, err
		}
		if strings.Contains(string(result), "Transaction amount too small") {
			continue
		}
		err = json.Unmarshal(result, &response)
		if err != nil {
			fmt.Println("Error unmarshalling JSON: ", err)
			return TransactionInfo{}, err
		}
		if response.Error != nil {
			return TransactionInfo{}, errors.New("error in sending to address")
		}
		return response, nil
	}
	return TransactionInfo{}, errors.New("error in sending to address 5 tries")
}

func SignRawTransaction(tx string, wallet string) (string, error) {
	data := []interface{}{tx, nil, "ALL|ANYONECANPAY"}
	var response SignedTransactionInfo
	result, err := SendRPC("signrawtransactionwithwallet", data, wallet)
	if err != nil {
		fmt.Println("error creating utxo for fee : ", err)
		return "", err
	}

	err = json.Unmarshal(result, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
		return "", err
	}
	if response.Error != nil {
		return "", errors.New("error in signing raw transaction")
	}
	return response.Result.Hex, nil
}
