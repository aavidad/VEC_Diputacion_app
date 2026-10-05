package intentoscopias

import (
	"context"

	http "vec-diputacion-granada/internal/modules/administracion/adapters/httpcopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
	d "vec-diputacion-granada/internal/vec/domain"
	v "vec-diputacion-granada/internal/vec/ports"
)

// Acreditacion is the original recorded identity of one exact failed attempt.
// CorrelacionRef must come from the trusted attempt source, not be echoed from
// the query or reconstructed from a current person/profile lookup.
type Acreditacion struct {
	CorrelacionRef string
	Resultado      d.ResultadoContextoActorRegistradoV2
	Vinculo        d.VinculoAutenticacionActorV2
}

// FuenteIntento retrieves the original bound pair by exact server correlation.
// A revoked attempt may retain its original accreditation: this is evidence,
// not a new session resolution or a grant of permission. Never wire a test
// factory, caller-provided DTO or person/profile-only source here in production.
type FuenteIntento interface {
	RecuperarIntento(context.Context, string) (Acreditacion, error)
}

// Registrar is compatible with httpcopias.AuditorFrontera. It records a failed
// operation only; successful business effects retain their own atomic audit.
// One call invokes at most one common Append; no retry can repeat the effect.
func (a *Auditor) Registrar(ctx context.Context, intento http.Denegacion) error {
	if ctx == nil || a == nil || ausente(a.fuente) || ausente(a.registro) || a.noNominal == nil {
		return p.ErrNoDisponible
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.config.Plazo)
	defer cancel()
	motivo, existe := a.config.Motivos[intento.Codigo]
	if !existe {
		return p.ErrNoDisponible
	}
	// Pre-session failures use their own non-nominal boundary.
	// Partial identities fail closed; they are never completed with fabricated data.
	sinIdentidad := intento.ActorPersonaRef == "" && intento.PerfilActivoRef == "" && intento.CorrelacionRef == ""
	if !sinIdentidad && (intento.ActorPersonaRef == "" || intento.PerfilActivoRef == "" || !d.ReferenciaCorrelacionAutorizacionV2Valida(intento.CorrelacionRef)) {
		return p.ErrNoDisponible
	}
	if sinIdentidad {
		if a.noNominal(ctx, intento) != nil {
			return p.ErrNoDisponible
		}
		return nil
	}
	op, existe := a.config.Operaciones[p.Operacion(intento.Accion)]
	if intento.Accion == "" {
		op, existe = a.config.FronteraNominal, true
	}
	if !existe || intento.ActorPersonaRef == "" || intento.PerfilActivoRef == "" || !d.ReferenciaCorrelacionAutorizacionV2Valida(intento.CorrelacionRef) {
		return p.ErrNoDisponible
	}
	acreditacion, err := a.fuente.RecuperarIntento(ctx, intento.CorrelacionRef)
	if err != nil || ctx.Err() != nil || acreditacion.CorrelacionRef != intento.CorrelacionRef || acreditacion.Resultado.Validar() != nil || acreditacion.Vinculo.ValidarPara(acreditacion.Resultado) != nil || acreditacion.Resultado.Contexto.PersonaRef != intento.ActorPersonaRef || acreditacion.Resultado.Contexto.PerfilActivoRef != intento.PerfilActivoRef {
		return p.ErrNoDisponible
	}
	identidad, err := acreditacion.Vinculo.Datos()
	if err != nil || string(identidad.Superficie) != a.config.Canal || !identidad.CuentaPrivilegiada || identidad.MetodoObservado != d.AuthMethodCertificate || identidad.GarantiaObservada != d.AuthAssuranceHigh {
		return p.ErrNoDisponible
	}
	recurso := intento.RecursoRef
	if intento.Accion == "" || recurso == "" || recurso == "copias" {
		recurso = op.RecursoFallback
	}
	resultado := d.ResultadoIntentoAuditoriaError
	if intento.Codigo == "autenticacion_requerida" || intento.Codigo == "acceso_denegado" {
		resultado = d.ResultadoIntentoAuditoriaDenegado
	}
	datos := d.DatosIntentoAuditoria{Accion: op.Accion, ModuloID: "administracion", RecursoRef: recurso, FinalidadRef: op.FinalidadRef, Resultado: resultado, Motivo: motivo, Proceso: a.config.Proceso, Canal: a.config.Canal, CorrelacionRef: intento.CorrelacionRef}
	// HTTP references are broader than the common audit reference grammar.
	// Keep the failed attempt with its configured opaque fallback, never raw data.
	if datos.Validar() != nil {
		datos.RecursoRef = op.RecursoFallback
		if datos.Validar() != nil {
			return p.ErrNoDisponible
		}
	}
	ref, err := v.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return p.ErrNoDisponible
	}
	orden, err := v.NuevaOrdenIntentoAuditoria(ref, acreditacion.Resultado, acreditacion.Vinculo, datos)
	if err != nil || ctx.Err() != nil {
		return p.ErrNoDisponible
	}
	acuse, err := a.registro.AppendIntentoAuditoria(ctx, orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return p.ErrNoDisponible
	}
	return nil
}
