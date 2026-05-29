package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strconv"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"

	"github.com/twilight-project/forkoracle-go/address"
	"github.com/twilight-project/forkoracle-go/comms"
	db "github.com/twilight-project/forkoracle-go/db"
	"github.com/twilight-project/forkoracle-go/judge"
	"github.com/twilight-project/forkoracle-go/transaction_signer"
	btcOracleTypes "github.com/twilight-project/forkoracle-go/types"
	utils "github.com/twilight-project/forkoracle-go/utils"
	bridgetypes "twilight-project/nyks/x/bridge/types"
)

const (
	HandlerSigningRefund       = "signing_refund"
	HandlerSigningSweep        = "signing_sweep"
	HandlerProposeAddress      = "propose_address"
	HandlerProcessSweep        = "process_sweep"
	HandlerProcessSweepDirect  = "process_sweep_direct"
	HandlerSweepProposal       = "sweep_proposal"
)

var (
	handler        string
	runAll         bool
	verbose        bool
	showHelp       bool
	reserveId      int
	roundId        int
	newAddress     string
	btcTxHash      string
	btcBlockHeight int
)

func init() {
	flag.StringVar(&handler, "handler", "", "Handler to trigger: signing_refund, signing_sweep, or propose_address")
	flag.BoolVar(&runAll, "all", false, "Run all signer handlers (signing_refund then signing_sweep)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.IntVar(&reserveId, "reserve-id", 0, "Reserve ID (required for propose_address and process_sweep_direct)")
	flag.IntVar(&roundId, "round-id", 0, "Round ID (required for process_sweep_direct, sweep_proposal)")
	flag.StringVar(&newAddress, "new-address", "", "New reserve BTC address (required for sweep_proposal)")
	flag.StringVar(&btcTxHash, "btc-tx-hash", "", "BTC transaction hash (required for sweep_proposal)")
	flag.IntVar(&btcBlockHeight, "btc-block-height", 0, "BTC block height (required for sweep_proposal)")
}

func main() {
	flag.Parse()

	if showHelp {
		printUsage()
		os.Exit(0)
	}

	// Validate arguments
	if !runAll && handler == "" {
		fmt.Println("Error: Must specify --handler or --all")
		printUsage()
		os.Exit(1)
	}

	validHandlers := map[string]bool{
		HandlerSigningRefund:      true,
		HandlerSigningSweep:       true,
		HandlerProposeAddress:     true,
		HandlerProcessSweep:       true,
		HandlerProcessSweepDirect: true,
		HandlerSweepProposal:      true,
	}

	if handler != "" && !validHandlers[handler] {
		fmt.Printf("Error: Invalid handler '%s'. Must be one of: signing_refund, signing_sweep, propose_address, process_sweep, process_sweep_direct, sweep_proposal\n", handler)
		os.Exit(1)
	}

	// Initialize (reusing pattern from main.go)
	accountName, oracleAddr, dbconn := initialize()
	defer dbconn.Close()

	// Route based on handler type - judge role handlers
	if handler == HandlerProposeAddress {
		runProposeAddress(accountName, oracleAddr, dbconn)
		return
	}
	if handler == HandlerProcessSweep {
		fmt.Println("[MANUAL-TRIGGER] Executing handler: process_sweep")
		judge.ProcessSweep(accountName, dbconn, oracleAddr)
		fmt.Println("[MANUAL-TRIGGER] Handler process_sweep completed")
		return
	}
	if handler == HandlerProcessSweepDirect {
		runProcessSweepDirect(accountName, oracleAddr, dbconn)
		return
	}
	if handler == HandlerSweepProposal {
		runSweepProposal(accountName, oracleAddr)
		return
	}

	// Signer handlers
	if err := validateSignerRole(oracleAddr); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	address.RegisterAddressOnSigners(dbconn)

	if runAll {
		fmt.Println("Running all signer handlers...")
		runHandler(HandlerSigningRefund, accountName, dbconn, oracleAddr)
		runHandler(HandlerSigningSweep, accountName, dbconn, oracleAddr)
	} else {
		runHandler(handler, accountName, dbconn, oracleAddr)
	}

	fmt.Println("Manual trigger completed successfully")
}

func initialize() (string, string, *sql.DB) {
	fmt.Println("[DEBUG] Loading config file...")
	utils.InitConfigFile()
	fmt.Println("[DEBUG] Config loaded. Getting fragments...")
	comms.GetAllFragments()
	fmt.Println("[DEBUG] Fragments retrieved. Initializing DB...")
	dbconn := db.InitDB()
	fmt.Println("[DEBUG] DB initialized. Reading config values...")

	oracleAddr := viper.GetString("own_address")
	accountName := viper.GetString("accountName")
	walletName := viper.GetString("wallet_name")

	fmt.Printf("[DEBUG] accountName: %s, oracleAddr: %s, wallet: %s\n", accountName, oracleAddr, walletName)

	fmt.Printf("[DEBUG] Loading BTC wallet: %s ...\n", walletName)
	utils.LoadBtcWallet(walletName)
	feeWalletName := viper.GetString("fee_wallet_name")
	fmt.Printf("[DEBUG] Loading fee wallet: %s ...\n", feeWalletName)
	utils.LoadBtcWallet(feeWalletName)
	fmt.Println("[DEBUG] BTC wallets loaded.")

	if verbose {
		fmt.Printf("Initialized with:\n")
		fmt.Printf("  Account Name: %s\n", accountName)
		fmt.Printf("  Oracle Address: %s\n", oracleAddr)
		fmt.Printf("  Running Mode: %s\n", viper.GetString("running_mode"))
	}

	return accountName, oracleAddr, dbconn
}

func validateSignerRole(signerAddr string) error {
	judgeAddress := viper.GetString("judge_address")
	fragments := comms.GetAllFragments()

	var fragment btcOracleTypes.Fragment
	found := false
	for _, f := range fragments.Fragments {
		if f.JudgeAddress == judgeAddress {
			fragment = f
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("no fragment found with judge address: %s", judgeAddress)
	}

	for _, signer := range fragment.Signers {
		if signer.SignerAddress == signerAddr {
			return nil
		}
	}

	return fmt.Errorf("signer %s is not registered with judge %s", signerAddr, judgeAddress)
}

func runProposeAddress(accountName string, judgeAddr string, dbconn *sql.DB) {
	if reserveId <= 0 {
		fmt.Println("Error: --reserve-id is required for propose_address handler")
		os.Exit(1)
	}

	// Validate judge has a fragment
	fragments := comms.GetAllFragments()
	found := false
	for _, f := range fragments.Fragments {
		if f.JudgeAddress == judgeAddr {
			found = true
			break
		}
	}
	if !found {
		fmt.Printf("Error: No fragment found with judge address: %s\n", judgeAddr)
		os.Exit(1)
	}

	// Fetch reserves from chain and find the matching one
	btcReserves := comms.GetBtcReserves()
	var reserve btcOracleTypes.BtcReserve
	found = false
	for _, r := range btcReserves.BtcReserves {
		rid, _ := strconv.Atoi(r.ReserveId)
		if rid == reserveId {
			reserve = r
			found = true
			break
		}
	}
	if !found {
		fmt.Printf("Error: Reserve with ID %d not found on chain\n", reserveId)
		os.Exit(1)
	}

	roundId, _ := strconv.Atoi(reserve.RoundId)
	nextRound := uint64(roundId + 1)

	fmt.Printf("[DEBUG] Reserve ID: %d, Current Round: %d, Next Round: %d\n", reserveId, roundId, nextRound)
	fmt.Printf("[DEBUG] Current Reserve Address: %s\n", reserve.ReserveAddress)

	// Check if already proposed
	proposed := db.CheckIfAddressIsProposed(dbconn, int64(nextRound), uint64(reserveId))
	if proposed {
		fmt.Printf("Address already proposed for reserve %d, round %d\n", reserveId, nextRound)
		return
	}

	fmt.Printf("Proposing address for reserve %d, round %d\n", reserveId, nextRound)
	address.ProposeAddress(accountName, uint64(reserveId), nextRound, reserve.ReserveAddress, judgeAddr, dbconn)
	fmt.Println("Manual trigger completed successfully")
}

func runHandler(handlerName string, accountName string, dbconn *sql.DB, signerAddr string) {
	fmt.Printf("Executing handler: %s\n", handlerName)

	switch handlerName {
	case HandlerSigningRefund:
		transaction_signer.ProcessTxSigningRefund(accountName, dbconn, signerAddr)
	case HandlerSigningSweep:
		transaction_signer.ProcessTxSigningSweep(accountName, dbconn, signerAddr)
	default:
		fmt.Printf("Unknown handler: %s\n", handlerName)
	}

	fmt.Printf("Handler %s completed\n", handlerName)
}

func runProcessSweepDirect(accountName string, judgeAddr string, dbconn *sql.DB) {
	if reserveId <= 0 {
		fmt.Println("Error: --reserve-id is required for process_sweep_direct handler")
		os.Exit(1)
	}
	if roundId <= 0 {
		fmt.Println("Error: --round-id is required for process_sweep_direct handler")
		os.Exit(1)
	}

	fmt.Printf("[MANUAL-TRIGGER] process_sweep_direct: reserve-id=%d, round-id=%d\n", reserveId, roundId)

	// Step 1: Fetch reserve from chain
	fmt.Println("[DEBUG] Fetching BTC reserves from chain...")
	btcReserves := comms.GetBtcReserves()
	var reserve btcOracleTypes.BtcReserve
	found := false
	for _, r := range btcReserves.BtcReserves {
		rid, _ := strconv.Atoi(r.ReserveId)
		if rid == reserveId {
			reserve = r
			found = true
			break
		}
	}
	if !found {
		fmt.Printf("[ERROR] Reserve with ID %d not found on chain\n", reserveId)
		os.Exit(1)
	}
	fmt.Printf("[DEBUG] Found reserve: ID=%s, Address=%s, RoundId=%s\n", reserve.ReserveId, reserve.ReserveAddress, reserve.RoundId)

	// Step 2: Query sweep address from DB (no height restriction)
	fmt.Printf("[DEBUG] Querying sweep address from DB for: %s\n", reserve.ReserveAddress)
	addresses := db.QuerySweepAddress(dbconn, reserve.ReserveAddress)
	if len(addresses) <= 0 {
		fmt.Printf("[ERROR] No sweep address found in DB for: %s\n", reserve.ReserveAddress)
		os.Exit(1)
	}
	currentSweepAddress := addresses[0]
	fmt.Printf("[DEBUG] Sweep address found: %s, unlock_height=%d\n", currentSweepAddress.Address, currentSweepAddress.Unlock_height)

	// Step 3: Query UTXOs
	fmt.Printf("[DEBUG] Querying UTXOs for address: %s\n", currentSweepAddress.Address)
	utxos := db.QueryUtxo(dbconn, currentSweepAddress.Address)
	if len(utxos) <= 0 {
		fmt.Printf("[ERROR] No UTXOs found for address: %s\n", currentSweepAddress.Address)
		os.Exit(1)
	}
	fmt.Printf("[DEBUG] Found %d UTXOs\n", len(utxos))
	for i, u := range utxos {
		fmt.Printf("[DEBUG]   UTXO[%d]: txid=%s, vout=%d, amount=%d\n", i, u.Txid, u.Vout, u.Amount)
	}

	// Step 4: Get proposed sweep address for next round
	fmt.Printf("[DEBUG] Getting proposed sweep address for reserve=%d, round=%d\n", reserveId, roundId)
	sweepAddressResp := comms.GetProposedSweepAddress(uint64(reserveId), uint64(roundId))
	if sweepAddressResp.ProposeSweepAddressMsg.BtcAddress == "" {
		fmt.Printf("[ERROR] No proposed sweep address found for reserve=%d, round=%d\n", reserveId, roundId)
		os.Exit(1)
	}
	newSweepAddress := sweepAddressResp.ProposeSweepAddressMsg.BtcAddress
	fmt.Printf("[DEBUG] New sweep address: %s\n", newSweepAddress)

	// Step 5: Get withdraw requests
	fmt.Printf("[DEBUG] Getting withdraw snapshot for reserve=%d, round=%d\n", reserveId, roundId)
	withdrawRequests := comms.GetWithdrawSnapshot(uint64(reserveId), uint64(roundId)).WithdrawRequests
	fmt.Printf("[DEBUG] Found %d withdraw requests\n", len(withdrawRequests))
	for i, w := range withdrawRequests {
		fmt.Printf("[DEBUG]   Withdraw[%d]: addr=%s, amount=%s\n", i, w.WithdrawAddress, w.WithdrawAmount)
	}

	// Step 6: Generate sweep tx
	fmt.Println("[DEBUG] Generating sweep transaction...")
	sweepTxHex, psbt, sweepTxId, totalAmount, _, err := judge.GenerateSweepTx(
		currentSweepAddress.Address, newSweepAddress, accountName,
		withdrawRequests, int64(currentSweepAddress.Unlock_height), utxos, dbconn,
	)
	if err != nil {
		fmt.Printf("[ERROR] Failed to generate sweep tx: %v\n", err)
		os.Exit(1)
	}
	if sweepTxHex == "" {
		fmt.Println("[ERROR] No sweep tx generated (no funds in current address)")
		os.Exit(1)
	}
	fmt.Printf("[DEBUG] Sweep tx generated: txId=%s, totalAmount=%d\n", sweepTxId, totalAmount)

	// Step 7: Broadcast to chain
	fmt.Println("[DEBUG] Broadcasting unsigned sweep tx to chain...")
	cosmos := comms.GetCosmosClient()
	msg := &bridgetypes.MsgUnsignedTxSweep{TxId: sweepTxId, BtcUnsignedSweepTx: psbt, ReserveId: uint64(reserveId), RoundId: uint64(roundId), JudgeAddress: judgeAddr}
	comms.SendTransactionUnsignedSweepTx(accountName, cosmos, msg)
	fmt.Println("[DEBUG] Broadcast complete")

	// Step 8: Record in DB
	fmt.Println("[DEBUG] Recording in DB...")
	db.InsertUnSignedSweeptx(dbconn, sweepTxHex, int64(reserveId), int64(roundId))
	db.MarkAddressArchived(dbconn, currentSweepAddress.Address)
	fmt.Println("[MANUAL-TRIGGER] process_sweep_direct completed successfully")
}

func runSweepProposal(accountName string, oracleAddr string) {
	if reserveId <= 0 {
		fmt.Println("Error: --reserve-id is required for sweep_proposal handler")
		os.Exit(1)
	}
	if roundId <= 0 {
		fmt.Println("Error: --round-id is required for sweep_proposal handler")
		os.Exit(1)
	}
	if newAddress == "" {
		fmt.Println("Error: --new-address is required for sweep_proposal handler")
		os.Exit(1)
	}
	if btcTxHash == "" {
		fmt.Println("Error: --btc-tx-hash is required for sweep_proposal handler")
		os.Exit(1)
	}
	if btcBlockHeight <= 0 {
		fmt.Println("Error: --btc-block-height is required for sweep_proposal handler")
		os.Exit(1)
	}

	// Fetch reserve from chain to get judge address
	fmt.Println("[DEBUG] Fetching BTC reserves from chain...")
	btcReserves := comms.GetBtcReserves()
	var reserve btcOracleTypes.BtcReserve
	found := false
	for _, r := range btcReserves.BtcReserves {
		rid, _ := strconv.Atoi(r.ReserveId)
		if rid == reserveId {
			reserve = r
			found = true
			break
		}
	}
	if !found {
		fmt.Printf("[ERROR] Reserve with ID %d not found on chain\n", reserveId)
		os.Exit(1)
	}

	fmt.Println("[MANUAL-TRIGGER] Sending sweep proposal message")
	fmt.Printf("[DEBUG] ReserveId: %d\n", reserveId)
	fmt.Printf("[DEBUG] NewReserveAddress: %s\n", newAddress)
	fmt.Printf("[DEBUG] JudgeAddress: %s\n", reserve.JudgeAddress)
	fmt.Printf("[DEBUG] BtcTxHash: %s\n", btcTxHash)
	fmt.Printf("[DEBUG] UnlockHeight: %d\n", btcBlockHeight)
	fmt.Printf("[DEBUG] RoundId: %d\n", roundId)
	fmt.Printf("[DEBUG] OracleAddress: %s\n", oracleAddr)

	cosmos := comms.GetCosmosClient()
	msg := &bridgetypes.MsgSweepProposal{
		ReserveId:             uint64(reserveId),
		NewReserveAddress:     newAddress,
		JudgeAddress:          reserve.JudgeAddress,
		BtcRelayCapacityValue: 0,
		BtcTxHash:             btcTxHash,
		UnlockHeight:          uint64(btcBlockHeight),
		RoundId:               uint64(roundId),
		BtcBlockNumber:        uint64(btcBlockHeight),
		OracleAddress:         oracleAddr,
	}

	comms.SendTransactionSweepProposal(accountName, cosmos, msg)
	fmt.Println("[MANUAL-TRIGGER] Sweep proposal sent successfully")
}

func printUsage() {
	fmt.Println("manual-trigger - Manually trigger btc-oracle event handlers")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  manual-trigger --handler=<handler_name>")
	fmt.Println("  manual-trigger --all")
	fmt.Println()
	fmt.Println("Handlers:")
	fmt.Println("  signing_refund   - Process unsigned refund transactions (signer role)")
	fmt.Println("  signing_sweep    - Process unsigned sweep transactions (signer role)")
	fmt.Println("  propose_address  - Propose a new reserve address (judge role, requires --reserve-id)")
	fmt.Println("  process_sweep    - Process sweep transactions (judge role)")
	fmt.Println("  process_sweep_direct - Process sweep for old blocks, bypasses height window (judge role, requires --reserve-id and --round-id)")
	fmt.Println("  sweep_proposal       - Send sweep proposal message (requires --reserve-id, --round-id, --new-address, --btc-tx-hash, --btc-block-height)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  manual-trigger --handler=signing_refund")
	fmt.Println("  manual-trigger --handler=signing_sweep --verbose")
	fmt.Println("  manual-trigger --handler=propose_address --reserve-id=1")
	fmt.Println("  manual-trigger --handler=process_sweep")
	fmt.Println("  manual-trigger --handler=process_sweep_direct --reserve-id=1 --round-id=5")
	fmt.Println("  manual-trigger --handler=sweep_proposal --reserve-id=1 --round-id=10 --new-address=bc1q... --btc-tx-hash=abc123... --btc-block-height=935821")
	fmt.Println("  manual-trigger --all")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
}
