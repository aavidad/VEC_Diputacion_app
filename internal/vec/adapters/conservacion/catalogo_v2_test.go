package conservacion

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecapp "vec-diputacion-granada/internal/vec/application"
)

func TestCatalogoV2SeisOriginalesCT(t *testing.T) {
	reloj := relojFijo{time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)}
	c, err := NuevoCatalogoProvisionalV2(reloj)
	if err != nil || !c.Provisional() {
		t.Fatalf("catálogo v2 provisional: %v", err)
	}
	tipos := []ctports.TipoBorradorRRHH{
		ctports.BorradorInformeDefinitivo, ctports.BorradorResolucion,
		ctports.BorradorDiligencia, ctports.BorradorTomaPosesion,
		ctports.BorradorNotificacion, ctports.BorradorComunicacionCentro,
	}
	expediente := "ref:" + strings.Repeat("ab", 32)
	refs := map[string]bool{}
	politicas := map[string]bool{}
	for _, tipo := range tipos {
		clave := "contratacion_temporal.borrador." + string(tipo) + ".v1"
		ref, err := c.TipoDocumentalRef(clave)
		if err != nil || ref != referencia("tipo", clave) || refs[ref] {
			t.Fatalf("tipo original %q: ref=%q err=%v", clave, ref, err)
		}
		refs[ref] = true
		if !c.CustodiaOriginalCTReservada(ref) || c.CustodiaFirmadoReservada(ref) {
			t.Fatalf("custodia original %q incorrecta", clave)
		}
		s, err := c.SolicitudPara(clave, expediente)
		if err != nil || s.VersionPolitica() != 2 || politicas[s.PoliticaRef()] ||
			s.PoliticaRef() != referencia("politica", clave+"\x00v2") {
			t.Fatalf("política v2 de %q: %v", clave, err)
		}
		politicas[s.PoliticaRef()] = true
		if _, err := vecapp.ResolverPoliticaConservacionDocumentalAdmitiendoProvisional(context.Background(), c, reloj, s); err != nil {
			t.Fatalf("resolución v2 de %q: %v", clave, err)
		}
	}
	if len(refs) != 6 {
		t.Fatalf("refs originales: %d", len(refs))
	}
	// El firmado tiene otra clase de custodia; no puede fingirse original.
	firmado, err := c.TipoDocumentalRef("contratacion_temporal.resolucion_firmada.v1")
	if err != nil || !c.CustodiaFirmadoReservada(firmado) || c.CustodiaOriginalCTReservada(firmado) {
		t.Fatalf("tipo firmado confundido con original: %v", err)
	}
}

func TestCatalogoV2ConservaV1HistoricaYRetiraElGenerico(t *testing.T) {
	reloj := relojFijo{time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)}
	v1, err := NuevoCatalogoProvisional(reloj)
	if err != nil {
		t.Fatal(err)
	}
	v2, err := NuevoCatalogoProvisionalV2(reloj)
	if err != nil {
		t.Fatal(err)
	}
	expediente := "ref:" + strings.Repeat("cd", 32)
	historica, err := v1.SolicitudPara("contratacion_temporal.borrador.v1", expediente)
	if err != nil || historica.VersionPolitica() != 1 {
		t.Fatalf("política histórica: %v", err)
	}
	if _, err := vecapp.ResolverPoliticaConservacionDocumentalAdmitiendoProvisional(context.Background(), v1, reloj, historica); err != nil {
		t.Fatalf("v1 histórica ilegible: %v", err)
	}
	if _, err := v2.TipoDocumentalRef("contratacion_temporal.borrador.v1"); err != ErrTipoNoCatalogado {
		t.Fatalf("alta genérica antigua aún catalogada en v2: %v", err)
	}
	if _, err := vecapp.ResolverPoliticaConservacionDocumentalAdmitiendoProvisional(context.Background(), v2, reloj, historica); err == nil {
		t.Fatal("política v1 aceptada por catálogo v2")
	}
	clave := "contratacion_temporal.borrador.resolucion.v1"
	nueva, err := v2.SolicitudPara(clave, expediente)
	if err != nil || nueva.VersionPolitica() != 2 {
		t.Fatalf("política nueva: %v", err)
	}
	if bytes.Equal(nueva.HuellaPoliticaSHA256(), historica.HuellaPoliticaSHA256()) || nueva.PoliticaRef() == historica.PoliticaRef() {
		t.Fatal("v1 y v2 comparten política o huella")
	}
	if _, err := v1.TipoDocumentalRef(clave); err != ErrTipoNoCatalogado {
		t.Fatalf("tipo v2 admitido en v1: %v", err)
	}
}

func TestCatalogoV2RechazaTipoOriginalDuplicadoYV1Marcada(t *testing.T) {
	reloj := relojFijo{time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)}
	var documento catalogoJSON
	if err := json.Unmarshal(catalogoV2, &documento); err != nil {
		t.Fatal(err)
	}
	for _, entrada := range documento.Politicas {
		if entrada.Custodia == custodiaOriginalCT {
			documento.Politicas = append(documento.Politicas, entrada)
			break
		}
	}
	duplicado, err := json.Marshal(documento)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NuevoCatalogo(duplicado, reloj); err != ErrCatalogoInvalido {
		t.Fatalf("tipo original duplicado admitido: %v", err)
	}
	documento.Politicas = documento.Politicas[:len(documento.Politicas)-1]
	documento.Version = 1
	versionFalsa, err := json.Marshal(documento)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NuevoCatalogo(versionFalsa, reloj); err != ErrCatalogoInvalido {
		t.Fatalf("custodia original con versión v1 admitida: %v", err)
	}
}
