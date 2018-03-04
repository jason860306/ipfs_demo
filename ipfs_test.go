package ipfs_core

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/op/go-logging"

	"github.com/jason860306/ipfs_demo/ipfs_cmds"
)

var log_test = logging.MustGetLogger("test")

func TestIpfs(t *testing.T) {
	args := flag.Args()
	os.Args = append([]string{os.Args[0]}, args...)
	if len(os.Args) != 2 {
		fmt.Printf("usage: %s [cli|srv]", os.Args[0])
		os.Exit(1)
	}
	//=========================================== Init ===========================================
	// Set repo path
	repoPath, err := Init()
	if err != nil {
		os.Exit(1)
	}
	fmt.Printf("ipfs_demo repo initialized at %s\n", repoPath)

	//=========================================== Start ===========================================
	err = Start(repoPath)
	if err != nil {
		log_test.Infof("%s\n", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Daemon is ready\n")

	//=========================================== Set ipfs log level ===========================================
	logmsg, logerr := ipfs_cmds.Log(Node.Context, "all", "debug")
	if logerr != nil {
		log_test.Error(logerr.Error())
		os.Exit(1)
	}
	log_test.Info(logmsg)

	if os.Args[1] == "srv" {
		//=========================================== Add ===========================================
		//README.md on linux:   zb2rhneqJaf4y9vQpb9o1yqyejARwiR9PDuz8bXjRTAE5iLT9
		//README.md on windows: zb2rhjwNxFKtD3Qg4nV3Qf4CH77bvEn7ndzM4ysXCwxvpLXeo
		hash, err := ipfs_cmds.AddFile(Node.Context, filepath.Join("./", "README.md"))
		if err != nil {
			log_test.Info(err.Error())
			os.Exit(1)
		}
		if runtime.GOOS == "windows" && hash != "zb2rhjwNxFKtD3Qg4nV3Qf4CH77bvEn7ndzM4ysXCwxvpLXeo" {
			log_test.Info("Ipfs add file on windows failed")
		} else if hash != "zb2rhneqJaf4y9vQpb9o1yqyejARwiR9PDuz8bXjRTAE5iLT9" {
			log_test.Info("Ipfs add file on uni* failed")
		} else {
			log_test.Info("Ipfs add file successfully: ", hash)
		}
		//test.bin: zdj7WdnQBd3Yf4KPuUTZ9mkAQ6Rfd87H4h2f7d3KxzgW4kJ9U
		hash, err = ipfs_cmds.AddFile(Node.Context, filepath.Join("./resource", "test.bin"))
		if err != nil {
			log_test.Info(err.Error())
			os.Exit(1)
		}
		if hash != "zb2rhjwNxFKtD3Qg4nV3Qf4CH77bvEn7ndzM4ysXCwxvpLXeo" {
			log_test.Info("Ipfs add file failed")
		} else {
			log_test.Info("Ipfs add file successfully: ", hash)
		}
	} else if os.Args[1] == "cli" {
		//// =========================================== Connect peers ===========================================
		////peer := "/ip4/192.168.222.180/tcp/4001/ipfs/QmXZf95S6CDpzyeyFGr8SU2b3hm68FrxUyRpTgTR5YxZ56"
		////peer := "/ip4/138.197.232.22/tcp/4001/ipfs/QmZjmQH4e7opwmeFc23vUZ4nwuPw1oJgFKSpJoAJgrpQiy"
		//peer := "/ip4/97.64.43.18/tcp/4001/ipfs/QmPUrqtsYZzpebQ4sYHiQqjtTGCEGUVu194jhHuVpBnGb3"
		//for i := 0; i < 5; i++ {
		//	peers, err := ipfs.ConnectTo(Node.Context, peer)
		//	if err != nil {
		//		log_test.Info(err.Error())
		//		//os.Exit(1)
		//		continue
		//	}
		//	log_test.Infof("connect %s successfully\n", peer)
		//	for i, peer := range peers {
		//		log_test.Infof("#%d: %s\n", i, peer)
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
		dataText, err := ipfs_cmds.Cat(Node.Context, file_hash, time.Second*120)
		if err != nil {
			log_test.Info(err.Error())
			<-time.After(1 * time.Second)
			continue
		} else {
			log_test.Infof("Cat %s as follow:\n%s", file_hash, dataText)
			break
		}
	}

	//=========================================== Swarm peers ===========================================
	go func() {
		for {
			pbool := make(chan []string)
			go func() {
				peers, err := ipfs_cmds.ConnectedPeers(Node.Context)
				if err != nil {
					errInfo := make([]string, 1)
					errInfo = append(errInfo, err.Error())
					pbool <- errInfo
					log_test.Info(err.Error())
				}
				pbool <- peers
			}()
			peers := <-pbool
			if len(peers) == 0 {
				log_test.Infof("No peers in swarm")
			} else {
				for i, peer := range peers {
					log_test.Infof("peer #%d: %s\n", i, peer)
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
			err := ipfs_cmds.Pin(Node.Context, hash)
			if err != nil {
				log_test.Info(err.Error())
				<-time.After(1 * time.Second)
			} else {
				log_test.Infof("Pin %s Ok!", hash)
				break
			}
		}
	}
	// pin ls
	objs1, err := ipfs_cmds.PinLs(Node.Context)
	if err != nil {
		log_test.Info(err.Error())
	} else {
		for i, obj := range objs1 {
			log_test.Infof("obj #%d: %s\n", i, obj)
		}
	}
	// unpin
	for _, hash := range fhash {
		for j := 0; j < 3; j++ {
			err := ipfs_cmds.UnPinDir(Node.Context, hash)
			if err != nil {
				log_test.Info(err.Error())
				<-time.After(1 * time.Second)
			} else {
				log_test.Infof("UnPin %s Ok!", hash)
				break
			}
		}
	}
	// pin ls
	objs2, err := ipfs_cmds.PinLs(Node.Context)
	if err != nil {
		log_test.Info(err.Error())
	} else {
		for i, obj := range objs2 {
			log_test.Infof("obj #%d: %s\n", i, obj)
		}
	}

	//=========================================== Get ===========================================
	for _, hash := range fhash {
		for j := 0; j < 3; j++ {
			home, herr := homedir.Dir()
			if herr != nil {
				log_test.Error(herr.Error())
				os.Exit(1)
			}
			var fnamebuf bytes.Buffer
			fnamebuf.WriteString(hash)
			fnamebuf.WriteString("_")
			fnamebuf.WriteString(strconv.Itoa(j))
			ofpath := filepath.Join(home, fnamebuf.String())
			d, err := ipfs_cmds.Get(Node.Context, hash, ofpath, time.Second*120)
			if err != nil {
				log_test.Info(err.Error())
				<-time.After(1 * time.Second)
			} else {
				log_test.Infof("Get %s Ok!", d)
				break
			}
		}
	}

	//=========================================== End ===========================================
	fmt.Print("Press 'Enter' to continue ...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	os.Exit(1)
}
