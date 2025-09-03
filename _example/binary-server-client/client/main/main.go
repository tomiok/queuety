package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/tomiok/queuety/manager"
)

func main() {
	conn, err := manager.Connect("tcp4", ":9845", nil)
	if err != nil {
		panic(err)
	}

	// set connection to use binary format
	conn.SetDefaultFormat(manager.FormatBinary)

	topic, err := conn.NewTopic("text-files")
	if err != nil {
		panic(err)
	}

	go func() {
		for fileContent := range manager.Consume(conn, topic) {
			fmt.Printf("received text file (%d bytes):\n %s\n", len(fileContent), fileContent)
		}
	}()

	time.Sleep(1 * time.Second)

	// find all file1.txt, file2.txt, etc.
	files := findTextFiles()
	if len(files) == 0 {
		log.Println("no files found. Create file1.txt, file2.txt, etc. manually")
		return
	}

	fmt.Printf("Found files: %v\n", files)

	// Send text files every 3 seconds
	fileIndex := 0
	for {
		fileName := files[fileIndex%len(files)]

		// Read text file as byte slice
		content, err := os.ReadFile(fileName)
		if err != nil {
			log.Printf("error reading file %s: %v\n", fileName, err)
			fileIndex++
			continue
		}

		// Send file content as byte slice
		err = conn.PublishBinary(topic, content)
		if err != nil {
			panic(err)
		}

		fileIndex++
		time.Sleep(1 * time.Second)
	}
}

func findTextFiles() []string {
	var files []string

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return files
	}

	// Look for file1.txt, file2.txt, file3.txt, etc. in current directory
	for i := 1; i <= 10; i++ {
		filename := fmt.Sprintf("file%d.txt", i)
		fullPath := filepath.Join(currentDir, filename)

		if _, err := os.Stat(fullPath); err == nil {
			files = append(files, filename)
		}
	}

	return files
}
