package p2p

import (
	"crypto/rand"
	"testing"
	"time"
)

var CHUNKSIZE = 1024

func TestTracker(t *testing.T) {
	servers := 1
	DATA_SIZE := 4 * CHUNKSIZE
	data := make([]byte, DATA_SIZE)
	for i := 0; i < DATA_SIZE; i++ {
		data[i] = byte(i)
	}
	cfg := makeConfig(t, data, servers, false, CHUNKSIZE)
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
func TestSmall(t *testing.T) {
	servers := 2
	DATA_SIZE := 4 * CHUNKSIZE
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, true, CHUNKSIZE)
	defer cfg.cleanup()
	cfg.begin("Test - Small")

	cfg.VerifyDataErr(0, true)
	for i := 1; i < servers; i++ {
		time.Sleep(50 * time.Millisecond)
		cfg.StartPeer(i, true)
	}

	for !VerifyAll(servers, cfg) {
		time.Sleep(10 * time.Millisecond)
	}

	// Choke all peers from seed peer, see if Peer 2 can get from Peer 1

	//cfg.VerifyDataErr(2, true)
	cfg.end()
}
func TestSinglePeerDownload(t *testing.T) {
	servers := 2
	DATA_SIZE := 4000 * CHUNKSIZE
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, false, CHUNKSIZE)
	defer cfg.cleanup()
	cfg.begin("Test - SinglePeerDownload")

	cfg.VerifyDataErr(0, true)
	cfg.StartPeer(1, true)
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
	DATA_SIZE := 4000 * CHUNKSIZE
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, false, CHUNKSIZE)
	defer cfg.cleanup()
	cfg.begin("Test - MultiPeerDownload")

	cfg.VerifyDataErr(0, true)
	for i := 1; i < cfg.n; i++ {
		cfg.StartPeer(i, true)
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
	DATA_SIZE := 4000 * CHUNKSIZE
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, false, CHUNKSIZE)
	defer cfg.cleanup()
	cfg.begin("Test - NonSeedSingleDownload")

	cfg.VerifyDataErr(0, true)
	cfg.StartPeer(1, true)
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
	cfg.StartPeer(2, true)
	peerList := make([]int, 0)
	peerList = append(peerList, 2)
	DPrintf("peerlist %v", peerList)
	cfg.MultiVerify(peerList)
	//cfg.VerifyDataErr(2, true)
	cfg.end()
}

func TestLargeFile(t *testing.T) {
	servers := 5
	DATA_SIZE := 100 * 1000 * CHUNKSIZE
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, false, CHUNKSIZE)
	defer cfg.cleanup()
	cfg.begin("Test - LargeFile")

	cfg.VerifyDataErr(0, true)
	for i := 1; i < servers; i++ {
		cfg.StartPeer(i, true)
	}
	peerList := make([]int, 0)
	for i := 1; i < servers; i++ {
		peerList = append(peerList, i)
	}
	cfg.MultiVerify(peerList)
	// for !VerifyAll(servers, cfg) {
	// 	time.Sleep(50 * time.Millisecond)
	// }

	// Choke all peers from seed peer, see if Peer 2 can get from Peer 1

	//cfg.VerifyDataErr(2, true)
	cfg.end()
}

func TestManyPeers(t *testing.T) {
	servers := 100
	DATA_SIZE := 4 * CHUNKSIZE
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, false, CHUNKSIZE)
	defer cfg.cleanup()
	cfg.begin("Test - ManyPeers")

	cfg.VerifyDataErr(0, true)
	for i := 1; i < servers; i++ {
		cfg.StartPeer(i, true)
	}
	peerList := make([]int, 0)
	for i := 1; i < servers; i++ {
		peerList = append(peerList, i)
	}
	cfg.MultiVerify(peerList)
	// for !VerifyAll(servers, cfg) {
	// 	time.Sleep(50 * time.Millisecond)
	// }

	// Choke all peers from seed peer, see if Peer 2 can get from Peer 1

	//cfg.VerifyDataErr(2, true)
	cfg.end()
}
func TestManyPeersOverTime(t *testing.T) {
	servers := 100
	DATA_SIZE := 10 * CHUNKSIZE
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, false, CHUNKSIZE)
	defer cfg.cleanup()
	cfg.begin("Test - ManyPeersSlow")

	cfg.VerifyDataErr(0, true)
	for i := 1; i < servers; i++ {
		time.Sleep(50 * time.Millisecond)
		cfg.StartPeer(i, true)
	}

	peerList := make([]int, 0)
	for i := 1; i < servers; i++ {
		peerList = append(peerList, i)
	}
	cfg.MultiVerify(peerList)

	// Choke all peers from seed peer, see if Peer 2 can get from Peer 1

	//cfg.VerifyDataErr(2, true)
	cfg.end()
}

func TestSmallUnreliable(t *testing.T) {
	servers := 5
	DATA_SIZE := 10 * CHUNKSIZE
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, true, CHUNKSIZE)
	defer cfg.cleanup()
	cfg.begin("Test - SmallUnreliable")

	cfg.VerifyDataErr(0, true)
	for i := 1; i < servers; i++ {
		time.Sleep(50 * time.Millisecond)
		cfg.StartPeer(i, true)
	}

	peerList := make([]int, 0)
	for i := 1; i < servers; i++ {
		peerList = append(peerList, i)
	}
	cfg.MultiVerify(peerList)

	// Choke all peers from seed peer, see if Peer 2 can get from Peer 1

	//cfg.VerifyDataErr(2, true)
	cfg.end()
}

func TestManyPeersUnreliable(t *testing.T) {
	servers := 100
	DATA_SIZE := 10 * CHUNKSIZE
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, true, CHUNKSIZE)
	defer cfg.cleanup()
	cfg.begin("Test - ManyPeersUnreliable")

	cfg.VerifyDataErr(0, true)
	for i := 1; i < servers; i++ {
		time.Sleep(10 * time.Millisecond)
		cfg.StartPeer(i, true)
	}

	peerList := make([]int, 0)
	for i := 1; i < servers; i++ {
		peerList = append(peerList, i)
	}
	cfg.MultiVerify(peerList)

	// Choke all peers from seed peer, see if Peer 2 can get from Peer 1

	//cfg.VerifyDataErr(2, true)
	cfg.end()
}

func TestHalfSeedersUnreliable(t *testing.T) {
	servers := 100
	DATA_SIZE := 10 * CHUNKSIZE
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, true, CHUNKSIZE)
	defer cfg.cleanup()
	cfg.begin("Test - HalfSeedingUnreliable")

	cfg.VerifyDataErr(0, true)
	for i := 1; i < servers; i++ {
		// time.Sleep(10 * time.Millisecond)
		if nrand()%10 == 0 {
			cfg.StartPeer(i, false)
		} else {
			cfg.StartPeer(i, true)

		}
	}

	peerList := make([]int, 0)
	for i := 1; i < servers; i++ {
		peerList = append(peerList, i)
	}
	cfg.MultiVerify(peerList)

	// Choke all peers from seed peer, see if Peer 2 can get from Peer 1

	//cfg.VerifyDataErr(2, true)
	cfg.end()
}

func TestLargeFileUnreliable(t *testing.T) {
	servers := 5

	DATA_SIZE := 1000 * CHUNKSIZE
	data := make([]byte, DATA_SIZE)
	rand.Read(data)

	cfg := makeConfig(t, data, servers, true, CHUNKSIZE)
	defer cfg.cleanup()
	cfg.begin("Test - LargerFileUnreliable")

	cfg.VerifyDataErr(0, true)
	for i := 1; i < servers; i++ {
		time.Sleep(10 * time.Millisecond)
		cfg.StartPeer(i, true)
	}

	peerList := make([]int, 0)
	for i := 1; i < servers; i++ {
		peerList = append(peerList, i)
	}
	cfg.MultiVerify(peerList)

	// Choke all peers from seed peer, see if Peer 2 can get from Peer 1

	//cfg.VerifyDataErr(2, true)
	cfg.end()
}

func VerifyAll(numServers int, cfg *testConfig) bool {
	for i := 1; i < numServers; i++ {
		owned, matched := cfg.VerifyData(i)
		if !owned || !(matched) {
			return false
		}
	}
	return true
}
