package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	// Argument check
	if len(os.Args) < 2 {
		fmt.Println("Usage: agent.exe <1|2>")
		fmt.Println("  1 : Send AT+FUS? signal")
		fmt.Println("  2 : Send AT+SUDDLMOD=0,0 signal")
		os.Exit(1)
	}

	var cmdStr string
	var cmdName string

	switch os.Args[1] {
	case "1":
		cmdStr = "AT+FUS?\r\n"
		cmdName = "AT+FUS?"
	case "2":
		cmdStr = "AT+SUDDLMOD=0,0\r\n"
		cmdName = "AT+SUDDLMOD=0,0"
	default:
		fmt.Fprintln(os.Stderr, "[-] Error: Invalid parameter. Use only 1 or 2.")
		os.Exit(1)
	}

	// 1. Physical hardware check
	fmt.Println("[*] Checking physical USB bus...")
	cmdLive := exec.Command("pnputil", "/enum-devices", "/connected", "/class", "USB")
	liveOutput, err := cmdLive.Output()
	if err != nil || !strings.Contains(strings.ToUpper(string(liveOutput)), "VID_04E8") {
		fmt.Fprintln(os.Stderr, "[-] Error: Samsung device not found. Check the cable or drivers.")
		os.Exit(1)
	}
	fmt.Println("[+] Samsung USB Composite Device detected.")

	// 2. Finding registered modem ports via PowerShell
	fmt.Println("[*] Scanning COM ports via Registry...")
	psCmd := `Get-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e96d-e325-11ce-bfc1-08002be10318}\0*' -ErrorAction SilentlyContinue | Where-Object { $_.MatchingDeviceId -match 'VID_04E8' -or $_.ProviderName -match 'SAMSUNG' } | Select-Object -ExpandProperty AttachedTo`
	cmdPort := exec.Command("powershell", "-Command", psCmd)
	portOutput, err := cmdPort.Output()
	
	ports := strings.Fields(strings.TrimSpace(string(portOutput)))
	if err != nil || len(ports) == 0 {
		fmt.Fprintln(os.Stderr, "[-] Error: No active COM port found in Registry.")
		os.Exit(1)
	}
	fmt.Printf("[+] Possible ports: %v\n", ports)

	// 3. Port Sweeping and Signal Transmission
	fmt.Println("[*] Testing ports for live connection...")
	var activeFile *os.File
	var activePort string

	for _, p := range ports {
		portPath := `\\.\` + p
		file, err := os.OpenFile(portPath, os.O_RDWR, 0)
		if err == nil {
			activeFile = file
			activePort = p
			break
		}
	}

	if activeFile == nil {
		fmt.Fprintln(os.Stderr, "[-] Error: All ports are busy or unreachable. Close other open software.")
		os.Exit(1)
	}
	defer activeFile.Close()

	fmt.Printf("[+] Successfully connected to port %s.\n", activePort)
	fmt.Printf("[*] Sending command %s...\n", cmdName)

	// Write signal to the device
	_, err = activeFile.Write([]byte(cmdStr))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Error: Failed to write data: %v\n", err)
		os.Exit(1)
	}

	time.Sleep(500 * time.Millisecond)
	
	// Read response
	buf := make([]byte, 1024)
	n, err := activeFile.Read(buf)
	if err != nil && n == 0 {
		fmt.Println("[-] Warning: Signal sent but no response received (Device might be rebooting now).")
		os.Exit(0)
	}

	fmt.Println("[+] Device Response:")
	fmt.Println(strings.TrimSpace(string(buf[:n])))
}
