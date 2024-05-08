package p2p

import (
	"fmt"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	fmt.Println("hello world")
	fmt.Printf("%v\n", 7/3)
}

func TestTracker(t *testing.T) {
	servers := 1
<<<<<<< HEAD
	DATA_SIZE := 1024 * 4
=======
	DATA_SIZE := 4 * 1024
>>>>>>> 28a539dda1ca8db381539d1ad79cbdd9ea464748
	data := make([]byte, DATA_SIZE)
	for i := 0; i < DATA_SIZE; i++ {
		data[i] = byte(i)
	}
	cfg := makeConfig(t, data, servers, false)
	defer cfg.cleanup()
	fmt.Print(cfg.peers[0].ChunksOwned)
	owned, matched := cfg.VerifyData(0)
	if !owned {
		t.Fatal("Tracker peer does not owned seed data")
	}
	if !matched {
		t.Fatal("Tracker did not copy data correctly")
	}

}

func TestDownloads(t *testing.T) {
	servers := 2
	DATA_SIZE := 1024 * 4
	data := make([]byte, DATA_SIZE)
	for i := 0; i < DATA_SIZE; i++ {
		data[i] = byte(i)
	}
	cfg := makeConfig(t, data, servers, false)
	defer cfg.cleanup()

	owned, matched := cfg.VerifyData(0)
	if !owned {
		t.Fatal("Tracker peer does not owned seed data")
	}
	if !matched {
		t.Fatal("Tracker did not copy data correctly")
	}
	cfg.StartPeer(1)
	// time.Sleep(100 * time.Millisecond)
	owned, matched = cfg.VerifyData(1)
	for !owned {
		time.Sleep(100 * time.Millisecond)
		owned, matched = cfg.VerifyData(1)
<<<<<<< HEAD
		//fmt.Print(cfg.peers[1].ChunksOwned)
=======
		fmt.Print(cfg.peers[2].ChunksOwned)
>>>>>>> 28a539dda1ca8db381539d1ad79cbdd9ea464748
	}

	if !owned {
		t.Fatal("Tracker peer does not owned seed data")
	}
	if !matched {
		t.Fatal("Tracker did not copy data correctly")
	}
}
