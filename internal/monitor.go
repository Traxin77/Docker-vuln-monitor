package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoURI string
var gitMonitor = "git-monitor" // Path to git-monitor binary

func init() {
    // Load .env file
    if err := godotenv.Load(); err != nil {
        fmt.Println("Error loading .env file:", err)
    }

    // Get MongoDB URI from the environment
    mongoURI = os.Getenv("MONGO_URI")
    if mongoURI == "" {
        fmt.Println("MONGO_URI is not set in the environment variables.")
        os.Exit(1)
    }
}


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
	if err := ExecuteCommand(dockerizeCmd); err != nil {
		fmt.Printf("%s: %v\n", repoName, err)
		return
	}

	outputFile := fmt.Sprintf("/app/%s_%s.txt", repoName, time.Now().Format("2006-01-02"))
	scanCmd := exec.Command("./bin/docker-vuln", "scan", repoName, "-o", outputFile)
	if err := ExecuteCommand(scanCmd); err != nil {
		fmt.Printf("%s: %v\n", repoName, err)
		return
	}

	UploadToMongoDB(outputFile, repoName)
}

func ExecuteCommand(cmd *exec.Cmd) error {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Command failed: %s\n", stderr.String())
		return err
	}
	return nil
}

func UploadToMongoDB(filePath, repoName string) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
	fmt.Printf("Loaded MONGO_URI: %s\n", mongoURI)
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
    if err != nil {
        fmt.Printf("Error connecting to MongoDB: %v\n", err)
        return
    }
    defer client.Disconnect(ctx)

    db := client.Database("docker-monitor")
    collection := db.Collection("scan_results")

    file, err := os.Open(filePath)
    if err != nil {
        fmt.Printf("Error opening file %s: %v\n", filePath, err)
        return
    }
    defer file.Close()

    content, err := io.ReadAll(file)
    if err != nil {
        fmt.Printf("Error reading file %s: %v\n", filePath, err)
        return
    }

    document := map[string]interface{}{
        "repo_name": repoName,
        "scan_date": time.Now(),
        "content":   string(content),
    }

    _, err = collection.InsertOne(ctx, document)
    if err != nil {
        fmt.Printf("Error inserting document into MongoDB: %v\n", err)
        return
    }

    fmt.Printf("File %s successfully saved to MongoDB as a document for %s\n", filePath, repoName)
}
