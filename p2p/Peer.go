package p2p

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"torrent/labrpc"
)

type flag struct {
	peer int
	flag bool
}

type PeerInfo struct {
	am_choking      bool // I'm choking
	am_interested   bool //
	peer_choking    bool
	peer_interested bool
}

type Peer struct {
	mu            sync.Mutex
	DataOwned     []byte
	ChunksOwned   []bool
	Hashes        []byte
	NChunks       int
	allpeers      []*labrpc.ClientEnd
	knownPeerInfo map[int]PeerInfo
	knownPeers    []int
	Tracker       *labrpc.ClientEnd
	choked        []flag //indicate which peers don't recieve upload chunks
	me            int
	dead          int32
}

func (P *Peer) HeartBeat(peer int) {
	P.mu.Lock()
	defer P.mu.Unlock()

	args := EmptyArgs{}

	reply := EmptyReply{}

	P.mu.Unlock()
	ok := P.allpeers[peer].Call("Peer.ReceiveHeartbeat", &args, &reply)

	for !ok {
		ok = P.allpeers[peer].Call("Peer.ReceiveHeartbeat", &args, &reply)
	}
}

func (P *Peer) AddPeerFromRPCCall(peer int) {
	P.mu.Lock()
	defer P.mu.Unlock()
	for _, p := range P.knownPeers {
		if p == peer {
			return
		}
	}
	P.knownPeers = append(P.knownPeers, peer)
	DPrintf("%v", P.knownPeers)
	P.knownPeerInfo[peer] = PeerInfo{
		am_choking:      true,
		am_interested:   false,
		peer_choking:    true,
		peer_interested: false,
	}

	DPrintf("NEWPEERADDED: %d, me: %d\n", peer, P.me)

	go P.ticker(peer)

}

func (P *Peer) ReceiveHeartbeat(args *EmptyArgs, reply *EmptyReply) {}

func (P *Peer) ChangeChokeStatus(peer int, status bool) {
	P.mu.Lock()

	pinfo := P.knownPeerInfo[peer]

	P.knownPeerInfo[peer] = PeerInfo{
		am_choking:      status,
		am_interested:   pinfo.am_interested,
		peer_choking:    pinfo.peer_choking,
		peer_interested: pinfo.peer_interested}

	args := StatusArgs{Status: status,
		Peer: P.me}

	reply := EmptyReply{}

	P.mu.Unlock()
	ok := P.allpeers[peer].Call("Peer.ChokeStatus", &args, &reply)

	for !ok {
		ok = P.allpeers[peer].Call("Peer.ChokeStatus", &args, &reply)
	}
}

func (P *Peer) ChokeStatus(args *StatusArgs, reply *EmptyReply) {
	P.AddPeerFromRPCCall(args.Peer)

	P.mu.Lock()
	pinfo := P.knownPeerInfo[args.Peer]
	pinfo.peer_choking = args.Status
	P.knownPeerInfo[args.Peer] = pinfo
	P.mu.Unlock()
}

func (P *Peer) ChangeInterestStatus(peer int, status bool) {
	P.mu.Lock()

	pinfo := P.knownPeerInfo[peer]

	P.knownPeerInfo[peer] = PeerInfo{
		am_choking:      pinfo.am_choking,
		am_interested:   status,
		peer_choking:    pinfo.peer_choking,
		peer_interested: pinfo.peer_interested}

	args := StatusArgs{Status: status,
		Peer: P.me}

	reply := EmptyReply{}

	P.mu.Unlock()
	ok := P.allpeers[peer].Call("Peer.InterestStatus", &args, &reply)

	for !ok {
		ok = P.allpeers[peer].Call("Peer.InterestStatus", &args, &reply)
	}

}

func (P *Peer) InterestStatus(args *StatusArgs, reply *EmptyReply) {
	P.AddPeerFromRPCCall(args.Peer)

	P.mu.Lock()
	pinfo := P.knownPeerInfo[args.Peer]
	pinfo.peer_interested = args.Status
	P.knownPeerInfo[args.Peer] = pinfo
	P.mu.Unlock()
}

func (P *Peer) SendHaveUpdate(peer int, chunk int) {
	P.mu.Lock()

	args := HaveUpdateArgs{Chunk: chunk,
		Peer: P.me}

	reply := EmptyReply{}

	P.mu.Unlock()
	ok := P.allpeers[peer].Call("Peer.HaveUpdate", &args, &reply)

	for !ok {
		ok = P.allpeers[peer].Call("Peer.HaveUpdate", &args, &reply)
	}
}

func (P *Peer) HaveUpdate(args *HaveUpdateArgs, reply *EmptyReply) {
	P.AddPeerFromRPCCall(args.Peer)
	P.mu.Lock()
	defer P.mu.Unlock()
	if !P.ChunksOwned[args.Chunk] {
		pinfo := P.knownPeerInfo[args.Peer]
		pinfo.am_interested = true
		P.knownPeerInfo[args.Peer] = pinfo
		go P.ChangeInterestStatus(args.Peer, true)
	}
}

func (P *Peer) GetMetaData() {
	args := MetaDataArgs{}
	reply := MetaDataReply{}

	ok := P.Tracker.Call("Tracker.SendFileMetaData", &args, &reply)
	if !ok {
		ok = P.Tracker.Call("Tracker.SendFileMetaData", &args, &reply)

	}

	P.Hashes = reply.Hashes
	P.DataOwned = make([]byte, ((len(P.Hashes) / 32) * 1024))
}

func (P *Peer) GetChunk(peer int, chunk int) bool {

	args := SendChunkArgs{}
	args.Chunk = chunk
	reply := SendChunkReply{}
	ok := P.allpeers[peer].Call("Peer.SendChunk", &args, &reply)
	if ok && reply.Valid {

		if P.CheckHash(reply.Data, chunk) {

			for i := 0; i < 1024; i++ {
				P.DataOwned[chunk*1024+i] = reply.Data[i]
			}
			P.ChunksOwned[chunk] = true
			// DPrintf("Peer %v got chunk %v from peer %v", P.me, chunk, peer)
			for i := 0; i < len(P.knownPeers); i++ {
				P.SendHaveUpdate(P.knownPeers[i], chunk)
			}
			return true
		}
	}
	return false
}

func (P *Peer) SendChunk(args *SendChunkArgs, reply *SendChunkReply) {
	P.AddPeerFromRPCCall(args.Me)
	P.mu.Lock()
	defer P.mu.Unlock()
	if !P.chokeStatus(args.Me) && P.ChunksOwned[args.Chunk] {
		reply.Data = make([]byte, 1024)
		for i := 0; i < 1024; i++ {
			reply.Data[i] = P.DataOwned[args.Chunk*1024+i]
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

// Requests a list of chunks that peer has, returns those we don't own
func (P *Peer) GetChunksToRequest(peer int) []int {
	args := SendChunksOwnedArgs{Me: P.me}
	reply := SendChunksOwnedReply{}

	ok := P.allpeers[peer].Call("Peer.SendChunksOwned", &args, &reply)
	if ok {
		ChunksToRequest := make([]int, 0)

		for i := 0; i < P.NChunks; i++ {
			if reply.ChunksOwned[i] && (!P.ChunksOwned[i]) {
				ChunksToRequest = append(ChunksToRequest, i)
			}
		}

		return ChunksToRequest
	}

	return make([]int, 0)
}

// RPC response that provides boolean list of file chunks owned
func (P *Peer) SendChunksOwned(args *SendChunksOwnedArgs, reply *SendChunksOwnedReply) {
	P.AddPeerFromRPCCall(args.Me)

	reply.ChunksOwned = P.ChunksOwned

}

// Will set choked for all peers to chokeFlag
func (P *Peer) ChokeToggle(chokeFlag bool) {
	P.mu.Lock()
	defer P.mu.Unlock()
	for i := 0; i < len(P.choked); i++ {
		P.choked[i].flag = chokeFlag
	}
}

func (P *Peer) chokeStatus(peer int) bool {
	for _, flag := range P.choked {
		if flag.peer == peer {
			return flag.flag
		}
	}
	P.choked = append(P.choked, flag{peer, false})
	return false
}

func (pr *Peer) Kill() {
	atomic.StoreInt32(&pr.dead, 1)
	// Your code here, if desired.
}

func (pr *Peer) killed() bool {
	z := atomic.LoadInt32(&pr.dead)
	return z == 1
}

func (P *Peer) ReloadPeers() {
	args := SendPeerArgs{P.me}
	reply := SendPeerReply{}
	ok := P.Tracker.Call("Tracker.SendPeers", &args, &reply)
	if !ok {
		fmt.Printf("sent not work")
	}

	P.mu.Lock()
	P.knownPeers = make([]int, 0)
	P.knownPeerInfo = make(map[int]PeerInfo)
	for i := 0; i < len(reply.Peers); i++ {
		P.knownPeers = append(P.knownPeers, reply.Peers[i])
		P.knownPeerInfo[reply.Peers[i]] = PeerInfo{
			am_choking:      true,
			am_interested:   false,
			peer_choking:    true,
			peer_interested: false,
		}
	}

	P.mu.Unlock()
}

func (P *Peer) Starttickers() {
	// DPrintf("in ticker\n")

	P.ReloadPeers()
	// for len(P.knownPeers) == 0 {
	// 	P.ReloadPeers()
	// }

	for p := range P.knownPeers {
		go P.ticker(p)
	}

}

func (P *Peer) ticker(peer int) {
	for P.killed() == false {
		P.mu.Lock()
		if P.knownPeerInfo[peer].peer_interested || nrand()%10 == 0 {
			// fmt.Printf("me: %d, peer: %d\n", P.me, peer)
			P.mu.Unlock()

			P.ChangeChokeStatus(peer, false)
			P.mu.Lock()
		}

		P.mu.Unlock()
		toRequest := P.GetChunksToRequest(peer)
		if len(toRequest) != 0 {
			P.ChangeInterestStatus(peer, true)
		}
		P.mu.Lock()
		// fmt.Print(!P.knownPeerInfo[peer].peer_choking, P.knownPeerInfo[peer].am_interested, P.me, peer, P.ChunksOwned, "here\n")

		for len(toRequest) > 0 && !P.knownPeerInfo[peer].peer_choking && P.knownPeerInfo[peer].am_interested {
			idx := nrand() % len(toRequest)
			chunk := toRequest[idx]
			toRequest = append(toRequest[:idx], toRequest[idx+1:]...)
			if !P.ChunksOwned[chunk] {
				P.mu.Unlock()
				P.GetChunk(peer, chunk)
				P.mu.Lock()
			}
		}
		P.mu.Unlock()

		time.Sleep(10 * time.Millisecond)
	}
}

func MakePeer(hashes []byte, tracker *labrpc.ClientEnd, me int, allPeers []*labrpc.ClientEnd) *Peer {

	P := &Peer{}
	P.mu.Lock()
	P.Hashes = hashes
	P.Tracker = tracker
	P.allpeers = allPeers
	P.knownPeerInfo = make(map[int]PeerInfo)
	P.NChunks = len(hashes) / 32
	P.DataOwned = make([]byte, P.NChunks*1024)
	P.ChunksOwned = make([]bool, (len(hashes) / 32))
	P.me = me
	P.mu.Unlock()

	go P.Starttickers()

	return P
}

// func (P *Peer) jump() {
// 	args := SendPeerArgs{P.me}
// 	reply := SendPeerReply{}
// 	ok := P.Tracker.Call("Tracker.SendPeers", &args, &reply)
// 	if !ok {
// 		DPrintf("sent not work")
// 	}
// 	for i := 0; i < len(reply.Peers); i++ {
// 		P.knownPeers = append(P.knownPeers, reply.Peers[i])
// 	}
// 	DPrintf("Peer %v recieved peers: %v", P.me, P.knownPeers)
// 	for i := 0; i < len(P.knownPeers); i++ {
// 		go P.ticker(P.knownPeers[i])
// 	}

// }

func (P *Peer) randomUnchoke() {
	for P.killed() == false {

		if len(P.knownPeers) > 0 {
			P.mu.Lock()
			randPeer := nrand() % len(P.knownPeers)
			P.mu.Unlock()

			P.ChangeChokeStatus(P.knownPeers[randPeer], false)
		}
		// time.Sleep(50 * time.Millisecond)
	}
}

func MakeSeedPeer(hashes []byte, data []byte, allPeers []*labrpc.ClientEnd, tracker *labrpc.ClientEnd) *Peer {
	P := &Peer{}
	P.mu.Lock()
	P.Hashes = hashes
	P.Tracker = tracker
	P.allpeers = allPeers
	P.knownPeerInfo = make(map[int]PeerInfo)
	P.NChunks = len(hashes) / 32
	P.DataOwned = data
	P.ChunksOwned = make([]bool, (len(hashes) / 32))
	P.me = 0
	for i := 0; i < len(P.ChunksOwned); i++ {
		P.ChunksOwned[i] = true
	}
	P.mu.Unlock()

	go P.Starttickers()
	return P
}
