// vec-cronos-incidencias consumes only synthetic JSON through bounded stdin.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"os"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoefectos"
	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoincidencias"
	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type entrada struct {
	Snapshot       string `json:"snapshot"`
	Textos         string `json:"textos"`
	Idioma         string `json:"idioma"`
	SnapshotSHA256 string `json:"snapshot_sha256"`
	TextosSHA256   string `json:"textos_sha256"`
}

type lectorFijo struct {
	snapshot ports.SnapshotEnsayoIncidencias
}

func (l lectorFijo) LeerSnapshotEnsayoIncidencias(ctx context.Context) (ports.SnapshotEnsayoIncidencias, error) {
	if err := ctx.Err(); err != nil {
		return ports.SnapshotEnsayoIncidencias{}, err
	}
	return l.snapshot, nil
}

func ejecutar(ctx context.Context, args []string, in io.Reader, out io.Writer) error {
	f := flag.NewFlagSet("vec-cronos-incidencias", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	if f.Parse(args) != nil || f.NArg() != 0 || len(args) != 0 || ctx == nil || in == nil || out == nil {
		return domain.ErrIncidenciasPeriodoInvalido
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	b, err := io.ReadAll(io.LimitReader(in, catalogoincidencias.LimiteJSON+1))
	if err != nil {
		return err
	}
	defer clear(b)
	h := sha256.Sum256(b)
	var e entrada
	if !utf8.Valid(b) || catalogoefectos.DecodificarEstricto(b, hex.EncodeToString(h[:]), &e) != nil {
		return domain.ErrIncidenciasPeriodoInvalido
	}
	c, err := catalogoincidencias.CargarTextos([]byte(e.Textos), e.TextosSHA256, e.Idioma)
	if err != nil {
		return err
	}
	s, err := catalogoincidencias.CargarSnapshot([]byte(e.Snapshot), e.SnapshotSHA256)
	if err != nil {
		return err
	}
	r, err := application.EnsayarIncidencias(ctx, lectorFijo{s})
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(struct {
		ports.ResultadoEnsayoIncidencias
		Idioma       string            `json:"idioma"`
		TextosSHA256 string            `json:"textos_sha256"`
		Textos       map[string]string `json:"textos"`
	}{r, e.Idioma, e.TextosSHA256, c.Textos})
}

func correr(ctx context.Context, args []string, in io.Reader, out, diagnostico io.Writer) int {
	if ejecutar(ctx, args, in, out) != nil {
		// The diagnostic remains a machine code if the text catalogue is invalid.
		if diagnostico == nil || json.NewEncoder(diagnostico).Encode(map[string]string{"error": "entrada_invalida"}) != nil {
			return 2
		}
		return 2
	}
	return 0
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// Standard streams may block inside Read/Write. The process deadline must
	// terminate those operations too; Ctrl-C retains the operating system default.
	stop := context.AfterFunc(ctx, func() { os.Exit(2) })
	code := correr(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	cancel()
	os.Exit(code)
}
