package p2p

import (
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"torrent/labrpc"
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

var TRACKERID = 0

type testConfig struct {
	mu        sync.Mutex
	net       *labrpc.Network
	t         *testing.T
	tracker   *Tracker
	peers     []*Peer
	endnames  [][]string
	endpoints [][]*labrpc.ClientEnd
	data      []byte
	hashes    []byte
	connected []bool // whether each peer is on the net
	n         int
	finished  int32

	start time.Time
	t0    time.Time
}

var ncpu_once sync.Once

func makeConfig(t *testing.T, data []byte, n int, unreliable bool) *testConfig {
	ncpu_once.Do(func() {
		if runtime.NumCPU() < 2 {
			fmt.Printf("warning: only one CPU, which may conceal locking bugs\n")
		}
		rand.Seed(makeSeed())
	})
	runtime.GOMAXPROCS(4)
	cfg := &testConfig{}
	cfg.t = t
	cfg.n = n
	cfg.net = labrpc.MakeNetwork()
	cfg.peers = make([]*Peer, cfg.n)
	cfg.endnames = make([][]string, cfg.n)
	cfg.endpoints = make([][]*labrpc.ClientEnd, cfg.n)
	cfg.start = time.Now()
	cfg.data = make([]byte, len(data))
	for i, filebyte := range data {
		cfg.data[i] = filebyte
	}
	// for i := 0; i < cfg.n; i++ {
	// 	for j := 0; j < cfg.n; j++ {
	// 		cfg.endpoints[i][j] = cfg.net.MakeEnd(cfg.endnames[i][j])
	// 		cfg.net.Enable(cfg.endnames[i][j], false)
	// 	}
	// }

	cfg.StartTracker(data)

	cfg.net.Reliable(!unreliable)

	return cfg
}

func (cfg *testConfig) FileHashes() {
	hashes := make([]byte, (len(cfg.data)/1024)*32)
	for chunk := 0; chunk < len(cfg.data); chunk++ {
		H := sha256.Sum256(cfg.data[chunk : chunk+1024])
		for i := 0; i < 32; i++ {
			hashes[chunk*32+i] = H[i]
		}
	}
	cfg.hashes = hashes
}

func (cfg *testConfig) StartTracker(Data []byte) {

	cfg.tracker = MakeTracker(Data, cfg.endpoints)

	service := labrpc.MakeService(cfg.tracker)
	srv := labrpc.MakeServer()
	srv.AddService(service)
	cfg.net.AddServer(TRACKERID, srv)

	P := MakeSeedPeer(cfg.hashes, cfg.data)
	cfg.peers[0] = P
	svcP := labrpc.MakeService(P)
	srvP := labrpc.MakeServer()
	srvP.AddService(svcP)
	cfg.net.AddServer(1, srvP)
}

func (cfg *testConfig) StartPeer(i int) *Peer {
	endname := randstring(20)
	serverEnd := cfg.net.MakeEnd(endname)
	cfg.net.Connect(endname, TRACKERID)

	P := MakePeer(cfg.hashes, serverEnd)

	cfg.peers[i] = P

	svc := labrpc.MakeService(P)
	srv := labrpc.MakeServer()
	srv.AddService(svc)
	cfg.net.AddServer(i, srv)

	cfg.connect(i)
	return P
}

// Outputs first whether data is owned completely, secondly whether it matches
func (cfg *testConfig) VerifyData(i int) (bool, bool) {
	peer := cfg.peers[i]
	// num chunks = num_bytes/1024 rounded up
	for i := 0; i < (len(cfg.data) / 1024); i++ {
		if !peer.ChunksOwned[i] {
			return false, false
		}
	}
	for i, databyte := range cfg.data {
		if peer.DataOwned[i] != databyte {
			return true, false
		}
	}
	return true, true
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

func (cfg *testConfig) cleanup() {
	atomic.StoreInt32(&cfg.finished, 1)
	for i := 0; i < len(cfg.peers); i++ {
		if cfg.peers[i] != nil {
			cfg.peers[i].Kill()
		}
	}
	cfg.net.Cleanup()
}

///////
// Timing functions
///////

func (cfg *testConfig) checkTimeout() {
	// enforce a two minute real-time limit on each test
	if !cfg.t.Failed() && time.Since(cfg.start) > 120*time.Second {
		cfg.t.Fatal("test took longer than 120 seconds")
	}
}

// start a Test.
// print the Test message.
func (cfg *testConfig) begin(description string) {
	fmt.Printf("%s ...\n", description)
	cfg.t0 = time.Now()
}

// end a Test -- the fact that we got here means there
// was no failure.
// print the Passed message,
// and some performance numbers.
func (cfg *testConfig) end() {
	cfg.checkTimeout()
	if cfg.t.Failed() == false {
		t := time.Since(cfg.t0).Seconds() // real time
		npeers := cfg.n                   // number of peers

		fmt.Printf("  ... Passed --")
		fmt.Printf("  %4.1f  %d\n", t, npeers)
	}
}
