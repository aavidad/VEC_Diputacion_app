package domain

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrSolicitudAutorizacionInvalida = errors.New("vec: solicitud de autorizacion invalida")
	ErrConfiguracionAccesoInvalida   = errors.New("vec: configuracion de acceso invalida")
	ErrAutorizacionDenegada          = errors.New("vec: autorizacion denegada")
	ErrDecisionAutorizacionInvalida  = errors.New("vec: decision de autorizacion invalida")
)

const (
	// VigenciaMaximaDecisionAutorizacion limita la reutilizacion de una
	// decision. Un adaptador puede emitir decisiones con menor vigencia.
	VigenciaMaximaDecisionAutorizacion = 5 * time.Minute
	comodinAutorizacion                = "*"
	maximoElementosAutorizacion        = 512
)

// RecursoAutorizable contiene solo el contexto de recurso obtenido por el
// servidor. Los adaptadores de entrada no deben copiar atributos declarados
// por el cliente sin verificarlos antes.
type RecursoAutorizable struct {
	Referencia string            `json:"referencia"`
	ModuloID   string            `json:"modulo_id"`
	Tipo       string            `json:"tipo"`
	Ambitos    map[string]string `json:"ambitos,omitempty"`
	Atributos  map[string]string `json:"atributos,omitempty"`
}

type contextoRecursoAutorizacionCanonico struct {
	Ambitos   map[string]string `json:"ambitos"`
	Atributos map[string]string `json:"atributos"`
}

func (r RecursoAutorizable) Validar() error {
	if !textoAutorizacionSinComodinSeguro(r.Referencia, 512, false) ||
		!textoAutorizacionSinComodinSeguro(r.ModuloID, 128, false) ||
		!textoAutorizacionSinComodinSeguro(r.Tipo, 128, false) {
		return ErrSolicitudAutorizacionInvalida
	}
	if !mapaAutorizacionValido(r.Ambitos) || !mapaAutorizacionValido(r.Atributos) {
		return ErrSolicitudAutorizacionInvalida
	}
	for clave, valor := range r.Ambitos {
		if strings.ContainsRune(clave, '*') || strings.ContainsRune(valor, '*') {
			return ErrSolicitudAutorizacionInvalida
		}
	}
	return nil
}

// HuellaContextoAutorizacionSHA256 liga la decision a los ambitos y atributos
// resueltos por el servidor sin conservarlos en la evidencia. Los adaptadores
// deben aportar claves de catalogo o referencias opacas; si un valor sensible
// no puede evitarse, debe llegar tokenizado/HMAC antes de construir el recurso.
func (r RecursoAutorizable) HuellaContextoAutorizacionSHA256() (string, error) {
	if err := r.Validar(); err != nil {
		return "", err
	}
	return huellaAutorizacion(contextoRecursoAutorizacionCanonico{
		Ambitos: clonarMapaAutorizacion(r.Ambitos), Atributos: clonarMapaAutorizacion(r.Atributos),
	})
}

// SolicitudAutorizacion selecciona exactamente un perfil activo. Roles y
// permisos incluidos en Principal son informativos y nunca son autoridad para
// resolver esta solicitud.
type SolicitudAutorizacion struct {
	Principal       Principal `json:"principal"`
	PerfilActivoRef string    `json:"perfil_activo_ref"`
	// ContextoActor y VinculoAutenticacionActor son capacidades internas. No
	// deben reconstruirse desde el cuerpo de una peticion; la frontera de
	// identidad las resuelve y revalida antes de llamar al PDP.
	ContextoActor             ContextoActor               `json:"-"`
	VinculoAutenticacionActor VinculoAutenticacionActorV1 `json:"-"`
	// ReferenciaMotivo fija para V2 la entrada exacta de un catalogo publicado.
	// Es una capacidad interna resuelta por una frontera confiable, nunca un
	// campo reconstruido directamente desde el cuerpo de una peticion.
	ReferenciaMotivo ReferenciaEntradaCatalogo `json:"-"`
	Accion           string                    `json:"accion"`
	Recurso          RecursoAutorizable        `json:"recurso"`
	Finalidad        string                    `json:"finalidad"`
	CorrelacionRef   string                    `json:"correlacion_ref"`
	Motivo           string                    `json:"motivo"`
}

func (s SolicitudAutorizacion) Validar() error {
	if err := s.Principal.Validate(); err != nil {
		return fmt.Errorf("%w: principal", ErrSolicitudAutorizacionInvalida)
	}
	if !textoAutorizacionSinComodinSeguro(s.PerfilActivoRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(s.Accion, 256, false) ||
		!textoAutorizacionSinComodinSeguro(s.Finalidad, 512, false) ||
		!textoAutorizacionSinComodinSeguro(s.CorrelacionRef, 512, false) ||
		!textoAutorizacionSeguro(s.Motivo, 1024, true) {
		return ErrSolicitudAutorizacionInvalida
	}
	if err := s.Recurso.Validar(); err != nil {
		return err
	}
	return nil
}

// ValidarVinculoAutenticacionActor exige la variante apta para producir una
// decision durable. Validar por si solo conserva la validacion sintactica de
// una solicitud, pero nunca basta para conceder ni registrar una decision.
func (s SolicitudAutorizacion) ValidarVinculoAutenticacionActor() error {
	if s.Validar() != nil || s.ContextoActor.Validar() != nil ||
		s.Principal.ID != s.ContextoActor.Principal.ID ||
		s.Principal.AuthMethod != s.ContextoActor.Principal.AuthMethod ||
		s.Principal.AuthAssurance != s.ContextoActor.Principal.AuthAssurance ||
		s.PerfilActivoRef != s.ContextoActor.PerfilActivoRef ||
		s.VinculoAutenticacionActor.ValidarPara(s.ContextoActor) != nil {
		return ErrSolicitudAutorizacionInvalida
	}
	return nil
}

// TieneReferenciaMotivoAutorizacionV2 distingue de forma exacta una solicitud
// nueva de una historica. Un valor parcialmente rellenado cuenta como presente
// y sera rechazado por el constructor nominal V2.
func (s SolicitudAutorizacion) TieneReferenciaMotivoAutorizacionV2() bool {
	return s.ReferenciaMotivo != (ReferenciaEntradaCatalogo{})
}

// ReferenciaMotivoAutorizacionV2Valida aplica el perfil especializado de
// motivos de autorizacion V2. Ademas de la referencia de catalogo valida,
// exige una version portable, una huella no nula y una clave opaca de 128 bits.
// La existencia y vigencia de la entrada requieren resolver el catalogo.
func ReferenciaMotivoAutorizacionV2Valida(referencia ReferenciaEntradaCatalogo) bool {
	return referencia.Validar() == nil &&
		int64(referencia.CatalogoVersion) <= int64(1<<31-1) &&
		referencia.CatalogoHuellaSHA256 != strings.Repeat("0", sha256.Size*2) &&
		claveOpacaMotivoAutorizacionV2Valida(referencia.EntradaClave)
}

// ReferenciaCorrelacionAutorizacionV2Valida exige el identificador opaco de
// 128 bits reservado a solicitudes V2. La frontera debe generarlo con
// crypto/rand; nunca se deriva de datos del usuario o del expediente.
func ReferenciaCorrelacionAutorizacionV2Valida(referencia string) bool {
	return referenciaOpacaHex128AutorizacionV2Valida(referencia, "correlacion_")
}

// claveOpacaMotivoAutorizacionV2Valida aplica minimizacion por construccion:
// solo admite un identificador opaco de 128 bits generado por el servidor. La
// etiqueta humana permanece en el catalogo y nunca entra en la decision.
func claveOpacaMotivoAutorizacionV2Valida(clave string) bool {
	return referenciaOpacaHex128AutorizacionV2Valida(clave, "motivo_")
}

func referenciaOpacaHex128AutorizacionV2Valida(valor, prefijo string) bool {
	if len(valor) != len(prefijo)+32 || !strings.HasPrefix(valor, prefijo) {
		return false
	}
	for indice := len(prefijo); indice < len(valor); indice++ {
		caracter := valor[indice]
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

type EstadoVersionRol string

const (
	EstadoVersionRolPublicada EstadoVersionRol = "publicada"
	EstadoVersionRolRetirada  EstadoVersionRol = "retirada"
)

// ConcesionRol es la unica parte que concede acceso (RBAC). Las politicas
// ABAC pueden reducir esta concesion, pero nunca crearla ni ampliarla.
type ConcesionRol struct {
	Accion           string        `json:"accion"`
	ModuloID         string        `json:"modulo_id"`
	TipoRecurso      string        `json:"tipo_recurso"`
	Finalidades      []string      `json:"finalidades"`
	GarantiaMinima   AuthAssurance `json:"garantia_minima"`
	CamposPermitidos []string      `json:"campos_permitidos,omitempty"`
	Obligaciones     []string      `json:"obligaciones,omitempty"`
}

func (c ConcesionRol) Validar() error {
	// Accion, recurso, finalidad y campos se enumeran. Un comodin RBAC
	// ampliaria silenciosamente el rol al incorporar capacidades o datos nuevos.
	if !textoAutorizacionSinComodinSeguro(c.Accion, 256, false) ||
		!textoAutorizacionSinComodinSeguro(c.ModuloID, 128, false) ||
		!textoAutorizacionSinComodinSeguro(c.TipoRecurso, 128, false) ||
		!c.GarantiaMinima.Valida() ||
		!listaAutorizacionValida(c.Finalidades, false, true) ||
		!listaAutorizacionValida(c.CamposPermitidos, false, false) ||
		!listaAutorizacionValida(c.Obligaciones, false, false) {
		return ErrConfiguracionAccesoInvalida
	}
	return nil
}

func (c ConcesionRol) AdmiteFinalidad(finalidad string) bool {
	return contieneAutorizacionExacta(c.Finalidades, finalidad)
}

// VersionRol es una instantanea inmutable. Una asignacion siempre referencia
// una version concreta, nunca "la ultima".
type VersionRol struct {
	RolID                string           `json:"rol_id"`
	Version              int              `json:"version"`
	Nombre               string           `json:"nombre"`
	Estado               EstadoVersionRol `json:"estado"`
	Concesiones          []ConcesionRol   `json:"concesiones"`
	PublicadaPor         string           `json:"publicada_por"`
	PublicadaEn          time.Time        `json:"publicada_en"`
	RetiradaPor          string           `json:"retirada_por,omitempty"`
	RetiradaEn           time.Time        `json:"retirada_en,omitempty"`
	RetiradaRef          string           `json:"retirada_ref,omitempty"`
	MotivoRetiradaCodigo string           `json:"motivo_retirada_codigo,omitempty"`
}

func (v VersionRol) Referencia() string {
	// No se corrigen identificadores para fabricar una referencia valida a
	// partir de configuracion no canonica. Validar decide si la version puede
	// utilizarse; Referencia conserva exactamente el identificador recibido.
	return "rol:" + v.RolID + ":v" + strconv.Itoa(v.Version)
}

func (v VersionRol) Validar() error {
	if !textoAutorizacionSinComodinSeguro(v.RolID, 128, false) || v.Version < 1 ||
		!textoAutorizacionSeguro(v.Nombre, 512, true) ||
		(v.Estado != EstadoVersionRolPublicada && v.Estado != EstadoVersionRolRetirada) ||
		len(v.Concesiones) == 0 || len(v.Concesiones) > maximoElementosAutorizacion ||
		!textoAutorizacionSinComodinSeguro(v.PublicadaPor, 512, false) || !instanteAutorizacionCanonico(v.PublicadaEn) {
		return ErrConfiguracionAccesoInvalida
	}
	claves := make(map[string]struct{}, len(v.Concesiones))
	for _, concesion := range v.Concesiones {
		if err := concesion.Validar(); err != nil {
			return err
		}
		clave := concesion.Accion + "\x00" + concesion.ModuloID + "\x00" + concesion.TipoRecurso
		if _, repetida := claves[clave]; repetida {
			return ErrConfiguracionAccesoInvalida
		}
		claves[clave] = struct{}{}
	}
	if v.Estado == EstadoVersionRolPublicada {
		if !v.RetiradaEn.IsZero() || v.RetiradaPor != "" || v.RetiradaRef != "" || v.MotivoRetiradaCodigo != "" {
			return ErrConfiguracionAccesoInvalida
		}
	} else if !instanteAutorizacionCanonico(v.RetiradaEn) || !textoAutorizacionSinComodinSeguro(v.RetiradaPor, 512, false) || v.RetiradaEn.Before(v.PublicadaEn) {
		return ErrConfiguracionAccesoInvalida
	} else if !textoAutorizacionSinComodinSeguro(v.RetiradaRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(v.MotivoRetiradaCodigo, 128, false) {
		return ErrConfiguracionAccesoInvalida
	}
	return nil
}

func (v VersionRol) HuellaSHA256() (string, error) {
	if err := v.Validar(); err != nil {
		return "", err
	}
	return huellaAutorizacion(v)
}

type EstadoControlVigenciaVersionRol string

const (
	EstadoControlVigenciaVersionRolHabilitada EstadoControlVigenciaVersionRol = "habilitada"
	EstadoControlVigenciaVersionRolRetirada   EstadoControlVigenciaVersionRol = "retirada"
)

// ControlVigenciaVersionRol separa la instantanea inmutable del rol de su
// retirada global. Publicar una v2 retirada nunca se interpreta como retirada
// de v1: cada control nombra exactamente la version afectada y tiene su propia
// secuencia CAS.
type ControlVigenciaVersionRol struct {
	VersionRolRef  string                          `json:"version_rol_ref"`
	Revision       uint64                          `json:"revision"`
	Estado         EstadoControlVigenciaVersionRol `json:"estado"`
	ActualizadoPor string                          `json:"actualizado_por"`
	ActualizadoEn  time.Time                       `json:"actualizado_en"`
	ActoRef        string                          `json:"acto_ref,omitempty"`
	MotivoCodigo   string                          `json:"motivo_codigo,omitempty"`
}

func (c ControlVigenciaVersionRol) Validar() error {
	if !textoAutorizacionSinComodinSeguro(c.VersionRolRef, 512, false) || c.Revision == 0 ||
		(c.Estado != EstadoControlVigenciaVersionRolHabilitada && c.Estado != EstadoControlVigenciaVersionRolRetirada) ||
		!textoAutorizacionSinComodinSeguro(c.ActualizadoPor, 512, false) || !instanteAutorizacionCanonico(c.ActualizadoEn) {
		return ErrConfiguracionAccesoInvalida
	}
	if c.Estado == EstadoControlVigenciaVersionRolHabilitada {
		if c.ActoRef != "" || c.MotivoCodigo != "" {
			return ErrConfiguracionAccesoInvalida
		}
	} else if !textoAutorizacionSinComodinSeguro(c.ActoRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(c.MotivoCodigo, 128, false) {
		return ErrConfiguracionAccesoInvalida
	}
	return nil
}

func (c ControlVigenciaVersionRol) HuellaSHA256() (string, error) {
	if err := c.Validar(); err != nil {
		return "", err
	}
	return huellaAutorizacion(c)
}

type EstadoAsignacionPerfil string

const (
	EstadoAsignacionPerfilActiva   EstadoAsignacionPerfil = "activa"
	EstadoAsignacionPerfilRevocada EstadoAsignacionPerfil = "revocada"
)

// AmbitoPerfil restringe una dimension del recurso. Varias dimensiones se
// combinan con AND; los valores de una misma dimension se combinan con OR.
// No existen comodines positivos: cada valor autorizable se enumera de forma
// exacta para que una ampliacion futura del catalogo no amplie esta asignacion.
type AmbitoPerfil struct {
	Clave   string   `json:"clave"`
	Valores []string `json:"valores"`
}

func (a AmbitoPerfil) Validar() error {
	// "global" era una excepcion heredada equivalente a global=["*"]. Se
	// rechaza tambien como clave para impedir que configuracion antigua parezca
	// valida con una semantica nueva y mas restrictiva.
	if !textoAutorizacionSinComodinSeguro(a.Clave, 128, false) || a.Clave == "global" ||
		!listaAutorizacionValida(a.Valores, false, true) {
		return ErrConfiguracionAccesoInvalida
	}
	return nil
}

// AsignacionPerfil enlaza un perfil activo opaco con una unica version de
// rol. Seleccionar un perfil impide sumar permisos de otros perfiles del
// mismo principal.
type AsignacionPerfil struct {
	AsignacionID    string                 `json:"asignacion_id"`
	Version         int                    `json:"version"`
	PerfilActivoRef string                 `json:"perfil_activo_ref"`
	PrincipalID     string                 `json:"principal_id"`
	VersionRolRef   string                 `json:"version_rol_ref"`
	Estado          EstadoAsignacionPerfil `json:"estado"`
	Ambitos         []AmbitoPerfil         `json:"ambitos"`
	VigenteDesde    time.Time              `json:"vigente_desde"`
	VigenteHasta    time.Time              `json:"vigente_hasta"`
	EmitidaPor      string                 `json:"emitida_por"`
	EmitidaEn       time.Time              `json:"emitida_en"`
	RevocadaPor     string                 `json:"revocada_por,omitempty"`
	RevocadaEn      time.Time              `json:"revocada_en,omitempty"`
	RevocacionRef   string                 `json:"revocacion_ref,omitempty"`
}

func (a AsignacionPerfil) Referencia() string {
	return "asignacion:" + a.AsignacionID + ":v" + strconv.Itoa(a.Version)
}

func (a AsignacionPerfil) Validar() error {
	if !textoAutorizacionSinComodinSeguro(a.AsignacionID, 512, false) || a.Version < 1 ||
		!textoAutorizacionSinComodinSeguro(a.PerfilActivoRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(a.PrincipalID, 512, false) ||
		!textoAutorizacionSinComodinSeguro(a.VersionRolRef, 512, false) ||
		(a.Estado != EstadoAsignacionPerfilActiva && a.Estado != EstadoAsignacionPerfilRevocada) ||
		len(a.Ambitos) == 0 || len(a.Ambitos) > maximoElementosAutorizacion ||
		!instanteAutorizacionCanonico(a.VigenteDesde) || !instanteAutorizacionCanonico(a.VigenteHasta) ||
		!a.VigenteHasta.After(a.VigenteDesde) ||
		!textoAutorizacionSinComodinSeguro(a.EmitidaPor, 512, false) || !instanteAutorizacionCanonico(a.EmitidaEn) {
		return ErrConfiguracionAccesoInvalida
	}
	if a.VigenteDesde.Before(a.EmitidaEn) {
		return ErrConfiguracionAccesoInvalida
	}
	claves := make(map[string]struct{}, len(a.Ambitos))
	for _, ambito := range a.Ambitos {
		if err := ambito.Validar(); err != nil {
			return err
		}
		if _, repetida := claves[ambito.Clave]; repetida {
			return ErrConfiguracionAccesoInvalida
		}
		claves[ambito.Clave] = struct{}{}
	}
	if a.Estado == EstadoAsignacionPerfilActiva {
		if !a.RevocadaEn.IsZero() || a.RevocadaPor != "" || a.RevocacionRef != "" {
			return ErrConfiguracionAccesoInvalida
		}
	} else if !instanteAutorizacionCanonico(a.RevocadaEn) || !textoAutorizacionSinComodinSeguro(a.RevocadaPor, 512, false) ||
		!textoAutorizacionSinComodinSeguro(a.RevocacionRef, 512, false) || a.RevocadaEn.Before(a.EmitidaEn) {
		return ErrConfiguracionAccesoInvalida
	}
	return nil
}

func (a AsignacionPerfil) VigenteEn(instante time.Time) bool {
	instante = instante.UTC()
	return a.Estado == EstadoAsignacionPerfilActiva && !instante.Before(a.VigenteDesde) && instante.Before(a.VigenteHasta)
}

func (a AsignacionPerfil) Cubre(recurso RecursoAutorizable) bool {
	if a.Validar() != nil || recurso.Validar() != nil {
		return false
	}
	// Catalogo positivo y cerrado: una dimension nueva del recurso no queda
	// implicitamente fuera de control y la ausencia de una esperada no equivale
	// a acceso ilimitado.
	if len(recurso.Ambitos) != len(a.Ambitos) {
		return false
	}
	for _, ambito := range a.Ambitos {
		valor, existe := recurso.Ambitos[ambito.Clave]
		if !existe || !contieneAutorizacionExacta(ambito.Valores, valor) {
			return false
		}
	}
	return true
}

func (a AsignacionPerfil) HuellaSHA256() (string, error) {
	if err := a.Validar(); err != nil {
		return "", err
	}
	return huellaAutorizacion(a)
}
