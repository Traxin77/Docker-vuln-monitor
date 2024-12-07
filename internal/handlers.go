package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"sync"
)

var (
	repos []string
	mu    sync.Mutex
)

func HandleRepos(w http.ResponseWriter, r *http.Request) {
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
	if err := RunCommand(cmd); err != nil {
		http.Error(w, "Failed to add repository to git-monitor", http.StatusInternalServerError)
		return
	}

	// Save the repo to the local list
	repos = append(repos, repo.Repo)
	if err := SaveReposToFile("data/repos.txt"); err != nil {
		http.Error(w, "Failed to save repository", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Repository link received and saved")
}

func HandleTestPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "Server is running"}`))
}
