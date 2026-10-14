package application

import (
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func aplicarEfectosSaldoC3(base ports.ConsultaSaldo, fuente ports.FuenteSaldo, zona *time.Location) (ports.ConsultaSaldo, error) {
	e := fuente.EfectosC3
	if e == nil {
		return base, nil
	}
	if e.SnapshotRef == "" || !domain.HuellaEfectosValida(e.FuenteSHA256) || e.CorteUTC.IsZero() || e.CorteUTC.Location() != time.UTC ||
		!e.CoberturaPermisos || e.Permisos == nil || e.Programaciones == nil || e.Politicas == nil || len(e.Permisos) > 1000 || len(e.Programaciones) > 367 || len(e.Politicas) > 30 {
		return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
	}
	base.Resumen.SnapshotRef = e.SnapshotRef
	programaciones := make(map[string]domain.ProgramacionSaldoEfectosC3, len(e.Programaciones))
	for _, p := range e.Programaciones {
		if p.Fecha == "" || programaciones[p.Fecha].Referencia != "" {
			return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
		}
		programaciones[p.Fecha] = p
	}
	permisos := make(map[string][]domain.PermisoConcedidoSaldoEfectosC3)
	for _, p := range e.Permisos {
		desde, errDesde := time.Parse("2006-01-02", p.Desde)
		hasta, errHasta := time.Parse("2006-01-02", p.Hasta)
		if errDesde != nil || errHasta != nil || p.Desde > p.Hasta || hasta.Sub(desde) > 366*24*time.Hour || p.Hasta < base.Periodo.Desde || p.Desde > base.Periodo.Hasta {
			return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
		}
		if p.Desde < base.Periodo.Desde {
			desde, _ = time.Parse("2006-01-02", base.Periodo.Desde)
		}
		if p.Hasta > base.Periodo.Hasta {
			hasta, _ = time.Parse("2006-01-02", base.Periodo.Hasta)
		}
		for dia := desde; !dia.After(hasta); dia = dia.AddDate(0, 0, 1) {
			fecha := dia.Format("2006-01-02")
			permiso := p
			permiso.ColectivoRef = programaciones[fecha].ColectivoRef
			permiso.CentroRef = programaciones[fecha].CentroRef
			permisos[fecha] = append(permisos[fecha], permiso)
		}
	}
	trabajadoExacto := map[string]int64{}
	for _, m := range fuente.MovimientosSaldo {
		if m.Tipo == "trabajado" {
			trabajadoExacto[m.Fecha] += m.DeltaMicrosegundos
		}
	}
	var totalBase, totalCredito, totalAjustado int64
	todosDeterminados := true
	for i := range base.Detalle {
		dia := &base.Detalle[i]
		if dia.SaldoMinutos != nil {
			v := *dia.SaldoMinutos
			dia.SaldoBaseMinutos = &v
			totalBase += v
		}
		concedidos := permisos[dia.Fecha]
		if len(concedidos) == 0 {
			if dia.SaldoMinutos != nil {
				cero := int64(0)
				dia.CreditoPermisoMinutos = &cero
				totalAjustado += *dia.SaldoMinutos
			} else {
				todosDeterminados = false
			}
			continue
		}
		if dia.SaldoMinutos == nil {
			dia.CausaEfecto = "fuente_no_determinable"
			todosDeterminados = false
			continue
		}
		programada, ok := programaciones[dia.Fecha]
		if !ok || programada.Referencia != dia.ProgramacionRef || programada.Version != dia.ProgramacionVersion || programada.FuenteRef != dia.FuenteRef || dia.PrevistosMinutos == nil || programada.Minutos != *dia.PrevistosMinutos {
			dia.CausaEfecto = "programacion_no_determinable"
			dia.SaldoMinutos = nil
			dia.Estado = ports.EstadoSaldoNoDisponible
			todosDeterminados = false
			continue
		}
		politica, ok := politicaAplicableSaldoC3(e.Politicas, concedidos, dia.Fecha, e.CorteUTC)
		if !ok {
			dia.CausaEfecto = "regla_no_disponible"
			dia.SaldoMinutos = nil
			dia.Estado = ports.EstadoSaldoNoDisponible
			todosDeterminados = false
			continue
		}
		r, err := domain.CalcularSaldoEfectosC3(domain.DiaSaldoEfectosC3{Fecha: dia.Fecha, ConocidoEn: e.CorteUTC, Programacion: &programada,
			TrabajadoMinutos: dia.TrabajadosMinutos, SinTrabajo: trabajadoExacto[dia.Fecha] == 0, Completo: dia.Estado == ports.EstadoSaldoDisponible, Permisos: concedidos}, politica, zona)
		if err != nil {
			return ports.ConsultaSaldo{}, ports.ErrDependenciaNoDisponible
		}
		dia.CausaEfecto = r.Causa
		dia.PoliticaEfectosRef = r.PoliticaRef
		dia.PoliticaEfectosVersion = r.PoliticaVersion
		dia.PoliticaEfectosSHA256 = r.PoliticaSHA256
		dia.ReglaEfectoRef = r.ReglaRef
		dia.ResolucionPermisoRef = r.ResolucionRef
		dia.ResolucionPermisoVersion = r.ResolucionVersion
		dia.FuenteEfectoRef = r.FuenteRef
		if r.SaldoAjustadoMinutos == nil || r.CreditoPermisoMinutos == nil {
			dia.SaldoMinutos = nil
			dia.Estado = ports.EstadoSaldoNoDisponible
			todosDeterminados = false
			continue
		}
		dia.CreditoPermisoMinutos = r.CreditoPermisoMinutos
		dia.SaldoMinutos = r.SaldoAjustadoMinutos
		dia.MovimientosEfecto = []ports.MovimientoEfectoC3{{Tipo: "credito_permiso", DeltaMinutos: *r.CreditoPermisoMinutos,
			SolicitudRef: r.SolicitudRef, ResolucionRef: r.ResolucionRef, ResolucionVersion: r.ResolucionVersion,
			PoliticaRef: r.PoliticaRef, PoliticaVersion: r.PoliticaVersion, PoliticaSHA256: r.PoliticaSHA256,
			ReglaRef: r.ReglaRef, FuenteRef: r.FuenteRef, CatalogoVersionRef: concedidos[0].CatalogoVersionRef,
			CatalogoFuenteRef: concedidos[0].CatalogoFuenteRef, ResolucionDecisionRef: concedidos[0].ResolucionDecisionRef,
			ResolucionAuditoriaRef: concedidos[0].ResolucionAuditoriaRef, ResueltaEn: concedidos[0].ResueltaEn,
			PendienteJustificar: concedidos[0].PendienteJustificar}}
		totalCredito += *r.CreditoPermisoMinutos
		totalAjustado += *r.SaldoAjustadoMinutos
	}
	if base.Resumen.SaldoMinutos != nil {
		base.Resumen.SaldoBaseMinutos = &totalBase
	}
	if todosDeterminados && base.Resumen.Estado == ports.EstadoSaldoDisponible {
		base.Resumen.SaldoMinutos = &totalAjustado
		base.Resumen.CreditoPermisoMinutos = &totalCredito
	} else {
		base.Resumen.SaldoMinutos = nil
		base.Resumen.CreditoPermisoMinutos = nil
		if base.Resumen.Estado == ports.EstadoSaldoDisponible {
			base.Resumen.Estado = ports.EstadoSaldoNoDisponible
		}
	}
	return base, nil
}

func politicaAplicableSaldoC3(politicas []domain.PoliticaSaldoEfectosC3, permisos []domain.PermisoConcedidoSaldoEfectosC3, fecha string, corte time.Time) (domain.PoliticaSaldoEfectosC3, bool) {
	var elegida domain.PoliticaSaldoEfectosC3
	if len(permisos) != 1 {
		return elegida, false
	}
	c := permisos[0]
	for _, p := range politicas {
		if p.Validar() != nil || p.PublicadaEn.After(corte) {
			continue
		}
		for _, r := range p.Reglas {
			if r.PermisoRef == c.PermisoRef && r.CatalogoVersionRef == c.CatalogoVersionRef && r.CentroRef == c.CentroRef && r.ColectivoRef == c.ColectivoRef && fecha >= r.VigenteDesde && fecha < r.HastaExclusivo {
				if elegida.Version > 0 && elegida.Referencia != p.Referencia {
					return domain.PoliticaSaldoEfectosC3{}, false
				}
				if elegida.Version < p.Version {
					elegida = p
				}
			}
		}
	}
	return elegida, elegida.Version > 0
}
