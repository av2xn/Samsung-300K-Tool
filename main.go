package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	// Argüman kontrolü
	if len(os.Args) < 2 {
		fmt.Println("Kullanım: agent.exe <1|2>")
		fmt.Println("  1 : AT+FUS? sinyali gönder")
		fmt.Println("  2 : AT+SUDDLMOD=0,0 sinyali gönder")
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
		fmt.Fprintln(os.Stderr, "[-] Hata: Geçersiz parametre. Sadece 1 veya 2 kullanın.")
		os.Exit(1)
	}

	// 1. Fiziksel donanım kontrolü
	fmt.Println("[*] Fiziksel USB veriyolu kontrol ediliyor...")
	cmdLive := exec.Command("pnputil", "/enum-devices", "/connected", "/class", "USB")
	liveOutput, err := cmdLive.Output()
	if err != nil || !strings.Contains(strings.ToUpper(string(liveOutput)), "VID_04E8") {
		fmt.Fprintln(os.Stderr, "[-] Hata: Samsung cihazı bulunamadı. Kablo veya sürücüleri kontrol edin.")
		os.Exit(1)
	}
	fmt.Println("[+] Samsung USB Composite Device tespit edildi.")

	// 2. PowerShell ile kayıtlı modem portlarını bulma
	fmt.Println("[*] Registry üzerinden COM portları taranıyor...")
	psCmd := `Get-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e96d-e325-11ce-bfc1-08002be10318}\0*' -ErrorAction SilentlyContinue | Where-Object { $_.MatchingDeviceId -match 'VID_04E8' -or $_.ProviderName -match 'SAMSUNG' } | Select-Object -ExpandProperty AttachedTo`
	cmdPort := exec.Command("powershell", "-Command", psCmd)
	portOutput, err := cmdPort.Output()
	
	ports := strings.Fields(strings.TrimSpace(string(portOutput)))
	if err != nil || len(ports) == 0 {
		fmt.Fprintln(os.Stderr, "[-] Hata: Registry'de aktif bir COM portu bulunamadı.")
		os.Exit(1)
	}
	fmt.Printf("[+] Olası portlar: %v\n", ports)

	// 3. Port Sweeping ve Sinyal İletimi
	fmt.Println("[*] Canlı bağlantı için portlar test ediliyor...")
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
		fmt.Fprintln(os.Stderr, "[-] Hata: Bütün portlar meşgul veya ulaşılamaz durumda. Açık olan diğer yazılımları kapatın.")
		os.Exit(1)
	}
	defer activeFile.Close()

	fmt.Printf("[+] %s portuna başarıyla bağlanıldı.\n", activePort)
	fmt.Printf("[*] %s komutu gönderiliyor...\n", cmdName)

	// Sinyali cihaza yaz
	_, err = activeFile.Write([]byte(cmdStr))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Hata: Veri yazılamadı: %v\n", err)
		os.Exit(1)
	}

	time.Sleep(500 * time.Millisecond)
	
	// Yanıtı oku
	buf := make([]byte, 1024)
	n, err := activeFile.Read(buf)
	if err != nil && n == 0 {
		fmt.Println("[-] Uyarı: Sinyal gönderildi fakat yanıt alınamadı (Cihaz şu an yeniden başlıyor olabilir).")
		os.Exit(0)
	}

	fmt.Println("[+] Cihaz Yanıtı:")
	fmt.Println(strings.TrimSpace(string(buf[:n])))
}