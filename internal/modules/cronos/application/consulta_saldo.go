package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type ServicioConsultaSaldo struct {
	repositorio ports.RepositorioConsultaSaldo
	reloj       ports.Reloj
	zona        *time.Location
}

func NuevoServicioConsultaSaldo(repo ports.RepositorioConsultaSaldo, reloj ports.Reloj, zona *time.Location) (*ServicioConsultaSaldo, error) {
	if repo == nil || reloj == nil || zona == nil {
		return nil, ErrServiceDependencyRequired
	}
	return &ServicioConsultaSaldo{repo, reloj, zona}, nil
}

func (s *ServicioConsultaSaldo) ConsultarSaldo(ctx context.Context, orden ports.OrdenConsultaSaldo, tipo ports.PeriodoSaldo, desdeArg, hastaArg string) (ports.ConsultaSaldo, error) {
	if s == nil || s.repositorio == nil || s.reloj == nil || s.zona == nil || ctx == nil {
		return ports.ConsultaSaldo{}, ports.ErrConsultaSaldoInvalida
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return ports.ConsultaSaldo{}, err
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 {
		return ports.ConsultaSaldo{}, ports.ErrConsultaSaldoInvalida
	}
	ahora := s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() || !actor.Instantanea.VigenteEn(ahora) {
		return ports.ConsultaSaldo{}, ports.ErrConsultaSaldoInvalida
	}
	desde, hasta, err := resolverPeriodoSaldo(tipo, desdeArg, hastaArg, ahora.In(s.zona), s.zona)
	if err != nil {
		return ports.ConsultaSaldo{}, err
	}
	desdeTexto, hastaTexto := desde.Format("2006-01-02"), hasta.Format("2006-01-02")
	fuente, err := s.repositorio.ConsultarFuenteSaldo(ctx, orden, empleados[0], desdeTexto, hastaTexto, s.zona.String())
	if err != nil {
		return ports.ConsultaSaldo{}, err
	}
	if fuente.EmpleadoRef != empleados[0] || fuente.Desde != desdeTexto || fuente.Hasta != hastaTexto || fuente.ZonaHoraria != s.zona.String() || len(fuente.Marcajes) > 10000 || len(fuente.Jornadas) > 367 || len(fuente.MovimientosSaldo) > 20000 {
		return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
	}
	return construirConsultaSaldo(tipo, desde, hasta, fuente, s.zona)
}

func resolverPeriodoSaldo(tipo ports.PeriodoSaldo, desdeArg, hastaArg string, ahora time.Time, zona *time.Location) (time.Time, time.Time, error) {
	hoy := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, zona)
	if tipo != ports.PeriodoSaldoRango && (desdeArg != "" || hastaArg != "") {
		return time.Time{}, time.Time{}, ports.ErrConsultaSaldoInvalida
	}
	switch tipo {
	case ports.PeriodoSaldoHoy:
		return hoy, hoy, nil
	case ports.PeriodoSaldoSemana:
		diasDesdeLunes := (int(hoy.Weekday()) + 6) % 7
		lunes := hoy.AddDate(0, 0, -diasDesdeLunes)
		return lunes, lunes.AddDate(0, 0, 6), nil
	case ports.PeriodoSaldoMes:
		inicio := time.Date(hoy.Year(), hoy.Month(), 1, 0, 0, 0, 0, zona)
		return inicio, inicio.AddDate(0, 1, -1), nil
	case ports.PeriodoSaldoAnio:
		inicio := time.Date(hoy.Year(), 1, 1, 0, 0, 0, 0, zona)
		return inicio, time.Date(hoy.Year(), 12, 31, 0, 0, 0, 0, zona), nil
	case ports.PeriodoSaldoRango:
		desde, err1 := fechaCivilSaldo(desdeArg, zona)
		hasta, err2 := fechaCivilSaldo(hastaArg, zona)
		if err1 != nil || err2 != nil || hasta.Before(desde) || hasta.After(desde.AddDate(1, 0, 0)) {
			return time.Time{}, time.Time{}, ports.ErrConsultaSaldoInvalida
		}
		return desde, hasta, nil
	default:
		return time.Time{}, time.Time{}, ports.ErrConsultaSaldoInvalida
	}
}

func fechaCivilSaldo(valor string, zona *time.Location) (time.Time, error) {
	if len(valor) != 10 {
		return time.Time{}, ports.ErrConsultaSaldoInvalida
	}
	fecha, err := time.ParseInLocation("2006-01-02", valor, zona)
	if err != nil || fecha.Format("2006-01-02") != valor {
		return time.Time{}, ports.ErrConsultaSaldoInvalida
	}
	return fecha, nil
}

func construirConsultaSaldo(tipo ports.PeriodoSaldo, desde, hasta time.Time, fuente ports.FuenteSaldo, zona *time.Location) (ports.ConsultaSaldo, error) {
	jornadas := make(map[string]ports.JornadaPrevista, len(fuente.Jornadas))
	for _, j := range fuente.Jornadas {
		fecha, err := fechaCivilSaldo(j.Fecha, zona)
		if err != nil || fecha.Before(desde) || fecha.After(hasta) || j.MinutosPrevistos < 0 || j.MinutosPrevistos > 24*60 || j.PoliticaVersionRef == "" {
			return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
		}
		if _, existe := jornadas[j.Fecha]; existe {
			return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
		}
		jornadas[j.Fecha] = j
	}
	hechos := make([]domain.HechoSaldo, 0, len(fuente.Marcajes))
	for _, m := range fuente.Marcajes {
		if m.MarcajeRef == "" || m.OrigenRef == "" || m.Canal.CanalRef == "" || m.Canal.OrigenRef != m.OrigenRef || m.Canal.PoliticaVersionRef == "" || m.Canal.CalidadRef == "" || m.InstanteUTC.Location() != time.UTC {
			return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
		}
		if m.TipoOrigen != nil && *m.TipoOrigen != "terminal" && *m.TipoOrigen != "remoto" {
			return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
		}
		hechos = append(hechos, domain.HechoSaldo{Movimiento: m.Movimiento, InstanteUTC: m.InstanteUTC})
	}
	type asientosDia struct {
		trabajado, previsto int64
		previstoExiste      bool
	}
	libro := make(map[string]asientosDia)
	for _, mov := range fuente.MovimientosSaldo {
		fecha, err := fechaCivilSaldo(mov.Fecha, zona)
		if err != nil || fecha.Before(desde) || fecha.After(hasta) || len(mov.Fuentes) == 0 {
			return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
		}
		for _, ref := range mov.Fuentes {
			if ref == "" {
				return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
			}
		}
		a := libro[mov.Fecha]
		switch mov.Tipo {
		case "trabajado":
			if mov.DeltaMicrosegundos <= 0 {
				return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
			}
			a.trabajado += mov.DeltaMicrosegundos
		case "previsto":
			if mov.DeltaMicrosegundos > 0 || a.previstoExiste {
				return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
			}
			a.previsto = mov.DeltaMicrosegundos
			a.previstoExiste = true
		default:
			return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
		}
		libro[mov.Fecha] = a
	}
	finExclusivoUTC := hasta.AddDate(0, 0, 1).UTC()
	tiempos, err := domain.CalcularTiempoSaldo(hechos, zona, desde.UTC(), finExclusivoUTC)
	if err != nil {
		return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
	}
	resultado := ports.ConsultaSaldo{Periodo: ports.PeriodoConsultaSaldo{Tipo: tipo, Desde: desde.Format("2006-01-02"), Hasta: hasta.Format("2006-01-02")}, Detalle: make([]ports.DetalleSaldoDia, 0, 366)}
	var totalPrevisto, totalTrabajado, totalSaldo int64
	completo := true
	incompleto := false
	for fecha := desde; !fecha.After(hasta); fecha = fecha.AddDate(0, 0, 1) {
		clave := fecha.Format("2006-01-02")
		tiempo := tiempos[clave]
		asientos := libro[clave]
		if asientos.trabajado != int64(tiempo.Trabajado/time.Microsecond) {
			return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
		}
		trabajado := domain.MinutosCompletos(tiempo.Trabajado)
		dia := ports.DetalleSaldoDia{Fecha: clave, TrabajadosMinutos: trabajado, PausasMinutos: domain.MinutosCompletos(tiempo.Pausa), Estado: ports.EstadoSaldoDisponible, Marcajes: make([]ports.MarcajeDia, 0)}
		if j, ok := jornadas[clave]; ok {
			if !asientos.previstoExiste || asientos.previsto != -j.MinutosPrevistos*60*1000000 {
				return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
			}
			previsto := j.MinutosPrevistos
			dia.TurnoRef, dia.PoliticaVersionRef = j.TurnoRef, j.PoliticaVersionRef
			dia.PrevistosMinutos = &previsto
			totalPrevisto += previsto
		} else {
			if asientos.previstoExiste {
				return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
			}
			completo = false
			dia.Estado = ports.EstadoSaldoNoDisponible
		}
		if tiempo.Incompleto {
			incompleto = true
			dia.Estado = ports.EstadoSaldoIncompleto
		}
		if dia.PrevistosMinutos != nil && !tiempo.Incompleto {
			saldo := minutosSaldoLibro(asientos.trabajado + asientos.previsto)
			dia.SaldoMinutos = &saldo
			totalSaldo += saldo
		}
		for _, m := range fuente.Marcajes {
			if m.InstanteUTC.In(zona).Format("2006-01-02") == clave {
				dia.Marcajes = append(dia.Marcajes, ports.MarcajeDia{InstanteUTC: m.InstanteUTC, Movimiento: m.Movimiento, Origen: m.TipoOrigen})
			}
		}
		resultado.Detalle = append(resultado.Detalle, dia)
		totalTrabajado += trabajado
	}
	resultado.Resumen = ports.ResumenConsultaSaldo{TrabajadosMinutos: totalTrabajado, Estado: ports.EstadoSaldoDisponible}
	if !fuente.Completo && completo {
		incompleto = true
		for i := range resultado.Detalle {
			resultado.Detalle[i].Estado = ports.EstadoSaldoIncompleto
			resultado.Detalle[i].SaldoMinutos = nil
		}
	}
	if completo {
		resultado.Resumen.PrevistosMinutos = &totalPrevisto
		saldo := totalSaldo
		resultado.Resumen.SaldoMinutos = &saldo
	} else {
		resultado.Resumen.Estado = ports.EstadoSaldoNoDisponible
	}
	if incompleto {
		resultado.Resumen.Estado = ports.EstadoSaldoIncompleto
		resultado.Resumen.SaldoMinutos = nil
	}
	return resultado, nil
}

func minutosSaldoLibro(microsegundos int64) int64 {
	const minuto = int64(60 * 1000000)
	minutos := microsegundos / minuto
	if microsegundos < 0 && microsegundos%minuto != 0 {
		minutos--
	}
	return minutos
}
