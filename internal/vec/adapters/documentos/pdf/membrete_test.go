package pdf

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

var contenidoMembrete = domain.ContenidoDocumento{
	Titulo:   "Resolución de prueba",
	Parrafos: []string{"Primer párrafo inventado.", "Segundo párrafo con eñe y €."},
}

// huellaSinMembrete fija los bytes nuevos sin logo, con PDF 1.4 y /ID.
// Los documentos ya custodiados conservan sus bytes y huellas anteriores.
const huellaSinMembrete = "0072debb0faa108876591ea64318cfd60e69ed651d13d938577210e0f60da605"
const huellaSinMembreteAnterior = "3d38ccd3da2c1b96e9bba7767857a5396cfc97009435f49b12ef6244ef328baf"

func TestRenderizadorSinMembreteConservaLosBytesAnteriores(t *testing.T) {
	datos, err := Renderizador{}.Renderizar(context.Background(), contenidoMembrete)
	if err != nil {
		t.Fatalf("Renderizar() error = %v", err)
	}
	suma := sha256.Sum256(datos)
	if got := hex.EncodeToString(suma[:]); got != huellaSinMembrete {
		t.Fatalf("la salida sin membrete cambió: %s", got)
	}
	// Al retirar únicamente la cabecera 1.4 y /ID, todos los bytes anteriores,
	// incluido el contenido visible y su estructura, siguen idénticos.
	lineaID := regexp.MustCompile(`/ID \[<[0-9a-f]{32}> <[0-9a-f]{32}>\]\n`)
	if len(lineaID.FindAll(datos, -1)) != 1 {
		t.Fatal("el PDF debe contener un único /ID de dos elementos")
	}
	anterior := lineaID.ReplaceAll(datos, nil)
	anterior[len("%PDF-1.")] = '3'
	sumaAnterior := sha256.Sum256(anterior)
	if got := hex.EncodeToString(sumaAnterior[:]); got != huellaSinMembreteAnterior {
		t.Fatalf("cambiaron bytes ajenos a la cabecera y /ID: %s", got)
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
