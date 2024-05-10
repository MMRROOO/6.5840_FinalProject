package p2p

import "fmt"

const Debug bool = true

func DPrintf(format string, a ...interface{}) {
	if Debug {
		fmt.Printf(format, a...)
	}
}

type StatusArgs struct {
	Status bool
	Peer   int
}
type HaveUpdateArgs struct {
	Chunk int
	Peer  int
}

type EmptyReply struct{}

type EmptyArgs struct{}

type MetaDataArgs struct {
}

type MetaDataReply struct {
	Hashes  []byte
	NChunks int
}

type SendChunkArgs struct {
	Me    int
	Chunk int
}

type SendChunkReply struct {
	Data  []byte
	Valid bool
}

type SendChunksOwnedArgs struct {
	Me int
}

type SendChunksOwnedReply struct {
	ChunksOwned []bool
}

type SendPeerArgs struct {
	Me int
}

type SendPeerReply struct {
	Peers []int
}
