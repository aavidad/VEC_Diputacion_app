package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"log/slog"
	"net/http"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
)

//go:embed provision_ensayos.json
var ensayosJSON []byte

type casoCicloLocal struct {
	CasoRef     string `json:"caso_ref"`
	Decision    string `json:"decision"`
	TituloClave string `json:"titulo_clave"`
}
type catalogoEnsayosLocales struct {
	AdjudicacionRef string           `json:"adjudicacion_ref"`
	CicloRef        string           `json:"ciclo_ref"`
	Casos           []casoCicloLocal `json:"casos"`
}

func leerCatalogoEnsayos() (catalogoEnsayosLocales, error) {
	var c catalogoEnsayosLocales
	err := simulacion.Decodificar(bytes.NewReader(ensayosJSON), &c)
	return c, err
}

func responderEnsayo(w http.ResponseWriter, datos any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(datos); err != nil {
		slog.Warn("provision_respuesta_no_entregada", "operacion", "ensayo_local")
		return
	}
}
