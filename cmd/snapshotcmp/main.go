package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

func main() {
	const (
		httpClientTimeout = 5 * time.Second
		diffContextLines  = 3
	)
	var (
		baseScalaURL, baseGoURL string
		targetBlockHeight       int
	)
	flag.StringVar(&baseScalaURL, "scala-url", "", "base Scala node HTTP API URL")
	flag.StringVar(&baseGoURL, "go-url", "", "base Go node HTTP API URL")
	flag.IntVar(&targetBlockHeight, "height", 0, "target block height")
	flag.Parse()
	if baseScalaURL == "" {
		fmt.Println("Scala node URL is not set") //nolint:forbidigo // it's CLI tool
		return
	}
	if baseGoURL == "" {
		fmt.Println("Go node URL is not set") //nolint:forbidigo // it's CLI tool
		return
	}
	if targetBlockHeight == 0 {
		fmt.Println("target block is not set") //nolint:forbidigo // it's CLI tool
		return
	}
	cl := &http.Client{Timeout: httpClientTimeout}
	defer cl.CloseIdleConnections()

	ids, err := getBlockIDs(cl, baseScalaURL, targetBlockHeight)
	if err != nil {
		panic(fmt.Errorf("failed to get block IDs: %w", err))
	}
	scalaSnap, err := getScalaBlockSnapshotByTxIDs(cl, baseScalaURL, ids)
	if err != nil {
		panic(fmt.Errorf("failed to get Scala snapshot: %w", err))
	}
	goSnap, err := getGoBlockSnapshot(cl, baseGoURL, targetBlockHeight)
	if err != nil {
		panic(fmt.Errorf("failed to get Go snapshot: %w", err))
	}

	if diffErr := printDiffs(os.Stdout, scalaSnap, goSnap, ids, diffContextLines); diffErr != nil {
		panic(fmt.Errorf("failed to print diffs: %w", diffErr))
	}
}

func printDiffs(w io.Writer, scalaSnap, goSnap blockSnapshotJSON, ids []string, diffContextLines int) error {
	switch { // the order of checks is important
	case len(scalaSnap) != len(ids):
		return fmt.Errorf("scala snapshots does not contain all transactions: snapshots=%d, IDs=%d", len(scalaSnap), len(ids))
	case len(scalaSnap) != len(goSnap):
		return fmt.Errorf("snapshots have different length: Scala=%d, Go=%d", len(scalaSnap), len(goSnap))
	case len(scalaSnap) == 0:
		return nil
	case diffContextLines <= 0:
		return fmt.Errorf("context depth should be positive, got %d", diffContextLines)
	}
	scalaSnap.sortFields()
	goSnap.sortFields()
	for i := range scalaSnap {
		txID := ids[i]
		txNum := i + 1
		const (
			firstName  = "Scala"
			secondName = "Go"
		)
		diff, diffErr := scalaSnap[i].diff(goSnap[i], firstName, secondName, diffContextLines)
		if diffErr != nil {
			return fmt.Errorf("failed to calculate diff for transaction#%d (%s): %w", txNum, txID, diffErr)
		}
		if diff != "" {
			if _, err := fmt.Fprintf(w, "Transaction#%d (%s) diff:\n", txNum, ids[i]); err != nil {
				return fmt.Errorf("failed to write diff heade for transaction#%d (%s): %w", txNum, txID, err)
			}
			if _, err := fmt.Fprintln(w, diff); err != nil {
				return fmt.Errorf("failed to write diff: %w", err)
			}
		} else {
			if _, err := fmt.Fprintf(w, "Transaction#%d (%s) is equal\n", txNum, ids[i]); err != nil {
				return fmt.Errorf("failed to write 'is equal' message for transaction#%d (%s): %w", txNum, txID, err)
			}
		}
		if _, err := fmt.Fprintln(w, "-------------------------------------------"); err != nil {
			return fmt.Errorf("failed to write separator: %w", err)
		}
	}
	return nil
}

func getGoBlockSnapshot(cl *http.Client, baseURL string, targetBlock int) (_ blockSnapshotJSON, err error) {
	snapshotURL, err := url.JoinPath(baseURL, "go/blocks/snapshot/at", strconv.Itoa(targetBlock))
	if err != nil {
		return nil, fmt.Errorf("failed to join URL: %w", err)
	}
	snapshot, err := cl.Get(snapshotURL) //nolint:noctx // no need in context here
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}
	defer func() {
		if bErr := snapshot.Body.Close(); bErr != nil {
			err = errors.Join(err, bErr)
		}
	}()
	var bs blockSnapshotJSON
	if jsErr := json.NewDecoder(snapshot.Body).Decode(&bs); jsErr != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", jsErr)
	}
	return bs, nil
}

func getBlockIDs(cl *http.Client, baseScalaURL string, targetBlock int) (_ []string, err error) {
	blockURL, err := url.JoinPath(baseScalaURL, "/blocks/at", strconv.Itoa(targetBlock))
	if err != nil {
		return nil, fmt.Errorf("failed to join URL: %w", err)
	}
	blockResp, err := cl.Get(blockURL) //nolint:noctx // no need in context here
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}
	defer func() {
		if bErr := blockResp.Body.Close(); bErr != nil {
			err = errors.Join(err, bErr)
		}
	}()
	var bIDs blockIDs
	if jsErr := json.NewDecoder(blockResp.Body).Decode(&bIDs); jsErr != nil {
		return nil, fmt.Errorf("failed to unmarshal block IDs: %w", jsErr)
	}
	return bIDs.IDs(), nil
}

func getScalaBlockSnapshotByTxIDs(cl *http.Client, baseScalaURL string, ids []string) (blockSnapshotJSON, error) {
	const packSize = 100
	if len(ids) == 0 {
		return nil, nil
	}
	bs := make(blockSnapshotJSON, 0, len(ids))
	packsCount, packCountRemainder := len(ids)/packSize, len(ids)%packSize
	for i := 0; i < packsCount; i++ {
		txIDsForRequest := ids[i*packSize : (i+1)*packSize]
		snapshotsPack, err := readSnapshotsPack(cl, baseScalaURL, txIDsForRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to read snapshots pack: %w", err)
		}
		bs = append(bs, snapshotsPack...)
	}
	if packCountRemainder != 0 {
		txIDsForRequest := ids[packsCount*packSize:]
		snapshotsPack, err := readSnapshotsPack(cl, baseScalaURL, txIDsForRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to read snapshots pack: %w", err)
		}
		bs = append(bs, snapshotsPack...)
	}
	return bs, nil
}

func readSnapshotsPack(cl *http.Client, baseURL string, txIDsForRequest []string) (_ []txSnapshotJSON, err error) {
	type idsJSON struct {
		IDs []string `json:"ids"`
	}
	targetURL, err := url.JoinPath(baseURL, "/transactions/snapshot")
	if err != nil {
		return nil, fmt.Errorf("failed to join URL: %w", err)
	}
	jsonData, err := json.Marshal(idsJSON{IDs: txIDsForRequest})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	resp, err := cl.Post( //nolint:noctx // no need in context here
		targetURL,
		"application/json",
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to post request: %w", err)
	}
	defer func() {
		if bErr := resp.Body.Close(); bErr != nil {
			err = errors.Join(err, bErr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var snapshots []txSnapshotJSON
	if jsErr := json.NewDecoder(resp.Body).Decode(&snapshots); jsErr != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", jsErr)
	}
	return snapshots, nil
}
