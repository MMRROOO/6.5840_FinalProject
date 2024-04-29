# 6.5840_FinalProject

API:

PEER:
CheckHash(chuck, index)
    Checks to make sure that the hash of the file is the same as the hash at the index

    RPCS:
    SendDataOwned()
        Sends list of indexes of owned chunks.
    SendData(index)
        Sends chuck at index 

Host:
    RPCS:
    SendFileMetaData()
        Sends #of chunks in a file, as well as the hashes of each chunk to the peer.

    


