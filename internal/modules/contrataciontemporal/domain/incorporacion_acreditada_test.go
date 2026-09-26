package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestConfirmarGINPIXAnadeActuacionComoCT124(t *testing.T) {
	e, original := expedienteNombramientoPrueba(t)
	instante := instantePrueba(e)
	datos := DatosConfirmacionGINPIX{Numero: "GX-2027-0042", ConfirmadaEn: time.Date(2027, 2, 10, 0, 0, 0, 0, time.UTC)}
	act := DatosActuacion{AccionClave: AccionConfirmarGINPIX, ActorRef: "per_actor", UnidadRef: e.Asignacion.UnidadRef,
		ReciboRef: "recibo:ginpix:1", RealizadaEn: instante, FaseDestino: FaseNombramiento, EstadoDestino: EstadoEnCurso,
		DocumentosRef: []string{"ginpix:GX-2027-0042"}}
	siguiente, err := e.ConfirmarGINPIX(e.Version, datos, act)
	if err != nil {
		t.Fatal(err)
	}
	// La proyección Go es la mezcla que hace CT124: versión, instante y actuación.
	esperado := original
	esperado["version"] = float64(e.Version + 1)
	esperado["actualizado_en"] = instante.Format(time.RFC3339Nano)
	acts := append([]any(nil), original["actuaciones"].([]any)...)
	acts = append(acts, mapaJSON(t, siguiente.Actuaciones[len(siguiente.Actuaciones)-1]))
	esperado["actuaciones"] = acts
	if !reflect.DeepEqual(mapaJSON(t, siguiente), esperado) {
		t.Fatal("la proyección Go de la confirmación de GINPIX no coincide con la mezcla de CT124")
	}
	for nombre, mutar := range map[string]func(*DatosActuacion, *DatosConfirmacionGINPIX){
		"documento distinto": func(a *DatosActuacion, _ *DatosConfirmacionGINPIX) { a.DocumentosRef = []string{"ginpix:OTRO"} },
		"sin documento":      func(a *DatosActuacion, _ *DatosConfirmacionGINPIX) { a.DocumentosRef = nil },
		"acción ajena":       func(a *DatosActuacion, _ *DatosConfirmacionGINPIX) { a.AccionClave = AccionCesarNombramiento },
		"estado completado":  func(a *DatosActuacion, _ *DatosConfirmacionGINPIX) { a.EstadoDestino = EstadoCompletado },
		"número inválido":    func(_ *DatosActuacion, d *DatosConfirmacionGINPIX) { d.Numero = "-x" },
		"fecha con hora":     func(_ *DatosActuacion, d *DatosConfirmacionGINPIX) { d.ConfirmadaEn = d.ConfirmadaEn.Add(time.Hour) },
	} {
		a, d := act, datos
		mutar(&a, &d)
		if _, err := e.ConfirmarGINPIX(e.Version, d, a); err == nil {
			t.Fatalf("%s: se esperaba rechazo", nombre)
		}
	}
	if _, err := e.ConfirmarGINPIX(e.Version-1, datos, act); err != ErrVersionEnConflicto {
		t.Fatalf("versión esperada: %v", err)
	}
	if !NumeroGINPIXValido("GX-2027-0042") || NumeroGINPIXValido("") || datos.DocumentoGINPIX() != "ginpix:GX-2027-0042" {
		t.Fatal("formato del número de GINPIX")
	}
}
