package main

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"log"
	"os"
	"os/signal"
	"time"
)

type pong struct {
}

type ping struct {
}

type pongActor struct {
	timeOut bool
}

func (p *pongActor) Receive(ctx actor.Context) {
	// Dead letter occurs because the PID of Future process ends and goes away when Future times out
	// so the pongActor fails to send message.
	switch ctx.Message().(type) {
	case *ping:
		var sleep time.Duration
		if p.timeOut {
			sleep = 2500 * time.Millisecond
			p.timeOut = false
		} else {
			sleep = 300 * time.Millisecond
			p.timeOut = true
		}
		time.Sleep(sleep)

		ctx.Respond(&pong{})
	}
}

type pingActor struct {
	pongPid *actor.PID
}

func (p *pingActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case struct{}:
		// Output becomes somewhat like below.
		// See a diagram at https://raw.githubusercontent.com/oklahomer/protoactor-go-future-example/master/docs/wait/timeline.png
		//
		// 2018/10/13 17:03:22 Received pong message &main.pong{}
		// 2018/10/13 17:03:24 Timed out
		// 2018/10/13 08:03:26 [ACTOR] [DeadLetter] pid="nonhost/future$4" message=&{} sender="nil"
		// 2018/10/13 17:03:26 Received pong message &main.pong{}
		// 2018/10/13 17:03:28 Timed out
		// 2018/10/13 08:03:30 [ACTOR] [DeadLetter] pid="nonhost/future$6" message=&{} sender="nil"
		// 2018/10/13 17:03:30 Received pong message &main.pong{}
		future := ctx.RequestFuture(p.pongPid, &ping{}, 1*time.Second)
		// Future.Result internally waits until response comes or times out
		result, err := future.Result()
		if err != nil {
			log.Print("Timed out")
			return
		}

		log.Printf("Received pong message %#v", result)

	}
}

func main() {
	// Setup actor system
	system := actor.NewActorSystem()

	// Setup a pong actor that receives ping payload, sleeps for a certain interval, and sends back pong payload.
	// When the interval is longer than the timeout duration of Future, response fails with a DeadLetter.
	pongProps := actor.PropsFromProducer(func() actor.Actor {
		return &pongActor{}
	})
	pongPid := system.Root.Spawn(pongProps)

	// Run a ping actor that periodically sends ping payload
	pingProps := actor.PropsFromProducer(func() actor.Actor {
		return &pingActor{
			pongPid: pongPid,
		}
	})
	pingPid := system.Root.Spawn(pingProps)

	// Subscribe to signal to finish interaction
	finish := make(chan os.Signal, 1)
	signal.Notify(finish, os.Interrupt, os.Kill)

	// Periodically send ping payload till signal comes
	ticker := time.NewTicker(2 * time.Second)
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
