package p2p

import (
	"crypto/sha256"
	"time"
)

type Peer struct {
	DataOwned   []byte
	ChunksOwned []bool
	Hashes      []byte
	Peers       []*labrpc.ClientEnd
	Tracker     *labrpc.ClientEnd
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
	ok := peer.Call("Host.SendChunk", &args, &reply)

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

func (P *Peer) GetChunksToRequest(peer int) []int {
	args := GetChunksToRequestArgs{}
	reply := GetChunksToRequestReply{}
	ok := P.Host.Call("Host.SendChunksOwned", &args, &reply)

	if ok {
		ChunksToRequest := make([]int, 0)
		for i := 0; i < len(ChunksToRequest); i++ {
			if reply.ChunksOwned[i] & (!P.ChunksOwned[i]) {
				ChunksToRequest = append(ChunksToRequest, i)
			}
		}

		return ChunksToRequest
	}
	return make([]int, 0)
}

func (P *Peer) SendChunksOwned(args *GetChunksToRequestArgs, reply *GetChunksToRequestReply) {
	reply.ChunksOwned = P.ChunksOwned

}

func (P *Peer) ticker(peer int) {
	for killed() == false {
		toRequest := P.GetChunksToRequest(peer)
		for i := 0; i < len(toRequest); i++ {
			P.GetChunk(peer, toRequest[i])
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func MakePeer(hashes []byte, tracker *labrpc.ClientEnd) *Peer {
	P := &Peer{}
	P.Hashes = hashes
	P.Tracker = tracker
	P.ChunksOwned = make([]bool, (len(hashes) / 32))

	args := SendPeerArgs{}
	reply := SendPeerReply{}
	P.Tracker.Call("Tracker.SendPeers", &args, &reply)
	P.Peers = reply.Peers
	for i := 0; i < len(P.Peers); i++ {
		go P.ticker(i)
	}

	return P
}

func MakeSeedPeer(hashes []byte, data []byte, tracker *labrpc.ClientEnd) *Peer {
	P := &Peer{}
	P.Hashes = hashes
	P.Tracker = tracker
	P.DataOwned = data
	P.ChunksOwned = make([]bool, (len(data) / 1024))
	for i := 0; i < len(P.ChunksOwned); i++ {
		P.ChunksOwned[i] = true
	}
	return P
}
