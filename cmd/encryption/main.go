package main

import (
	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/encryption"
)

func main() {
	arrCert, err := encryption.CreateCert()
	if err != nil {
		constants.Logger.ErrorLog(err)
		return
	}

	encryption.SaveKeyInFile(&arrCert[0], "publicKey.cer")
	encryption.SaveKeyInFile(&arrCert[1], "privateKey.pfx")

	//pvkData, _ := os.ReadFile("privateKey.pfx")
	//pvkBlock, _ := pem.Decode(pvkData)
	//pvk, err := x509.ParsePKCS1PrivateKey(pvkBlock.Bytes)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//certData, _ := os.ReadFile("publicKey.cer")
	//certBlock, _ := pem.Decode(certData)
	//cert, err := x509.ParseCertificate(certBlock.Bytes)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//rk := cert.PublicKey.(*rsa.PublicKey)
	//encryptedBytes, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rk, []byte("super secret message"), nil)
	//if err != nil {
	//	panic(err)
	//}
	//
	//decryptedBytes, err := pvk.Decrypt(nil, encryptedBytes, &rsa.OAEPOptions{Hash: crypto.SHA256})
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println("decrypted message: ", string(decryptedBytes))

}
