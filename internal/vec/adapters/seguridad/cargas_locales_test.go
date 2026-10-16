package seguridad

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestHMACSubidaLocalReinicioDominiosYRedaccion(t *testing.T) {
	correlacion := []byte("12345678901234567890123456789012")
	seudonimo := []byte("22345678901234567890123456789012")
	solicitud := []byte("32345678901234567890123456789012")
	if !RaicesHMACSubidaLocalDistintas(correlacion, seudonimo, solicitud) ||
		RaicesHMACSubidaLocalDistintas(correlacion, correlacion, solicitud) {
		t.Fatal("no separó las tres raíces")
	}
	uno, err := NuevoSeudonimizadorSujetoSubidaLocal(seudonimo)
	if err != nil {
		t.Fatal(err)
	}
	dos, err := NuevoSeudonimizadorSujetoSubidaLocal(seudonimo)
	if err != nil {
		t.Fatal(err)
	}
	sellador, err := NuevoSelladorSolicitudSubidaLocal(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	q, err := ports.NuevaSolicitudSeudonimizarSujetoAlmacen("per_ficticia", "subida_local:ref:carga")
	if err != nil {
		t.Fatal(err)
	}
	refUno, err := uno.SeudonimizarSujetoAlmacen(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	refDos, err := dos.SeudonimizarSujetoAlmacen(context.Background(), q)
	if err != nil || refUno != refDos || !strings.HasPrefix(refUno, "hmac-sha256:local_seudonimo_v1:") {
		t.Fatalf("seudónimo cambió tras reiniciar: %v", err)
	}
	h1, err := sellador.SellarSolicitudCargaDocumental(context.Background(), []byte("orden uno"))
	if err != nil {
		t.Fatal(err)
	}
	h2, err := sellador.SellarSolicitudCargaDocumental(context.Background(), []byte("orden dos"))
	if err != nil || h1 == h2 || h1 == refUno {
		t.Fatal("peticiones y dominios confundidos")
	}
	contenido, err := json.Marshal(sellador)
	if err != nil || strings.Contains(string(contenido), "323456") || strings.Contains(fmt.Sprintf("%+v", uno), "223456") {
		t.Fatal("clave revelada por formato")
	}
}

func TestHMACSubidaLocalRechazaRaicesInvalidas(t *testing.T) {
	if _, err := NuevoSeudonimizadorSujetoSubidaLocal(make([]byte, 32)); err == nil {
		t.Fatal("aceptó raíz nula")
	}
	if _, err := NuevoSelladorSolicitudSubidaLocal([]byte("corta")); err == nil {
		t.Fatal("aceptó raíz corta")
	}
	var nulo *SelladorSolicitudSubidaLocal
	if _, err := nulo.SellarSolicitudCargaDocumental(context.Background(), []byte("orden")); err == nil {
		t.Fatal("aceptó receptor nulo")
	}
}
