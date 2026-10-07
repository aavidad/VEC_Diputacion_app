// Command vec-cargos-ct executes a reviewed private central provision plan.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/vec/adapters/postgres/cargosct"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	os.Exit(ejecutar(ctx, os.Args[1:], os.Stdout))
}
func ejecutar(ctx context.Context, args []string, out io.Writer) int {
	f := flag.NewFlagSet("vec-cargos-ct", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	fuente := f.String("fuente", "", "")
	dsnFile := f.String("dsn-archivo", "", "")
	op := f.String("operacion", "", "")
	recursoOp := f.String("recurso-operacion", "preparar", "")
	if ctx == nil || f.Parse(args) != nil || f.NArg() != 0 || *fuente == "" {
		return fallo(out, 2)
	}
	b, err := bootstrap.LeerMaterialProvisionExterna(*fuente, 1<<20)
	if err != nil {
		return fallo(out, 2)
	}
	defer clear(b)
	if *op == "recurso" {
		plan, err := cargosct.LeerPlan(b)
		if err != nil {
			return fallo(out, 2)
		}
		r, err := plan.Recurso(*recursoOp)
		if err != nil {
			return fallo(out, 2)
		}
		if json.NewEncoder(out).Encode(r) != nil {
			return 1
		}
		return 0
	}
	s, err := cargosct.LeerSolicitud(b)
	if err != nil {
		return fallo(out, 2)
	}
	switch *op {
	case "preparar", "aprobar", "aplicar", "recuperar":
	default:
		return fallo(out, 2)
	}
	if *dsnFile == "" {
		return fallo(out, 2)
	}
	db, err := bootstrap.LeerMaterialProvisionExterna(*dsnFile, 16<<10)
	if err != nil {
		return fallo(out, 2)
	}
	defer clear(db)
	config, err := pgxpool.ParseConfig(strings.TrimSpace(string(db)))
	if err != nil {
		return fallo(out, 2)
	}
	config.MaxConns = 1
	config.MinConns = 0
	config.ConnConfig.ConnectTimeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fallo(out, 1)
	}
	defer pool.Close()
	r, err := cargosct.Ejecutar(ctx, pool, *op, s)
	if err != nil {
		return fallo(out, 1)
	}
	if json.NewEncoder(out).Encode(r) != nil {
		return 1
	}
	return 0
}
func fallo(out io.Writer, code int) int {
	if json.NewEncoder(out).Encode(struct {
		Error string `json:"error"`
	}{"cargo_ct_rechazado"}) != nil {
		return 1
	}
	return code
}
