package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/usedbytes/mini_mouse/ui/pose"
	"github.com/usedbytes/bot_matrix/datalink"
	"github.com/usedbytes/bot_matrix/datalink/rpcconn"
)

type Xact struct {
	toSend []datalink.Packet
	received []datalink.Packet
}

var angle = float64(0)

func (x *Xact) Transact(pkts []datalink.Packet) ([]datalink.Packet, error) {

	report := pose.PoseReport{
		Timestamp: time.Now(),
		Heading: angle,
	}

	angle += 0.05

	return []datalink.Packet{ *(report.Packet()) }, nil
}

func main() {
	fmt.Println("Mini Mouse Pose Sender")

	x := &Xact {
		toSend: make([]datalink.Packet, 10),
	}

	srv, err := rpcconn.NewRPCServ(x)
	if err != nil {
		panic(err)
	}

	addr := ":5556"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	log.Printf("Listening on %s...\n", addr)

	srv.Serve(l)
}
