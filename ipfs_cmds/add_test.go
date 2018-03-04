package ipfs_cmds

import (
	"path"
	"testing"
)

func TestAddFile(t *testing.T) {
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
}

func TestAddDirectory(t *testing.T) {
	ctx, err := MockCmdsCtx()
	if err != nil {
		t.Error(err)
	}
	root, err := AddDirectory(ctx, path.Join("./", "root"))
	if err != nil {
		t.Error(err)
	}
	if root != "zdj7WgdBhLbZ9f1Z8G3PobEHYk6ArexXBTWTjSCPv97oC4G1U" {
		t.Error("Ipfs add directory failed")
	}
}
