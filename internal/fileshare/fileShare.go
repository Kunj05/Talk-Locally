package fileshare

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/ipfs/boxo/blockstore"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
)

// Global blockstore instance
var (
	store blockstore.Blockstore
	once  sync.Once
)

// Initialize the blockstore (called once)
func initBlockstore() blockstore.Blockstore {
	once.Do(func() {
		// We are using an in-memory datastore
		store = blockstore.NewBlockstore(datastore.NewMapDatastore()) // No need to wrap with sync
	})
	return store
}

// AddFileToOfflineStore adds a file to the offline blockstore and returns the CID of the stored block
func AddFileToOfflineStore(filePath string) (cid.Cid, error) {
	// Read the file data into memory
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("error reading file: %v", err)
	}

	// Initialize blockstore
	store := initBlockstore()

	// Create a new block with the file data
	blk := block.NewBlock(fileData)

	// Store the block in the blockstore
	err = store.Put(context.Background(), blk)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("error storing block: %v", err)
	}

	// Return the CID of the stored block
	return blk.Cid(), nil
}

// RetrieveFileFromStore retrieves a file from the offline blockstore by its CID
func RetrieveFileFromStore(fileCid cid.Cid) ([]byte, error) {
	// Initialize blockstore
	store := initBlockstore()

	// Retrieve the block using the provided CID
	blk, err := store.Get(context.Background(), fileCid)
	if err != nil {
		return nil, fmt.Errorf("error retrieving block: %v", err)
	}

	// Return the raw data of the block
	return blk.RawData(), nil
}
