package bootstrap

import (
	"context"
	"strings"
	"testing"

	ports "vec-diputacion-granada/internal/modules/administracion/ports/destinocopias"
)

func TestOrdenCopiasNoAceptaCifradoDeComponentes(t *testing.T) {
	var maestra [32]byte
	for i := range maestra {
		maestra[i] = byte(i + 1)
	}
	orden, err := NuevoProtectorOrdenCopiasDesarrollo(maestra, "clave:sintetica", "version:1", 4096)
	if err != nil {
		t.Fatal(err)
	}
	componentes, err := NuevoProtectorCopiasDesarrollo(maestra, "clave:sintetica", "version:1", 4096)
	if err != nil {
		t.Fatal(err)
	}
	vinculo := ports.Vinculo{ConjuntoRef: "orden:sintetica", ComponenteRef: "componente:sintetico", Posicion: 1, ManifiestoSHA256: strings.Repeat("a", 64)}
	ctx := context.Background()
	claro := []byte(`{"operacion_ref":"operacion:sintetica"}`)
	cifradoOrden, err := orden.Proteger(ctx, vinculo, claro)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := componentes.Recuperar(ctx, vinculo, cifradoOrden); err == nil {
		t.Fatal("se aceptó una orden como componente")
	}
	cifradoComponente, err := componentes.Proteger(ctx, vinculo, claro)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orden.Recuperar(ctx, vinculo, cifradoComponente); err == nil {
		t.Fatal("se aceptó un componente como orden")
	}
	// El proveedor ya cargado y el ejecutor externo recuperan la misma orden.
	composicion := &ComposicionSeguridadDesarrollo{emisorKMS: &emisorKMSDesarrollo{claveEnvoltura: derivarClaveDesarrollo(maestra, "vec.kms.desarrollo.envoltura.v1")}}
	proveedor, err := composicion.ProtectorOrdenCopiasDesarrollo("clave:sintetica", "version:1", 4096)
	if err != nil {
		t.Fatal(err)
	}
	recuperado, err := proveedor.Recuperar(ctx, vinculo, cifradoOrden)
	if err != nil || string(recuperado) != string(claro) {
		t.Fatalf("orden no recuperada: %v", err)
	}
	clear(recuperado)
}

func TestOrdenCopiasSinMaterialNoCreaProveedor(t *testing.T) {
	if _, err := NuevoProtectorOrdenCopiasDesarrollo([32]byte{}, "clave:sintetica", "version:1", 4096); err == nil {
		t.Fatal("material ausente aceptado")
	}
	var composicion *ComposicionSeguridadDesarrollo
	if _, err := composicion.ProtectorOrdenCopiasDesarrollo("clave:sintetica", "version:1", 4096); err == nil {
		t.Fatal("composición ausente aceptada")
	}
}
