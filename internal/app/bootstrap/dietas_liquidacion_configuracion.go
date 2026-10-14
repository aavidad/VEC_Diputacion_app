package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"vec-diputacion-granada/config"
)

var errLiquidacionEconomicaNoDisponible = errors.New("bootstrap: liquidacion economica de dietas no disponible")

type configuracionLiquidacionDietas struct {
	Version    int  `json:"version"`
	Habilitada bool `json:"habilitada"`
}

// El selector privado habilita la función. Las tarifas se resuelven en SQL
// por la versión del expediente y su fecha, desde la misma fuente publicada.
func leerConfiguracionLiquidacionDietas(cfg config.Config) (*configuracionLiquidacionDietas, error) {
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "dietas-liquidacion.json")
	info, err := os.Lstat(ruta)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return nil, errLiquidacionEconomicaNoDisponible
	}
	contenido, err := leerFicheroMaterialSeguro(ruta, 64<<10)
	if err != nil {
		return nil, errLiquidacionEconomicaNoDisponible
	}
	defer borrarBytes(contenido)
	if validarClavesJSONUnicas(contenido) != nil {
		return nil, errLiquidacionEconomicaNoDisponible
	}
	var c configuracionLiquidacionDietas
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) || c.Version != 1 {
		return nil, errLiquidacionEconomicaNoDisponible
	}
	if !c.Habilitada {
		return nil, nil
	}
	return &c, nil
}
