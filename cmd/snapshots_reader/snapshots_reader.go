package main

import (
	"encoding/binary"
	"fmt"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"log"
	"os"
)

func unmarshalSnapshot(snapshotsBytes []byte, scheme proto.Scheme) (proto.BlockSnapshot, error) {
	var tsProto g.TransactionStateSnapshot
	err := tsProto.UnmarshalVT(snapshotsBytes)
	if err != nil {
		return proto.BlockSnapshot{}, err
	}
	atomicTS, err := proto.TxSnapshotsFromProtobuf(scheme, &tsProto)
	if err != nil {
		return proto.BlockSnapshot{}, err
	}
	fmt.Println(atomicTS)
	return proto.BlockSnapshot{}, nil
}

func main() {
	snapshotsBody, err := os.Open("/home/alex/Documents/snapshots-1834298")
	if err != nil {
		log.Fatalf("unable to read file: %v", err)
	}

	//defer func() {
	//	if err := snapshotsBody.Close(); err != nil {
	//		zap.S().Fatalf("Failed to close blockchain file: %v", err)
	//	}
	//}()
	//blockSnapshot := proto.BlockSnapshot{}
	//err = blockSnapshot.UnmarshalBinary(snapshotsBody, proto.StageNetScheme)
	//if err != nil {
	//	return

	var nBlocks uint64 = 1834290
	snapshotsSizeBytes := make([]byte, 4)
	//blocksIndex := 0
	readPos := int64(0)
	//totalSize := 0
	//prevSpeed := float64(0)
	//increasingSize := true
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
		}

		fmt.Println(len(blocksSnapshots))
		//for {
		//	if snapshotsSize <= 0 {
		//		break
		//	}
		//	snapshotChunkSize := make([]byte, 4)
		//	if _, err := snapshotsBody.ReadAt(snapshotChunkSize, readPos); err != nil {
		//		log.Fatal(err)
		//	}
		//	readPos += 4
		//	snapshotsSize = snapshotsSize - 4 // cut off the size chunk from the overall size number
		//
		//	snapshotSize := binary.BigEndian.Uint32(snapshotChunkSize)
		//	snapshotSize = snapshotSize - 4 // snapshotSize include 4 bytes of the int size number itself, we don't need it
		//
		//	snapshotBody := make([]byte, snapshotSize)
		//	if _, err := snapshotsBody.ReadAt(snapshotBody, readPos); err != nil {
		//		log.Fatal(err)
		//	}
		//	snapshot, err := unmarshalSnapshot(snapshotBody, proto.StageNetScheme)
		//	if err != nil {
		//		log.Fatal(err)
		//	}
		//	fmt.Println(snapshot)
		//	readPos += int64(snapshotSize)
		//	snapshotsSize = snapshotsSize - snapshotSize // cut off the snapshot chunk size from the overall size number
		//}
		////if height < startHeight {
		////	readPos += int64(size)
		////	continue
		////}
		//snapshots := make([]byte, size)
		//if _, err := snapshotsBody.ReadAt(block, readPos); err != nil {
		//	log.Fatal(err)
		//}
		//readPos += int64(size)
		//blocks[blocksIndex] = block
		//blocksIndex++
		//start := time.Now()
		//if err := st.AddBlocks(blocks[:blocksIndex]); err != nil {
		//	return err
		//}
		//elapsed := time.Since(start)
		//speed := float64(totalSize) / float64(elapsed)
		//maxSize, increasingSize = calculateNextMaxSizeAndDirection(maxSize, speed, prevSpeed, increasingSize)
		//prevSpeed = speed
		//totalSize = 0
		//blocksIndex = 0
		//if err := maybePersistTxs(st); err != nil {
		//	return err
		//}
	}
}
