package p2p

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

type Tracker struct {
	Hashes      []byte
	activePeers []int
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
	fmt.Printf("sending peers\n")
	reply.Peers = make([]int, 1)
	for i := 0; i < 1; i++ {
		reply.Peers[i] = T.activePeers[nrand()%len(T.activePeers)]
	}
}

func MakeTracker(data []byte) *Tracker {
	t := &Tracker{}
	t.Hashes = FileHashes(data)
	// t.peers = endpoints
	t.activePeers = make([]int, 0)
	t.activePeers = append(t.activePeers, 0)

	return t
}
