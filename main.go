package main

import (
	"github.com/louis296/turn-server/pkg/http_server"
	"github.com/louis296/turn-server/pkg/turn_server"
)

func main() {
	turn, err := turn_server.NewTurnServer("127.0.0.1", "turn-server", 19302)
	if err != nil {
		panic(err)
	}

	httpServer := http_server.NewHttpServer(turn)

	httpServer.Bind("/api/turn", "0.0.0.0", 9000)
}
