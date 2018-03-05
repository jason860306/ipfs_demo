package ipfs_core

import (
	"context"
	"sort"

	"github.com/ipfs/go-ipfs/commands"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/namesys"
	"github.com/ipfs/go-ipfs/repo/config"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/ipfs/go-ipfs/thirdparty/ds-help"
	"github.com/op/go-logging"

	namepb "github.com/ipfs/go-ipfs/namesys/pb"
	ipath "github.com/ipfs/go-ipfs/path"
	ipfsrepo "github.com/ipfs/go-ipfs/repo"

	"gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	"gx/ipfs/QmT7PnPxYkeKPCG8pAnucfcjrXc15Q7FgvFv7YC24EPrw8/go-libp2p-kad-dht"
	"gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"

	dhtutil "gx/ipfs/QmUCS9EnqNq1kCnJds2eLDypBiS21aSiCf1MVzSUVB9TGA/go-libp2p-kad-dht/util"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	p2phost "gx/ipfs/QmaSxYRuMq4pkpBBG2CYaRrPx2z7NmMVEs34b9g61biQA6/go-libp2p-host"
	recpb "gx/ipfs/QmbxkgUceEcuSZ4ZdBA3x74VUDSSYjHYmmeEqkjxbtZ6Jg/go-libp2p-record/pb"
)

var log_start = logging.MustGetLogger("start")

var DHTOption core.RoutingOption = constructDHTRouting

// Prints the addresses of the host
func printSwarmAddrs(node *core.IpfsNode) {
	var addrs []string
	for _, addr := range node.PeerHost.Addrs() {
		addrs = append(addrs, addr.String())
	}
	sort.Sort(sort.StringSlice(addrs))

	for _, addr := range addrs {
		log_start.Infof("Swarm listening on %s\n", addr)
	}
}

func constructDHTRouting(ctx context.Context, host p2phost.Host, dstore ipfsrepo.Datastore) (routing.IpfsRouting, error) {
	dhtRouting := dht.NewDHT(ctx, host, dstore)
	dhtRouting.Validator[core.IpnsValidatorTag] = namesys.IpnsRecordValidator
	dhtRouting.Selector[core.IpnsValidatorTag] = namesys.IpnsSelectorFunc
	return dhtRouting, nil
}

func Start(repoPath string) (e error) {
	//=========================================== Start ===========================================
	// IPFS node setup
	r, err := fsrepo.Open(repoPath)
	if err != nil {
		log_start.Error(err)
		return err
	}
	cctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := r.Config()
	if err != nil {
		log_start.Error(err)
		return err
	}

	ncfg := &core.BuildCfg{
		Repo:   r,
		Online: true,
		ExtraOpts: map[string]bool{
			"mplex": true,
		},
		DNSResolver: namesys.NewDNSResolver(),
		Routing:     DHTOption,
	}

	nd, err := core.NewNode(cctx, ncfg)
	if err != nil {
		log_start.Error(err)
		return err
	}
	nd.SetLocal(false)

	ctx := commands.Context{}
	ctx.Online = true
	ctx.ConfigRoot = repoPath
	ctx.LoadConfig = func(path string) (*config.Config, error) {
		return fsrepo.ConfigAt(repoPath)
	}
	ctx.ConstructNode = func() (*core.IpfsNode, error) {
		return nd, nil
	}

	// Set IPNS query size
	querySize := cfg.Ipns.QuerySize
	if querySize <= 20 && querySize > 0 {
		dhtutil.QuerySize = int(querySize)
	} else {
		dhtutil.QuerySize = 16
	}
	namesys.UsePersistentCache = cfg.Ipns.UsePersistentCache

	log_start.Info("Peer ID: ", nd.Identity.Pretty())
	printSwarmAddrs(nd)

	// Get current directory root hash
	_, ipnskey := namesys.IpnsKeysForID(nd.Identity)
	ival, err := nd.Repo.Datastore().Get(dshelp.NewKeyFromBinary([]byte(ipnskey)))
	if err != nil {
		log_start.Error(err)
		return err
	}
	val := ival.([]byte)
	dhtrec := new(recpb.Record)
	proto.Unmarshal(val, dhtrec)
	ipnsEntry := new(namepb.IpnsEntry)
	proto.Unmarshal(dhtrec.GetValue(), ipnsEntry)

	// Push nodes
	var pushNodes []peer.ID
	//for _, pnd := range dataSharing.PushTo {
	//	p, err := peer.IDB58Decode(pnd)
	//	if err != nil {
	//		log_start.Error("Invalid peerID in DataSharing config")
	//		return err
	//	}
	//	pushNodes = append(pushNodes, p)
	//}

	Node = &SaturnNode{
		Context:             ctx,
		IpfsNode:            nd,
		RootHash:            ipath.Path(ipnsEntry.Value).String(),
		RepoPath:            repoPath,
		PushNodes:           pushNodes,
		UserAgent:           USERAGENT,
		AcceptStoreRequests: true,
		IPNSBackupAPI:       cfg.Ipns.BackUpAPI,
	}

	return nil
}
