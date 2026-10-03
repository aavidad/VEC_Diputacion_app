package competenciacentral

import (
	"context"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// RevalidadorVinculoCertificado se liga a la transacción final por composición.
// No abre ni confirma otra transacción ni resuelve otra identidad sustituta.
type RevalidadorVinculoCertificado interface {
	RevalidarVinculoCertificadoFirmante(context.Context, VinculoCertificadoFirmante) error
}

// RevalidadorRelacionCT protege el vínculo expediente/unidad y documento
// resueltos por CT dentro de la misma transacción del efecto o del replay.
// La implementación pertenece al propietario CT; no abre transacciones.
type RevalidadorRelacionCT interface {
	RevalidarRelacionRecursoCT(context.Context, RelacionRecursoCT) error
}

// Revalidador conserva el contrato transaccional central de K. Sus dependencias
// protegen todas las fuentes hasta COMMIT y se invocan también para replay.
type Revalidador struct {
	asignaciones vecports.RevalidadorAsignacionesCompetencialesTransaccionV1
	certificados RevalidadorVinculoCertificado
	relaciones   RevalidadorRelacionCT
	reloj        Reloj
}

func NuevoRevalidador(asignaciones vecports.RevalidadorAsignacionesCompetencialesTransaccionV1, certificados RevalidadorVinculoCertificado, relaciones RevalidadorRelacionCT, reloj Reloj) (*Revalidador, error) {
	if interfazNula(asignaciones) || interfazNula(certificados) || interfazNula(relaciones) || interfazNula(reloj) {
		return nil, ctports.ErrCompetenciaFirmanteNoDisponible
	}
	return &Revalidador{asignaciones: asignaciones, certificados: certificados, relaciones: relaciones, reloj: reloj}, nil
}

func (r *Revalidador) RevalidarCompetenciaCentral(ctx context.Context, a Acreditacion) error {
	if ctx == nil || r == nil || interfazNula(r.asignaciones) || interfazNula(r.certificados) || interfazNula(r.relaciones) || interfazNula(r.reloj) {
		return ctports.ErrCompetenciaFirmanteNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !a.proyeccion.Vigente || !a.recurso.coincide(a.solicitud, a.relacion, a.proyeccion.Solicitud) ||
		a.evidencia.ValidarParaEn(a.solicitud, r.reloj.Ahora()) != nil ||
		a.vinculo.PrincipalRef != a.solicitud.PersonaRef || a.vinculo.CertificadoHuella != a.solicitud.CertificadoHuellaSHA256 {
		return ctports.ErrCompetenciaFirmanteNoAcreditada
	}
	if err := r.relaciones.RevalidarRelacionRecursoCT(ctx, a.relacion); err != nil {
		return errorFuente(ctx, err)
	}
	if err := r.certificados.RevalidarVinculoCertificadoFirmante(ctx, a.vinculo); err != nil {
		return errorFuente(ctx, err)
	}
	solicitud, err := clonarSolicitud(a.solicitud)
	if err != nil {
		return ctports.ErrCompetenciaFirmanteNoAcreditada
	}
	if err := r.asignaciones.RevalidarAsignacionCompetencialV1(ctx, solicitud, clonarEvidencia(a.evidencia)); err != nil {
		return errorFuente(ctx, err)
	}
	if err := r.relaciones.RevalidarRelacionRecursoCT(ctx, a.relacion); err != nil {
		return errorFuente(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !a.recurso.coincide(a.solicitud, a.relacion, a.proyeccion.Solicitud) || a.evidencia.ValidarParaEn(a.solicitud, r.reloj.Ahora()) != nil {
		return ctports.ErrCompetenciaFirmanteNoAcreditada
	}
	return nil
}
