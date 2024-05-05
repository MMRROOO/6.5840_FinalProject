package main

import (
	"crypto/sha256"
)

type Peer struct {
	DataOwned   []byte
	ChunksOwned []bool
	Hashes      []byte
	Peers       []*labrpc.ClientEnd
	Host        *labrpc.ClientEnd
}

func (P *Peer) GetMetaData() {
	args := MetaDataArgs{}
	reply := MetaDataReply{}

	ok := P.Host.Call("Host.SendFileMetaData", &args, &reply)
	if !ok {
		ok = P.Host.Call("Host.SendFileMetaData", &args, &reply)

	}

	P.Hashes = reply.Hashes
	P.DataOwned = make([]byte, ((len(P.Hashes) / 32) * 1024))
}

func (P *Peer) GetChunk(peer int, chunk int) bool {
	args := GetChunkArgs{}
	args.Chunk = chunk
	reply := GetChunkReply{}
	ok := P.Host.Call("Host.SendChunk", &args, &reply)

	if ok && reply.Valid {
		if P.CheckHash(reply.Data, chunk) {
			for i := 0; i < 1024; i++ {
				P.DataOwned[chunk*1024+i] = reply.Data[i]
			}
			return true
		}
	}
	return false
}

func (P *Peer) SendChunk(args *GetChunkArgs, reply *GetChunkReply) {
	if P.ChunksOwned[args.Chunk] {
		for i := 0; i < 1024; i++ {
			reply.Data[i] = P.DataOwned[args.chunk*1024+i]
		}

		reply.Valid = true
		return
	}
	reply.Valid = false
}

func (P *Peer) CheckHash(Data []byte, chunk int) bool {
	if len(Data) != 1024 {
		return false
	}
	newHash := sha256.Sum256(Data)

	for i := 0; i < 32; i++ {
		if newHash[i] != P.Hashes[chunk*32+i] {
			return false
		}
	}
	return true
}
