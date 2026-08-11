package main

import (
	"bytes"
	"fmt"
)

func (a *analisis) validarRetirada() error {
	b := a.de("handoff.go")
	fin := a.cuerpo("finRetiradaO3cM38")
	for _, p := range []string{
		"duracionRetiradaO3cM38 = 3 * time.Second", "tiempoMonotonoO3cM38(ahora)",
		"tiempoMonotonoO3cM38(bootstrap)", "ahora.Before(bootstrap)", "ahora.Add(duracionRetiradaO3cM38)",
		"fin.After(ahora)", "bootstrap.Before(fin)", "fin = bootstrap", "tiempoMonotonoO3cM38(fin)",
	} {
		if !bytes.Contains(append(b, fin...), []byte(p)) {
			return fmt.Errorf("deadline omite %s", p)
		}
	}
	selector := a.cuerpo("seleccionarPidfdRetiradaO3cM38")
	for _, p := range []string{"if primario.integra", "primario.fd, primario.viva, true", "if reserva.integra", "reserva.fd, reserva.viva, true", "return -1, false, false"} {
		if err := exigir(selector, p, 1, "selector "+p); err != nil {
			return err
		}
	}
	pidfd := a.cuerpo("pidfdRetiradaO3cM38")
	for _, p := range []string{
		"identidadPidfdBarreraO3bM38(c, c.pidfdPrimario)", "identidadPidfdBarreraO3bM38(c, c.pidfdReserva)",
		"pidfdVivoBarreraO3bM38(c, c.pidfdPrimario)", "pidfdVivoBarreraO3bM38(c, c.pidfdReserva)",
		"seleccionarPidfdRetiradaO3cM38",
	} {
		if err := exigir(pidfd, p, 1, "pidfd retirada "+p); err != nil {
			return err
		}
	}
	kill := a.cuerpo("enviarKillRetiradaO3cM38")
	llamada := "syscall.Syscall6(sysPidfdSendSignal, uintptr(fd), uintptr(syscall.SIGKILL), 0, 0, 0, 0)"
	if err := exigir(kill, llamada, 1, "SIGKILL individual flag0"); err != nil {
		return err
	}
	drenaje := a.cuerpo("drenarAdoptadosO3cM38")
	for _, p := range []string{"syscall.Wait4(-1, &estado, syscall.WNOHANG, nil)", "errors.Is(err, syscall.ECHILD)", "pid == 0"} {
		if err := exigir(drenaje, p, 1, "drenaje "+p); err != nil {
			return err
		}
	}
	grupo := a.cuerpo("grupoAusenteO3cM38")
	if err := exigir(grupo, "errors.Is(err, syscall.ESRCH)", 1, "grupo ESRCH"); err != nil {
		return err
	}
	cierres := a.cuerpo("cerrarRecursosRetiradaO3cM38")
	for _, p := range []string{
		"cerrarPidfdConLeaseO3aM38(c.lease, c.pidfdPrimario)", "c.pidfdPrimario = -1",
		"cerrarPidfdConLeaseO3aM38(c.lease, c.pidfdReserva)", "c.pidfdReserva = -1",
		"cerrarUnoConLeaseO3aM38(c.lease, c.controlFD", "c.controlFD = nil",
		"cerrarUnoConLeaseO3aM38(c.lease, c.terminal", "c.terminal = nil",
	} {
		if err := exigir(cierres, p, 1, "cierres "+p); err != nil {
			return err
		}
	}
	liberar := a.cuerpo("liberarRetiradaO3cM38")
	if err := enOrden(liberar, []string{"c.observador.liberar()", "c.lease.liberar()", "ownerObservador.CompareAndSwap", "ownerLease.CompareAndSwap"}, "observador→lease→owners liberados"); err != nil {
		return err
	}
	principal := a.cuerpo("retirarAntesContO3cM38")
	for _, p := range []string{"finRetiradaO3cM38(time.Now(), a.custodia.finBootstrap)", "vivo && enviarKillRetiradaO3cM38", "esperarTerminalRetiradaO3cM38", "esperarConLeaseO3aM38", "drenarAdoptadosO3cM38", "grupoAusenteO3cM38", "cerrarRecursosRetiradaO3cM38", "inventarioLiberadoO3cM38", "liberarRetiradaO3cM38", "continuacionC8RetiradoM38"} {
		if !bytes.Contains(principal, []byte(p)) {
			return fmt.Errorf("principal omite %s", p)
		}
	}
	if err := enOrden(principal, []string{"esperarTerminalRetiradaO3cM38", "esperarConLeaseO3aM38", "drenarAdoptadosO3cM38", "grupoAusenteO3cM38", "cerrarRecursosRetiradaO3cM38", "inventarioLiberadoO3cM38", "liberarRetiradaO3cM38", "continuacionC8RetiradoM38"}, "terminal→Wait→ECHILD→ESRCH→cierres→inventario→liberar→C8"); err != nil {
		return err
	}
	if err := exigir(principal, "return errRetiradaO3cM38", 1, "fin cerrado C8"); err != nil {
		return err
	}
	if bytes.Count(principal, []byte("esperarConLeaseO3aM38(")) != 1 || bytes.Count(principal, []byte("finRetiradaO3cM38(time.Now(), a.custodia.finBootstrap)")) != 1 || bytes.Contains(principal, []byte("syscall.Write")) || bytes.Contains(principal, []byte(".Write(")) {
		return fmt.Errorf("Wait no único o escritura TERMINAL")
	}
	return nil
}
