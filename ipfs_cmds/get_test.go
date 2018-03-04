package ipfs_cmds

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/mitchellh/go-homedir"
)

func TestGet(t *testing.T) {
	ctx, err := MockCmdsCtx()
	if err != nil {
		t.Error(err)
	}
	hash, err := AddFile(ctx, path.Join("./", "root", "test"))
	if err != nil {
		t.Error(err)
	}
	if hash != "zb2rhj7crUKTQYRGCRATFaQ6YFLTde2YzdqbbhAASkL9uRDXn" {
		t.Error("Ipfs add file failed")
	}

	//=========================================== Get ===========================================
	home, herr := homedir.Dir()
	if herr != nil {
		log.Error(herr.Error())
		os.Exit(1)
	}
	var fnamebuf bytes.Buffer
	fnamebuf.WriteString(hash)
	ofpath := filepath.Join(home, fnamebuf.String())
	dataText, err := Get(ctx, hash, ofpath, time.Second*10)
	if err != nil {
		log.Info(err.Error())
		os.Exit(1)
	} else {
		log.Infof("Get %s as follow:\n%s", hash, dataText)
	}
}
