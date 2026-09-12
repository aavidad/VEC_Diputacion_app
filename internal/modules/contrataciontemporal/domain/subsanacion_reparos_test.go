package domain

import (
	"errors"
	"testing"
	"time"
)

func TestRegistrarSubsanacionReparoConservaFiscalizacionDesfavorable(t *testing.T) {
	expediente := expedienteFiscalizablePrueba(t)
	instanteFiscalizacion := expediente.ActualizadoEn.Add(time.Minute)
	var err error
	expediente, err = expediente.RegistrarFiscalizacion(expediente.Version, DatosRegistrarFiscalizacion{
		FiscalizacionRef: "fiscalizacion:subsanacion:sintetica:01", Resultado: FiscalizacionDesfavorable,
		UnidadFiscalizadoraRef: "unidad:intervencion:sintetica:01", Observaciones: "Reparo sintético.",
		FiscalizadaEn: instanteFiscalizacion, RetornoRef: "retorno:fiscalizacion:subsanacion:01",
	}, DatosActuacion{AccionClave: AccionRegistrarFiscalizacion, ActorRef: "actor:intervencion:sintetico:01", UnidadRef: "unidad:intervencion:sintetica:01", ReciboRef: "recibo:fiscalizacion:subsanacion:01", RealizadaEn: instanteFiscalizacion, FaseDestino: FaseSubsanacionUnidad, EstadoDestino: EstadoIncidencia, Observaciones: "Reparo sintético.", DocumentosRef: []string{expediente.InformeJuridico.DocumentoRef}})
	if err != nil {
		t.Fatal(err)
	}
	anterior := expediente
	siguiente, err := expediente.RegistrarSubsanacionReparo(expediente.Version,
		DatosSubsanacionReparo{RetornoRef: expediente.Fiscalizacion.Retorno.RetornoRef, Observaciones: "Se aporta la corrección sintética."},
		DatosActuacion{AccionClave: AccionRegistrarSubsanacionReparo, ActorRef: "actor:unidad:sintetico:01", UnidadRef: expediente.Asignacion.UnidadRef, ReciboRef: "recibo:subsanacion:sintetica:01", RealizadaEn: expediente.ActualizadoEn.Add(time.Minute), FaseDestino: FaseSubsanacionUnidad, EstadoDestino: EstadoIncidencia, Observaciones: "Se aporta la corrección sintética.", RetornoRef: expediente.Fiscalizacion.Retorno.RetornoRef},
	)
	if err != nil {
		t.Fatal(err)
	}
	if siguiente.Version != anterior.Version+1 || siguiente.FaseActual != FaseSubsanacionUnidad || siguiente.EstadoActual != EstadoIncidencia || siguiente.Fiscalizacion.Resultado != FiscalizacionDesfavorable || siguiente.Fiscalizacion.Retorno.RetornoRef != anterior.Fiscalizacion.Retorno.RetornoRef {
		t.Fatalf("subsanación no conservadora: %#v", siguiente)
	}
}

func TestRegistrarSubsanacionReparoRechazaFueraDeFase(t *testing.T) {
	expediente := expedienteFiscalizablePrueba(t)
	_, err := expediente.RegistrarSubsanacionReparo(expediente.Version,
		DatosSubsanacionReparo{RetornoRef: "retorno:fiscalizacion:subsanacion:01", Observaciones: "Corrección sintética."},
		DatosActuacion{AccionClave: AccionRegistrarSubsanacionReparo, ActorRef: "actor:unidad:sintetico:01", UnidadRef: expediente.Asignacion.UnidadRef, ReciboRef: "recibo:subsanacion:sintetica:01", RealizadaEn: expediente.ActualizadoEn.Add(time.Minute), FaseDestino: FaseSubsanacionUnidad, EstadoDestino: EstadoIncidencia, Observaciones: "Corrección sintética."},
	)
	if !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("subsanación fuera de fase aceptada: %v", err)
	}
}
