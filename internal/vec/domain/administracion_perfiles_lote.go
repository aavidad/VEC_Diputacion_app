package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

const MaximoCambiosLoteAdministracionPerfiles = 32

// CambioPerfilAdministracion no acepta clase ni concesiones del cliente. La
// autoridad consulta la versión exacta publicada y reconstruye su circuito.
type CambioPerfilAdministracion struct {
	Operacion     OperacionAdministracionPerfiles
	RolVersionRef string
	Objetivo      PreimagenAdministracionPerfiles
}

// SolicitudLoteAdministracionPerfiles es una única orden con preimagen CAS de
// una persona y cuenta. Su huella conserva el orden de los cambios. Publicar
// otra versión del rol no migra asignaciones históricas.
type SolicitudLoteAdministracionPerfiles struct {
	OperacionRef            string
	Actor                   ContextoActor
	Evidencia               EvidenciaSesionAdministracionPerfiles `json:"-"`
	InstantaneaAutorizacion InstantaneaAutorizacion
	Cambios                 []CambioPerfilAdministracion
	Motivo                  ReferenciaEntradaCatalogo
	ReferenciaActo          string
	CorrelacionRef          string
	HuellaSolicitudSHA256   string
}

func (s SolicitudLoteAdministracionPerfiles) validarEstructura() error {
	if !ReferenciaAdministracionPerfilesValida(s.OperacionRef, "acto_admin:") ||
		s.Actor.Validar() != nil || s.InstantaneaAutorizacion.Validar() != nil ||
		s.Motivo.Validar() != nil || !ReferenciaActoAdministracionValida(s.ReferenciaActo) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) ||
		s.Actor.PersonaRef != s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		s.Actor.PerfilActivoRef != s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		len(s.Cambios) == 0 || len(s.Cambios) > MaximoCambiosLoteAdministracionPerfiles {
		return ErrActoAdministracionPerfilesInvalido
	}
	perfiles := make(map[string]bool, len(s.Cambios))
	vinculos := make(map[string]bool, len(s.Cambios))
	rolesAmbitos := make(map[string]bool, len(s.Cambios))
	primera := s.Cambios[0].Objetivo
	for _, c := range s.Cambios {
		p := c.Objetivo
		clave := c.RolVersionRef + "\x00" + p.UnidadRef + "\x00" + p.CentroRef
		if !RolVersionAdministracionPerfilesValido(c.RolVersionRef) ||
			p.ValidarPara(c.Operacion, ClaseControlPerfilOrdinario) != nil ||
			p.PersonaRef == s.Actor.PersonaRef || p.PersonaRef != primera.PersonaRef ||
			p.PersonaVersion != primera.PersonaVersion || p.CuentaRef != primera.CuentaRef ||
			p.CuentaVersion != primera.CuentaVersion ||
			p.ProcedenciaRef != primera.ProcedenciaRef || p.ProcedenciaVersion != primera.ProcedenciaVersion ||
			p.ProcedenciaHuellaSHA256 != primera.ProcedenciaHuellaSHA256 ||
			perfiles[p.PerfilRef] || vinculos[p.VinculoRef] || rolesAmbitos[clave] {
			return ErrActoAdministracionPerfilesInvalido
		}
		perfiles[p.PerfilRef], vinculos[p.VinculoRef], rolesAmbitos[clave] = true, true, true
	}
	return nil
}

// CanonicoYHuella liga la preimagen completa, el actor, el perfil y la referencia
// de asignación a la orden. Correlación, evidencia e instantánea actuales se
// revalidan separadamente en cada acceso, también al recuperar el recibo original.
func (s SolicitudLoteAdministracionPerfiles) CanonicoYHuella() ([]byte, string, error) {
	if err := s.validarEstructura(); err != nil {
		return nil, "", err
	}
	canonico := struct {
		Esquema         string
		OperacionRef    string
		ActorPersonaRef string
		PerfilActivoRef string
		AsignacionRef   string
		Cambios         []CambioPerfilAdministracion
		Motivo          ReferenciaEntradaCatalogo
		ReferenciaActo  string
	}{"administracion_perfiles_lote:v2", s.OperacionRef, s.Actor.PersonaRef, s.Actor.PerfilActivoRef,
		s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), s.Cambios, s.Motivo, s.ReferenciaActo}
	b, err := json.Marshal(canonico)
	if err != nil {
		return nil, "", ErrActoAdministracionPerfilesInvalido
	}
	h := sha256.Sum256(b)
	return b, hex.EncodeToString(h[:]), nil
}

func (s SolicitudLoteAdministracionPerfiles) Validar() error {
	_, h, err := s.CanonicoYHuella()
	if err != nil || !HuellaAdministracionPerfilesValida(s.HuellaSolicitudSHA256) || h != s.HuellaSolicitudSHA256 {
		return ErrActoAdministracionPerfilesInvalido
	}
	return nil
}

// ReciboLoteAdministracionPerfiles se emite sólo tras el commit íntegro. Sus
// recibos por cambio comparten acto, auditoría, fecha y correlación de la orden.
type ReciboLoteAdministracionPerfiles struct {
	OperacionRef          string
	ActoRef               string
	ReciboRef             string
	AuditoriaRef          string
	HuellaSolicitudSHA256 string
	ConfirmadoEn          time.Time
	Cambios               []ReciboAdministracionPerfiles
}

func (r ReciboLoteAdministracionPerfiles) ValidarPara(s SolicitudLoteAdministracionPerfiles) error {
	if s.Validar() != nil || r.OperacionRef != s.OperacionRef ||
		r.HuellaSolicitudSHA256 != s.HuellaSolicitudSHA256 ||
		!ReferenciaAdministracionPerfilesValida(r.ActoRef, "acto_admin:") ||
		!ReferenciaAdministracionPerfilesValida(r.ReciboRef, "recibo_admin:") ||
		!referenciaAuditoriaAdministracionPerfiles(r.AuditoriaRef) ||
		!instanteContextoActorCanonico(r.ConfirmadoEn) || len(r.Cambios) != len(s.Cambios) {
		return ErrActoAdministracionPerfilesInvalido
	}
	for i, c := range s.Cambios {
		recibo := r.Cambios[i]
		solicitud := SolicitudActoAdministracionPerfiles{OperacionRef: s.OperacionRef, Actor: s.Actor,
			Evidencia: s.Evidencia, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
			Operacion: c.Operacion, Clase: ClaseControlPerfilOrdinario, RolVersionRef: c.RolVersionRef,
			Objetivo: c.Objetivo, Motivo: s.Motivo, CorrelacionRef: s.CorrelacionRef, ReferenciaActo: s.ReferenciaActo}
		if recibo.ValidarPara(solicitud) != nil || recibo.ActoRef != r.ActoRef ||
			recibo.ReciboRef != r.ReciboRef || recibo.AuditoriaRef != r.AuditoriaRef ||
			recibo.CorrelacionRef != r.Cambios[0].CorrelacionRef ||
			!recibo.ConfirmadoEn.Equal(r.ConfirmadoEn) {
			return ErrActoAdministracionPerfilesInvalido
		}
	}
	return nil
}

func (r ReciboAdministracionPerfiles) ValidarPara(s SolicitudActoAdministracionPerfiles) error {
	if s.Validar() != nil || r.Validar() != nil || r.OperacionRef != s.OperacionRef || r.PropuestaRef != "" ||
		r.ActorPersonaRef != s.Actor.PersonaRef || r.PerfilActivoRef != s.Actor.PerfilActivoRef ||
		r.AsignacionPerfilRef != s.InstantaneaAutorizacion.AsignacionPerfil.Referencia() ||
		!ReferenciaCorrelacionAutorizacionV2Valida(r.CorrelacionRef) || r.Motivo != s.Motivo ||
		r.ObjetivoPersonaRef != s.Objetivo.PersonaRef || r.PerfilRef != s.Objetivo.PerfilRef ||
		r.VinculoRef != s.Objetivo.VinculoRef || r.UnidadRef != s.Objetivo.UnidadRef || r.CentroRef != s.Objetivo.CentroRef ||
		r.RolVersionRef != s.RolVersionRef || r.ReferenciaActo != s.ReferenciaActo ||
		r.HuellaAntesSHA256 != s.Objetivo.HuellaSHA256 {
		return ErrActoAdministracionPerfilesInvalido
	}
	if s.Operacion == OperacionOtorgarPerfil && (r.EstadoPosterior != EstadoVinculoContextoActorActivo ||
		r.VersionPosterior != 1 || !r.VigenteDesde.Equal(s.Objetivo.VigenteDesde) || !r.VigenteHasta.Equal(s.Objetivo.VigenteHasta)) {
		return ErrActoAdministracionPerfilesInvalido
	}
	if s.Operacion == OperacionRevocarPerfil && (r.EstadoPosterior != EstadoVinculoContextoActorRevocado ||
		r.VersionPosterior != s.Objetivo.VinculoVersion+1 || s.Objetivo.VinculoVersion == ^uint64(0)) {
		return ErrActoAdministracionPerfilesInvalido
	}
	return nil
}
