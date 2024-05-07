package p2p

import "torrent/labrpc"

type MetaDataArgs struct {
}

type MetaDataReply struct {
	Hashes  []byte
	NChunks int
}

type SendChunkArgs struct {
	Chunk int
}

type SendChunkReply struct {
	Data  []byte
	Valid bool
}

type SendChunksOwnedArgs struct {
}

type SendChunksOwnedReply struct {
	ChunksOwned []bool
}

type SendPeerArgs struct {
	Me int
}

type SendPeerReply struct {
	Peers []*labrpc.ClientEnd
}
