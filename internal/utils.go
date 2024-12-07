package internal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func RunCommand(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func LoadReposFromFile(filename string) {
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
		exec.Command(gitMonitor, "add", repo).Run()
	}
}

func SaveReposToFile(filename string) error {
	mu.Lock()
	defer mu.Unlock()

	data := []byte(strings.Join(repos, "\n"))
	return os.WriteFile(filename, data, 0644)
}
