package main

import (
	"devbase/engine"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	fmt.Println("=== DevBase Scanner Performance Test ===\n")

	// Test on current project directory
	testDir := "."
	if len(os.Args) > 1 {
		testDir = os.Args[1]
	}

	fmt.Printf("Scanning directory: %s\n\n", testDir)

	startTime := time.Now()
	projects, err := engine.ScanDirectory(testDir)
	elapsed := time.Since(startTime)

	if err != nil {
		log.Fatalf("Scan failed: %v", err)
	}

	fmt.Printf("✓ Scan completed in %v\n", elapsed)
	fmt.Printf("✓ Found %d projects\n\n", len(projects))

	if len(projects) > 0 {
		fmt.Println("First 10 projects found:")
		for i, p := range projects {
			if i >= 10 {
				break
			}
			fmt.Printf("%d. %s (%s)\n", i+1, p.Name, p.Path)
		}
	}
}
