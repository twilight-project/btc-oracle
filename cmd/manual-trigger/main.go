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
	"github.com/twilight-project/forkoracle-go/transaction_signer"
	btcOracleTypes "github.com/twilight-project/forkoracle-go/types"
	utils "github.com/twilight-project/forkoracle-go/utils"
)

const (
	HandlerSigningRefund  = "signing_refund"
	HandlerSigningSweep   = "signing_sweep"
	HandlerProposeAddress = "propose_address"
)

var (
	handler   string
	runAll    bool
	verbose   bool
	showHelp  bool
	reserveId int
)

func init() {
	flag.StringVar(&handler, "handler", "", "Handler to trigger: signing_refund, signing_sweep, or propose_address")
	flag.BoolVar(&runAll, "all", false, "Run all signer handlers (signing_refund then signing_sweep)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.IntVar(&reserveId, "reserve-id", 0, "Reserve ID (required for propose_address handler)")
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
		HandlerSigningRefund:  true,
		HandlerSigningSweep:   true,
		HandlerProposeAddress: true,
	}

	if handler != "" && !validHandlers[handler] {
		fmt.Printf("Error: Invalid handler '%s'. Must be one of: signing_refund, signing_sweep, propose_address\n", handler)
		os.Exit(1)
	}

	// Initialize (reusing pattern from main.go)
	accountName, oracleAddr, dbconn := initialize()
	defer dbconn.Close()

	// Route based on handler type
	if handler == HandlerProposeAddress {
		runProposeAddress(accountName, oracleAddr, dbconn)
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
	fmt.Println("[DEBUG] BTC wallet loaded.")

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
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  manual-trigger --handler=signing_refund")
	fmt.Println("  manual-trigger --handler=signing_sweep --verbose")
	fmt.Println("  manual-trigger --handler=propose_address --reserve-id=1")
	fmt.Println("  manual-trigger --all")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
}
