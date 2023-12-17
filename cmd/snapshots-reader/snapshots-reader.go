package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	dataSnapshots, err := os.ReadFile("/home/alex/Documents/snapshots-1834298")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(dataSnapshots))
	fmt.Println(dataSnapshots[0:10000])

}
