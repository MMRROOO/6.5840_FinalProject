package p2pGoRPC

import "fmt"

const Debug bool = false

func DPrintf(format string, a ...interface{}) {
	if Debug {
		fmt.Printf(format, a...)
	}
}

type StatusArgs struct {
	Status bool
	Peer   string
}
type HaveUpdateArgs struct {
	Chunk int
	Peer  string
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
	Me    string
	Chunk int
}

type SendChunkReply struct {
	Data  []byte
	Valid bool
}

type SendChunksOwnedArgs struct {
	Me string
}

type SendChunksOwnedReply struct {
	ChunksOwned []bool
}

type SendPeerArgs struct {
	Me string
}

type SendPeerReply struct {
	Peers []string
}
