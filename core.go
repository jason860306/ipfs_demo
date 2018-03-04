package ipfs_core

import (
	"github.com/ipfs/go-ipfs/commands"
	"github.com/ipfs/go-ipfs/core"
	"github.com/op/go-logging"

	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

var (
	VERSION   = "0.11.1"
	USERAGENT = "/Saturn:" + VERSION + "/"
)

var log_core = logging.MustGetLogger("core")

var Node *SaturnNode

type SaturnNode struct {
	// Context for issuing IPFS commands
	Context commands.Context

	// IPFS node object
	IpfsNode *core.IpfsNode

	/* The roothash of the node directory inside the openbazaar repo.
	   This directory hash is published on IPNS at our peer ID making
	   the directory publicly viewable on the network. */
	RootHash string

	// The path to the openbazaar repo in the file system
	RepoPath string

	// Optional nodes to push user data to
	PushNodes []peer.ID

	// The user-agent for this node
	UserAgent string

	// Allow other nodes to push data to this node for storage
	AcceptStoreRequests bool

	// Last ditch API to find records that dropped out of the DHT
	IPNSBackupAPI string
}
