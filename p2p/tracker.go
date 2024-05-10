package p2p

import (
	"crypto/rand"
	"crypto/sha256"
	"math/big"
	"sync"
)

type Tracker struct {
	mu          sync.Mutex
	Hashes      []byte
	activePeers map[int]bool
}

func nrand() int {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return int(x)
}

func (T *Tracker) SendFileMetaData(args MetaDataArgs, reply MetaDataReply) {
	reply.Hashes = T.Hashes
	reply.NChunks = len(T.Hashes)
}

func FileHashes(file []byte) []byte {
	hashes := make([]byte, (len(file)/1024)*32)

	for chunk := 0; chunk < (len(file) / 1024); chunk++ {
		H := sha256.Sum256(file[chunk : chunk+1024])
		for i := 0; i < 32; i++ {
			hashes[chunk*32+i] = H[i]
		}
	}
	return hashes
}

func (T *Tracker) SendPeers(args *SendPeerArgs, reply *SendPeerReply) {
	T.mu.Lock()
	defer T.mu.Unlock()
	// DPrintf("sending peers\n")
	_, ok := T.activePeers[args.Me]
	if !ok {
		T.activePeers[args.Me] = true
	}
	NPEERS := 5
	reply.Peers = make([]int, NPEERS)
	peerList := make([]int, 0)
	for peer := range T.activePeers {
		peerList = append(peerList, peer)
	}
	reply.Peers = GetUniqueSubslicePeers(peerList, NPEERS, args.Me)
}

func GetUniqueSubslicePeers(active []int, numpeers int, peer int) []int {
	ret := make([]int, 0)
	a := active
	for i := 0; i < len(a); i++ {
		if a[i] == peer {
			a = append(a[:i], a[i+1:]...)
			break
		}
	}

	for i := 0; i < numpeers; i++ {
		if len(ret) == len(active)-1 {
			return ret
		}
		j := nrand() % len(a)
		ret = append(ret, a[j])

		a = append(a[:j], a[j+1:]...)
	}
	return ret
}

func MakeTracker(data []byte) *Tracker {
	t := &Tracker{}
	t.Hashes = FileHashes(data)
	// t.peers = endpoints
	t.activePeers = make(map[int]bool, 0)
	t.activePeers[0] = true

	return t
}
