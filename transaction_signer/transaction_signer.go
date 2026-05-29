package transaction_signer

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/viper"
	comms "github.com/twilight-project/forkoracle-go/comms"
	db "github.com/twilight-project/forkoracle-go/db"
	btcOracleTypes "github.com/twilight-project/forkoracle-go/types"
	"github.com/twilight-project/forkoracle-go/utils"
	bridgetypes "twilight-project/nyks/x/bridge/types"
)

func ProcessTxSigningSweep(accountName string, dbconn *sql.DB, signerAddr string) {
	fmt.Println("starting Sweep Tx Signer")
	wallet := viper.GetString("wallet_name")
	btcPubKey := viper.GetString("btc_public_key")
	fmt.Printf("[DEBUG] wallet: %s, btcPubKey: %s, signerAddr: %s\n", wallet, btcPubKey, signerAddr)

	SweepTxs := comms.GetAllUnsignedSweepTx()
	fmt.Printf("[DEBUG] Found %d unsigned sweep transactions\n", len(SweepTxs.UnsignedTxSweepMsgs))

	for i, tx := range SweepTxs.UnsignedTxSweepMsgs {
		reserveId, _ := strconv.Atoi(tx.ReserveId)
		roundId, _ := strconv.Atoi(tx.RoundId)
		fmt.Printf("\n[DEBUG] Processing sweep tx %d/%d: reserveId=%d, roundId=%d, judgeAddr=%s\n",
			i+1, len(SweepTxs.UnsignedTxSweepMsgs), reserveId, roundId, tx.JudgeAddress)

		for {
			correspondingRefundTx := comms.GetBroadCastedRefundTx(uint64(reserveId), uint64(roundId))
			if correspondingRefundTx.ReserveId != "" {
				break
			}
			fmt.Printf("corresponding refund tx not found  reserve Id: %d roundid : %d\n", reserveId, roundId)
			time.Sleep(1 * time.Minute)
		}

		fmt.Printf("corresponding refund tx found  reserve Id: %d roundid : %d\n", reserveId, roundId)
		// the Sweep tx sent to the chain is in Hex format
		// encode it into base64 before passing to the decodePsbt function
		sweepTx64, err := utils.HexToBase64(tx.BtcUnsignedSweepTx)
		if err != nil {
			fmt.Printf("[DEBUG] SKIP: error converting hex to base64: %v\n", err)
			continue
		}
		decodedPsbt, err := comms.DecodePsbt(sweepTx64, wallet)
		if err != nil {
			fmt.Printf("[DEBUG] SKIP: error decoding sweep tx: %v\n", err)
			continue
		}

		if len(decodedPsbt.Inputs) <= 0 {
			fmt.Println("[DEBUG] SKIP: no inputs in decoded PSBT")
			continue
		}

		script := decodedPsbt.Inputs[0].WitnessScript.Hex
		fmt.Printf("[DEBUG] witness script hex: %s\n", script)

		addresses := db.QueryUnsignedSweepAddressByScript(dbconn, script)
		if len(addresses) <= 0 {
			fmt.Printf("[DEBUG] SKIP: no address found in DB for script\n")
			continue
		}

		reserveAddress := addresses[0]
		fmt.Printf("[DEBUG] found address: %s, Signed_sweep: %v\n", reserveAddress.Address, reserveAddress.Signed_sweep)

		if reserveAddress.Signed_sweep {
			fmt.Printf("[DEBUG] SKIP: address already signed sweep\n")
			continue
		}

		fragments := comms.GetAllFragments()
		var fragment btcOracleTypes.Fragment
		found := false
		for _, f := range fragments.Fragments {
			if f.JudgeAddress == tx.JudgeAddress {
				fragment = f
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("[DEBUG] SKIP: No fragment found with judge address: %s\n", tx.JudgeAddress)
			return
		}

		found = false
		for _, signer := range fragment.Signers {
			if signer.SignerAddress == signerAddr {
				found = true
			}
		}

		if !found {
			fmt.Printf("[DEBUG] SKIP: Signer %s is not registered with judge %s\n", signerAddr, tx.JudgeAddress)
			continue
		}

		fmt.Printf("[DEBUG] Signing PSBT with wallet: %s\n", wallet)
		signatures, err := comms.SignPsbt(sweepTx64, wallet)
		if err != nil {
			fmt.Printf("[DEBUG] SKIP: error signing psbt: %v\n", err)
			continue
		}

		fmt.Println("[DEBUG] Sweep Signature obtained: ", signatures)
		cosmos := comms.GetCosmosClient()
		msg := &bridgetypes.MsgSignSweep{
			ReserveId:       uint64(reserveId),
			RoundId:         uint64(roundId),
			SignerPublicKey: btcPubKey,
			SweepSignature:  signatures,
			SignerAddress:   signerAddr,
		}

		fmt.Printf("[DEBUG] Sending MsgSignSweep to chain: reserveId=%d, roundId=%d, signerAddr=%s\n",
			reserveId, roundId, signerAddr)
		comms.SendTransactionSignSweep(accountName, cosmos, msg)
		fmt.Printf("[DEBUG] MsgSignSweep sent successfully\n")

		db.MarkAddressSignedSweep(dbconn, reserveAddress.Address)
		fmt.Printf("[DEBUG] Marked address %s as signed sweep in DB\n", reserveAddress.Address)

		// newAddress := comms.GetProposedSweepAddress(uint64(reserveId), uint64(roundId))
		db.InsertTransaction(dbconn, decodedPsbt.Tx.TxID, reserveAddress.Address, uint64(reserveId), uint64(roundId))
		fmt.Printf("[DEBUG] Inserted transaction into DB\n")
	}
	fmt.Println("finishing sweep tx signer")
}

func ProcessTxSigningRefund(accountName string, dbconn *sql.DB, signerAddr string) {
	fmt.Println("starting Refund Tx Signer")
	// wallet := viper.GetString("wallet_name")
	btcPubKey := viper.GetString("btc_public_key")
	refundTxs := comms.GetAllUnsignedRefundTx()

	for _, tx := range refundTxs.UnsignedTxRefundMsgs {
		// decodedPsbt, err := comms.DecodePsbt(tx.BtcUnsignedRefundTx, wallet)
		// if err != nil {
		// 	fmt.Println("error decoding sweep tx : inside processSweepTx : ", err)
		// 	continue
		// }

		// if len(decodedPsbt.Inputs) <= 0 {
		// 	fmt.Println("signing: no inputs")
		// 	continue
		// }

		// script := decodedPsbt.Inputs[0].WitnessScript.Hex
		// addresses := db.QueryUnsignedRefundAddressByScript(dbconn, script)
		// if len(addresses) <= 0 {
		// 	continue
		// }
		// reserveAddress := addresses[0]

		// if reserveAddress.Signed_refund {
		// 	continue
		// }
		// signatures, err := comms.SignPsbt(tx.BtcUnsignedRefundTx, wallet)
		// if err != nil {
		// 	fmt.Println("error signing psbt : inside processSweepTx : ", err)
		// 	continue
		// }

		reserveId, _ := strconv.Atoi(tx.ReserveId)
		roundId, _ := strconv.Atoi(tx.RoundId)

		fmt.Println("Refund Signature : junk signature")
		cosmos := comms.GetCosmosClient()
		msg := &bridgetypes.MsgSignRefund{
			ReserveId:       uint64(reserveId),
			RoundId:         uint64(roundId),
			SignerPublicKey: btcPubKey,
			RefundSignature: []string{"3045022100a6fbb0b1a49b65789e2c33a76c12488f66e12edf24a6ddacbe6a4e4e44f4d79f02205ad4c7e0bb27ae984e7f2cd9d41423f68b2a0c8aaee0f1c409bdd7e3f67d3c7d"},
			SignerAddress:   signerAddr,
		}

		comms.SendTransactionSignRefund(accountName, cosmos, msg)

		db.MarkAddressSignedRefund(dbconn)
	}

	fmt.Println("finishing refund tx signer")
}
