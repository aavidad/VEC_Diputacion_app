// Fixture local: un escritor PostgreSQL real con admisión y drenaje por flock.
// No es un adaptador para servicios VEC ni forma parte del binario de captura.
package main

import (
	"context"
	"math"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		os.Exit(1)
	}
	modo, dir := os.Args[1], os.Args[2]
	raiz, err := os.OpenRoot(dir) // #nosec G703 -- explicitly selected private synthetic fixture root, never received from HTTP.
	if err != nil {
		os.Exit(1)
	}
	defer raiz.Close()
	lock, err := raiz.OpenFile("writer.lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		os.Exit(1)
	}
	defer lock.Close()
	fd := lock.Fd()
	if fd > uintptr(math.MaxInt) {
		os.Exit(1)
	}
	fdInt := int(fd) // #nosec G115 -- checked against math.MaxInt above; syscall descriptor from this fixture only.
	for {
		if syscall.Flock(fdInt, syscall.LOCK_EX) != nil {
			os.Exit(1)
		}
		_, cerrado := raiz.Stat("closed")
		switch modo {
		case "close":
			if raiz.WriteFile("closed", []byte("closed"), 0600) != nil {
				os.Exit(1)
			}
		case "drain", "check":
			if cerrado != nil {
				os.Exit(1)
			}
		case "open":
			if cerrado == nil && raiz.Remove("closed") != nil {
				os.Exit(1)
			}
		case "writer":
			if os.IsNotExist(cerrado) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				cmd := exec.CommandContext(ctx, "/usr/lib/postgresql/18/bin/psql", "--no-password", "--no-psqlrc", "--dbname=fixture", "--command=INSERT INTO prueba.eventos DEFAULT VALUES")
				cmd.Env = []string{"PGHOST=/var/run/postgresql", "PGUSER=postgres", "LANG=C"}
				err = cmd.Run()
				cancel()
				if err != nil {
					os.Exit(1)
				}
			}
		default:
			os.Exit(1)
		}
		if syscall.Flock(fdInt, syscall.LOCK_UN) != nil {
			os.Exit(1)
		}
		if modo != "writer" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}
