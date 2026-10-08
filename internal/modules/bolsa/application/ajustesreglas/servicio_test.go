package ajustesreglas

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/reglas"
)

func TestReplayExigeIntencionOriginalCompleta(t *testing.T) {
	version := 1
	fecha := "2026-10-09T10:00:00.000000Z"
	referencia := "acuerdo-1"
	clave := "123e4567-e89b-42d3-a456-426614174000"
	p := Solicitud{ClaveIdempotencia: clave, VersionEsperada: &version, VigenteDesde: &fecha,
		Cambios:     []CambioSolicitado{{ReglaClave: "b05.plazo_respuesta", Campo: "cantidad", Nuevo: "2"}},
		MotivoClave: "correccion_error", Referencia: &referencia}
	original := Material{Operacion: "ajustar", CatalogoID: CatalogoAjustes, ClaveIdempotencia: clave,
		VersionEsperada: version, VigenteDesde: &fecha, MotivoClave: p.MotivoClave, Referencia: &referencia,
		Cambios: []Cambio{{ReglaClave: "b05.plazo_respuesta", Campo: "cantidad", Anterior: "1", Nuevo: "2"}}}
	if !mismaSolicitud(original, p, &fecha) {
		t.Fatal("replay idéntico rechazado")
	}
	mutada := p
	mutada.Cambios = []CambioSolicitado{{ReglaClave: "b05.plazo_respuesta", Campo: "cantidad", Nuevo: "3"}}
	if mismaSolicitud(original, mutada, &fecha) {
		t.Fatal("clave reutilizada con otro valor")
	}
	mutada = p
	otra := "acuerdo-2"
	mutada.Referencia = &otra
	if mismaSolicitud(original, mutada, &fecha) {
		t.Fatal("clave reutilizada con otra referencia")
	}
	mutada = p
	mutada.VigenteDesde = nil
	if mismaSolicitud(original, mutada, nil) {
		t.Fatal("se confundió fecha omitida con fecha programada")
	}
}

func TestCabezaFuturaNoSePresentaVigente(t *testing.T) {
	ahora := time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC)
	ajustes := map[string]map[string]string{}
	h, err := reglas.HuellaAjustes(ajustes)
	if err != nil {
		t.Fatal(err)
	}
	cabeza := &reglas.VersionAjustes{CatalogoID: CatalogoAjustes, Version: 1, HuellaSHA256: h,
		VigenteDesde: ahora.Add(time.Hour), Ajustes: ajustes}
	l := Lectura{Cabeza: cabeza, CabezaPublicadaEn: ahora.Add(-time.Minute), ConsultadaEn: ahora}
	if err := validarLectura(l); err != nil {
		t.Fatalf("cabeza futura válida: %v", err)
	}
	cabeza.VigenteDesde = ahora.Add(-time.Second)
	if err := validarLectura(l); err == nil {
		t.Fatal("cabeza efectiva sin vigente_hoy aceptada")
	}
}
