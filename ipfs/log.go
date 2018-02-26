package ipfs

import (
	"errors"

	"github.com/ipfs/go-ipfs/commands"
	corecmds "github.com/ipfs/go-ipfs/core/commands"
)

func Log(ctx commands.Context, subsys string, level string) (string, error) {
	args := []string{"log", "level", subsys, level}
	req, cmd, err := NewRequest(ctx, args)
	if err != nil {
		return "", err
	}
	res := commands.NewResponse(req)
	cmd.Run(req, res)

	out, ok := res.Output().(*corecmds.MessageOutput)
	if !ok {
		return "", errors.New("Get log message failed.")
	}

	return out.Message, nil
}