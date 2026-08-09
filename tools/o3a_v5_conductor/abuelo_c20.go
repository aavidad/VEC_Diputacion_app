package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

const prSetChildSubreaper = 36

func contarFD() (int, error) {
	entradas, err := os.ReadDir("/proc/self/fd")
	return len(entradas), err
}

func activarSubreaper() error {
	_, _, errno := syscall.RawSyscall6(syscall.SYS_PRCTL, prSetChildSubreaper, 1, 0, 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func recolectarHasta(fin time.Time) (int, int, error) {
	recogidos, ultimo := 0, 0
	for time.Now().Before(fin) {
		var estado syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &estado, syscall.WNOHANG, nil)
		if errors.Is(err, syscall.ECHILD) {
			return recogidos, ultimo, nil
		}
		if err != nil {
			return recogidos, ultimo, err
		}
		if pid > 0 {
			recogidos++
			ultimo = pid
			continue
		}
		time.Sleep(time.Millisecond)
	}
	return recogidos, ultimo, errors.New("watchdog de descendientes vencido")
}

func main() {
	if len(os.Args) != 6 {
		os.Exit(64)
	}
	esperado, err := strconv.Atoi(os.Args[3])
	if err != nil {
		os.Exit(64)
	}
	minReap, err := strconv.Atoi(os.Args[4])
	if err != nil || activarSubreaper() != nil {
		os.Exit(64)
	}
	if exec.Command("/bin/true").Run() != nil {
		os.Exit(65)
	}
	inicial, err := contarFD()
	if err != nil {
		os.Exit(65)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, os.Args[1], "--autoprueba-o3a-caso", os.Args[2])
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	estado := 0
	if err != nil {
		var salida *exec.ExitError
		if !errors.As(err, &salida) {
			os.Exit(65)
		}
		estado = salida.ExitCode()
	}
	recogidos, ultimo, err := recolectarHasta(time.Now().Add(2 * time.Second))
	final, errFD := contarFD()
	grupoCero := ultimo == 0
	if ultimo > 0 {
		grupoCero = errors.Is(syscall.Kill(-ultimo, 0), syscall.ESRCH)
	}
	valido := ctx.Err() == nil && err == nil && errFD == nil && estado == esperado &&
		stdout.Len() == 0 && stderr.Len() == 0 && recogidos >= minReap && final == inicial && grupoCero
	resultado := "GO"
	if !valido {
		resultado = "NO-GO"
	}
	fmt.Printf("%s\t%s\t%d\t%d\t%d\t%d\t%d\t%t\t%s\n", os.Args[5], os.Args[2], estado, stdout.Len(), stderr.Len(), recogidos, final-inicial, grupoCero, resultado)
	if !valido {
		os.Exit(1)
	}
}
