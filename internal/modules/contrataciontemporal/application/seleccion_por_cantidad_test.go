package application

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

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

// F1-19: el nombramiento derivado copia del padre la fiscalización favorable y
// queda con una persona. No entra en ninguna vía de selección, ni nueva ni de
// recuperación; el padre y el expediente ordinario siguen igual.
func TestSeleccionPorCantidadRechazaNombramientoDerivado(t *testing.T) {
	contenido, err := os.ReadFile("../domain/testdata/expediente_nombramiento_v7.json")
	if err != nil {
		t.Fatal(err)
	}
	var padre domain.Expediente
	if err := json.Unmarshal(contenido, &padre); err != nil {
		t.Fatal(err)
	}
	padre.Actuaciones = padre.Actuaciones[:6]
	padre.Version = 6
	padre.FaseActual = domain.FaseFiscalizacion
	padre.ActualizadoEn = padre.Actuaciones[5].RealizadaEn
	padre.Solicitud.NumeroPersonas = 2
	padre.Solicitud.JornadaMinutos = 420
	padre.Solicitud.PuestoSolicitado = "puesto:administrativo:01"
	if err := padre.Validar(); err != nil {
		t.Fatalf("fixture padre: %v", err)
	}
	preimagen, err := json.Marshal(padre)
	if err != nil {
		t.Fatal(err)
	}
	hijo, err := domain.NuevoExpedienteDerivado(domain.AltaExpedienteDerivado{
		Padre: padre, PreimagenPadreJSON: string(preimagen),
		Referencia: ports.HijoRefDerivacionNombramiento(padre.OrganizacionRef, padre.Referencia, 1),
		OfertaRef:  "oferta:" + strings.Repeat("a", 64), NumeroPlaza: 1,
		ActorRef: "actor:rrhh:sintetico:01", ReciboRef: "recibo:derivacion:sintetica:01",
		CreadaEn: padre.ActualizadoEn.Add(time.Minute),
	})
	if err != nil || hijo.Solicitud.NumeroPersonas != 1 || hijo.Fiscalizacion == nil {
		t.Fatalf("fixture hijo: %v", err)
	}
	ordinario := padre.Clonar()
	ordinario.Solicitud.NumeroPersonas = 1
	if err := ordinario.Validar(); err != nil {
		t.Fatalf("fixture ordinario: %v", err)
	}
	casos := []struct {
		nombre                string
		e                     domain.Expediente
		recuperar             bool
		err                   error
		singular, sel, recupe int
	}{
		{"hijo nuevo", hijo, false, ports.ErrEstadoExpedienteNoSeleccionable, 0, 0, 0},
		{"hijo recuperación", hijo, true, ports.ErrEstadoExpedienteNoSeleccionable, 0, 0, 0},
		{"padre plural", padre, false, nil, 0, 1, 0},
		{"ordinario singular", ordinario, false, nil, 1, 0, 0},
	}
	for _, c := range casos {
		q := SolicitudSeleccionLlamamiento{ExpedienteRef: c.e.Referencia, VersionEsperada: c.e.Version,
			ClaveIdempotencia: "018f47a2-6b31-4c80-8a95-4d2e707c5a11", RecuperarOferta: c.recuperar}
		singular, oferta := &singularPorCantidadPrueba{}, &ofertaPorCantidadPrueba{}
		_, err := SeleccionarSegunNumeroPersonasCT(t.Context(), ports.ExpedienteParaSeleccion{
			Fiscalizado: c.e, VersionActual: c.e.Version,
		}, q, singular, oferta)
		if !errors.Is(err, c.err) || (c.err == nil && err != nil) || singular.llamadas != c.singular ||
			oferta.seleccion != c.sel || oferta.recuperacion != c.recupe {
			t.Fatalf("%s: err=%v singular=%d selección=%d recuperación=%d",
				c.nombre, err, singular.llamadas, oferta.seleccion, oferta.recuperacion)
		}
	}
}
