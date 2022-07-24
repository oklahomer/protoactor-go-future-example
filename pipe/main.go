package main

import (
	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/router"
	"log"
	"os"
	"os/signal"
	"time"
)

type pong struct {
	count int
}

type ping struct {
	count int
}

type pingActor struct {
	count     int
	routerPid *actor.PID
}

func (p *pingActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case struct{}:
		p.count++
		// Output becomes somewhat like below.
		// See a diagram at https://raw.githubusercontent.com/oklahomer/protoactor-go-future-example/main/docs/pipe/timeline.png
		//
		// 2022/07/24 14:55:25 Received pong message &main.pong{count:1}
		// 2022/07/24 14:55:28 Received pong message &main.pong{count:4}
		// 2022/07/24 14:55:28 Received pong message &main.pong{count:3}
		// 2022/07/24 14:55:28 DEBUG [ACTOR]       [DeadLetter] pid="nonhost/future$e" msg=*main.pong sender="nil"
		// 2022/07/24 14:55:31 Received pong message &main.pong{count:7}
		// 2022/07/24 14:55:31 Received pong message &main.pong{count:6}
		// 2022/07/24 14:55:31 DEBUG [ACTOR]       [DeadLetter] pid="nonhost/future$h" msg=*main.pong sender="nil"
		// 2022/07/24 14:55:34 Received pong message &main.pong{count:10}
		// 2022/07/24 14:55:34 Received pong message &main.pong{count:9}
		// 2022/07/24 14:55:34 DEBUG [ACTOR]       [DeadLetter] pid="nonhost/future$k" msg=*main.pong sender="nil"
		message := &ping{
			count: p.count,
		}
		ctx.RequestFuture(p.routerPid, message, 2500*time.Millisecond).PipeTo(ctx.Self())

	case *pong:
		log.Printf("Received pong message %#v", msg)

	}
}

func main() {
	// Set up the actor system
	system := actor.NewActorSystem()

	// Set up a pool of pong actors that receive a ping payload, sleep for a certain interval, and send back a pong payload.
	// When the interval is longer than the timeout duration of the Future, a response fails with a DeadLetter.
	pongProps := router.NewRoundRobinPool(
		10,
		actor.WithFunc(func(ctx actor.Context) {
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
		}),
	)
	pongRouterPid := system.Root.Spawn(pongProps)

	// Run a ping actor that periodically sends a ping payload
	pingProps := actor.PropsFromProducer(func() actor.Actor {
		return &pingActor{
			count:     0,
			routerPid: pongRouterPid,
		}
	})
	pingPid := system.Root.Spawn(pingProps)

	// Subscribe to a signal to finish the interaction
	finish := make(chan os.Signal, 1)
	signal.Notify(finish, os.Interrupt, os.Kill)

	// Periodically send a ping payload till a signal comes
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
