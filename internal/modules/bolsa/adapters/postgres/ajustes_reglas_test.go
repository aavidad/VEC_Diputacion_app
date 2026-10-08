package postgres

import (
	"testing"
	"time"

	app "vec-diputacion-granada/internal/modules/bolsa/application/ajustesreglas"
	"vec-diputacion-granada/internal/vec/reglas"
)

func TestRestaurarAjustesExigeCanonicoYHuella(t *testing.T) {
	ajustes := map[string]map[string]string{"b05.plazo_respuesta": {"cantidad": "2"}}
	canon, err := reglas.CanonicoAjustes(ajustes)
	if err != nil {
		t.Fatal(err)
	}
	h, err := reglas.HuellaAjustes(ajustes)
	if err != nil {
		t.Fatal(err)
	}
	desde := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	v, err := restaurarVersionAjustesBolsa(app.CatalogoAjustes, 1, h, string(canon), desde)
	if err != nil || v.Version != 1 {
		t.Fatalf("versión válida rechazada: %v", err)
	}
	if _, err := restaurarVersionAjustesBolsa(app.CatalogoAjustes, 1, h, `{ "b05.plazo_respuesta" : {"cantidad":"2"} }`, desde); err == nil {
		t.Fatal("representación no canónica aceptada")
	}
	if _, err := restaurarVersionAjustesBolsa(app.CatalogoAjustes, 1, h[:63]+"0", string(canon), desde); err == nil {
		t.Fatal("huella distinta aceptada")
	}
}
