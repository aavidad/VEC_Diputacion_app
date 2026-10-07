package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreparacionConservaHuellaExactaSinAprobacion(t *testing.T) {
	const ruta = "../../data/demo/reglas/cronos-efectos-permisos.configurable.json"
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	if codigo := ejecutar([]string{ruta}, &salida); codigo != 0 {
		t.Fatalf("codigo: %d", codigo)
	}
	var resumen resultado
	if err := json.Unmarshal(salida.Bytes(), &resumen); err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(contenido)
	if resumen.ContenidoSHA256 != hex.EncodeToString(huella[:]) || resumen.Estado != "propuesta_sin_aprobar" || resumen.Reglas != 3 {
		t.Fatalf("resumen: %+v", resumen)
	}
	if codigo := ejecutar([]string{ruta}, escritorFallido{}); codigo != 1 {
		t.Fatalf("salida fallida: %d", codigo)
	}
}

func TestPreparacionNoEmiteResumenParaEntradaRechazada(t *testing.T) {
	dir := t.TempDir()
	invalido := filepath.Join(dir, "invalido.json")
	excesivo := filepath.Join(dir, "excesivo.json")
	for ruta, contenido := range map[string][]byte{invalido: []byte(`{"version_esquema":1}`), excesivo: bytes.Repeat([]byte(" "), 1<<20+1)} {
		if err := os.WriteFile(ruta, contenido, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, argumentos := range [][]string{nil, {invalido, excesivo}, {filepath.Join(dir, "ausente.json")}, {invalido}, {excesivo}} {
		var salida bytes.Buffer
		if ejecutar(argumentos, &salida) == 0 || salida.Len() != 0 {
			t.Fatal("entrada rechazada emitió resumen")
		}
	}
}

type escritorFallido struct{}

func (escritorFallido) Write([]byte) (int, error) { return 0, errors.New("fallo_sintetico") }

func TestRechazoRegistraEtapaSinRutaNiContenido(t *testing.T) {
	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&registro, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })
	var salida bytes.Buffer
	ruta := filepath.Join(t.TempDir(), "dato_privado_sintetico.json")
	if codigo := ejecutar([]string{ruta}, &salida); codigo != 1 || salida.Len() != 0 {
		t.Fatalf("rechazo: codigo=%d resumen=%s", codigo, salida.String())
	}
	if !strings.Contains(registro.String(), "etapa=entrada") || strings.Contains(registro.String(), ruta) || strings.Contains(registro.String(), "dato_privado_sintetico") {
		t.Fatal("el registro no minimizó el rechazo")
	}
}
