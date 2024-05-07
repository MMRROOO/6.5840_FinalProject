package p2p

import (
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"math/rand"
	"runtime"
	"sync"

	"6.5840/6.5840_FinalProject/labrpc"
)

func makeSeed() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := crand.Int(crand.Reader, max)
	x := bigx.Int64()
	return x
}

func randstring(n int) string {
	b := make([]byte, 2*n)
	crand.Read(b)
	s := base64.URLEncoding.EncodeToString(b)
	return s[0:n]
}

var SERVERID = 0

type testConfig struct {
	mu        sync.Mutex
	net       *labrpc.Network
	tracker   *Tracker
	peers     []*Peer
	endnames  [][]string
	Data      []byte
	hashes    []byte
	connected []bool // whether each peer is on the net
	n         int
}

var ncpu_once sync.Once

func makeConfig(data []byte, n int, unreliable bool) *testConfig {
	ncpu_once.Do(func() {
		if runtime.NumCPU() < 2 {
			fmt.Printf("warning: only one CPU, which may conceal locking bugs\n")
		}
		rand.Seed(makeSeed())
	})
	runtime.GOMAXPROCS(4)
	cfg := &testConfig{}
	cfg.n = n
	cfg.net = labrpc.MakeNetwork()
	cfg.peers = make([]*Peer, cfg.n)
	cfg.endnames = make([][]string, cfg.n)

	cfg.StartServer(data)

	cfg.net.Reliable(!unreliable)

	return cfg
}

func (cfg *testConfig) StartServer(Data []byte) {

	cfg.tracker = StartTracker(Data, cfg.endnames)

	service := labrpc.MakeService(cfg.tracker)
	srv := labrpc.MakeServer()
	srv.AddService(service)
	cfg.net.AddServer(0, srv)
}

func (cfg *testConfig) StartPeer() *Peer {
	endname := randstring(20)
	end := cfg.net.MakeEnd(endname)
	cfg.net.Connect(endname, SERVERID)

	P := MakePeer(cfg.hashes, end)
	return P
}

func (cfg *testConfig) disconnect(i int) {
	// fmt.Printf("disconnect(%d)\n", i)

	cfg.connected[i] = false

	// outgoing ClientEnds
	for j := 0; j < cfg.n; j++ {
		if cfg.endnames[i] != nil {
			endname := cfg.endnames[i][j]
			cfg.net.Enable(endname, false)
		}
	}

	// incoming ClientEnds
	for j := 0; j < cfg.n; j++ {
		if cfg.endnames[j] != nil {
			endname := cfg.endnames[j][i]
			cfg.net.Enable(endname, false)
		}
	}
}

func (cfg *testConfig) connect(i int) {
	// fmt.Printf("connect(%d)\n", i)

	cfg.connected[i] = true

	// outgoing ClientEnds
	for j := 0; j < cfg.n; j++ {
		if cfg.connected[j] {
			endname := cfg.endnames[i][j]
			cfg.net.Enable(endname, true)
		}
	}

	// incoming ClientEnds
	for j := 0; j < cfg.n; j++ {
		if cfg.connected[j] {
			endname := cfg.endnames[j][i]
			cfg.net.Enable(endname, true)
		}
	}
}

// func simpleTest() {

// }
