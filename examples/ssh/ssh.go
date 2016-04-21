package main

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
	"path/filepath"
)

func main() {

	if len(os.Args) != 4 {
		basename := filepath.Base(os.Args[0])
		log.Fatalf("usage: %s hostname username password", basename)
	}

	host := os.Args[1]
	user := os.Args[2]
	pass := os.Args[3]

	// Create client config
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
	}

	// Connect to ssh server
	log.Printf("** opening: %s", host)
	conn, err := ssh.Dial("tcp", host, config)
	if err != nil {
		log.Fatalf("unable to connect: %s", err)
	}
	defer conn.Close()
	log.Printf("** connected: %s", host)

	// Create a session
	session, err := conn.NewSession()
	if err != nil {
		log.Fatalf("unable to create session: %s", err)
	}
	defer session.Close()
	log.Printf("** session open")

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO: 0, // disable echoing
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		log.Fatalf("request for pseudo terminal failed: %s", err)
	}
	log.Printf("** pseudo-terminal ready")

	var bufOut, bufErr bytes.Buffer
	session.Stdout = &bufOut
	session.Stderr = &bufErr

	writer, inErr := session.StdinPipe()
	if inErr != nil {
		log.Fatalf("StdinPipe: %s", inErr)
	}

	if shellErr := session.Shell(); shellErr != nil {
		log.Fatalf("Remote shell error: %s", shellErr)
	}

	// commands for juniper junos
	cmds := []string{"set cli screen-length 0\n", "sh ver\n", "sh conf | disp set\n", "exit\n"}

	for _, c := range cmds {
		log.Printf("** sending command: [%s]", c)
		_, sendErr := writer.Write([]byte(c))
		if sendErr != nil {
			log.Printf("write error: %v", sendErr)
		}
	}

	log.Printf("** waiting")

	if waitErr := session.Wait(); waitErr != nil {
		log.Printf("wait error: %v", waitErr)
	}

	log.Printf("** done: out=[%s] err=[%s]", bufOut.String(), bufErr.String())
}
