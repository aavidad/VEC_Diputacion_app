package domain

import (
	"errors"
	"reflect"
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

func TestRegistrarFiscalizacionTrasSubsanacionConservaPrefijoYLigaReparo(t *testing.T) {
	antecedente := expedienteConSubsanacionParaRefiscalizacion(t)
	retornoAnterior := antecedente.Fiscalizacion.Retorno.RetornoRef
	versionAnterior := antecedente.Version
	actuacionesAnteriores := append([]Actuacion(nil), antecedente.Actuaciones...)
	instante := antecedente.ActualizadoEn.Add(time.Minute)
	siguiente, err := antecedente.RegistrarFiscalizacion(versionAnterior,
		DatosRegistrarFiscalizacion{
			FiscalizacionRef:       "fiscalizacion:refiscalizacion:sintetica:01",
			Resultado:              FiscalizacionFavorable,
			UnidadFiscalizadoraRef: "unidad:intervencion:sintetica:01",
			FiscalizadaEn:          instante,
		},
		DatosActuacion{
			AccionClave: AccionRegistrarFiscalizacion, ActorRef: "actor:intervencion:sintetico:01",
			UnidadRef: "unidad:intervencion:sintetica:01", ReciboRef: "recibo:refiscalizacion:sintetica:01",
			RealizadaEn: instante, FaseDestino: FaseFiscalizacion, EstadoDestino: EstadoEnCurso,
			DocumentosRef: []string{antecedente.InformeJuridico.DocumentoRef}, RetornoRef: retornoAnterior,
		},
	)
	if err != nil {
		t.Fatalf("refiscalizar tras subsanación: %v", err)
	}
	if siguiente.Validar() != nil || siguiente.Version != versionAnterior+1 ||
		siguiente.FaseActual != FaseFiscalizacion || siguiente.EstadoActual != EstadoEnCurso ||
		siguiente.Fiscalizacion.Resultado != FiscalizacionFavorable || siguiente.Fiscalizacion.Retorno != nil ||
		len(siguiente.Actuaciones) != len(actuacionesAnteriores)+1 {
		t.Fatalf("resultado de refiscalización inválido: %#v", siguiente)
	}
	for indice, actuacion := range actuacionesAnteriores {
		if !reflect.DeepEqual(siguiente.Actuaciones[indice], actuacion) {
			t.Fatalf("se alteró actuación histórica %d", indice)
		}
	}
	ultima := siguiente.Actuaciones[len(siguiente.Actuaciones)-1]
	if ultima.RetornoRef != retornoAnterior || ultima.Secuencia != uint64(len(siguiente.Actuaciones)) ||
		ultima.VersionExpediente != siguiente.Version {
		t.Fatalf("la nueva fiscalización no quedó ligada al reparo anterior: %#v", ultima)
	}
}

func TestRegistrarFiscalizacionTrasSubsanacionDesfavorableAbreRetornoNuevo(t *testing.T) {
	antecedente := expedienteConSubsanacionParaRefiscalizacion(t)
	retornoAnterior := antecedente.Fiscalizacion.Retorno.RetornoRef
	instante := antecedente.ActualizadoEn.Add(time.Minute)
	siguiente, err := antecedente.RegistrarFiscalizacion(antecedente.Version,
		DatosRegistrarFiscalizacion{
			FiscalizacionRef: "fiscalizacion:refiscalizacion:sintetica:02", Resultado: FiscalizacionDesfavorable,
			UnidadFiscalizadoraRef: "unidad:intervencion:sintetica:01", Observaciones: "Nuevo reparo sintético.",
			FiscalizadaEn: instante, RetornoRef: "retorno:fiscalizacion:refiscalizacion:02",
		},
		DatosActuacion{
			AccionClave: AccionRegistrarFiscalizacion, ActorRef: "actor:intervencion:sintetico:01",
			UnidadRef: "unidad:intervencion:sintetica:01", ReciboRef: "recibo:refiscalizacion:sintetica:02",
			RealizadaEn: instante, FaseDestino: FaseSubsanacionUnidad, EstadoDestino: EstadoIncidencia,
			Observaciones: "Nuevo reparo sintético.", DocumentosRef: []string{antecedente.InformeJuridico.DocumentoRef},
			RetornoRef: retornoAnterior,
		},
	)
	if err != nil {
		t.Fatalf("refiscalizar desfavorable: %v", err)
	}
	if siguiente.Validar() != nil || siguiente.Fiscalizacion.Retorno == nil ||
		siguiente.Fiscalizacion.Retorno.RetornoRef == retornoAnterior ||
		siguiente.Actuaciones[len(siguiente.Actuaciones)-1].RetornoRef != retornoAnterior {
		t.Fatalf("el nuevo reparo no conserva ambos retornos: %#v", siguiente)
	}
}

func TestRegistrarFiscalizacionTrasSubsanacionRechazaRetornoYaUsado(t *testing.T) {
	antecedente := expedienteConSubsanacionParaRefiscalizacion(t)
	retornoUsado := antecedente.Fiscalizacion.Retorno.RetornoRef
	instante := antecedente.ActualizadoEn.Add(time.Minute)
	_, err := antecedente.RegistrarFiscalizacion(antecedente.Version,
		DatosRegistrarFiscalizacion{
			FiscalizacionRef: "fiscalizacion:refiscalizacion:sintetica:repetida", Resultado: FiscalizacionDesfavorable,
			UnidadFiscalizadoraRef: "unidad:intervencion:sintetica:01", Observaciones: "Reparo repetido sintético.",
			FiscalizadaEn: instante, RetornoRef: retornoUsado,
		},
		DatosActuacion{
			AccionClave: AccionRegistrarFiscalizacion, ActorRef: "actor:intervencion:sintetico:01",
			UnidadRef: "unidad:intervencion:sintetica:01", ReciboRef: "recibo:refiscalizacion:sintetica:repetida",
			RealizadaEn: instante, FaseDestino: FaseSubsanacionUnidad, EstadoDestino: EstadoIncidencia,
			Observaciones: "Reparo repetido sintético.", DocumentosRef: []string{antecedente.InformeJuridico.DocumentoRef},
			RetornoRef: retornoUsado,
		},
	)
	if !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("se aceptó retorno de fiscalización ya usado: %v", err)
	}
}

func expedienteConSubsanacionParaRefiscalizacion(t *testing.T) Expediente {
	t.Helper()
	expediente := expedienteFiscalizablePrueba(t)
	instanteFiscalizacion := expediente.ActualizadoEn.Add(time.Minute)
	var err error
	expediente, err = expediente.RegistrarFiscalizacion(expediente.Version,
		DatosRegistrarFiscalizacion{FiscalizacionRef: "fiscalizacion:origen:sintetica:01", Resultado: FiscalizacionDesfavorable,
			UnidadFiscalizadoraRef: "unidad:intervencion:sintetica:01", Observaciones: "Reparo origen sintético.",
			FiscalizadaEn: instanteFiscalizacion, RetornoRef: "retorno:fiscalizacion:origen:01"},
		DatosActuacion{AccionClave: AccionRegistrarFiscalizacion, ActorRef: "actor:intervencion:sintetico:01",
			UnidadRef: "unidad:intervencion:sintetica:01", ReciboRef: "recibo:fiscalizacion:origen:01",
			RealizadaEn: instanteFiscalizacion, FaseDestino: FaseSubsanacionUnidad, EstadoDestino: EstadoIncidencia,
			Observaciones: "Reparo origen sintético.", DocumentosRef: []string{expediente.InformeJuridico.DocumentoRef}},
	)
	if err != nil {
		t.Fatal(err)
	}
	instanteSubsanacion := expediente.ActualizadoEn.Add(time.Minute)
	expediente, err = expediente.RegistrarSubsanacionReparo(expediente.Version,
		DatosSubsanacionReparo{RetornoRef: expediente.Fiscalizacion.Retorno.RetornoRef, Observaciones: "Corrección sintética."},
		DatosActuacion{AccionClave: AccionRegistrarSubsanacionReparo, ActorRef: "actor:unidad:sintetico:01",
			UnidadRef: expediente.Asignacion.UnidadRef, ReciboRef: "recibo:subsanacion:origen:01", RealizadaEn: instanteSubsanacion,
			FaseDestino: FaseSubsanacionUnidad, EstadoDestino: EstadoIncidencia, Observaciones: "Corrección sintética.",
			RetornoRef: expediente.Fiscalizacion.Retorno.RetornoRef},
	)
	if err != nil {
		t.Fatal(err)
	}
	return expediente
}
