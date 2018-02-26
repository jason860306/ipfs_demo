package ipfs

import (
	"io"
	"time"
	"github.com/ipfs/go-ipfs/commands"
)

func Get(ctx commands.Context, hash string, ofpath string, timeout time.Duration) ([]byte, error) {
	args := []string{"get", hash, "-o", ofpath}
	req, cmd, err := NewRequestWithTimeout(ctx, args, timeout)
	if err != nil {
		return nil, err
	}
	res := commands.NewResponse(req)
	cmd.Run(req, res)

	if res.Error() != nil {
		return nil, res.Error()
	}
	resp := res.Output()
	reader := resp.(io.Reader)
	b := make([]byte, res.Length())
	_, err = reader.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}