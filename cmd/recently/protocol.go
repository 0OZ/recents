package main

import "fmt"

// Command is a request sent from a client to the daemon over the unix socket.
type Command string

const cmdClear Command = "!clear"

// Response is the single-line reply the daemon writes back to a client.
type Response string

const respCleared Response = "ok cleared"

func respRecorded(p ProjectPath) Response {
	return Response(fmt.Sprintf("ok %s", p))
}
