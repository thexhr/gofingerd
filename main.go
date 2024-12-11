package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"strings"
	"syscall"
)

type localUser struct {
	userName string
	name string
	homeDirectory string
	shell string
	forward string
}


func validateUserString(userName string) error {
	const max_len = 32

	if len(userName) < 1 || len(userName) > max_len {
		return fmt.Errorf("Too long or too short")
	}

	if strings.ContainsAny(userName, " ") {
		return fmt.Errorf("Invalid whitespace found")
	}

	if strings.ContainsAny(userName, "|;/&$`?=+()\\") {
		return fmt.Errorf("Invalid characters found")
	}

	if strings.Contains(userName, "@") {
		if strings.Count(userName, "@") > 1 {
			return fmt.Errorf("Found more than one @")
		}
	}

	return nil
}

func (l localUser) userExists () bool {
	_, err := user.Lookup(l.userName)
	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

func (l *localUser) getShell() (error) {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return fmt.Errorf("Cannot open /etc/passwd")
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		sline := strings.Split(line, ":")
		if len(sline) < 7 {
			continue
		}
		if sline[0] == l.userName {
			l.shell = sline[6]
		}
	}

	return nil
}

func (l *localUser) getUserDetails() (error) {
	if len(l.userName) <= 0 {
		return fmt.Errorf("Invalid username")
	}

	user, err := user.Lookup(l.userName)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("Lookup failed: %v", err.Error())
	}

	l.homeDirectory = user.HomeDir
	l.name = user.Name

	return nil
}

func (l *localUser) mailForward() (error) {
	var hd string
	if hd = os.Getenv("HOME"); hd == "" {
		return fmt.Errorf("$HOME not defined")
	}

	forwardPath := hd + "/.forward"
	f, err := os.Open(forwardPath)
	if err != nil {
		return fmt.Errorf("Cannot open %s", forwardPath)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	var line string
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Index(line, "#") == 1 || len(line) == 0 {
			continue
		}
	}

	l.forward = line
	return nil
}

func showUserDetails(msg string) (string, error) {
	resp := ""
	if err := validateUserString(msg); err != nil {
		return "", fmt.Errorf("Error: %s", err.Error())
	}

	u := localUser{userName: msg}
	if err := u.getUserDetails(); err != nil {
		return "", fmt.Errorf("Error: %s", err.Error())
	}
	if err := u.getShell(); err != nil {
		return "", fmt.Errorf("Error: %s", err.Error())
	}

	u.mailForward()

	if u.userExists() != false {
		resp = fmt.Sprintf("Login: %-32s Name : %s\n", u.userName, u.name)
		resp += fmt.Sprintf("Directory: %-28s Shell: %s\n", u.homeDirectory, u.shell)
		if len(u.forward) > 0 {
			resp += fmt.Sprintf("Mail forwarded to %s\n", u.forward)
		}
	}

	return resp, nil
}

func main() {

	log.SetFlags(0)
	if syscall.Getuid() != 0 {
		log.Fatal("You need to run as root")
	}

	ln, err := net.Listen("tcp", ":79")
	if err != nil {
		log.Fatal("Cannot open listening socket: ", err.Error())
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept() error: %s", err.Error())
			continue
		}

		buf := make([]byte, 33)
		i, err := conn.Read(buf)
		if err != nil {
			log.Fatal("Cannot read from socket", i, err.Error())
		}
		userName := strings.TrimSpace(string(buf[:i]))
		resp, err := showUserDetails(userName)
		if err != nil {
			conn.Close()
		}
		conn.Write([]byte(resp))
		conn.Close()
	}
}
