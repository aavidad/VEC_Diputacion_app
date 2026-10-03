// Command vec-provisionar-cargos-ct prepara una provisión fuera de petición.
// Solo lee la fuente V3; el efecto exige un vínculo confiable que esta CLI no
// crea ni admite por JSON.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/bootstrap"
)

type dependencias struct {
	preparar func(context.Context, string, bootstrap.ConfiguracionProvisionCargoCT, time.Time) (bootstrap.ResumenProvisionCargoCT, error)
	reloj    func() time.Time
}

func main() {
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelar()
	os.Exit(ejecutar(ctx, os.Args[1:], os.Stdout, os.Stderr, dependencias{prepararPostgreSQL, time.Now}))
}

func ejecutar(ctx context.Context, args []string, salida, errores io.Writer, d dependencias) int {
	flags := flag.NewFlagSet("vec-provisionar-cargos-ct", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var fuente, dsnArchivo string
	flags.StringVar(&fuente, "fuente", "", "")
	flags.StringVar(&dsnArchivo, "dsn-archivo", "", "")
	if ctx == nil || flags.Parse(args) != nil || flags.NArg() != 0 || fuente == "" || dsnArchivo == "" ||
		d.preparar == nil || d.reloj == nil {
		return rechazar(errores, "provision_cargo_uso_invalido", 2)
	}
	c, err := bootstrap.CargarConfiguracionProvisionCargoCT(fuente)
	if err != nil {
		return rechazar(errores, "provision_cargo_fuente_invalida", 2)
	}
	b, err := bootstrap.LeerMaterialProvisionExterna(dsnArchivo, 16<<10)
	if err != nil {
		return rechazar(errores, "provision_cargo_conexion_invalida", 2)
	}
	dsn := strings.TrimSpace(string(b))
	clear(b)
	if dsn == "" {
		return rechazar(errores, "provision_cargo_conexion_invalida", 2)
	}
	ctx, cancelar := context.WithTimeout(ctx, 20*time.Second)
	defer cancelar()
	r, err := d.preparar(ctx, dsn, c, d.reloj().UTC().Truncate(time.Microsecond))
	if err != nil {
		return rechazar(errores, "provision_cargo_preimagen_rechazada", 1)
	}
	if json.NewEncoder(salida).Encode(r) != nil {
		return rechazar(errores, "provision_cargo_salida_fallida", 1)
	}
	return 0
}

func prepararPostgreSQL(ctx context.Context, dsn string, c bootstrap.ConfiguracionProvisionCargoCT, ahora time.Time) (bootstrap.ResumenProvisionCargoCT, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return bootstrap.ResumenProvisionCargoCT{}, bootstrap.ErrProvisionCargoCT
	}
	config.MaxConns = 1
	config.MinConns = 0
	config.ConnConfig.ConnectTimeout = 5 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return bootstrap.ResumenProvisionCargoCT{}, bootstrap.ErrProvisionCargoCT
	}
	defer pool.Close()
	plan, err := bootstrap.PrepararProvisionCargoCT(ctx, pool, c, ahora)
	if err != nil {
		return bootstrap.ResumenProvisionCargoCT{}, err
	}
	return plan.Resumen(), nil
}

func rechazar(w io.Writer, codigo string, estado int) int {
	if json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{codigo}) != nil {
		return 1
	}
	return estado
}
