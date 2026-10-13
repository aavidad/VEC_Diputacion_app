package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
)

type salidaError struct {
	Codigo    string `json:"codigo"`
	Estado    string `json:"estado"`
	SQLState  string `json:"sqlstate"`
	Rutina    string `json:"rutina"`
	Clave     string `json:"clave,omitempty"`
	Observado string `json:"observado,omitempty"`
	Esperado  string `json:"esperado,omitempty"`
}

var errDenegado = errors.New("CA42")

func escribirError(err error) error {
	var e *contextopg.ErrorPublicacionVinculoF1
	if !errors.As(err, &e) {
		e = &contextopg.ErrorPublicacionVinculoF1{SQLState: "XX000", Routine: "cli"}
	}
	return json.NewEncoder(os.Stdout).Encode(salidaError{Codigo: "CA42", Estado: "error", SQLState: e.SQLState, Rutina: e.Routine,
		Clave: e.Clave, Observado: e.Observado, Esperado: e.Esperado})
}

func ejecutar(args []string) error {
	var s contextopg.SolicitudPublicacionVinculoF1
	var comprobar, aplicar bool
	var huella string
	fs := flag.NewFlagSet("vec-publicar-vinculo-corporativo", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&comprobar, "comprobar", false, "")
	fs.BoolVar(&aplicar, "aplicar", false, "")
	fs.StringVar(&s.OperacionRef, "operacion", "", "")
	fs.StringVar(&s.CuentaRef, "cuenta", "", "")
	fs.StringVar(&s.PersonaRef, "persona", "", "")
	fs.StringVar(&s.PerfilRef, "perfil", "", "")
	fs.StringVar(&s.VinculoContextoRef, "contexto", "", "")
	fs.StringVar(&s.VinculoCorporativoRef, "vinculo", "", "")
	fs.StringVar(&s.OrganizacionRef, "organizacion", "", "")
	fs.StringVar(&huella, "huella", "", "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || comprobar == aplicar ||
		os.Getenv("VEC_VINCULO_CORPORATIVO_DSN") == "" {
		return &contextopg.ErrorPublicacionVinculoF1{SQLState: "22023", Routine: "configuracion"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	config, err := pgxpool.ParseConfig(os.Getenv("VEC_VINCULO_CORPORATIVO_DSN"))
	if err != nil {
		return &contextopg.ErrorPublicacionVinculoF1{SQLState: "08000", Routine: "conexion"}
	}
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return &contextopg.ErrorPublicacionVinculoF1{SQLState: "08000", Routine: "conexion"}
	}
	defer pool.Close()
	resultado, err := contextopg.EjecutarPublicacionVinculoF1(ctx, pool, s, aplicar, huella)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(os.Stdout).Encode(resultado); err != nil {
		return &contextopg.ErrorPublicacionVinculoF1{SQLState: "58030", Routine: "salida"}
	}
	if resultado.Estado == "denegado" {
		return errDenegado
	}
	return nil
}

func main() {
	if err := ejecutar(os.Args[1:]); err != nil {
		if !errors.Is(err, errDenegado) {
			if fallo := escribirError(err); fallo != nil {
				os.Exit(2)
			}
		}
		os.Exit(1)
	}
}
