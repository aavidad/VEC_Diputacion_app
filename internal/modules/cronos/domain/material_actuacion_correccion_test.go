package domain

import (
	"bytes"
	"errors"
	"testing"
)

func materialDecisionCorreccionPrueba() MaterialActuacionCorreccion {
	return MaterialActuacionCorreccion{
		ActorRef: "per_AAAAAAAAAAAAAAAAAAAAAA", PerfilRef: "prf_BBBBBBBBBBBBBBBBBBBBBB", EmpleadoRef: "emp_CCCCCCCCCCCCCCCCCCCCCC",
		SolicitudRef: "correccion:cronos:solicitud-0001", ClaveOperacion: "actuacion-0001",
		Paso: PasoDecisionResponsable, Resultado: ResultadoFavorable, Motivo: "Olvido acreditado",
		VersionEsperada: 1, ZonaHoraria: ZonaSaldoPeninsula,
	}
}

func TestMaterialDecisionAtaMotivoPasoYVersionEnHuella(t *testing.T) {
	m := materialDecisionCorreccionPrueba()
	primero, err := m.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	otra := m
	otra.Motivo = "No acreditado"
	segundo, err := otra.Canonico()
	if err != nil || bytes.Equal(primero, segundo) {
		t.Fatal("otro motivo conserva material", err)
	}
	otra = m
	otra.Paso, otra.VersionEsperada = PasoResolucionRRHH, 2
	tercero, err := otra.Canonico()
	if err != nil || bytes.Equal(primero, tercero) {
		t.Fatal("otro paso conserva material", err)
	}
	for nombre, alterar := range map[string]func(*MaterialActuacionCorreccion){
		"denegación sin motivo": func(x *MaterialActuacionCorreccion) { x.Resultado = ResultadoDesfavorable; x.Motivo = "" },
		"aplicación sin motivo": func(x *MaterialActuacionCorreccion) {
			x.Paso = PasoAplicacion
			x.VersionEsperada = 3
			x.Resultado = ""
			x.Motivo = ""
		},
		"aplicación con resultado": func(x *MaterialActuacionCorreccion) { x.Paso = PasoAplicacion; x.VersionEsperada = 3 },
		"motivo con control":       func(x *MaterialActuacionCorreccion) { x.Motivo = "a\nb" },
		"versión ajena":            func(x *MaterialActuacionCorreccion) { x.VersionEsperada = 2 },
	} {
		x := m
		alterar(&x)
		if _, err := x.Canonico(); !errors.Is(err, ErrCorreccionInvalida) {
			t.Fatalf("acepta %s: %v", nombre, err)
		}
	}
}
