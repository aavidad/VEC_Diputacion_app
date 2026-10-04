package application

import (
	"context"
	"fmt"
	"reflect"
	"time"

	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/ports"
)

// ServicioPreparacionBases conserva propuestas incompletas bajo Bolsa.
// El consumidor interno debe obtener la capacidad del nucleo; este servicio
// no resuelve identidad ni convierte una propuesta en convocatoria formal.
type ServicioPreparacionBases struct {
	repositorio ports.RepositorioPreparacionBases
	reloj       core.Reloj
}

func NuevoServicioPreparacionBases(repo ports.RepositorioPreparacionBases, reloj core.Reloj) (*ServicioPreparacionBases, error) {
	if dependenciaPreparacionNula(repo) || dependenciaPreparacionNula(reloj) {
		return nil, ports.ErrPreparacionBasesNoDisponible
	}
	return &ServicioPreparacionBases{repo, reloj}, nil
}

func (s *ServicioPreparacionBases) Guardar(ctx context.Context, orden ports.GuardarPreparacionBases) (ports.ResultadoPreparacionBases, error) {
	var vacio ports.ResultadoPreparacionBases
	if err := s.validarContexto(ctx); err != nil {
		return vacio, err
	}
	// El instante de uso procede siempre del reloj del servidor.
	orden.SolicitadaEn = s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if orden.Validar() != nil {
		return vacio, ports.ErrPreparacionBasesInvalida
	}
	material, err := orden.Material.Canonico()
	if err != nil {
		return vacio, err
	}
	orden.Material = material
	intencion, err := prep.HuellaIntencion(orden.Esperada, material, orden.Ambito)
	if err != nil {
		return vacio, err
	}
	huella, err := material.HuellaSHA256()
	if err != nil {
		return vacio, err
	}
	exacta := prep.Esperada{PreparacionRef: orden.Esperada.PreparacionRef, Revision: orden.Esperada.Revision + 1, HuellaMaterialSHA256: huella}
	resultado, err := s.repositorio.GuardarPreparacionBases(ctx, orden)
	if err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if resultado.Version.Ambito != orden.Ambito || resultado.Recibo.HuellaIntencionSHA256 != intencion {
		return vacio, ports.ErrResultadoPreparacionBasesInvalido
	}
	if err := validarResultadoPreparacion(resultado, exacta, orden.Autorizacion, orden.SolicitadaEn); err != nil {
		return vacio, err
	}
	return clonarResultadoPreparacion(resultado)
}

func (s *ServicioPreparacionBases) Consultar(ctx context.Context, orden ports.ConsultarPreparacionBases) (ports.ResultadoPreparacionBases, error) {
	var vacio ports.ResultadoPreparacionBases
	if err := s.validarContexto(ctx); err != nil {
		return vacio, err
	}
	orden.SolicitadaEn = s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if orden.Validar() != nil {
		return vacio, ports.ErrPreparacionBasesInvalida
	}
	resultado, err := s.repositorio.ConsultarPreparacionBases(ctx, orden)
	if err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if resultado.Version.Ambito != orden.Ambito {
		return vacio, ports.ErrResultadoPreparacionBasesInvalido
	}
	if err := validarResultadoPreparacion(resultado, orden.Exacta, orden.Autorizacion, orden.SolicitadaEn); err != nil {
		return vacio, err
	}
	return clonarResultadoPreparacion(resultado)
}

func validarResultadoPreparacion(r ports.ResultadoPreparacionBases, exacta prep.Esperada, e core.EvidenciaUsoDecisionAutorizacion, instante time.Time) error {
	d, err := e.Datos()
	if err != nil {
		return fmt.Errorf("%w: %w", ports.ErrResultadoPreparacionBasesInvalido, err)
	}
	if err := r.Version.Validar(); err != nil {
		return fmt.Errorf("%w: %w", ports.ErrResultadoPreparacionBasesInvalido, err)
	}
	if r.Version.Estado != exacta || r.AutorizacionRef != d.Decision.DecisionRef ||
		r.HuellaAutorizacionSHA256 != d.HuellaDecisionSHA256 || !r.AccedidaEn.Equal(instante) ||
		!instanteCanonicoPreparacion(r.Recibo.ConfirmadaEn) || r.Recibo.ConfirmadaEn.After(instante) ||
		!prep.HuellaValida(r.Recibo.HuellaIntencionSHA256) {
		return ports.ErrResultadoPreparacionBasesInvalido
	}
	vistas := make(map[string]bool, 6)
	for _, referencia := range []string{r.Recibo.ReciboRef, r.Recibo.HistoriaRef, r.Recibo.AuditoriaRef, r.Recibo.EventoRef,
		r.ConsumoAutorizacionRef, r.AuditoriaAccesoRef} {
		if !prep.IdentificadorValido(referencia) || vistas[referencia] {
			return ports.ErrResultadoPreparacionBasesInvalido
		}
		vistas[referencia] = true
	}
	return nil
}

func clonarResultadoPreparacion(r ports.ResultadoPreparacionBases) (ports.ResultadoPreparacionBases, error) {
	m, err := r.Version.Material.Canonico()
	if err != nil {
		return ports.ResultadoPreparacionBases{}, ports.ErrResultadoPreparacionBasesInvalido
	}
	r.Version.Material = m
	return r, nil
}

func instanteCanonicoPreparacion(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Equal(t.Truncate(time.Microsecond))
}

func (s *ServicioPreparacionBases) validarContexto(ctx context.Context) error {
	if ctx == nil || s == nil || dependenciaPreparacionNula(s.repositorio) || dependenciaPreparacionNula(s.reloj) {
		return ports.ErrPreparacionBasesNoDisponible
	}
	return ctx.Err()
}

func dependenciaPreparacionNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}
