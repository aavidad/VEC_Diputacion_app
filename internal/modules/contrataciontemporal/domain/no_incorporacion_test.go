package domain

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRegistrarNoIncorporacionVuelveAFiscalizacionComoCT124(t *testing.T) {
	e, original := expedienteNombramientoPrueba(t)
	instante := instantePrueba(e)
	// Confirmación de cuatro ojos: quien confirma es quien resuelve.
	datos := DatosNoIncorporacion{Paso: PasoNoIncorporacionConfirmar, PropuestaRef: "recibo:ct124:np1", MotivoClave: "no_presentado",
		ConsecuenciaClave: "b24.sancion.baja_llamamiento_directo", ResolucionRef: "resolucion:rrhh:2026/0142", ResolucionSHA256: strings.Repeat("b", 64),
		ResueltaPor: "per_segunda", SegundaPersona: true, FechaNotificacion: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)}
	act := DatosActuacion{AccionClave: AccionRegistrarNoIncorporacion, ActorRef: "per_segunda", UnidadRef: e.Asignacion.UnidadRef,
		ReciboRef: "recibo:no-incorporacion:1", RealizadaEn: instante, FaseDestino: FaseFiscalizacion, EstadoDestino: EstadoEnCurso,
		DocumentosRef: []string{"resolucion:rrhh:2026/0142"}}
	siguiente, err := e.RegistrarNoIncorporacion(e.Version, datos, act)
	if err != nil {
		t.Fatal(err)
	}
	// La proyección Go es la mezcla que hace CT124: versión, instante, fase y actuación.
	esperado := original
	esperado["version"] = float64(e.Version + 1)
	esperado["actualizado_en"] = instante.Format(time.RFC3339Nano)
	esperado["fase_actual"] = string(FaseFiscalizacion)
	acts := append([]any(nil), original["actuaciones"].([]any)...)
	acts = append(acts, mapaJSON(t, siguiente.Actuaciones[len(siguiente.Actuaciones)-1]))
	esperado["actuaciones"] = acts
	if !reflect.DeepEqual(mapaJSON(t, siguiente), esperado) {
		t.Fatal("la proyección Go de la no incorporación no coincide con la mezcla de CT124")
	}
	for nombre, mutar := range map[string]func(*DatosActuacion, *DatosNoIncorporacion){
		"resuelve otro":      func(_ *DatosActuacion, d *DatosNoIncorporacion) { d.ResueltaPor = "per_otro" },
		"sin propuesta":      func(_ *DatosActuacion, d *DatosNoIncorporacion) { d.PropuestaRef = "" },
		"paso desconocido":   func(_ *DatosActuacion, d *DatosNoIncorporacion) { d.Paso = "firmar" },
		"otra fase":          func(a *DatosActuacion, _ *DatosNoIncorporacion) { a.FaseDestino = FaseNombramiento },
		"documento distinto": func(a *DatosActuacion, _ *DatosNoIncorporacion) { a.DocumentosRef = []string{"resolucion:otra"} },
		"acción ajena":       func(a *DatosActuacion, _ *DatosNoIncorporacion) { a.AccionClave = AccionCesarNombramiento },
		"motivo inválido":    func(_ *DatosActuacion, d *DatosNoIncorporacion) { d.MotivoClave = "No" },
		"huella vacía":       func(_ *DatosActuacion, d *DatosNoIncorporacion) { d.ResolucionSHA256 = strings.Repeat("0", 64) },
		"fecha con hora": func(_ *DatosActuacion, d *DatosNoIncorporacion) {
			d.FechaNotificacion = d.FechaNotificacion.Add(time.Hour)
		},
	} {
		a, d := act, datos
		mutar(&a, &d)
		if _, err := e.RegistrarNoIncorporacion(e.Version, d, a); err == nil {
			t.Fatalf("%s: se esperaba rechazo", nombre)
		}
	}
	sinSegunda := datos
	sinSegunda.Paso, sinSegunda.PropuestaRef, sinSegunda.SegundaPersona, sinSegunda.ResueltaPor = PasoNoIncorporacionRegistrar, "", false, "per_declarada"
	if _, err := e.RegistrarNoIncorporacion(e.Version, sinSegunda, act); err != nil {
		t.Fatalf("sin segunda persona se registra en un paso con quien resolvió declarado: %v", err)
	}
	conSegunda := sinSegunda
	conSegunda.SegundaPersona = true
	if _, err := e.RegistrarNoIncorporacion(e.Version, conSegunda, act); err == nil {
		t.Fatal("con segunda persona no hay registro de un solo paso")
	}
	// Proponer y rechazar: sin efecto, siguen en nombramiento.
	for _, paso := range []string{PasoNoIncorporacionProponer, PasoNoIncorporacionRechazar} {
		d := datos
		d.Paso = paso
		if paso == PasoNoIncorporacionProponer {
			d.PropuestaRef, d.ResueltaPor = "", ""
		}
		a := act
		a.AccionClave, a.FaseDestino = d.AccionActuacion(), FaseNombramiento
		sig, err := e.RegistrarNoIncorporacion(e.Version, d, a)
		if err != nil || sig.FaseActual != FaseNombramiento || sig.Version != e.Version+1 || d.ConEfecto() {
			t.Fatalf("%s: %v", paso, err)
		}
		a.FaseDestino = FaseFiscalizacion
		if _, err := e.RegistrarNoIncorporacion(e.Version, d, a); err == nil {
			t.Fatalf("%s con vuelta a fiscalización", paso)
		}
	}
	if _, err := siguiente.RegistrarNoIncorporacion(siguiente.Version, datos, act); err == nil {
		t.Fatal("un expediente ya en fiscalización admitió otra no incorporación")
	}
}
