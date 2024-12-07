package internal

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var gitMonitor = "git-monitor" // Path to git-monitor binary

func MonitorRepositories() {
	for {
		mu.Lock()
		cmd := exec.Command(gitMonitor, "check")
		output, err := cmd.CombinedOutput()
		mu.Unlock()

		fmt.Printf("Git-monitor output:\n%s\n", output)
		if err != nil {
			fmt.Printf("Error checking repositories: %v\n", err)
			time.Sleep(30 * time.Second)
			continue
		}

		ProcessMonitorOutput(string(output))
		time.Sleep(30 * time.Second)
	}
}

func ProcessMonitorOutput(output string) {
	fmt.Printf("Processing git-monitor output:\n%s\n", output)

	if strings.Contains(output, "Already up-to-date.") {
		fmt.Println("No new push events detected.")
		return
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "|") {
			parts := strings.Split(line, "|")
			if len(parts) < 3 {
				continue
			}

			repoName := strings.TrimSpace(parts[0])
			remoteURL := strings.TrimSpace(parts[2])
			baseURL := strings.Split(remoteURL, "/commits")[0]

			go HandleNewPush(repoName, baseURL)
		}
	}
}

func HandleNewPush(repoName, repoURL string) {
	fmt.Printf("Starting Dockerization and scanning for repo: %s, URL: %s\n", repoName, repoURL)

	dockerizeCmd := exec.Command("./bin/docker-vuln", "dock", repoURL)
	if err := RunCommand(dockerizeCmd); err != nil {
		fmt.Printf("Error dockerizing repository %s: %v\n", repoName, err)
		return
	}

	outputFile := fmt.Sprintf("/app/%s_%s.txt", repoName, time.Now().Format("2006-01-02"))
	scanCmd := exec.Command("./bin/docker-vuln", "scan", repoName, "-o", outputFile)
	if err := RunCommand(scanCmd); err != nil {
		fmt.Printf("Error scanning repository %s: %v\n", repoName, err)
		return
	}

	catCmd := exec.Command("cat", outputFile)
	RunCommand(catCmd)
}
