package pdf

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

var contenidoMembrete = domain.ContenidoDocumento{
	Titulo:   "Resolución de prueba",
	Parrafos: []string{"Primer párrafo inventado.", "Segundo párrafo con eñe y €."},
}

// huellaSinMembrete es la salida del renderizador antes de existir el
// membrete. Si cambia, cambian las huellas guardadas de documentos ya
// custodiados: no se actualiza sin revisar a todos los consumidores.
const huellaSinMembrete = "3d38ccd3da2c1b96e9bba7767857a5396cfc97009435f49b12ef6244ef328baf"

func TestRenderizadorSinMembreteConservaLosBytesAnteriores(t *testing.T) {
	datos, err := Renderizador{}.Renderizar(context.Background(), contenidoMembrete)
	if err != nil {
		t.Fatalf("Renderizar() error = %v", err)
	}
	suma := sha256.Sum256(datos)
	if got := hex.EncodeToString(suma[:]); got != huellaSinMembrete {
		t.Fatalf("la salida sin membrete cambió: %s", got)
	}
	if bytes.Contains(datos, []byte("/Subtype /Image")) {
		t.Fatal("sin membrete no debe haber imágenes")
	}
}

func TestRenderizadorConMembreteIncluyeLogoYPasaValidador(t *testing.T) {
	r := Renderizador{Membrete: true}
	datos, err := r.Renderizar(context.Background(), contenidoMembrete)
	if err != nil {
		t.Fatalf("Renderizar() error = %v", err)
	}
	if !bytes.Contains(datos, []byte("/Subtype /Image")) {
		t.Fatal("el PDF con membrete no contiene la imagen del logotipo")
	}
	if err := r.ValidarSalida(context.Background(), datos); err != nil {
		t.Fatalf("ValidarSalida() error = %v", err)
	}
	repetido, err := r.Renderizar(context.Background(), contenidoMembrete)
	if err != nil || !bytes.Equal(datos, repetido) {
		t.Fatalf("el PDF con membrete no es determinista: %v", err)
	}
	sin, _ := Renderizador{}.Renderizar(context.Background(), contenidoMembrete)
	if bytes.Equal(datos, sin) {
		t.Fatal("el membrete no cambió la salida")
	}
}
