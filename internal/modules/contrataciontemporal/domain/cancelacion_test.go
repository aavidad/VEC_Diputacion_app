package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"
)

func expedienteCancelacionPrueba(t *testing.T, fichero string) Expediente {
	t.Helper()
	contenido, err := os.ReadFile("testdata/" + fichero)
	if err != nil {
		t.Fatal(err)
	}
	var e Expediente
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		t.Fatal(err)
	}
	if err := e.Validar(); err != nil {
		t.Fatalf("fixture %s no válido: %v", fichero, err)
	}
	return e
}

var fasesCancelacionPrueba = []ClaveFase{"solicitud", "asignacion_unidad", "informe_juridico"}

func actuacionCancelacionPrueba(e Expediente, observaciones string) DatosActuacion {
	return DatosActuacion{AccionClave: AccionCancelarExpediente, ActorRef: "per_antonio_reyes", UnidadRef: e.UnidadActual(),
		ReciboRef: "recibo:cancelacion:prueba", RealizadaEn: e.ActualizadoEn.Add(time.Minute), FaseDestino: e.FaseActual,
		EstadoDestino: EstadoCancelado, Observaciones: observaciones}
}

func TestCancelarAntesDeLaFiscalizacionDejaElExpedienteCanceladoEnSuFase(t *testing.T) {
	e := expedienteCancelacionPrueba(t, "expediente_asignacion_v3.json")
	datos := DatosCancelacion{MotivoClave: "necesidad_desaparecida", Observaciones: "El centro ya no necesita el refuerzo",
		Canal: CanalCancelacionRRHH, FasesAdmitidas: fasesCancelacionPrueba}
	cancelado, err := e.Cancelar(e.Version, datos, actuacionCancelacionPrueba(e, datos.Observaciones))
	if err != nil {
		t.Fatalf("cancelar: %v", err)
	}
	ultima := cancelado.Actuaciones[len(cancelado.Actuaciones)-1]
	if cancelado.EstadoActual != EstadoCancelado || cancelado.FaseActual != e.FaseActual || cancelado.Version != e.Version+1 ||
		ultima.EstadoOrigen != EstadoEnCurso || ultima.AccionClave != AccionCancelarExpediente || len(cancelado.Actuaciones) != len(e.Actuaciones)+1 {
		t.Fatalf("resultado inesperado: %+v", ultima)
	}
	if e.EstadoActual != EstadoEnCurso || len(e.Actuaciones) != 3 {
		t.Fatal("el original no debe cambiar")
	}
	// Terminal: ninguna transición posterior, tampoco otra cancelación.
	if _, err := cancelado.Cancelar(cancelado.Version, datos, actuacionCancelacionPrueba(cancelado, datos.Observaciones)); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("segunda cancelación: %v", err)
	}
}

func TestTrasLaFiscalizacionNoSePuedeCancelar(t *testing.T) {
	e := expedienteCancelacionPrueba(t, "expediente_nombramiento_v7.json")
	if !e.Fiscalizado() {
		t.Fatal("el fixture v7 está fiscalizado")
	}
	// Aunque el catálogo admitiera la fase, la barrera de la fiscalización se mantiene.
	datos := DatosCancelacion{MotivoClave: "necesidad_desaparecida", Canal: CanalCancelacionRRHH,
		FasesAdmitidas: []ClaveFase{"nombramiento", "fiscalizacion"}}
	if e.CancelableEn(datos.FasesAdmitidas) {
		t.Fatal("expediente fiscalizado cancelable")
	}
	if _, err := e.Cancelar(e.Version, datos, actuacionCancelacionPrueba(e, "")); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("cancelación tras fiscalización: %v", err)
	}
	// La modificación tras el nombramiento retira la fiscalización vigente,
	// pero la historia la conserva.
	sinProyeccion := e.Clonar()
	sinProyeccion.Fiscalizacion = nil
	if !sinProyeccion.Fiscalizado() {
		t.Fatal("la historia de fiscalización debe bastar")
	}
}

func TestCancelacionRespetaLasFasesDelCatalogoYLaActuacion(t *testing.T) {
	e := expedienteCancelacionPrueba(t, "expediente_asignacion_v3.json")
	base := DatosCancelacion{MotivoClave: "error_solicitud", Canal: CanalCancelacionCentro, FasesAdmitidas: []ClaveFase{"solicitud"}}
	if _, err := e.Cancelar(e.Version, base, actuacionCancelacionPrueba(e, "")); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("fase no admitida por el catálogo: %v", err)
	}
	base.FasesAdmitidas = fasesCancelacionPrueba
	casos := map[string]func(*DatosCancelacion, *DatosActuacion){
		"sin motivo":            func(d *DatosCancelacion, _ *DatosActuacion) { d.MotivoClave = "" },
		"canal desconocido":     func(d *DatosCancelacion, _ *DatosActuacion) { d.Canal = "intervencion" },
		"fases repetidas":       func(d *DatosCancelacion, _ *DatosActuacion) { d.FasesAdmitidas = []ClaveFase{"solicitud", "solicitud"} },
		"observación con borde": func(d *DatosCancelacion, a *DatosActuacion) { d.Observaciones = " texto"; a.Observaciones = " texto" },
		"otra acción":           func(_ *DatosCancelacion, a *DatosActuacion) { a.AccionClave = AccionCerrarExpediente },
		"cambia de fase":        func(_ *DatosCancelacion, a *DatosActuacion) { a.FaseDestino = "informe_juridico" },
		"estado completado":     func(_ *DatosCancelacion, a *DatosActuacion) { a.EstadoDestino = EstadoCompletado },
		"otra unidad":           func(_ *DatosCancelacion, a *DatosActuacion) { a.UnidadRef = "unidad:desarrollo:otra" },
		"observación distinta":  func(_ *DatosCancelacion, a *DatosActuacion) { a.Observaciones = "otra" },
		"con documentos":        func(_ *DatosCancelacion, a *DatosActuacion) { a.DocumentosRef = []string{"documento:x"} },
	}
	for nombre, alterar := range casos {
		d, a := base, actuacionCancelacionPrueba(e, "")
		alterar(&d, &a)
		if _, err := e.Cancelar(e.Version, d, a); err == nil {
			t.Fatalf("%s: se esperaba rechazo", nombre)
		}
	}
	if _, err := e.Cancelar(e.Version-1, base, actuacionCancelacionPrueba(e, "")); !errors.Is(err, ErrVersionEnConflicto) {
		t.Fatalf("versión desfasada: %v", err)
	}
}

func TestElCentroCancelaSuPeticionEnSolicitud(t *testing.T) {
	e := expedienteCancelacionPrueba(t, "expediente_solicitud_v1.json")
	datos := DatosCancelacion{MotivoClave: "desistimiento_centro", Canal: CanalCancelacionCentro, FasesAdmitidas: fasesCancelacionPrueba}
	cancelado, err := e.Cancelar(1, datos, actuacionCancelacionPrueba(e, ""))
	if err != nil || cancelado.EstadoActual != EstadoCancelado || cancelado.FaseActual != "solicitud" || cancelado.Version != 2 {
		t.Fatalf("cancelación en solicitud: %+v %v", cancelado.Actuaciones, err)
	}
}
