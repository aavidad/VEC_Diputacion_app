//go:build ignore && linux && amd64

// Supervisor probatorio M38; no forma parte del producto.
// El runner lo compila por fichero para no incorporarlo al paquete del capturador.
package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

const (
	estadoUso   = 64
	estadoFallo = 65

	// Números ABI de Linux/amd64. La fuente no admite otra arquitectura.
	sysPidfdSendSignal = uintptr(424)
	sysPidfdOpen       = uintptr(434)

	// PIDFD_SIGNAL_PROCESS_GROUP está disponible desde Linux 6.9.
	pidfdSignalProcessGroup = uintptr(1 << 2)
	banderaPidfdDesconocida = uintptr(1 << 31)
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "--autoprueba" {
		os.Exit(estadoUso)
	}
	if err := autoprobarABI(); err != nil {
		fmt.Fprintf(os.Stderr, "autoprueba del supervisor: %v\n", err)
		os.Exit(estadoFallo)
	}
	fmt.Println("autoprueba=ok pidfd_grupo=disponible")
}

// autoprobarABI acredita únicamente la primitiva necesaria para C4b-2. No
// crea hijos, recursos de caso ni canales y no ejecuta trabajo operativo.
func autoprobarABI() error {
	pid := syscall.Getpid()
	if pid <= 1 {
		return fmt.Errorf("PID propio no acreditable: %d", pid)
	}
	if err := syscall.Setpgid(pid, pid); err != nil {
		return fmt.Errorf("no se pudo formar el grupo propio: %w", err)
	}
	if grupo := syscall.Getpgrp(); grupo != pid {
		return fmt.Errorf("grupo propio inesperado: pid=%d pgid=%d", pid, grupo)
	}
	grupoConsultado, err := syscall.Getpgid(pid)
	if err != nil {
		return fmt.Errorf("no se pudo consultar el grupo propio: %w", err)
	}
	if grupoConsultado != pid {
		return fmt.Errorf("consulta del grupo propio discrepante: %d", grupoConsultado)
	}

	pidfd, err := abrirPidfd(pid)
	if err != nil {
		return fmt.Errorf("pidfd_open sobre el proceso propio: %w", err)
	}
	if pidfd < 0 {
		return fmt.Errorf("pidfd_open devolvió un descriptor negativo: %d", pidfd)
	}
	abierto := true
	defer func() {
		if abierto {
			_ = syscall.Close(pidfd)
		}
	}()

	if err := enviarSenalPidfd(pidfd, 0, pidfdSignalProcessGroup); err != nil {
		return fmt.Errorf("señal cero al grupo propio mediante pidfd: %w", err)
	}

	banderasMutantes := pidfdSignalProcessGroup | banderaPidfdDesconocida
	err = enviarSenalPidfd(pidfd, 0, banderasMutantes)
	if !errors.Is(err, syscall.EINVAL) {
		return fmt.Errorf("banderas adversas: se esperaba EINVAL y se obtuvo %v", err)
	}
	if err := enviarSenalPidfd(pidfd, 0, pidfdSignalProcessGroup); err != nil {
		return fmt.Errorf("el mutante alteró el pidfd válido: %w", err)
	}

	if err := syscall.Close(pidfd); err != nil {
		return fmt.Errorf("cierre del pidfd propio: %w", err)
	}
	abierto = false
	err = enviarSenalPidfd(pidfd, 0, pidfdSignalProcessGroup)
	if !errors.Is(err, syscall.EBADF) {
		return fmt.Errorf("pidfd cerrado: se esperaba EBADF y se obtuvo %v", err)
	}
	return nil
}

// abrirPidfd invoca pidfd_open(2) de forma cruda porque syscall no expone un
// envoltorio estable y el contrato fija explícitamente Linux/amd64.
func abrirPidfd(pid int) (int, error) {
	descriptor, _, errno := syscall.Syscall(
		sysPidfdOpen,
		uintptr(pid),
		0,
		0,
	)
	if errno != 0 {
		return -1, errno
	}
	maximoEntero := uintptr(^uint(0) >> 1)
	if descriptor > maximoEntero {
		return -1, errors.New("pidfd_open devolvió un descriptor fuera de rango")
	}
	return int(descriptor), nil
}

// enviarSenalPidfd invoca pidfd_send_signal(2). Una señal cero no altera el
// grupo: solo acredita que el kernel acepta la autoridad pidfd y su bandera.
func enviarSenalPidfd(pidfd int, senal syscall.Signal, banderas uintptr) error {
	_, _, errno := syscall.Syscall6(
		sysPidfdSendSignal,
		uintptr(pidfd),
		uintptr(senal),
		0,
		banderas,
		0,
		0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}
