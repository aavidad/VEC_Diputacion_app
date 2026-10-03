package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrActoAdministracionPerfilesInvalido    = errors.New("vec: acto de administracion de perfiles invalido")
	ErrControlAdministracionPerfilesInvalido = errors.New("vec: control de administracion de perfiles invalido")
)

// ClaseControlAdministracionPerfiles determina el circuito, no los permisos.
// La clase y la version del rol deben proceder del catalogo publicado y ser
// comprobadas otra vez por la autoridad durable al aplicar el acto.
type ClaseControlAdministracionPerfiles string

const (
	ClaseControlPerfilOrdinario     ClaseControlAdministracionPerfiles = "ordinario"
	ClaseControlPerfilAdministrador ClaseControlAdministracionPerfiles = "administrador"
	ClaseControlPerfilIntervencion  ClaseControlAdministracionPerfiles = "intervencion"
)

func (c ClaseControlAdministracionPerfiles) Valida() bool {
	return c == ClaseControlPerfilOrdinario || c == ClaseControlPerfilAdministrador ||
		c == ClaseControlPerfilIntervencion
}

func (c ClaseControlAdministracionPerfiles) RequiereDobleControl() bool {
	return c == ClaseControlPerfilAdministrador || c == ClaseControlPerfilIntervencion
}

// PuedeProponerBajaPropia permite que quien deja el perfil administrador
// proponga su revocacion. El cierre sigue exigiendo aprobacion de otra persona.
// No se extiende al alta ni a Intervencion sin una fuente que lo autorice.
func PuedeProponerBajaPropia(clase ClaseControlAdministracionPerfiles, operacion OperacionAdministracionPerfiles) bool {
	return clase == ClaseControlPerfilAdministrador && operacion == OperacionRevocarPerfil
}

// ContinuidadAdministradores separa la guarda de disponibilidad del numero
// de personas necesario para nuevos actos con doble control. Arrancar exige
// dos; despues de una baja aprobada puede quedar una. Con una sola persona
// siguen posibles las operaciones ordinarias autorizadas, pero no un nuevo
// acto sensible por el circuito normal de dos personas.
type ContinuidadAdministradores struct {
	EfectivosAntes   uint64
	EfectivosDespues uint64
}

func (c ContinuidadAdministradores) ValidarBaja() error {
	if c.EfectivosAntes < 2 || c.EfectivosDespues < 1 ||
		c.EfectivosAntes <= c.EfectivosDespues || c.EfectivosAntes-c.EfectivosDespues != 1 {
		return ErrControlAdministracionPerfilesInvalido
	}
	return nil
}

func (c ContinuidadAdministradores) AdmiteNuevoActoSensible() bool {
	return c.EfectivosDespues >= 2
}

type OperacionAdministracionPerfiles string

const (
	OperacionOtorgarPerfil OperacionAdministracionPerfiles = "otorgar"
	OperacionRevocarPerfil OperacionAdministracionPerfiles = "revocar"
)

func (o OperacionAdministracionPerfiles) Valida() bool {
	return o == OperacionOtorgarPerfil || o == OperacionRevocarPerfil
}

// PreimagenAdministracionPerfiles fija el estado que el adaptador deberá
// cotejar bajo bloqueo. Un otorgamiento propone referencias nuevas con
// versiones cero; la autoridad durable comprueba su ausencia antes de
// crearlas. Una revocacion exige referencias y versiones actuales exactas.
type PreimagenAdministracionPerfiles struct {
	// UnidadRef identifica el ámbito elegido de la lectura central. La autoridad
	// vuelve a cotejarlo; un valor declarado no acredita competencia.
	UnidadRef               string
	CentroRef               string
	CuentaRef               string
	CuentaVersion           uint64
	PersonaRef              string
	PersonaVersion          uint64
	PerfilRef               string
	PerfilVersion           uint64
	VinculoRef              string
	VinculoVersion          uint64
	HuellaSHA256            string
	RevisionContinuidad     uint64
	ProcedenciaRef          string
	ProcedenciaVersion      uint64
	ProcedenciaHuellaSHA256 string
	VigenteDesde            time.Time
	VigenteHasta            time.Time
}

func (p PreimagenAdministracionPerfiles) ValidarPara(operacion OperacionAdministracionPerfiles, clase ClaseControlAdministracionPerfiles) error {
	if (p.UnidadRef != "" && !textoAutorizacionSinComodinSeguro(p.UnidadRef, 256, false)) ||
		(p.CentroRef != "" && !textoAutorizacionSinComodinSeguro(p.CentroRef, 256, false)) ||
		(p.UnidadRef == "" && p.CentroRef == "") ||
		!referenciaOpacaAdministracionPerfiles(p.CuentaRef, "cta_") || p.CuentaVersion == 0 ||
		!referenciaOpacaAdministracionPerfiles(p.PersonaRef, "per_") || p.PersonaVersion == 0 ||
		!huellaAdministracionPerfiles(p.HuellaSHA256) || !procedenciaAdministracionPerfiles(p.ProcedenciaRef) ||
		p.ProcedenciaVersion == 0 || !huellaAdministracionPerfiles(p.ProcedenciaHuellaSHA256) ||
		!operacion.Valida() || !clase.Valida() {
		return ErrActoAdministracionPerfilesInvalido
	}
	if clase.RequiereDobleControl() && p.RevisionContinuidad == 0 {
		return ErrControlAdministracionPerfilesInvalido
	}
	if operacion == OperacionOtorgarPerfil &&
		(!referenciaOpacaAdministracionPerfiles(p.PerfilRef, "prf_") || p.PerfilVersion != 0 ||
			!referenciaOpacaAdministracionPerfiles(p.VinculoRef, "vca_") || p.VinculoVersion != 0 ||
			!instanteContextoActorCanonico(p.VigenteDesde) ||
			!instanteContextoActorCanonico(p.VigenteHasta) || !p.VigenteHasta.After(p.VigenteDesde)) {
		return ErrActoAdministracionPerfilesInvalido
	}
	if operacion == OperacionRevocarPerfil &&
		(!referenciaOpacaAdministracionPerfiles(p.PerfilRef, "prf_") || p.PerfilVersion == 0 ||
			!referenciaOpacaAdministracionPerfiles(p.VinculoRef, "vca_") || p.VinculoVersion == 0 ||
			(!p.VigenteDesde.IsZero() || !p.VigenteHasta.IsZero())) {
		return ErrActoAdministracionPerfilesInvalido
	}
	return nil
}

// ValidarBootstrap admite la continuidad inicial vacia, pero nunca una
// cuenta/persona sin versiones acreditadas ni un perfil previo reciclado.
func (p PreimagenAdministracionPerfiles) ValidarBootstrap() error {
	if !referenciaOpacaAdministracionPerfiles(p.CuentaRef, "cta_") || p.CuentaVersion == 0 ||
		!referenciaOpacaAdministracionPerfiles(p.PersonaRef, "per_") || p.PersonaVersion == 0 ||
		!huellaAdministracionPerfiles(p.HuellaSHA256) || p.RevisionContinuidad != 0 ||
		!procedenciaAdministracionPerfiles(p.ProcedenciaRef) || p.ProcedenciaVersion == 0 ||
		!huellaAdministracionPerfiles(p.ProcedenciaHuellaSHA256) ||
		!instanteContextoActorCanonico(p.VigenteHasta) ||
		!referenciaOpacaAdministracionPerfiles(p.PerfilRef, "prf_") || p.PerfilVersion != 0 ||
		!referenciaOpacaAdministracionPerfiles(p.VinculoRef, "vca_") || p.VinculoVersion != 0 {
		return ErrControlAdministracionPerfilesInvalido
	}
	return nil
}

func procedenciaAdministracionPerfiles(valor string) bool {
	return len(valor) > 0 && len(valor) <= 256 && !strings.ContainsAny(valor, "* \t\n\r")
}

// SolicitudActoAdministracionPerfiles es una orden interna. Su validacion
// estructural no concede acceso; el efecto exige el PDP V3 y el CAS durable
// en la misma transaccion. RolVersionRef es una seleccion de catalogo, nunca
// una definicion de permisos enviada por el cliente.
type SolicitudActoAdministracionPerfiles struct {
	// ReferenciaActo es una referencia administrativa opcional. No sustituye
	// la procedencia técnica ni concede permisos.
	ReferenciaActo          string
	OperacionRef            string
	Actor                   ContextoActor
	Evidencia               EvidenciaSesionAdministracionPerfiles `json:"-"`
	InstantaneaAutorizacion InstantaneaAutorizacion
	Operacion               OperacionAdministracionPerfiles
	Clase                   ClaseControlAdministracionPerfiles
	RolVersionRef           string
	Objetivo                PreimagenAdministracionPerfiles
	Motivo                  ReferenciaEntradaCatalogo
	CorrelacionRef          string
}

func (s SolicitudActoAdministracionPerfiles) Validar() error {
	if !ReferenciaActoAdministracionValida(s.ReferenciaActo) ||
		(!referenciaHexAdministracionPerfiles(s.OperacionRef, "acto_admin:") &&
			!referenciaHexAdministracionPerfiles(s.OperacionRef, "propuesta_admin:")) ||
		s.Actor.Validar() != nil || s.InstantaneaAutorizacion.Validar() != nil ||
		!s.Operacion.Valida() || !s.Clase.Valida() ||
		!rolVersionAdministracionPerfiles(s.RolVersionRef) ||
		s.Objetivo.ValidarPara(s.Operacion, s.Clase) != nil ||
		s.Motivo.Validar() != nil ||
		!ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) ||
		(s.Actor.PersonaRef == s.Objetivo.PersonaRef && !PuedeProponerBajaPropia(s.Clase, s.Operacion)) ||
		s.Actor.PerfilActivoRef != s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		s.Actor.PersonaRef != s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID {
		return ErrActoAdministracionPerfilesInvalido
	}
	return nil
}

// DecisionPropuestaAdministracionPerfiles cierra una propuesta una sola vez.
// La aprobacion no admite alterar el objetivo, rol, operacion ni preimagen:
// esos datos se recuperan de la propuesta inmutable por referencia y huella.
type DecisionPropuestaAdministracionPerfiles string

const (
	DecisionAprobarPropuestaPerfil  DecisionPropuestaAdministracionPerfiles = "aprobada"
	DecisionRechazarPropuestaPerfil DecisionPropuestaAdministracionPerfiles = "rechazada"
)

func (d DecisionPropuestaAdministracionPerfiles) Valida() bool {
	return d == DecisionAprobarPropuestaPerfil || d == DecisionRechazarPropuestaPerfil
}

type SolicitudCierrePropuestaAdministracionPerfiles struct {
	OperacionRef            string
	PropuestaRef            string
	PropuestaHuellaSHA256   string
	ProponentePersonaRef    string
	ObjetivoPersonaRef      string
	Aprobador               ContextoActor
	Evidencia               EvidenciaSesionAdministracionPerfiles `json:"-"`
	InstantaneaAutorizacion InstantaneaAutorizacion
	Decision                DecisionPropuestaAdministracionPerfiles
	Motivo                  ReferenciaEntradaCatalogo
	CorrelacionRef          string
}

func (s SolicitudCierrePropuestaAdministracionPerfiles) Validar() error {
	if !referenciaHexAdministracionPerfiles(s.OperacionRef, "cierre_admin:") ||
		!referenciaHexAdministracionPerfiles(s.PropuestaRef, "propuesta_admin:") ||
		!huellaAdministracionPerfiles(s.PropuestaHuellaSHA256) ||
		!referenciaOpacaAdministracionPerfiles(s.ProponentePersonaRef, "per_") ||
		!referenciaOpacaAdministracionPerfiles(s.ObjetivoPersonaRef, "per_") ||
		s.Aprobador.Validar() != nil || s.InstantaneaAutorizacion.Validar() != nil ||
		!s.Decision.Valida() || s.Motivo.Validar() != nil ||
		!ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef) ||
		s.Aprobador.PersonaRef == s.ProponentePersonaRef ||
		s.Aprobador.PersonaRef == s.ObjetivoPersonaRef ||
		s.Aprobador.PersonaRef != s.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		s.Aprobador.PerfilActivoRef != s.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef {
		return ErrControlAdministracionPerfilesInvalido
	}
	return nil
}

// ReciboAdministracionPerfiles acredita exclusivamente un COMMIT durable.
// La autoridad de persistencia lo devuelve con auditoria e historia de solo
// adicion; una propuesta pendiente no es un acto aplicado.
type ReciboAdministracionPerfiles struct {
	UnidadRef           string
	CentroRef           string
	ActorPersonaRef     string
	PerfilActivoRef     string
	AsignacionPerfilRef string
	CorrelacionRef      string
	RolVersionRef       string
	VigenteDesde        time.Time
	VigenteHasta        time.Time
	Motivo              ReferenciaEntradaCatalogo
	ReferenciaActo      string
	OperacionRef        string
	ActoRef             string
	ReciboRef           string
	PropuestaRef        string
	AuditoriaRef        string
	ObjetivoPersonaRef  string
	PerfilRef           string
	VinculoRef          string
	EstadoPosterior     EstadoVinculoContextoActor
	VersionPosterior    uint64
	HuellaAntesSHA256   string
	HuellaDespuesSHA256 string
	ConfirmadoEn        time.Time
}

func (r ReciboAdministracionPerfiles) Validar() error {
	if !ReferenciaActoAdministracionValida(r.ReferenciaActo) || (r.UnidadRef != "" && !textoAutorizacionSinComodinSeguro(r.UnidadRef, 256, false)) {
		return ErrActoAdministracionPerfilesInvalido
	}
	if !referenciaHexAdministracionPerfiles(r.OperacionRef, "acto_admin:") &&
		!referenciaHexAdministracionPerfiles(r.OperacionRef, "cierre_admin:") ||
		!referenciaHexAdministracionPerfiles(r.ActoRef, "acto_admin:") ||
		!referenciaHexAdministracionPerfiles(r.ReciboRef, "recibo_admin:") ||
		(r.PropuestaRef != "" && !referenciaHexAdministracionPerfiles(r.PropuestaRef, "propuesta_admin:")) ||
		!referenciaAuditoriaAdministracionPerfiles(r.AuditoriaRef) ||
		!referenciaOpacaAdministracionPerfiles(r.ObjetivoPersonaRef, "per_") ||
		!referenciaOpacaAdministracionPerfiles(r.PerfilRef, "prf_") ||
		!referenciaOpacaAdministracionPerfiles(r.VinculoRef, "vca_") ||
		r.VersionPosterior == 0 || !r.EstadoPosterior.Valido() ||
		!huellaAdministracionPerfiles(r.HuellaAntesSHA256) ||
		!huellaAdministracionPerfiles(r.HuellaDespuesSHA256) ||
		!instanteContextoActorCanonico(r.ConfirmadoEn) {
		return ErrActoAdministracionPerfilesInvalido
	}
	return nil
}

func referenciaOpacaAdministracionPerfiles(valor, prefijo string) bool {
	if !strings.HasPrefix(valor, prefijo) || len(valor) < len(prefijo)+22 || len(valor) > len(prefijo)+128 {
		return false
	}
	for _, c := range valor[len(prefijo):] {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func referenciaHexAdministracionPerfiles(valor, prefijo string) bool {
	if !strings.HasPrefix(valor, prefijo) || len(valor) != len(prefijo)+32 {
		return false
	}
	for _, c := range valor[len(prefijo):] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func referenciaAuditoriaAdministracionPerfiles(valor string) bool {
	if len(valor) == 0 || len(valor) > 256 || strings.ContainsAny(valor, "* \t\n\r") {
		return false
	}
	return true
}

func huellaAdministracionPerfiles(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	for _, c := range valor {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func HuellaAdministracionPerfilesValida(valor string) bool {
	return huellaAdministracionPerfiles(valor)
}
func RolVersionAdministracionPerfilesValido(valor string) bool {
	return rolVersionAdministracionPerfiles(valor)
}
func ReferenciaAdministracionPerfilesValida(valor, prefijo string) bool {
	switch prefijo {
	case "propuesta_admin:", "cierre_admin:", "acto_admin:", "recibo_admin:":
		return referenciaHexAdministracionPerfiles(valor, prefijo)
	default:
		return false
	}
}

func rolVersionAdministracionPerfiles(valor string) bool {
	if !strings.HasPrefix(valor, "rol:") || len(valor) > 256 || strings.ContainsAny(valor, "* \t\n\r") {
		return false
	}
	partes := strings.Split(valor, ":")
	if len(partes) != 3 || partes[1] == "" || len(partes[2]) < 2 || partes[2][0] != 'v' {
		return false
	}
	for _, c := range partes[1] {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	if partes[2][1] == '0' {
		return false
	}
	for _, c := range partes[2][1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ReferenciaActoAdministracionValida permite una referencia legible, sin
// controles ni contenido multilínea que se confunda con otra anotación.
func ReferenciaActoAdministracionValida(valor string) bool {
	if !utf8.ValidString(valor) || utf8.RuneCountInString(valor) > 256 || strings.TrimSpace(valor) != valor {
		return false
	}
	for _, c := range valor {
		if unicode.IsControl(c) || c == '\u2028' || c == '\u2029' {
			return false
		}
	}
	return true
}
