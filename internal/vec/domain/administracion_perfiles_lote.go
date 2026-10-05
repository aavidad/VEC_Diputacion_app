package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"time"
)

const MaximoCambiosLoteAdministracionPerfiles = 32

var organizacionLoteAdministracion = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,127}$`)

type InicioVigenciaLoteAdministracion string

const (
	InicioVigenciaLoteInmediato  InicioVigenciaLoteAdministracion = "inmediato"
	InicioVigenciaLoteProgramado InicioVigenciaLoteAdministracion = "programado"
)

// CambioPerfilAdministracion no acepta clase ni concesiones del cliente. La
// autoridad consulta la versión exacta publicada y reconstruye su circuito.
type CambioPerfilAdministracion struct {
	Operacion      OperacionAdministracionPerfiles
	InicioVigencia InicioVigenciaLoteAdministracion
	RolVersionRef  string
	Objetivo       PreimagenAdministracionPerfiles
}

func (c CambioPerfilAdministracion) validarLote() error {
	p := c.Objetivo
	if !c.Operacion.Valida() || !RolVersionAdministracionPerfilesValido(c.RolVersionRef) ||
		(p.UnidadRef != "" && !textoAutorizacionSinComodinSeguro(p.UnidadRef, 256, false)) ||
		(p.CentroRef != "" && !textoAutorizacionSinComodinSeguro(p.CentroRef, 256, false)) ||
		p.UnidadRef == "" || p.CentroRef != "" ||
		!referenciaOpacaAdministracionPerfiles(p.CuentaRef, "cta_") || p.CuentaVersion == 0 ||
		!referenciaOpacaAdministracionPerfiles(p.PersonaRef, "per_") || p.PersonaVersion == 0 ||
		!referenciaOpacaAdministracionPerfiles(p.PerfilRef, "prf_") ||
		!referenciaOpacaAdministracionPerfiles(p.VinculoRef, "vca_") ||
		!huellaAdministracionPerfiles(p.HuellaSHA256) ||
		!procedenciaAdministracionPerfiles(p.ProcedenciaRef) || p.ProcedenciaVersion == 0 ||
		!huellaAdministracionPerfiles(p.ProcedenciaHuellaSHA256) {
		return ErrActoAdministracionPerfilesInvalido
	}
	switch c.Operacion {
	case OperacionOtorgarPerfil:
		if p.PerfilVersion != 0 || p.VinculoVersion != 0 ||
			!instanteContextoActorCanonico(p.VigenteHasta) {
			return ErrActoAdministracionPerfilesInvalido
		}
		switch c.InicioVigencia {
		case InicioVigenciaLoteInmediato:
			if !p.VigenteDesde.IsZero() {
				return ErrActoAdministracionPerfilesInvalido
			}
		case InicioVigenciaLoteProgramado:
			if !instanteContextoActorCanonico(p.VigenteDesde) || !p.VigenteHasta.After(p.VigenteDesde) {
				return ErrActoAdministracionPerfilesInvalido
			}
		default:
			return ErrActoAdministracionPerfilesInvalido
		}
	case OperacionRevocarPerfil:
		// CA20 sube juntas las versiones de perfil y vínculo; una lectura que
		// las desacople debe cambiar también esta comprobación.
		if c.InicioVigencia != "" || p.PerfilVersion == 0 || p.VinculoVersion == 0 ||
			p.PerfilVersion != p.VinculoVersion ||
			!p.VigenteDesde.IsZero() || !p.VigenteHasta.IsZero() {
			return ErrActoAdministracionPerfilesInvalido
		}
	}
	return nil
}

// SolicitudLoteAdministracionPerfiles es una única orden con preimagen CAS de
// una persona y cuenta. Su huella conserva el orden de los cambios. Publicar
// otra versión del rol no migra asignaciones históricas.
type SolicitudLoteAdministracionPerfiles struct {
	OperacionRef            string
	OrganizacionRef         string
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
		!organizacionLoteAdministracion.MatchString(s.OrganizacionRef) ||
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
		if c.validarLote() != nil ||
			p.PersonaRef == s.Actor.PersonaRef || p.PersonaRef != primera.PersonaRef ||
			p.PersonaVersion != primera.PersonaVersion || p.CuentaRef != primera.CuentaRef ||
			p.CuentaVersion != primera.CuentaVersion ||
			p.ProcedenciaRef != primera.ProcedenciaRef || p.ProcedenciaVersion != primera.ProcedenciaVersion ||
			p.ProcedenciaHuellaSHA256 != primera.ProcedenciaHuellaSHA256 ||
			p.UnidadRef != primera.UnidadRef ||
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
		OrganizacionRef string
		Cambios         []CambioPerfilAdministracion
		Motivo          ReferenciaEntradaCatalogo
		ReferenciaActo  string
	}{"administracion_perfiles_lote:v3", s.OperacionRef, s.Actor.PersonaRef, s.Actor.PerfilActivoRef,
		s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), s.OrganizacionRef, s.Cambios, s.Motivo, s.ReferenciaActo}
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
// FuentesSHA256 es la huella que calcula la autoridad de los descriptores de
// ámbito (organización y unidad) cotejados con sus fuentes propietarias en la
// misma transacción; el cliente nunca la aporta. Inicios lleva, por cambio, el
// instante efectivo: en «inmediato» es el del COMMIT (ConfirmadoEn); en
// «programado», la fecha futura pedida. Las bajas llevan un inicio vacío.
type ReciboLoteAdministracionPerfiles struct {
	OperacionRef          string
	ActoRef               string
	ReciboRef             string
	AuditoriaRef          string
	HuellaSolicitudSHA256 string
	FuentesSHA256         string
	ConfirmadoEn          time.Time
	Cambios               []ReciboAdministracionPerfiles
	Inicios               []InicioEfectivoLoteAdministracion
}

type InicioEfectivoLoteAdministracion struct {
	Modo         InicioVigenciaLoteAdministracion
	VigenteDesde time.Time
}

func (r ReciboLoteAdministracionPerfiles) ValidarPara(s SolicitudLoteAdministracionPerfiles) error {
	if s.Validar() != nil || r.OperacionRef != s.OperacionRef ||
		r.HuellaSolicitudSHA256 != s.HuellaSolicitudSHA256 ||
		!HuellaAdministracionPerfilesValida(r.FuentesSHA256) ||
		!ReferenciaAdministracionPerfilesValida(r.ActoRef, "acto_admin:") ||
		!ReferenciaAdministracionPerfilesValida(r.ReciboRef, "recibo_admin:") ||
		!referenciaAuditoriaAdministracionPerfiles(r.AuditoriaRef) ||
		!instanteContextoActorCanonico(r.ConfirmadoEn) || len(r.Cambios) != len(s.Cambios) ||
		len(r.Inicios) != len(s.Cambios) {
		return ErrActoAdministracionPerfilesInvalido
	}
	for i, c := range s.Cambios {
		recibo := r.Cambios[i]
		inicio := r.Inicios[i]
		if recibo.Validar() != nil || recibo.OperacionRef != s.OperacionRef || recibo.PropuestaRef != "" ||
			recibo.ActorPersonaRef != s.Actor.PersonaRef || recibo.PerfilActivoRef != s.Actor.PerfilActivoRef ||
			recibo.AsignacionPerfilRef != s.InstantaneaAutorizacion.AsignacionPerfil.Referencia() ||
			!ReferenciaCorrelacionAutorizacionV2Valida(recibo.CorrelacionRef) || recibo.Motivo != s.Motivo ||
			recibo.ObjetivoPersonaRef != c.Objetivo.PersonaRef || recibo.PerfilRef != c.Objetivo.PerfilRef ||
			recibo.VinculoRef != c.Objetivo.VinculoRef || recibo.UnidadRef != c.Objetivo.UnidadRef ||
			recibo.CentroRef != c.Objetivo.CentroRef || recibo.RolVersionRef != c.RolVersionRef ||
			recibo.ReferenciaActo != s.ReferenciaActo || recibo.HuellaAntesSHA256 != c.Objetivo.HuellaSHA256 ||
			recibo.ActoRef != r.ActoRef ||
			recibo.ReciboRef != r.ReciboRef || recibo.AuditoriaRef != r.AuditoriaRef ||
			recibo.CorrelacionRef != r.Cambios[0].CorrelacionRef ||
			!recibo.ConfirmadoEn.Equal(r.ConfirmadoEn) {
			return ErrActoAdministracionPerfilesInvalido
		}
		switch c.Operacion {
		case OperacionOtorgarPerfil:
			if recibo.EstadoPosterior != EstadoVinculoContextoActorActivo || recibo.VersionPosterior != 1 ||
				inicio.Modo != c.InicioVigencia || !inicio.VigenteDesde.Equal(recibo.VigenteDesde) ||
				!recibo.VigenteHasta.Equal(c.Objetivo.VigenteHasta) || !recibo.VigenteHasta.After(inicio.VigenteDesde) {
				return ErrActoAdministracionPerfilesInvalido
			}
			if c.InicioVigencia == InicioVigenciaLoteInmediato {
				if !inicio.VigenteDesde.Equal(r.ConfirmadoEn) {
					return ErrActoAdministracionPerfilesInvalido
				}
			} else if !inicio.VigenteDesde.Equal(c.Objetivo.VigenteDesde) || !inicio.VigenteDesde.After(r.ConfirmadoEn) {
				return ErrActoAdministracionPerfilesInvalido
			}
		case OperacionRevocarPerfil:
			if inicio.Modo != "" || !inicio.VigenteDesde.IsZero() ||
				recibo.EstadoPosterior != EstadoVinculoContextoActorRevocado ||
				recibo.VersionPosterior != c.Objetivo.VinculoVersion+1 ||
				c.Objetivo.VinculoVersion == ^uint64(0) {
				return ErrActoAdministracionPerfilesInvalido
			}
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
