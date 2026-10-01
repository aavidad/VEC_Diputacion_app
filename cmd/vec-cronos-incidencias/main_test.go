package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoincidencias"
	"vec-diputacion-granada/internal/modules/cronos/domain"
)

func shaCLITest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func entradaCLITest(t *testing.T, idioma string) ([]byte, entrada) {
	t.Helper()
	snapshot, err := os.ReadFile("../../data/demo/cronos/incidencias-periodo.json")
	if err != nil {
		t.Fatal(err)
	}
	textos, err := os.ReadFile("../../web/static/textos/" + idioma + "/cronos-incidencias-ensayo.json")
	if err != nil {
		t.Fatal(err)
	}
	e := entrada{Snapshot: string(snapshot), Textos: string(textos), Idioma: idioma, SnapshotSHA256: shaCLITest(snapshot), TextosSHA256: shaCLITest(textos)}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return b, e
}
func TestCLIIncidenciasEjemploESENConCorteYCobertura(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			b, e := entradaCLITest(t, idioma)
			var out, diagnostico bytes.Buffer
			if code := correr(context.Background(), nil, bytes.NewReader(b), &out, &diagnostico); code != 0 {
				t.Fatalf("code %d: %s", code, diagnostico.String())
			}
			var r struct {
				Idioma         string                          `json:"idioma"`
				SnapshotSHA256 string                          `json:"snapshot_sha256"`
				TextosSHA256   string                          `json:"textos_sha256"`
				Textos         map[string]string               `json:"textos"`
				Dias           []domain.RegistroDiaIncidencias `json:"dias"`
			}
			if err := json.Unmarshal(out.Bytes(), &r); err != nil {
				t.Fatal(err)
			}
			if r.Idioma != idioma || r.SnapshotSHA256 != e.SnapshotSHA256 || r.TextosSHA256 != e.TextosSHA256 || r.Textos["registro_incompleto"] == "" || len(r.Dias) != 6 {
				t.Fatalf("lost identity or source: %+v", r)
			}
			want := []string{"registrado", "registro_incompleto", "sin_registros", "registro_incompleto", "no_evaluado", "no_evaluado"}
			for i, w := range want {
				if r.Dias[i].Estado != w {
					t.Fatalf("day %d: %s want %s", i, r.Dias[i].Estado, w)
				}
			}
			if r.Dias[4].HechosRegistrados != nil || r.Dias[5].EvaluadoHastaUTC != nil {
				t.Fatal("future extrapolated")
			}
		})
	}
}
func TestCLIIncidenciasRechazaSinEmitirResultado(t *testing.T) {
	b, e := entradaCLITest(t, "es")
	badHash := e
	badHash.SnapshotSHA256 = strings.Repeat("0", 64)
	badHashBytes, _ := json.Marshal(badHash)
	missing := e
	missing.Textos = ""
	missingBytes, _ := json.Marshal(missing)
	cases := map[string][]byte{"bad hash": badHashBytes, "missing catalogue": missingBytes, "duplicate": append([]byte(`{"idioma":"es",`), b[1:]...), "unknown": append([]byte(`{"unknown":true,`), b[1:]...), "trailing": append(append([]byte{}, b...), []byte(`{}`)...), "oversize": []byte(strings.Repeat(" ", catalogoincidencias.LimiteJSON+1)), "bad UTF8": bytes.Replace(b, []byte(`demo:snapshot`), []byte{0xff}, 1)}
	for name, bad := range cases {
		t.Run(name, func(t *testing.T) {
			var out, diagnostico bytes.Buffer
			if code := correr(context.Background(), nil, bytes.NewReader(bad), &out, &diagnostico); code != 2 || out.Len() != 0 || !strings.Contains(diagnostico.String(), "entrada_invalida") {
				t.Fatalf("invalid result: %d %s %s", code, out.String(), diagnostico.String())
			}
		})
	}
	var out bytes.Buffer
	if ejecutar(context.Background(), []string{"path"}, bytes.NewReader(b), &out) == nil || out.Len() != 0 {
		t.Fatal("accepted path arguments")
	}
}

type falloIOIncidencias struct{ err error }

func (f falloIOIncidencias) Read([]byte) (int, error)  { return 0, f.err }
func (f falloIOIncidencias) Write([]byte) (int, error) { return 0, f.err }
func TestCLIIncidenciasPropagaLecturaEscrituraYCancelacion(t *testing.T) {
	b, _ := entradaCLITest(t, "es")
	causa := errors.New("synthetic IO failure")
	if err := ejecutar(context.Background(), nil, falloIOIncidencias{causa}, io.Discard); !errors.Is(err, causa) {
		t.Fatalf("lost read error: %v", err)
	}
	if err := ejecutar(context.Background(), nil, bytes.NewReader(b), falloIOIncidencias{causa}); !errors.Is(err, causa) {
		t.Fatalf("lost write error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ejecutar(ctx, nil, bytes.NewReader(b), io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatalf("lost cancellation: %v", err)
	}
	if code := correr(context.Background(), nil, falloIOIncidencias{causa}, io.Discard, falloIOIncidencias{causa}); code != 2 {
		t.Fatal("diagnostic failure reported success")
	}
}
