package main

import (
	"encoding/binary"
	"fmt"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
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

	defer func() {
		if err := snapshotsBody.Close(); err != nil {
			zap.S().Fatalf("Failed to close blockchain file: %v", err)
		}
	}()
	var nBlocks uint64 = 1834290
	snapshotsChunkSize := make([]byte, 4)
	//blocksIndex := 0
	readPos := int64(0)
	//totalSize := 0
	//prevSpeed := float64(0)
	//increasingSize := true
	for height := uint64(1); height <= nBlocks; height++ {
		if _, err := snapshotsBody.ReadAt(snapshotsChunkSize, readPos); err != nil {
			log.Fatal(err)
		}
		snapshotsSize := binary.BigEndian.Uint32(snapshotsChunkSize)
		if snapshotsSize == 0 {
			readPos += 4
			continue
		}
		snapshotsSize = snapshotsSize - 4 // snapshotsSize include 4 bytes of the int size number itself, we don't need it
		//if size > MaxBlockSize || size == 0 {
		//	return errors.New("corrupted blockchain file: invalid block size")
		//}
		readPos += 4
		if snapshotsSize != 0 {
			fmt.Println()
			//blockSnapshot := proto.BlockSnapshot{}
			//snapshots := make([]byte, snapshotsSize)
			//if _, err := snapshotsBody.ReadAt(snapshots, readPos); err != nil {
			//	log.Fatal(err)
			//}
			//err := blockSnapshot.UnmarshalBinary(snapshots, proto.StageNetScheme)
			//if err != nil {
			//	return
			//}
		}

		for {
			if snapshotsSize <= 0 {
				break
			}
			snapshotChunkSize := make([]byte, 4)
			if _, err := snapshotsBody.ReadAt(snapshotChunkSize, readPos); err != nil {
				log.Fatal(err)
			}
			readPos += 4
			snapshotsSize = snapshotsSize - 4 // cut off the size chunk from the overall size number

			snapshotSize := binary.BigEndian.Uint32(snapshotChunkSize)
			snapshotSize = snapshotSize - 4 // snapshotSize include 4 bytes of the int size number itself, we don't need it

			snapshotBody := make([]byte, snapshotSize)
			if _, err := snapshotsBody.ReadAt(snapshotBody, readPos); err != nil {
				log.Fatal(err)
			}
			snapshot, err := unmarshalSnapshot(snapshotBody, proto.StageNetScheme)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(snapshot)
			readPos += int64(snapshotSize)
			snapshotsSize = snapshotsSize - snapshotSize // cut off the snapshot chunk size from the overall size number
		}
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
