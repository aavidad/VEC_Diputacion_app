package domain

import (
	"testing"
	"time"
)

// expedienteModificadoPrueba aplica una modificación de jornada al fixture v7
// para dejarlo en fiscalización, como CT116.
func expedienteModificadoPrueba(t *testing.T) Expediente {
	t.Helper()
	e, _ := expedienteNombramientoPrueba(t)
	instante := instantePrueba(e)
	datos := DatosModificacionTrasNombramiento{MotivoClave: "cambio_jornada", Periodo: e.Analisis.Periodo, Jornada: 5000,
		Coste: Importe{Moneda: "EUR", Centimos: 2000000}, FuenteCoste: "autoridad:ct:desarrollo:calculo-coste",
		FaseRetorno: FaseFiscalizacion, Observaciones: "Reducción de jornada"}
	act := DatosActuacion{AccionClave: AccionModificarTrasNombramiento, ActorRef: "per_actor", UnidadRef: e.Asignacion.UnidadRef,
		ReciboRef: "recibo:mod:1", RealizadaEn: instante, FaseDestino: FaseFiscalizacion, EstadoDestino: EstadoEnCurso,
		Observaciones: datos.Observaciones}
	modificado, err := e.ModificarTrasNombramiento(e.Version, datos, act)
	if err != nil {
		t.Fatal(err)
	}
	return modificado
}

func fiscalizarModificacionPrueba(e Expediente, resultado ResultadoFiscalizacion, observaciones, retorno, retornoPrevio string) (Expediente, error) {
	instante := e.ActualizadoEn.Add(time.Minute)
	fase, estado := e.DestinoFiscalizacion(resultado)
	return e.RegistrarFiscalizacion(e.Version,
		DatosRegistrarFiscalizacion{FiscalizacionRef: "fiscalizacion:mod:1", Resultado: resultado,
			UnidadFiscalizadoraRef: "unidad:intervencion", Observaciones: observaciones, FiscalizadaEn: instante, RetornoRef: retorno},
		DatosActuacion{AccionClave: AccionRegistrarFiscalizacion, ActorRef: "per_interventor", UnidadRef: "unidad:intervencion",
			ReciboRef: "recibo:fiscalizacion:mod:1", RealizadaEn: instante, FaseDestino: fase, EstadoDestino: estado,
			Observaciones: observaciones, DocumentosRef: []string{e.InformeJuridico.DocumentoRef}, RetornoRef: retornoPrevio})
}

func TestFiscalizacionDeModificacionVuelveAlNombramiento(t *testing.T) {
	e := expedienteModificadoPrueba(t)
	if !e.EsModificacionPendienteFiscalizacion() || e.ReciboModificacionPendienteFiscalizacion() != "recibo:mod:1" {
		t.Fatal("la modificación en fiscalización no se reconoce")
	}
	for _, caso := range []struct {
		resultado     ResultadoFiscalizacion
		observaciones string
	}{{FiscalizacionFavorable, ""}, {FiscalizacionFavorableConObservaciones, "Revisar la jornada en el próximo periodo"}} {
		siguiente, err := fiscalizarModificacionPrueba(e, caso.resultado, caso.observaciones, "", "")
		if err != nil {
			t.Fatalf("%s: %v", caso.resultado, err)
		}
		if siguiente.FaseActual != FaseNombramiento || siguiente.EstadoActual != EstadoEnCurso || siguiente.Fiscalizacion == nil ||
			siguiente.Fiscalizacion.Resultado != caso.resultado || siguiente.Analisis.PorcentajeJornada != 5000 || siguiente.Validar() != nil {
			t.Fatalf("%s: no vuelve al nombramiento con la fiscalización vigente", caso.resultado)
		}
		if siguiente.EsModificacionPendienteFiscalizacion() || !siguiente.enNombramientoVigente() {
			t.Fatalf("%s: el expediente debe quedar en nombramiento vigente", caso.resultado)
		}
	}
}

func TestFiscalizacionDesfavorableDeModificacionVuelveALaUnidad(t *testing.T) {
	e := expedienteModificadoPrueba(t)
	siguiente, err := fiscalizarModificacionPrueba(e, FiscalizacionDesfavorable, "Coste no justificado", "retorno:mod:1", "")
	if err != nil {
		t.Fatal(err)
	}
	if siguiente.FaseActual != FaseSubsanacionUnidad || siguiente.EstadoActual != EstadoIncidencia ||
		siguiente.Fiscalizacion.Retorno == nil || siguiente.Fiscalizacion.Retorno.RetornoRef != "retorno:mod:1" {
		t.Fatal("el desfavorable debe volver a la unidad gestora con su retorno")
	}
}

func TestFiscalizacionDeModificacionRechazaFormasAjenas(t *testing.T) {
	e := expedienteModificadoPrueba(t)
	if _, err := fiscalizarModificacionPrueba(e, FiscalizacionFavorable, "", "", "retorno:previo"); err == nil {
		t.Fatal("una fiscalización de modificación no cita un retorno previo")
	}
	// Quedarse en fiscalización (forma de CT52/CT93) no vale para la modificación.
	instante := e.ActualizadoEn.Add(time.Minute)
	if _, err := e.RegistrarFiscalizacion(e.Version,
		DatosRegistrarFiscalizacion{FiscalizacionRef: "fiscalizacion:mod:1", Resultado: FiscalizacionFavorable,
			UnidadFiscalizadoraRef: "unidad:intervencion", FiscalizadaEn: instante},
		DatosActuacion{AccionClave: AccionRegistrarFiscalizacion, ActorRef: "per_interventor", UnidadRef: "unidad:intervencion",
			ReciboRef: "recibo:fiscalizacion:mod:1", RealizadaEn: instante, FaseDestino: FaseFiscalizacion, EstadoDestino: EstadoEnCurso,
			DocumentosRef: []string{e.InformeJuridico.DocumentoRef}}); err == nil {
		t.Fatal("la fiscalización de una modificación debe volver al nombramiento")
	}
	nombrado, _ := expedienteNombramientoPrueba(t)
	if nombrado.EsModificacionPendienteFiscalizacion() || nombrado.ReciboModificacionPendienteFiscalizacion() != "" {
		t.Fatal("un expediente sin modificación no está pendiente de fiscalizar")
	}
	if _, err := fiscalizarModificacionPrueba(nombrado, FiscalizacionFavorable, "", "", ""); err == nil {
		t.Fatal("un expediente en nombramiento no se fiscaliza")
	}
}
