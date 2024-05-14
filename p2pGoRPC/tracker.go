package p2pGoRPC

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"net/http"
	"net/rpc"
	"sync"
)

type Tracker struct {
	mu          sync.Mutex
	Hashes      []byte
	activePeers map[string]bool
	Endpoint    string
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

func (T *Tracker) SendPeers(args *SendPeerArgs, reply *SendPeerReply) error {
	T.mu.Lock()
	defer T.mu.Unlock()
	// DPrintf("sending peers\n")
	_, ok := T.activePeers[args.Me]
	if !ok {
		T.activePeers[args.Me] = true
	}
	NPEERS := 5
	reply.Peers = make([]string, NPEERS)
	peerList := make([]string, 0)
	for peer := range T.activePeers {
		peerList = append(peerList, peer)
	}
	reply.Peers = GetUniqueSubslicePeers(peerList, NPEERS, args.Me)
	return nil
}

func GetUniqueSubslicePeers(active []string, numpeers int, peer string) []string {
	ret := make([]string, 0)
	a := active
	for i := 0; i < len(a); i++ {
		if a[i] == peer {
			a = append(a[:i], a[i+1:]...)
			break
		}
	}

	for i := 0; i < numpeers; i++ {
		if len(ret) == len(active)-1 {
			return ret
		}
		j := nrand() % len(a)
		ret = append(ret, a[j])

		a = append(a[:j], a[j+1:]...)
	}
	return ret
}

func MakeTracker(data []byte) *Tracker {
	t := &Tracker{}
	t.Hashes = FileHashes(data)
	t.Endpoint = CreateEndpointHost(t)
	t.activePeers = make(map[string]bool, 0)
	P := MakeSeedPeer(t.Hashes, data, t.Endpoint)
	t.activePeers[P.me] = true

	return t
}

func CreateEndpointHost(T *Tracker) string {
	/*
		i is ID of the peer (unique)
	*/

	ownEndpoint := "18.29.72.162:8080"

	go RegisterWithEndpointHost(T, ":8080")

	return ownEndpoint
}

func RegisterWithEndpointHost(T *Tracker, e string) {
	rpc.Register(T)
	rpc.HandleHTTP()
	err := http.ListenAndServe(e, nil)
	if err != nil {
		fmt.Print("network error", err)
	}

}
