// CLI local de preparación de identidades sintéticas H6. No monta una API.
package main

import (
	"encoding/json"
	"flag"
	"os"

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

func ejecutar(args []string, salida *os.File) int {
	banderas := flag.NewFlagSet("vec-provisionar-identidades-internas", flag.ContinueOnError)
	banderas.SetOutput(os.Stderr)
	fuente := banderas.String("fuente", "", "fuente")
	modo := banderas.String("modo", "", "plan|reconcile|apply")
	planSHA := banderas.String("plan-sha256", "", "plan-sha256")
	if banderas.Parse(args) != nil || banderas.NArg() != 0 || *fuente == "" ||
		(*modo != "plan" && *modo != "reconcile" && *modo != "apply") {
		_ = json.NewEncoder(salida).Encode(resultado{Estado: "denegado", Codigo: "argumentos_invalidos"})
		return 2
	}
	plan, err := bootstrap.LeerFuenteIdentidadesInternasH6(*fuente)
	if err != nil {
		_ = json.NewEncoder(salida).Encode(resultado{Estado: "denegado", Codigo: "fuente_invalida"})
		return 1
	}
	if *modo == "reconcile" {
		plan, err = bootstrap.ReconciliarIdentidadesInternasH6(plan, *planSHA)
		if err != nil {
			_ = json.NewEncoder(salida).Encode(resultado{Estado: "denegado", Codigo: "plan_no_coincide"})
			return 1
		}
	}
	if *modo == "apply" {
		if *planSHA == "" || *planSHA != plan.PlanSHA256 {
			_ = json.NewEncoder(salida).Encode(resultado{Estado: "denegado", Codigo: "plan_no_coincide"})
			return 1
		}
		if err := bootstrap.AplicarIdentidadesInternasH6(plan); err != nil {
			_ = json.NewEncoder(salida).Encode(resultado{Estado: "denegado", Codigo: "autoridad_no_disponible"})
			return 1
		}
	}
	_ = json.NewEncoder(salida).Encode(resultado{Estado: plan.Estado, Plan: &plan})
	return 0
}
