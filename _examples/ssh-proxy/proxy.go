package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
)

func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

func main() {
	ssh.Handle(func(s ssh.Session) {
		// var cmd *exec.Cmd
		// if len(s.Command()) > 0 && false {
		// 	cmd = exec.Command(s.Command()[0], s.Command()[1:len(s.Command())]...)
		// } else {
		// 	cmd = exec.Command("sh", "-i")
		// }
		cmd := exec.Command("sh")
		ptyReq, winCh, isPty := s.Pty()
		if isPty {
			cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
			cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", os.Getenv("HOME")))
			f, err := pty.Start(cmd)
			if err != nil {
				panic(err)
			}
			go func() {
				for win := range winCh {
					setWinsize(f, win.Width, win.Height)
				}
			}()
			go func() {
				io.Copy(f, s) // stdin
			}()
			io.Copy(s, f) // stdout
			cmd.Wait()
		} else {
			println("No PTY requested.")
			stdout, _ := cmd.StdoutPipe()
			stdin, _ := cmd.StdinPipe()
			stderr, _ := cmd.StderrPipe()
			cmd.Start()

			go func() {
				io.Copy(stdin, s) // stdin
			}()
			go func() {
				io.Copy(s.Stderr(), stderr) // stderr
			}()
			io.Copy(s, stdout) // stdout
			cmd.Wait()
		}
	})

	log.Println("starting ssh server on port 2222...")
	log.Fatal(ssh.ListenAndServe(":2222", nil))
}
