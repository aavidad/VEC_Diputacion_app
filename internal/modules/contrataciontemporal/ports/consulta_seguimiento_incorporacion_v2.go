package ports

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// VistaSeguimientoIncorporacionV2 es la proyeccion de lectura del seguimiento
// resultante de una incorporacion. No expone autoridad, material canonico ni
// huellas de la historia durable.
type VistaSeguimientoIncorporacionV2 struct {
	Esquema                string                                       `json:"esquema"`
	Alcance                string                                       `json:"alcance"`
	ExpedienteRef          string                                       `json:"expediente_ref"`
	VersionExpediente      uint64                                       `json:"version_expediente"`
	ReciboIncorporacionRef string                                       `json:"recibo_incorporacion_ref"`
	SeguimientoRef         string                                       `json:"seguimiento_ref"`
	VersionSeguimiento     uint64                                       `json:"version_seguimiento"`
	EstadoClave            domain.ClaveCatalogo                         `json:"estado_clave"`
	Periodo                domain.IntervaloSeguimiento                  `json:"periodo"`
	RegistradoEn           time.Time                                    `json:"registrado_en"`
	Actuaciones            []ActuacionVisibleSeguimientoIncorporacionV2 `json:"actuaciones"`
	EjercicioSintetico     bool                                         `json:"ejercicio_sintetico"`
	FirmaOficial           bool                                         `json:"firma_oficial"`
	EficaciaAdministrativa bool                                         `json:"eficacia_administrativa"`
}

// ActuacionVisibleSeguimientoIncorporacionV2 contiene solo los datos
// publicables de cada actuacion del seguimiento.
type ActuacionVisibleSeguimientoIncorporacionV2 struct {
	ActuacionRef    string                        `json:"actuacion_ref"`
	TransicionClave domain.ClaveCatalogo          `json:"transicion_clave"`
	EstadoOrigen    domain.ClaveCatalogo          `json:"estado_origen"`
	EstadoDestino   domain.ClaveCatalogo          `json:"estado_destino"`
	EfectivoEn      time.Time                     `json:"efectivo_en"`
	RegistradaEn    time.Time                     `json:"registrada_en"`
	Documentos      []domain.DocumentoSeguimiento `json:"documentos"`
}
