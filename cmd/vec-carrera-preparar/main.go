package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

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
	if len(args) == 3 && args[0] == "--expediente-sintetico" {
		return runExpediente(args[1], args[2], in, out, errOut)
	}
	if len(args) == 0 {
		return run(in, out, errOut)
	}
	if len(args) == 1 && args[0] == "--antecedentes-sinteticos" {
		return runAntecedentes(in, out, errOut)
	}
	if len(args) == 2 && args[0] == "--politica-grado-sintetica" {
		return runPoliticaGrado(args[1], in, out, errOut)
	}
	_ = json.NewEncoder(errOut).Encode(struct {
		Clave string `json:"error_clave"`
	}{adapter.ErrEntrada.Error()})
	return 1
}

func main() { os.Exit(runArgs(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

type catalogoPoliticaDTO struct {
	Alcance              string             `json:"alcance"`
	Referencia           string             `json:"referencia"`
	Version              string             `json:"version"`
	Fuente               string             `json:"fuente"`
	AprobacionReferencia string             `json:"aprobacion_referencia"`
	Regimenes            []string           `json:"regimenes"`
	Fuentes              []domain.Fuente    `json:"fuentes"`
	Procedencia          domain.Evidencia   `json:"procedencia"`
	Vigencia             domain.Periodo     `json:"vigencia"`
	Vias                 []domain.Evidencia `json:"vias"`
	Periodos             []domain.Evidencia `json:"periodos"`
	Limites              []domain.Evidencia `json:"limites"`
	Evidencias           []domain.Evidencia `json:"evidencias"`
}

func leerPoliticaGrado(ruta string) (domain.CatalogoPoliticaGrado, error) {
	raiz, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return domain.CatalogoPoliticaGrado{}, adapter.ErrEntrada
	}
	defer raiz.Close()
	// La ruta es explícita del operador. Un fichero especial se abre sin
	// bloquear y se rechaza antes de leerlo; el catálogo no abre otras rutas.
	f, err := raiz.OpenFile(filepath.Base(ruta), os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return domain.CatalogoPoliticaGrado{}, adapter.ErrEntrada
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return domain.CatalogoPoliticaGrado{}, adapter.ErrEntrada
	}
	b, err := io.ReadAll(io.LimitReader(f, adapter.MaxBytes+1))
	if err != nil || len(b) > adapter.MaxBytes {
		return domain.CatalogoPoliticaGrado{}, adapter.ErrEntrada
	}
	tokens := json.NewDecoder(bytes.NewReader(b))
	if err := miembrosPoliticaUnicos(tokens, 0); err != nil {
		return domain.CatalogoPoliticaGrado{}, adapter.ErrEntrada
	}
	if _, err := tokens.Token(); err != io.EOF {
		return domain.CatalogoPoliticaGrado{}, adapter.ErrEntrada
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var c catalogoPoliticaDTO
	if err := d.Decode(&c); err != nil {
		return domain.CatalogoPoliticaGrado{}, adapter.ErrEntrada
	}
	return domain.CatalogoPoliticaGrado{
		Alcance:  c.Alcance,
		Politica: domain.Politica{Referencia: c.Referencia, Version: c.Version, Fuente: c.Fuente, AprobacionReferencia: c.AprobacionReferencia, Regimenes: c.Regimenes},
		Fuentes:  c.Fuentes, Procedencia: c.Procedencia, Vigencia: c.Vigencia,
		Vias: c.Vias, Periodos: c.Periodos, Limites: c.Limites, Evidencias: c.Evidencias,
	}, nil
}

func miembrosPoliticaUnicos(d *json.Decoder, profundidad int) error {
	if profundidad > 32 {
		return adapter.ErrEntrada
	}
	t, err := d.Token()
	if err != nil {
		return adapter.ErrEntrada
	}
	delim, complejo := t.(json.Delim)
	if !complejo {
		return nil
	}
	if delim != '{' && delim != '[' {
		return adapter.ErrEntrada
	}
	vistos := make(map[string]bool)
	for d.More() {
		if delim == '{' {
			t, err := d.Token()
			if err != nil {
				return adapter.ErrEntrada
			}
			clave, ok := t.(string)
			if !ok || clave != strings.ToLower(clave) || strings.Trim(clave, "abcdefghijklmnopqrstuvwxyz_") != "" || vistos[clave] {
				return adapter.ErrEntrada
			}
			vistos[clave] = true
		}
		if err := miembrosPoliticaUnicos(d, profundidad+1); err != nil {
			return adapter.ErrEntrada
		}
	}
	fin, err := d.Token()
	if err != nil || (delim == '{' && fin != json.Delim('}')) || (delim == '[' && fin != json.Delim(']')) {
		return adapter.ErrEntrada
	}
	return nil
}

func runPoliticaGrado(ruta string, in io.Reader, out, errOut io.Writer) int {
	e, err := (adapter.Entrada{Reader: in}).Leer()
	var c domain.CatalogoPoliticaGrado
	if err == nil {
		c, err = leerPoliticaGrado(ruta)
	}
	var resultado application.PreparacionConPoliticaGrado
	if err == nil {
		resultado, err = (application.Servicio{}).PrepararConPoliticaGradoSintetica(e, c)
	}
	var preparacion bytes.Buffer
	if err == nil {
		err = (adapter.Salida{Writer: &preparacion}).Escribir(resultado.Preparacion)
	}
	if err == nil {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if enc.Encode(struct {
			Preparacion json.RawMessage              `json:"preparacion"`
			Politica    domain.RevisionPoliticaGrado `json:"politica_grado_sintetica"`
		}{preparacion.Bytes(), resultado.Politica}) != nil {
			err = adapter.ErrSalida
		}
	}
	if err != nil {
		_ = json.NewEncoder(errOut).Encode(struct {
			Clave string `json:"error_clave"`
		}{err.Error()})
		return 1
	}
	return 0
}
