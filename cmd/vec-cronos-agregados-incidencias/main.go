package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"syscall"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoagregadosincidencias"
	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type lectorFijo struct {
	snapshot ports.SnapshotEnsayoAgregadosIncidencias
}

func (l lectorFijo) LeerSnapshotEnsayoAgregadosIncidencias(context.Context) (ports.SnapshotEnsayoAgregadosIncidencias, error) {
	return l.snapshot, nil
}

func leerAcotado(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, os.ErrInvalid
	}
	b, err := io.ReadAll(io.LimitReader(r, catalogoagregadosincidencias.LimiteJSON+1))
	if err != nil || len(b) == 0 || len(b) > catalogoagregadosincidencias.LimiteJSON {
		return nil, os.ErrInvalid
	}
	return b, nil
}
func leerArchivo(ruta string) (b []byte, err error) {
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0) // #nosec G304 -- explicit local fixture path, no leaf symlink; regular, bounded and exact SHA checked.
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := f.Close(); err == nil {
			err = e
		}
	}()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > catalogoagregadosincidencias.LimiteJSON {
		return nil, os.ErrInvalid
	}
	return leerAcotado(f)
}

func ejecutar(ctx context.Context, args []string, entrada io.Reader, salida io.Writer) error {
	f := flag.NewFlagSet("vec-cronos-agregados-incidencias", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	snapshot := f.String("snapshot", "", "")
	shaSnapshot := f.String("snapshot-sha256", "", "")
	textos := f.String("textos", "", "")
	shaTextos := f.String("textos-sha256", "", "")
	idioma := f.String("idioma", "", "")
	if ctx == nil || ctx.Err() != nil || salida == nil || f.Parse(args) != nil || f.NArg() != 0 || *snapshot == "" || *textos == "" || *idioma == "" {
		return os.ErrInvalid
	}
	bt, err := leerArchivo(*textos)
	if err != nil {
		return os.ErrInvalid
	}
	defer clear(bt)
	c, err := catalogoagregadosincidencias.CargarTextos(bt, *shaTextos, *idioma)
	if err != nil {
		return os.ErrInvalid
	}
	var bs []byte
	if *snapshot == "-" {
		bs, err = leerAcotado(entrada)
	} else {
		bs, err = leerArchivo(*snapshot)
	}
	if err != nil {
		return os.ErrInvalid
	}
	defer clear(bs)
	s, err := catalogoagregadosincidencias.CargarSnapshot(bs, *shaSnapshot)
	if err != nil {
		return os.ErrInvalid
	}
	r, err := application.EnsayarAgregadosIncidencias(ctx, lectorFijo{s})
	if err != nil {
		return err
	}
	return json.NewEncoder(salida).Encode(struct {
		ports.ResultadoEnsayoAgregadosIncidencias
		Idioma       string            `json:"idioma"`
		TextosSHA256 string            `json:"textos_sha256"`
		Textos       map[string]string `json:"textos"`
	}{r, *idioma, *shaTextos, c.Textos})
}

func correr(args []string, entrada io.Reader, salida, diagnostico io.Writer) int {
	return correrContexto(context.Background(), args, entrada, salida, diagnostico)
}

func correrContexto(ctx context.Context, args []string, entrada io.Reader, salida, diagnostico io.Writer) int {
	if ejecutar(ctx, args, entrada, salida) != nil {
		if diagnostico != nil {
			if json.NewEncoder(diagnostico).Encode(struct {
				Error  string `json:"error"`
				Estado string `json:"estado"`
				Total  *int   `json:"total_incidencias"`
			}{Error: "entrada_invalida", Estado: "desconocido"}) != nil {
				return 2
			}
		}
		return 2
	}
	return 0
}
func main() {
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	// The process guardian also bounds a blocked stdin or output writer. The
	// default OS interrupt action remains active, including during blocked I/O.
	detener := context.AfterFunc(ctx, func() { os.Exit(2) })
	codigo := correrContexto(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	detener()
	cancelar()
	os.Exit(codigo)
}
