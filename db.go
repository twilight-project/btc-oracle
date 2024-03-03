package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/spf13/viper"
)

func initDB() *sql.DB {
	psqlconn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", viper.Get("DB_host"), viper.Get("DB_port"), viper.Get("DB_user"), viper.Get("DB_password"), viper.Get("DB_name"))
	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		log.Println("DB error : ", err)
		panic(err)
	}
	fmt.Println("DB initialized")
	return db
}

func insertNotifications(element WatchtowerNotification) {
	_, err := dbconn.Exec("INSERT into notification VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		element.Block,
		element.Receiving,
		element.Satoshis,
		element.Height,
		element.Receiving_txid,
		false,
		element.Sending_txinputs[0].Address,
		element.Receiving_vout,
		-1,
	)
	if err != nil {
		fmt.Println("An error occured while insert Notification query: ", err)
	}
}

func markProcessedNotifications(element WatchtowerNotification) {
	_, err := dbconn.Exec("update notification set archived = true where txid = $1 and sending = $2",
		element.Receiving_txid,
		element.Sending,
	)
	if err != nil {
		fmt.Println("An error occured while mark notification query: ", err)
	}
}

func queryNotification() []WatchtowerNotification {
	DB_reader, err := dbconn.Query("select * from notification where archived = false")
	if err != nil {
		fmt.Println("An error occured while query Notification query: ", err)
	}

	defer DB_reader.Close()
	addresses := make([]WatchtowerNotification, 0)

	for DB_reader.Next() {
		address := WatchtowerNotification{}
		err := DB_reader.Scan(
			&address.Block,
			&address.Receiving,
			&address.Satoshis,
			&address.Height,
			&address.Receiving_txid,
			&address.Archived,
			&address.Sending,
			&address.Receiving_vout,
			&address.Sending_vout,
		)
		if err != nil {
			fmt.Println(err)
		}
		addresses = append(addresses, address)
	}
	return addresses
}

func queryUtxo(address string) []Utxo {
	DB_reader, err := dbconn.Query("select txid, Receiving_vout, satoshis from notification where receiving = $1", address)
	if err != nil {
		fmt.Println("An error occured while query utxo: ", err)
	}

	defer DB_reader.Close()
	utxos := make([]Utxo, 0)

	for DB_reader.Next() {
		utxo := Utxo{}
		err := DB_reader.Scan(
			&utxo.Txid,
			&utxo.Vout,
			&utxo.Amount,
		)
		if err != nil {
			fmt.Println(err)
		}
		utxos = append(utxos, utxo)
	}
	return utxos
}

func queryAmount(receiving_vout uint32, receiving_txid string) uint64 {
	DB_reader, err := dbconn.Query("select satoshis from notification where receiving_vout = $1 and txid = $2", receiving_vout, receiving_txid)
	if err != nil {
		fmt.Println("An error occured query amount query: ", err)
	}

	defer DB_reader.Close()
	var intValue uint64
	for DB_reader.Next() {
		err := DB_reader.Scan(
			&intValue,
		)
		if err != nil {
			fmt.Println(err)
		}
	}
	return intValue
}

func insertSweepAddress(address string, script []byte, preimage []byte, unlock_height int64, parent_address string, owned bool) {
	_, err := dbconn.Exec("INSERT into address VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
		address,
		script,
		preimage,
		unlock_height,
		parent_address,
		false,
		false,
		false,
		false,
		false,
		owned,
	)
	if err != nil {
		fmt.Println("An error occured while executing insert sweep address query: ", err)
	}
}

// func markProcessedSweepAddress(address string) {
// 	_, err := dbconn.Exec("update address set archived = true where address = $1",
// 		address,
// 	)
// 	if err != nil {
// 		fmt.Println("An error occured while mark sweep address query: ", err)
// 	}
// }

// not used yet
// func updateAddressUnlockHeight(address string, height int) {
// 	_, err := dbconn.Exec("update address set unlock_height = $1 where address = $2",
// 		height,
// 		address,
// 	)
// 	if err != nil {
// 		fmt.Println("An error occured while update unlock height query: ", err)
// 	}
// }

func markAddressSignedSweep(address string) {
	_, err := dbconn.Exec("update address set signed_sweep = $1 where address = $2",
		true,
		address,
	)
	if err != nil {
		fmt.Println("An error occured while mark sweep address signed query: ", err)
	}
}

func markAddressBroadcastedSweep(address string) {
	_, err := dbconn.Exec("update address set broadcast_sweep = $1 where address = $2",
		true,
		address,
	)
	if err != nil {
		fmt.Println("An error occured while mark sweep address signed query: ", err)
	}
}

func markAddressBroadcastedRefund(address string) {
	_, err := dbconn.Exec("update address set broadcast_refund = $1 where address = $2",
		true,
		address,
	)
	if err != nil {
		fmt.Println("An error occured while mark sweep address signed query: ", err)
	}
}

func markAddressArchived(address string) {
	_, err := dbconn.Exec("update address set archived = $1 where address = $2",
		true,
		address,
	)
	if err != nil {
		fmt.Println("An error occured while mark sweep address signed query: ", err)
	}
}

func UpdateAddressUnlockHeight(address string, height int64) {
	_, err := dbconn.Exec("update address set unlock_height = $1 where address = $2",
		height,
		address,
	)
	if err != nil {
		fmt.Println("An error occured while mark sweep address signed query: ", err)
	}
}

func markAddressSignedRefund(address string) {
	_, err := dbconn.Exec("update address set signed_refund = $1 where address = $2",
		true,
		address,
	)
	if err != nil {
		fmt.Println("An error occured while mark sweep address signed query: ", err)
	}
}

func querySweepAddressesByHeight(height uint64, owned bool) []SweepAddress {
	// fmt.Println("getting address for height: ", height)
	DB_reader, err := dbconn.Query("select address, script, preimage, parent_address from address where unlock_height = $1 and archived = false and owned = $2", height, owned)
	if err != nil {
		fmt.Println("An error occured while query address by height: ", err)
	}

	defer DB_reader.Close()
	addresses := make([]SweepAddress, 0)

	for DB_reader.Next() {
		address := SweepAddress{}
		err := DB_reader.Scan(
			&address.Address,
			&address.Script,
			&address.Preimage,
			&address.Parent_address,
		)
		if err != nil {
			fmt.Println(err)
		}
		addresses = append(addresses, address)
	}

	return addresses
}

func querySweepAddressesOrderByHeight(limit int) []SweepAddress {
	DB_reader, err := dbconn.Query("SELECT * FROM address WHERE archived = false ORDER BY unlock_height DESC LIMIT $1", limit)
	if err != nil {
		fmt.Println("An error occured while query sweep address: ", err)
	}

	defer DB_reader.Close()
	addresses := make([]SweepAddress, 0)

	for DB_reader.Next() {
		address := SweepAddress{}
		err := DB_reader.Scan(
			&address.Address,
			&address.Script,
			&address.Preimage,
			&address.Unlock_height,
			&address.Parent_address,
			&address.Signed_refund,
			&address.Signed_sweep,
			&address.Archived,
			&address.BroadcastSweep,
			&address.BroadcastRefund,
			&address.Owned,
		)
		if err != nil {
			fmt.Println(err)
		}
		addresses = append(addresses, address)
	}

	return addresses
}

func querySweepAddress(addr string) []SweepAddress {
	DB_reader, err := dbconn.Query("select * from address where address = $1", addr)
	if err != nil {
		fmt.Println("An error occured while query sweep address: ", err)
	}

	defer DB_reader.Close()
	addresses := make([]SweepAddress, 0)

	for DB_reader.Next() {
		address := SweepAddress{}
		err := DB_reader.Scan(
			&address.Address,
			&address.Script,
			&address.Preimage,
			&address.Unlock_height,
			&address.Parent_address,
			&address.Signed_refund,
			&address.Signed_sweep,
			&address.Archived,
			&address.BroadcastSweep,
			&address.BroadcastRefund,
			&address.Owned,
		)
		if err != nil {
			fmt.Println(err)
		}
		addresses = append(addresses, address)
	}

	return addresses
}

func queryAllSweepAddresses() []SweepAddress {
	DB_reader, err := dbconn.Query("select address, script, preimage, parent_address from address where archived = false")
	if err != nil {
		fmt.Println("An error occured while query all sweep addresses: ", err)
	}

	defer DB_reader.Close()
	addresses := make([]SweepAddress, 0)

	for DB_reader.Next() {
		address := SweepAddress{}
		err := DB_reader.Scan(
			&address.Address,
			&address.Script,
			&address.Preimage,
			&address.Parent_address,
		)
		if err != nil {
			fmt.Println(err)
		}
		addresses = append(addresses, address)
	}

	return addresses
}

func queryUnsignedSweepAddressByScript(script []byte) []SweepAddress {
	DB_reader, err := dbconn.Query("select * from address where script = $1", script)
	if err != nil {
		fmt.Println("An error occured while query sweep address: ", err)
	}

	defer DB_reader.Close()
	addresses := make([]SweepAddress, 0)

	for DB_reader.Next() {
		address := SweepAddress{}
		err := DB_reader.Scan(
			&address.Address,
			&address.Script,
			&address.Preimage,
			&address.Unlock_height,
			&address.Parent_address,
			&address.Signed_refund,
			&address.Signed_sweep,
			&address.Archived,
			&address.BroadcastSweep,
			&address.BroadcastRefund,
			&address.Owned,
		)
		if err != nil {
			fmt.Println(err)
		}
		addresses = append(addresses, address)
	}

	return addresses
}

func queryUnsignedRefundAddressByScript(script []byte) []SweepAddress {
	DB_reader, err := dbconn.Query("select * from address where script = $1 and signed_refund = false and archived = false", script)
	if err != nil {
		fmt.Println("An error occured while query sweep address: ", err)
	}

	defer DB_reader.Close()
	addresses := make([]SweepAddress, 0)

	for DB_reader.Next() {
		address := SweepAddress{}
		err := DB_reader.Scan(
			&address.Address,
			&address.Script,
			&address.Preimage,
			&address.Unlock_height,
			&address.Parent_address,
			&address.Signed_refund,
			&address.Signed_sweep,
			&address.Archived,
			&address.BroadcastSweep,
			&address.BroadcastRefund,
			&address.Owned,
		)
		if err != nil {
			fmt.Println(err)
		}
		addresses = append(addresses, address)
	}

	return addresses
}

func querySweepAddressScript(address string) []byte {
	DB_reader, err := dbconn.Query("select script from address where address = $1", address)
	if err != nil {
		fmt.Println("An error occured while query script sweep address: ", err)
	}

	defer DB_reader.Close()
	script := []byte{}

	for DB_reader.Next() {
		err := DB_reader.Scan(
			&script,
		)
		if err != nil {
			fmt.Println(err)
		}
	}

	return script
}

// func querySweepAddressByScript(script []byte) string {
// 	DB_reader, err := dbconn.Query("select script from address where script = $1", script)
// 	if err != nil {
// 		fmt.Println("An error occured while query script: ", err)
// 	}

// 	defer DB_reader.Close()
// 	var address string

// 	for DB_reader.Next() {
// 		err := DB_reader.Scan(
// 			&address,
// 		)
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 	}

// 	return address
// }

// func querySweepAddressPreimage(address string) []byte {
// 	DB_reader, err := dbconn.Query("select preimage from address where address = $1", address)
// 	if err != nil {
// 		fmt.Println("An error occured while query preimage: ", err)
// 	}

// 	defer DB_reader.Close()
// 	preimage := []byte{}

// 	for DB_reader.Next() {
// 		err := DB_reader.Scan(
// 			&preimage,
// 		)
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 	}

// 	return preimage
// }

// func querySweepAddressByParentAddress(address string) []SweepAddress {
// 	DB_reader, err := dbconn.Query("select * from address where parent_address = $1", address)
// 	if err != nil {
// 		fmt.Println("An error occured while query address by parent address: ", err)
// 	}

// 	defer DB_reader.Close()
// 	addresses := make([]SweepAddress, 0)

// 	for DB_reader.Next() {
// 		address := SweepAddress{}
// 		err := DB_reader.Scan(
// 			&address.Address,
// 			&address.Script,
// 			&address.Preimage,
// 			&address.Unlock_height,
// 			&address.Parent_address,
// 			&address.Signed_refund,
// 			&address.Signed_sweep,
// 			&address.Archived,
// 			&address.BroadcastSweep,
// 			&address.BroadcastRefund,
// 			&address.Owned,
// 		)
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 		addresses = append(addresses, address)
// 	}
// 	return addresses
// }

func queryAllAddressOnly() []string {
	DB_reader, err := dbconn.Query("select address from address;")
	if err != nil {
		fmt.Println("An error occured while query address: ", err)
	}

	defer DB_reader.Close()
	addresses := make([]string, 0)

	for DB_reader.Next() {
		address := ""
		err := DB_reader.Scan(
			&address,
		)
		if err != nil {
			fmt.Println(err)
		}
		addresses = append(addresses, address)
	}
	return addresses
}

func insertTransaction(txid string, address string, reserve uint64, round uint64) {
	_, err := dbconn.Exec("INSERT into transaction VALUES ($1, $2, $3, $4, $5)",
		txid,
		address,
		reserve,
		round,
		true,
	)
	if err != nil {
		fmt.Println("An error occured while executing insert watched transaction query: ", err)
	}
}

func queryWatchedTransactions() []WatchedTx {
	DB_reader, err := dbconn.Query("select * from transaction where watched = true;")
	if err != nil {
		fmt.Println("An error occured while query transactions: ", err)
	}

	defer DB_reader.Close()
	txs := make([]WatchedTx, 0)

	for DB_reader.Next() {
		tx := WatchedTx{}
		err := DB_reader.Scan(
			&tx.Txid,
			&tx.Address,
			&tx.Reserve,
			&tx.Round,
			&tx.Watched,
		)
		if err != nil {
			fmt.Println(err)
		}
		txs = append(txs, tx)
	}
	return txs
}

func markTransactionProcessed(txid string) {
	_, err := dbconn.Exec("update transaction set watched = false where txid = $1",
		txid,
	)
	if err != nil {
		fmt.Println("An error occured while mark tx id processed: ", err)
	}
}

func insertProposedAddress(current string, proposed string, unlock_height int64, roundID int64, reserveID int64) {
	_, err := dbconn.Exec("INSERT into proposed_address VALUES ($1, $2, $3, $4, $5)",
		current,
		proposed,
		unlock_height,
		reserveID,
		roundID,
	)
	if err != nil {
		fmt.Println("An error occured while executing insert watched transaction query: ", err)
	}
}

// func checkIfAddressIsProposed(roundID int64, reserveID int64) int {
// 	DB_reader, err := dbconn.Query("select sum(proposed) from proposed_address where round_id = $1, reserve_id = $2;", roundID, reserveID)
// 	if err != nil {
// 		fmt.Println("An error occured while query proposed addresses: ", err)
// 	}

// 	defer DB_reader.Close()
// 	var length int

// 	for DB_reader.Next() {
// 		err := DB_reader.Scan(
// 			&length,
// 		)
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 	}
// 	return length
// }

func checkIfAddressIsProposed(roundID int64) bool {
	DB_reader, err := dbconn.Query("SELECT 1 FROM proposed_address WHERE \"roundId\" = $1 LIMIT 1;", roundID)
	if err != nil {
		fmt.Println("An error occurred while querying proposed addresses:", err)
		return true // Return false on error
	}
	defer DB_reader.Close()

	return DB_reader.Next() // Return true if there is at least one result row
}

func insertSignedtx(tx []byte, unlock_height int64) {
	_, err := dbconn.Exec("INSERT into signed_tx VALUES ($1, $2)",
		tx,
		unlock_height,
	)
	if err != nil {
		fmt.Println("An error occured while executing insert signed sweep tx: ", err)
	}
}

func querySignedTx(unlock_height int64) [][]byte {
	DB_reader, err := dbconn.Query("select tx from signed_tx where unlock_height <= $1", unlock_height)
	if err != nil {
		fmt.Println("An error occured while query script signed tx: ", err)
	}

	defer DB_reader.Close()
	txs := [][]byte{}

	for DB_reader.Next() {
		tx := []byte{}
		err := DB_reader.Scan(
			&tx,
		)
		if err != nil {
			fmt.Println(err)
		}

		txs = append(txs, tx)
	}
	return txs
}

func deleteSignedTx(tx []byte) {
	_, err := dbconn.Exec("DELETE FROM signed_tx WHERE tx = $1", tx)
	if err != nil {
		fmt.Println("An error occurred while executing delete signed tx: ", err)
	} else {
		fmt.Println("Transaction successfully deleted")
	}
}
