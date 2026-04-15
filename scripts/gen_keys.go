package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func main() {
	if err := os.MkdirAll("secrets", 0755); err != nil {
		fmt.Println("Error creating secrets folder:", err)
		return
	}

	// Generate RSA key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("Error generating key:", err)
		return
	}

	// Save Private Key
	privFile, err := os.Create("secrets/jwt_private.pem")
	if err != nil {
		fmt.Println("Error creating private key file:", err)
		return
	}
	defer privFile.Close()

	privBytes := x509.MarshalPKCS1PrivateKey(privKey)
	if err := pem.Encode(privFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}); err != nil {
		fmt.Println("Error encoding private key:", err)
		return
	}

	// Save Public Key
	pubFile, err := os.Create("secrets/jwt_public.pem")
	if err != nil {
		fmt.Println("Error creating public key file:", err)
		return
	}
	defer pubFile.Close()

	pubBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		fmt.Println("Error marshaling public key:", err)
		return
	}
	if err := pem.Encode(pubFile, &pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}); err != nil {
		fmt.Println("Error encoding public key:", err)
		return
	}
}
