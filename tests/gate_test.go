package tests

import (
	"github.com/davyxu/actornet/actor"
	"github.com/davyxu/actornet/gate"
	"github.com/davyxu/actornet/nexus"
	"github.com/davyxu/actornet/proto"
	"github.com/davyxu/cellnet"
	"github.com/davyxu/cellnet/socket"
	"sync"
	"testing"
	"time"
)

func startBackend() {
	actor.StartSystem()

	domain := actor.CreateDomain("backend")

	nexus.ConnectSingleton("127.0.0.1:7111", "server")

	var wg sync.WaitGroup

	wg.Add(1)

	onRouteMsg := func(c actor.Context) {

		switch msg := c.Msg().(type) {
		case *proto.TestMsgACK:

			log.Debugln("server recv", msg, c.Source())

			if msg.Msg == "hello" {
				wg.Done()

				if c.Source() != nil {
					log.Debugln("send back")

					c.Reply(msg)

					// 通知网关退出
					actor.NewPID("gate", "system").Tell(&proto.SystemExit{Code: 0})
				}

			}
		}

	}

	domain.Spawn(actor.NewTemplate().WithID("lobby").WithFunc(func(c actor.Context) {

		switch msg := c.Msg().(type) {
		case *proto.BindClientREQ:

			log.Debugln("bind", c.Source())

			pid := domain.Spawn(actor.NewTemplate().WithFunc(onRouteMsg))

			c.Reply(&proto.BindClientACK{
				ClientSessionID: msg.ClientSessionID,
				ID:              pid.Id,
			})

		}

	}))

	wg.Wait()

	time.Sleep(time.Second)
}

func startGate() {
	actor.StartSystem()

	nexus.Listen("127.0.0.1:7111")

	gate.Listen("127.0.0.1:8031", actor.NewPID("backend", "lobby"))

	actor.LoopSystem()

	time.Sleep(time.Second)
}

func startClient() {
	peer := socket.NewConnector(nil)

	peer.Start("127.0.0.1:8031")

	var wg sync.WaitGroup
	wg.Add(1)

	// 客户端连接
	cellnet.RegisterMessage(peer, "coredef.SessionConnected", func(ev *cellnet.Event) {

		// 绑定网关
		ev.Send(&proto.BindClientREQ{})
	})

	// 绑定完成, 可以发包
	cellnet.RegisterMessage(peer, "proto.BindClientACK", func(ev *cellnet.Event) {

		ev.Send(&proto.TestMsgACK{"hello"})
	})

	cellnet.RegisterMessage(peer, "proto.TestMsgACK", func(ev *cellnet.Event) {

		msg := ev.Msg.(*proto.TestMsgACK)

		if msg.Msg == "hello" {
			wg.Done()
		}

	})

	wg.Wait()
}

// 单独测试后台
func TestLinkBackend(t *testing.T) {
	startBackend()
}

// 单独测试网关
func TestLinkGate(t *testing.T) {
	startGate()
}

// 单独测试客户端
func TestLinkClient(t *testing.T) {
	startClient()
}

// 后台, 网关客户端合体在一个进程测试
func TestAllInOneGate(t *testing.T) {
	go startGate()
	go startBackend()
	startClient()
}
