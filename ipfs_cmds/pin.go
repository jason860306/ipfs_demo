package ipfs_cmds

import (
	"bytes"
	"fmt"

	"github.com/ipfs/go-ipfs/commands"

	corecmds "github.com/ipfs/go-ipfs/core/commands"

	"gx/ipfs/QmVmDhyTTUcQXFD1rRQ64fGLMSAoaQvNH3hwuaCFAPq2hy/errors"
)

/* pin a Object given its hash. */
func Pin(ctx commands.Context, rootHash string) error {
	args := []string{"pin", "add", "/ipfs/" + rootHash}
	req, cmd, err := NewRequest(ctx, args)
	if err != nil {
		return err
	}
	res := commands.NewResponse(req)
	cmd.Run(req, res)
	if res.Error() != nil {
		return res.Error()
	}
	return nil
}

/* Recursively pin a directory given its hash. */
func PinDir(ctx commands.Context, rootHash string) error {
	return PinDir(ctx, rootHash)
}

/* Recursively un-pin a directory given its hash.
   This will allow it to be garbage collected. */
func UnPinDir(ctx commands.Context, rootHash string) error {
	args := []string{"pin", "rm", "/ipfs/" + rootHash}
	req, cmd, err := NewRequest(ctx, args)
	if err != nil {
		return err
	}
	res := commands.NewResponse(req)
	cmd.Run(req, res)
	if res.Error() != nil {
		return res.Error()
	}
	return nil
}

func PinLs(ctx commands.Context) ([]string, error) {
	args := []string{"pin", "ls"}
	req, cmd, err := NewRequest(ctx, args)
	if err != nil {
		return nil, err
	}
	res := commands.NewResponse(req)
	cmd.Run(req, res)
	if res.Error() != nil {
		return nil, res.Error()
	}
	keys, ok := res.Output().(*corecmds.RefKeyList)
	if !ok {
		return nil, errors.Errorf("expected type %T, got an invalid Type", keys)
	}
	var objs []string
	out := new(bytes.Buffer)
	for k, v := range keys.Keys {
		fmt.Fprintf(out, "%s %s\n", k, v.Type)
		objs = append(objs, out.String())
	}
	return objs, nil
}
