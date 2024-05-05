package p2p

import (
	crand "crypto/rand"
	"fmt"
	"math/big"
	"math/rand"
	"runtime"
	"sync"

	"6.5840_FINALPROJECT/labrpc"
)

func makeSeed() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := crand.Int(crand.Reader, max)
	x := bigx.Int64()
	return x
}

type testConfig struct {
	mu        sync.Mutex
	net       *labrpc.Network
	tracker   *Tracker
	peers     []*Peer
	Data      []byte
	connected []bool // whether each peer is on the net
}

var ncpu_once sync.Once

func makeConfig(Data []byte, unreliable bool) *testConfig {
	ncpu_once.Do(func() {
		if runtime.NumCPU() < 2 {
			fmt.Printf("warning: only one CPU, which may conceal locking bugs\n")
		}
		rand.Seed(makeSeed())
	})
	runtime.GOMAXPROCS(4)
	cfg := &testConfig{}
	cfg.net = labrpc.MakeNetwork()
	cfg.peers = make([]*Peer, 0)
	cfg.StartServer(Data)

	cfg.net.Reliable(!unreliable)

	return cfg
}

func (cfg *testConfig) StartServer(Data []byte) {

	cfg.tracker = StartTracker(Data)

	kvsvc := labrpc.MakeService(cfg.tracker)
	srv := labrpc.MakeServer()
	srv.AddService(kvsvc)
	cfg.net.AddServer(0, srv)
}

// func simpleTest() {

// }
