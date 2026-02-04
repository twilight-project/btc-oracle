package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

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
	HandlerSigningRefund = "signing_refund"
	HandlerSigningSweep  = "signing_sweep"
)

var (
	handler  string
	runAll   bool
	verbose  bool
	showHelp bool
)

func init() {
	flag.StringVar(&handler, "handler", "", "Handler to trigger: signing_refund or signing_sweep")
	flag.BoolVar(&runAll, "all", false, "Run all signer handlers (signing_refund then signing_sweep)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
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

	if handler != "" && handler != HandlerSigningRefund && handler != HandlerSigningSweep {
		fmt.Printf("Error: Invalid handler '%s'. Must be '%s' or '%s'\n",
			handler, HandlerSigningRefund, HandlerSigningSweep)
		os.Exit(1)
	}

	// Initialize (reusing pattern from main.go)
	accountName, signerAddr, dbconn := initialize()
	defer dbconn.Close()

	// Validate signer role
	if err := validateSignerRole(signerAddr); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Register addresses on signers (as done in startTransactionSigner)
	address.RegisterAddressOnSigners(dbconn)

	// Execute handlers
	if runAll {
		fmt.Println("Running all signer handlers...")
		runHandler(HandlerSigningRefund, accountName, dbconn, signerAddr)
		runHandler(HandlerSigningSweep, accountName, dbconn, signerAddr)
	} else {
		runHandler(handler, accountName, dbconn, signerAddr)
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

	signerAddr := viper.GetString("own_address")
	accountName := viper.GetString("accountName")
	walletName := viper.GetString("wallet_name")

	fmt.Printf("[DEBUG] accountName: %s, signerAddr: %s, wallet: %s\n", accountName, signerAddr, walletName)

	// Validate running mode (warn if not signer)
	runningMode := viper.GetString("running_mode")
	if runningMode != "signer" {
		fmt.Printf("Warning: running_mode is '%s', expected 'signer'. Proceeding anyway.\n", runningMode)
	}

	fmt.Printf("[DEBUG] Loading BTC wallet: %s ...\n", walletName)
	utils.LoadBtcWallet(walletName)
	fmt.Println("[DEBUG] BTC wallet loaded.")

	if verbose {
		fmt.Printf("Initialized with:\n")
		fmt.Printf("  Account Name: %s\n", accountName)
		fmt.Printf("  Signer Address: %s\n", signerAddr)
		fmt.Printf("  Running Mode: %s\n", runningMode)
	}

	return accountName, signerAddr, dbconn
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
	fmt.Println("Handlers (for signer role):")
	fmt.Println("  signing_refund  - Process unsigned refund transactions (triggered by unsigned_tx_refund event)")
	fmt.Println("  signing_sweep   - Process unsigned sweep transactions (triggered by broadcast_tx_refund event)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  manual-trigger --handler=signing_refund")
	fmt.Println("  manual-trigger --handler=signing_sweep --verbose")
	fmt.Println("  manual-trigger --all")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
}
