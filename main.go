package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/ktr0731/go-fuzzyfinder"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"
)

type SSHHost struct {
	Name         string
	HostName     string
	Port         string
	User         string
	Comment      string
	IdentityFile string
}

func (h SSHHost) Display() string {
	info := fmt.Sprintf("%-20s", h.Name)
	if h.User != "" {
		info += fmt.Sprintf(" user=%-10s", h.User)
	}
	if h.HostName != "" {
		info += fmt.Sprintf(" host=%-20s", h.HostName)
	}
	if h.Port != "" && h.Port != "22" {
		info += fmt.Sprintf(" port=%s", h.Port)
	}
	if h.Comment != "" {
		info += fmt.Sprintf(" (%s)", h.Comment)
	}
	return info
}

func parseSSHConfig(path string) ([]SSHHost, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var hosts []SSHHost
	var current *SSHHost
	var lastComment string

	hostRe := regexp.MustCompile(`(?i)^Host\s+(.+)$`)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 주석 저장
		if strings.HasPrefix(line, "#") {
			lastComment = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			continue
		}

		if line == "" {
			continue
		}

		// Host 라인 파싱
		if matches := hostRe.FindStringSubmatch(line); matches != nil {
			hostName := matches[1]
			// 이전 호스트 저장
			if current != nil {
				hosts = append(hosts, *current)
			}
			// 와일드카드 호스트 제외
			if strings.Contains(hostName, "*") {
				current = nil
				continue
			}
			current = &SSHHost{
				Name:    hostName,
				Port:    "22",
				Comment: lastComment,
			}
			lastComment = ""
			continue
		}

		if current == nil {
			continue
		}

		// 속성 파싱 (공백 또는 = 구분자 지원)
		var key, value string
		if idx := strings.Index(line, "="); idx != -1 {
			key = strings.ToLower(strings.TrimSpace(line[:idx]))
			value = strings.TrimSpace(line[idx+1:])
		} else {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) < 2 {
				parts = strings.SplitN(line, "\t", 2)
				if len(parts) < 2 {
					continue
				}
			}
			key = strings.ToLower(strings.TrimSpace(parts[0]))
			value = strings.TrimSpace(parts[1])
		}

		switch key {
		case "hostname":
			current.HostName = value
		case "port":
			current.Port = value
		case "user":
			current.User = value
		case "identityfile":
			current.IdentityFile = expandPath(value)
		}
	}

	if current != nil {
		hosts = append(hosts, *current)
	}

	return hosts, scanner.Err()
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func getSSHConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "config")
}

func getDefaultKeyPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	sshDir := filepath.Join(home, ".ssh")
	return []string{
		filepath.Join(sshDir, "id_ed25519"),
		filepath.Join(sshDir, "id_rsa"),
		filepath.Join(sshDir, "id_ecdsa"),
		filepath.Join(sshDir, "id_dsa"),
	}
}

func getSSHAgentAuth() ssh.AuthMethod {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil
	}

	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers)
}

func getKeyAuth(keyPath string) ssh.AuthMethod {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		// 암호화된 키인 경우 패스워드 요청
		if strings.Contains(err.Error(), "passphrase") {
			fmt.Printf("Enter passphrase for key '%s': ", keyPath)
			passphrase, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return nil
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, passphrase)
			if err != nil {
				return nil
			}
		} else {
			return nil
		}
	}

	return ssh.PublicKeys(signer)
}

func getPasswordAuth() ssh.AuthMethod {
	return ssh.PasswordCallback(func() (string, error) {
		fmt.Print("Password: ")
		password, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return "", err
		}
		return string(password), nil
	})
}

func getKeyboardInteractiveAuth() ssh.AuthMethod {
	return ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		if instruction != "" {
			fmt.Println(instruction)
		}
		answers := make([]string, len(questions))
		for i, question := range questions {
			fmt.Print(question)
			if echos[i] {
				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				answers[i] = strings.TrimSpace(answer)
			} else {
				password, _ := term.ReadPassword(int(syscall.Stdin))
				fmt.Println()
				answers[i] = string(password)
			}
		}
		return answers, nil
	})
}

var verbose = os.Getenv("MC_DEBUG") != ""

func debugLog(format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

func connectSSH(host SSHHost) error {
	// 호스트 주소 결정
	hostname := host.HostName
	if hostname == "" {
		hostname = host.Name
	}
	addr := fmt.Sprintf("%s:%s", hostname, host.Port)

	// 사용자 결정
	user := host.User
	if user == "" {
		user = os.Getenv("USER")
	}

	debugLog("Connecting to %s@%s", user, addr)
	debugLog("IdentityFile from config: %s", host.IdentityFile)

	// 인증 방법 수집
	var authMethods []ssh.AuthMethod

	// 1. 지정된 IdentityFile (최우선)
	if host.IdentityFile != "" {
		debugLog("Trying IdentityFile: %s", host.IdentityFile)
		if _, err := os.Stat(host.IdentityFile); err != nil {
			debugLog("IdentityFile not found: %v", err)
		} else if keyAuth := getKeyAuth(host.IdentityFile); keyAuth != nil {
			debugLog("Added key auth from IdentityFile: %s", host.IdentityFile)
			authMethods = append(authMethods, keyAuth)
		} else {
			debugLog("Failed to load key from IdentityFile: %s", host.IdentityFile)
		}
	}

	// 2. SSH Agent (IdentityFile이 없는 경우에만)
	if host.IdentityFile == "" {
		if agentAuth := getSSHAgentAuth(); agentAuth != nil {
			debugLog("Added SSH Agent auth")
			authMethods = append(authMethods, agentAuth)
		}
	}

	// 3. 기본 키 파일들 (IdentityFile이 없는 경우에만)
	if host.IdentityFile == "" {
		for _, keyPath := range getDefaultKeyPaths() {
			if _, err := os.Stat(keyPath); err == nil {
				if keyAuth := getKeyAuth(keyPath); keyAuth != nil {
					debugLog("Added key auth from default key: %s", keyPath)
					authMethods = append(authMethods, keyAuth)
				}
			}
		}
	}

	// 4. 패스워드 인증
	authMethods = append(authMethods, getPasswordAuth())

	// 5. Keyboard Interactive
	authMethods = append(authMethods, getKeyboardInteractiveAuth())

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 실제 환경에서는 known_hosts 검증 필요
	}

	// SSH 연결
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer client.Close()

	// 세션 생성
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// 터미널 설정
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("failed to set raw terminal: %v", err)
	}
	defer term.Restore(fd, oldState)

	// 터미널 크기 가져오기
	width, height, err := term.GetSize(fd)
	if err != nil {
		width, height = 80, 24
	}

	// PTY 요청
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	termType := os.Getenv("TERM")
	if termType == "" {
		termType = "xterm-256color"
	}

	if err := session.RequestPty(termType, height, width, modes); err != nil {
		return fmt.Errorf("failed to request PTY: %v", err)
	}

	// 입출력 연결
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// 윈도우 크기 변경 처리
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	go func() {
		for range sigwinch {
			if w, h, err := term.GetSize(fd); err == nil {
				session.WindowChange(h, w)
			}
		}
	}()
	defer signal.Stop(sigwinch)

	// 쉘 시작
	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %v", err)
	}

	// 세션 종료 대기
	return session.Wait()
}

func main() {
	configPath := getSSHConfigPath()
	if configPath == "" {
		fmt.Fprintln(os.Stderr, "Error: cannot find home directory")
		os.Exit(1)
	}

	hosts, err := parseSSHConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing SSH config: %v\n", err)
		os.Exit(1)
	}

	if len(hosts) == 0 {
		fmt.Fprintln(os.Stderr, "No SSH hosts found in config")
		os.Exit(1)
	}

	// 초기 검색어 설정 (커맨드 라인 인자)
	initialQuery := ""
	if len(os.Args) > 1 {
		initialQuery = strings.Join(os.Args[1:], " ")
	}

	// go-fuzzyfinder로 호스트 선택
	idx, err := fuzzyfinder.Find(
		hosts,
		func(i int) string {
			return hosts[i].Display()
		},
		fuzzyfinder.WithPromptString("SSH > "),
		fuzzyfinder.WithHeader("Select a host to connect"),
		fuzzyfinder.WithCursorPosition(fuzzyfinder.CursorPositionTop),
		fuzzyfinder.WithQuery(initialQuery),
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			host := hosts[i]
			return fmt.Sprintf("Name:     %s\nHost:     %s\nUser:     %s\nPort:     %s\nKey:      %s\nComment:  %s",
				host.Name, host.HostName, host.User, host.Port, host.IdentityFile, host.Comment)
		}),
	)
	if err != nil {
		// 취소됨 (ESC 또는 Ctrl+C)
		os.Exit(0)
	}

	host := hosts[idx]

	// SSH 접속
	fmt.Printf("Connecting to %s...\n", host.Name)
	if err := connectSSH(host); err != nil {
		fmt.Fprintf(os.Stderr, "SSH error: %v\n", err)
		os.Exit(1)
	}
}