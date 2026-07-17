package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
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

type EstadoPoliticaRestrictiva string

const (
	EstadoPoliticaRestrictivaPublicada EstadoPoliticaRestrictiva = "publicada"
	EstadoPoliticaRestrictivaRetirada  EstadoPoliticaRestrictiva = "retirada"
)

type EfectoPoliticaRestrictiva string

const (
	EfectoPoliticaRestringir EfectoPoliticaRestrictiva = "restringir"
	EfectoPoliticaDenegar    EfectoPoliticaRestrictiva = "denegar"
)

type RestriccionAtributoRecurso struct {
	Clave             string   `json:"clave"`
	ValoresPermitidos []string `json:"valores_permitidos"`
}

func (r RestriccionAtributoRecurso) Validar() error {
	if !textoAutorizacionSinComodinSeguro(r.Clave, 128, false) ||
		!listaAutorizacionValida(r.ValoresPermitidos, true, true) {
		return ErrConfiguracionAccesoInvalida
	}
	return nil
}

// PoliticaRestrictiva expresa ABAC de efecto exclusivamente restrictivo. No
// existe un efecto "permitir": sin una concesion RBAC previa siempre se
// deniega.
type PoliticaRestrictiva struct {
	PoliticaID            string                       `json:"politica_id"`
	Version               int                          `json:"version"`
	Nombre                string                       `json:"nombre"`
	Estado                EstadoPoliticaRestrictiva    `json:"estado"`
	Efecto                EfectoPoliticaRestrictiva    `json:"efecto"`
	Acciones              []string                     `json:"acciones"`
	Modulos               []string                     `json:"modulos"`
	TiposRecurso          []string                     `json:"tipos_recurso"`
	FinalidadesPermitidas []string                     `json:"finalidades_permitidas,omitempty"`
	GarantiaMinima        AuthAssurance                `json:"garantia_minima,omitempty"`
	Restricciones         []RestriccionAtributoRecurso `json:"restricciones,omitempty"`
	RestringeCampos       bool                         `json:"restringe_campos,omitempty"`
	CamposPermitidos      []string                     `json:"campos_permitidos,omitempty"`
	Obligaciones          []string                     `json:"obligaciones,omitempty"`
	VigenteDesde          time.Time                    `json:"vigente_desde"`
	VigenteHasta          time.Time                    `json:"vigente_hasta"`
	PublicadaPor          string                       `json:"publicada_por"`
	PublicadaEn           time.Time                    `json:"publicada_en"`
	RetiradaPor           string                       `json:"retirada_por,omitempty"`
	RetiradaEn            time.Time                    `json:"retirada_en,omitempty"`
}

func (p PoliticaRestrictiva) Referencia() string {
	return "politica:" + p.PoliticaID + ":v" + strconv.Itoa(p.Version)
}

func (p PoliticaRestrictiva) Validar() error {
	if !textoAutorizacionSinComodinSeguro(p.PoliticaID, 128, false) || p.Version < 1 ||
		!textoAutorizacionSeguro(p.Nombre, 512, true) ||
		(p.Estado != EstadoPoliticaRestrictivaPublicada && p.Estado != EstadoPoliticaRestrictivaRetirada) ||
		(p.Efecto != EfectoPoliticaRestringir && p.Efecto != EfectoPoliticaDenegar) ||
		!listaAutorizacionValida(p.Acciones, true, true) ||
		!listaAutorizacionValida(p.Modulos, true, true) ||
		!listaAutorizacionValida(p.TiposRecurso, true, true) ||
		!listaAutorizacionValida(p.FinalidadesPermitidas, true, false) ||
		(p.GarantiaMinima != "" && !p.GarantiaMinima.Valida()) ||
		!listaAutorizacionValida(p.Obligaciones, false, false) ||
		!instanteAutorizacionCanonico(p.VigenteDesde) || !instanteAutorizacionCanonico(p.VigenteHasta) ||
		!p.VigenteHasta.After(p.VigenteDesde) ||
		!textoAutorizacionSinComodinSeguro(p.PublicadaPor, 512, false) || !instanteAutorizacionCanonico(p.PublicadaEn) {
		return ErrConfiguracionAccesoInvalida
	}
	if p.VigenteDesde.Before(p.PublicadaEn) || len(p.Restricciones) > maximoElementosAutorizacion {
		return ErrConfiguracionAccesoInvalida
	}
	if p.RestringeCampos {
		if !listaAutorizacionValida(p.CamposPermitidos, true, false) {
			return ErrConfiguracionAccesoInvalida
		}
	} else if len(p.CamposPermitidos) != 0 {
		return ErrConfiguracionAccesoInvalida
	}
	claves := make(map[string]struct{}, len(p.Restricciones))
	for _, restriccion := range p.Restricciones {
		if err := restriccion.Validar(); err != nil {
			return err
		}
		if _, repetida := claves[restriccion.Clave]; repetida {
			return ErrConfiguracionAccesoInvalida
		}
		claves[restriccion.Clave] = struct{}{}
	}
	if p.Estado == EstadoPoliticaRestrictivaPublicada {
		if !p.RetiradaEn.IsZero() || p.RetiradaPor != "" {
			return ErrConfiguracionAccesoInvalida
		}
	} else if !instanteAutorizacionCanonico(p.RetiradaEn) || !textoAutorizacionSinComodinSeguro(p.RetiradaPor, 512, false) || p.RetiradaEn.Before(p.PublicadaEn) {
		return ErrConfiguracionAccesoInvalida
	}
	return nil
}

func (p PoliticaRestrictiva) VigenteEn(instante time.Time) bool {
	instante = instante.UTC()
	return p.Estado == EstadoPoliticaRestrictivaPublicada && !instante.Before(p.VigenteDesde) && instante.Before(p.VigenteHasta)
}

func (p PoliticaRestrictiva) AplicaA(s SolicitudAutorizacion) bool {
	return contieneAutorizacionRestrictiva(p.Acciones, s.Accion) &&
		contieneAutorizacionRestrictiva(p.Modulos, s.Recurso.ModuloID) &&
		contieneAutorizacionRestrictiva(p.TiposRecurso, s.Recurso.Tipo)
}

func (p PoliticaRestrictiva) Cumple(s SolicitudAutorizacion) bool {
	if len(p.FinalidadesPermitidas) > 0 && !contieneAutorizacionRestrictiva(p.FinalidadesPermitidas, s.Finalidad) {
		return false
	}
	for _, restriccion := range p.Restricciones {
		valor, existe := s.Recurso.Atributos[restriccion.Clave]
		if !existe || !contieneAutorizacionRestrictiva(restriccion.ValoresPermitidos, valor) {
			return false
		}
	}
	return true
}

func (p PoliticaRestrictiva) HuellaSHA256() (string, error) {
	if err := p.Validar(); err != nil {
		return "", err
	}
	return huellaAutorizacion(p)
}

// InstantaneaAutorizacion agrupa en una sola lectura coherente todos los datos
// mutables que intervienen en una decision. RevisionCatalogoPoliticas cambia
// ante cualquier publicacion o retirada y CatalogoPoliticasHuellaSHA256 fija el
// conjunto completo de versiones actuales, incluidas las retiradas y las que
// no resulten aplicables a una solicitud concreta.
type InstantaneaAutorizacion struct {
	AsignacionPerfil              AsignacionPerfil          `json:"asignacion_perfil"`
	VersionRol                    VersionRol                `json:"version_rol"`
	ControlVigenciaVersionRol     ControlVigenciaVersionRol `json:"control_vigencia_version_rol"`
	Politicas                     []PoliticaRestrictiva     `json:"politicas"`
	RevisionCatalogoPoliticas     uint64                    `json:"revision_catalogo_politicas"`
	CatalogoPoliticasHuellaSHA256 string                    `json:"catalogo_politicas_huella_sha256"`
}

func (i InstantaneaAutorizacion) Validar() error {
	if err := i.AsignacionPerfil.Validar(); err != nil {
		return ErrConfiguracionAccesoInvalida
	}
	if err := i.VersionRol.Validar(); err != nil {
		return ErrConfiguracionAccesoInvalida
	}
	if err := i.ControlVigenciaVersionRol.Validar(); err != nil {
		return ErrConfiguracionAccesoInvalida
	}
	if i.AsignacionPerfil.VersionRolRef != i.VersionRol.Referencia() ||
		i.ControlVigenciaVersionRol.VersionRolRef != i.VersionRol.Referencia() ||
		i.ControlVigenciaVersionRol.ActualizadoEn.Before(i.VersionRol.PublicadaEn) ||
		(i.VersionRol.Estado == EstadoVersionRolRetirada &&
			i.ControlVigenciaVersionRol.Estado != EstadoControlVigenciaVersionRolRetirada) ||
		i.RevisionCatalogoPoliticas == 0 ||
		!huellaSHA256AutorizacionValida(i.CatalogoPoliticasHuellaSHA256) {
		return ErrConfiguracionAccesoInvalida
	}
	huella, err := HuellaCatalogoPoliticasAutorizacion(i.Politicas)
	if err != nil || huella != i.CatalogoPoliticasHuellaSHA256 {
		return ErrConfiguracionAccesoInvalida
	}
	return nil
}

type entradaHuellaCatalogoPoliticasAutorizacion struct {
	Referencia   string `json:"referencia"`
	HuellaSHA256 string `json:"huella_sha256"`
}

// HuellaCatalogoPoliticasAutorizacion calcula una huella determinista del
// manifiesto completo referencia+huella. El orden fisico de lectura no forma
// parte del significado del catalogo, pero ningun identificador ni valor se
// recorta, cambia de caja o corrige para aceptar configuracion no canonica.
func HuellaCatalogoPoliticasAutorizacion(politicas []PoliticaRestrictiva) (string, error) {
	if len(politicas) > maximoElementosAutorizacion {
		return "", ErrConfiguracionAccesoInvalida
	}
	referencias := make([]string, 0, len(politicas))
	huellas := make(map[string]string, len(politicas))
	identificadores := make(map[string]struct{}, len(politicas))
	for _, politica := range politicas {
		if err := politica.Validar(); err != nil {
			return "", ErrConfiguracionAccesoInvalida
		}
		if _, repetido := identificadores[politica.PoliticaID]; repetido {
			return "", ErrConfiguracionAccesoInvalida
		}
		identificadores[politica.PoliticaID] = struct{}{}
		referencia := politica.Referencia()
		huella, err := politica.HuellaSHA256()
		if err != nil {
			return "", ErrConfiguracionAccesoInvalida
		}
		referencias = append(referencias, referencia)
		huellas[referencia] = huella
	}
	return huellaCatalogoPoliticasDesdeEvidencias(referencias, huellas)
}

// HuellaEvidenciasCatalogoPoliticasAutorizacion permite verificar el mismo
// manifiesto cuando el adaptador duradero ya ha materializado referencias y
// huellas inmutables sin reconstruir los documentos de politica.
func HuellaEvidenciasCatalogoPoliticasAutorizacion(
	referencias []string,
	huellas map[string]string,
) (string, error) {
	return huellaCatalogoPoliticasDesdeEvidencias(referencias, huellas)
}

// DecisionAutorizacion es una evidencia breve, no un permiso permanente. Las
// referencias y huellas fijan exactamente la configuracion evaluada.
type DecisionAutorizacion struct {
	DecisionRef                           string                      `json:"decision_ref"`
	Concedida                             bool                        `json:"concedida"`
	Codigo                                string                      `json:"codigo"`
	PrincipalID                           string                      `json:"principal_id"`
	PerfilActivoRef                       string                      `json:"perfil_activo_ref"`
	Accion                                string                      `json:"accion"`
	RecursoRef                            string                      `json:"recurso_ref"`
	ModuloID                              string                      `json:"modulo_id,omitempty"`
	TipoRecurso                           string                      `json:"tipo_recurso,omitempty"`
	ContextoRecursoHuellaSHA256           string                      `json:"contexto_recurso_huella_sha256,omitempty"`
	Finalidad                             string                      `json:"finalidad"`
	CorrelacionRef                        string                      `json:"correlacion_ref"`
	EsquemaHuellaSolicitud                string                      `json:"esquema_huella_solicitud,omitempty"`
	SolicitudHuellaSHA256                 string                      `json:"solicitud_huella_sha256,omitempty"`
	EsquemaHuellaMotivo                   string                      `json:"esquema_huella_motivo,omitempty"`
	MotivoHuellaSHA256                    string                      `json:"motivo_huella_sha256,omitempty"`
	VinculoAutenticacionActor             VinculoAutenticacionActorV1 `json:"vinculo_autenticacion_actor"`
	AsignacionRef                         string                      `json:"asignacion_ref,omitempty"`
	AsignacionHuellaSHA256                string                      `json:"asignacion_huella_sha256,omitempty"`
	VersionRolRef                         string                      `json:"version_rol_ref,omitempty"`
	VersionRolHuellaSHA256                string                      `json:"version_rol_huella_sha256,omitempty"`
	ControlVigenciaVersionRolRef          string                      `json:"control_vigencia_version_rol_ref,omitempty"`
	ControlVigenciaVersionRolRevision     uint64                      `json:"control_vigencia_version_rol_revision,omitempty"`
	ControlVigenciaVersionRolHuellaSHA256 string                      `json:"control_vigencia_version_rol_huella_sha256,omitempty"`
	RevisionCatalogoPoliticas             uint64                      `json:"revision_catalogo_politicas,omitempty"`
	CatalogoPoliticasHuellaSHA256         string                      `json:"catalogo_politicas_huella_sha256,omitempty"`
	PoliticasEvaluadasRefs                []string                    `json:"politicas_evaluadas_refs,omitempty"`
	PoliticasEvaluadasHuellasSHA256       map[string]string           `json:"politicas_evaluadas_huellas_sha256,omitempty"`
	// PoliticasRefs contiene solo el subconjunto aplicable. La evidencia del
	// catalogo completo se conserva por separado en PoliticasEvaluadasRefs.
	PoliticasRefs          []string          `json:"politicas_refs,omitempty"`
	PoliticasHuellasSHA256 map[string]string `json:"politicas_huellas_sha256,omitempty"`
	GarantiaMinima         AuthAssurance     `json:"garantia_minima,omitempty"`
	CamposPermitidos       []string          `json:"campos_permitidos,omitempty"`
	Obligaciones           []string          `json:"obligaciones,omitempty"`
	EmitidaEn              time.Time         `json:"emitida_en"`
	ValidaHasta            time.Time         `json:"valida_hasta"`
}

func (d DecisionAutorizacion) Validar() error {
	datosVinculo, errVinculo := d.VinculoAutenticacionActor.Datos()
	if errVinculo != nil {
		return ErrDecisionAutorizacionInvalida
	}
	if !textoAutorizacionSinComodinSeguro(d.DecisionRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.Codigo, 128, false) ||
		!textoAutorizacionSinComodinSeguro(d.PrincipalID, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.PerfilActivoRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.Accion, 256, false) ||
		!textoAutorizacionSinComodinSeguro(d.RecursoRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.Finalidad, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.CorrelacionRef, 512, false) ||
		d.validarVinculoSolicitudAutorizacion() != nil ||
		d.PrincipalID != datosVinculo.PrincipalID ||
		d.PerfilActivoRef != datosVinculo.PerfilActivoRef ||
		!instanteAutorizacionCanonico(d.EmitidaEn) || !instanteAutorizacionCanonico(d.ValidaHasta) ||
		!d.ValidaHasta.After(d.EmitidaEn) ||
		d.EmitidaEn.Before(datosVinculo.SesionRevalidadaEn) ||
		d.ValidaHasta.After(datosVinculo.SesionValidaHasta) ||
		d.ValidaHasta.Sub(d.EmitidaEn) > VigenciaMaximaDecisionAutorizacion ||
		!listaAutorizacionValida(d.PoliticasRefs, false, false) ||
		!listaAutorizacionValida(d.CamposPermitidos, false, false) ||
		!listaAutorizacionValida(d.Obligaciones, false, false) {
		return ErrDecisionAutorizacionInvalida
	}
	if d.Concedida {
		if d.Codigo != "concedida" || !textoAutorizacionSinComodinSeguro(d.AsignacionRef, 512, false) ||
			!huellaSHA256AutorizacionValida(d.AsignacionHuellaSHA256) ||
			!textoAutorizacionSinComodinSeguro(d.VersionRolRef, 512, false) ||
			!huellaSHA256AutorizacionValida(d.VersionRolHuellaSHA256) ||
			!d.GarantiaMinima.Valida() ||
			!CumpleGarantiaAutenticacion(datosVinculo.GarantiaObservada, d.GarantiaMinima) {
			return ErrDecisionAutorizacionInvalida
		}
	} else if d.Codigo == "concedida" {
		return ErrDecisionAutorizacionInvalida
	}
	if len(d.PoliticasHuellasSHA256) != len(d.PoliticasRefs) {
		return ErrDecisionAutorizacionInvalida
	}
	for _, referencia := range d.PoliticasRefs {
		if !huellaSHA256AutorizacionValida(d.PoliticasHuellasSHA256[referencia]) {
			return ErrDecisionAutorizacionInvalida
		}
	}
	if d.tieneEvidenciaInstantanea() {
		if err := d.validarEvidenciaInstantanea(); err != nil {
			return err
		}
	}
	return nil
}

// ValidarEvidenciaInstantanea exige el formato reforzado que deben registrar
// los adaptadores de autorizacion. Validar conserva temporalmente la lectura de
// evidencias historicas anteriores, pero el registro CAS nunca las admite.
func (d DecisionAutorizacion) ValidarEvidenciaInstantanea() error {
	if err := d.Validar(); err != nil {
		return err
	}
	if !d.tieneEvidenciaInstantanea() {
		return ErrDecisionAutorizacionInvalida
	}
	return d.validarEvidenciaInstantanea()
}

// ValidarEvidenciaInstantaneaSolicitudLigadaV2 es el contrato para decisiones
// nuevas y efectos durables. ValidarEvidenciaInstantanea conserva lectura
// historica, pero nunca basta para crear una capacidad ejecutable V2.
func (d DecisionAutorizacion) ValidarEvidenciaInstantaneaSolicitudLigadaV2() error {
	if err := d.ValidarEvidenciaInstantanea(); err != nil || !d.TieneSolicitudLigadaV2() ||
		!ReferenciaCorrelacionAutorizacionV2Valida(d.CorrelacionRef) {
		return ErrDecisionAutorizacionInvalida
	}
	return nil
}

// TieneSolicitudLigadaV2 informa solo de la validez estructural de los dos
// compromisos. La procedencia sigue dependiendo del PDP y del registro.
func (d DecisionAutorizacion) TieneSolicitudLigadaV2() bool {
	return d.EsquemaHuellaSolicitud == EsquemaHuellaSolicitudAutorizacionV2 &&
		huellaSHA256AutorizacionValida(d.SolicitudHuellaSHA256) &&
		d.SolicitudHuellaSHA256 != strings.Repeat("0", sha256.Size*2) &&
		d.EsquemaHuellaMotivo == EsquemaHuellaMotivoAutorizacionV2 &&
		huellaSHA256AutorizacionValida(d.MotivoHuellaSHA256) &&
		d.MotivoHuellaSHA256 != strings.Repeat("0", sha256.Size*2)
}

func (d DecisionAutorizacion) validarVinculoSolicitudAutorizacion() error {
	sinVinculo := d.EsquemaHuellaSolicitud == "" && d.SolicitudHuellaSHA256 == "" &&
		d.EsquemaHuellaMotivo == "" && d.MotivoHuellaSHA256 == ""
	if sinVinculo || d.TieneSolicitudLigadaV2() {
		return nil
	}
	return ErrDecisionAutorizacionInvalida
}

func (d DecisionAutorizacion) tieneEvidenciaInstantanea() bool {
	return d.RevisionCatalogoPoliticas != 0 || d.CatalogoPoliticasHuellaSHA256 != "" ||
		d.ControlVigenciaVersionRolRef != "" || d.ControlVigenciaVersionRolRevision != 0 ||
		d.ControlVigenciaVersionRolHuellaSHA256 != "" ||
		len(d.PoliticasEvaluadasRefs) != 0 || len(d.PoliticasEvaluadasHuellasSHA256) != 0
}

func (d DecisionAutorizacion) validarEvidenciaInstantanea() error {
	if d.RevisionCatalogoPoliticas == 0 ||
		!huellaSHA256AutorizacionValida(d.CatalogoPoliticasHuellaSHA256) ||
		!textoAutorizacionSinComodinSeguro(d.AsignacionRef, 512, false) ||
		!huellaSHA256AutorizacionValida(d.AsignacionHuellaSHA256) ||
		!textoAutorizacionSinComodinSeguro(d.VersionRolRef, 512, false) ||
		!huellaSHA256AutorizacionValida(d.VersionRolHuellaSHA256) ||
		d.ControlVigenciaVersionRolRef != d.VersionRolRef ||
		d.ControlVigenciaVersionRolRevision == 0 ||
		!huellaSHA256AutorizacionValida(d.ControlVigenciaVersionRolHuellaSHA256) {
		return ErrDecisionAutorizacionInvalida
	}
	if !textoAutorizacionSinComodinSeguro(d.ModuloID, 128, false) ||
		!textoAutorizacionSinComodinSeguro(d.TipoRecurso, 128, false) ||
		!huellaSHA256AutorizacionValida(d.ContextoRecursoHuellaSHA256) {
		return ErrDecisionAutorizacionInvalida
	}
	huella, err := huellaCatalogoPoliticasDesdeEvidencias(
		d.PoliticasEvaluadasRefs,
		d.PoliticasEvaluadasHuellasSHA256,
	)
	if err != nil || huella != d.CatalogoPoliticasHuellaSHA256 {
		return ErrDecisionAutorizacionInvalida
	}
	for _, referencia := range d.PoliticasRefs {
		huellaEvaluada, evaluada := d.PoliticasEvaluadasHuellasSHA256[referencia]
		if !evaluada || huellaEvaluada != d.PoliticasHuellasSHA256[referencia] {
			return ErrDecisionAutorizacionInvalida
		}
	}
	return nil
}

func huellaCatalogoPoliticasDesdeEvidencias(referencias []string, huellas map[string]string) (string, error) {
	if !listaAutorizacionValida(referencias, false, false) || len(huellas) != len(referencias) {
		return "", ErrConfiguracionAccesoInvalida
	}
	referenciasOrdenadas := append([]string(nil), referencias...)
	sort.Strings(referenciasOrdenadas)
	entradas := make([]entradaHuellaCatalogoPoliticasAutorizacion, 0, len(referenciasOrdenadas))
	for _, referencia := range referenciasOrdenadas {
		huella, existe := huellas[referencia]
		if !existe || !huellaSHA256AutorizacionValida(huella) {
			return "", ErrConfiguracionAccesoInvalida
		}
		entradas = append(entradas, entradaHuellaCatalogoPoliticasAutorizacion{
			Referencia: referencia, HuellaSHA256: huella,
		})
	}
	return huellaAutorizacion(entradas)
}

func (d DecisionAutorizacion) VigenteEn(instante time.Time) bool {
	if d.ValidarEvidenciaInstantanea() != nil {
		return false
	}
	datosVinculo, err := d.VinculoAutenticacionActor.Datos()
	if err != nil {
		return false
	}
	return d.Concedida && !instante.UTC().Before(d.EmitidaEn) && instante.UTC().Before(d.ValidaHasta) &&
		!instante.UTC().Before(datosVinculo.SesionRevalidadaEn) &&
		instante.UTC().Before(datosVinculo.SesionValidaHasta)
}

// VigenteParaEfectoEn excluye expresamente decisiones historicas sin el
// compromiso V2 de solicitud y motivo.
func (d DecisionAutorizacion) VigenteParaEfectoEn(instante time.Time) bool {
	return d.ValidarEvidenciaInstantaneaSolicitudLigadaV2() == nil && d.VigenteEn(instante)
}

func GarantiaAutenticacionMasAlta(primera, segunda AuthAssurance) (AuthAssurance, error) {
	nivelPrimero, validaPrimera := nivelGarantia(primera)
	nivelSegundo, validaSegunda := nivelGarantia(segunda)
	if !validaPrimera || !validaSegunda {
		return "", ErrConfiguracionAccesoInvalida
	}
	if nivelSegundo > nivelPrimero {
		return segunda, nil
	}
	return primera, nil
}

func nivelGarantia(garantia AuthAssurance) (int, bool) {
	switch garantia {
	case AuthAssuranceLow:
		return 1, true
	case AuthAssuranceSubstantial:
		return 2, true
	case AuthAssuranceHigh:
		return 3, true
	default:
		return 0, false
	}
}

func huellaAutorizacion(valor any) (string, error) {
	contenido, err := json.Marshal(valor)
	if err != nil {
		return "", fmt.Errorf("%w: serializar", ErrConfiguracionAccesoInvalida)
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func mapaAutorizacionValido(valores map[string]string) bool {
	if len(valores) > maximoElementosAutorizacion {
		return false
	}
	for clave, valor := range valores {
		if !textoAutorizacionTecnicoASCII(clave, 128) || !textoAutorizacionSeguro(valor, 512, true) {
			return false
		}
	}
	return true
}

func clonarMapaAutorizacion(valores map[string]string) map[string]string {
	copia := make(map[string]string, len(valores))
	for clave, valor := range valores {
		copia[clave] = valor
	}
	return copia
}

func textoAutorizacionSeguro(valor string, maximo int, permiteEspacios bool) bool {
	if maximo < 1 || valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximo || !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || (!permiteEspacios && unicode.IsSpace(caracter)) {
			return false
		}
	}
	return true
}

// textoAutorizacionSinComodinSeguro se usa en toda dimension positiva. No
// basta con rechazar el valor exactamente "*": admitir "bolsa.*" o
// "expediente:*" conservaria una sintaxis ambigua que otro conector podria
// interpretar como ampliacion. Las concesiones solo contienen valores
// literales, completos y exactos.
func textoAutorizacionSinComodinSeguro(valor string, maximo int, permiteEspacios bool) bool {
	return textoAutorizacionSeguro(valor, maximo, permiteEspacios) &&
		textoAutorizacionASCIIVisible(valor, permiteEspacios) &&
		!strings.ContainsRune(valor, '*')
}

func listaAutorizacionValida(valores []string, permiteComodin, obligatoria bool) bool {
	if obligatoria && len(valores) == 0 {
		return false
	}
	if len(valores) > maximoElementosAutorizacion {
		return false
	}
	vistos := make(map[string]struct{}, len(valores))
	for _, valor := range valores {
		if !textoAutorizacionTecnicoASCII(valor, 512) ||
			(strings.ContainsRune(valor, '*') && (!permiteComodin || valor != comodinAutorizacion)) {
			return false
		}
		if _, repetido := vistos[valor]; repetido {
			return false
		}
		vistos[valor] = struct{}{}
	}
	return true
}

func textoAutorizacionTecnicoASCII(valor string, maximo int) bool {
	return textoAutorizacionSeguro(valor, maximo, false) && textoAutorizacionASCIIVisible(valor, false)
}

func textoAutorizacionASCIIVisible(valor string, permiteEspacios bool) bool {
	for _, caracter := range []byte(valor) {
		if caracter < 0x21 || caracter > 0x7e {
			if !permiteEspacios || caracter != ' ' {
				return false
			}
		}
	}
	return true
}

func huellaSHA256AutorizacionValida(valor string) bool {
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) {
		return false
	}
	contenido, err := hex.DecodeString(valor)
	return err == nil && len(contenido) == sha256.Size
}

// PostgreSQL timestamptz conserva microsegundos. Exigir UTC y precision
// microsegundo y el intervalo comun RFC 3339/JSON (anos 0001..9999) antes de
// calcular huellas impide que una ida y vuelta por el adaptador duradero cambie
// silenciosamente una instantanea o un CAS. UnixMicro tambien queda asi dentro
// de su intervalo definido.
func instanteAutorizacionCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func contieneAutorizacionExacta(valores []string, buscado string) bool {
	for _, valor := range valores {
		if valor == buscado {
			return true
		}
	}
	return false
}

// contieneAutorizacionRestrictiva solo se usa en politicas que no conceden:
// el comodin puede hacer que una denegacion o restriccion abarque mas casos,
// pero nunca crea una capacidad positiva.
func contieneAutorizacionRestrictiva(valores []string, buscado string) bool {
	return contieneAutorizacionExacta(valores, comodinAutorizacion) ||
		contieneAutorizacionExacta(valores, buscado)
}
