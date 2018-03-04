package ipfs_core

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	teardown()
	os.Exit(retCode)
}

func setup() {
	os.MkdirAll(path.Join("./", "root"), os.ModePerm)
	d1 := []byte("hello world")
	ioutil.WriteFile(path.Join("./", "root", "test"), d1, 0644)
}

func teardown() {
	os.RemoveAll(path.Join("./", "root"))
}
