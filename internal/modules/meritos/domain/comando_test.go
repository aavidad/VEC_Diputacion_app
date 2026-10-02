package domain

import (
	"bytes"
	"strings"
	"testing"

	vec "vec-diputacion-granada/internal/vec/domain"
)

func TestComandoCanonicoNormalizaEvidenciaSinAlterarEntrada(t *testing.T) {
	hecho := paquetePrueba().Hechos[0]
	c := ComandoHecho{Esquema: EsquemaComandoHecho, Accion: "meritos.hecho.declarar", ActorRef: hecho.PersonaRef, ClaveIdempotencia: "clave:prueba", Hecho: hecho,
		Motivo: vec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("b", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"}, FechaCorte: "2026-10-01"}
	primero, err := c.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	c.Hecho.Evidencias = []vec.ReferenciaDocumento{}
	segundo, err := c.RepresentacionCanonica()
	if err != nil || !bytes.Equal(primero, segundo) {
		t.Fatal("nil y vacío cambian idempotencia", err)
	}
	c.Hecho.Evidencias = []vec.ReferenciaDocumento{{ID: "documento:b", Version: 1}, {ID: "documento:a", Version: 2}}
	ordenado, err := c.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	if c.Hecho.Evidencias[0].ID != "documento:b" {
		t.Fatal("modifica el dato aportado")
	}
	c.Hecho.Evidencias[0], c.Hecho.Evidencias[1] = c.Hecho.Evidencias[1], c.Hecho.Evidencias[0]
	repetido, err := c.RepresentacionCanonica()
	if err != nil || !bytes.Equal(ordenado, repetido) {
		t.Fatal("orden de evidencia cambia significado", err)
	}
}
