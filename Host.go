package main


type Host struct {
	SeedingPeer Peer,
	Data [1024][]byte,
	Hashes [32][]byte,

}

func (H *Host) SendFileMetaData(args MetaDataArgs, reply MetaDataReply){
	reply.Hashes = H.Hashes
	reply.NChunks = len(H.Hashes)
}


func FileHashes(file [1024][]byte) [32][]byte{
	hashes := Make([32][]byte, len(file))
	for chunk :=0; chunk < len(file); chunk ++{
		hashes[chunk] = sha256.Sum256(file[chunk])
	}
	return hashes
}
