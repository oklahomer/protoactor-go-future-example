This repository shows usages of `actor.Future`.

# Future.Wait/Future.Result
Future.Wait() or Future.Result() wait until the response comes or the execution times out.
Since the actor blocks, incoming messages stuck in the mailbox.

![](https://raw.githubusercontent.com/oklahomer/protoactor-go-future-example/master/docs/wait/timeline.png)

Example code is located at [./wait](https://github.com/oklahomer/protoactor-go-future-example/tree/master/wait).

# Future.PipeTo
When the response comes back before Future times out, the response message is sent to the PipeTo destination.
This does not block the actor so the incoming messages are executed as they come in.
If the later message's execution is finished before the previous one, the response for the later message is received first as depicted in the below diagram.
When the execution times out, the response is sent to dead letter mailbox and the caller actor never gets noticed.

![](https://raw.githubusercontent.com/oklahomer/protoactor-go-future-example/master/docs/pipe/timeline.png)

Example code is located at [./pipe](https://github.com/oklahomer/protoactor-go-future-example/tree/master/pipe).

# Context.AwaitFuture
The message execution is done in the same way as Future.PipeTo, but a callback function is called even when the execution times out.

![](https://raw.githubusercontent.com/oklahomer/protoactor-go-future-example/master/docs/await_future/timeline.png)

Example code is located at [./await_future](https://github.com/oklahomer/protoactor-go-future-example/tree/master/await_future).

# References
- [[Golang] Protoactor-go 101: Introduction to golang's actor model implementation](https://blog.oklahome.net/2018/07/protoactor-go-introduction.html)
- [[Golang] Protoactor-go 101: How actors communicate with each other](https://blog.oklahome.net/2018/09/protoactor-go-messaging-protocol.html)
- [[Golang] protoactor-go 101: How actor.Future works to synchronize concurrent task execution](https://blog.oklahome.net/2018/11/protoactor-go-how-future-works.html)
- [[Golang] protoactor-go 201: How middleware works to intercept incoming and outgoing messages](https://blog.oklahome.net/2018/11/protoactor-go-middleware.html)
- [[Golang] protoactor-go 201: Use plugins to add behaviors to an actor](https://blog.oklahome.net/2018/12/protoactor-go-use-plugin-to-add-behavior.html)

# Other Example Codes
- [oklahomer/protoactor-go-sender-example](https://github.com/oklahomer/protoactor-go-sender-example)
  - Some example codes to illustrate how protoactor-go refers to sender process
- [/oklahomer/protoactor-go-middleware-example](https://github.com/oklahomer/protoactor-go-middleware-example)
  - Some example codes to illustrate how protoactor-go use Middleware
