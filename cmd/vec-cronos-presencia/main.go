package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"syscall"
	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogopresencia"
	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type lectorFijo struct{ snapshot ports.SnapshotEnsayoPresencia }

func (l lectorFijo) LeerSnapshotEnsayoPresencia(context.Context) (ports.SnapshotEnsayoPresencia, error) {
	return l.snapshot, nil
}

func leer(ruta string) (b []byte, err error) {
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0) // #nosec G304 -- explicit fixture path; nonblocking, no leaf symlink, regular and bounded; exact SHA required.
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := f.Close(); err == nil {
			err = e
		}
	}()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > catalogopresencia.LimiteJSON {
		return nil, os.ErrInvalid
	}
	b, err = io.ReadAll(io.LimitReader(f, catalogopresencia.LimiteJSON+1))
	if err != nil || len(b) > catalogopresencia.LimiteJSON {
		return nil, os.ErrInvalid
	}
	return b, nil
}

func ejecutar(args []string, salida io.Writer) error {
	f := flag.NewFlagSet("vec-cronos-presencia", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	snapshot := f.String("snapshot", "", "")
	shaSnapshot := f.String("snapshot-sha256", "", "")
	textos := f.String("textos", "", "")
	shaTextos := f.String("textos-sha256", "", "")
	idioma := f.String("idioma", "", "")
	if f.Parse(args) != nil || f.NArg() != 0 || *snapshot == "" || *textos == "" || *idioma == "" {
		return os.ErrInvalid
	}
	bt, err := leer(*textos)
	if err != nil {
		return os.ErrInvalid
	}
	defer clear(bt)
	c, err := catalogopresencia.CargarTextos(bt, *shaTextos, *idioma)
	if err != nil {
		return os.ErrInvalid
	}
	bs, err := leer(*snapshot)
	if err != nil {
		return os.ErrInvalid
	}
	defer clear(bs)
	s, err := catalogopresencia.CargarSnapshot(bs, *shaSnapshot)
	if err != nil {
		return os.ErrInvalid
	}
	r, err := application.EnsayarPresencia(context.Background(), lectorFijo{s})
	if err != nil {
		return os.ErrInvalid
	}
	return json.NewEncoder(salida).Encode(struct {
		ports.ResultadoEnsayoPresencia
		Idioma       string            `json:"idioma"`
		TextosSHA256 string            `json:"textos_sha256"`
		Textos       map[string]string `json:"textos"`
	}{r, *idioma, *shaTextos, c.Textos})
}

func correr(args []string, salida, diagnostico io.Writer) int {
	if ejecutar(args, salida) != nil {
		// A machine code is stable even when the catalogue itself cannot be read.
		if json.NewEncoder(diagnostico).Encode(map[string]string{"error": "entrada_invalida"}) != nil {
			return 2
		}
		return 2
	}
	return 0
}
func main() { os.Exit(correr(os.Args[1:], os.Stdout, os.Stderr)) }
