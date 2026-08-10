//go:build ignore && linux && amd64

package main

import (
	"bytes"
	"errors"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

const (
	pidfdSignalGrupoO3bM38 = uintptr(1 << 2)
	maximoStatO3bM38       = 4096
)

var errStopO3bM38 = errors.New("auto-STOP O3b no acreditado")

func finStopO3bM38(ahora, finBootstrap time.Time) time.Time {
	fin := ahora.Add(time.Second)
	if finBootstrap.Before(fin) {
		return finBootstrap
	}
	return fin
}

func sondaGrupoCeroO3bM38(c *custodiaO3aM38, pidfd int) error {
	return operarConLeaseBarreraO3bM38(c, func() error {
		_, _, errno := syscall.Syscall6(sysPidfdSendSignal, uintptr(pidfd), 0, 0, pidfdSignalGrupoO3bM38, 0, 0)
		if errno != 0 {
			return errno
		}
		return nil
	})
}

func leerStatStopO3bM38(c *custodiaO3aM38) ([]byte, error) {
	pid := c.cmd.Process.Pid
	ruta := "/proc/" + strconv.Itoa(pid) + "/stat"
	fd := -1
	err := operarConLeaseBarreraO3bM38(c, func() error {
		var fallo error
		fd, fallo = syscall.Open(ruta, syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
		return fallo
	})
	if err != nil {
		return nil, err
	}
	buffer := make([]byte, maximoStatO3bM38+1)
	n := 0
	errLectura := operarConLeaseBarreraO3bM38(c, func() error {
		var fallo error
		n, fallo = syscall.Read(fd, buffer)
		return fallo
	})
	errCierre := operarConLeaseBarreraO3bM38(c, func() error { return syscall.Close(fd) })
	if errors.Is(errCierre, errLeaseBarreraO3bM38) {
		return nil, errCierre
	}
	if errLectura != nil {
		return nil, errLectura
	}
	if errCierre != nil {
		return nil, errCierre
	}
	if n == 0 || n > maximoStatO3bM38 {
		return nil, errStopO3bM38
	}
	return buffer[:n], nil
}

func muestraTStopO3bM38(datos []byte) bool {
	// P4 solo acredita T en dos muestras. El parser y la identidad
	// de campos exactos pertenecen exclusivamente a P5.
	for i := len(datos) - 3; i >= 0; i-- {
		if datos[i] == ')' && datos[i+1] == ' ' {
			return datos[i+2] == 'T' && i+3 < len(datos) && datos[i+3] == ' '
		}
	}
	return false
}

func precedenciaStopO3bM38(a *autoridadCapturaO3bM38) error {
	if err := leerControlO3bM38(a.custodia); err != nil {
		return err
	}
	if causa, err := revalidarAutoridadBarreraO3bM38(a.custodia); err != nil || causa != 0 {
		if err != nil {
			return err
		}
		return errStopO3bM38
	}
	if causa, fatal, err := acreditarPidfdBarreraO3bM38(a.custodia); err != nil || causa != 0 {
		if fatal || errors.Is(err, errLeaseBarreraO3bM38) {
			fatalBarreraO3bM38(a)
			select {}
		}
		if err != nil {
			return err
		}
		return errStopO3bM38
	}
	return sondaGrupoCeroO3bM38(a.custodia, a.custodia.pidfdPrimario)
}

func observarStopO3bM38(a *autoridadCapturaO3bM38) error {
	if a == nil || a.estado != capturaB2TicketCerradoM38 || a.custodia == nil || a.custodia.cmd == nil ||
		a.custodia.cmd.Process == nil || a.custodia.cmd.Process.Pid <= 0 {
		return retirarBarreraO3bM38(a, barreraO3bPidfdM38)
	}
	fin := finStopO3bM38(time.Now(), a.custodia.finBootstrap)
	if !time.Now().Before(fin) {
		return retirarBarreraO3bM38(a, barreraO3bBootstrapM38)
	}
	// El Bash recibe el EOF de ticket de forma asíncrona. P4 no observa un
	// estado intermedio: cede un quantum acotado y su primera evidencia /proc
	// ya es contractual; cualquier valor distinto de T retira.
	time.Sleep(100 * time.Millisecond)
	if !time.Now().Before(fin) {
		return retirarBarreraO3bM38(a, barreraO3bBootstrapM38)
	}
	if err := precedenciaStopO3bM38(a); err != nil {
		return resolverFalloOperacionBarreraO3bM38(a, err, barreraO3bPidfdM38)
	}
	primera, err := leerStatStopO3bM38(a.custodia)
	if err != nil {
		return resolverFalloOperacionBarreraO3bM38(a, err, barreraO3bPidfdM38)
	}
	if !muestraTStopO3bM38(primera) {
		return retirarBarreraO3bM38(a, barreraO3bPidfdM38)
	}
	runtime.Gosched()
	if !time.Now().Before(fin) {
		return retirarBarreraO3bM38(a, barreraO3bBootstrapM38)
	}
	if err := precedenciaStopO3bM38(a); err != nil {
		return resolverFalloOperacionBarreraO3bM38(a, err, barreraO3bPidfdM38)
	}
	segunda, err := leerStatStopO3bM38(a.custodia)
	if err != nil {
		return resolverFalloOperacionBarreraO3bM38(a, err, barreraO3bPidfdM38)
	}
	if !time.Now().Before(fin) {
		return retirarBarreraO3bM38(a, barreraO3bBootstrapM38)
	}
	if !muestraTStopO3bM38(segunda) || !bytes.Equal(primera, segunda) {
		return retirarBarreraO3bM38(a, barreraO3bPidfdM38)
	}
	if !transicionCapturaO3bM38(a.estado, capturaB3StopObservadoM38) {
		fatalBarreraO3bM38(a)
		select {}
	}
	a.estado = capturaB3StopObservadoM38
	return nil
}
