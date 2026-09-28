package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type repositorioLecturaRetornosPrueba struct {
	*repositorioSituacionPrueba
	lecturas int
}

func (r *repositorioLecturaRetornosPrueba) ListarReincorporacionesTitular(context.Context, string, string, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]ports.ReincorporacionTitularFicha, error) {
	r.lecturas++
	return nil, nil
}

type autorizadorLecturaRetornosPrueba struct {
	accion, recurso, finalidad string
	llamadas                   int
}

func (a *autorizadorLecturaRetornosPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, solicitud dominiovec.SolicitudAutorizacionLigadaV3, _ dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	datos, err := solicitud.Datos()
	if err == nil {
		a.accion, a.recurso, a.finalidad = datos.Accion, datos.Recurso.Referencia, datos.Finalidad
	}
	return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, dominiovec.ErrAutorizacionDenegada
}

func TestConsultaReincorporacionTitularDeniegaAjenoYRevocadoSinLeer(t *testing.T) {
	ahora := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	base := solicitudSituacionPrueba(t, ahora)
	q := ports.SolicitudConsultarReincorporacionesTitular{Vinculo: base.Vinculo, ResultadoContexto: base.ResultadoContexto,
		BolsaRef: base.BolsaRef, ParticipacionRef: base.ParticipacionRef, Correlacion: base.Correlacion,
		MotivoAutorizacion: base.MotivoAutorizacion}
	repo := &repositorioLecturaRetornosPrueba{repositorioSituacionPrueba: &repositorioSituacionPrueba{pertenece: false}}
	autorizador := &autorizadorLecturaRetornosPrueba{}
	servicio, _ := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, autorizador, repo, func() time.Time { return ahora })
	if _, err := servicio.ListarReincorporacionesTitular(context.Background(), q); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) ||
		autorizador.llamadas != 0 || repo.lecturas != 0 {
		t.Fatalf("ajeno: err=%v autorizaciones=%d lecturas=%d", err, autorizador.llamadas, repo.lecturas)
	}
	repo.pertenece = true
	if _, err := servicio.ListarReincorporacionesTitular(context.Background(), q); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) ||
		autorizador.llamadas != 1 || repo.lecturas != 0 || autorizador.accion != ports.AccionConsultarReincorporacionTitular ||
		autorizador.finalidad != ports.FinalidadConsultarReincorporacionTitular || autorizador.recurso != q.ParticipacionRef {
		t.Fatalf("revocado: err=%v auth=%+v lecturas=%d", err, autorizador, repo.lecturas)
	}
}

type fuenteRetornosPrueba struct {
	eventos []ports.PublicacionReincorporacionTitularCT
	err     error
}

type fuenteFijaRetornosPrueba struct {
	evento ports.PublicacionReincorporacionTitularCT
}

func (f fuenteFijaRetornosPrueba) LeerReincorporacionesTitular(context.Context, *ports.CursorContratosParticipacion, int) ([]ports.PublicacionReincorporacionTitularCT, error) {
	return []ports.PublicacionReincorporacionTitularCT{f.evento}, nil
}

func (f fuenteRetornosPrueba) LeerReincorporacionesTitular(_ context.Context, desde *ports.CursorContratosParticipacion, limite int) ([]ports.PublicacionReincorporacionTitularCT, error) {
	if f.err != nil {
		return nil, f.err
	}
	var salida []ports.PublicacionReincorporacionTitularCT
	for _, e := range f.eventos {
		if desde == nil || e.OrigenPosicion > desde.Posicion ||
			(e.OrigenPosicion == desde.Posicion && e.OrigenRef > desde.OrigenRef) {
			salida = append(salida, e)
		}
		if len(salida) == limite {
			break
		}
	}
	return salida, nil
}

type buzonRetornosPrueba struct {
	cursor    ports.CursorContratosParticipacion
	hay       bool
	recibidos []ports.PublicacionReincorporacionTitularCT
	fallarEn  string
}

func (b *buzonRetornosPrueba) CursorReincorporacionesTitular(context.Context) (ports.CursorContratosParticipacion, bool, error) {
	return b.cursor, b.hay, nil
}

func (b *buzonRetornosPrueba) RegistrarReincorporacionTitular(_ context.Context, e ports.PublicacionReincorporacionTitularCT) (ports.ResultadoReincorporacionTitularCT, error) {
	if e.EventoRef == b.fallarEn {
		return ports.ResultadoReincorporacionTitularCT{}, ports.ErrReincorporacionTitularNoDisponible
	}
	b.recibidos = append(b.recibidos, e)
	b.cursor = ports.CursorContratosParticipacion{Posicion: e.OrigenPosicion, OrigenRef: e.OrigenRef}
	b.hay = true
	return ports.ResultadoReincorporacionTitularCT{Estado: "cese_aplicado"}, nil
}

func TestRecepcionRetornoRecuperaDesdeCursorTrasFallo(t *testing.T) {
	ref1, ref2 := "evento:ct:retorno:uno", "evento:ct:retorno:dos"
	f := fuenteRetornosPrueba{eventos: []ports.PublicacionReincorporacionTitularCT{
		{EventoRef: ref1, OrigenRef: ref1, HuellaSHA256: strings.Repeat("a", 64), OrigenPosicion: 10},
		{EventoRef: ref2, OrigenRef: ref2, HuellaSHA256: strings.Repeat("b", 64), OrigenPosicion: 11},
	}}
	b := &buzonRetornosPrueba{fallarEn: ref2}
	s, err := NuevoServicioRecepcionReincorporacionesTitular(f, b)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Entregar(context.Background(), 1)
	if !errors.Is(err, ports.ErrReincorporacionTitularNoDisponible) || r.Nuevas != 1 || len(b.recibidos) != 1 {
		t.Fatalf("primera pasada: resultado=%+v error=%v recibidos=%d", r, err, len(b.recibidos))
	}
	b.fallarEn = ""
	r, err = s.Entregar(context.Background(), 1)
	if err != nil || r.Nuevas != 1 || len(b.recibidos) != 2 || b.cursor.OrigenRef != ref2 {
		t.Fatalf("recuperación: resultado=%+v error=%v cursor=%+v", r, err, b.cursor)
	}
}

func TestRecepcionRetornoRechazaFuenteQueRetrocede(t *testing.T) {
	ref := "evento:ct:retorno:uno"
	f := fuenteFijaRetornosPrueba{evento: ports.PublicacionReincorporacionTitularCT{
		EventoRef: ref, OrigenRef: ref, HuellaSHA256: strings.Repeat("a", 64), OrigenPosicion: 10}}
	b := &buzonRetornosPrueba{cursor: ports.CursorContratosParticipacion{Posicion: 10, OrigenRef: ref}, hay: true}
	s, _ := NuevoServicioRecepcionReincorporacionesTitular(f, b)
	r, err := s.Entregar(context.Background(), 1)
	if !errors.Is(err, ports.ErrReincorporacionTitularNoDisponible) || r.Nuevas != 0 || len(b.recibidos) != 0 {
		t.Fatalf("fuente retrocedida: resultado=%+v error=%v", r, err)
	}
	if _, err := NuevoServicioRecepcionReincorporacionesTitular(nil, b); err == nil {
		t.Fatal("fuente ausente aceptada")
	}
}
