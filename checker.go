package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
	"path/filepath"

	"github.com/fatih/color"
)

// Config holds the application configuration
type Config struct {
	MaxWorkers int    `json:"max_workers"`
	Timeout    int    `json:"timeout"`
	DefaultURL string `json:"default_url"`
	Mode       string `json:"mode"`
}

var (
	config     Config
	configFile = "config.json"
)

func init() {
	loadConfig()
}

func loadConfig() {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		config = Config{
			MaxWorkers: 10,
			Timeout:    10,
			DefaultURL: "example.com",
			Mode:       "HTTP",
		}
		saveConfig()
		return
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("Error parsing config file:", err)
		os.Exit(1)
	}
}

func saveConfig() {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Println("Error encoding config:", err)
		return
	}

	err = ioutil.WriteFile(configFile, data, 0644)
	if err != nil {
		fmt.Println("Error saving config:", err)
	}
}

func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func checkProxy(proxy, url string, timeout time.Duration) (bool, string) {
	proxyURL, err := proxyURLFromString(proxy)
	if err != nil {
		return false, color.RedString("Error: ") + proxy + " - Invalid proxy format"
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, color.RedString("Error: ") + proxy + " - Failed to create request"
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		errorMsg := ""
		switch {
		case strings.Contains(err.Error(), "context deadline exceeded"):
			errorMsg = "Timed out"
		case strings.Contains(err.Error(), "connection refused"):
			errorMsg = "Connection refused"
		case strings.Contains(err.Error(), "no such host"):
			errorMsg = "Host not found"
		case strings.Contains(err.Error(), "EOF"):
			errorMsg = "Connection closed unexpectedly"
		case strings.Contains(err.Error(), "malformed HTTP response"):
			errorMsg = "Invalid response from proxy"
		case strings.Contains(err.Error(), "i/o timeout"):
			errorMsg = "Connection timed out"
		default:
			errorMsg = "Unknown error"
		}
		return false, color.RedString("Error: ") + proxy + " - " + errorMsg + " (" + fmt.Sprintf("%d", duration.Milliseconds()) + " ms)"
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true, color.GreenString("Success: ") + proxy + " is working (" + fmt.Sprintf("%d", duration.Milliseconds()) + " ms)"
	}

	return false, color.YellowString("Warning: ") + proxy + " - Unexpected status code " + fmt.Sprintf("%d", resp.StatusCode) + " (" + fmt.Sprintf("%d", duration.Milliseconds()) + " ms)"
}

func testProxies(proxies []string) []string {
	if len(proxies) == 0 {
		color.Red("Error: No proxies found.")
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print(color.CyanString("Enter the URL to test (default: %s): ", config.DefaultURL))
	url, _ := reader.ReadString('\n')
	url = strings.TrimSpace(url)
	if url == "" {
		url = config.DefaultURL
	}
	url = formatURL(url)

	color.Magenta("\nTesting %d proxies against %s", len(proxies), url)
	color.Magenta("Each proxy will be tested 3 times to ensure reliability.\n")

	var wg sync.WaitGroup
	resultChan := make(chan string, len(proxies)*3)
	workingProxies := make(chan string, len(proxies))

	// Start worker goroutines
	for _, proxy := range proxies {
		wg.Add(1)
		go func(proxy string) {
			defer wg.Done()
			successCount := 0

			for i := 0; i < 3; i++ {
				success, result := checkProxy(proxy, url, time.Duration(config.Timeout)*time.Second)
				if success {
					successCount++
				}
				resultChan <- fmt.Sprintf("Test %d: %s", i+1, result)
			}

			if successCount == 3 {
				workingProxies <- proxy
				resultChan <- color.GreenString("✓ %s passed all 3 tests", proxy)
			} else {
				resultChan <- color.RedString("✗ %s failed (%d/3 successful)", proxy, successCount)
			}
		}(proxy)
	}

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
		close(workingProxies)
	}()

	// Collect and print results
	var results []string
	working := make([]string, 0)

	for result := range resultChan {
		fmt.Println(result)
		results = append(results, result)
	}

	for proxy := range workingProxies {
		working = append(working, proxy)
	}

	// Print summary
	totalTested := len(proxies)
	totalWorking := len(working)
	successRate := float64(totalWorking) / float64(totalTested) * 100

	summaryContent := []string{
		fmt.Sprintf("Total proxies tested: %d", totalTested),
		fmt.Sprintf("Working proxies (passed all 3 tests): %d", totalWorking),
		fmt.Sprintf("Success rate: %.2f%%", successRate),
	}

	summary := createBox("Summary", summaryContent...)
	color.Cyan("\n" + summary)

	// Save working proxies
	saveProxies(working)

	return working
}

func proxyURLFromString(proxy string) (*url.URL, error) {
	proxy = strings.TrimSpace(proxy)
	if !strings.HasPrefix(proxy, "http://") && !strings.HasPrefix(proxy, "https://") {
		proxy = "http://" + proxy
	}
	return url.Parse(proxy)
}

func createProxyList() {
    fmt.Println(color.CyanString("Paste your proxies below. Press Enter twice when you're done:"))
    
    var proxies []string
    reader := bufio.NewReader(os.Stdin)
    
    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            fmt.Println("Error reading input:", err)
            return
        }
        
        line = strings.TrimSpace(line)
        if line == "" {
            break // Exit the loop if user presses Enter twice
        }
        
        proxies = append(proxies, line)
    }
    
    if len(proxies) == 0 {
        color.Red("No proxies entered.")
        return
    }
    
    // Save proxies to file
    file, err := os.Create("proxies.txt")
    if err != nil {
        color.Red("Error creating proxies.txt: %v", err)
        return
    }
    defer file.Close()
    
    for _, proxy := range proxies {
        fmt.Fprintln(file, proxy)
    }
    
    color.Green("Proxy list saved to proxies.txt with %d proxies.", len(proxies))
}

func loadProxies() []string {
    possiblePaths := []string{
        "proxies.txt", // Current working directory
        filepath.Join(getCurrentDirectory(), "proxies.txt"), // Absolute path to current working directory
    }

    // Add executable directory path if we can get it
    if execPath, err := os.Executable(); err == nil {
        possiblePaths = append(possiblePaths, filepath.Join(filepath.Dir(execPath), "proxies.txt"))
    }

    var file *os.File
    var err error
    var openedPath string

    for _, path := range possiblePaths {
        color.Yellow("Looking for proxies.txt at: %s", path)
        file, err = os.Open(path)
        if err == nil {
            openedPath = path
            break
        }
    }

    if err != nil {
        color.Red("Error opening proxies.txt: %v", err)
        color.Yellow("Checked the following locations:")
        for _, path := range possiblePaths {
            color.Yellow("- %s", path)
        }
        return nil
    }
    defer file.Close()

    color.Green("Successfully opened proxies.txt at: %s", openedPath)

    var proxies []string
    scanner := bufio.NewScanner(file)
    lineCount := 0
    for scanner.Scan() {
        lineCount++
        proxy := strings.TrimSpace(scanner.Text())
        if proxy != "" {
            proxies = append(proxies, proxy)
        }
    }

    if err := scanner.Err(); err != nil {
        color.Red("Error reading proxies.txt: %v", err)
        return nil
    }

    if len(proxies) == 0 {
        if lineCount == 0 {
            color.Red("proxies.txt is empty")
        } else {
            color.Yellow("File contains %d lines, but no valid proxies found", lineCount)
        }
    } else {
        color.Green("Loaded %d proxies from proxies.txt (File contains %d lines)", len(proxies), lineCount)
    }

    return proxies
}

func getCurrentDirectory() string {
    currentDir, err := os.Getwd()
    if err != nil {
        return "Unable to get current directory"
    }
    return currentDir
}

func saveProxies(proxies []string) {
	file, err := os.Create("proxies.txt")
	if err != nil {
		fmt.Println("Error creating proxies.txt:", err)
		return
	}
	defer file.Close()

	for _, proxy := range proxies {
		fmt.Fprintln(file, proxy)
	}

	color.Green("Working proxies saved to proxies.txt")
}

func formatURL(url string) string {
	if config.Mode == "HTTP" {
		if !strings.HasPrefix(url, "http://") {
			return "http://www." + strings.TrimPrefix(url, "www.")
		}
	} else { // HTTPS
		if !strings.HasPrefix(url, "https://") {
			return "https://www." + strings.TrimPrefix(url, "www.")
		}
	}
	return url
}

func createBox(title string, content ...string) string {
	lines := append([]string{title}, content...)
	maxLen := 0
	for _, line := range lines {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}
	maxLen += 2 // Add some padding

	box := "┌" + strings.Repeat("─", maxLen+2) + "┐\n"
	box += "│ " + padCenter(title, maxLen) + " │\n"
	box += "├" + strings.Repeat("─", maxLen+2) + "┤\n"
	for _, line := range content {
		box += "│ " + padRight(line, maxLen) + " │\n"
	}
	box += "└" + strings.Repeat("─", maxLen+2) + "┘"
	return box
}

func padCenter(s string, width int) string {
	if len(s) >= width {
		return s
	}
	leftPad := (width - len(s)) / 2
	rightPad := width - len(s) - leftPad
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}


func changeSettings() {
	color.Cyan("Current settings:")
	fmt.Printf("max_workers: %d\n", config.MaxWorkers)
	fmt.Printf("timeout: %d\n", config.Timeout)
	fmt.Printf("default_url: %s\n", config.DefaultURL)
	fmt.Printf("mode: %s\n", config.Mode)

	reader := bufio.NewReader(os.Stdin)

	color.Yellow("\nEnter new values (or press Enter to keep current value):")

	fmt.Printf("max_workers (%d): ", config.MaxWorkers)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		fmt.Sscanf(input, "%d", &config.MaxWorkers)
	}

	fmt.Printf("timeout (%d): ", config.Timeout)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		fmt.Sscanf(input, "%d", &config.Timeout)
	}

	fmt.Printf("default_url (%s): ", config.DefaultURL)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		config.DefaultURL = input
	}

	for {
		fmt.Printf("mode (%s) - Enter 'HTTP' or 'HTTPS': ", config.Mode)
		input, _ = reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			break
		}
		if input == "HTTP" || input == "HTTPS" {
			config.Mode = input
			break
		}
		color.Red("Invalid input. Please enter 'HTTP' or 'HTTPS'.")
	}

	saveConfig()
	color.Green("Settings updated successfully.")
}

func displayMenu(proxies []string) {
    menu := createBox("Advanced Proxy Checker Tool - Version 2.0",
        fmt.Sprintf("Proxies Loaded: %d", len(proxies)),
        fmt.Sprintf("Current Mode: %s", config.Mode),
        "",
        "Options:",
        "1) Test Proxies",
        "2) Change settings",
        "3) Create proxy list",
        "4) Exit")

    fmt.Print(color.GreenString(menu))
}

func main() {
    proxies := loadProxies()
    if proxies == nil {
        color.Red("Failed to load proxies. Please check the proxies.txt file and try again.")
        os.Exit(1)
    }

    for {
        clearScreen()
        displayMenu(proxies)

        reader := bufio.NewReader(os.Stdin)
        fmt.Print(color.CyanString("\nEnter your choice (1-5): "))
        choice, _ := reader.ReadString('\n')
        choice = strings.TrimSpace(choice)

        switch choice {
        case "1":
            proxies = testProxies(proxies)
        case "2":
            changeSettings()
        case "3":
            createProxyList()
            proxies = loadProxies() // Reload proxies after creating new list
        case "4":
            proxies = reloadProxies()
        case "5":
            color.Green("Thank you for using the Advanced Proxy Checker Tool. Goodbye!")
            os.Exit(0)
        default:
            color.Red("Invalid choice. Please try again.")
        }

        fmt.Print(color.YellowString("\nPress Enter to continue..."))
        reader.ReadString('\n')
    }
}

func reloadProxies() []string {
    color.Yellow("Reloading proxies...")
    return loadProxies()
}