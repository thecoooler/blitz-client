// Read a client catalog without compiling the exporter or installer.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/thecoooler/blitz-client/client"
)

func main() {
	directory := ""
	if len(os.Args) > 1 {
		directory = os.Args[1]
	}
	resolved, err := client.ResolveGameDir(directory)
	if err != nil {
		log.Fatal(err)
	}
	catalog, err := client.DiscoverClientCatalog(resolved.ResourceDir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("maps: %d, tanks: %d, model pairs: %d\n",
		len(catalog.Maps), len(catalog.Tanks), len(catalog.ModelTanks))
}
