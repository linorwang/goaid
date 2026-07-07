package pay

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

func TestUnionPaySignAndVerify(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(123456),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	client, err := NewUnionPayClient(UnionPayConfig{
		MerchantID:    "777290058110048",
		PrivateKeyPEM: string(privateKeyPEM),
		SignCertPEM:   string(certPEM),
		VerifyCertPEM: string(certPEM),
	})
	if err != nil {
		t.Fatalf("new unionpay client: %v", err)
	}

	signed, err := client.SignParams(map[string]string{
		"orderId": "202607070001",
		"txnAmt":  "100",
	})
	if err != nil {
		t.Fatalf("sign params: %v", err)
	}
	if err := client.VerifyParams(signed); err != nil {
		t.Fatalf("verify params: %v", err)
	}
}
