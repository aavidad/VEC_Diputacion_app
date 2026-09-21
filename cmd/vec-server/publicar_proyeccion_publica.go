package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"strings"

	"vec-diputacion-granada/config"
	postgrespublico "vec-diputacion-granada/internal/modules/bolsa/adapters/postgrespublico"
	canonicopublico "vec-diputacion-granada/internal/modules/bolsa/publico/canonico"
)

var errPublicacionPublicaCLI = errors.New("publicacion publica rechazada")

type argumentosPublicarProyeccionPublica struct {
	proyeccionV2 string
	manifiestoV2 string
	bolsasV1     string
}

type publicadorProyeccionPublica func(context.Context, string, []byte, []byte, string) error

func leerArgumentosPublicarProyeccionPublica(args []string) (argumentosPublicarProyeccionPublica, error) {
	f := flag.NewFlagSet("publicar-proyeccion-publica", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var a argumentosPublicarProyeccionPublica
	f.StringVar(&a.proyeccionV2, "proyeccion-v2", "", "")
	f.StringVar(&a.manifiestoV2, "manifiesto-v2", "", "")
	f.StringVar(&a.bolsasV1, "bolsas-v1", "", "")
	if f.Parse(args) != nil {
		return argumentosPublicarProyeccionPublica{}, errPublicacionPublicaCLI
	}
	a.proyeccionV2 = strings.TrimSpace(a.proyeccionV2)
	a.manifiestoV2 = strings.TrimSpace(a.manifiestoV2)
	a.bolsasV1 = strings.TrimSpace(a.bolsasV1)
	if a.proyeccionV2 == "" || a.manifiestoV2 == "" || a.bolsasV1 == "" || f.NArg() != 0 {
		return argumentosPublicarProyeccionPublica{}, errPublicacionPublicaCLI
	}
	return a, nil
}

func ejecutarPublicacionProyeccionPublica(
	ctx context.Context,
	args []string,
	salida io.Writer,
	cfg config.Config,
	publicar publicadorProyeccionPublica,
) error {
	if ctx == nil || salida == nil || publicar == nil {
		return errPublicacionPublicaCLI
	}
	a, err := leerArgumentosPublicarProyeccionPublica(args)
	if err != nil {
		return errPublicacionPublicaCLI
	}
	proyeccionV2, err := leerFicheroLimitado(a.proyeccionV2, 256*1024*1024)
	if err != nil {
		return errPublicacionPublicaCLI
	}
	manifiestoV2, err := leerFicheroLimitado(a.manifiestoV2, 256*1024*1024)
	if err != nil {
		return errPublicacionPublicaCLI
	}
	bolsasV1, err := leerFicheroLimitado(a.bolsasV1, 64*1024*1024)
	if err != nil {
		return errPublicacionPublicaCLI
	}
	material, err := canonicopublico.PrepararMaterialPublicacionV3(proyeccionV2, manifiestoV2, bolsasV1)
	if err != nil {
		return errPublicacionPublicaCLI
	}
	dsn, err := cfg.BolsaPublicaPostgreSQL.DSN()
	if err != nil {
		return errPublicacionPublicaCLI
	}
	if err := publicar(ctx, dsn, material.ProyeccionV2, material.BolsasV1, material.AnclaManifiestoSHA256); err != nil {
		return errPublicacionPublicaCLI
	}
	_, _ = io.WriteString(salida, "publicacion_publica ancla="+material.AnclaManifiestoSHA256+"\n")
	return nil
}

func leerFicheroLimitado(ruta string, maximo int64) ([]byte, error) {
	if ruta == "" || maximo < 1 {
		return nil, errPublicacionPublicaCLI
	}
	fichero, err := os.Open(ruta)
	if err != nil {
		return nil, errPublicacionPublicaCLI
	}
	defer fichero.Close()
	contenido, err := io.ReadAll(io.LimitReader(fichero, maximo+1))
	if err != nil || int64(len(contenido)) > maximo {
		return nil, errPublicacionPublicaCLI
	}
	return contenido, nil
}

func publicarProyeccionPublicaPostgreSQL(ctx context.Context, dsn string, proyeccionV2, bolsasV1 []byte, ancla string) error {
	return postgrespublico.PublicarProyeccionV3(ctx, dsn, proyeccionV2, bolsasV1, ancla)
}
