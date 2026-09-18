package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"vec-diputacion-granada/config"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	importacionpg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgresimportacionconvoca"
	protector "vec-diputacion-granada/internal/modules/bolsa/adapters/protectorstagingdesarrollo"
	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

var ErrConstitucionBolsaNoDisponible = errors.New("bootstrap: constitucion de bolsa no disponible")

// Actor de la confirmación en el perfil de desarrollo: el subcomando se ejecuta
// por el operador de RRHH dentro del contenedor. En producción el actor vendrá
// de la identidad mTLS de la ruta de confirmación.
const actorConstitucionBolsaDesarrollo = "actor:rrhh:constitucion-bolsa"

// SolicitudConstitucionBolsa identifica el acta por el fichero importado (su
// huella SHA-256) y la categoría RPT con la que se importó.
type SolicitudConstitucionBolsa struct {
	Fichero   string
	Categoria string
}

// EjecutarConstitucionBolsa compone el recuperador del acta (pool de
// importación + protector de desarrollo) y el repositorio de constitución
// (pool de Bolsa) y confirma el acta como bolsa constituida.
func EjecutarConstitucionBolsa(ctx context.Context, cfg config.Config, s SolicitudConstitucionBolsa) (ports.ReciboConstitucion, error) {
	if ctx == nil || !cfg.DevelopmentEnabledByDoubleKey() {
		return ports.ReciboConstitucion{}, ErrConstitucionBolsaNoDisponible
	}
	categoria, err := validarCategoriaImportacionConvoca(cfg, s.Categoria)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	contenido, err := os.ReadFile(s.Fichero)
	if err != nil || len(contenido) == 0 {
		return ports.ReciboConstitucion{}, ErrConstitucionBolsaNoDisponible
	}
	suma := sha256.Sum256(contenido)
	borrarBytes(contenido)
	huella := hex.EncodeToString(suma[:])
	material, err := cargarMaterialSeguridadDesarrollo(cfg)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	defer borrarMaterialImportacionConvoca(material)
	p, err := protector.Nuevo(material.claveKMS)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	derivador, err := protector.NuevoDerivadorCandidato(material.claveKMS)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	poolImportacion, err := abrirPoolImportacionConvoca(ctx, cfg)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	defer poolImportacion.Close()
	recuperador, err := importacionpg.NuevoRepositorioRecuperacionPostgreSQL(poolImportacion, p)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	poolBolsa, err := abrirBolsaLlamamientosPostgreSQLDesarrollo(ctx, cfg.ContratacionTemporalPostgreSQL)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	defer poolBolsa.Close()
	repositorio, err := postgresbolsa.NuevoRepositorioConstitucionPostgreSQL(poolBolsa)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	servicio, err := constitucion.NuevoServicio(recuperador, repositorio, derivador, time.Now)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	return servicio.Constituir(ctx, constitucion.Solicitud{
		HuellaFicheroSHA256: huella, CategoriaRef: categoria, ActorRef: actorConstitucionBolsaDesarrollo,
	})
}
