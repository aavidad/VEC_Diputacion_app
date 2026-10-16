package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type resolutorConsultaBolsaCerrada struct {
	docports.ResolutorActoSubidaLocal
	llamadas int
}

func (r *resolutorConsultaBolsaCerrada) ResolverActoSubidaLocal(context.Context,
	docports.SolicitudSubidaLocal) (docports.ActoSubidaLocal, error) {
	r.llamadas++
	return docports.ActoSubidaLocal{}, docports.ErrSubidaLocalDenegada
}

type autoridadConsultaBolsaPrueba struct {
	docports.AutoridadActoSubidaLocalV3
	llamadas int
}

func (a *autoridadConsultaBolsaPrueba) AutorizarSubidaLocal(_ context.Context, pre []byte,
	_ vec.ResultadoContextoActorRegistradoV2, id, recurso, finalidad, correlacion string,
) (docports.AutorizacionActoSubidaLocalV3, error) {
	a.llamadas++
	if len(pre) == 0 || recurso != id || finalidad != "custodiar_justificante_renuncia_bolsa" {
		return docports.AutorizacionActoSubidaLocalV3{}, docports.ErrSubidaLocalDenegada
	}
	return docports.AutorizacionActoSubidaLocalV3{
		Accion:  docports.AccionBolsaJustificanteRenunciaSubirLocal,
		CargaID: id, RecursoRef: recurso, Finalidad: finalidad, CorrelacionRef: correlacion}, nil
}

type registroConsultaBolsaPrueba struct {
	docports.RegistroSubidaLocal
	instantanea docports.InstantaneaSubidaLocal
	llamadas    int
}

func (r *registroConsultaBolsaPrueba) ConsultarPropia(_ context.Context,
	q docports.ConsultaPersistenciaSubidaLocal) (docports.InstantaneaSubidaLocal, error) {
	r.llamadas++
	if q.Accion != docports.AccionBolsaJustificanteRenunciaSubirLocal ||
		q.Fase != docports.FaseSubidaLocalConsultarCarga || q.CargaID == "" || q.DocumentoID != "" ||
		q.AutorizacionV3.CargaID != q.CargaID {
		return docports.InstantaneaSubidaLocal{}, docports.ErrSubidaLocalDenegada
	}
	r.instantanea.Resultado.CargaID = q.CargaID
	return r.instantanea, nil
}

type correlacionConsultaBolsaPrueba struct{ ref string }

func (c correlacionConsultaBolsaPrueba) CorrelacionSubida(context.Context, string, string) (string, error) {
	return c.ref, nil
}

func TestConsultarSubidaBolsaRecuperaCargaTrasCerrarLlamamiento(t *testing.T) {
	actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC), "per_0123456789abcdefghijkl",
		"prf_0123456789abcdefghijkl", vec.AuthMethodCertificate, vec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &resolutorConsultaBolsaCerrada{}
	autoridad := &autoridadConsultaBolsaPrueba{}
	registro := &registroConsultaBolsaPrueba{instantanea: docports.InstantaneaSubidaLocal{
		Resultado: docports.ResultadoSubidaLocal{Estado: docports.EstadoSubidaLocalAdmitida,
			ComprobanteRef: "ref:" + strings.Repeat("a", 64), DocumentoVersion: 1,
			ReciboRef: "ref:" + strings.Repeat("b", 64), RegistradoEn: time.Now().UTC()},
		PersonaRef: actor.Contexto.PersonaRef,
		RecursoRef: "justificante-renuncia:" + strings.Repeat("c", 64),
	}}
	s := &ServicioSubidaLocal{c: ConfiguracionServicioSubidaLocal{
		ModuloID: "bolsa", TipoRecurso: "justificante_renuncia",
		Finalidad:       "custodiar_justificante_renuncia_bolsa",
		AccionSubir:     docports.AccionBolsaJustificanteRenunciaSubirLocal,
		AccionDescargar: docports.AccionBolsaJustificanteRenunciaDescargarLocal},
		d: DependenciasServicioSubidaLocal{Resolutor: resolver, Autoridad: autoridad,
			Registro: registro, Correlaciones: correlacionConsultaBolsaPrueba{
				ref: "correlacion_" + strings.Repeat("d", 32)}}}
	q := docports.SolicitudConsultaLocal{ActorRegistradoV2: actor, VinculoV2: vinculo,
		ClaveIdempotencia: "renuncia-cerrada"}
	resultado, err := s.Consultar(context.Background(), q)
	if err != nil || resultado.Estado != docports.EstadoSubidaLocalAdmitida ||
		resultado.ComprobanteRef != registro.instantanea.Resultado.ComprobanteRef ||
		resolver.llamadas != 0 || autoridad.llamadas != 1 || registro.llamadas != 1 {
		t.Fatalf("recuperación histórica: resultado=%+v error=%v resoluciones=%d autorizaciones=%d lecturas=%d",
			resultado, err, resolver.llamadas, autoridad.llamadas, registro.llamadas)
	}
	registro.instantanea.PersonaRef = "per_abcdefghijkl0123456789"
	if _, err := s.Consultar(context.Background(), q); !errors.Is(err, docports.ErrSubidaLocalDenegada) {
		t.Fatalf("se devolvió carga de otro titular: %v", err)
	}
	registro.instantanea.PersonaRef = actor.Contexto.PersonaRef
	registro.instantanea.RecursoRef = "hecho:" + strings.Repeat("c", 64)
	if _, err := s.Consultar(context.Background(), q); !errors.Is(err, docports.ErrSubidaLocalDenegada) {
		t.Fatalf("se devolvió recurso de otro módulo: %v", err)
	}
	registro.instantanea.RecursoRef = "justificante-renuncia:" + strings.Repeat("c", 64)
	registro.instantanea.ModuloID = "meritos"
	registro.instantanea.TipoRecurso = "hecho"
	registro.instantanea.Finalidad = "custodiar_evidencia_meritos"
	if _, err := s.Consultar(context.Background(), q); !errors.Is(err, docports.ErrSubidaLocalDenegada) {
		t.Fatalf("se devolvió postimagen de Méritos: %v", err)
	}
}

type resolutorSubidaErrorTecnico struct{ err error }

func (r resolutorSubidaErrorTecnico) ResolverActoSubidaLocal(context.Context, docports.SolicitudSubidaLocal) (docports.ActoSubidaLocal, error) {
	return docports.ActoSubidaLocal{}, r.err
}
func TestSubidaLocalConservaErrorTecnicoDelDueno(t *testing.T) {
	for _, causa := range []error{docports.ErrSubidaLocalNoDisponible, context.DeadlineExceeded} {
		s := &ServicioSubidaLocal{d: DependenciasServicioSubidaLocal{Resolutor: resolutorSubidaErrorTecnico{causa}}}
		_, err := s.acto(context.Background(), docports.SolicitudSubidaLocal{}, vec.ResultadoContextoActorRegistradoV2{})
		if !errors.Is(err, causa) || errors.Is(err, docports.ErrSubidaLocalDenegada) {
			t.Fatalf("error tecnico convertido en denegacion: %v", err)
		}
	}
}
