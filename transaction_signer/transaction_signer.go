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
	bridgetypes "github.com/twilight-project/nyks/x/bridge/types"
)

func ProcessTxSigningSweep(accountName string, dbconn *sql.DB, signerAddr string) {
	fmt.Println("starting Sweep Tx Signer")
	wallet := viper.GetString("wallet_name")
	btcPubKey := viper.GetString("btc_public_key")
	SweepTxs := comms.GetAllUnsignedSweepTx()

	for _, tx := range SweepTxs.UnsignedTxSweepMsgs {
		reserveId, _ := strconv.Atoi(tx.ReserveId)
		roundId, _ := strconv.Atoi(tx.RoundId)

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
		sweepTx64, _ := utils.HexToBase64(tx.BtcUnsignedSweepTx)
		decodedPsbt, err := comms.DecodePsbt(sweepTx64, wallet)
		if err != nil {
			fmt.Println("error decoding sweep tx : inside processSweepTx ->DecodePSBT : ", err)
			continue
		}

		if len(decodedPsbt.Inputs) <= 0 {
			fmt.Println("signing: no inputs")
			continue
		}

		script := decodedPsbt.Inputs[0].WitnessScript.Hex

		addresses := db.QueryUnsignedSweepAddressByScript(dbconn, script)
		if len(addresses) <= 0 {
			fmt.Println("signing: no address")
			continue
		}

		reserveAddress := addresses[0]

		if reserveAddress.Signed_sweep {
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
			fmt.Println("No fragment found with the specified judge address")
			return
		}

		found = false
		for _, signer := range fragment.Signers {
			if signer.SignerAddress == signerAddr {
				found = true
			}
		}

		if !found {
			fmt.Println("Signer is not registered with the provided judge")
		}

		signatures, err := comms.SignPsbt(sweepTx64, wallet)
		if err != nil {
			fmt.Println("error signing psbt : inside processSweepTx ->SignPSBT: ", err)
			continue
		}

		fmt.Println("Sweep Signature : ", signatures)
		cosmos := comms.GetCosmosClient()
		msg := &bridgetypes.MsgSignSweep{
			ReserveId:       uint64(reserveId),
			RoundId:         uint64(roundId),
			SignerPublicKey: btcPubKey,
			SweepSignature:  signatures,
			SignerAddress:   signerAddr,
		}

		comms.SendTransactionSignSweep(accountName, cosmos, msg)

		db.MarkAddressSignedSweep(dbconn, reserveAddress.Address)

		// newAddress := comms.GetProposedSweepAddress(uint64(reserveId), uint64(roundId))
		db.InsertTransaction(dbconn, decodedPsbt.Tx.TxID, reserveAddress.Address, uint64(reserveId), uint64(roundId))
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
			RefundSignature: []string{"junkSignature"},
			SignerAddress:   signerAddr,
		}

		comms.SendTransactionSignRefund(accountName, cosmos, msg)

		db.MarkAddressSignedRefund(dbconn)
	}

	fmt.Println("finishing refund tx signer")
}
