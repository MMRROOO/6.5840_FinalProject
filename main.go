package main

import (
	"fmt"
	"time"
	"torrent/p2pGoRPC"
)

// main functino for tracker
// func main() {
// 	data := make([]byte, 10*1024)
// 	// rand.Read(data)
// 	p2pGoRPC.MakeTracker(data)

// 	for {
// 	}

// }

// main function for peer
func main() {
	// data := make([]byte, 1000*1024)
	P := p2pGoRPC.MakePeer(10, "18.29.72.162:8080", 1024, true)

	for {
		fmt.Print(P.ChunksOwned, "\n")
		time.Sleep(500 * time.Millisecond)
	}

}
