package ipfs

import (
	"path"
	"testing"
	"time"
	"os"
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
	dataText, err := Get(ctx, hash, time.Second * 10)
	if err != nil {
		log.Info(err.Error())
		os.Exit(1)
	} else {
		log.Infof("Get %s as follow:\n%s", hash, dataText)
	}
}
