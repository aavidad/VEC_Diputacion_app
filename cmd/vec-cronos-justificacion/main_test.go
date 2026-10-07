package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogojustificacion"
	"vec-diputacion-granada/internal/modules/cronos/domain"
)

func sha(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func argumentos(t *testing.T, idioma string) []string {
	t.Helper()
	rutas := []string{
		"../../data/demo/cronos/justificacion-escenario.json",
		"../../data/demo/cronos/justificacion-politica.json",
		"../../web/static/textos/" + idioma + "/cronos-justificacion-ensayo.json",
	}
	claves := []string{"escenario", "politica", "textos"}
	args := make([]string, 0, 12)
	for i, ruta := range rutas {
		b, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		args = append(args, "-"+claves[i], ruta, "-"+claves[i]+"-sha256", sha(b))
	}
	return args
}

func TestEnsayoAceptacionYRechazoSinRecibo(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			var salida, errores bytes.Buffer
			if codigo := run(argumentos(t, idioma), &salida, &errores); codigo != 0 || errores.Len() != 0 {
				t.Fatalf("codigo %d, error %s", codigo, errores.String())
			}
			var got resultado
			if err := json.Unmarshal(salida.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if !got.Demostracion || got.Aviso == "" || len(got.Escenarios) != 2 ||
				got.Escenarios[0].Anexo.Estado != domain.JustificacionPendiente || got.Escenarios[0].Anexo.Version != 1 ||
				got.Escenarios[0].Revision.Estado != domain.JustificacionAceptada || got.Escenarios[0].Revision.Version != 2 ||
				got.Escenarios[1].Revision.Estado != domain.JustificacionRechazada || got.Escenarios[1].Revision.Version != 2 ||
				strings.Contains(salida.String(), "recibo") || strings.Contains(salida.String(), "registrada") {
				t.Fatal("resultado del ensayo incoherente")
			}
		})
	}
}

func TestEntradaInvalidaSinDatos(t *testing.T) {
	base := argumentos(t, "es")
	casos := [][]string{nil, append(append([]string{}, base...), "sobra")}
	wrongSHA := append([]string{}, base...)
	wrongSHA[3] = strings.Repeat("0", 64)
	casos = append(casos, wrongSHA)
	original, err := os.ReadFile(base[1])
	if err != nil {
		t.Fatal(err)
	}
	var conCambio catalogojustificacion.Escenarios
	if err := json.Unmarshal(original, &conCambio); err != nil {
		t.Fatal(err)
	}
	vinculoCambiado := conCambio.Escenarios[0].Vinculo
	vinculoCambiado.Documento.ID = "ref:" + strings.Repeat("9", 64)
	conCambio.Escenarios[0].VinculoRevision = &vinculoCambiado
	cambio, err := json.Marshal(conCambio)
	if err != nil {
		t.Fatal(err)
	}
	var politica catalogojustificacion.Politica
	bPolitica, err := os.ReadFile(base[5])
	if err != nil || json.Unmarshal(bPolitica, &politica) != nil {
		t.Fatal("politica de prueba invalida")
	}
	p := domain.PoliticaJustificacion{Referencia: politica.Referencia, Version: politica.Version, SHA256: sha(bPolitica), CatalogoVersionRef: politica.CatalogoVersionRef, PermisoRef: politica.PermisoRef, TipoDocumentalRef: politica.TipoDocumentalRef, CustodioID: politica.CustodioID, MotivosRef: politica.MotivosRef}
	x := conCambio.Escenarios[0]
	anexo, err := domain.PrepararAnexoJustificacion(x.Solicitud, p, nil, x.Vinculo, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.PrepararRevisionJustificacion(x.Solicitud, p, anexo, vinculoCambiado, 1, x.Decision, x.MotivoRef); err != domain.ErrJustificacionConflicto {
		t.Fatal("el cambio de referencia debe crear conflicto")
	}
	for nombre, contenido := range map[string][]byte{
		"duplicada":           bytes.Replace(original, []byte(`"version_esquema": 1,`), []byte(`"version_esquema": 1, "version_esquema": 1,`), 1),
		"otro_documento":      bytes.Replace(original, []byte(`"version_esperada": 1`), []byte(`"version_esperada": 2`), 1),
		"referencia_cambiada": cambio,
	} {
		t.Run(nombre, func(t *testing.T) {
			ruta := filepath.Join(t.TempDir(), "escenario.json")
			if err := os.WriteFile(ruta, contenido, 0600); err != nil {
				t.Fatal(err)
			}
			args := append([]string{}, base...)
			args[1], args[3] = ruta, sha(contenido)
			var out, diag bytes.Buffer
			if code := run(args, &out, &diag); code != 2 || out.Len() != 0 || diag.String() != "{\"codigo\":\"entrada_invalida\"}\n" {
				t.Fatal("fallo sin diagnóstico mínimo")
			}
		})
	}
	for _, args := range casos {
		var out, diag bytes.Buffer
		if code := run(args, &out, &diag); code != 2 || out.Len() != 0 || diag.String() != "{\"codigo\":\"entrada_invalida\"}\n" {
			t.Fatal("fallo sin diagnóstico mínimo")
		}
	}
}

func TestRechazaEnlaceYFIFO(t *testing.T) {
	dir := t.TempDir()
	enlace := filepath.Join(dir, "enlace")
	if err := os.Symlink("ausente", enlace); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{enlace, fifo} {
		if _, err := leer(ruta); err == nil {
			t.Fatal("archivo no regular admitido")
		}
	}
}
