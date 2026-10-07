package main

import (
	"encoding/json"
	"flag"
	"io"
	"time"
	"vec-diputacion-granada/internal/vec/domain"
)

func prepararPlan(f *flag.FlagSet, parseErr error, fuente, destino, conexion, aprobacion, acuse string, fallo func(string) int, emitir func(io.Writer, diagnostico, int) int, salida io.Writer) int {
	if parseErr != nil || f.NArg() != 0 || conexion != "" || aprobacion != "" || acuse != "" || !rutasDistintas(fuente, destino, f.Lookup("textos").Value.String()) || f.Lookup("timeout").Value.String() != "" {
		return fallo("uso_invalido")
	}
	b, e := leerPrivado(fuente)
	if e != nil {
		return fallo("entrada_insegura")
	}
	defer clear(b)
	var p domain.PlanIdentidadInternaSinteticaV1
	if decodificarEstricto(b, &p) != nil || p.ValidarEn(time.Now()) != nil {
		return fallo("entrada_invalida")
	}
	_, sha, e := p.CanonicoYHuella()
	if e != nil {
		return fallo("entrada_invalida")
	}
	doc, e := json.Marshal(documento{Plan: p, HuellaPlanSHA256: sha})
	if e != nil {
		return fallo("entrada_invalida")
	}
	defer clear(doc)
	if crearOComparar(destino, append(doc, '\n')) != nil {
		return fallo("acuse_inseguro")
	}
	return emitir(salida, diagnostico{Codigo: "plan_preparado", Estado: "preparado"}, 0)
}
