package main

import (
	"context"
	"log"

	"github.com/femnad/sporand/cmd"
)

func main() {
	err := cmd.Generate(context.Background())
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}
