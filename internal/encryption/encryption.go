package encryption

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/andynikk/advancedmetrics/internal/constants"
)

type RsaPublicKey struct {
	*rsa.PublicKey
}

type RsaPrivateKey struct {
	*rsa.PrivateKey
}

func (rk *RsaPublicKey) RsaEncrypt(msg []byte) ([]byte, error) {
	encryptedBytes, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rk.PublicKey, msg, nil)
	return encryptedBytes, err
}

func (rk *RsaPrivateKey) RsaDecrypt(msgByte []byte) ([]byte, error) {
	decryptedBytes, err := rk.PrivateKey.Decrypt(nil, msgByte, &rsa.OAEPOptions{Hash: crypto.SHA256})
	return decryptedBytes, err
}

func CreateCert() ([]bytes.Buffer, error) {
	var numSert int64
	var subjectKeyId string
	var lenKeyByte int

	fmt.Print("Введите уникальный номер сертификата: ")
	if _, err := fmt.Fscan(os.Stdin, &numSert); err != nil {
		constants.Logger.ErrorLog(err)
		return nil, err
	}

	fmt.Print("Введите ИД ключа субъекта (пример ввода 12346): ")
	if _, err := fmt.Fscan(os.Stdin, &subjectKeyId); err != nil {
		constants.Logger.ErrorLog(err)
		return nil, err
	}

	fmt.Print("Длина ключа в байтах: ")
	if _, err := fmt.Fscan(os.Stdin, &lenKeyByte); err != nil {
		constants.Logger.ErrorLog(err)
		return nil, err
	}

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(numSert),
		Subject: pkix.Name{
			Organization: []string{"AdvancedMetrics"},
			Country:      []string{"RU"},
		},
		NotBefore: time.Now(),
		NotAfter: time.Now().AddDate(constants.TimeLivingCertificateYaer, constants.TimeLivingCertificateMounth,
			constants.TimeLivingCertificateDay),
		SubjectKeyId: []byte(subjectKeyId),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, lenKeyByte)
	if err != nil {
		return nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	var certPEM bytes.Buffer
	_ = pem.Encode(&certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	var privateKeyPEM bytes.Buffer
	_ = pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return []bytes.Buffer{certPEM, privateKeyPEM}, nil
}

func SaveKeyInFile(key *bytes.Buffer, pathFile string) {
	file, err := os.Create(pathFile)
	if err != nil {
		return
	}
	_, err = file.WriteString(key.String())
	if err != nil {
		return
	}
}
