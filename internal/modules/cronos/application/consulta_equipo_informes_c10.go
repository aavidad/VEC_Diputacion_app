package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	AudienciaPresenciaEquipo   = "vec_cronos_v1.presencia_equipo.consultar.v1"
	AudienciaIncidenciasEquipo = "vec_cronos_v1.incidencias_equipo.consultar.v1"
	AccionPresenciaEquipo      = "cronos.presencia.equipo.consultar"
	AccionIncidenciasEquipo    = "cronos.incidencias.equipo.consultar"
	FinalidadPresenciaEquipo   = "consultar_presencia_equipo"
	FinalidadIncidenciasEquipo = "consultar_incidencias_equipo"
)

func ContratoConsultaEquipo(vista domain.TipoConsultaEquipo, alcance ports.AlcanceConsultaEquipo) (accion, finalidad, audiencia string, ok bool) {
	switch {
	case vista == domain.ConsultaPresenciaEquipo && (alcance == ports.AlcanceEquipoJefatura || alcance == ports.AlcanceEquipoRRHH):
		return AccionPresenciaEquipo, FinalidadPresenciaEquipo, AudienciaPresenciaEquipo, true
	case vista == domain.ConsultaIncidenciasEquipo && (alcance == ports.AlcanceEquipoJefatura || alcance == ports.AlcanceEquipoRRHH):
		return AccionIncidenciasEquipo, FinalidadIncidenciasEquipo, AudienciaIncidenciasEquipo, true
	default:
		return "", "", "", false
	}
}

type ServicioConsultaEquipo struct {
	repo  ports.RepositorioConsultaEquipo
	reloj ports.Reloj
	zona  *time.Location
}

func NuevoServicioConsultaEquipo(repo ports.RepositorioConsultaEquipo, reloj ports.Reloj, zona *time.Location) (*ServicioConsultaEquipo, error) {
	if repo == nil || reloj == nil || zona == nil || (zona.String() != domain.ZonaSaldoPeninsula && zona.String() != domain.ZonaSaldoCanarias) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ServicioConsultaEquipo{repo, reloj, zona}, nil
}

func (s *ServicioConsultaEquipo) ConsultarEquipo(ctx context.Context, o ports.OrdenConsultaEquipo, vista domain.TipoConsultaEquipo, equipo, desde, hasta, cursor string) (ports.ResultadoConsultaEquipo, error) {
	if s == nil || s.repo == nil || s.reloj == nil || s.zona == nil || ctx == nil || ctx.Err() != nil {
		return ports.ResultadoConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := o.ContextoActor()
	if err != nil {
		return ports.ResultadoConsultaEquipo{}, err
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	ahora := s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if err != nil || len(empleados) != 1 || ahora.IsZero() || !actor.Instantanea.VigenteEn(ahora) {
		return ports.ResultadoConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	if _, _, _, ok := ContratoConsultaEquipo(vista, o.Alcance()); !ok {
		return ports.ResultadoConsultaEquipo{}, ports.ErrConsultaEquipoInvalida
	}
	d, err := time.Parse("2006-01-02", desde)
	if err != nil || d.Format("2006-01-02") != desde {
		return ports.ResultadoConsultaEquipo{}, ports.ErrConsultaEquipoInvalida
	}
	h, err := time.Parse("2006-01-02", hasta)
	if err != nil || h.Format("2006-01-02") != hasta || h.Before(d) || h.Sub(d) > 31*24*time.Hour {
		return ports.ResultadoConsultaEquipo{}, ports.ErrConsultaEquipoInvalida
	}
	if len(equipo) == 0 || len(equipo) > 128 || len(cursor) > 256 {
		return ports.ResultadoConsultaEquipo{}, ports.ErrConsultaEquipoInvalida
	}
	foto, err := s.repo.FotoEquipo(ctx, equipo)
	if err != nil {
		return ports.ResultadoConsultaEquipo{}, err
	}
	if foto.EquipoRef != equipo || foto.Desde > desde || foto.HastaExclusivo <= hasta {
		return ports.ResultadoConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	m := domain.MaterialConsultaEquipo{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleados[0], Foto: foto, Tipo: vista, Desde: desde, Hasta: hasta, ZonaHoraria: s.zona.String(), Alcance: domain.AlcanceConsultaEquipo(o.Alcance()), Cursor: cursor}
	if _, err := m.Canonico(); err != nil {
		return ports.ResultadoConsultaEquipo{}, ports.ErrConsultaEquipoInvalida
	}
	fuente, err := s.repo.ConsultarEquipo(ctx, o, m)
	if err != nil {
		return ports.ResultadoConsultaEquipo{}, err
	}
	if fuente.EquipoRef != equipo || fuente.Alcance != o.Alcance() || fuente.Desde != desde || fuente.Hasta != hasta || fuente.Vista != vista || fuente.VersionRef != foto.VersionRef || fuente.ZonaHoraria != s.zona.String() || fuente.CorteUTC.IsZero() || fuente.CorteUTC.Location() != time.UTC || fuente.CorteUTC.After(ahora.Add(time.Minute)) || fuente.Filas == nil || len(fuente.Filas) > 100 || len(fuente.SiguienteCursor) > 256 {
		return ports.ResultadoConsultaEquipo{}, ports.ErrDependenciaNoDisponible
	}
	resultado := ports.ResultadoConsultaEquipo{EquipoRef: equipo, Alcance: o.Alcance(), Desde: desde, Hasta: hasta, Vista: vista, VersionRef: foto.VersionRef, Filas: make([]ports.FilaConsultaEquipo, 0, len(fuente.Filas)), SiguienteCursor: fuente.SiguienteCursor}
	ultimo := ""
	for _, f := range fuente.Filas {
		clave := f.EmpleadoRef + "|" + f.Fecha
		if f.EmpleadoRef == "" || f.EmpleadoEtiqueta == "" || len([]rune(f.EmpleadoEtiqueta)) > 120 || f.Fecha < desde || f.Fecha > hasta || (ultimo != "" && clave <= ultimo) || (cursor != "" && clave <= cursor) || f.Marcajes == nil || len(f.Marcajes) > 100 {
			return ports.ResultadoConsultaEquipo{}, ports.ErrDependenciaNoDisponible
		}
		ultimo = clave
		fecha, err := time.ParseInLocation("2006-01-02", f.Fecha, s.zona)
		if err != nil {
			return ports.ResultadoConsultaEquipo{}, ports.ErrDependenciaNoDisponible
		}
		inicio, corte := fecha.UTC(), fecha.AddDate(0, 0, 1).UTC()
		if fuente.CorteUTC.Before(corte) {
			corte = fuente.CorteUTC
		}
		if corte.Before(inicio) {
			corte = inicio
		}
		hechos := make([]domain.HechoSaldo, 0, len(f.Marcajes))
		for _, m := range f.Marcajes {
			hechos = append(hechos, domain.HechoSaldo{Movimiento: m.Movimiento, InstanteUTC: m.InstanteUTC.UTC()})
		}
		fila := ports.FilaConsultaEquipo{EmpleadoRef: f.EmpleadoRef, EmpleadoEtiqueta: f.EmpleadoEtiqueta, Fecha: f.Fecha, HechosRegistrados: int64(len(hechos)), Cobertura: domain.CoberturaRegistroIncompleta}
		if f.CoberturaCompleta {
			fila.Cobertura = domain.CoberturaRegistroCompleta
		}
		if vista == domain.ConsultaPresenciaEquipo {
			evaluacion, err := domain.EvaluarPresenciaAlCorte(hechos, inicio, corte, f.CoberturaCompleta)
			if err != nil {
				return ports.ResultadoConsultaEquipo{}, ports.ErrDependenciaNoDisponible
			}
			fila.Estado, fila.Causa = string(evaluacion.Estado), string(evaluacion.Causa)
		} else {
			marcajes := make([]domain.HechoRegistroIncidencia, 0, len(f.Marcajes))
			for _, m := range f.Marcajes {
				marcajes = append(marcajes, domain.HechoRegistroIncidencia{Ref: m.Ref, Version: m.Version, Movimiento: m.Movimiento, InstanteUTC: m.InstanteUTC.UTC()})
			}
			dias, err := domain.ConsultarIncidenciasPeriodo(domain.ConsultaIncidenciasPeriodo{Desde: f.Fecha, Hasta: fecha.AddDate(0, 0, 1).Format("2006-01-02"), Zona: s.zona.String(), CorteUTC: fuente.CorteUTC, AntecedentesCompletos: f.AntecedentesCompletos, Cobertura: []domain.CoberturaRegistro{{Fecha: f.Fecha, Estado: fila.Cobertura}}, Hechos: marcajes})
			if err != nil || len(dias) != 1 {
				return ports.ResultadoConsultaEquipo{}, ports.ErrDependenciaNoDisponible
			}
			fila.Estado = dias[0].Estado
			if dias[0].HechosRegistrados != nil {
				fila.HechosRegistrados = int64(*dias[0].HechosRegistrados)
			}
		}
		resultado.Filas = append(resultado.Filas, fila)
		resultado.Agregado.Total++
		if fila.Estado == string(domain.PresenciaIndeterminada) || fila.Estado == "indeterminado" || fila.Estado == "no_evaluado" {
			resultado.Agregado.Indeterminado++
		}
		if fila.Estado == string(domain.PresenciaRegistrada) || fila.Estado == "registrado" {
			resultado.Agregado.Registrado++
		}
		if fila.Estado == "registro_incompleto" {
			resultado.Agregado.Incidencias++
		}
	}
	return resultado, nil
}

var _ ports.CasoUsoConsultaEquipo = (*ServicioConsultaEquipo)(nil)
