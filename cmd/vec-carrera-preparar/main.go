package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"

	adapter "vec-diputacion-granada/internal/modules/carrera/adapters/json"
	"vec-diputacion-granada/internal/modules/carrera/application"
	"vec-diputacion-granada/internal/modules/carrera/domain"
	"vec-diputacion-granada/internal/modules/carrera/ports"
)

func run(in io.Reader, out, errOut io.Writer) int {
	if err := (application.Servicio{}).Ejecutar(adapter.Entrada{Reader: in}, adapter.Salida{Writer: out}); err != nil {
		_ = json.NewEncoder(errOut).Encode(struct {
			Clave string `json:"error_clave"`
		}{err.Error()})
		return 1
	}
	return 0
}

// El modo optativo ejercita el consumidor H05 con el JSON sintético existente.
// Sus periodos siguen declarados y los números carecen de acto reconocido.
type lectorSintetico struct{ escenario domain.Escenario }

func (l lectorSintetico) ConsultarAntecedentesSinteticos(_ context.Context, q ports.ConsultaAntecedentesSinteticos) (ports.InstantaneaAntecedentesSinteticos, error) {
	for _, c := range l.escenario.Casos {
		if c.Referencia != q.CasoRef {
			continue
		}
		a := ports.InstantaneaAntecedentesSinteticos{
			Alcance: l.escenario.Alcance, CasoRef: c.Referencia, Version: l.escenario.Version,
			Regimen: c.Regimen, GrupoSubgrupo: c.GrupoSubgrupo, GrupoProfesional: c.GrupoProfesional,
			Fuentes: c.Fuentes,
		}
		if c.NivelPuesto != nil {
			a.Ocupaciones = []ports.OcupacionAntecedente{{Nivel: c.NivelPuesto}}
		}
		if c.GradoPersonal != nil {
			a.Grado = &ports.GradoAntecedente{Valor: *c.GradoPersonal}
		}
		for _, p := range c.Periodos {
			a.Servicios = append(a.Servicios, ports.ServicioAntecedente{Referencia: p.Evidencia.Referencia, Estado: "declarado", Periodo: p})
		}
		return a, nil
	}
	return ports.InstantaneaAntecedentesSinteticos{}, application.ErrAntecedentes
}

func runAntecedentes(in io.Reader, out, errOut io.Writer) int {
	e, err := (adapter.Entrada{Reader: in}).Leer()
	var resultado application.PreparacionAntecedentes
	if err == nil {
		resultado, err = (application.Servicio{}).PrepararConAntecedentesSinteticos(context.Background(), e, lectorSintetico{e})
	}
	var preparacion bytes.Buffer
	if err == nil {
		err = (adapter.Salida{Writer: &preparacion}).Escribir(resultado.Preparacion)
	}
	if err != nil {
		_ = json.NewEncoder(errOut).Encode(struct {
			Clave string `json:"error_clave"`
		}{err.Error()})
		return 1
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(struct {
		Preparacion  json.RawMessage                `json:"preparacion"`
		Antecedentes []application.AntecedentesCaso `json:"antecedentes_sinteticos"`
	}{json.RawMessage(preparacion.Bytes()), resultado.Casos}); err != nil {
		_ = json.NewEncoder(errOut).Encode(struct {
			Clave string `json:"error_clave"`
		}{adapter.ErrSalida.Error()})
		return 1
	}
	return 0
}

func runArgs(args []string, in io.Reader, out, errOut io.Writer) int {
	if len(args) == 0 {
		return run(in, out, errOut)
	}
	if len(args) == 1 && args[0] == "--antecedentes-sinteticos" {
		return runAntecedentes(in, out, errOut)
	}
	_ = json.NewEncoder(errOut).Encode(struct {
		Clave string `json:"error_clave"`
	}{adapter.ErrEntrada.Error()})
	return 1
}

func main() { os.Exit(runArgs(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
