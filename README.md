# Advanced Proxy Checker Tool

An advanced tool designed for efficiently testing and verifying the functionality of multiple proxies. This tool allows for concurrent testing, provides detailed error reporting, and summarizes the success rate of tested proxies.

![Alt text](https://images2.imgbox.com/6d/ce/5VL6vGlK_o.png)<!-- Replace with the actual path to your image -->

## Features

- Test HTTP and HTTPS proxies
- Configure the number of test attempts per proxy
- Customizable timeout settings
- Concurrent testing for faster results
- Built-in proxy loader/saver
- User-friendly CLI interface with colored output
- Saves working proxies to a file for easy reuse

## Prerequisites

- Go 1.16 or higher

## Installation

1. Clone the repository:
   `git clone https://github.com/PoppingXanax/proxychecker.git`
   `cd proxychecker`
3. Install dependencies:
   `go mod tidy`

## Usage

1. Create a file named `proxies.txt` in the same directory as the tool, and add your proxies, one per line. For example:
   ```
   192.168.1.1:8080
   10.0.0.1:3128
   proxy.example.com:8000
   ```
2. Run the tool:
- On Windows:
  ```
  go run checker.go
  ```
- On Mac/Linux:
  ```
  go run checker.go
  ```

3. Follow the on-screen prompts to test your proxies.

## Building the Tool

To create an executable for your specific platform:

- On Windows:
  `go build -o proxychecker.exe checker.go`
- On Mac/Linux:
  `go build -o proxychecker main.go`

After building, you can run the tool using:
- Windows: `.\proxy-checker.exe`
- Mac/Linux: `./proxy-checker`

## Configuration

You can modify the following settings through the tool's interface:

- Max Workers: Number of concurrent workers for proxy testing
- Timeout: Maximum time (in seconds) to wait for a proxy response
- Default URL: The URL used for testing proxies
- Mode: HTTP or HTTPS

## Stats

[![Hits](https://hits.seeyoufarm.com/api/count/incr/badge.svg?url=https%3A%2F%2Fgithub.com%2FPoppingXanax%2Fproxychecker&count_bg=%2300CE9E&title_bg=%234308D7&icon=&icon_color=%23E7E7E7&title=hits&edge_flat=false)](https://hits.seeyoufarm.com)
