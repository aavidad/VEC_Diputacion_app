package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

type OrdenFronteraIdentidadTecnica = domain.OrdenFronteraIdentidadTecnica
type MotivoFronteraIdentidadTecnica = domain.MotivoFronteraIdentidadTecnica
type AcuseFronteraIdentidadTecnica = domain.AcuseFronteraIdentidadTecnica

const (
	MotivoFronteraIdentidadCertificadoRequerido   = domain.MotivoCertificadoRequerido
	MotivoFronteraIdentidadAutenticacionRequerida = domain.MotivoAutenticacionRequerida
	MotivoFronteraIdentidadAccesoDenegado         = domain.MotivoAccesoDenegado
	MotivoFronteraIdentidadMetodoNoPermitido      = domain.MotivoMetodoNoPermitido
	MotivoFronteraIdentidadRecursoNoEncontrado    = domain.MotivoRecursoNoEncontrado
	MotivoFronteraIdentidadSolicitudInvalida      = domain.MotivoSolicitudInvalida
	MotivoFronteraIdentidadServicioNoDisponible   = domain.MotivoServicioNoDisponible
	MotivoFronteraIdentidadRespuestaIncompatible  = domain.MotivoRespuestaIncompatible
)

var (
	ErrOrdenFronteraIdentidadTecnicaInvalida  = domain.ErrOrdenFronteraIdentidadTecnicaInvalida
	ErrAcuseFronteraIdentidadTecnicaInvalido  = domain.ErrAcuseFronteraIdentidadTecnicaInvalido
	ErrFronteraIdentidadTecnicaNoDisponible   = domain.ErrFronteraIdentidadTecnicaNoDisponible
	ErrFronteraIdentidadTecnicaCommitIncierto = domain.ErrFronteraIdentidadTecnicaCommitIncierto
)

// NuevaOrdenFronteraIdentidadTecnica genera la correlación en el servidor,
// con método/ruta esperados y motivo catalogado. No recibe identidad del HTTP.
func NuevaOrdenFronteraIdentidadTecnica(
	metodoEsperado, ruta string, motivo MotivoFronteraIdentidadTecnica,
) (OrdenFronteraIdentidadTecnica, error) {
	return domain.NuevaOrdenFronteraIdentidadTecnica(metodoEsperado, ruta, motivo)
}

// RegistradorFronteraIdentidadTecnica confirma el asiento técnico previo a
// acreditar certificado. El consumidor espera el acuse antes de rechazar;
// un error no habilita START ni degrada el hecho a actor inventado.
type RegistradorFronteraIdentidadTecnica interface {
	RegistrarRechazoInicioSesion(context.Context, OrdenFronteraIdentidadTecnica) (AcuseFronteraIdentidadTecnica, error)
}
