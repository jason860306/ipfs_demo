package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"

	"crypto/rand"
	"encoding/pem"

	libp2p "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

func exportRsaPrivateKeyAsPemStr(privkey libp2p.PrivKey) string {
	rsaSK, ok := privkey.(*libp2p.RsaPrivateKey)
	if !ok {
		return ""
	}
	privkey_bytes := libp2p.MarshalRsaPrivateKey(rsaSK)
	privkey_pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkey_bytes,
		},
	)
	return string(privkey_pem)
}

func exportRsaPrivateKey(privkey libp2p.PrivKey) []byte {
	rsaSK, ok := privkey.(*libp2p.RsaPrivateKey)
	if !ok {
		return nil
	}
	return libp2p.MarshalRsaPrivateKey(rsaSK)
}

func parseRsaPrivateKeyFromPemStr(privPEM string) (libp2p.PrivKey, error) {

	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	privkey, err := libp2p.UnmarshalRsaPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return privkey, nil
}

func parseRsaPrivateKey(skbytes []byte) (libp2p.PrivKey, error) {
	privkey, err := libp2p.UnmarshalRsaPrivateKey(skbytes)
	if err != nil {
		return nil, err
	}
	return privkey, nil
}

func exportRsaPublicKeyAsPemStr(pubkey libp2p.PubKey) (string, error) {
	rsaPK, ok := pubkey.(*libp2p.RsaPublicKey)
	if !ok {
		os.Exit(1)
	}
	pubkey_bytes, err := libp2p.MarshalRsaPublicKey(rsaPK)
	if err != nil {
		return "", err
	}
	pubkey_pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pubkey_bytes,
		},
	)

	return string(pubkey_pem), nil
}

func exportRsaPublicKey(pubkey libp2p.PubKey) ([]byte, error) {
	rsaPK, ok := pubkey.(*libp2p.RsaPublicKey)
	if !ok {
		return nil, errors.New("fetch rsapubkey failed")
	}
	pubkey_bytes, err := libp2p.MarshalRsaPublicKey(rsaPK)
	if err != nil {
		return nil, err
	}
	return pubkey_bytes, nil
}

func parseRsaPublicKeyFromPemStr(pubPEM string) (libp2p.PubKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	pub, err := libp2p.UnmarshalRsaPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pub, nil
}

func parseRsaPublicKey(pkbytes []byte) (libp2p.PubKey, error) {
	pub, err := libp2p.UnmarshalRsaPublicKey(pkbytes)
	if err != nil {
		return nil, err
	}
	return pub, nil
}

func TestIpfsKey(t *testing.T) {
	const nBitsForKeypair = 4096
	skMarshal, _, err := libp2p.GenerateKeyPairWithReader(libp2p.RSA, nBitsForKeypair, rand.Reader /*reader*/)
	if err != nil {
		os.Exit(1)
	}

	//=========================================== via pem ===========================================
	// Export the keys to pem string
	priv_pem := exportRsaPrivateKeyAsPemStr(skMarshal)
	pub_pem, _ := exportRsaPublicKeyAsPemStr(skMarshal.GetPublic())

	// Import the keys from pem string
	priv_parsed, _ := parseRsaPrivateKeyFromPemStr(priv_pem)
	pub_parsed, _ := parseRsaPublicKeyFromPemStr(pub_pem)

	// Export the newly imported keys
	priv_parsed_pem := exportRsaPrivateKeyAsPemStr(priv_parsed)
	pub_parsed_pem, _ := exportRsaPublicKeyAsPemStr(pub_parsed)

	fmt.Println(priv_parsed_pem)
	fmt.Println(pub_parsed_pem)

	// Check that the exported/imported keys match the original keys
	if priv_pem != priv_parsed_pem && pub_pem != pub_parsed_pem {
		fmt.Println("Failure: Export and Import did not result in same Keys")
	} else {
		fmt.Println("Success")
	}

	//=========================================== via rawdata ===========================================
	// Export the keys to pem string
	priv_pem1 := exportRsaPrivateKey(skMarshal)
	pub_pem1, _ := exportRsaPublicKey(skMarshal.GetPublic())

	// Import the keys from pem string
	priv_parsed1, _ := parseRsaPrivateKey(priv_pem1)
	pub_parsed1, _ := parseRsaPublicKey(pub_pem1)

	// Export the newly imported keys
	priv_parsed_pem1 := exportRsaPrivateKey(priv_parsed1)
	pub_parsed_pem1, _ := exportRsaPublicKey(pub_parsed1)

	fmt.Printf("%v\n", priv_parsed_pem1)
	fmt.Printf("%v\n", pub_parsed_pem1)

	// Check that the exported/imported keys match the original keys
	if !bytes.Equal(priv_pem1, priv_parsed_pem1) ||
		!bytes.Equal(pub_pem1, pub_parsed_pem1) {
		fmt.Println("Failure: Export and Import did not result in same Keys")
	} else {
		fmt.Println("Success")
	}
}
