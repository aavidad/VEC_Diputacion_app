package application

import (
	"context"
	"strings"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func EnsayarSaldoPermisos(ctx context.Context, lector ports.LectorSnapshotSaldoPermisos, p domain.PoliticaEfectosPermisoSaldo) (ports.ResultadoEnsayoSaldoPermisos, error) {
	if ctx == nil || ctx.Err() != nil || lector == nil || p.Validar() != nil {
		return ports.ResultadoEnsayoSaldoPermisos{}, domain.ErrEfectoPermisoSaldoInvalido
	}
	s, err := lector.LeerSnapshotSaldoPermisos(ctx)
	if err != nil || ctx.Err() != nil || s.VersionEsquema != 1 || !s.Demostracion || s.PoliticaRef != p.Referencia || s.PoliticaVersion != p.Version || s.PoliticaSHA256 != p.SHA256 || !domain.HuellaEfectosValida(s.SHA256) || len(s.Escenarios) < 1 || len(s.Escenarios) > 20 {
		return ports.ResultadoEnsayoSaldoPermisos{}, domain.ErrEfectoPermisoSaldoInvalido
	}
	r := ports.ResultadoEnsayoSaldoPermisos{Demostracion: true, PoliticaRef: p.Referencia, PoliticaVersion: p.Version, PoliticaSHA256: p.SHA256, EscenariosSHA256: s.SHA256, Escenarios: []ports.ResultadoEscenarioSaldoPermisos{}}
	vistos := map[string]bool{}
	for _, e := range s.Escenarios {
		if !strings.HasPrefix(e.Referencia, "demo:") || !domain.ReferenciaEfectosValida(e.Referencia) || vistos[e.Referencia] || len(e.Dias) < 1 || len(e.Dias) > 367 {
			return ports.ResultadoEnsayoSaldoPermisos{}, domain.ErrEfectoPermisoSaldoInvalido
		}
		vistos[e.Referencia] = true
		re := ports.ResultadoEscenarioSaldoPermisos{Referencia: e.Referencia, Dias: []ports.EfectoPermisoSaldo{}}
		fechas := map[string]bool{}
		for _, d := range e.Dias {
			if fechas[d.Fecha] || ctx.Err() != nil {
				return ports.ResultadoEnsayoSaldoPermisos{}, domain.ErrEfectoPermisoSaldoInvalido
			}
			fechas[d.Fecha] = true
			hecho := domain.DiaSaldoPermisos{Fecha: d.Fecha, Programacion: (*domain.ProgramacionDiaPermisoSaldo)(d.Programacion), TrabajadosMinutos: d.TrabajadosMinutos, Completo: d.Completo, SinAnomalias: d.SinAnomalias, SinTrabajo: d.SinTrabajo}
			if d.Permisos != nil {
				hecho.Permisos = make([]domain.ConcesionPermisoSaldo, len(d.Permisos))
				for i, c := range d.Permisos {
					hecho.Permisos[i] = domain.ConcesionPermisoSaldo(c)
				}
			}
			efecto, err := domain.CalcularEfectoPermisoSaldo(hecho, p)
			if err != nil {
				return ports.ResultadoEnsayoSaldoPermisos{}, err
			}
			re.Dias = append(re.Dias, ports.EfectoPermisoSaldo(efecto))
		}
		r.Escenarios = append(r.Escenarios, re)
	}
	return r, nil
}
