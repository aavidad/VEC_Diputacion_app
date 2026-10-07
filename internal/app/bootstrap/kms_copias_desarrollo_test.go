package bootstrap

import (
	"bytes"
	"context"
	"testing"
	app "vec-diputacion-granada/internal/modules/administracion/application/destinocopias"
	ports "vec-diputacion-granada/internal/modules/administracion/ports/destinocopias"
)

func TestCopiasUsaProveedorCargadoYSubclaveDedicada(t *testing.T) {
	emisor, _, _ := nuevosProveedoresKMSPrueba(t)
	c := &ComposicionSeguridadDesarrollo{emisorKMS: emisor}
	p, e := c.ProtectorCopiasDesarrollo("clave:copias:fixture", "v1", 4096)
	if e != nil {
		t.Fatal(e)
	}
	var maestra [32]byte
	for i := range maestra {
		maestra[i] = byte(i + 1)
	}
	offline, e := NuevoProtectorCopiasDesarrollo(maestra, "clave:copias:fixture", "v1", 4096)
	if e != nil {
		t.Fatal(e)
	}
	v := ports.Vinculo{ConjuntoRef: "conjunto:fixture", ComponenteRef: "componente:fixture", Posicion: 1, ManifiestoSHA256: app.Huella([]byte("fixture"))}
	claro := []byte("contenido sintetico")
	b, e := p.Proteger(context.Background(), v, claro)
	if e != nil {
		t.Fatal(e)
	}
	back, e := offline.Recuperar(context.Background(), v, b)
	if e != nil || !bytes.Equal(back, claro) {
		t.Fatalf("KMS composition: %v", e)
	}
	wrongVersion, _ := NuevoProtectorCopiasDesarrollo(maestra, "clave:copias:fixture", "v2", 4096)
	if _, e = wrongVersion.Recuperar(context.Background(), v, b); e == nil {
		t.Fatal("key version not authenticated")
	}
	if _, e = NuevoProtectorCopiasDesarrollo([32]byte{}, "clave:copias:fixture", "v1", 4096); e == nil {
		t.Fatal("zero key")
	}
	other := derivarClaveDesarrollo(emisor.claveEnvoltura, dominioCopiasDesarrollo)
	if bytes.Equal(other[:], emisor.claveEnvoltura[:]) {
		t.Fatal("no key separation")
	}
}
