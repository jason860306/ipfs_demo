package ipfs_core

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ipfs/go-ipfs/commands"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/namesys"
	namepb "github.com/ipfs/go-ipfs/namesys/pb"
	ipfsrepo "github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/config"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/ipfs/go-ipfs/thirdparty/ds-help"

	dhtutil "gx/ipfs/QmUCS9EnqNq1kCnJds2eLDypBiS21aSiCf1MVzSUVB9TGA/go-libp2p-kad-dht/util"
	"gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	recpb "gx/ipfs/QmbxkgUceEcuSZ4ZdBA3x74VUDSSYjHYmmeEqkjxbtZ6Jg/go-libp2p-record/pb"

	"gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	"gx/ipfs/QmT7PnPxYkeKPCG8pAnucfcjrXc15Q7FgvFv7YC24EPrw8/go-libp2p-kad-dht"
	p2phost "gx/ipfs/QmaSxYRuMq4pkpBBG2CYaRrPx2z7NmMVEs34b9g61biQA6/go-libp2p-host"

	"github.com/jason860306/ipfs_demo/ipfs_cmds"
)

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

var DHTOption core.RoutingOption = constructDHTRouting

func constructDHTRouting(ctx context.Context, host p2phost.Host, dstore ipfsrepo.Datastore) (routing.IpfsRouting, error) {
	dhtRouting := dht.NewDHT(ctx, host, dstore)
	dhtRouting.Validator[core.IpnsValidatorTag] = namesys.IpnsRecordValidator
	dhtRouting.Selector[core.IpnsValidatorTag] = namesys.IpnsSelectorFunc
	return dhtRouting, nil
}

func TestIpfs(t *testing.T) {
	args := flag.Args()
	os.Args = append([]string{os.Args[0]}, args...)
	if len(os.Args) != 2 {
		fmt.Printf("usage: %s [cli|srv]", os.Args[0])
		os.Exit(1)
	}
	//=========================================== Init ===========================================
	// Set repo path
	repoPath, err := GetRepoPath()
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(repoPath)

	passwd := strings.Replace("123456", "'", "''", -1)
	Mnemonic := ""
	creationDate := time.Now()

	err = InitializeRepo(repoPath, passwd, Mnemonic, creationDate)
	if err == ErrRepoExists {
		//reader := bufio.NewReader(os.Stdin)
		fmt.Print("Force overwriting the db will destroy your existing keys and history. Are you really, really sure you want to continue? (y/n): ")
		//resp, _ := reader.ReadString('\n')
		resp := "no\n"
		if strings.ToLower(resp) == "y\n" || strings.ToLower(resp) == "yes\n" || strings.ToLower(resp)[:1] == "y" {
			os.RemoveAll(repoPath)
			err = InitializeRepo(repoPath, passwd, Mnemonic, creationDate)
			if err != nil {
				os.Exit(1)
			}
			fmt.Printf("ipfs_demo repo initialized at %s\n", repoPath)
		} else {
			//os.Exit(1)
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
		log.Error(err)
		os.Exit(1)
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

	//=========================================== Set ipfs log level ===========================================
	//logmsg, logerr := ipfs.Log(ctx, "all", "debug")
	//if logerr != nil {
	//	log.Error(logerr.Error())
	//} else {
	//	log.Info(logmsg)
	//}

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

	if os.Args[1] == "srv" {
		//=========================================== Add ===========================================
		//README.md on linux:   zb2rhneqJaf4y9vQpb9o1yqyejARwiR9PDuz8bXjRTAE5iLT9
		//README.md on windows: zb2rhjwNxFKtD3Qg4nV3Qf4CH77bvEn7ndzM4ysXCwxvpLXeo
		hash, err := ipfs_cmds.AddFile(ctx, path.Join("./", "README.md"))
		if err != nil {
			log.Info(err.Error())
			os.Exit(1)
		}
		if runtime.GOOS == "windows" && hash != "zb2rhjwNxFKtD3Qg4nV3Qf4CH77bvEn7ndzM4ysXCwxvpLXeo" {
			log.Info("Ipfs add file on windows failed")
		} else if hash != "zb2rhneqJaf4y9vQpb9o1yqyejARwiR9PDuz8bXjRTAE5iLT9" {
			log.Info("Ipfs add file on uni* failed")
		} else {
			log.Info("Ipfs add file successfully: ", hash)
		}
		//test.bin: zdj7WdnQBd3Yf4KPuUTZ9mkAQ6Rfd87H4h2f7d3KxzgW4kJ9U
		hash, err = ipfs_cmds.AddFile(ctx, path.Join("./resource", "test.bin"))
		if err != nil {
			log.Info(err.Error())
			os.Exit(1)
		}
		if hash != "zb2rhjwNxFKtD3Qg4nV3Qf4CH77bvEn7ndzM4ysXCwxvpLXeo" {
			log.Info("Ipfs add file failed")
		} else {
			log.Info("Ipfs add file successfully: ", hash)
		}
	} else if os.Args[1] == "cli" {
		//// =========================================== Connect peers ===========================================
		////peer := "/ip4/192.168.222.180/tcp/4001/ipfs/QmXZf95S6CDpzyeyFGr8SU2b3hm68FrxUyRpTgTR5YxZ56"
		////peer := "/ip4/138.197.232.22/tcp/4001/ipfs/QmZjmQH4e7opwmeFc23vUZ4nwuPw1oJgFKSpJoAJgrpQiy"
		//peer := "/ip4/97.64.43.18/tcp/4001/ipfs/QmPUrqtsYZzpebQ4sYHiQqjtTGCEGUVu194jhHuVpBnGb3"
		//for i := 0; i < 5; i++ {
		//	peers, err := ipfs.ConnectTo(ctx, peer)
		//	if err != nil {
		//		log.Info(err.Error())
		//		//os.Exit(1)
		//		continue
		//	}
		//	log.Infof("connect %s successfully\n", peer)
		//	for i, peer := range peers {
		//		log.Infof("#%d: %s\n", i, peer)
		//	}
		//	break
		//}
	}

	////=========================================== Cat ===========================================
	var file_hash string
	if runtime.GOOS == "windows" {
		file_hash = "zb2rhjwNxFKtD3Qg4nV3Qf4CH77bvEn7ndzM4ysXCwxvpLXeo"
	} else {
		file_hash = "zb2rhneqJaf4y9vQpb9o1yqyejARwiR9PDuz8bXjRTAE5iLT9"
	}
	for i := 0; i < 3; i++ {
		dataText, err := ipfs_cmds.Cat(ctx, file_hash, time.Second*120)
		if err != nil {
			log.Info(err.Error())
			<-time.After(1 * time.Second)
			continue
		} else {
			log.Infof("Cat %s as follow:\n%s", file_hash, dataText)
			break
		}
	}

	//=========================================== Swarm peers ===========================================
	go func() {
		for {
			pbool := make(chan []string)
			go func() {
				peers, err := ipfs_cmds.ConnectedPeers(ctx)
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
				//break
			}
			<-time.After(1 * time.Second)
		}
	}()

	var fhash []string
	fhash = append(fhash, "zdj7WdnQBd3Yf4KPuUTZ9mkAQ6Rfd87H4h2f7d3KxzgW4kJ9U")
	if runtime.GOOS == "windows" {
		fhash = append(fhash, "zb2rhjwNxFKtD3Qg4nV3Qf4CH77bvEn7ndzM4ysXCwxvpLXeo")
	} else {
		fhash = append(fhash, "zb2rhneqJaf4y9vQpb9o1yqyejARwiR9PDuz8bXjRTAE5iLT9")
	}
	//=========================================== Pin ===========================================
	// pin add
	for _, hash := range fhash {
		for j := 0; j < 3; j++ {
			err := ipfs_cmds.Pin(ctx, hash)
			if err != nil {
				log.Info(err.Error())
				<-time.After(1 * time.Second)
			} else {
				log.Infof("Pin %s Ok!", hash)
				break
			}
		}
	}
	// pin ls
	objs1, err := ipfs_cmds.PinLs(ctx)
	if err != nil {
		log.Info(err.Error())
	} else {
		for i, obj := range objs1 {
			log.Infof("obj #%d: %s\n", i, obj)
		}
	}
	// unpin
	for _, hash := range fhash {
		for j := 0; j < 3; j++ {
			err := ipfs_cmds.UnPinDir(ctx, hash)
			if err != nil {
				log.Info(err.Error())
				<-time.After(1 * time.Second)
			} else {
				log.Infof("UnPin %s Ok!", hash)
				break
			}
		}
	}
	// pin ls
	objs2, err := ipfs_cmds.PinLs(ctx)
	if err != nil {
		log.Info(err.Error())
	} else {
		for i, obj := range objs2 {
			log.Infof("obj #%d: %s\n", i, obj)
		}
	}

	//=========================================== Get ===========================================
	for _, hash := range fhash {
		for j := 0; j < 3; j++ {
			home, herr := homedir.Dir()
			if herr != nil {
				log.Error(herr.Error())
				os.Exit(1)
			}
			var fnamebuf bytes.Buffer
			fnamebuf.WriteString(hash)
			fnamebuf.WriteString("_")
			fnamebuf.WriteString(strconv.Itoa(j))
			ofpath := filepath.Join(home, fnamebuf.String())
			d, err := ipfs_cmds.Get(ctx, hash, ofpath, time.Second*120)
			if err != nil {
				log.Info(err.Error())
				<-time.After(1 * time.Second)
			} else {
				log.Infof("Get %s Ok!", d)
				break
			}
		}
	}

	//=========================================== End ===========================================
	fmt.Print("Press 'Enter' to continue ...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	os.Exit(1)
}
