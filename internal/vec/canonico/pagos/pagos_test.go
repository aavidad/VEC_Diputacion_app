package pagos

import (
	"slices"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestIdentificadoresDeterministasConservanContrato(t *testing.T) {
	const (
		orden  = "cob_0123456789abcdefghijkl"
		huella = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	)
	auditoria := IDAuditoria(orden, 3, huella, domain.HechoCobroOrdenCreada, domain.AccionCobroCrearOrden)
	if esperado := "aud_cob_c125647034607ea0e8172d5eec78a151e9e6c4f30ca9abf1b754e0ac907bf3dc"; auditoria != esperado {
		t.Fatalf("ID de auditoria distinto: %q", auditoria)
	}
	evento := IDEvento(
		orden, 3, 3, huella, domain.HechoCobroOrdenCreada, domain.EstadoCobroCreada, domain.AccionCobroCrearOrden,
	)
	if esperado := "evt_cob_db1a1cae4c8f8e4cfd060fdfdf2bd2a2f5b54fb6c3e6d249622433ca9cd0a441"; evento != esperado {
		t.Fatalf("ID de evento distinto: %q", evento)
	}
}

func TestConfiguracionOrigenTieneBytesCanonicosSinMutarEntrada(t *testing.T) {
	publicadaEn := time.Date(2026, time.July, 18, 10, 11, 12, 123456789, time.UTC)
	origen := OrigenPasarelaCobroPublicado{
		ID: "pasarela_corporativa", Version: 2, BaseHTTPS: "https://pagos.example.test",
		RutasPermitidas: []string{"/z", "/a"}, CamposHandoffPermitidos: []string{"operacion", "firma"},
		PublicadaEn: publicadaEn,
	}
	rutasOriginales := slices.Clone(origen.RutasPermitidas)
	camposOriginales := slices.Clone(origen.CamposHandoffPermitidos)
	bytes, err := BytesConfiguracionOrigen(origen)
	if err != nil {
		t.Fatal(err)
	}
	const esperado = `{"VersionEsquema":1,"ID":"pasarela_corporativa","Version":2,` +
		`"BaseHTTPS":"https://pagos.example.test","Rutas":["/a","/z"],` +
		`"Campos":["firma","operacion"],"PublicadaEn":"2026-07-18T10:11:12.123456789Z"}`
	if string(bytes) != esperado {
		t.Fatalf("bytes canonicos distintos:\n%s", bytes)
	}
	if !slices.Equal(origen.RutasPermitidas, rutasOriginales) || !slices.Equal(origen.CamposHandoffPermitidos, camposOriginales) {
		t.Fatal("la canonizacion modifico las listas del llamador")
	}
	huella, err := HuellaConfiguracionOrigen(origen)
	if err != nil || huella != "0109bafadb29b3e5c41700d013db18dd9aea91b66bdb30abd5a5ec9c7521cf39" {
		t.Fatalf("huella canonica distinta: %q, %v", huella, err)
	}
	otroOrden := origen
	otroOrden.RutasPermitidas = []string{"/a", "/z"}
	otroOrden.CamposHandoffPermitidos = []string{"firma", "operacion"}
	if !ConfiguracionesOrigenIguales(origen, otroOrden) {
		t.Fatal("el orden accidental de los conjuntos cambio su identidad")
	}
}

func TestValidacionRechazaDatosDeTarjetaYRutasAmbiguas(t *testing.T) {
	for _, valor := range []string{
		"PAN=4111 1111 1111 1111",
		"cVv:123",
		"４１１１－１１１１－１１１１－１１１１",
	} {
		if !ContieneDatoTarjeta(valor) {
			t.Errorf("no se detecto dato de tarjeta en %q", valor)
		}
	}
	if ContieneDatoTarjeta("referencia opaca de operacion") {
		t.Fatal("un texto ordinario se clasifico como dato de tarjeta")
	}
	for _, ruta := range []string{"//host.externo", "/operaciones/../iniciar", "/operaciones/%2e%2e/iniciar", "/operaciones/iniciar?token=uno"} {
		if RutaHandoffValida(ruta) {
			t.Errorf("se acepto ruta ambigua %q", ruta)
		}
	}
	if !RutaHandoffValida("/operaciones/iniciar") {
		t.Fatal("se rechazo una ruta canonica")
	}
	if !HuellaHMACDeDominioValida("hmac-sha256:peticion-v1:"+strings.Repeat("a", 64), "peticion-v1") ||
		HuellaHMACDeDominioValida("hmac-sha256:otro:"+strings.Repeat("a", 64), "peticion-v1") {
		t.Fatal("la validacion HMAC no quedo ligada a su dominio")
	}
}
