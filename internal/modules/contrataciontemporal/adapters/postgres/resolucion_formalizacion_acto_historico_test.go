package postgres

import (
	"encoding/json"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestActoResolucionHistoricoDecodificacionCerrada(t *testing.T) {
	s := solicitudResolucionPrueba()
	r := reciboResolucionPrueba(s)
	for _, version := range []uint64{8, 9, 12} {
		p := ports.PreparacionResolucionFormalizacion{
			ExpedienteRef: s.ExpedienteRef, PropuestaRef: s.PropuestaRef,
			VersionEsperada: 7, VersionActual: version, Recibo: &r,
		}
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		actual, err := decodificarActoResolucion(string(b), s.ExpedienteRef, version)
		if err != nil || actual != r {
			t.Fatal("recibo v8 perdido tras avanzar expediente", version, err)
		}
		for _, raw := range []string{
			string(b) + "{}", "null", strings.Repeat("x", 16385),
			strings.Replace(string(b), `"ExpedienteRef":"expediente:prueba"`, `"ExpedienteRef":"expediente:ajeno"`, 1),
			strings.Replace(string(b), `"DocumentoSHA256":"`+r.DocumentoSHA256+`"`, `"DocumentoSHA256":"privado"`, 1),
		} {
			if _, err := decodificarActoResolucion(raw, s.ExpedienteRef, version); err == nil {
				t.Fatal("lectura aceptó recibo ajeno o adulterado")
			}
		}
		if _, err := decodificarActoResolucion(string(b), "expediente:ajeno", version); err == nil {
			t.Fatal("expediente ajeno")
		}
	}
	p := ports.PreparacionResolucionFormalizacion{ExpedienteRef: s.ExpedienteRef, PropuestaRef: s.PropuestaRef, VersionEsperada: 7, VersionActual: 7}
	b, _ := json.Marshal(p)
	if _, err := decodificarActoResolucion(string(b), s.ExpedienteRef, 7); err == nil {
		t.Fatal("v7 sin acto")
	}
}
