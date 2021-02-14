package main

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/router"
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
		// See a diagram at https://raw.githubusercontent.com/oklahomer/protoactor-go-future-example/master/docs/await_future/timeline.png
		//
		// 2018/10/14 16:10:49 Received pong response: &main.pong{count:1}
		// 2018/10/14 16:10:52 Received pong response: &main.pong{count:4}
		// 2018/10/14 16:10:52 Failed to handle: 2. message: &main.tick{count:2}.
		// 2018/10/14 16:10:53 Received pong response: &main.pong{count:3}
		// 2018/10/14 07:10:53 [ACTOR] [DeadLetter] pid="nonhost/future$e" message=&{'\x02'} sender="nil"
		// 2018/10/14 16:10:55 Received pong response: &main.pong{count:7}
		// 2018/10/14 16:10:55 Failed to handle: 5. message: &main.tick{count:5}.
		// 2018/10/14 16:10:56 Received pong response: &main.pong{count:6}
		// 2018/10/14 07:10:56 [ACTOR] [DeadLetter] pid="nonhost/future$h" message=&{'\x05'} sender="nil"
		// 2018/10/14 16:10:58 Received pong response: &main.pong{count:10}
		// 2018/10/14 16:10:58 Failed to handle: 8. message: &main.tick{count:8}.
		// 2018/10/14 16:10:59 Received pong response: &main.pong{count:9}
		// 2018/10/14 07:10:59 [ACTOR] [DeadLetter] pid="nonhost/future$k" message=&{'\b'} sender="nil"

		message := &ping{
			count: msg.count,
		}
		future := ctx.RequestFuture(p.routerPid, message, 2500*time.Millisecond)

		cnt := msg.count
		ctx.AwaitFuture(future, func(res interface{}, err error) {
			if err != nil {
				// Context.Message() returns the exact message that was present on Context.AwaitFuture call.
				// ref. https://github.com/AsynkronIT/protoactor-go/blob/3992780c0af683deb5ec3746f4ec5845139c6e42/actor/local_context.go#L289
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
