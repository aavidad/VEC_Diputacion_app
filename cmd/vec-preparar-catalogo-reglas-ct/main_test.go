package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/reglas"
)

const aprobacionSintetica = "aprobacion:rrhh:ensayo_sintetico"

func fuentePrueba(t *testing.T, ejemplo bool) []byte {
	t.Helper()
	b, err := os.ReadFile("../../data/demo/reglas/ct_reglas.ejemplo.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	if ejemplo {
		return b
	}
	var paquete paqueteCatalogo
	if err := json.Unmarshal(b, &paquete); err != nil {
		t.Fatal(err)
	}
	demostracion := false
	paquete.Fuente.Demostracion = &demostracion
	paquete.Fuente.Aviso = "Fuente de ensayo sintético."
	paquete.Catalogo.FuenteRef = "documento:reglas-ct:ensayo"
	paquete.Catalogo.AprobacionRef = aprobacionSintetica
	b, err = json.Marshal(paquete)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestPrepararGeneraBytesCanonicosYHuellaDelFuenteExacto(t *testing.T) {
	fuente := fuentePrueba(t, false)
	primeros, m, err := preparar(fuente, false, aprobacionSintetica)
	if err != nil {
		t.Fatal(err)
	}
	segundos, repetido, err := preparar(fuente, false, aprobacionSintetica)
	if err != nil || !bytes.Equal(primeros, segundos) || m != repetido {
		t.Fatal("preparación no determinista", err)
	}
	if bytes.HasSuffix(primeros, []byte("\n")) || !json.Valid(primeros) {
		t.Fatal("bytes canónicos inválidos")
	}
	suma := sha256.Sum256(primeros)
	if m.CanonicoSHA256 != hex.EncodeToString(suma[:]) || m.CatalogoID != reglas.CatalogoContratacionTemporal ||
		m.Version != 1 || m.Ejemplo || m.FuenteRef != "documento:reglas-ct:ensayo" {
		t.Fatalf("metadatos incompatibles: %+v", m)
	}
	suma = sha256.Sum256(fuente)
	if m.FuenteSHA256 != hex.EncodeToString(suma[:]) {
		t.Fatal("huella de fuente incorrecta")
	}
}

func TestPrepararRechazaEjemploSalvoEnsayoExplicito(t *testing.T) {
	fuente := fuentePrueba(t, true)
	if _, _, err := preparar(fuente, false, ""); err == nil {
		t.Fatal("ejemplo aceptado para publicación real")
	}
	_, m, err := preparar(fuente, true, "")
	if err != nil || !m.Ejemplo {
		t.Fatal("ejemplo de ensayo no marcado", err)
	}
}

func TestPrepararExigeReferenciaDeAprobacionExactaParaFuenteReal(t *testing.T) {
	fuente := fuentePrueba(t, false)
	for _, ref := range []string{"", "otra:aprobacion", " " + aprobacionSintetica} {
		if _, _, err := preparar(fuente, false, ref); err == nil {
			t.Fatalf("referencia %q aceptada", ref)
		}
	}
}

func TestPrepararRespetaLimitesDelPublicadorCT190(t *testing.T) {
	var paquete paqueteCatalogo
	if err := json.Unmarshal(fuentePrueba(t, false), &paquete); err != nil {
		t.Fatal(err)
	}
	paquete.Catalogo.AprobacionRef = strings.Repeat("a", 201)
	contenido, err := json.Marshal(paquete)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := preparar(contenido, false, paquete.Catalogo.AprobacionRef); err == nil {
		t.Fatal("aprobación superior al límite CT190 aceptada")
	}
	paquete.Catalogo.AprobacionRef = aprobacionSintetica
	paquete.Catalogo.Version = 10_000_000
	paquete.Catalogo.VersionAnteriorRef = paquete.Catalogo.ID + ":9999999"
	contenido, err = json.Marshal(paquete)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := preparar(contenido, false, aprobacionSintetica); err == nil {
		t.Fatal("versión superior al límite CT190 aceptada")
	}
}

func TestPrepararRechazaAmbiguedadYFuenteAdulterada(t *testing.T) {
	fuente := fuentePrueba(t, false)
	casos := [][]byte{
		append(append([]byte(nil), fuente...), []byte("{}")...),
		bytes.Replace(fuente, []byte(`"version_esquema":1`), []byte(`"version_esquema":1,"version_esquema":1`), 1),
		bytes.Replace(fuente, []byte(`"fuente_ref":"documento:reglas-ct:ensayo"`), []byte(`"fuente_ref":"paquete:ejemplo:vec:v1"`), 1),
		bytes.Replace(fuente, []byte(`"estado":"publicado"`), []byte(`"estado":"borrador"`), 1),
		bytes.Replace(fuente, []byte(`"catalogo":`), []byte(`"catalogo_extra":{},"catalogo":`), 1),
		bytes.Replace(fuente, []byte(`"cantidad":"5"`), []byte(`"cantidad":"-1"`), 1),
		bytes.Replace(fuente, []byte(`"version_esquema":1,`), []byte(`"version_esquema":1,"VERSION_ESQUEMA":2,`), 1),
		bytes.Replace(fuente, []byte(`"atributos":{`), []byte(`"ATRIBUTOS":{"cantidad":"50"},"atributos":{`), 1),
	}
	for i, caso := range casos {
		if bytes.Equal(caso, fuente) {
			t.Fatalf("caso %d no alteró fuente", i)
		}
		if _, _, err := preparar(caso, false, aprobacionSintetica); err == nil {
			t.Fatalf("caso %d aceptado", i)
		}
	}
}

func TestEjecutarCreaFicherosPrivadosSinSobrescribir(t *testing.T) {
	if validarRutaSalida("/tmp/catalogo.json") == nil || validarRutaSalida("catalogo.json") == nil {
		t.Fatal("destino fuera del directorio controlado")
	}
	dir, err := os.MkdirTemp(".", "ct190-prueba-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	dir, err = filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := validarRutaSalida(filepath.Join(dir, "inexistente", "salida.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("no se propagó el error del directorio", err)
	}
	fuente := filepath.Join(dir, "fuente.json")
	canonico := filepath.Join(dir, "canonico.json")
	rutaManifiesto := filepath.Join(dir, "manifiesto.json")
	if err := os.WriteFile(fuente, fuentePrueba(t, false), 0600); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if codigo := ejecutar([]string{"-fuente", fuente, "-salida", canonico,
		"-manifiesto", rutaManifiesto, "-aprobacion-ref", "otra:referencia"}, &out, &errs); codigo == 0 {
		t.Fatal("aprobación diferente aceptada")
	}
	if _, err := os.Stat(canonico); !os.IsNotExist(err) {
		t.Fatal("catálogo generado tras rechazo", err)
	}
	args := []string{"-fuente", fuente, "-salida", canonico, "-manifiesto", rutaManifiesto,
		"-aprobacion-ref", aprobacionSintetica}
	out.Reset()
	errs.Reset()
	if codigo := ejecutar(args, &out, &errs); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, errs.String())
	}
	contenido, err := os.ReadFile(canonico)
	if err != nil {
		t.Fatal(err)
	}
	var m manifiesto
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	archivo, err := os.ReadFile(rutaManifiesto)
	if err != nil {
		t.Fatal(err)
	}
	var guardado manifiesto
	if err := json.Unmarshal(archivo, &guardado); err != nil || guardado.AprobacionRef != aprobacionSintetica {
		t.Fatal("manifiesto distinto", err)
	}
	guardado.AprobacionRef = ""
	if guardado != m {
		t.Fatal("resumen distinto")
	}
	suma := sha256.Sum256(contenido)
	if m.CanonicoSHA256 != hex.EncodeToString(suma[:]) {
		t.Fatal("huella de artefacto distinta")
	}
	info, err := os.Stat(canonico)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("permisos de salida incorrectos", err)
	}
	out.Reset()
	errs.Reset()
	if codigo := ejecutar(args, &out, &errs); codigo == 0 || !strings.Contains(errs.String(), "salida_fallida") {
		t.Fatal("sobrescritura aceptada")
	}
}
