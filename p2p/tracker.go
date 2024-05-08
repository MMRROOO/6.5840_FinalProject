package p2p

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"

	"torrent/labrpc"
)

type Tracker struct {
	Hashes      []byte
	peers       [][]*labrpc.ClientEnd
	activePeers []bool
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

	for i := 0; i < 10; i++ {
		reply.Peers[i] = T.peers[args.Me][nrand()%len(T.peers)]
	}
}

func MakeTracker(data []byte, endpoints [][]*labrpc.ClientEnd) *Tracker {
	t := &Tracker{}
	t.Hashes = FileHashes(data)
	t.peers = endpoints
	t.activePeers = make([]bool, len(endpoints))
	t.activePeers[0] = true

	return t
}
