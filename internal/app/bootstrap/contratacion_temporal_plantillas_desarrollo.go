package bootstrap

import (
	"context"
	"errors"
	"os"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// rutaPlantillasCTEjemplo es el catálogo de plantillas que trae el
// repositorio. VEC_CT_PLANTILLAS_SOURCE_PATH lo sustituye por otro con el
// mismo formato; en ambos casos se valida entero al arrancar.
const rutaPlantillasCTEjemplo = "data/demo/plantillas/ct_plantillas_documentos.ejemplo.demo.json"

var errPlantillasCTNoDisponibles = errors.New("bootstrap: catalogo de plantillas de contratacion temporal no disponible")

// cargarPlantillasBorradorCTDesarrollo lee y valida el catálogo. Un catálogo
// ilegible, con campos desconocidos o fuera de límites impide arrancar en
// lugar de servir documentos con texto incompleto.
func cargarPlantillasBorradorCTDesarrollo(cfg config.Config, instante time.Time) (*informejuridico.PlantillasBorrador, error) {
	reglas, _, err := cfg.ReglasEjemploDesarrollo()
	if err != nil {
		return nil, err
	}
	candidatas := []string{reglas.CTPlantillasSourcePath}
	if reglas.CTPlantillasSourcePath == "" {
		candidatas = []string{rutaPlantillasCTEjemplo, "../../../" + rutaPlantillasCTEjemplo}
	}
	for _, ruta := range candidatas {
		if _, err := os.Stat(ruta); os.IsNotExist(err) && reglas.CTPlantillasSourcePath == "" {
			continue
		}
		return plantillasBorradorCTDesdeFichero(ruta, instante)
	}
	return nil, errPlantillasCTNoDisponibles
}

func plantillasBorradorCTDesdeFichero(ruta string, instante time.Time) (*informejuridico.PlantillasBorrador, error) {
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		return nil, errors.Join(errPlantillasCTNoDisponibles, err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	versiones, err := consulta.ListarVersionesCatalogo(ctx, informejuridico.CatalogoPlantillasBorradorID)
	if err != nil {
		return nil, errors.Join(errPlantillasCTNoDisponibles, err)
	}
	var elegido *vecdomain.CatalogoConfigurable
	for i := range versiones {
		if versiones[i].Estado == vecdomain.EstadoCatalogoPublicado && (elegido == nil || versiones[i].Version > elegido.Version) {
			elegido = &versiones[i]
		}
	}
	if elegido == nil {
		return nil, errPlantillasCTNoDisponibles
	}
	plantillas, err := informejuridico.NuevasPlantillasBorrador(*elegido, instante)
	if err != nil {
		return nil, errors.Join(errPlantillasCTNoDisponibles, err)
	}
	return plantillas, nil
}
