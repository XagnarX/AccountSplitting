package lib

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

func GWallets(numberOfWallets int) (records [][]string, err error) {
	// 生成指定数量的钱包地址和私钥，并将它们写入文件
	for i := 0; i < numberOfWallets; i++ {
		// 生成一个新的私钥
		privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		if err != nil {
			log.Fatal(err)
		}
		// 将私钥转换为字节序列
		privateKeyBytes := privateKey.D.Bytes()
		// 将字节序列转换为十六进制字符串
		privateKeyHex := hex.EncodeToString(privateKeyBytes)
		// 转换一次作为privateKey是否有效的检查
		_, err = crypto.HexToECDSA(privateKeyHex)
		if err != nil {
			i--
			continue
		}
		// 根据私钥生成公钥和地址
		publicKey := privateKey.Public()
		publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			return records, errors.New("生成公钥失败！")
		}
		address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
		// 将私钥和地址写入文件
		records = append(records, []string{privateKeyHex, address})
	}
	return records, nil
}
func GWalletsAndWirte(numberOfWallets int, fileName string) error {
	// 创建名为 secret.csv 的文件，并写入表头
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	records, err := GWallets(numberOfWallets)
	if err != nil {
		log.Fatal(err)
		return err
	}
	for _, record := range records {
		if err := writer.Write(record); err != nil {
			log.Fatal(err)
			return err
		}
	}
	log.Printf("%d 个钱包地址和私钥已生成并写入文件！", numberOfWallets)
	return nil
}
func GmwsAndWirte(numWallets int, csvFile string) error {
	file, err := os.Create(csvFile)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	// 写入CSV文件头
	err = writer.Write([]string{"Address", "Private Key", "Mnemonic"})
	if err != nil {
		log.Fatalf("Failed to write header to CSV file: %v", err)
		return err
	}
	for i := 0; i < numWallets; i++ {
		address, privateKey, mnemonic, err := GMnemonicW()
		if err != nil {
			log.Fatalf("Failed to generate wallet: %v", err)
			return err
		}
		err = writer.Write([]string{address.Hex(), privateKey, mnemonic})
		if err != nil {
			log.Fatalf("Failed to write wallet to CSV file: %v", err)
			return err
		}
		log.Printf("Generated wallet %d: %s\n", i+1, address.Hex())
	}
	log.Println("All wallets generated and saved to CSV file successfully.")
	return nil
}
func GMnemonicW() (common.Address, string, string, error) {
	// 生成助记词
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return common.Address{}, "", "", err
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return common.Address{}, "", "", err
	}
	// 生成种子
	seed := bip39.NewSeed(mnemonic, "")
	// 从种子生成主私钥
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return common.Address{}, "", "", err
	}
	// 使用 BIP-44 路径 m/44'/60'/0'/0/0 生成子私钥
	purpose, _ := masterKey.NewChildKey(bip32.FirstHardenedChild + 44)
	coinType, _ := purpose.NewChildKey(bip32.FirstHardenedChild + 60)
	account, _ := coinType.NewChildKey(bip32.FirstHardenedChild)
	change, _ := account.NewChildKey(0)
	addressKey, _ := change.NewChildKey(0)
	privateKeyECDSA, err := crypto.ToECDSA(addressKey.Key)
	if err != nil {
		return common.Address{}, "", "", err
	}
	privateKey := fmt.Sprintf("%x", crypto.FromECDSA(privateKeyECDSA))
	address := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)
	return address, privateKey, mnemonic, nil
}
