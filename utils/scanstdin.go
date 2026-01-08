package utils

import (
	"bufio"
	"log"
	"os"
	"strings"
)

// ScanStdIn reads lines from standard input and sends each line to the apiUrl channel.
// The channel is closed when EOF is reached or an error occurs.
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
