package p2p

import (
	"fmt"
	"testing"
)

func TestBasic(t *testing.T) {
	fmt.Println("hello world")
	fmt.Printf("%v\n", 7/3)
}

func TestTracker(t *testing.T) {
	servers := 1
	DATA_SIZE := 4000
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
}
