package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ipfs/go-ipfs/commands"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/namesys"
	"github.com/ipfs/go-ipfs/repo/config"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/ipfs/go-ipfs/thirdparty/ds-help"
	"github.com/jason860306/ipfs_demo/ipfs"

	"bufio"
	dhtutil "gx/ipfs/QmUCS9EnqNq1kCnJds2eLDypBiS21aSiCf1MVzSUVB9TGA/go-libp2p-kad-dht/util"
	"gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	recpb "gx/ipfs/QmbxkgUceEcuSZ4ZdBA3x74VUDSSYjHYmmeEqkjxbtZ6Jg/go-libp2p-record/pb"

	"github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/mitchellh/go-homedir"
	namepb "github.com/ipfs/go-ipfs/namesys/pb"
	"github.com/op/go-logging"
	"github.com/tyler-smith/go-bip39"
)

const RepoVersion = "6" // version

var log = logging.MustGetLogger("ipfs_demo")
var ErrRepoExists = errors.New("IPFS configuration file exists. Reinitializing would overwrite your keys. Use -f to force overwrite.") // error message

var DefaultBootstrapAddresses = []string{
	"/ip4/107.170.133.32/tcp/4001/ipfs/QmUZRGLhcKXF1JyuaHgKm23LvqcoMYwtb9jmh8CkP4og3K", // Le Marché Serpette
	"/ip4/139.59.174.197/tcp/4001/ipfs/QmZfTbnpvPwxCjpCG3CXJ7pfexgkBZ2kgChAiRJrTK1HsM", // Brixton Village
	"/ip4/139.59.6.222/tcp/4001/ipfs/QmRDcEDK9gSViAevCHiE6ghkaBCU7rTuQj4BDpmCzRvRYg",   // Johari
	"/ip4/46.101.198.170/tcp/4001/ipfs/QmePWxsFT9wY3QuukgVDB7XZpqdKhrqJTHTXU7ECLDWJqX", // Duo Search
}

/* Returns the directory to store repo data in.
   It depends on the OS and whether or not we are on testnet. */
func GetRepoPath() (string, error) {
	// Set default base path and directory name
	homePath := "~"
	directoryName := ".ipfs_demo"

	// Join the path and directory name, then expand the home path
	fullPath, err := homedir.Expand(filepath.Join(homePath, directoryName))
	if err != nil {
		return "", err
	}

	// Return the shortest lexical representation of the path
	return filepath.Clean(fullPath), nil
}

func checkWriteable(dir string) error {
	_, err := os.Stat(dir)
	if err == nil {
		// Directory exists, make sure we can write to it
		testfile := path.Join(dir, "test")
		fi, err := os.Create(testfile)
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("%s is not writeable by the current user", dir)
			}
			return fmt.Errorf("Unexpected error while checking writeablility of repo root: %s", err)
		}
		fi.Close()
		return os.Remove(testfile)
	}

	if os.IsNotExist(err) {
		// Directory does not exist, check that we can create it
		return os.Mkdir(dir, 0775)
	}

	if os.IsPermission(err) {
		return fmt.Errorf("Cannot write to %s, incorrect permissions", err)
	}

	return err
}

func datastoreConfig(repoRoot string) config.Datastore {
	return config.Datastore{
		StorageMax:         "10GB",
		StorageGCWatermark: 90, // 90%
		GCPeriod:           "1h",
		BloomFilterSize:    0,
		HashOnRead:         false,
		Spec: map[string]interface{}{
			"type": "mount",
			"mounts": []interface{}{
				map[string]interface{}{
					"mountpoint": "/blocks",
					"type":       "measure",
					"prefix":     "flatfs.datastore",
					"child": map[string]interface{}{
						"type":      "flatfs",
						"path":      "blocks",
						"sync":      true,
						"shardFunc": "/repo/flatfs/shard/v1/next-to-last/2",
					},
				},
				map[string]interface{}{
					"mountpoint": "/",
					"type":       "measure",
					"prefix":     "leveldb.datastore",
					"child": map[string]interface{}{
						"type":        "levelds",
						"path":        "datastore",
						"compression": "none",
					},
				},
			},
		},
	}
}

func InitConfig(repoRoot string) (*config.Config, error) {
	bootstrapPeers, err := config.ParseBootstrapPeers(DefaultBootstrapAddresses)
	if err != nil {
		return nil, err
	}

	datastore := datastoreConfig(repoRoot)

	conf := &config.Config{

		// Setup the node's default addresses.
		// NOTE: two swarm listen addrs, one TCP, one UTP.
		Addresses: config.Addresses{
			Swarm: []string{
				"/ip4/0.0.0.0/tcp/4001",
				"/ip6/::/tcp/4001",
				"/ip4/0.0.0.0/tcp/9005/ws",
				"/ip6/::/tcp/9005/ws",
			},
			API:     "",
			Gateway: "/ip4/127.0.0.1/tcp/4002",
		},

		Datastore: datastore,
		Bootstrap: config.BootstrapPeerStrings(bootstrapPeers),
		Discovery: config.Discovery{
			config.MDNS{
				Enabled:  false,
				Interval: 10,
			},
		},

		// Setup the node mount points
		Mounts: config.Mounts{
			IPFS: "/ipfs",
			IPNS: "/ipns",
		},

		Ipns: config.Ipns{
			ResolveCacheSize: 128,
			RecordLifetime:   "7d",
			RepublishPeriod:  "24h",
		},

		Gateway: config.Gateway{
			RootRedirect: "",
			Writable:     false,
			PathPrefixes: []string{},
		},
	}

	return conf, nil
}

func createMnemonic(newEntropy func(int) ([]byte, error), newMnemonic func([]byte) (string, error)) (string, error) {
	entropy, err := newEntropy(128)
	if err != nil {
		return "", err
	}
	mnemonic, err := newMnemonic(entropy)
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}

func addConfigExtensions(repoRoot string) error {
	r, err := fsrepo.Open(repoRoot)
	if err != nil { // NB: repo is owned by the node
		return err
	}
	return r.Close()
}

func initializeIpnsKeyspace(repoRoot string, privKeyBytes []byte) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, err := fsrepo.Open(repoRoot)
	if err != nil { // NB: repo is owned by the node
		return err
	}
	cfg, err := r.Config()
	if err != nil {
		log.Error(err)
		return err
	}
	identity, err := ipfs.IdentityFromKey(privKeyBytes)
	if err != nil {
		return err
	}

	cfg.Identity = identity
	nd, err := core.NewNode(ctx, &core.BuildCfg{Repo: r})
	if err != nil {
		return err
	}
	defer nd.Close()

	err = nd.SetupOfflineRouting()
	if err != nil {
		return err
	}

	return namesys.InitializeKeyspace(ctx, nd.DAG, nd.Namesys, nd.Pinning, nd.PrivateKey)
}

func DoInit(repoRoot string, nBitsForKeypair int, password string, mnemonic string, creationDate time.Time) error {

	if fsrepo.IsInitialized(repoRoot) {
		return ErrRepoExists
	}

	if err := checkWriteable(repoRoot); err != nil {
		return err
	}

	conf, err := InitConfig(repoRoot)
	if err != nil {
		return err
	}

	if mnemonic == "" {
		mnemonic, err = createMnemonic(bip39.NewEntropy, bip39.NewMnemonic)
		if err != nil {
			return err
		}
	}
	seed := bip39.NewSeed(mnemonic, "Secret Passphrase")
	fmt.Printf("Generating Ed25519 keypair...")
	identityKey, err := ipfs.IdentityKeyFromSeed(seed, nBitsForKeypair)
	if err != nil {
		return err
	}
	fmt.Printf("Done\n")

	identity, err := ipfs.IdentityFromKey(identityKey)
	if err != nil {
		return err
	}
	conf.Identity = identity

	log.Infof("Initializing ipfs_demo node at %s\n", repoRoot)
	if err := fsrepo.Init(repoRoot, conf); err != nil {
		return err
	}

	if err := addConfigExtensions(repoRoot); err != nil {
		return err
	}

	f, err := os.Create(path.Join(repoRoot, "repover"))
	if err != nil {
		return err
	}
	_, werr := f.Write([]byte(RepoVersion))
	if werr != nil {
		return werr
	}
	f.Close()

	return initializeIpnsKeyspace(repoRoot, identityKey)
}

func InitializeRepo(dataDir, password, mnemonic string, creationDate time.Time) error {
	// Initialize the IPFS repo if it does not already exist
	return DoInit(dataDir, 4096, password, mnemonic, creationDate)
}

// Prints the addresses of the host
func printSwarmAddrs(node *core.IpfsNode) {
	var addrs []string
	for _, addr := range node.PeerHost.Addrs() {
		addrs = append(addrs, addr.String())
	}
	sort.Sort(sort.StringSlice(addrs))

	for _, addr := range addrs {
		log.Infof("Swarm listening on %s\n", addr)
	}
}

func main() {
	//=========================================== Init ===========================================
	// Set repo path
	repoPath, err := GetRepoPath()
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(repoPath)

	passwd := strings.Replace("123456", "'", "''", -1)
	//Mnemonic := func() string {
	//	t := time.Now()
	//	h := md5.New()
	//	io.WriteString(h, "ipfs_demo")
	//	io.WriteString(h, t.String())
	//	passwd := fmt.Sprintf("%x", h.Sum(nil))
	//	return passwd
	//}()
	Mnemonic := ""
	creationDate := time.Now()

	err = InitializeRepo(repoPath, passwd, Mnemonic, creationDate)
	if err == ErrRepoExists {
		//reader := bufio.NewReader(os.Stdin)
		fmt.Print("Force overwriting the db will destroy your existing keys and history. Are you really, really sure you want to continue? (y/n): ")
		//resp, _ := reader.ReadString('\n')
		resp := "yes\n"
		if strings.ToLower(resp) == "y\n" || strings.ToLower(resp) == "yes\n" || strings.ToLower(resp)[:1] == "y" {
			os.RemoveAll(repoPath)
			err = InitializeRepo(repoPath, passwd, Mnemonic, creationDate)
			if err != nil {
				os.Exit(1)
			}
			fmt.Printf("ipfs_demo repo initialized at %s\n", repoPath)
		} else {
			os.Exit(1)
		}
	} else if err != nil {
		os.Exit(1)
	}
	fmt.Printf("ipfs_demo repo initialized at %s\n", repoPath)

	//=========================================== Start ===========================================
	// IPFS node setup
	r, err := fsrepo.Open(repoPath)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	cctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := r.Config()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	//identityKey, err := sqliteDB.Config().GetIdentityKey()
	//if err != nil {
	//	log.Error(err)
	//	os.Exit(1)
	//}
	//identity, err := ipfs.IdentityFromKey(identityKey)
	//if err != nil {
	//	os.Exit(1)
	//}
	//cfg.Identity = identity

	ncfg := &core.BuildCfg{
		Repo:   r,
		Online: true,
		ExtraOpts: map[string]bool{
			"mplex": true,
		},
		//DNSResolver: dnsResolver,
		//Routing:     DHTOption,
	}

	nd, err := core.NewNode(cctx, ncfg)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// Set IPNS query size
	querySize := cfg.Ipns.QuerySize
	if querySize <= 20 && querySize > 0 {
		dhtutil.QuerySize = int(querySize)
	} else {
		dhtutil.QuerySize = 16
	}
	namesys.UsePersistentCache = cfg.Ipns.UsePersistentCache

	log.Info("Peer ID: ", nd.Identity.Pretty())
	printSwarmAddrs(nd)

	// Get current directory root hash
	_, ipnskey := namesys.IpnsKeysForID(nd.Identity)
	ival, hasherr := nd.Repo.Datastore().Get(dshelp.NewKeyFromBinary([]byte(ipnskey)))
	if hasherr != nil {
		log.Error(hasherr)
		os.Exit(1)
	}
	val := ival.([]byte)
	dhtrec := new(recpb.Record)
	proto.Unmarshal(val, dhtrec)
	e := new(namepb.IpnsEntry)
	proto.Unmarshal(dhtrec.GetValue(), e)

	fmt.Printf("Daemon is ready\n")
	//=========================================== Add ===========================================

	ctx := commands.Context{}
	ctx.Online = true
	ctx.ConfigRoot = repoPath
	ctx.LoadConfig = func(path string) (*config.Config, error) {
		return fsrepo.ConfigAt(repoPath)
	}
	ctx.ConstructNode = func() (*core.IpfsNode, error) {
		return nd, nil
	}
	hash, err := ipfs.AddFile(ctx, path.Join("./", "README.md"))
	if err != nil {
		log.Info(err.Error())
		os.Exit(1)
	}
	//test.bin: zdj7WdnQBd3Yf4KPuUTZ9mkAQ6Rfd87H4h2f7d3KxzgW4kJ9U
	//README.md on linux:   zb2rhneqJaf4y9vQpb9o1yqyejARwiR9PDuz8bXjRTAE5iLT9
	//README.md on windows: zb2rhjwNxFKtD3Qg4nV3Qf4CH77bvEn7ndzM4ysXCwxvpLXeo
	if hash != "zb2rhjwNxFKtD3Qg4nV3Qf4CH77bvEn7ndzM4ysXCwxvpLXeo" {
		log.Info("Ipfs add file failed")
	} else {
		log.Info("Ipfs add file successfully: ", hash)
	}

	//=========================================== Cat ===========================================
	dataText, err := ipfs.Cat(ctx, hash, time.Second*10)
	if err != nil {
		log.Info(err.Error())
		os.Exit(1)
	} else {
		log.Infof("Cat %s as follow:\n%s", hash, dataText)
	}

	//=========================================== Swarm peers ===========================================
	for i := 0; i < 3; i++ {
		<-time.After(2 * time.Second)

		pbool := make(chan []string)
		go func() {
			peers, err := ipfs.ConnectedPeers(ctx)
			if err != nil {
				errInfo := make([]string, 1)
				errInfo = append(errInfo, err.Error())
				pbool <- errInfo
				log.Info(err.Error())
			}
			pbool <- peers
		}()
		peers := <-pbool
		if len(peers) == 0 {
			log.Infof("No peers in swarm")
		} else {
			for i, peer := range peers {
				log.Infof("peer #%d: %s\n", i, peer)
			}
			break
		}
	}

	//=========================================== Connect peers ===========================================
	peer := "/ip4/192.168.0.108/tcp/4001/ipfs/QmXtnUAnkG6evE2R9Qi5HMwkU81dAmb6RMKaqyrNRF1XWH"
	peers, err := ipfs.ConnectTo(ctx, peer)
	if err != nil {
		log.Info(err.Error())
		os.Exit(1)
	}
	log.Infof("connect %s successfully\n", peer)
	for i, peer := range peers {
		log.Infof("#%d: %s\n", i, peer)
	}

	//=========================================== Get ===========================================
	for i := 0; i < 3; i++ {
		<-time.After(2 * time.Second)

		// bbool := make(chan []byte)
		// cbool := make(chan bool)
		// go func() {
		d, err := ipfs.Get(ctx, hash, time.Second*10)
		if err != nil {
			// cbool <- false
			// bbool <- []byte(err.Error())
			log.Info(err.Error())
		} else {
			// cbool <- true
			// bbool <- d
			log.Infof("Get %s Ok!", hash)
		}
		// }()
		// d := <-bbool
		// getOk := <-cbool
		if /*getOk*/ len(d) != 0 {
			log.Infof("%s", d)
			break
		}
	}

	//=========================================== End ===========================================
	fmt.Print("Press 'Enter' to continue ...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	os.Exit(1)
}
