//go:build ignore && linux && amd64

// Primitivas Linux compartidas del supervisor probatorio M38.
package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

// supervisarM38 permanece cerrado hasta que G2-O aporte el protocolo revisado.
func supervisarM38() int {
	return estadoUso
}

func duplicarPidfd(pidfd int) (int, error) {
	descriptor, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfd), uintptr(syscall.F_DUPFD_CLOEXEC), 0)
	if errno != 0 {
		return -1, errno
	}
	if descriptor > uintptr(^uint(0)>>1) {
		_ = syscall.Close(int(descriptor))
		return -1, errors.New("duplicado pidfd fuera de rango")
	}
	return int(descriptor), nil
}

func esperarTerminal(pidfd int) error {
	type pollfd struct {
		fd               int32
		eventos, retorno int16
	}
	p := pollfd{fd: int32(pidfd), eventos: 1}
	fin := time.Now().Add(tiempoEspera)
	for {
		restante := time.Until(fin)
		if restante <= 0 {
			return errors.New("plazo de terminalidad agotado")
		}
		milisegundos := restante.Milliseconds()
		if milisegundos == 0 {
			milisegundos = 1
		}
		p.retorno = 0
		n, _, errno := syscall.Syscall(syscall.SYS_POLL, uintptr(unsafe.Pointer(&p)), 1, uintptr(milisegundos))
		if errno == syscall.EINTR {
			continue
		}
		if errno != 0 || n != 1 || p.retorno&1 == 0 {
			return fmt.Errorf("terminalidad no acreditada: n=%d eventos=%x errno=%v", n, p.retorno, errno)
		}
		return nil
	}
}

func activarSubreaper() error {
	if _, _, e := syscall.Syscall6(syscall.SYS_PRCTL, 36, 1, 0, 0, 0, 0); e != 0 {
		return e
	}
	var valor int32
	_, _, e := syscall.Syscall6(syscall.SYS_PRCTL, 37, uintptr(unsafe.Pointer(&valor)), 0, 0, 0, 0)
	if e != 0 || valor != 1 {
		return errors.New("subreaper no acreditado")
	}
	return nil
}

func contarFD() (int, error) {
	entradas, err := os.ReadDir("/proc/self/fd")
	return len(entradas), err
}

func exigirESRCH(pidfd int) error {
	err := enviar(pidfd, 0, pidfdSignalProcessGroup)
	if !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("grupo terminal: se esperaba ESRCH y se obtuvo %v", err)
	}
	return nil
}

func enviar(pidfd int, senal syscall.Signal, banderas uintptr) error {
	_, _, errno := syscall.Syscall6(sysPidfdSendSignal, uintptr(pidfd), uintptr(senal), 0, banderas, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}
