package main

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

func processTxSigningSweep(accountName string) {
	fmt.Println("starting Sweep Tx Signer")
	SweepTxs := getAllUnsignedSweepTx()

	for _, tx := range SweepTxs.UnsignedTxSweepMsgs {
		reserveId, _ := strconv.Atoi(tx.ReserveId)
		roundId, _ := strconv.Atoi(tx.RoundId)

		for {
			correspondingRefundTx := getBroadCastedRefundTx(uint64(reserveId), uint64(roundId))
			if correspondingRefundTx.ReserveId != "" {
				break
			}
			fmt.Printf("corresponding refund tx not found  reserve Id: %d roundid : %d\n", reserveId, roundId)
			time.Sleep(1 * time.Minute)
		}

		fmt.Printf("corresponding refund tx found  reserve Id: %d roundid : %d\n", reserveId, roundId)

		sweepTx, err := createTxFromHex(tx.BtcUnsignedSweepTx)
		if err != nil {
			fmt.Println("error decoding sweep tx : inside processSweepTx : ", err)
			log.Fatal(err)
		}

		addresses := queryUnsignedSweepAddressByScript(sweepTx.TxIn[0].Witness[0])
		if len(addresses) <= 0 {
			fmt.Println("signing: no address")
			continue
		}

		reserveAddress := addresses[0]

		if reserveAddress.Signed_sweep {
			continue
		}

		sweepSignatures := signTx(sweepTx, reserveAddress.Script)

		fmt.Println("Sweep Signature : ", sweepSignatures)
		sendSweepSign(sweepSignatures, reserveAddress.Address, accountName, uint64(reserveId), uint64(roundId))

		markAddressSignedSweep(reserveAddress.Address)
		insertTransaction(sweepTx.TxHash().String(), reserveAddress.Address, uint64(reserveId), uint64(roundId))
	}

	fmt.Println("finishing sweep tx signer")
}

func processTxSigningRefund(accountName string) {
	fmt.Println("starting Refund Tx Signer")
	refundTxs := getAllUnsignedRefundTx()

	for _, tx := range refundTxs.UnsignedTxRefundMsgs {
		refundTx, err := createTxFromHex(tx.BtcUnsignedRefundTx)
		if err != nil {
			fmt.Println("error decoding sweep tx : inside processSweepTx : ", err)
			log.Fatal(err)
		}

		fmt.Println("signing refund tx")
		addresses := queryUnsignedRefundAddressByScript(refundTx.TxIn[0].Witness[0])
		if len(addresses) <= 0 {
			continue
		}
		reserveAddress := addresses[0]

		if reserveAddress.Signed_refund {
			continue
		}
		refundSignature := signTx(refundTx, reserveAddress.Script)

		reserveId, _ := strconv.Atoi(tx.ReserveId)
		roundId, _ := strconv.Atoi(tx.RoundId)

		fmt.Println("Refund Signature : ", refundSignature)
		sendRefundSign(refundSignature[0], reserveAddress.Address, accountName, uint64(reserveId), uint64(roundId))
		markAddressSignedRefund(reserveAddress.Address)
	}

	fmt.Println("finishing refund tx signer")
}

func startTransactionSigner(accountName string) {
	fmt.Println("starting Transaction Signer")
	go nyksEventListener("unsigned_tx_refund", accountName, "signing_refund")
	nyksEventListener("broadcast_tx_refund", accountName, "signing_sweep")
	fmt.Println("finishing bridge")
}
