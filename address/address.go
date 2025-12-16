package address

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	comms "github.com/twilight-project/forkoracle-go/comms"
	db "github.com/twilight-project/forkoracle-go/db"
	btcOracleTypes "github.com/twilight-project/forkoracle-go/types"
	utils "github.com/twilight-project/forkoracle-go/utils"
	bridgetypes "github.com/twilight-project/nyks/x/bridge/types"

	"golang.org/x/crypto/ripemd160"
)

func hash160(data []byte) []byte {
	hash := sha256.Sum256(data)
	ripemdHash := ripemd160.New()
	_, err := ripemdHash.Write(hash[:])
	if err != nil {
		fmt.Println("Error computing hash:", err)
		return nil
	}
	hash160 := ripemdHash.Sum(nil)
	return hash160
}

func Preimage() ([]byte, error) {
	preimage := make([]byte, 32)
	_, err := rand.Read(preimage)
	if err != nil {
		fmt.Println("Error generating preimage:", err)
		return nil, err
	}
	return preimage, nil
}

func buildDescriptor(preimage []byte, unlockHeight int64, judgeAddr string) (string, error) {
	var fragment btcOracleTypes.Fragment
	fragments := comms.GetAllFragments()
	for _, f := range fragments.Fragments {
		if f.JudgeAddress == judgeAddr {
			fragment = f
		}
	}
	signers := fragment.Signers

	required := len(signers) * 2 / 3
	if required == 0 {
		required = 1
	}
	multiscript_params := fmt.Sprintf("%d", required)
	for _, signer := range signers {
		multiscript_params += fmt.Sprintf(",%s", signer.SignerBtcPublicKey)
	}
	descriptorScript := fmt.Sprintf("wsh(and_v(v:multi(%s),after(%d)))", multiscript_params, unlockHeight)

	fmt.Println(descriptorScript)
	return descriptorScript, nil
}

func GenerateAddress(unlock_height int64, oldReserveAddress string, judgeAddr string, dbconn *sql.DB) (string, string) {
	wallet := viper.GetString("wallet_name")
	preimage, err := Preimage()
	if err != nil {
		fmt.Println(err)
	}
	descriptor, err := buildDescriptor(preimage, unlock_height, judgeAddr)
	if err != nil {
		fmt.Println(err)
	}

	resp, err := comms.GetDescriptorInfo(descriptor, wallet)
	if err != nil {
		fmt.Println("error in getting descriptorinfo : ", err)
		return "", ""
	}

	fmt.Println("Descriptor : ", resp.Descriptor)

	err = comms.ImportDescriptor(resp.Descriptor, wallet)
	if err != nil {
		fmt.Println("error in importing descriptor : ", err)
	}

	address, err := comms.GetNewAddress(wallet)
	if err != nil {
		fmt.Println("error in getting address : ", err)
	}

	addressInfo, err := comms.GetAddressInfo(address, wallet)
	if err != nil {
		fmt.Println("Error getting address info : ", err)
		return "", ""
	}
	// Decode Hex string to bytes

	db.InsertSweepAddress(dbconn, address, addressInfo.Hex, preimage, int64(unlock_height), oldReserveAddress, true)

	return address, addressInfo.Hex
}

func proposeAddress(accountName string, reserveId uint64, roundId uint64, oldAddress string, judgeAddr string, dbconn *sql.DB) {
	number := fmt.Sprintf("%v", viper.Get("unlocking_time"))
	wallet := viper.GetString("wallet_name")
	unlockingTime, _ := strconv.Atoi(number)

	var lastSweepAddress btcOracleTypes.SweepAddress
	addresses := db.QuerySweepAddressesOrderByHeight(dbconn, 1)
	if len(addresses) == 0 {
		fmt.Println("address proposer : no Sweep address found")
		return
	}

	lastSweepAddress = addresses[0]

	unlockHeight := lastSweepAddress.Unlock_height + int64(unlockingTime)
	newReserveAddress := GenerateAndRegisterNewProposedAddress(dbconn, accountName, unlockHeight, oldAddress, judgeAddr)

	addressInfo, err := comms.GetAddressInfo(newReserveAddress, wallet)
	if err != nil {
		fmt.Println("error getting address info : ", err)
		return
	}

	fmt.Println("proposing address : ", newReserveAddress)
	fmt.Println("roundId : ", roundId)
	fmt.Println("reserveId : ", reserveId)

	cosmos_client := comms.GetCosmosClient()
	msg := &bridgetypes.MsgProposeSweepAddress{
		BtcScript:    addressInfo.Hex,
		BtcAddress:   newReserveAddress,
		JudgeAddress: judgeAddr,
		ReserveId:    reserveId,
		RoundId:      roundId,
	}

	comms.SendTransactionSweepAddressProposal(accountName, cosmos_client, msg)
	db.InsertProposedAddress(dbconn, oldAddress, newReserveAddress, unlockHeight, int64(roundId), int64(reserveId))

	fmt.Println("finishing propose Address after proposing")
}

func ProcessProposeAddress(accountName string, judgeAddr string, dbconn *sql.DB) {
	fmt.Println("Process propose address started")
	number := fmt.Sprintf("%v", viper.Get("sweep_preblock"))
	sweepInitateBlockHeight, _ := strconv.Atoi(number)

	for {

		resp := comms.GetAttestations("20")
		if len(resp.Attestations) <= 0 {
			time.Sleep(1 * time.Minute)
			fmt.Println("no attestaions (start judge)")
			continue
		}

		btcReserves := comms.GetBtcReserves()

		for _, attestation := range resp.Attestations {
			height, _ := strconv.Atoi(attestation.Proposal.Height)
			if !attestation.Observed {
				continue
			}

			addresses := db.QuerySweepAddressesByHeight(dbconn, uint64(height+sweepInitateBlockHeight), true)
			if len(addresses) <= 0 {
				continue
			}

			var reserve btcOracleTypes.BtcReserve
			for _, r := range btcReserves.BtcReserves {
				if r.ReserveAddress == addresses[0].Address {
					reserve = r
				}
			}
			roundId, _ := strconv.Atoi(reserve.RoundId)
			reserveId, _ := strconv.Atoi(reserve.ReserveId)

			proposed := db.CheckIfAddressIsProposed(dbconn, int64(roundId+1), uint64(reserveId))
			if proposed {
				break
			}
			fmt.Println("Sweep Address found, proposing address for reserve and round : ", reserveId, roundId+1)
			proposeAddress(accountName, uint64(reserveId), uint64(roundId+1), reserve.ReserveAddress, judgeAddr, dbconn)
			break
		}

	}
}

func GenerateAndRegisterNewProposedAddress(dbconn *sql.DB, accountName string, height int64, oldReserveAddress string, oracleAddr string) string {
	newSweepAddress, _ := GenerateAddress(height, oldReserveAddress, oracleAddr, dbconn)
	registerAddressOnForkscanner(newSweepAddress)
	return newSweepAddress
}

func GenerateAndRegisterNewBtcReserveAddress(dbconn *sql.DB, accountName string, height int64, judgeAddr string, fragmentId int) string {
	newSweepAddress, script := GenerateAddress(height, "", judgeAddr, dbconn)
	registerReserveAddressOnNyks(accountName, newSweepAddress, script, judgeAddr, fragmentId)
	registerAddressOnForkscanner(newSweepAddress)

	return newSweepAddress
}

func registerReserveAddressOnNyks(accountName string, address string, script string, judgeAddr string, fragmentId int) {

	cosmos := comms.GetCosmosClient()

	msg := &bridgetypes.MsgRegisterReserveAddress{
		FragmentId:     uint64(fragmentId),
		ReserveScript:  script,
		ReserveAddress: address,
		JudgeAddress:   judgeAddr,
	}

	// store response in txResp
	txResp, err := comms.SendTransactionRegisterReserveAddress(accountName, cosmos, msg)
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

	resp, err := http.Post("http://"+viper.GetString("forkscanner_host")+":"+viper.GetString("forkscanner_rpc_port"), "application/json", strings.NewReader(string(data)))
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

func UnRegisterAddressOnForkscanner(address string) {
	dt := time.Now().UTC()
	dt = dt.AddDate(1, 0, 0)

	request_body := map[string]interface{}{
		"method":  "remove_watched_addresses",
		"id":      1,
		"jsonrpc": "2.0",
		"params": map[string]interface{}{
			"remove": []string{address},
		},
	}

	data, err := json.Marshal(request_body)
	if err != nil {
		log.Fatalf("Post: %v", err)
	}
	fmt.Println(string(data))

	resp, err := http.Post("http://"+viper.GetString("forkscanner_host")+":"+viper.GetString("forkscanner_rpc_port"), "application/json", strings.NewReader(string(data)))
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

func RegisterAddressOnValidators(dbconn *sql.DB) {
	// {add check to see if the address already exists}
	fmt.Println("registering address on validators")
	savedAddress := db.QueryAllAddressOnly(dbconn)
	respReserve := comms.GetReserveAddresses()
	if len(respReserve.Addresses) > 0 {
		for _, address := range respReserve.Addresses {
			if !utils.StringInSlice(address.ReserveAddress, savedAddress) {
				registerAddressOnForkscanner(address.ReserveAddress)
				decodedScript := utils.DecodeBtcScript(address.ReserveScript)
				height := utils.GetUnlockHeightFromScript(decodedScript)
				db.InsertSweepAddress(dbconn, address.ReserveAddress, address.ReserveScript, nil, height, "", false)
			}
		}
	}
	respProposed := comms.GetProposedAddresses()
	if len(respProposed.ProposeSweepAddressMsgs) > 0 {
		for _, address := range respProposed.ProposeSweepAddressMsgs {
			if !utils.StringInSlice(address.BtcAddress, savedAddress) {
				registerAddressOnForkscanner(address.BtcAddress)
				decodedScript := utils.DecodeBtcScript(address.BtcScript)
				height := utils.GetUnlockHeightFromScript(decodedScript)
				db.InsertSweepAddress(dbconn, address.BtcAddress, address.BtcScript, nil, height, "", false)
			}
		}
	}
}

func RegisterAddressOnSigners(dbconn *sql.DB) {
	// {add check to see if the address already exists}
	fmt.Println("registering address on signers")
	savedAddress := db.QueryAllAddressOnly(dbconn)
	respReserve := comms.GetReserveAddresses()
	if len(respReserve.Addresses) > 0 {
		for _, address := range respReserve.Addresses {
			if !utils.StringInSlice(address.ReserveAddress, savedAddress) {
				decodedScript := utils.DecodeBtcScript(address.ReserveScript)
				height := utils.GetUnlockHeightFromScript(decodedScript)
				db.InsertSweepAddress(dbconn, address.ReserveAddress, address.ReserveScript, nil, height+1, "", false)
			}
		}
	}
	respProposed := comms.GetProposedAddresses()
	if len(respProposed.ProposeSweepAddressMsgs) > 0 {
		for _, address := range respProposed.ProposeSweepAddressMsgs {
			if !utils.StringInSlice(address.BtcAddress, savedAddress) {
				decodedScript := utils.DecodeBtcScript(address.BtcScript)
				height := utils.GetUnlockHeightFromScript(decodedScript)
				db.InsertSweepAddress(dbconn, address.BtcAddress, address.BtcScript, nil, height+1, "", false)
			}
		}
	}
}
