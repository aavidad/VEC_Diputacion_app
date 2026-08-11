//go:build ignore && linux && amd64

package main

import (
	"syscall"
	"time"
)

const (
	duracionCasoO3cM38     = 180 * time.Second
	pidfdSignalGrupoO3cM38 = uintptr(1 << 2)
)

// tiempoMonotonoO3cM38 distingue una marca producida por time.Now de una
// reconstruida externamente: Round(0) elimina deliberadamente el componente
// monotónico de time.Time.
func tiempoMonotonoO3cM38(marca time.Time) bool {
	return !marca.IsZero() && marca != marca.Round(0)
}

func finCasoExactoO3cM38(ahora time.Time) (time.Time, bool) {
	if !tiempoMonotonoO3cM38(ahora) {
		return time.Time{}, false
	}
	fin := ahora.Add(duracionCasoO3cM38)
	return fin, tiempoMonotonoO3cM38(fin) && fin.After(ahora) && fin.Sub(ahora) == duracionCasoO3cM38
}

func autoridadContValidaO3cM38(r *revalidacionO3cM38) bool {
	if r == nil || r.auto != r || r.autoridad == nil || !r.autoridad.es(continuacionC1RevalidadoM38) ||
		r.autoridad.custodia == nil || r.autoridad.salida == nil || r.autoridad.salida.auto != r.autoridad.salida ||
		r.autoridad.salida.ahoraCaso != (time.Time{}) || r.autoridad.salida.finCaso != (time.Time{}) ||
		r.autoridad.salida.retornoCont != 0 || r.autoridad.salida.primera.Load() != 0 ||
		!r.autoridad.autoridad.poseeO3c() {
		return false
	}
	c := r.autoridad.custodia
	return permisoContMemoriaValidoO3cM38(c.lease, r.permiso, c.pidfdPrimario)
}

func permisoContMemoriaValidoO3cM38(l *leaseGuardiaO3aM38, p permisoGuardiaO3aM38, primario int) bool {
	return l != nil && l.auto == l && l.registro != nil && l.registro.auto == l.registro &&
		l.registro.leases[l] == l.generacion && l.tid == l.registro.tid && l.estado.Load() == 2 &&
		p.lease == l && p.generacion == l.secuencia && p.operacion == l.operacion &&
		p.cardinalidad == l.cardinal && p.objetivos == l.objetivos && p.estadoPrevio == 3 &&
		p.operacion == operacionContO3cM38 && p.cardinalidad == 1 && p.objetivos == [2]int{primario, -1}
}

func consolidarContO3cM38(l *leaseGuardiaO3aM38, p permisoGuardiaO3aM38, primario int) bool {
	return permisoContMemoriaValidoO3cM38(l, p, primario) && l.estado.CompareAndSwap(2, p.estadoPrevio)
}

func fatalContO3cM38(a *autoridadContinuacionO3cM38) {
	if a != nil {
		a.estado = continuacionCFFatalM38
	}
	fatalO3cM38()
}

// intentarContO3cM38 consume la capacidad antes del reloj. Tras la lectura
// monotónica solo hay cálculo puro, el syscall literal y su consolidación.
// errno se conserva sin interpretación: P4/O4a decidirán su significado.
func intentarContO3cM38(entrada **revalidacionO3cM38) *autoridadContinuacionO3cM38 {
	if entrada == nil || *entrada == nil {
		fatalContO3cM38(nil)
	}
	r := *entrada
	*entrada = nil
	if !autoridadContValidaO3cM38(r) {
		if r != nil {
			fatalContO3cM38(r.autoridad)
		}
		fatalContO3cM38(nil)
	}
	a := r.autoridad
	c := a.custodia
	permiso := r.permiso
	r.auto = nil

	ahoraCaso := time.Now()
	finCaso, marcaValida := finCasoExactoO3cM38(ahoraCaso)
	if !marcaValida || !ahoraCaso.Before(c.finBootstrap) {
		_ = c.lease.estado.CompareAndSwap(2, 5)
		fatalContO3cM38(a)
	}
	_, _, retornoRaw := syscall.Syscall6(sysPidfdSendSignal, uintptr(c.pidfdPrimario), uintptr(syscall.SIGCONT), 0, pidfdSignalGrupoO3cM38, 0, 0)
	if !consolidarContO3cM38(c.lease, permiso, c.pidfdPrimario) {
		fatalContO3cM38(a)
	}

	a.salida.ahoraCaso = ahoraCaso
	a.salida.finCaso = finCaso
	a.salida.retornoCont = int(retornoRaw)
	if !transicionContinuacionO3cM38(a.estado, continuacionC2ContIntentadoM38) {
		fatalContO3cM38(a)
	}
	a.estado = continuacionC2ContIntentadoM38
	return a
}
