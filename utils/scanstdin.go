package utils

import (
	"bufio"
	"log"
	"os"
)

// ScanStdIn reads lines from standard input and sends each line to the apiUrl channel.
// The channel is closed when EOF is reached or an error occurs.
func ScanStdIn(apiUrl chan string) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		apiUrl <- scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading stdin: %v\n", err)
	}

	close(apiUrl)
}
