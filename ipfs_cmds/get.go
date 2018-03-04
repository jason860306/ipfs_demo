package ipfs_cmds

import (
	//"io"
	"github.com/ipfs/go-ipfs/commands"
	"time"
)

func Get(ctx commands.Context, hash string, ofpath string, timeout time.Duration) ([]byte, error) {
	args := []string{"get", hash, "-o", ofpath}
	req, cmd, err := NewRequestWithTimeout(ctx, args, timeout)
	if err != nil {
		return nil, err
	}
	res := commands.NewResponse(req)
	cmd.PreRun(req)
	cmd.Run(req, res)
	if res.Error() != nil {
		return nil, res.Error()
	}
	//resp := res.Output()
	//reader := resp.(io.Reader)
	//b := make([]byte, res.Length())
	//_, err = reader.Read(b)
	//if err != nil {
	//	return nil, err
	//}
	cmd.PostRun(req, res)
	b := []byte{'g', 'o', 'l', 'a', 'n', 'g'}
	return b, nil
}
