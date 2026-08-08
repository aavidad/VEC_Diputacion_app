//go:build ignore && linux && amd64

// Supervisor probatorio M38; no forma parte del producto.
package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	estadoUso               = 64
	estadoFallo             = 65
	sysPidfdSendSignal      = uintptr(424) // Linux/amd64.
	sysPidfdOpen            = uintptr(434)
	pidfdSignalProcessGroup = uintptr(1 << 2)
	banderaPidfdDesconocida = uintptr(1 << 31)
	tiempoEspera            = 3 * time.Second
)

type proceso struct {
	cmd      *exec.Cmd
	pidfd    int
	grupo    bool
	esperado bool
}

type recursos struct {
	procesos  []*proceso
	adoptados []int
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--autoprueba" {
		if err := autoprobar(); err != nil {
			fmt.Fprintf(os.Stderr, "autoprueba del supervisor: %v\n", err)
			os.Exit(estadoFallo)
		}
		fmt.Println("autoprueba=ok pidfd_grupo=operativo")
		return
	}
	if len(os.Args) == 4 && os.Args[1] == "--ayudante-interno" {
		fd, err := autenticar(os.Args[2], os.Args[3])
		if err != nil {
			os.Exit(estadoUso)
		}
		if err = servirAyudante(os.Args[2], fd); err != nil {
			os.Exit(estadoFallo)
		}
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "--supervisar-m38" {
		os.Exit(supervisarM38())
	}
	os.Exit(estadoUso)
}

func autoprobar() (err error) {
	if err = prepararNetpoll(); err != nil {
		return err
	}
	inicial, err := contarFD()
	if err != nil || !sinHijos() {
		return errors.New("estado inicial no aislado")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err = autoprobarABI(); err != nil {
		return err
	}
	if err = autoprobarTramasM38(); err != nil {
		return err
	}
	if err = autoprobarSobreS0M38(); err != nil {
		return err
	}
	if err = autoprobarControlPreinicioM38(); err != nil {
		return err
	}
	if err = activarSubreaper(); err != nil {
		return err
	}
	r := &recursos{}
	defer func() {
		if e := r.limpiar(); err == nil && e != nil {
			err = e
		}
	}()
	for _, ignora := range []bool{false, true} {
		if err = probarGrupo(r, ignora); err != nil {
			return err
		}
	}
	if err = probarRollbackPosteriorAStart(); err != nil {
		return err
	}
	if err = probarRollbackHandoffFallido(); err != nil {
		return err
	}
	if err = probarLiderRecogido(r); err != nil {
		return err
	}
	if err = r.limpiar(); err != nil {
		return err
	}
	final, err := contarFD()
	if err != nil || final != inicial || !sinHijos() {
		return fmt.Errorf("residuos: fd=%d/%d", final, inicial)
	}
	return nil
}

func probarRollbackPosteriorAStart() (err error) {
	r := &recursos{}
	defer func() {
		if e := r.limpiar(); err == nil && e != nil {
			err = e
		}
	}()
	lider, _, err := r.iniciar("cooperar", 0, false)
	if err != nil {
		return err
	}
	if _, _, err = r.iniciar("ignorar-term", lider.cmd.Process.Pid, false); err != nil {
		return err
	}
	reserva, err := duplicarPidfd(lider.pidfd)
	if err != nil {
		return err
	}
	reservaAbierta := true
	defer func() {
		if reservaAbierta {
			_ = syscall.Close(reserva)
		}
	}()
	if err = r.limpiar(); err == nil {
		err = exigirESRCH(reserva)
	}
	errCierre := syscall.Close(reserva)
	if errCierre == nil {
		reservaAbierta = false
	} else if err == nil {
		err = errCierre
	}
	if err == nil && !sinHijos() {
		err = errors.New("el rollback posterior a Start dejó hijos")
	}
	return err
}

func probarRollbackHandoffFallido() (err error) {
	r := &recursos{}
	defer func() {
		if e := r.limpiar(); err == nil && e != nil {
			err = e
		}
	}()
	lider, canal, err := r.iniciar("lider-descendiente", 0, true)
	if err != nil {
		return err
	}
	canalAbierto := true
	defer func() {
		if canalAbierto {
			_ = syscall.Close(canal)
		}
	}()
	reserva, err := duplicarPidfd(lider.pidfd)
	if err != nil {
		return err
	}
	reservaAbierta := true
	defer func() {
		if reservaAbierta {
			_ = syscall.Close(reserva)
		}
	}()
	if recibido, fallo := recibirPidfd(canal, true); fallo == nil {
		_ = syscall.Close(recibido)
		return errors.New("el handoff truncado fue aceptado")
	}
	if err = r.limpiar(); err == nil {
		err = exigirESRCH(reserva)
	}
	if errCierre := syscall.Close(canal); errCierre == nil {
		canalAbierto = false
	} else if err == nil {
		err = errCierre
	}
	if errCierre := syscall.Close(reserva); errCierre == nil {
		reservaAbierta = false
	} else if err == nil {
		err = errCierre
	}
	if err == nil && !sinHijos() {
		err = errors.New("el rollback del handoff dejó hijos")
	}
	return err
}

func prepararNetpoll() error {
	lector, escritor, err := os.Pipe()
	if err != nil {
		return err
	}
	errPlazo := lector.SetReadDeadline(time.Now())
	errLector := lector.Close()
	errEscritor := escritor.Close()
	if errPlazo != nil || errLector != nil || errEscritor != nil {
		return errors.New("no se pudo estabilizar el inventario de FD del runtime")
	}
	return nil
}

func probarGrupo(r *recursos, ignora bool) error {
	lider, _, err := r.iniciar("cooperar", 0, false)
	if err != nil {
		return err
	}
	modo := "cooperar"
	if ignora {
		modo = "ignorar-term"
	}
	miembro, _, err := r.iniciar(modo, lider.cmd.Process.Pid, false)
	if err != nil {
		return err
	}
	if !ignora {
		err = enviar(lider.pidfd, 0, pidfdSignalProcessGroup|banderaPidfdDesconocida)
		if !errors.Is(err, syscall.EINVAL) {
			return fmt.Errorf("flag adverso: %v", err)
		}
	}
	if err = detenerYObservar(lider, miembro); err == nil {
		if ignora {
			err = enviar(lider.pidfd, syscall.SIGTERM, pidfdSignalProcessGroup)
			if err == nil {
				err = enviar(lider.pidfd, syscall.SIGCONT, pidfdSignalProcessGroup)
			}
		} else {
			err = enviar(lider.pidfd, syscall.SIGCONT, pidfdSignalProcessGroup)
			if err == nil {
				err = enviar(lider.pidfd, syscall.SIGTERM, pidfdSignalProcessGroup)
			}
		}
	}
	if err != nil {
		return err
	}
	if err = esperarProceso(lider); err != nil {
		return err
	}
	if ignora {
		time.Sleep(50 * time.Millisecond)
		if err = enviar(lider.pidfd, 0, pidfdSignalProcessGroup); err != nil {
			return errors.New("el miembro no ignoró TERM")
		}
		err = enviar(lider.pidfd, syscall.SIGKILL, pidfdSignalProcessGroup)
	}
	if err == nil {
		err = esperarProceso(miembro)
	}
	if err == nil {
		err = exigirESRCH(lider.pidfd)
	}
	if err != nil {
		return err
	}
	if !ignora {
		cerrado := miembro.pidfd
		if err = syscall.Close(cerrado); err != nil {
			return fmt.Errorf("cierre del pidfd mutante: %w", err)
		}
		miembro.pidfd = -1
		if err = enviar(cerrado, 0, 0); !errors.Is(err, syscall.EBADF) {
			return fmt.Errorf("pidfd cerrado: %v", err)
		}
	}
	return nil
}

func probarLiderRecogido(r *recursos) error {
	senuelo, _, err := r.iniciar("ignorar-term", 0, false)
	if err != nil {
		return err
	}
	lider, canal, err := r.iniciar("lider-descendiente", 0, true)
	if err != nil {
		return err
	}
	canalAbierto := true
	defer func() {
		if canalAbierto {
			_ = syscall.Close(canal)
		}
	}()
	fdDesc, err := recibirPidfd(canal, false)
	if err != nil {
		return err
	}
	r.adoptados = append(r.adoptados, fdDesc)
	if _, err = syscall.Write(canal, []byte("SALIR")); err != nil {
		return err
	}
	if err = syscall.Close(canal); err != nil {
		return fmt.Errorf("cierre del canal de handoff: %w", err)
	}
	canalAbierto = false
	if err = esperarProceso(lider); err != nil {
		return err
	}
	if err = enviar(lider.pidfd, 0, pidfdSignalProcessGroup); err == nil {
		err = enviar(lider.pidfd, syscall.SIGKILL, pidfdSignalProcessGroup)
	}
	if err == nil {
		err = esperarTerminal(fdDesc)
	}
	if err == nil {
		err = recogerAdoptado()
	}
	if err == nil {
		err = exigirESRCH(lider.pidfd)
	}
	if err != nil {
		return err
	}
	if err = syscall.Close(fdDesc); err != nil {
		return fmt.Errorf("cierre del pidfd adoptado: %w", err)
	}
	r.adoptados[0] = -1
	if err = enviar(senuelo.pidfd, 0, pidfdSignalProcessGroup); err != nil {
		return errors.New("el señuelo fue alcanzado")
	}
	if err = enviar(senuelo.pidfd, syscall.SIGKILL, 0); err == nil {
		err = esperarProceso(senuelo)
	}
	if err == nil {
		err = exigirESRCH(senuelo.pidfd)
	}
	return err
}

func (r *recursos) iniciar(modo string, pgid int, conservarCanal bool) (*proceso, int, error) {
	nonce := make([]byte, 16)
	_, err := rand.Read(nonce)
	if err != nil {
		return nil, -1, err
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, -1, err
	}
	par, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_SEQPACKET|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, -1, err
	}
	archivo := os.NewFile(uintptr(par[1]), "capacidad-padre")
	pidfd := -1
	cmd := exec.Command(exe, "--ayudante-interno", modo, fmt.Sprintf("%x", nonce))
	cmd.ExtraFiles = []*os.File{archivo}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: pgid, PidFD: &pidfd}
	errInicio := cmd.Start()
	errArchivo := archivo.Close()
	if errInicio != nil {
		errCanal := syscall.Close(par[0])
		if errCanal != nil {
			return nil, -1, fmt.Errorf("Start: %v; cierre del canal: %w", errInicio, errCanal)
		}
		return nil, -1, errInicio
	}
	p := &proceso{cmd: cmd, pidfd: pidfd, grupo: pgid == 0}
	r.procesos = append(r.procesos, p)
	if errArchivo != nil {
		_ = syscall.Close(par[0])
		return p, -1, fmt.Errorf("cierre de la capacidad heredada: %w", errArchivo)
	}
	if pidfd < 0 {
		if err = syscall.Close(par[0]); err != nil {
			return p, -1, fmt.Errorf("Start sin pidfd y canal no cerrable: %w", err)
		}
		if err = esperarProceso(p); err != nil {
			return p, -1, err
		}
		return p, -1, errors.New("Start no entregó pidfd; ayudante recogido por EOF")
	}
	if err = syscall.SetsockoptTimeval(par[0], syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &syscall.Timeval{Sec: 3}); err != nil {
		errCierre := syscall.Close(par[0])
		if errCierre != nil {
			return p, -1, fmt.Errorf("plazo del canal: %v; cierre: %w", err, errCierre)
		}
		return p, -1, fmt.Errorf("plazo del canal interno: %w", err)
	}
	_, err = syscall.Write(par[0], []byte(fmt.Sprintf("%x|%s", nonce, modo)))
	buf := make([]byte, 16)
	var n int
	if err == nil {
		n, err = syscall.Read(par[0], buf)
	}
	if err == nil && string(buf[:n]) != "LISTO" {
		err = errors.New("ACK interno inválido")
	}
	if err != nil || !conservarCanal {
		errCierre := syscall.Close(par[0])
		if err == nil && errCierre != nil {
			err = fmt.Errorf("cierre del canal interno: %w", errCierre)
		}
		par[0] = -1
	}
	return p, par[0], err
}

func autenticar(modo, nonce string) (int, error) {
	if modo != "cooperar" && modo != "ignorar-term" && modo != "lider-descendiente" {
		return -1, errors.New("modo desconocido")
	}
	credencial, err := syscall.GetsockoptUcred(3, syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil || int(credencial.Pid) != syscall.Getppid() {
		return -1, errors.New("capacidad paterna ausente")
	}
	buf := make([]byte, 128)
	n, err := syscall.Read(3, buf)
	if err != nil || string(buf[:n]) != nonce+"|"+modo {
		return -1, errors.New("nonce inválido")
	}
	if modo == "ignorar-term" {
		signal.Ignore(syscall.SIGTERM)
	}
	_, err = syscall.Write(3, []byte("LISTO"))
	return 3, err
}

func servirAyudante(modo string, fd int) error {
	if modo != "lider-descendiente" {
		if err := syscall.Close(fd); err != nil {
			return err
		}
		for {
			time.Sleep(time.Hour)
		}
	}
	r := &recursos{}
	desc, _, err := r.iniciar("ignorar-term", syscall.Getpgrp(), false)
	if err != nil {
		return err
	}
	transferido := false
	defer func() {
		if !transferido {
			_ = r.limpiar()
		}
	}()
	if err = syscall.Sendmsg(fd, []byte("DESCENDIENTE"), syscall.UnixRights(desc.pidfd), nil, 0); err != nil {
		return err
	}
	buf := make([]byte, 16)
	n, err := syscall.Read(fd, buf)
	if err != nil || string(buf[:n]) != "SALIR" {
		return errors.New("handoff no confirmado")
	}
	transferido = true
	if err = syscall.Close(desc.pidfd); err != nil {
		return err
	}
	desc.pidfd = -1
	return syscall.Close(fd)
}

func recibirPidfd(fd int, forzarTruncado bool) (int, error) {
	tamanoDatos := 32
	if forzarTruncado {
		tamanoDatos = 1
	}
	datos := make([]byte, tamanoDatos)
	control := make([]byte, syscall.CmsgSpace(4))
	n, nc, flags, _, err := syscall.Recvmsg(fd, datos, control, syscall.MSG_CMSG_CLOEXEC)
	mensajes, errorControl := syscall.ParseSocketControlMessage(control[:nc])
	descriptores := make([]int, 0, 1)
	for i := range mensajes {
		recibidos, errorRights := syscall.ParseUnixRights(&mensajes[i])
		if errorRights != nil {
			errorControl = errorRights
			continue
		}
		descriptores = append(descriptores, recibidos...)
	}
	valido := err == nil && errorControl == nil && len(mensajes) == 1 &&
		flags&(syscall.MSG_TRUNC|syscall.MSG_CTRUNC) == 0 &&
		string(datos[:n]) == "DESCENDIENTE" && len(descriptores) == 1 && descriptores[0] >= 0
	if !valido {
		var errorCierre error
		for _, recibido := range descriptores {
			if errCierre := syscall.Close(recibido); errorCierre == nil && errCierre != nil {
				errorCierre = errCierre
			}
		}
		if errorCierre != nil {
			return -1, fmt.Errorf("handoff SCM_RIGHTS inválido; cierre: %w", errorCierre)
		}
		return -1, errors.New("handoff SCM_RIGHTS inválido y cerrado")
	}
	syscall.CloseOnExec(descriptores[0])
	return descriptores[0], nil
}

func detenerYObservar(procesos ...*proceso) error {
	if err := enviar(procesos[0].pidfd, syscall.SIGSTOP, pidfdSignalProcessGroup); err != nil {
		return err
	}
	fin := time.Now().Add(tiempoEspera)
	for time.Now().Before(fin) {
		todos := true
		for _, p := range procesos {
			datos, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", p.cmd.Process.Pid))
			i := strings.LastIndex(string(datos), ") ")
			todos = todos && err == nil && i >= 0 && i+2 < len(datos) && datos[i+2] == 'T'
		}
		if todos {
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return errors.New("STOP no observado")
}

func esperarProceso(p *proceso) error {
	if p.esperado {
		return errors.New("Wait duplicado")
	}
	var err error
	if p.pidfd >= 0 {
		err = esperarTerminal(p.pidfd)
	} else {
		err = esperarTerminalSinPidfd(p.cmd.Process.Pid)
	}
	if err != nil {
		return err
	}
	p.esperado = true
	err = p.cmd.Wait()
	if _, esperado := err.(*exec.ExitError); err != nil && !esperado {
		return err
	}
	return nil
}

func esperarTerminalSinPidfd(pid int) error {
	fin := time.Now().Add(tiempoEspera)
	for time.Now().Before(fin) {
		datos, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		i := strings.LastIndex(string(datos), ") ")
		if err == nil && i >= 0 && i+2 < len(datos) && datos[i+2] == 'Z' {
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return errors.New("ayudante sin pidfd no terminó por EOF")
}

func (r *recursos) limpiar() error {
	var primero error
	for _, p := range r.procesos {
		if !p.esperado {
			var err error
			if p.pidfd >= 0 {
				banderas := uintptr(0)
				if p.grupo {
					banderas = pidfdSignalProcessGroup
				}
				if errSenal := enviar(p.pidfd, syscall.SIGKILL, banderas); errSenal != nil &&
					!errors.Is(errSenal, syscall.ESRCH) && primero == nil {
					primero = errSenal
				}
				err = esperarProceso(p)
			} else {
				err = esperarProceso(p)
			}
			if primero == nil && err != nil {
				primero = err
			}
		}
	}
	for _, fd := range r.adoptados {
		if fd >= 0 {
			if err := enviar(fd, syscall.SIGKILL, 0); err != nil &&
				!errors.Is(err, syscall.ESRCH) && primero == nil {
				primero = err
			}
			if err := esperarTerminal(fd); primero == nil && err != nil {
				primero = err
			}
		}
	}
	if err := drenarHijos(); primero == nil && err != nil {
		primero = err
	}
	for _, p := range r.procesos {
		if p.grupo && p.pidfd >= 0 {
			if err := exigirESRCH(p.pidfd); primero == nil && err != nil {
				primero = err
			}
		}
		if p.pidfd >= 0 {
			if err := syscall.Close(p.pidfd); primero == nil && err != nil {
				primero = err
			}
			p.pidfd = -1
		}
	}
	for i, fd := range r.adoptados {
		if fd >= 0 {
			if err := syscall.Close(fd); primero == nil && err != nil {
				primero = err
			}
			r.adoptados[i] = -1
		}
	}
	r.procesos = nil
	r.adoptados = nil
	return primero
}

func drenarHijos() error {
	fin := time.Now().Add(tiempoEspera)
	for time.Now().Before(fin) {
		var estado syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &estado, syscall.WNOHANG, nil)
		if errors.Is(err, syscall.ECHILD) {
			return nil
		}
		if err != nil && !errors.Is(err, syscall.EINTR) {
			return err
		}
		if pid == 0 {
			time.Sleep(5 * time.Millisecond)
		}
	}
	return errors.New("drenaje de hijos no alcanzó ECHILD")
}

func autoprobarABI() error {
	pid := syscall.Getpid()
	if pid <= 1 {
		return errors.New("PID propio no acreditable")
	}
	if err := syscall.Setpgid(pid, pid); err != nil {
		return fmt.Errorf("grupo propio: %w", err)
	}
	if syscall.Getpgrp() != pid {
		return errors.New("grupo propio discrepante")
	}
	descriptor, _, errno := syscall.Syscall(sysPidfdOpen, uintptr(pid), 0, 0)
	if errno != 0 {
		return fmt.Errorf("pidfd_open propio: %w", errno)
	}
	pidfd := int(descriptor)
	errSenal := enviar(pidfd, 0, pidfdSignalProcessGroup)
	errCierre := syscall.Close(pidfd)
	if errSenal != nil {
		return fmt.Errorf("señal cero al grupo propio: %w", errSenal)
	}
	if errCierre != nil {
		return fmt.Errorf("cierre del pidfd propio: %w", errCierre)
	}
	return nil
}

func recogerAdoptado() error {
	var estado syscall.WaitStatus
	pid, err := syscall.Wait4(-1, &estado, syscall.WNOHANG, nil)
	if err != nil {
		return err
	}
	if pid <= 0 {
		return errors.New("descendiente no recolectado")
	}
	return nil
}

func sinHijos() bool {
	var estado syscall.WaitStatus
	_, err := syscall.Wait4(-1, &estado, syscall.WNOHANG, nil)
	return errors.Is(err, syscall.ECHILD)
}
