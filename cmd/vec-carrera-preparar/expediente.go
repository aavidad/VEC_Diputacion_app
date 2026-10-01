package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"syscall"

	adapter "vec-diputacion-granada/internal/modules/carrera/adapters/json"
	"vec-diputacion-granada/internal/modules/carrera/application"
	"vec-diputacion-granada/internal/modules/carrera/domain"
)

func leerAntecedentesExpediente(ruta string) (adapter.LectorAntecedentes, error) {
	raiz, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return adapter.LectorAntecedentes{}, adapter.ErrEntrada
	}
	defer raiz.Close()
	f, err := raiz.OpenFile(filepath.Base(ruta), os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return adapter.LectorAntecedentes{}, adapter.ErrEntrada
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return adapter.LectorAntecedentes{}, adapter.ErrEntrada
	}
	return adapter.LeerAntecedentes(f)
}

func runExpediente(politicaRuta, antecedentesRuta string, in io.Reader, out, errOut io.Writer) int {
	e, err := (adapter.Entrada{Reader: in}).Leer()
	var resultado application.PreparacionExpediente
	if err == nil {
		c, fallo := leerPoliticaGrado(politicaRuta)
		err = fallo
		if err == nil {
			lector, fallo := leerAntecedentesExpediente(antecedentesRuta)
			err = fallo
			if err == nil {
				resultado, err = (application.Servicio{}).PrepararExpedienteSintetico(context.Background(), e, lector, c)
			}
		}
	}
	var preparacion bytes.Buffer
	if err == nil {
		err = (adapter.Salida{Writer: &preparacion}).Escribir(resultado.Preparacion)
	}
	if err == nil {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if enc.Encode(struct {
			Preparacion json.RawMessage                      `json:"preparacion"`
			Politica    domain.RevisionPoliticaGrado         `json:"politica_grado_sintetica"`
			Casos       []application.RevisionExpedienteCaso `json:"revision_expediente"`
		}{preparacion.Bytes(), resultado.Politica, resultado.Casos}) != nil {
			err = adapter.ErrSalida
		}
	}
	if err != nil {
		if json.NewEncoder(errOut).Encode(struct {
			Clave string `json:"error_clave"`
		}{err.Error()}) != nil {
			return 1
		}
		return 1
	}
	return 0
}
