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
	return db
}

func insertNotifications(element WatchtowerNotification) {

	fmt.Println("inside insert DB")
	_, err := dbconn.Exec("INSERT into notification VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		element.Block,
		element.Receiving,
		element.Satoshis,
		element.Height,
		element.Receiving_txid,
		false,
		element.Sending_txinputs[0].Address,
		element.Receiving_vout,
		nil,
	)
	if err != nil {
		fmt.Println("An error occured while executing query: ", err)
	}
}

func markProcessedNotifications(element WatchtowerNotification) {
	_, err := dbconn.Exec("update notification set archived = true where txid = $1 and sending = $2",
		element.Receiving_txid,
		element.Sending,
	)
	if err != nil {
		fmt.Println("An error occured while executing query: ", err)
	}
}

func queryNotification() []WatchtowerNotification {
	DB_reader, err := dbconn.Query("select * from notification where archived = false")
	if err != nil {
		fmt.Println("An error occured while executing query: ", err)
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
	fmt.Println("addresses under watch : ", addresses)
	return addresses
}

func queryUtxo(address string) []Utxo {
	DB_reader, err := dbconn.Query("select txid, Receiving_vout, satoshis from notification where receiving = $1", address)
	if err != nil {
		fmt.Println("An error occured while executing query: ", err)
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
		fmt.Println("An error occured while executing query: ", err)
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

func insertSweepAddress(address string, script []byte, preimage []byte, unlock_height int64) {
	_, err := dbconn.Exec("INSERT into address VALUES ($1, $2, $3, $4)",
		address,
		script,
		preimage,
		unlock_height,
	)
	if err != nil {
		fmt.Println("An error occured while executing query: ", err)
	}
}

func markProcessedSweepAddress(address string) {
	_, err := dbconn.Exec("update address set archived = true where address = $1",
		address,
	)
	if err != nil {
		fmt.Println("An error occured while executing query: ", err)
	}
}

func updateAddressUnlockHeight(address string, height int) {
	_, err := dbconn.Exec("update address set unlock_height = $1 where address = $2",
		height,
		address,
	)
	if err != nil {
		fmt.Println("An error occured while executing query: ", err)
	}
}

func querySweepAddresses(height uint64) []SweepAddress {
	DB_reader, err := dbconn.Query("select address, script, preimage from address where unlock_height = $1 and archived = false", height)
	if err != nil {
		fmt.Println("An error occured while executing query: ", err)
	}

	defer DB_reader.Close()
	addresses := make([]SweepAddress, 0)

	for DB_reader.Next() {
		address := SweepAddress{}
		err := DB_reader.Scan(
			&address.Address,
			&address.Script,
			&address.Preimage,
		)
		if err != nil {
			fmt.Println(err)
		}
		addresses = append(addresses, address)
	}

	return addresses
}

func queryAllSweepAddresses() []SweepAddress {
	DB_reader, err := dbconn.Query("select address, script, preimage from address where archived = false")
	if err != nil {
		fmt.Println("An error occured while executing query: ", err)
	}

	defer DB_reader.Close()
	addresses := make([]SweepAddress, 0)

	for DB_reader.Next() {
		address := SweepAddress{}
		err := DB_reader.Scan(
			&address.Address,
			&address.Script,
			&address.Preimage,
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
		fmt.Println("An error occured while executing query: ", err)
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

func querySweepAddressPreimage(address string) []byte {
	DB_reader, err := dbconn.Query("select preimage from address where address = $1", address)
	if err != nil {
		fmt.Println("An error occured while executing query: ", err)
	}

	defer DB_reader.Close()
	preimage := []byte{}

	for DB_reader.Next() {
		err := DB_reader.Scan(
			&preimage,
		)
		if err != nil {
			fmt.Println(err)
		}
	}

	return preimage
}
