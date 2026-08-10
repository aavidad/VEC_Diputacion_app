//go:build ignore && linux && amd64

package main

import (
	"bytes"
	"errors"
	"os"
	"runtime"
	"strconv"
)

const camposStatO3bM38 = 52

var errIdentidadO3bM38 = errors.New("identidad O3b no acreditada")

type identidadProcesoO3bM38 struct {
	autoridad            *autoridadCapturaO3bM38
	pid, ppid, pgid, sid int
	inicio               uint64
}

type muestraStatO3bM38 struct {
	pid, ppid, pgid, sid int
	estado               byte
	inicio               uint64
}

func decimalCanonicoO3bM38(campo []byte, positivo bool) (uint64, bool) {
	if len(campo) == 0 || campo[0] == '+' || campo[0] == '-' || len(campo) > 1 && campo[0] == '0' {
		return 0, false
	}
	for _, b := range campo {
		if b < '0' || b > '9' {
			return 0, false
		}
	}
	valor, err := strconv.ParseUint(string(campo), 10, 64)
	return valor, err == nil && (!positivo || valor > 0)
}

func enteroProcesoO3bM38(campo []byte) (int, bool) {
	valor, valido := decimalCanonicoO3bM38(campo, true)
	maximo := uint64(^uint(0) >> 1)
	return int(valor), valido && valor <= maximo
}

func enteroCanonicoO3bM38(campo []byte) bool {
	if len(campo) == 0 || campo[0] == '+' || campo[0] == '-' && (len(campo) == 1 || campo[1] == '0') ||
		campo[0] != '-' && len(campo) > 1 && campo[0] == '0' {
		return false
	}
	digitos := campo
	if campo[0] == '-' {
		digitos = campo[1:]
	}
	for _, b := range digitos {
		if b < '0' || b > '9' {
			return false
		}
	}
	_, err := strconv.ParseInt(string(campo), 10, 64)
	return err == nil
}

func parsearStatO3bM38(datos []byte) (muestraStatO3bM38, error) {
	if len(datos) == 0 || len(datos) > maximoStatO3bM38 || datos[len(datos)-1] != '\n' || bytes.IndexByte(datos[:len(datos)-1], '\n') >= 0 {
		return muestraStatO3bM38{}, errIdentidadO3bM38
	}
	cierre := bytes.LastIndex(datos, []byte(") "))
	if cierre < 3 || cierre+4 >= len(datos) || datos[cierre+3] != ' ' {
		return muestraStatO3bM38{}, errIdentidadO3bM38
	}
	prefijo := datos[:cierre]
	apertura := bytes.Index(prefijo, []byte(" ("))
	if apertura < 1 || apertura+2 > cierre {
		return muestraStatO3bM38{}, errIdentidadO3bM38
	}
	pid, valido := enteroProcesoO3bM38(prefijo[:apertura])
	if !valido {
		return muestraStatO3bM38{}, errIdentidadO3bM38
	}
	restoCrudo := datos[cierre+4 : len(datos)-1]
	if len(restoCrudo) == 0 || bytes.Contains(restoCrudo, []byte("  ")) || bytes.IndexAny(restoCrudo, "\t\r\v\f") >= 0 {
		return muestraStatO3bM38{}, errIdentidadO3bM38
	}
	resto := bytes.Split(restoCrudo, []byte{' '})
	// pid, comm y estado preceden a los 49 campos numéricos restantes.
	if len(resto) != camposStatO3bM38-3 {
		return muestraStatO3bM38{}, errIdentidadO3bM38
	}
	// Linux define como signed únicamente estos campos de stat. El resto debe
	// ser decimal uint64 canónico; así no quedan tokens extra sin convertir.
	firmados := map[int]bool{3: true, 4: true, 12: true, 13: true, 14: true, 15: true, 16: true, 17: true, 20: true, 34: true, 35: true, 40: true, 48: true}
	for i, campo := range resto {
		if firmados[i] {
			if !enteroCanonicoO3bM38(campo) {
				return muestraStatO3bM38{}, errIdentidadO3bM38
			}
		} else if _, valido := decimalCanonicoO3bM38(campo, false); !valido {
			return muestraStatO3bM38{}, errIdentidadO3bM38
		}
	}
	ppid, okPPID := enteroProcesoO3bM38(resto[0])
	pgid, okPGID := enteroProcesoO3bM38(resto[1])
	sid, okSID := enteroProcesoO3bM38(resto[2])
	inicio, okInicio := decimalCanonicoO3bM38(resto[18], true)
	if !okPPID || !okPGID || !okSID || !okInicio {
		return muestraStatO3bM38{}, errIdentidadO3bM38
	}
	return muestraStatO3bM38{pid: pid, estado: datos[cierre+2], ppid: ppid, pgid: pgid, sid: sid, inicio: inicio}, nil
}

func muestraIdentidadEsperadaO3bM38(a *autoridadCapturaO3bM38, muestra muestraStatO3bM38, inicio uint64) bool {
	return a != nil && a.custodia != nil && a.custodia.cmd != nil && a.custodia.cmd.Process != nil &&
		a.sidSupervisor > 0 && muestra.pid == a.custodia.cmd.Process.Pid && muestra.estado == 'T' &&
		muestra.ppid == os.Getpid() && muestra.pgid == a.custodia.cmd.Process.Pid &&
		muestra.sid == a.sidSupervisor && muestra.inicio > 0 && (inicio == 0 || muestra.inicio == inicio)
}

func precedenciaIdentidadO3bM38(a *autoridadCapturaO3bM38) error {
	if err := leerControlO3bM38(a.custodia); err != nil {
		return err
	}
	if causa, err := revalidarAutoridadBarreraO3bM38(a.custodia); err != nil || causa != 0 {
		if err != nil {
			return err
		}
		return errIdentidadO3bM38
	}
	causa, fatal, err := acreditarPidfdBarreraO3bM38(a.custodia)
	if fatal || errors.Is(err, errLeaseBarreraO3bM38) {
		fatalBarreraO3bM38(a)
		select {}
	}
	if err != nil || causa != 0 {
		return errIdentidadO3bM38
	}
	return nil
}

func acreditarIdentidadO3bM38(a *autoridadCapturaO3bM38) (*identidadProcesoO3bM38, error) {
	if a == nil || a.estado != capturaB3StopObservadoM38 || a.custodia == nil {
		return nil, retirarBarreraO3bM38(a, barreraO3bPidfdM38)
	}
	if err := precedenciaIdentidadO3bM38(a); err != nil {
		return nil, resolverFalloOperacionBarreraO3bM38(a, err, barreraO3bPidfdM38)
	}
	primeraRaw, err := leerStatStopO3bM38(a.custodia)
	if err != nil {
		return nil, resolverFalloOperacionBarreraO3bM38(a, err, barreraO3bPidfdM38)
	}
	primera, err := parsearStatO3bM38(primeraRaw)
	if err != nil || !muestraIdentidadEsperadaO3bM38(a, primera, 0) {
		return nil, retirarBarreraO3bM38(a, barreraO3bPidfdM38)
	}
	runtime.Gosched()
	if err := precedenciaIdentidadO3bM38(a); err != nil {
		return nil, resolverFalloOperacionBarreraO3bM38(a, err, barreraO3bPidfdM38)
	}
	segundaRaw, err := leerStatStopO3bM38(a.custodia)
	if err != nil {
		return nil, resolverFalloOperacionBarreraO3bM38(a, err, barreraO3bPidfdM38)
	}
	segunda, err := parsearStatO3bM38(segundaRaw)
	if err != nil || !muestraIdentidadEsperadaO3bM38(a, segunda, primera.inicio) ||
		segunda.pid != primera.pid || segunda.ppid != primera.ppid || segunda.pgid != primera.pgid || segunda.sid != primera.sid {
		return nil, retirarBarreraO3bM38(a, barreraO3bPidfdM38)
	}
	if err := precedenciaIdentidadO3bM38(a); err != nil {
		return nil, resolverFalloOperacionBarreraO3bM38(a, err, barreraO3bPidfdM38)
	}
	if !transicionCapturaO3bM38(a.estado, capturaB4IdentidadAcreditadaM38) {
		fatalBarreraO3bM38(a)
		select {}
	}
	a.estado = capturaB4IdentidadAcreditadaM38
	return &identidadProcesoO3bM38{autoridad: a, pid: primera.pid, ppid: primera.ppid, pgid: primera.pgid, sid: primera.sid, inicio: primera.inicio}, nil
}
