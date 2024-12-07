package main

import (
	"fmt"
	"net/http"
	"os"

	"monitor/internal"
)

func main() {
	// Load existing repositories from file
	internal.LoadReposFromFile("data/repos.txt")

	// Start monitoring repositories for new commits
	go internal.MonitorRepositories()

	// Set up HTTP server
	http.HandleFunc("/repos", internal.HandleRepos)
	http.HandleFunc("/test", internal.HandleTestPage)

	port := "8080"
	fmt.Printf("Server is running on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}
