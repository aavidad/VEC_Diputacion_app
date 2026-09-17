package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
	"vec-diputacion-granada/config"
	postgres "vec-diputacion-granada/internal/modules/bolsa/adapters/postgresimportacionconvoca"
	protector "vec-diputacion-granada/internal/modules/bolsa/adapters/protectorstagingdesarrollo"
	xls "vec-diputacion-granada/internal/modules/bolsa/adapters/xlsconvoca"
	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
)

var ErrImportacionConvocaNoDisponible = errors.New("bootstrap: importacion Convoca no disponible")

type SolicitudImportacionConvoca struct{ Fichero, Categoria, BolsaRef string }

func EjecutarImportacionConvoca(ctx context.Context, cfg config.Config, s SolicitudImportacionConvoca) (aplicacion.ResultadoImportacion, error) {
	if ctx == nil || !cfg.DevelopmentEnabledByDoubleKey() {
		return aplicacion.ResultadoImportacion{}, ErrImportacionConvocaNoDisponible
	}
	categoria, err := validarCategoriaImportacionConvoca(cfg, s.Categoria)
	if err != nil {
		return aplicacion.ResultadoImportacion{}, err
	}
	contenido, err := os.ReadFile(s.Fichero)
	if err != nil {
		return aplicacion.ResultadoImportacion{}, ErrImportacionConvocaNoDisponible
	}
	defer borrarBytes(contenido)
	custodia, err := custodiarImportacionConvocaDesarrollo(cfg, contenido)
	if err != nil {
		return aplicacion.ResultadoImportacion{}, err
	}
	material, err := cargarMaterialSeguridadDesarrollo(cfg)
	if err != nil {
		return aplicacion.ResultadoImportacion{}, err
	}
	defer borrarMaterialImportacionConvoca(material)
	p, err := protector.Nuevo(material.claveKMS)
	if err != nil {
		return aplicacion.ResultadoImportacion{}, err
	}
	pool, err := abrirPoolImportacionConvoca(ctx, cfg)
	if err != nil {
		return aplicacion.ResultadoImportacion{}, err
	}
	defer pool.Close()
	repo, err := postgres.NuevoRepositorioPostgreSQL(pool, p)
	if err != nil {
		return aplicacion.ResultadoImportacion{}, err
	}
	servicio, err := aplicacion.NuevoServicio(xls.NuevoLector(), repo, time.Now)
	if err != nil {
		return aplicacion.ResultadoImportacion{}, err
	}
	return servicio.Importar(ctx, aplicacion.SolicitudImportacion{CategoriaRef: categoria, BolsaRef: bolsaRefImportacionConvoca(s.BolsaRef, s.Categoria, time.Now()), NombreFichero: filepath.Base(s.Fichero), FicheroCustodiadoRef: custodia, ActorRef: "actor:rrhh:importador-convoca", Contenido: contenido})
}

func bolsaRefImportacionConvoca(indicada, categoria string, ahora time.Time) string {
	if indicada = strings.TrimSpace(indicada); indicada != "" {
		return indicada
	}
	return "bolsa:" + strings.TrimSpace(categoria) + ":" + ahora.UTC().Format("2006-01-02")
}

func borrarMaterialImportacionConvoca(material materialSeguridadDesarrollo) {
	borrarBytes(material.claveKMS[:])
	borrarBytes(material.firmaAtestacionKMS)
	borrarBytes(material.firmaRevalidacionKMS)
	borrarBytes(material.claveTSA[:])
	material.idempotencia.borrar()
}
