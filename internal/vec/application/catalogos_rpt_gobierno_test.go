package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type preparadorGobiernoRPTPrueba struct {
	llamadas  int
	err       error
	resultado ports.PreparacionPropuestaGobiernoCategoriaRPT
}

func (p *preparadorGobiernoRPTPrueba) PrepararPropuestaGobiernoCategoriaRPT(context.Context, ports.BorradorPropuestaGobiernoCategoriaRPT) (ports.PreparacionPropuestaGobiernoCategoriaRPT, error) {
	p.llamadas++
	return p.resultado, p.err
}
func (p *preparadorGobiernoRPTPrueba) PrepararAprobacionGobiernoCategoriaRPT(context.Context, ports.MaterialAvanceGobiernoCategoriaRPT) (ports.PreparacionGobiernoCategoriaRPT, error) {
	p.llamadas++
	return ports.PreparacionGobiernoCategoriaRPT{}, p.err
}
func (p *preparadorGobiernoRPTPrueba) PrepararConfirmacionGobiernoCategoriaRPT(context.Context, ports.MaterialAvanceGobiernoCategoriaRPT) (ports.PreparacionGobiernoCategoriaRPT, error) {
	p.llamadas++
	return ports.PreparacionGobiernoCategoriaRPT{}, p.err
}

func materialAvanceGobiernoRPTPrueba() ports.MaterialAvanceGobiernoCategoriaRPT {
	return ports.MaterialAvanceGobiernoCategoriaRPT{
		PropuestaRef: "propuesta:ejemplo", HuellaSHA256: strings.Repeat("a", 64),
		ReciboRef: "recibo:ejemplo", RevisionEsperada: 1,
		CatalogoID: "catalogo.ejemplo", ModuloID: "bolsa",
	}
}

func TestServicioGobiernoRPTFallaAntesDeV3ConCASInvalidoYDependenciaCaida(t *testing.T) {
	p := &preparadorGobiernoRPTPrueba{err: errors.New("dsn privado y detalle ajeno")}
	s := &ServicioGobiernoCategoriaRPT{preparador: p}
	o := OrdenAvanzarGobiernoCategoriaRPT{Material: materialAvanceGobiernoRPTPrueba()}
	o.Material.RevisionEsperada = 3
	if _, err := s.Aprobar(t.Context(), o); !errors.Is(err, ErrOrdenGobiernoCategoriaRPTInvalida) || p.llamadas != 0 {
		t.Fatalf("CAS invalido: error=%v preparaciones=%d", err, p.llamadas)
	}
	o.Material.RevisionEsperada = 1
	if _, err := s.Aprobar(t.Context(), o); !errors.Is(err, ports.ErrGobiernoCategoriaRPTNoDisponible) || strings.Contains(err.Error(), "dsn privado") || p.llamadas != 1 {
		t.Fatalf("dependencia caida: error=%v preparaciones=%d", err, p.llamadas)
	}
	p.err = ports.ErrGobiernoCategoriaRPTDenegado
	if _, err := s.Aprobar(t.Context(), o); !errors.Is(err, ports.ErrGobiernoCategoriaRPTDenegado) || p.llamadas != 2 {
		t.Fatalf("denegacion: error=%v preparaciones=%d", err, p.llamadas)
	}
}

func TestServicioGobiernoRPTRechazaMaterialPreparadoConContenidoAjeno(t *testing.T) {
	id, revision, h := "categoria.ejemplo", int64(7), strings.Repeat("a", 64)
	motivo := domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos", CatalogoVersion: 1,
		CatalogoHuellaSHA256: h, EntradaClave: "motivo_0123456789abcdef0123456789abcdef",
	}
	b := ports.BorradorPropuestaGobiernoCategoriaRPT{
		PropuestaRef: "propuesta:ejemplo", ReciboRef: "recibo:ejemplo",
		Contenido: domain.ContenidoGobiernoCategoriaRPT{
			Accion:     domain.AccionGobiernoCategoriaRPTDeshabilitar,
			CatalogoID: "catalogo.ejemplo", ModuloID: "bolsa", Version: 2,
			CategoriaID: &id, RevisionEsperada: &revision,
			PreimagenesControl: map[string]domain.PreimagenControlGobiernoCategoriaRPT{
				id: {Version: 2, Revision: revision, HuellaSHA256: h, Estado: "habilitada"},
			},
			MotivoRef: motivo.Referencia(), FuenteRef: "resolucion-2026-99",
		},
	}
	preparado := ports.MaterialPropuestaGobiernoCategoriaRPT{
		PropuestaRef: b.PropuestaRef, ReciboRef: b.ReciboRef,
		Contenido: b.Contenido, HuellaSHA256: h,
	}
	preparado.Contenido.PreimagenesHuellaSHA256 = h
	preparado.Contenido.FuenteRef = "resolucion-ajena"
	p := &preparadorGobiernoRPTPrueba{resultado: ports.PreparacionPropuestaGobiernoCategoriaRPT{Material: preparado}}
	s := &ServicioGobiernoCategoriaRPT{preparador: p}
	_, err := s.Proponer(t.Context(), OrdenProponerGobiernoCategoriaRPT{
		Credenciales: CredencialesGobiernoCategoriaRPT{Motivo: motivo}, Borrador: b,
	})
	if !errors.Is(err, ports.ErrGobiernoCategoriaRPTNoConfiable) || p.llamadas != 1 {
		t.Fatalf("contenido preparado ajeno: error=%v preparaciones=%d", err, p.llamadas)
	}
}

func TestResultadoGobiernoRPTConservaReciboEnReplayConDecisionNueva(t *testing.T) {
	fecha := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	h := strings.Repeat("a", 64)
	for _, decision := range []string{"decision-primera", "decision-replay"} {
		resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(
			decision, h, h, "contexto-1", h,
			ports.AccionAprobarGobiernoCategoriaRPT, "propuesta:ejemplo", h,
			ports.AudienciaGobiernoCategoriaRPT, fecha, fecha.Add(time.Second),
		)
		if err != nil {
			t.Fatal(err)
		}
		r := ports.ResultadoGobiernoCategoriaRPT{
			PropuestaRef: "propuesta:ejemplo", HuellaSHA256: h,
			Revision: 2, Estado: domain.EstadoGobiernoCategoriaRPTUnaAprobacion,
			ReciboRef: "recibo:ejemplo",
			Evidencia: ports.EvidenciaGobiernoCategoriaRPT{
				DecisionRef: decision, EfectoRef: "propuesta:ejemplo", HuellaEfectoSHA256: h,
				ConsumoHuellaSHA256: h, AuditoriaRef: "auditoria-1", ConsumidaEn: fecha, ConsumoNuevo: true,
			},
		}
		if !resultadoGobiernoCategoriaRPTValido(r, r.PropuestaRef, h, r.ReciboRef, 2, r.Estado, resumen) {
			t.Fatalf("replay con decision nueva rechazado: %s", decision)
		}
		r.Evidencia.HuellaEfectoSHA256 = strings.Repeat("b", 64)
		if resultadoGobiernoCategoriaRPTValido(r, r.PropuestaRef, h, r.ReciboRef, 2, r.Estado, resumen) {
			t.Fatal("huella de recurso ajena aceptada")
		}
	}
}
