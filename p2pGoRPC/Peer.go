package p2pGoRPC

import (
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
	"sync"
	"sync/atomic"
	"time"
)

type flag struct {
	peer string
	flag bool
}

type PeerInfo struct {
	toRequest       []int
	rareRequests    []int
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
	knownPeerInfo map[string]PeerInfo
	knownPeers    []string
	Tracker       string
	choked        []flag //indicate which peers don't recieve upload chunks
	me            string
	dead          int32
	ChunkSize     int
	Seeding       bool
	start         time.Time
	rarest        map[int]string //map from chunk num to peer that could send
	not_rarest    map[int]bool   //set of chunks that are not in rarest but we don't have
}

// func (P *Peer) HeartBeat(peer int) {
// 	P.mu.Lock()
// 	defer P.mu.Unlock()

// 	args := EmptyArgs{}

// 	reply := EmptyReply{}

// 	P.mu.Unlock()
// 	ok := peer.Call("Peer.ReceiveHeartbeat", &args, &reply)

// 	c := 0
// 	for !ok && P.killed() == false && c < 5 {
// 		ok = peer.Call("Peer.ReceiveHeartbeat", &args, &reply)
// 	}
// 	return
// }

func (P *Peer) AddPeerFromRPCCall(peer string) {
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

func (P *Peer) ReceiveHeartbeat(args *EmptyArgs, reply *EmptyReply) error {
	return nil
}

func (P *Peer) ChangeChokeStatus(peer string, status bool) bool {
	P.mu.Lock()

	pinfo := P.knownPeerInfo[peer]
	pinfo.am_choking = status
	P.knownPeerInfo[peer] = pinfo

	args := StatusArgs{Status: status,
		Peer: P.me}

	reply := EmptyReply{}

	P.mu.Unlock()
	client, err := rpc.DialHTTP("tcp", peer)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	call_err := client.Call("Peer.ChokeStatus", &args, &reply)
	if call_err != nil {
		log.Fatal("arith error:", call_err)
	}
	// ok := peer.Call("Peer.ChokeStatus", &args, &reply)
	// c := 0
	// for !ok && P.killed() == false && c < 5 {
	// 	ok = peer.Call("Peer.ChokeStatus", &args, &reply)
	// }
	// if c == 5 {
	// 	return false
	// }
	return true
}

func (P *Peer) ChokeStatus(args *StatusArgs, reply *EmptyReply) error {
	P.AddPeerFromRPCCall(args.Peer)

	P.mu.Lock()
	pinfo := P.knownPeerInfo[args.Peer]
	pinfo.peer_choking = args.Status
	P.knownPeerInfo[args.Peer] = pinfo

	P.mu.Unlock()
	return nil
}

func (P *Peer) ChangeInterestStatus(peer string, status bool) bool {
	P.mu.Lock()

	pinfo := P.knownPeerInfo[peer]
	pinfo.am_interested = status
	P.knownPeerInfo[peer] = pinfo

	args := StatusArgs{Status: status,
		Peer: P.me}

	reply := EmptyReply{}

	P.mu.Unlock()
	client, err := rpc.DialHTTP("tcp", peer)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	call_err := client.Call("Peer.InterestStatus", &args, &reply)
	if call_err != nil {
		log.Fatal("arith error:", call_err)
	}

	// ok := peer.Call("Peer.InterestStatus", &args, &reply)

	// c := 5
	// for !ok && P.killed() == false && c < 5 {
	// 	ok = peer.Call("Peer.InterestStatus", &args, &reply)
	// }
	// if c == 5 {
	// 	return false
	// }
	return true

}

func (P *Peer) InterestStatus(args *StatusArgs, reply *EmptyReply) error {
	P.AddPeerFromRPCCall(args.Peer)

	P.mu.Lock()
	pinfo := P.knownPeerInfo[args.Peer]
	pinfo.peer_interested = args.Status
	P.knownPeerInfo[args.Peer] = pinfo

	P.mu.Unlock()
	return nil

}

func (P *Peer) SendHaveUpdate(peer string, chunk int) bool {
	P.mu.Lock()

	args := HaveUpdateArgs{Chunk: chunk,
		Peer: P.me}

	reply := EmptyReply{}

	P.mu.Unlock()

	client, err := rpc.DialHTTP("tcp", peer)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	call_err := client.Call("Peer.HaveUpdate", &args, &reply)
	if call_err != nil {
		log.Fatal("arith error:", call_err)
	}

	// c := 5
	// for !ok && P.killed() == false && c < 5 {
	// 	ok = peer.Call("Peer.HaveUpdate", &args, &reply)
	// }
	// if c == 5 {
	// 	return false
	// }
	return true
}

func (P *Peer) HaveUpdate(args *HaveUpdateArgs, reply *EmptyReply) error {
	P.AddPeerFromRPCCall(args.Peer)
	P.mu.Lock()
	defer P.mu.Unlock()

	if !P.ChunksOwned[args.Chunk] {
		pinfo := P.knownPeerInfo[args.Peer]
		pinfo.am_interested = true
		pinfo.toRequest = append(pinfo.toRequest, args.Chunk)
		P.knownPeerInfo[args.Peer] = pinfo
		// rarity calc
		rarity := P.incrRarity(args.Chunk, args.Peer)
		if rarity == 1 {
			pinfo.rareRequests = append(pinfo.rareRequests, args.Chunk)
		}

		go P.ChangeInterestStatus(args.Peer, true)

	}
	return nil

}

func (P *Peer) GetMetaData() {
	args := MetaDataArgs{}
	reply := MetaDataReply{}

	client, err := rpc.DialHTTP("tcp", P.Tracker)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	call_err := client.Call("Peer.SendFileMetaData", &args, &reply)
	if call_err != nil {
		log.Fatal("arith error:", call_err)
	}
	// ok := P.Tracker.Call("Tracker.SendFileMetaData", &args, &reply)
	// if !ok {
	// 	ok = P.Tracker.Call("Tracker.SendFileMetaData", &args, &reply)

	// }

	P.Hashes = reply.Hashes
	P.DataOwned = make([]byte, ((len(P.Hashes) / 32) * P.ChunkSize))
}

func (P *Peer) GetChunk(peer string, chunk int) bool {

	args := SendChunkArgs{}
	args.Me = P.me
	args.Chunk = chunk
	reply := SendChunkReply{}
	client, err := rpc.DialHTTP("tcp", peer)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	call_err := client.Call("Peer.SendChunk", &args, &reply)
	if call_err != nil {
		log.Fatal("arith error:", call_err)
	}
	if reply.Valid {

		if P.CheckHash(reply.Data, chunk) {

			for i := 0; i < P.ChunkSize; i++ {
				P.DataOwned[chunk*P.ChunkSize+i] = reply.Data[i]
			}
			P.mu.Lock()
			delete(P.rarest, chunk)
			delete(P.not_rarest, chunk)
			P.mu.Unlock()
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

func (P *Peer) getRarity(chunk int) int {
	if P.ChunksOwned[chunk] {
		return -1
	}
	_, ok := P.not_rarest[chunk]
	if ok {
		return 2
	}
	_, ok = P.rarest[chunk]
	if ok {
		return 1
	}
	return 0
}

func (P *Peer) incrRarity(chunk int, peer string) int {
	rarity := P.getRarity(chunk)
	if rarity == -1 || rarity == 2 {
		return rarity
	}
	if rarity == 1 {
		delete(P.rarest, chunk)
		P.not_rarest[chunk] = true
		return 2
	}
	if rarity == 0 {
		P.rarest[chunk] = peer
		return 1
	}
	return rarity
}

func (P *Peer) SendChunk(args *SendChunkArgs, reply *SendChunkReply) error {
	P.AddPeerFromRPCCall(args.Me)
	P.mu.Lock()
	defer P.mu.Unlock()
	if !P.chokeStatus(args.Me) && P.ChunksOwned[args.Chunk] {
		reply.Data = make([]byte, P.ChunkSize)
		for i := 0; i < P.ChunkSize; i++ {
			reply.Data[i] = P.DataOwned[args.Chunk*P.ChunkSize+i]
		}

		reply.Valid = true
		return nil
	}
	reply.Valid = false
	return nil

}

func (P *Peer) CheckHash(Data []byte, chunk int) bool {
	if len(Data) != P.ChunkSize {
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
func (P *Peer) GetChunksToRequest(peer string) ([]int, []int) {
	args := SendChunksOwnedArgs{Me: P.me}
	reply := SendChunksOwnedReply{}
	client, err := rpc.DialHTTP("tcp", peer)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	call_err := client.Call("Peer.SendChunksOwned", &args, &reply)
	if call_err != nil {
		log.Fatal("arith error:", call_err)
	}
	// ok := peer.Call("Peer.SendChunksOwned", &args, &reply)
	// for !ok {
	// 	ok = peer.Call("Peer.SendChunksOwned", &args, &reply)
	// }
	ChunksToRequest := make([]int, 0)
	RareChunks := make([]int, 0)
	P.mu.Lock()
	for i := 0; i < P.NChunks; i++ {
		if reply.ChunksOwned[i] && (!P.ChunksOwned[i]) {
			ChunksToRequest = append(ChunksToRequest, i)
			rarity := P.incrRarity(i, peer)
			if rarity == 1 {
				RareChunks = append(RareChunks, i)
			}
		}
	}
	P.mu.Unlock()

	return ChunksToRequest, RareChunks

}

// RPC response that provides boolean list of file chunks owned
func (P *Peer) SendChunksOwned(args *SendChunksOwnedArgs, reply *SendChunksOwnedReply) error {
	P.AddPeerFromRPCCall(args.Me)

	reply.ChunksOwned = P.ChunksOwned
	return nil
}

// Will set choked for all peers to chokeFlag
func (P *Peer) ChokeToggle(chokeFlag bool) {
	P.mu.Lock()
	defer P.mu.Unlock()
	for i := 0; i < len(P.choked); i++ {
		P.choked[i].flag = chokeFlag
	}
}

func (P *Peer) chokeStatus(peer string) bool {
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

// returns
func (P *Peer) addRarity(chunk int, peer string) {
	if P.ChunksOwned[chunk] {
		return
	}
	_, ok := P.rarest[chunk]
	if ok {
		delete(P.rarest, chunk)
		P.not_rarest[chunk] = true
	} else {
		P.rarest[chunk] = peer
	}
}

func (P *Peer) ReloadPeers() {
	args := SendPeerArgs{P.me}
	reply := SendPeerReply{}

	client, err := rpc.DialHTTP("tcp", P.Tracker)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	call_err := client.Call("Tracker.SendPeers", &args, &reply)
	if call_err != nil {
		log.Fatal("arith error:", call_err)
	}

	// for !ok && P.killed() == false {
	// 	ok = P.Tracker.Call("Tracker.SendPeers", &args, &reply)
	// }

	P.mu.Lock()
	P.knownPeers = make([]string, 0)
	P.knownPeerInfo = make(map[string]PeerInfo)
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

	for _, p := range P.knownPeers {
		go P.ticker(p)
	}

}
func removeFromList(val int, list []int) []int {
	for i := 0; i < len(list); i++ {
		if list[i] == val {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func (P *Peer) ticker(peer string) {
	toRequest, rareChunks := P.GetChunksToRequest(peer)
	P.mu.Lock()
	pinfo := P.knownPeerInfo[peer]

	pinfo.toRequest = toRequest
	pinfo.rareRequests = rareChunks
	P.knownPeerInfo[peer] = pinfo

	P.mu.Unlock()
	for P.killed() == false {
		P.mu.Lock()
		if P.knownPeerInfo[peer].peer_interested || nrand()%10 == 0 {
			// fmt.Printf("me: %d, peer: %d\n", P.me, peer)
			P.mu.Unlock()
			P.ChangeChokeStatus(peer, false)
			// if !P.ChangeChokeStatus(peer, false) {
			// 	P.mu.Lock()
			// 	P.knownPeers = removeFromList(peer, P.knownPeers)
			// 	delete(P.knownPeerInfo, peer)
			// 	P.mu.Unlock()
			// 	return
			// }
			P.mu.Lock()
		}
		toRequest := make([]int, len(P.knownPeerInfo[peer].toRequest))
		copy(toRequest, P.knownPeerInfo[peer].toRequest)

		// fmt.Print(toRequest)
		P.mu.Unlock()
		if len(toRequest) != 0 {
			P.ChangeInterestStatus(peer, true)
			// if !P.ChangeInterestStatus(peer, true) {
			// 	P.mu.Lock()
			// 	P.knownPeers = removeFromList(peer, P.knownPeers)
			// 	delete(P.knownPeerInfo, peer)
			// 	P.mu.Unlock()
			// 	return
			// }
		}
		P.mu.Lock()
		// fmt.Print(!P.knownPeerInfo[peer].peer_choking, P.knownPeerInfo[peer].am_interested, P.me, peer, P.ChunksOwned, toRequest, "here\n")

		for len(toRequest) > 0 && !P.knownPeerInfo[peer].peer_choking && P.knownPeerInfo[peer].am_interested && P.killed() == false {
			idx := nrand() % len(toRequest)
			chunk := toRequest[idx]
			rarest := false
			if len(pinfo.rareRequests) > 0 {
				for i := len(pinfo.rareRequests) - 1; i >= 0; i-- {
					rarity := P.getRarity(i)
					if rarity >= 1 {
						pinfo.rareRequests = pinfo.rareRequests[:i]
					}
					if rarity == 1 {
						chunk = i
						rarest = true
						break
					}
				}
			}
			if !rarest {
				toRequest = append(toRequest[:idx], toRequest[idx+1:]...)
			}
			// fmt.Print(P.knownPeerInfo[peer].toRequest)

			// fmt.Print("\n")

			if !P.ChunksOwned[chunk] {
				P.mu.Unlock()
				P.GetChunk(peer, chunk)
				P.mu.Lock()
			}
		}
		P.clearDownloaded(peer)

		if P.finished() && !P.Seeding {
			P.mu.Unlock()
			fmt.Print("killing: ", P.me)
			return
		}

		P.mu.Unlock()

		time.Sleep(10 * time.Millisecond)
	}
}

func (P *Peer) finished() bool {
	for i := 0; i < len(P.ChunksOwned); i++ {
		if !P.ChunksOwned[i] {
			return false
		}
	}
	return true
}

func (P *Peer) clearDownloaded(peer string) {
	newList := make([]int, 0)
	pinfo := P.knownPeerInfo[peer]

	for _, val := range P.knownPeerInfo[peer].toRequest {
		if !P.ChunksOwned[val] {
			newList = append(newList, val)
		}
	}

	pinfo.toRequest = newList

	P.knownPeerInfo[peer] = pinfo

}

func (P *Peer) findRarestChunk(peer int, toRequest []int) int {
	allChunks := make([]int, 0)
	for _, p := range P.knownPeers {
		allChunks = append(allChunks, P.knownPeerInfo[p].toRequest...)
	}
	if len(allChunks) == 0 {
		return 0
	}
	// slices.Sort(allChunks)
	// fmt.Print(len(allChunks), "\n")

	lowest_count := 100000
	current_count := 0
	rarest_val := allChunks[0]
	for _, v1 := range allChunks {
		for _, v2 := range allChunks {
			if v2 == v1 {
				current_count++
			}
		}
		if lowest_count > current_count {
			lowest_count = current_count
			rarest_val = v1
		}
		current_count = 0

	}

	for i, val := range toRequest {
		if val == rarest_val {
			return i
		}
	}
	return nrand() % len(toRequest)
}

func MakePeer(Nchunks int, tracker string, CSize int, Seeding bool) *Peer {

	P := &Peer{}
	P.mu.Lock()
	P.Tracker = tracker
	P.knownPeerInfo = make(map[string]PeerInfo)
	P.NChunks = Nchunks
	P.ChunkSize = CSize
	P.DataOwned = make([]byte, P.NChunks*P.ChunkSize)
	P.ChunksOwned = make([]bool, Nchunks)
	P.me = CreateEndpointSelf(P)
	P.Seeding = Seeding
	P.start = time.Now()
	P.rarest = make(map[int]string, 0)
	P.not_rarest = make(map[int]bool, 0)
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

// func (P *Peer) randomUnchoke() {
// 	for P.killed() == false {

// 		if len(P.knownPeers) > 0 {
// 			P.mu.Lock()
// 			randPeer := nrand() % len(P.knownPeers)
// 			P.mu.Unlock()

//				P.ChangeChokeStatus(P.knownPeers[randPeer], false)
//			}
//			// time.Sleep(50 * time.Millisecond)
//		}
//	}

var ipAddress = "localhost"

func handler(w http.ResponseWriter, req *http.Request) {
	/*
		get its own IP address so it can send to tracker
	*/
	ipAddress = req.Header.Get("X-Real-Ip") // Store the IP address from the request header
	if ipAddress == "" {
		ipAddress = req.Header.Get("X-Forwarded-For")
	}
	if ipAddress == "" {
		ipAddress = req.RemoteAddr
	}
	// ipAddress = "localhost"

}

func CreateEndpointSelf(P *Peer) string {
	/*
		i is ID of the peer (unique)
	*/

	ownEndpoint := "18.29.72.162:8090"

	fmt.Print(ownEndpoint, "\n")
	go RegisterWithEndpoint(P, ":8090")

	return ownEndpoint
}

func RegisterWithEndpoint(P *Peer, e string) {
	rpc.Register(P)
	http.HandleFunc("/"+e, handler)
	err := http.ListenAndServe(e, nil)
	if err != nil {
		fmt.Print("network error ", err, "\n")
	}
}

func MakeSeedPeer(hashes []byte, data []byte, tracker string) *Peer {
	P := &Peer{}
	P.mu.Lock()
	P.Hashes = hashes
	P.Tracker = tracker
	P.knownPeerInfo = make(map[string]PeerInfo)
	P.NChunks = len(hashes) / 32
	P.DataOwned = data
	P.ChunkSize = len(data) / P.NChunks
	P.ChunksOwned = make([]bool, (len(hashes) / 32))
	P.me = CreateEndpointSelf(P)
	P.Seeding = true
	P.start = time.Now()
	P.rarest = make(map[int]string, 0)
	P.not_rarest = make(map[int]bool, 0)
	for i := 0; i < len(P.ChunksOwned); i++ {
		P.ChunksOwned[i] = true
	}
	P.mu.Unlock()

	go P.Starttickers()
	return P
}
