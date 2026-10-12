// Decode a plain or DVPL resource without client discovery.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/thecoooler/blitz-client/resources"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: decode_resource <resource-file>")
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	decoded, container, err := resources.DecodeDVPL(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("container: %v, decoded bytes: %d\n", container, len(decoded))
}
