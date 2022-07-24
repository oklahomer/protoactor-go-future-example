package main

import (
	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/router"
	"log"
	"os"
	"os/signal"
	"time"
)

type tick struct {
	count int
}

type pong struct {
	count int
}

type ping struct {
	count int
}

type pingActor struct {
	routerPid *actor.PID
}

func (p *pingActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *tick:
		// Output becomes somewhat like below.
		// See a diagram at https://raw.githubusercontent.com/oklahomer/protoactor-go-future-example/main/docs/reenter/timeline.png
		//
		// 2022/07/24 14:42:54 Received pong response: &main.pong{count:1}
		// 2022/07/24 14:42:57 Received pong response: &main.pong{count:4}
		// 2022/07/24 14:42:58 Failed to handle: 2. message: &main.tick{count:2}.
		// 2022/07/24 14:42:58 Received pong response: &main.pong{count:3}
		// 2022/07/24 14:42:58 DEBUG [ACTOR]       [DeadLetter] pid="Address:\"nonhost\"  Id:\"future$e\"" msg=*main.pong sender="<nil>"
		// 2022/07/24 14:43:00 Received pong response: &main.pong{count:7}
		// 2022/07/24 14:43:01 Failed to handle: 5. message: &main.tick{count:5}.
		// 2022/07/24 14:43:01 Received pong response: &main.pong{count:6}
		// 2022/07/24 14:43:01 DEBUG [ACTOR]       [DeadLetter] pid="Address:\"nonhost\"  Id:\"future$h\"" msg=*main.pong sender="<nil>"
		// 2022/07/24 14:43:03 Received pong response: &main.pong{count:10}
		// 2022/07/24 14:43:04 Failed to handle: 8. message: &main.tick{count:8}.
		// 2022/07/24 14:43:04 Received pong response: &main.pong{count:9}
		// 2022/07/24 14:43:04 DEBUG [ACTOR]       [DeadLetter] pid="Address:\"nonhost\"  Id:\"future$k\"" msg=*main.pong sender="<nil>"

		message := &ping{
			count: msg.count,
		}
		future := ctx.RequestFuture(p.routerPid, message, 2500*time.Millisecond)

		cnt := msg.count
		ctx.ReenterAfter(future, func(res interface{}, err error) {
			if err != nil {
				// Context.Message() returns the exact same message that was present on Context.ReenterAfter call.
				// ref. https://github.com/asynkron/protoactor-go/blob/afd2d973a1d15185347a69c935af9f2c72880ac6/actor/actor_context.go#L547
				log.Printf("Failed to handle: %d. message: %#v.", cnt, ctx.Message())
				return
			}

			switch res.(type) {
			case *pong:
				log.Printf("Received pong response: %#v", res)

			default:
				log.Printf("Received unexpected response: %#v", res)

			}
		})
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
	count := 0
	for {
		select {
		case <-ticker.C:
			count++
			system.Root.Send(pingPid, &tick{count: count})

		case <-finish:
			log.Print("Finish")
			return

		}
	}
}
