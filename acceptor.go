/**
接受者
*/

package GoPaxosImpl

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
)

type Acceptor struct {
	lis net.Listener
	// 服务器 id
	id int
	// 接受者承诺的提案编号，如果为0，则表示接受者没有收到过任何Prepare消息
	minProposal int
	// 接受者已接受的提案编号，如果为0，则表示没有接受任何提案
	acceptedNumber int
	// 接受者已接受的提案值，如果没有接受任何提案，则为nil
	acceptedValue interface{}

	// 学习者id列表
	learners []int
}

// 分别处理Paxos算法两个阶段的请求

func (a *Acceptor) Prepare(args *MsgArgs, reply *MsgReply) error {
	if args.Number > a.minProposal {
		a.minProposal = args.Number
		reply.Number = a.acceptedNumber
		reply.Value = a.acceptedValue
		reply.Ok = true
	} else {
		reply.Ok = false
	}
	return nil
}

func (a *Acceptor) Accept(args *MsgArgs, reply *MsgReply) error {
	if args.Number >= a.minProposal {
		a.minProposal = args.Number
		a.acceptedNumber = args.Number
		a.acceptedValue = args.Value
		reply.Ok = true
		// 后台转发接受的提案给学习者
		for _, lid := range a.learners {
			go func(learner int) {
				addr := fmt.Sprintf("127.0.0.1:%d", learner)
				args.From = a.id
				args.To = learner
				resp := new(MsgReply)
				ok := call(addr, "Learner.Learn", args, resp)
				if !ok {
					return
				}
			}(lid)
		}
	} else {
		reply.Ok = false
	}
	return nil
}

// 初始化函数

func newAcceptor(id int, learners []int) *Acceptor {
	acceptor := &Acceptor{
		id:       id,
		learners: learners,
	}
	acceptor.server()
	return acceptor
}

func (a *Acceptor) server() {
	rpcs := rpc.NewServer()
	rpcs.Register(a)
	addr := fmt.Sprintf(":%d", a.id)
	l, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	a.lis = l
	go func() {
		for {
			conn, err := a.lis.Accept()
			if err != nil {
				continue
			}
			go rpcs.ServeConn(conn)
		}
	}()
}

// 关闭连接
func (a *Acceptor) close() {
	a.lis.Close()
}
