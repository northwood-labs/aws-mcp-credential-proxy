package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/northwood-labs/debug"
)

func main() {
	// Fetch credentials and return expiration time
	fetchCredentials := func() (*time.Time, error) {
		// Read the environment variables produced by AWS Vault.
		target, err := url.Parse(os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI"))
		if err != nil {
			log.Fatalln("Bad AWS_CONTAINER_CREDENTIALS_FULL_URI:", err.Error())
		}

		authToken := os.Getenv("AWS_CONTAINER_AUTHORIZATION_TOKEN")

		// Make HTTP request to endpoint to fetch credentials.
		client := &http.Client{}
		req, err := http.NewRequest("GET", target.String(), nil)
		if err != nil {
			log.Fatalln("Failed to create request:", err.Error())
		}

		if authToken != "" {
			req.Header.Set("Authorization", authToken)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		// Read response body.
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		// Parse the JSON response into a data object.
		var jsonData map[string]any
		err = json.Unmarshal(body, &jsonData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}

		var expiration *time.Time

		pp := debug.GetSpew()
		pp.Dump(jsonData)

		// Update the keys to match the environment variable names.
		for key, value := range jsonData {
			strValue := fmt.Sprintf("%v", value)

			switch key {
			case "AccessKeyId":
				err = os.Setenv("AWS_ACCESS_KEY_ID", strValue)
			case "SecretAccessKey":
				err = os.Setenv("AWS_SECRET_ACCESS_KEY", strValue)
			case "Token":
				err = os.Setenv("AWS_SESSION_TOKEN", strValue)
			case "Expiration":
				// Parse the expiration timestamp
				if expTime, parseErr := time.Parse(time.RFC3339, strValue); parseErr == nil {
					expiration = &expTime
				}
			default:
				continue
			}

			if err != nil {
				return nil, fmt.Errorf("failed to set environment variable: %w", err)
			}
		}

		return expiration, nil
	}

	// Fetch credentials initially
	expiration, err := fetchCredentials()
	if err != nil {
		log.Fatalln("Failed to fetch initial credentials:", err.Error())
	}

	// Start background credential refresh if we have an expiration time
	if expiration != nil {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			for {
				// Wait until expiration time (with a small buffer)
				// refreshTime := expiration.Add(-30 * time.Second) // Refresh 30 seconds before expiration
				refreshTime := time.Now().Add(30 * time.Second)
				WaitUntil(ctx, refreshTime)

				// Check if context was cancelled
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(1 * time.Second)
					fmt.Fprintf(os.Stderr, "%v\n", expiration)
				}

				// Fetch new credentials
				newExpiration, err := fetchCredentials()
				if err != nil {
					log.Printf("Failed to refresh credentials: %v", err)

					// Continue with existing credentials and retry in 1 minute
					time.Sleep(1 * time.Minute)

					continue
				}

				if newExpiration != nil {
					expiration = newExpiration
				}
			}
		}()
	}

	RunCmd(os.Args[1:])
}

// WaitUntil will block until the given time. Can be cancelled by canceling the
// context.
func WaitUntil(ctx context.Context, t time.Time) {
	diff := time.Until(t)
	if diff <= 0 {
		return
	}

	timer := time.NewTimer(diff)
	defer timer.Stop()

	select {
	case <-timer.C:
		return
	case <-ctx.Done():
		return
	}
}

// RunCmd runs the command that is passed downstream from this command.
func RunCmd(args []string) {
	// Find the "--" separator in command line arguments
	var cmdArgs []string

	for i, arg := range args {
		if arg == "--" {
			cmdArgs = args[i+1:]
			break
		}
	}

	if len(cmdArgs) == 0 {
		return
	}

	// Execute the command
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}

		log.Fatalln("Failed to execute command:", err.Error())
	}

}
