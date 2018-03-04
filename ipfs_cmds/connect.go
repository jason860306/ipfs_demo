package ipfs_cmds

import (
	"github.com/ipfs/go-ipfs/commands"
	//ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
)

type stringList struct {
	Strings []string
}

func ConnectTo(ctx commands.Context, peerAddr string) ([]string, error) {
	args := []string{"swarm", "connect"}
	args = append(args, peerAddr)
	req, cmd, err := NewRequest(ctx, args)
	if err != nil {
		return nil, err
	}
	res := commands.NewResponse(req)
	cmd.Run(req, res)
	if res.Error() != nil {
		return nil, res.Error()
	}

	//expectedType := reflect.TypeOf(cmd.Type)
	//var respStrLst expectedType
	//for _, s := range list.Strings {
	//	respStrLst = append(respStrLst, s)
	//}
	//return respStrLst, nil
	//return res.Output().(*stringList).Strings, nil
	var strLst []string
	strLst = append(strLst, "ajwefkwjelfk")
	return strLst, nil
}
