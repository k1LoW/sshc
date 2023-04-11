package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/k1LoW/sshc/v4"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// Command for SSH connection using your ssh_config ( ~/.ssh/config )
func main() {
	if len(os.Args) < 2 {
		_, _ = fmt.Fprintln(os.Stderr, "host required")
		os.Exit(2)
	}
	host := os.Args[1]

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := run(ctx, host); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		cancel()
	}()

	select {
	case <-sig:
		cancel()
	case <-ctx.Done():
	}
}

func run(ctx context.Context, host string) error {
	client, err := sshc.NewClient(host)
	if err != nil {
		return err
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	go func() {
		<-ctx.Done()
		client.Close()
	}()

	fd := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer terminal.Restore(fd, state)

	w, h, err := terminal.GetSize(fd)
	if err != nil {
		return err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	term := os.Getenv("TERM")
	if term == "" {
		term = "xterm-256color"
	}
	if err := sess.RequestPty(term, h, w, modes); err != nil {
		return err
	}

	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr
	sess.Stdin = os.Stdin

	if err := sess.Shell(); err != nil {
		return err
	}

	go func() {
		for {
			cw, ch, err := terminal.GetSize(fd)
			if err != nil {
				break
			}
			if cw != w || ch != h {
				if err := sess.WindowChange(ch, cw); err != nil {
					break
				}
				w = cw
				h = ch
			}
			time.Sleep(1 * time.Second)
		}
	}()

	if err := sess.Wait(); err != nil {
		if e, ok := err.(*ssh.ExitError); ok {
			switch e.ExitStatus() {
			case 130:
				return nil
			}
		}
		return err
	}
	return nil
}
