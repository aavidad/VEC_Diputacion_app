package application

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type singularPorCantidadPrueba struct{ llamadas int }

func (s *singularPorCantidadPrueba) SeleccionarYLlamarParaAdaptador(context.Context, SolicitudSeleccionLlamamiento) (DatosReciboSeleccionLlamamientoParaAdaptador, error) {
	s.llamadas++
	return DatosReciboSeleccionLlamamientoParaAdaptador{ReciboRef: "recibo:singular:prueba"}, nil
}

type ofertaPorCantidadPrueba struct{ seleccion, recuperacion int }

func (o *ofertaPorCantidadPrueba) SeleccionarOfertaParaAdaptador(context.Context, SolicitudSeleccionLlamamiento) (DatosReciboSeleccionOfertaParaAdaptador, error) {
	o.seleccion++
	return DatosReciboSeleccionOfertaParaAdaptador{Estado: "confirmado"}, nil
}

func (o *ofertaPorCantidadPrueba) RecuperarOfertaParaAdaptador(context.Context, SolicitudSeleccionLlamamiento) (DatosReciboSeleccionLlamamientoParaAdaptador, error) {
	o.recuperacion++
	return DatosReciboSeleccionLlamamientoParaAdaptador{Oferta: &DatosReciboSeleccionOfertaParaAdaptador{Estado: "confirmado"}}, nil
}

func TestSeleccionPorCantidadRecuperaVersionHistoricaSinIniciarNueva(t *testing.T) {
	contenido, err := os.ReadFile("../domain/testdata/expediente_solicitud_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var e domain.Expediente
	if err := json.Unmarshal(contenido, &e); err != nil {
		t.Fatal(err)
	}
	e.Solicitud.NumeroPersonas = 2
	e.Solicitud.JornadaMinutos = 1500
	e.Solicitud.PuestoSolicitado = "9"
	if err := e.Validar(); err != nil {
		t.Fatal(err)
	}
	q := SolicitudSeleccionLlamamiento{ExpedienteRef: e.Referencia, VersionEsperada: e.Version,
		ClaveIdempotencia: "018f47a2-6b31-4c80-8a95-4d2e707c5a11", RecuperarOferta: true}
	singular, oferta := &singularPorCantidadPrueba{}, &ofertaPorCantidadPrueba{}
	r, err := SeleccionarSegunNumeroPersonasCT(t.Context(), ports.ExpedienteParaSeleccion{
		Fiscalizado: e, VersionActual: e.Version + 1,
	}, q, singular, oferta)
	if err != nil || r.Oferta == nil || oferta.recuperacion != 1 || oferta.seleccion != 0 || singular.llamadas != 0 {
		t.Fatalf("recuperación histórica cambió de operación: r=%+v err=%v singular=%d selección=%d recuperación=%d",
			r, err, singular.llamadas, oferta.seleccion, oferta.recuperacion)
	}
}
