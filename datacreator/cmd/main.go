package main

import (
	"fmt"
	"os"

	"github.com/ideamans/go-jpeg-meta-web-strip/datacreator"
)

func main() {
	if err := datacreator.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Test data generation completed successfully!")
}
