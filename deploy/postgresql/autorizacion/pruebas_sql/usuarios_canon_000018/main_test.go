package main

import (
	"bytes"
	"testing"
	"time"
)

func TestDocumentosSQLConservanMarshalYHuellaDominio(t *testing.T) {
	ahora := time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC)
	r, c, a := documentos(ahora)
	rb, _, e := canonico(r)
	if e != nil {
		t.Fatal(e)
	}
	ab, _, e := canonico(a)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(rb, []byte(`"retirada_en":"0001-01-01T00:00:00Z"`)) || !bytes.Contains(ab, []byte(`"revocada_en":"0001-01-01T00:00:00Z"`)) {
		t.Fatal("el fixture perdió las fechas cero de Marshal")
	}
	if _, _, e := canonico(c); e != nil {
		t.Fatal(e)
	}
	if _, e := sqlPrueba(ahora); e != nil {
		t.Fatal(e)
	}
	r.RetiradaEn = ahora
	if _, _, e := canonico(r); e == nil {
		t.Fatal("el dominio admitió una retirada real en estado publicada")
	}
	a.RevocadaEn = ahora
	if _, _, e := canonico(a); e == nil {
		t.Fatal("el dominio admitió una revocación real en estado activa")
	}
}
