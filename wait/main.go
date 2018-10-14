package main

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"log"
	"os"
	"os/signal"
	"syscall"
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

		ctx.Sender().Tell(&pong{})
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
		future := p.pongPid.RequestFuture(&ping{}, 1*time.Second)
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
	pongProps := actor.FromProducer(func() actor.Actor {
		return &pongActor{}
	})
	pongPid := actor.Spawn(pongProps)

	pingProps := actor.FromProducer(func() actor.Actor {
		return &pingActor{
			pongPid: pongPid,
		}
	})
	pingPid := actor.Spawn(pingProps)

	finish := make(chan os.Signal, 1)
	signal.Notify(finish, os.Interrupt)
	signal.Notify(finish, syscall.SIGTERM)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pingPid.Tell(struct{}{})

		case <-finish:
			return
			log.Print("Finish")

		}
	}
}
