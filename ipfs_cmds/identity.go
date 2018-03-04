package ipfs_cmds

import (
	"fmt"
	//"bytes"
	//"crypto/hmac"
	//"crypto/sha256"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/ipfs/go-ipfs/repo/config"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	libp2p "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

func PeerIdFromPubKey(pk libp2p.PubKey) (string, error) {
	id, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return "", err
	}
	return id.Pretty(), nil
}

// identityConfig initializes a new identity.
func IdentityConfig(nbits int) (config.Identity, error) {
	// TODO guard higher up
	ident := config.Identity{}
	if nbits < 1024 {
		return ident, errors.New("Bitsize less than 1024 is considered unsafe.")
	}

	fmt.Printf("generating %v-bit RSA keypair...", nbits)
	sk, pk, err := libp2p.GenerateKeyPairWithReader(libp2p.RSA, nbits, rand.Reader)
	if err != nil {
		return ident, err
	}
	fmt.Printf("done\n")

	// currently storing key unencrypted. in the future we need to encrypt it.
	// TODO(security)
	skbytes, err := sk.Bytes()
	if err != nil {
		return ident, err
	}
	ident.PrivKey = base64.StdEncoding.EncodeToString(skbytes)

	id, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return ident, err
	}
	ident.PeerID = id.Pretty()
	fmt.Printf("peer identity: %s\n", ident.PeerID)
	return ident, nil
}
