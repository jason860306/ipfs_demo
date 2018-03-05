package ipfs_core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/op/go-logging"

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/namesys"
	"github.com/ipfs/go-ipfs/repo/config"
	"github.com/ipfs/go-ipfs/repo/fsrepo"

	"github.com/jason860306/ipfs_demo/ipfs_cmds"
)

const (
	RepoVersion   = "6" // version
	BitForKeyPair = 4096
)

var log_repo = logging.MustGetLogger("repo")
var errRepoExists = errors.New("IPFS configuration file exists. Reinitializing would overwrite your keys. Use -f to force overwrite.") // error message

var defBootstrapAddrs = []string{
	//"/ip4/107.170.133.32/tcp/4001/ipfs/QmUZRGLhcKXF1JyuaHgKm23LvqcoMYwtb9jmh8CkP4og3K", // Le Marché Serpette
	//"/ip4/139.59.174.197/tcp/4001/ipfs/QmZfTbnpvPwxCjpCG3CXJ7pfexgkBZ2kgChAiRJrTK1HsM", // Brixton Village
	//"/ip4/139.59.6.222/tcp/4001/ipfs/QmRDcEDK9gSViAevCHiE6ghkaBCU7rTuQj4BDpmCzRvRYg",   // Johari
	//"/ip4/46.101.198.170/tcp/4001/ipfs/QmePWxsFT9wY3QuukgVDB7XZpqdKhrqJTHTXU7ECLDWJqX", // Duo Search
	"/ip4/138.197.232.22/tcp/4001/ipfs/Qmeb6UB2vxhyQkb6ntkwaSVFnKjiQfPxuGWs1FtAXftSJc", // szj0306@do
	"/ip4/35.194.251.184/tcp/4001/ipfs/QmPgwsvLnrePX1ez8fbqd6gfLY3YWvrVfjtawV9X9mQhv7", // szj0306@gcp
}

/* Returns the directory to store repo data in.
   It depends on the OS and whether or not we are on testnet. */
func getRepoPath() (string, error) {
	// Set default base path and directory name
	directoryName := ".saturn"

	// Join the path and directory name, then expand the home path
	fullPath, err := homedir.Expand(filepath.Join("~", directoryName))
	if err != nil {
		return "", err
	}

	// Return the shortest lexical representation of the path
	return filepath.Clean(fullPath), nil
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

func initConfig(repoRoot string) (*config.Config, error) {
	bootstrapPeers, err := config.ParseBootstrapPeers(defBootstrapAddrs)
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
		Discovery: config.Discovery{config.MDNS{
			Enabled:  true,
			Interval: 10,
		}},

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

func initializeIpnsKeyspace(repoRoot string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, err := fsrepo.Open(repoRoot)
	if err != nil { // NB: repo is owned by the node
		return err
	}

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

func doInit(repoRoot string, nBitsForKeypair int) error {

	if fsrepo.IsInitialized(repoRoot) {
		return errRepoExists
	}

	if err := CheckWriteable(repoRoot); err != nil {
		return err
	}

	conf, err := initConfig(repoRoot)
	if err != nil {
		return err
	}

	identity, err := ipfs_cmds.IdentityConfig(nBitsForKeypair)
	if err != nil {
		return err
	}
	conf.Identity = identity

	log_repo.Infof("Initializing ipfs_demo node at %s\n", repoRoot)
	if err := fsrepo.Init(repoRoot, conf); err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(repoRoot, "repover"))
	if err != nil {
		return err
	}
	_, werr := f.Write([]byte(RepoVersion))
	if werr != nil {
		return werr
	}
	f.Close()

	return initializeIpnsKeyspace(repoRoot)
}

func initializeRepo(dataDir string) error {
	// Initialize the IPFS repo if it does not already exist
	return doInit(dataDir, BitForKeyPair)
}

func Init() (repoPtah string, err error) {
	//=========================================== Init ===========================================
	// Set repo path
	repoPath, err := getRepoPath()
	if err != nil {
		return "", err
	}
	fmt.Println(repoPath)

	err = initializeRepo(repoPath)
	if err != nil && err != errRepoExists {
		return "", err
	}
	if err == errRepoExists {
		//reader := bufio.NewReader(os.Stdin)
		fmt.Print("Force overwriting the db will destroy your existing keys and history. Are you really, really sure you want to continue? (y/n): ")
		//resp, _ := reader.ReadString('\n')
		resp := "yes\n"
		if strings.ToLower(resp) == "y\n" || strings.ToLower(resp) == "yes\n" || strings.ToLower(resp)[:1] == "y" {
			os.RemoveAll(repoPath)
			err = initializeRepo(repoPath)
			if err != nil {
				return "", err
			}
			fmt.Printf("ipfs_demo repo initialized at %s\n", repoPath)
		}
	}
	return repoPath, nil
}
