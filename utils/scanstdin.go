package utils

import (
	"bufio"
	"log"
	"os"
	"strings"
)

func ScanStdIn(urls chan<- string) {
	defer close(urls)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" {
			urls <- url
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading stdin: %v\n", err)
	}
}
