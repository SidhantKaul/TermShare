package pty

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

type PTYSession struct {
	ptyio *os.File
	cmd   *exec.Cmd
}

func CreatePtySession(writer io.Writer) (*PTYSession, error) {
	cmd := exec.Command("bash", "-i")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Printf("while Creating PTY: %s\n", err)
		return nil, err
	}

	session := PTYSession{ptmx, cmd}

	if _, err := term.MakeRaw(int(ptmx.Fd())); err != nil { // set terminal to raw mode, so that input's don't get echoed
		fmt.Printf("while setting terminal to raw mode: %s\n", err)
		ptmx.Close()
		cmd.Process.Kill()
		return nil, err
	}

	go session.CopyOutput(writer)

	// Wait for Bash to initialize
	time.Sleep(250 * time.Millisecond)

	session.FeedInput("true\n", writer)

	return &session, nil
}

func (s *PTYSession) ClosePtySession() error {
	if s.ptyio != nil {
		s.ptyio.Close()
	}

	// Step 2: Kill the child process if still running
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}

	return nil
}

func (s *PTYSession) CopyOutput(writer io.Writer) {

	errChan := make(chan error, 1)

	//go func() {
	defer func() {
		if f, ok := writer.(interface{ Flush() error }); ok {
			if err := f.Flush(); err != nil {
				errChan <- err
			}
		}
	}()

	for {

		buf := make([]byte, 1024)

		n, err := s.ptyio.Read(buf)

		if err != nil {
			errChan <- fmt.Errorf("PTY read error: %w", err)
			return
		}

		if _, err := writer.Write(buf[:n]); err != nil {
			errChan <- fmt.Errorf("writer error: %w", err)
			return
		}
	}
	//}()
}

// WriteInput writes client input to the PTY (e.g., from active editor)
func (session *PTYSession) FeedInput(data string, writer io.Writer) {

	_, err := session.ptyio.WriteString(data + "\n")

	if err != nil {
		log.Printf("Error while feeding input to bash %s", err)
	}
}
