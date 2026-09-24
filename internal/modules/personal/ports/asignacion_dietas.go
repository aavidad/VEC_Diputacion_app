package ports

import (
	"context"
	"errors"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrSolicitudAsignacionDietasInvalida      = errors.New("personal: solicitud de asignacion invalida")
	ErrAsignacionDietasNoDisponible           = errors.New("personal: asignacion no disponible")
	ErrAsignacionDietasDenegada               = errors.New("personal: asignacion denegada")
	ErrAutenticacionAsignacionDietasRequerida = errors.New("personal: autenticacion requerida para asignacion")
	ErrIdentidadAsignacionDietasNoDisponible  = errors.New("personal: identidad de asignacion no disponible")
	ErrVersionAsignacionDietas                = errors.New("personal: version de asignacion en conflicto")
	ErrClaveAsignacionDietas                  = errors.New("personal: clave de asignacion reutilizada")
)

const (
	AccionConsultarAsignacionDietas           = "personal.asignacion_dietas.consultar"
	AccionRegistrarInicialAsignacionDietas    = "personal.asignacion_dietas.registrar_inicial"
	AccionCorregirAsignacionDietas            = "personal.asignacion_dietas.corregir"
	AccionCorregirGrupoDietas                 = "personal.asignacion_dietas.grupo_corregir"
	AudienciaConsultarAsignacionDietas        = "vec_personal.asignacion_dietas.consultar.v1"
	AudienciaRegistrarInicialAsignacionDietas = "vec_personal.asignacion_dietas.registrar_inicial.v1"
	AudienciaCorregirAsignacionDietas         = "vec_personal.asignacion_dietas.corregir.v1"
	AudienciaCorregirGrupoDietas              = "vec_personal.asignacion_dietas.grupo_corregir.v1"
)

type ProveedorAutorizacionAsignacionDietas interface {
	AutorizarAsignacionDietas(context.Context, personaldomain.MaterialAsignacionDietas) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenAsignacionDietas struct {
	Material     personaldomain.MaterialAsignacionDietas
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ResultadoAsignacionDietas struct {
	Asignacion          personaldomain.AsignacionDietas `json:"asignacion"`
	ReciboRef           string                          `json:"recibo_ref"`
	DecisionRef         string                          `json:"decision_ref"`
	EfectoRef           string                          `json:"efecto_ref"`
	ConsumoHuellaSHA256 string                          `json:"consumo_huella_sha256"`
	AuditoriaRef        string                          `json:"auditoria_ref"`
	RegistradaEn        time.Time                       `json:"registrada_en"`
	EstadoLocal         string                          `json:"estado_local"`
}

type RepositorioAsignacionDietas interface {
	EjecutarAsignacionDietas(context.Context, OrdenAsignacionDietas) (ResultadoAsignacionDietas, error)
}
