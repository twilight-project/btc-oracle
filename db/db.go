package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/spf13/viper"
	btcOracleTypes "github.com/twilight-project/forkoracle-go/types"
)

func InitDB() *sql.DB {
	psqlconn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", viper.Get("DB_host"), viper.Get("DB_port"), viper.Get("DB_user"), viper.Get("DB_password"), viper.Get("DB_name"))
	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		log.Println("DB error : ", err)
		panic(err)
	}
	fmt.Println("DB initialized")
	return db
}

func InsertNotifications(dbconn *sql.DB, element btcOracleTypes.WatchtowerNotification) {
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

func MarkProcessedNotifications(dbconn *sql.DB, element btcOracleTypes.WatchtowerNotification) {
	_, err := dbconn.Exec("update notification set archived = true where txid = $1 and sending = $2",
		element.Receiving_txid,
		element.Sending,
	)
	if err != nil {
		fmt.Println("An error occured while mark notification query: ", err)
	}
}

func QueryNotification(dbconn *sql.DB) []btcOracleTypes.WatchtowerNotification {
	DB_reader, err := dbconn.Query("select * from notification where archived = false")
	if err != nil {
		fmt.Println("An error occured while query Notification query: ", err)
		return nil
	}

	defer DB_reader.Close()
	addresses := make([]btcOracleTypes.WatchtowerNotification, 0)

	for DB_reader.Next() {
		address := btcOracleTypes.WatchtowerNotification{}
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

func QueryUtxo(dbconn *sql.DB, address string) []btcOracleTypes.Utxo {
	DB_reader, err := dbconn.Query("select txid, Receiving_vout, satoshis from notification where receiving = $1", address)
	if err != nil {
		fmt.Println("An error occured while query utxo: ", err)
		return nil
	}

	defer DB_reader.Close()
	utxos := make([]btcOracleTypes.Utxo, 0)

	for DB_reader.Next() {
		utxo := btcOracleTypes.Utxo{}
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

func QueryAmount(dbconn *sql.DB, receiving_vout uint32, receiving_txid string) uint64 {
	DB_reader, err := dbconn.Query("select satoshis from notification where receiving_vout = $1 and txid = $2", receiving_vout, receiving_txid)
	if err != nil {
		fmt.Println("An error occured query amount query: ", err)
		return 0
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

func InsertSweepAddress(dbconn *sql.DB, address string, script string, preimage []byte, unlock_height int64, parent_address string, owned bool) {
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

func MarkAddressSignedSweep(dbconn *sql.DB, address string) {
	_, err := dbconn.Exec("update address set signed_sweep = $1 where address = $2",
		true,
		address,
	)
	if err != nil {
		fmt.Println("An error occured while mark sweep address signed query: ", err)
	}
}

func MarkAddressBroadcastedSweep(dbconn *sql.DB, address string) {
	_, err := dbconn.Exec("update address set broadcast_sweep = $1 where address = $2",
		true,
		address,
	)
	if err != nil {
		fmt.Println("An error occured while mark sweep address signed query: ", err)
	}
}

func MarkAddressBroadcastedRefund(dbconn *sql.DB) {
	_, err := dbconn.Exec("update address set broadcast_refund = true")
	if err != nil {
		fmt.Println("An error occured while mark sweep address signed query: ", err)
	}
}

func MarkAddressArchived(dbconn *sql.DB, address string) {
	_, err := dbconn.Exec("update address set archived = $1 where address = $2",
		true,
		address,
	)
	if err != nil {
		fmt.Println("An error occured while mark sweep address signed query: ", err)
	}
}

func UpdateAddressUnlockHeight(dbconn *sql.DB, address string, height int64) {
	_, err := dbconn.Exec("update address set unlock_height = $1 where address = $2",
		height,
		address,
	)
	if err != nil {
		fmt.Println("An error occured while mark sweep address signed query: ", err)
	}
}

func MarkAddressSignedRefund(dbconn *sql.DB) {
	_, err := dbconn.Exec("update address set signed_refund = true")
	if err != nil {
		fmt.Println("An error occured while mark sweep address signed query: ", err)
	}
}

func QuerySweepAddressesByHeight(dbconn *sql.DB, height uint64, owned bool) []btcOracleTypes.SweepAddress {
	// fmt.Println("getting address for height: ", height)
	var DB_reader *sql.Rows
	var err error
	DB_reader, err = dbconn.Query("select address, script, preimage, parent_address, unlock_height from address where unlock_height < $1 and unlock_height > $2-7 and archived = false and owned = $3", height, height, owned)

	if err != nil {
		fmt.Println("An error occured while query address by height: ", err)
		return nil
	}

	defer DB_reader.Close()
	addresses := make([]btcOracleTypes.SweepAddress, 0)

	for DB_reader.Next() {
		address := btcOracleTypes.SweepAddress{}
		err := DB_reader.Scan(
			&address.Address,
			&address.Script,
			&address.Preimage,
			&address.Parent_address,
			&address.Unlock_height,
		)
		if err != nil {
			fmt.Println(err)
		}
		addresses = append(addresses, address)
	}

	return addresses
}

func QueryAllSweepAddresses(dbconn *sql.DB, owned bool, archived bool) []btcOracleTypes.SweepAddress {
	// fmt.Println("getting address for height: ", height)
	DB_reader, err := dbconn.Query("select * from address where owned = $1 and archived = $2", owned, archived)
	if err != nil {
		fmt.Println("An error occured while query address by height: ", err)
		return nil
	}

	defer DB_reader.Close()
	addresses := make([]btcOracleTypes.SweepAddress, 0)

	for DB_reader.Next() {
		address := btcOracleTypes.SweepAddress{}
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

func QuerySweepAddressesOrderByHeight(dbconn *sql.DB, limit int) []btcOracleTypes.SweepAddress {
	DB_reader, err := dbconn.Query("SELECT * FROM address WHERE archived = false ORDER BY unlock_height DESC LIMIT $1", limit)
	if err != nil {
		fmt.Println("An error occured while query sweep address: ", err)
		return nil
	}

	defer DB_reader.Close()
	addresses := make([]btcOracleTypes.SweepAddress, 0)

	for DB_reader.Next() {
		address := btcOracleTypes.SweepAddress{}
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

func QuerySweepAddress(dbconn *sql.DB, addr string) []btcOracleTypes.SweepAddress {
	DB_reader, err := dbconn.Query("select * from address where address = $1", addr)
	if err != nil {
		fmt.Println("An error occured while query sweep address: ", err)
		return nil
	}

	defer DB_reader.Close()
	addresses := make([]btcOracleTypes.SweepAddress, 0)

	for DB_reader.Next() {
		address := btcOracleTypes.SweepAddress{}
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

func QueryUnsignedSweepAddressByScript(dbconn *sql.DB, script string) []btcOracleTypes.SweepAddress {
	DB_reader, err := dbconn.Query("select * from address where script = $1", script)
	if err != nil {
		fmt.Println("An error occured while query sweep address: ", err)
		return nil
	}

	defer DB_reader.Close()
	addresses := make([]btcOracleTypes.SweepAddress, 0)

	for DB_reader.Next() {
		address := btcOracleTypes.SweepAddress{}
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

func QueryUnsignedRefundAddressByScript(dbconn *sql.DB, script string) []btcOracleTypes.SweepAddress {
	DB_reader, err := dbconn.Query("select * from address where script = $1 and signed_refund = false and archived = false", script)
	if err != nil {
		fmt.Println("An error occured while query sweep address: ", err)
		return nil
	}

	defer DB_reader.Close()
	addresses := make([]btcOracleTypes.SweepAddress, 0)

	for DB_reader.Next() {
		address := btcOracleTypes.SweepAddress{}
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

func QuerySweepAddressScript(dbconn *sql.DB, address string) []byte {
	DB_reader, err := dbconn.Query("select script from address where address = $1", address)
	if err != nil {
		fmt.Println("An error occured while query script sweep address: ", err)
		return nil
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

func QueryAllAddressOnly(dbconn *sql.DB) []string {
	DB_reader, err := dbconn.Query("select address from address;")
	if err != nil {
		fmt.Println("An error occured while query address: ", err)
		return nil
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

func InsertTransaction(dbconn *sql.DB, txid string, address string, reserve uint64, round uint64) {
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

func QueryWatchedTransactions(dbconn *sql.DB) []btcOracleTypes.WatchedTx {
	DB_reader, err := dbconn.Query("select * from transaction where watched = true;")
	if err != nil {
		fmt.Println("An error occured while query transactions: ", err)
		return nil
	}

	defer DB_reader.Close()
	txs := make([]btcOracleTypes.WatchedTx, 0)

	for DB_reader.Next() {
		tx := btcOracleTypes.WatchedTx{}
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

func MarkTransactionProcessed(dbconn *sql.DB, txid string) {
	_, err := dbconn.Exec("update transaction set watched = false where txid = $1",
		txid,
	)
	if err != nil {
		fmt.Println("An error occured while mark tx id processed: ", err)
	}
}

func InsertProposedAddress(dbconn *sql.DB, current string, proposed string, unlock_height int64, roundID int64, reserveID int64) {
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

func CheckIfAddressIsProposed(dbconn *sql.DB, roundID int64, reserveId uint64) bool {
	DB_reader, err := dbconn.Query("SELECT 1 FROM proposed_address WHERE round_id = $1 and reserve_id = $2 LIMIT 1;", roundID, reserveId)
	if err != nil {
		fmt.Println("An error occurred while querying proposed addresses:", err)
		return true // Return false on error
	}
	defer DB_reader.Close()

	return DB_reader.Next() // Return true if there is at least one result row
}

func InsertSignedSweeptx(dbconn *sql.DB, tx string, unlockHeight int64) {
	_, err := dbconn.Exec("INSERT into signed_tx VALUES ($1, $2)",
		tx,
		unlockHeight,
	)
	if err != nil {
		fmt.Println("An error occured while executing insert signed sweep tx: ", err)
	}
}

func QuerySignedSweeptx(dbconn *sql.DB, height uint64) ([]btcOracleTypes.SignedTx, error) {
	DB_reader, err := dbconn.Query("select * from signed_tx where unlock_height <  $1", height)
	if err != nil {
		fmt.Println("An error occured while query unsigned Tx: ", err)
		return []btcOracleTypes.SignedTx{}, err
	}

	defer DB_reader.Close()
	var signedTxs []btcOracleTypes.SignedTx

	for DB_reader.Next() {
		var signedTx btcOracleTypes.SignedTx
		err := DB_reader.Scan(
			&signedTx.Tx,
			&signedTx.UnlockHeight,
		)
		if err != nil {
			fmt.Println(err)
		}

		signedTxs = append(signedTxs, signedTx)
	}

	return signedTxs, nil
}

func DeleteSignedSweeptx(dbconn *sql.DB, height uint64) {
	_, err := dbconn.Exec("Delete from signed_tx where unlock_height < $1",
		height,
	)
	if err != nil {
		fmt.Println("An error occured while executing deleting signed sweep tx: ", err)
	}
}

func InsertUnSignedSweeptx(dbconn *sql.DB, tx string, reserveId int64, roundId int64) {
	_, err := dbconn.Exec("INSERT into unsigned_sweep_tx VALUES ($1, $2, $3)",
		tx,
		reserveId,
		roundId,
	)
	if err != nil {
		fmt.Println("An error occured while executing insert unsigned sweep tx: ", err)
	}
}

func QueryUnSignedSweeptx(dbconn *sql.DB, reserveId int64, roundId int64) ([]btcOracleTypes.UnsignedTx, error) {
	DB_reader, err := dbconn.Query("select * from unsigned_sweep_tx where reserve_id = $1 and round_id = $2 Limit 1;", reserveId, roundId)
	if err != nil {
		fmt.Println("An error occured while query unsigned Tx: ", err)
		return []btcOracleTypes.UnsignedTx{}, err
	}

	defer DB_reader.Close()
	var unSignedTxs []btcOracleTypes.UnsignedTx

	for DB_reader.Next() {
		var unSignedTx btcOracleTypes.UnsignedTx
		err := DB_reader.Scan(
			&unSignedTx.Tx,
			&unSignedTx.ReserveId,
			&unSignedTx.RoundId,
		)
		if err != nil {
			fmt.Println(err)
		}

		unSignedTxs = append(unSignedTxs, unSignedTx)
	}

	return unSignedTxs, nil
}

func InsertUnSignedRefundtx(dbconn *sql.DB, tx string, reserveId int64, roundId int64) {
	_, err := dbconn.Exec("INSERT into unsigned_refund_tx VALUES ($1, $2, $3)",
		tx,
		reserveId,
		roundId,
	)
	if err != nil {
		fmt.Println("An error occured while executing insert unsigned sweep tx: ", err)
	}
}

func QueryUnSignedRefundtx(dbconn *sql.DB, reserveId int64, roundId int64) ([]btcOracleTypes.UnsignedTx, error) {
	DB_reader, err := dbconn.Query("select * from unsigned_refund_tx where reserve_id = $1 and round_id = $2 Limit 1;", reserveId, roundId)
	if err != nil {
		fmt.Println("An error occured while query unsigned Tx: ", err)
		return []btcOracleTypes.UnsignedTx{}, err
	}

	defer DB_reader.Close()
	var unSignedTxs []btcOracleTypes.UnsignedTx

	for DB_reader.Next() {
		var unSignedTx btcOracleTypes.UnsignedTx
		err := DB_reader.Scan(
			&unSignedTx.Tx,
			&unSignedTx.ReserveId,
			&unSignedTx.RoundId,
		)
		if err != nil {
			fmt.Println(err)
		}

		unSignedTxs = append(unSignedTxs, unSignedTx)
	}

	return unSignedTxs, nil
}
