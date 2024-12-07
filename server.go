package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Repo struct {
	Repo string `json:"repo"`
}

var (
	repos      []string
	mu         sync.Mutex
	gitMonitor = "git-monitor" // Path to git-monitor binary
)

func main() {
	// Load existing repositories from file
	loadReposFromFile("repos.txt")

	// Start monitoring repositories for new commits
	go monitorRepositories()

	// Set up HTTP server
	http.HandleFunc("/repos", handleRepos)
	http.HandleFunc("/test", handleTestPage)

	port := "8080"
	fmt.Printf("Server is running on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}

// Handle incoming repository submissions
func handleRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var repo Repo
	err := json.NewDecoder(r.Body).Decode(&repo)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// Add the repo to git-monitor
	cmd := exec.Command(gitMonitor, "add", repo.Repo)
	if err := runCommand(cmd); err != nil {
		http.Error(w, "Failed to add repository to git-monitor", http.StatusInternalServerError)
		return
	}

	// Save the repo to the local list
	repos = append(repos, repo.Repo)
	if err := saveReposToFile("repos.txt"); err != nil {
		http.Error(w, "Failed to save repository", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Repository link received and saved")
}

// Monitor repositories for new commits
func monitorRepositories() {
	for {
		mu.Lock()
		cmd := exec.Command(gitMonitor, "check")
		output, err := cmd.CombinedOutput() // CombinedOutput to capture stdout and stderr
		mu.Unlock()

		fmt.Printf("Git-monitor output:\n%s\n", output) // Log output

		if err != nil {
			fmt.Printf("Error checking repositories: %v\n", err)
			time.Sleep(30 * time.Second)
			continue
		}

		// Process output for new commits
		processMonitorOutput(string(output))
		time.Sleep(30 * time.Second)
	}
}

// Process the output of git-monitor for new commits
func processMonitorOutput(output string) {
	fmt.Printf("Processing git-monitor output:\n%s\n", output)

	if strings.Contains(output, "Already up-to-date.") {
		fmt.Println("No new push events detected.")
		return
	}

	fmt.Println("New push event detected!")

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "|") { // Process lines with repository info
			parts := strings.Split(line, "|")
			if len(parts) < 3 {
				fmt.Printf("Skipping malformed line: %s\n", line)
				continue
			}

			repoName := strings.TrimSpace(parts[0])
			remoteURL := strings.TrimSpace(parts[2])

			// Extract the base repository URL (remove branch/commits info)
			baseURL := strings.Split(remoteURL, "/commits")[0]

			fmt.Printf("Running handleNewPush: Repo=%s, URL=%s\n", repoName, baseURL)
			go handleNewPush(repoName, baseURL) // Run in a goroutine
		}
	}
}

// Handle new commits
func handleNewPush(repoName, repoURL string) {
	fmt.Printf("Starting Dockerization and scanning for repo: %s, URL: %s\n", repoName, repoURL)

	// Step 1: Dockerize the repository
	dockerizeCmd := exec.Command("./docker-vuln", "dock", repoURL)
	fmt.Printf("Running command: %v\n", dockerizeCmd.Args)
	if err := runCommand(dockerizeCmd); err != nil {
		fmt.Printf("Error dockerizing repository %s: %v\n", repoName, err)
		return
	}
	fmt.Printf("Successfully dockerized repository: %s\n", repoName)

	// Step 2: Scan the repository
	currentDate := time.Now().Format("2006-01-02")
	outputFile := fmt.Sprintf("/app/%s_%s.txt", repoName, currentDate) // Save to /app directory
	scanCmd := exec.Command("./docker-vuln", "scan", repoName, "-o", outputFile)
	fmt.Printf("Running command: %v\n", scanCmd.Args)
	if err := runCommand(scanCmd); err != nil {
		fmt.Printf("Error scanning repository %s: %v\n", repoName, err)
		return
	}
	fmt.Printf("Scan completed and saved to %s\n", outputFile)

	catCmd := exec.Command("cat", outputFile)
	runCommand(catCmd)
}

// Utility function to run shell commands
func runCommand(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Handle test page
func handleTestPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"status": "Server is running"}`)
}

// func startDockerDaemon() error {
// 	fmt.Println("Starting Docker daemon...")

// 	// Use `sh` to run the command in a shell, which understands `&`
// 	cmd := exec.Command("sh", "-c", "sudo dockerd &")

// 	// Capture output for debugging (optional)
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr

// 	// Start the command
// 	if err := cmd.Start(); err != nil {
// 		return fmt.Errorf("failed to start Docker daemon: %w", err)
// 	}

// 	// Wait for the Docker daemon to initialize
// 	fmt.Println("Waiting for Docker daemon to initialize...")
// 	time.Sleep(10 * time.Second)

// 	return nil
// }

// Load repositories from a file
func loadReposFromFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		fmt.Printf("Error reading repos file: %v\n", err)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	repos = strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, repo := range repos {
		// Add the repo to git-monitor
		exec.Command(gitMonitor, "add", repo).Run()
	}
}

// Save repositories to a file
func saveReposToFile(filename string) error {
	mu.Lock()
	defer mu.Unlock()

	data := []byte(strings.Join(repos, "\n"))
	return os.WriteFile(filename, data, 0644)
}
