// Package ports declara los contratos de Calendarios. Otros módulos consultan
// fechas y plazos a través de ConsultaCalendarios, nunca leyendo sus tablas.
package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/calendarios/domain"
)

// ConsultaVersiones pide, para un año, la última versión conocida hasta
// ConocidoEn de cada ámbito. Un ámbito sin versión simplemente no aparece.
type ConsultaVersiones struct {
	Anio       int
	Ambitos    []domain.Ambito
	ConocidoEn time.Time
}

// RepositorioCalendarios es el puerto de lectura de la historia de solo
// adición. No expone escritura: el catálogo se amplía mediante un acto de RRHH
// gobernado que no forma parte de este contrato.
type RepositorioCalendarios interface {
	VersionesVigentes(context.Context, ConsultaVersiones) ([]domain.VersionConDias, error)
	CentrosConCalendario(ctx context.Context, anio int, conocidoEn time.Time) ([]domain.VersionCalendario, error)
}

type Reloj interface {
	Ahora() time.Time
}

type SolicitudCalendarioCentro struct {
	CentroRef string
	Anio      int
	// ConocidoEn reconstruye lo que el sistema sabía en ese instante. Cero
	// significa ahora.
	ConocidoEn time.Time
}

type ResumenCalendario struct {
	DiasNaturales          int `json:"dias_naturales"`
	Laborables             int `json:"laborables"`
	HabilesAdministrativos int `json:"habiles_administrativos"`
	FestivosOficiales      int `json:"festivos_oficiales"`
	NoLaborablesCentro     int `json:"no_laborables_centro"`
}

type CalendarioCentro struct {
	CentroRef    string                     `json:"centro_ref"`
	Anio         int                        `json:"anio"`
	ConocidoEn   time.Time                  `json:"conocido_en"`
	Zona         string                     `json:"zona"`
	ComunidadRef string                     `json:"comunidad_ref"`
	MunicipioRef string                     `json:"municipio_ref"`
	Versiones    []domain.VersionCalendario `json:"versiones"`
	Dias         []domain.Clasificacion     `json:"dias"`
	Resumen      ResumenCalendario          `json:"resumen"`
}

type CentroConCalendario struct {
	CentroRef    string `json:"centro_ref"`
	Denominacion string `json:"denominacion"`
	MunicipioRef string `json:"municipio_ref"`
	VersionID    string `json:"version_id"`
	Numero       int    `json:"numero"`
}

// SolicitudCalculoPlazo admite exactamente uno de Inicio (fecha civil) o
// NotificadoEn (instante, convertido a fecha en Europe/Madrid).
type SolicitudCalculoPlazo struct {
	Inicio              domain.FechaCivil
	NotificadoEn        time.Time
	Unidad              domain.UnidadPlazo
	Cantidad            int
	MunicipioSede       string
	MunicipioResidencia string
	ConocidoEn          time.Time
}

type ResultadoCalculoPlazo struct {
	domain.ResultadoPlazo
	Unidad              domain.UnidadPlazo         `json:"unidad"`
	Cantidad            int                        `json:"cantidad"`
	MunicipioSede       string                     `json:"municipio_sede"`
	MunicipioResidencia string                     `json:"municipio_residencia,omitempty"`
	ConocidoEn          time.Time                  `json:"conocido_en"`
	Zona                string                     `json:"zona"`
	VersionesUtilizadas []domain.VersionCalendario `json:"versiones_utilizadas"`
}

// ConsultaCalendarios es el contrato que consumen Contratación, Bolsa,
// Cronos u otros procedimientos. La autorización de la finalidad corresponde
// a la frontera que lo invoca.
type ConsultaCalendarios interface {
	Centros(ctx context.Context, anio int, conocidoEn time.Time) ([]CentroConCalendario, error)
	CalendarioCentro(context.Context, SolicitudCalendarioCentro) (CalendarioCentro, error)
	CalcularPlazo(context.Context, SolicitudCalculoPlazo) (ResultadoCalculoPlazo, error)
}
