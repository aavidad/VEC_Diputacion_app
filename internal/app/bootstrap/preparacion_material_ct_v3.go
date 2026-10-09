package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"vec-diputacion-granada/config"
)

var errCoordenadasCTPreparacion = errors.New("vec: coordenadas CT de preparación no disponibles")

// CoordenadasCTPreparacion contiene únicamente datos públicos del material
// derivado por el mismo camino que usa el publicador de vec-server.
type CoordenadasCTPreparacion struct {
	RaizID, Audiencia, HuellaSPKI, EmisorID string
	RaizVersion                             uint64
}

// DerivarCoordenadasCTPreparacion no exporta la raíz privada ni la clave HMAC.
// La identidad del material se coteja después con el gobierno publicado.
func DerivarCoordenadasCTPreparacion(directorioIdempotencia string, ahora time.Time) (CoordenadasCTPreparacion, error) {
	var vacio CoordenadasCTPreparacion
	if !filepath.IsAbs(directorioIdempotencia) || filepath.Clean(directorioIdempotencia) != directorioIdempotencia ||
		filepath.Base(directorioIdempotencia) != "idempotencia" || dentroDeRepositorioGit(directorioIdempotencia) {
		return vacio, errCoordenadasCTPreparacion
	}
	resuelta, err := filepath.EvalSymlinks(directorioIdempotencia)
	if err != nil || resuelta != directorioIdempotencia {
		return vacio, errCoordenadasCTPreparacion
	}
	info, err := os.Lstat(directorioIdempotencia)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return vacio, errCoordenadasCTPreparacion
	}
	padre := filepath.Dir(directorioIdempotencia)
	idem, err := cargarMaterialIdempotenciaDesarrollo(padre, filepath.Join(padre, config.DevelopmentIdempotencyHMACConfigRelativePath))
	if err != nil {
		return vacio, errCoordenadasCTPreparacion
	}
	defer idem.borrar()
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&idem)
	if err != nil {
		return vacio, errCoordenadasCTPreparacion
	}
	defer derivador.borrar()
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, ahora)
	if err != nil {
		return vacio, errCoordenadasCTPreparacion
	}
	defer material.borrarCopiasEfimeras()
	return CoordenadasCTPreparacion{
		RaizID: material.claveID, RaizVersion: material.claveVersion,
		Audiencia:  audienciaAtestacionContratacionTemporalDesarrollo,
		HuellaSPKI: material.spkiHuella, EmisorID: material.emisorID,
	}, nil
}
