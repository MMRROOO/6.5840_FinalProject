package main

import (
	"crypto/rand"
	"crypto/sha256"
	"math/big"
)

type Host struct {
	Hashes []byte
	peers  []labrpc.ClientEnd
}

func nrand() int {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return int(x)
}

func (H *Host) SendFileMetaData(args MetaDataArgs, reply MetaDataReply) {
	reply.Hashes = H.Hashes
	reply.NChunks = len(H.Hashes)
}

func FileHashes(file []byte) []byte {
	hashes := make([]byte, (len(file)/1024)*32)
	for chunk := 0; chunk < len(file); chunk++ {
		H := sha256.Sum256(file[chunk : chunk+1024])
		for i := 0; i < 32; i++ {
			hashes[chunk*32+i] = H[i]
		}
	}
	return hashes
}

func (H *Host) SendPeers(args SendPeerArgs, reply SendPeerReply) {
	for i := 0; i < 10; i++ {
		reply.Peers[i] = H.peers[nrand()%len(H.peers)]
	}
}
