package main

type CmdServer int

type EchoReq struct {
	Message string
}

type EchoResp struct {
	Message string
}

func (s *CmdServer) Echo(req *EchoReq, resp *EchoResp) error {
	resp.Message = req.Message
	return nil
}
