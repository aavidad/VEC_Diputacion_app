// CLI local de preparación de identidades sintéticas H6. No monta una API.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"syscall"

	"vec-diputacion-granada/internal/app/bootstrap"
)

type resultado struct {
	Estado string                               `json:"estado"`
	Codigo string                               `json:"codigo,omitempty"`
	Plan   *bootstrap.PlanIdentidadesInternasH6 `json:"plan,omitempty"`
}

func main() {
	os.Exit(ejecutar(os.Args[1:], os.Stdout))
}

func ejecutar(args []string, salida io.Writer) int {
	banderas := flag.NewFlagSet("vec-provisionar-identidades-internas", flag.ContinueOnError)
	banderas.SetOutput(os.Stderr)
	fuente := banderas.String("fuente", "", "fuente")
	modo := banderas.String("modo", "", "plan|reconcile|apply")
	planSHA := banderas.String("plan-sha256", "", "plan-sha256")
	if banderas.Parse(args) != nil || banderas.NArg() != 0 || *fuente == "" ||
		(*modo != "plan" && *modo != "reconcile" && *modo != "apply") {
		return emitir(salida, resultado{Estado: "denegado", Codigo: "argumentos_invalidos"}, 2)
	}
	plan, err := bootstrap.LeerFuenteIdentidadesInternasH6(*fuente)
	if err != nil {
		return emitir(salida, resultado{Estado: "denegado", Codigo: "fuente_invalida"}, 1)
	}
	if *modo == "reconcile" {
		plan, err = bootstrap.ReconciliarIdentidadesInternasH6(plan, *planSHA)
		if err != nil {
			return emitir(salida, resultado{Estado: "denegado", Codigo: "plan_no_coincide"}, 1)
		}
	}
	if *modo == "apply" {
		if *planSHA == "" || *planSHA != plan.PlanSHA256 {
			return emitir(salida, resultado{Estado: "denegado", Codigo: "plan_no_coincide"}, 1)
		}
		if err := bootstrap.AplicarIdentidadesInternasH6(plan); err != nil {
			return emitir(salida, resultado{Estado: "denegado", Codigo: "autoridad_no_disponible"}, 1)
		}
	}
	return emitir(salida, resultado{Estado: plan.Estado, Plan: &plan}, 0)
}

func emitir(salida io.Writer, r resultado, codigo int) int {
	if err := json.NewEncoder(salida).Encode(r); err != nil {
		if errors.Is(err, syscall.EPIPE) {
			return 1
		}
		return 3
	}
	return codigo
}
