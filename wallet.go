package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"

	"github.com/btcsuite/btcd/btcec"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

func createWallet(passphrase string) (*bip32.Key, error) {
	entropy, _ := bip39.NewEntropy(256)
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		fmt.Println("Error generating mnemonic:", err)
		return nil, err
	}

	seed := bip39.NewSeed(mnemonic, passphrase)
	encryptedSeed, err := encrypt(seed, passphrase)
	if err != nil {
		fmt.Println("Error encrypting seed:", err)
	}
	writeToFile("seed.txt", encryptedSeed)

	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		fmt.Println("Error :", err)
		panic(err)
	}
	privatekey, err := masterKey.NewChildKey(0)
	if err != nil {
		fmt.Println("Error :", err)
		panic(err)
	}

	// Display mnemonic and keys
	fmt.Println("please save the mnemonic phrase in a safe place")
	fmt.Println("Mnemonic: ", mnemonic)

	return privatekey, nil
}

func loadWallet(passphrase string) (*bip32.Key, error) {
	encrypted_seed, err := readFromFile("seed.txt")
	if err != nil {
		fmt.Println("Error reading seed file:", err)
	}
	seed, err := decrypt(encrypted_seed, passphrase)
	if err != nil {
		fmt.Println("Error decrypting seed:", err)
	}
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		fmt.Println("Error :", err)
		panic(err)
	}
	privatekey, err := masterKey.NewChildKey(0)
	if err != nil {
		fmt.Println("Error :", err)
		panic(err)
	}

	return privatekey, nil
}

func createWalletFromMnemonic(mnemonic string, passphrase string) (*bip32.Key, error) {
	seed := bip39.NewSeed(mnemonic, passphrase)
	encryptedSeed, err := encrypt(seed, passphrase)
	if err != nil {
		fmt.Println("Error encrypting seed:", err)
	}
	writeToFile("seed.txt", encryptedSeed)

	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		fmt.Println("Error :", err)
		panic(err)
	}
	privatekey, err := masterKey.NewChildKey(0)
	if err != nil {
		fmt.Println("Error :", err)
		panic(err)
	}

	return privatekey, nil
}

// Encrypt method is to encrypt or hide any classified text
func encrypt(plainText []byte, MySecret string) ([]byte, error) {

	iv := make([]byte, aes.BlockSize)
	_, err := rand.Read(iv)
	if err != nil {
		fmt.Println("Error generating IV:", err)
		return nil, err
	}

	writeToFile("iv.txt", iv)

	fmt.Println("cipher length : ", len([]byte(MySecret)))
	block, err := aes.NewCipher([]byte(MySecret))
	if err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cipherText := make([]byte, len(plainText))
	cfb.XORKeyStream(cipherText, plainText)
	return cipherText, nil
}

func decrypt(cipherText []byte, MySecret string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(MySecret))
	if err != nil {
		return nil, err
	}
	iv, _ := readFromFile("iv.txt")
	cfb := cipher.NewCFBDecrypter(block, iv)
	plainText := make([]byte, len(cipherText))
	cfb.XORKeyStream(plainText, cipherText)
	return plainText, nil
}

func writeToFile(name string, data []byte) {
	file, err := os.Create(name)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	_, err = file.Write(data)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	defer file.Close()
}

func readFromFile(name string) ([]byte, error) {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return nil, err
	}

	// Convert the byte slice to a string and print it
	return data, nil
}

func initWallet() string {
	new_wallet := flag.Bool("new_wallet", true, "set to true if you want to create a new wallet")
	mnemonic := flag.String("mnemonic", "", "mnemonic for the wallet, leave empty to generate a new nemonic")
	flag.Parse()

	var err error

	walletPassphrase := fmt.Sprintf("%v", viper.Get("wallet_passphrase"))
	if *new_wallet == true {
		if *mnemonic != "" {
			masterPrivateKey, err = createWalletFromMnemonic(*mnemonic, walletPassphrase)
			if err != nil {
				fmt.Println("Error creating wallet from mnemonic:", err)
				panic(err)
			}
		} else {
			masterPrivateKey, err = createWallet(walletPassphrase)
			if err != nil {
				fmt.Println("Error creating wallet:", err)
				panic(err)
			}
		}
	} else {
		masterPrivateKey, err = loadWallet(walletPassphrase)
		if err != nil {
			fmt.Println("Error loading wallet:", err)
			panic(err)
		}
	}

	privkeybytes, err := masterPrivateKey.Serialize()
	if err != nil {
		fmt.Println("Error: converting private key to bytes : ", err)
	}

	privkey, _ := btcec.PrivKeyFromBytes(btcec.S256(), privkeybytes)

	fmt.Println("Private key : ", hex.EncodeToString(privkey.Serialize()))

	btcPubkey := hex.EncodeToString(privkey.PubKey().SerializeCompressed())
	fmt.Println("Public key : ", btcPubkey)

	fmt.Println("Wallet initialized")

	return btcPubkey

}

func getBtcPublicKey() string {
	privkeybytes, err := masterPrivateKey.Serialize()
	if err != nil {
		fmt.Println("Error: converting private key to bytes : ", err)
	}

	privkey, _ := btcec.PrivKeyFromBytes(btcec.S256(), privkeybytes)

	btcPubkey := hex.EncodeToString(privkey.PubKey().SerializeCompressed())
	return btcPubkey
}
