package main

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/router"
	"log"
	"os"
	"os/signal"
	"time"
)

type pong struct {
	count uint
}

type ping struct {
	count uint
}

type pingActor struct {
	count     uint
	routerPid *actor.PID
}

func (p *pingActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case struct{}:
		p.count++
		// Output becomes somewhat like below.
		// See a diagram at https://raw.githubusercontent.com/oklahomer/protoactor-go-future-example/master/docs/pipe/timeline.png

		// 2018/10/14 14:20:36 Received pong message &main.pong{count:1}
		// 2018/10/14 14:20:39 Received pong message &main.pong{count:4}
		// 2018/10/14 14:20:39 Received pong message &main.pong{count:3}
		// 2018/10/14 05:20:39 [ACTOR] [DeadLetter] pid="nonhost/future$e" message=&{'\x02'} sender="nil"
		// 2018/10/14 14:20:42 Received pong message &main.pong{count:7}
		// 2018/10/14 14:20:42 Received pong message &main.pong{count:6}
		// 2018/10/14 05:20:42 [ACTOR] [DeadLetter] pid="nonhost/future$h" message=&{'\x05'} sender="nil"
		// 2018/10/14 14:20:45 Received pong message &main.pong{count:10}
		// 2018/10/14 14:20:45 Received pong message &main.pong{count:9}
		// 2018/10/14 05:20:45 [ACTOR] [DeadLetter] pid="nonhost/future$k" message=&{'\b'} sender="nil"
		message := &ping{
			count: p.count,
		}
		ctx.RequestFuture(p.routerPid, message, 2500*time.Millisecond).PipeTo(ctx.Self())

	case *pong:
		log.Printf("Received pong message %#v", msg)

	}
}

func main() {
	// Setup actor system
	system := actor.NewActorSystem()

	// Setup a pool of pong actors that receive ping payload, sleep for a certain interval, and send back pong payload.
	// When the interval is longer than the timeout duration of Future, response fails with a DeadLetter.
	pongProps := router.NewRoundRobinPool(10).
		WithFunc(func(ctx actor.Context) {
			switch msg := ctx.Message().(type) {
			case *ping:
				var sleep time.Duration
				remainder := msg.count % 3
				if remainder == 0 {
					sleep = 1700 * time.Millisecond
				} else if remainder == 1 {
					sleep = 300 * time.Millisecond
				} else {
					sleep = 2900 * time.Millisecond
				}
				time.Sleep(sleep)

				message := &pong{
					count: msg.count,
				}
				ctx.Respond(message)
			}
		})
	pongRouterPid := system.Root.Spawn(pongProps)

	// Run a ping actor that periodically sends ping payload
	pingProps := actor.PropsFromProducer(func() actor.Actor {
		return &pingActor{
			count:     0,
			routerPid: pongRouterPid,
		}
	})
	pingPid := system.Root.Spawn(pingProps)

	// Subscribe to signal to finish interaction
	finish := make(chan os.Signal, 1)
	signal.Notify(finish, os.Interrupt, os.Kill)

	// Periodically send ping payload till signal comes
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			system.Root.Send(pingPid, struct{}{})

		case <-finish:
			log.Print("Finish")
			return

		}
	}
}
