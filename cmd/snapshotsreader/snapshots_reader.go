package main

import (
	"encoding/binary"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"log"
	"os"
)

func main() {
	snapshotsBody, err := os.Open("/home/alex/Documents/snapshots-1834298")
	if err != nil {
		log.Fatalf("unable to read file: %v", err)
	}

	var nBlocks uint64 = 1000000
	snapshotsSizeBytes := make([]byte, 4)
	readPos := int64(0)
	var blocksSnapshots []proto.BlockSnapshot
	for height := uint64(1); height <= nBlocks; height++ {
		if _, err := snapshotsBody.ReadAt(snapshotsSizeBytes, readPos); err != nil {
			log.Fatal(err)
		}
		snapshotsSize := binary.BigEndian.Uint32(snapshotsSizeBytes)
		if snapshotsSize == 0 {
			readPos += 4
			continue
		}
		if snapshotsSize != 0 {
			fmt.Println()
			snapshotsInBlock := proto.BlockSnapshot{}
			snapshots := make([]byte, snapshotsSize+4) // []{snapshot, size} + 4 bytes = size of all snapshots
			if _, err := snapshotsBody.ReadAt(snapshots, readPos); err != nil {
				log.Fatal(err)
			}
			err := snapshotsInBlock.UnmarshalBinaryImport(snapshots, proto.StageNetScheme)
			if err != nil {
				return
			}
			blocksSnapshots = append(blocksSnapshots, snapshotsInBlock)
			readPos += int64(snapshotsSize) + 4

			if height == 63397 || height == 63398 {
				fmt.Println("here")
			}
		}
	}
}
