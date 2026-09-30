// Command vec-provisionar-candidato-externo es un proceso interno explícito.
// Emite planes y recibos JSON sin datos de identidad ni secretos. No se monta
// como ruta HTTP ni se invoca en el arranque del portal.
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

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/bootstrap"
)

type dependencias struct {
	preparar func(context.Context, config.Config, string, bootstrap.PlanProvisionCandidatoExterno) (bootstrap.PlanProvisionCandidatoExterno, error)
	ejecutar func(context.Context, config.Config, string, bootstrap.PlanProvisionCandidatoExterno, string, string) (bootstrap.ResumenProvisionCandidatoExterno, error)
	reloj    func() time.Time
}

func main() {
	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	codigo := ejecutar(ctx, os.Args[1:], config.Load(), os.Stdout, os.Stderr, dependencias{bootstrap.CompletarPlanProvisionCandidatoExterno, bootstrap.EjecutarProvisionCandidatoExterno, time.Now})
	parar()
	os.Exit(codigo)
}

func ejecutar(ctx context.Context, args []string, cfg config.Config, salida, errores io.Writer, d dependencias) int {
	flags := flag.NewFlagSet("vec-provisionar-candidato-externo", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var fuente, fase, dsnArchivo, aprobar, preimagen, aliasArchivo string
	flags.StringVar(&fuente, "fuente", "", "")
	flags.StringVar(&fase, "fase", "", "")
	flags.StringVar(&dsnArchivo, "dsn-archivo", "", "")
	flags.StringVar(&aprobar, "aprobar", "", "")
	flags.StringVar(&preimagen, "preimagen", "", "")
	flags.StringVar(&aliasArchivo, "alias-archivo", "", "")
	if ctx == nil || flags.Parse(args) != nil || flags.NArg() != 0 || fuente == "" || fase == "" ||
		(aprobar == "") != (preimagen == "") || d.preparar == nil || d.ejecutar == nil || d.reloj == nil {
		return rechazar(errores, "provision_uso_invalido", 2)
	}
	f, err := bootstrap.CargarFuenteProvisionCandidatoExterno(fuente)
	if err != nil {
		return rechazar(errores, "provision_fuente_invalida", 2)
	}
	if aliasArchivo != "" {
		if fase != "identidad" || f.Identidad != nil {
			return rechazar(errores, "provision_alias_rechazado", 2)
		}
		f.Identidad, err = bootstrap.CargarAliasIdentidadCandidatoExterno(aliasArchivo, f.Snapshot.Cuenta.Referencia)
		if err != nil {
			return rechazar(errores, "provision_alias_rechazado", 2)
		}
	}
	p, err := bootstrap.PrepararProvisionCandidatoExterno(cfg, f, fase, d.reloj())
	if err != nil {
		return rechazar(errores, "provision_plan_rechazado", 2)
	}
	dsn := ""
	if fase == "contexto" || aprobar != "" {
		b, e := bootstrap.LeerMaterialProvisionExterna(dsnArchivo, 16<<10)
		if e != nil {
			return rechazar(errores, "provision_conexion_rechazada", 2)
		}
		dsn = strings.TrimSpace(string(b))
		clear(b)
		if dsn == "" {
			return rechazar(errores, "provision_conexion_rechazada", 2)
		}
	}
	ctx, cancelar := context.WithTimeout(ctx, 30*time.Second)
	defer cancelar()
	p, err = d.preparar(ctx, cfg, dsn, p)
	if err != nil {
		return rechazar(errores, "provision_preimagen_rechazada", 1)
	}
	if aprobar == "" {
		if json.NewEncoder(salida).Encode(p.Resumen()) != nil {
			return rechazar(errores, "provision_salida_fallida", 1)
		}
		return 0
	}
	r, err := d.ejecutar(ctx, cfg, dsn, p, aprobar, preimagen)
	if err != nil {
		return rechazar(errores, "provision_no_confirmada_reconciliar", 1)
	}
	if json.NewEncoder(salida).Encode(r) != nil {
		return rechazar(errores, "provision_recibo_no_entregado_reconciliar", 1)
	}
	return 0
}

func rechazar(w io.Writer, codigo string, estado int) int {
	if json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{codigo}) != nil {
		return 1
	}
	return estado
}
