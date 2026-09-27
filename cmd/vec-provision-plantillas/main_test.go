package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCargaProvisionUsaHuellaCanonicaDelCatalogoPublicado(t *testing.T) {
	ruta := filepath.Join("..", "..", "data", "demo", "plantillas", "ct_plantillas_documentos.ejemplo.demo.json")
	c, err := cargarCatalogo(context.Background(), ruta, time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	h, err := c.HuellaSHA256()
	if err != nil || len(h) != 64 || c.Version != 1 || c.Estado != "publicado" || len(c.Entradas) < 6 {
		t.Fatalf("catálogo de origen incompatible: %+v, %v", c, err)
	}
	r := reciboProvision{Resultado: "registrado", ReciboRef: "recibo:11111111-1111-4111-8111-111111111111", Version: int64(c.Version), Revision: int64(c.Revision), CatalogoHuellaSHA256: h, ContenidoJSONSHA256: strings.Repeat("a", 64), ProcedenciaRef: c.FuenteRef, RegistradaEn: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)}
	if !reciboCompatible(r, c, h) {
		t.Fatal("recibo legítimo rechazado")
	}
	r.CatalogoHuellaSHA256 = strings.Repeat("b", 64)
	if reciboCompatible(r, c, h) {
		t.Fatal("recibo con otra huella aceptado")
	}
}
