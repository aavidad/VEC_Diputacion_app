package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"syscall"

	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoefectos"
	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type lectorFijo struct {
	s ports.SnapshotEnsayoSaldoPermisos
}

func (l lectorFijo) LeerSnapshotSaldoPermisos(context.Context) (ports.SnapshotEnsayoSaldoPermisos, error) {
	return l.s, nil
}

func leer(ruta string) (b []byte, err error) {
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0) // #nosec G304 -- ruta explícita; regular acotado, sin enlaces/FIFO, SHA exacto obligatorio.
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := f.Close(); err == nil {
			err = e
		}
	}()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > catalogoefectos.LimiteJSON {
		return nil, os.ErrInvalid
	}
	b, err = io.ReadAll(io.LimitReader(f, catalogoefectos.LimiteJSON+1))
	return b, err
}

func ejecutar(args []string, salida io.Writer) error {
	f := flag.NewFlagSet("vec-cronos-ensayo", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	politica := f.String("catalogo", "", "")
	shaPolitica := f.String("catalogo-sha256", "", "")
	escenarios := f.String("escenarios", "", "")
	shaEscenarios := f.String("escenarios-sha256", "", "")
	idioma := f.String("idioma", "", "")
	if f.Parse(args) != nil || f.NArg() != 0 || *politica == "" || *escenarios == "" || *idioma == "" {
		return os.ErrInvalid
	}
	b, err := leer(*politica)
	if err != nil {
		return os.ErrInvalid
	}
	defer clear(b)
	c, err := catalogoefectos.Cargar(b, *shaPolitica)
	if err != nil || c.Textos[*idioma] == nil {
		return os.ErrInvalid
	}
	bEscenarios, err := leer(*escenarios)
	if err != nil {
		return os.ErrInvalid
	}
	defer clear(bEscenarios)
	var s ports.SnapshotEnsayoSaldoPermisos
	if catalogoefectos.DecodificarEstricto(bEscenarios, *shaEscenarios, &s) != nil {
		return os.ErrInvalid
	}
	s.SHA256 = *shaEscenarios
	r, err := application.EnsayarSaldoPermisos(context.Background(), lectorFijo{s}, c.Politica)
	if err != nil {
		return os.ErrInvalid
	}
	return json.NewEncoder(salida).Encode(struct {
		ports.ResultadoEnsayoSaldoPermisos
		Aviso  string            `json:"aviso"`
		Textos map[string]string `json:"textos"`
	}{r, c.Textos[*idioma]["aviso"], c.Textos[*idioma]})
}

func main() {
	if err := ejecutar(os.Args[1:], os.Stdout); err != nil {
		if json.NewEncoder(os.Stderr).Encode(map[string]string{"error": "entrada_invalida"}) != nil {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
