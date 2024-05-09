package p2p

import (
	"crypto/rand"
	"testing"
	"time"
)

func TestTracker(t *testing.T) {
	servers := 1
	DATA_SIZE := 4 * 1024
	data := make([]byte, DATA_SIZE)
	for i := 0; i < DATA_SIZE; i++ {
		data[i] = byte(i)
	}
	cfg := makeConfig(t, data, servers, false)
	defer cfg.cleanup()
	cfg.begin("Test - Tracker")
	for i, ownership := range cfg.peers[0].ChunksOwned {
		if !ownership {
			t.Fatalf("Tracker peer does not own chunk %v", i)
		}
	}
	owned, matched := cfg.VerifyData(0)
	if !owned {
		t.Fatal("Tracker peer does not owned seed data")
	}
	if !matched {
		t.Fatal("Tracker did not copy data correctly")
	}
	cfg.end()
}

func TestSinglePeerDownload(t *testing.T) {
	servers := 2
	DATA_SIZE := 4 * 1024
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, false)
	defer cfg.cleanup()
	cfg.begin("Test - SinglePeerDownload")

	cfg.VerifyDataErr(0, true)
	cfg.StartPeer(1)
	// time.Sleep(100 * time.Millisecond)
	owned, matched := cfg.VerifyData(1)
	for !owned {
		time.Sleep(100 * time.Millisecond)
		owned, matched = cfg.VerifyData(1)
	}
	if !matched {
		t.Fatal("Peer did not copy data correctly")
	}
	cfg.end()
}

func TestMultiPeerDownload(t *testing.T) {
	servers := 10
	DATA_SIZE := 4 * 1024
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, false)
	defer cfg.cleanup()
	cfg.begin("Test - MultiPeerDownload")

	cfg.VerifyDataErr(0, true)
	for i := 1; i < cfg.n; i++ {
		cfg.StartPeer(i)
	}
	peerList := make([]int, cfg.n-1)
	for i := 1; i < cfg.n; i++ {
		peerList[i-1] = i
	}
	cfg.MultiVerify(peerList)
	cfg.end()
}

func TestNonSeedSingleDownload(t *testing.T) {
	servers := 3
	DATA_SIZE := 4 * 1024
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, false)
	defer cfg.cleanup()
	cfg.begin("Test - NonSeedSingleDownload")

	cfg.VerifyDataErr(0, true)
	cfg.StartPeer(1)
	matched := false
	for owned := false; !owned; owned, matched = cfg.VerifyData(1) {
	}
	if !matched {
		t.Fatal("Peer did not copy data correctly")
	}
	// Choke all peers from seed peer, see if Peer 2 can get from Peer 1
	DPrintf("beginning choking test")
	seed := cfg.peers[0]
	seed.chokeStatus(2)
	seed.ChokeToggle(true)
	cfg.StartPeer(2)
	peerList := make([]int, 0)
	peerList = append(peerList, 2)
	DPrintf("peerlist %v", peerList)
	cfg.MultiVerify(peerList)
	//cfg.VerifyDataErr(2, true)
	cfg.end()
}
