// Package operacionescopias models declared backup progress without persistence,
// authority, verification of authenticity or platform effects.
package operacionescopias

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
)

type Estado string

const (
	Solicitada          Estado = "solicitada"
	Capturando          Estado = "capturando"
	Capturada           Estado = "capturada"
	Verificando         Estado = "verificando"
	VerificadaDeclarada Estado = "verificada_declarada"
	NoValidaDeclarada   Estado = "no_valida_declarada"
	AbandonadaDeclarada Estado = "abandonada_declarada"
)

var (
	ErrEntrada    = errors.New("operacion_entrada_invalida")
	ErrConflicto  = errors.New("operacion_conflicto_idempotencia")
	ErrVersion    = errors.New("operacion_conflicto_version")
	ErrVinculo    = errors.New("operacion_vinculo_distinto")
	ErrTransicion = errors.New("operacion_transicion_invalida")
	ErrHistoria   = errors.New("operacion_historia_invalida")
	ErrAbandono   = errors.New("operacion_abandono_no_confirmado")
	refPattern    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9:_-]{0,95}$`)
)

// Solicitud contains opaque references and hashes only, never backup content,
// paths, credentials or permissions. Its seal is supplied by the caller.
type Solicitud struct {
	Operacion string `json:"operacion"`
	Clave     string `json:"clave"`
	SHA256    string `json:"solicitud_sha256"`
	Conjunto  string `json:"conjunto"`
	Destino   string `json:"destino"`
	Politica  string `json:"politica"`
}

type Evidencia struct {
	Modo             string `json:"modo"`
	Conjunto         string `json:"conjunto"`
	ManifiestoSHA256 string `json:"manifiesto_sha256"`
	Ejecucion        string `json:"ejecucion"`
	Referencia       string `json:"referencia"`
	SHA256           string `json:"sha256"`
	Resultado        string `json:"resultado"`
}

type Comando struct {
	Clave            string               `json:"clave"`
	VersionEsperada  uint64               `json:"version_esperada"`
	SolicitudSHA256  string               `json:"solicitud_sha256"`
	Accion           string               `json:"accion"`
	ManifiestoSHA256 string               `json:"manifiesto_sha256,omitempty"`
	Ejecucion        string               `json:"ejecucion,omitempty"`
	Evidencia        *Evidencia           `json:"evidencia,omitempty"`
	Abandono         *ObservacionAbandono `json:"abandono,omitempty"`
}

// ObservacionAbandono binds a known stopped effect and cancelled lease supplied
// by the trusted executor. The model records it without issuing authority.
type ObservacionAbandono struct {
	Operacion         string `json:"operacion"`
	Destino           string `json:"destino"`
	FalloReferencia   string `json:"fallo_referencia"`
	FalloSHA256       string `json:"fallo_sha256"`
	Lease             string `json:"lease"`
	EstadoEfecto      string `json:"estado_efecto"`
	EstadoLease       string `json:"estado_lease"`
	EstadoPlataforma  string `json:"estado_plataforma"`
	EstadoVerificador string `json:"estado_verificador,omitempty"`
	EstadoVentana     string `json:"estado_ventana,omitempty"`
}

// Evento is a model receipt; it is not an audit record or durable receipt.
type Evento struct {
	Secuencia     uint64  `json:"secuencia"`
	VersionPrevia uint64  `json:"version_previa"`
	Version       uint64  `json:"version"`
	Operacion     string  `json:"operacion"`
	Conjunto      string  `json:"conjunto"`
	Destino       string  `json:"destino"`
	Politica      string  `json:"politica"`
	Comando       Comando `json:"comando"`
	SelloSHA256   string  `json:"sello_sha256"`
	Estado        Estado  `json:"estado"`
}

// Operacion is immutable through its public interface. Every successful new
// command returns a new value and appends exactly one event.
type Operacion struct {
	solicitud      Solicitud
	estado         Estado
	historia       []Evento
	manifiesto     string
	ejecucion      string
	fisica, logica bool
}

func Nueva(s Solicitud) (Operacion, error) {
	if !referencia(s.Operacion) || !referencia(s.Clave) || !huella(s.SHA256) || !referencia(s.Conjunto) || !referencia(s.Destino) || !referencia(s.Politica) {
		return Operacion{}, ErrEntrada
	}
	return Operacion{solicitud: s, estado: Solicitada}, nil
}

// Reservar only compares an existing reservation in memory. It cannot reserve
// a key globally; an eventual durable adapter must supply atomic uniqueness.
func Reservar(existente *Operacion, s Solicitud) (Operacion, bool, error) {
	nueva, err := Nueva(s)
	if err != nil {
		return Operacion{}, false, err
	}
	if existente == nil {
		return nueva, false, nil
	}
	if existente.solicitud != s {
		return Operacion{}, false, ErrConflicto
	}
	return *existente, true, nil
}

func (o Operacion) Estado() Estado  { return o.estado }
func (o Operacion) Version() uint64 { return uint64(len(o.historia)) }
func (o Operacion) Historia() []Evento {
	h := make([]Evento, len(o.historia))
	for i, e := range o.historia {
		h[i] = clonarEvento(e)
	}
	return h
}

func (o Operacion) Aplicar(c Comando) (Operacion, Evento, bool, error) {
	if o.estado == "" || !referencia(c.Clave) || !huella(c.SolicitudSHA256) {
		return o, Evento{}, false, ErrEntrada
	}
	sello := sellar(c)
	for _, e := range o.historia {
		if e.Comando.Clave != c.Clave {
			continue
		}
		if e.SelloSHA256 != sello {
			return o, Evento{}, false, ErrConflicto
		}
		return o, clonarEvento(e), true, nil
	}
	if c.SolicitudSHA256 != o.solicitud.SHA256 {
		return o, Evento{}, false, ErrVinculo
	}
	if c.VersionEsperada != o.Version() {
		return o, Evento{}, false, ErrVersion
	}
	next := o
	if err := next.transicion(c); err != nil {
		return o, Evento{}, false, err
	}
	e := Evento{Secuencia: o.Version() + 1, VersionPrevia: o.Version(), Version: o.Version() + 1, Operacion: o.solicitud.Operacion, Conjunto: o.solicitud.Conjunto, Destino: o.solicitud.Destino, Politica: o.solicitud.Politica, Comando: c, SelloSHA256: sello, Estado: next.estado}
	next.historia = append(o.Historia(), clonarEvento(e))
	return next, clonarEvento(e), false, nil
}

func (o *Operacion) transicion(c Comando) error {
	if c.Accion != "abandonar_captura" && c.Abandono != nil {
		return ErrEntrada
	}
	// Reject unused fields, which would otherwise allow ambiguous semantic seals.
	switch c.Accion {
	case "abandonar_captura":
		if (o.estado != Solicitada && o.estado != Capturando && o.estado != Capturada && o.estado != Verificando) || c.Abandono == nil || c.ManifiestoSHA256 != "" || c.Ejecucion != "" || c.Evidencia != nil {
			return ErrTransicion
		}
		a := c.Abandono
		if a.Operacion != o.solicitud.Operacion || a.Destino != o.solicitud.Destino {
			return ErrVinculo
		}
		if !referencia(a.FalloReferencia) || !huella(a.FalloSHA256) {
			return ErrEntrada
		}
		if !referencia(a.Lease) || a.EstadoEfecto != "inactivo" || a.EstadoLease != "cancelada" || a.EstadoPlataforma != "sin_efectos_pendientes" {
			return ErrAbandono
		}
		if (a.EstadoVerificador != "" && a.EstadoVerificador != "detenido") || (a.EstadoVentana != "" && a.EstadoVentana != "inactiva") {
			return ErrAbandono
		}
		if (o.estado == Capturada || o.estado == Verificando || a.FalloReferencia == "verificacion_fallida") && (a.EstadoVerificador == "" || a.EstadoVentana == "") {
			return ErrAbandono
		}
		o.estado = AbandonadaDeclarada
	case "iniciar_captura":
		if o.estado != Solicitada || !sinDatos(c) {
			return ErrTransicion
		}
		o.estado = Capturando
	case "confirmar_captura":
		if o.estado != Capturando || !huella(c.ManifiestoSHA256) || c.Ejecucion != "" || c.Evidencia != nil {
			return ErrTransicion
		}
		o.manifiesto, o.estado = c.ManifiestoSHA256, Capturada
	case "iniciar_verificacion":
		if o.estado != Capturada || c.ManifiestoSHA256 != o.manifiesto || !referencia(c.Ejecucion) || c.Evidencia != nil {
			return ErrTransicion
		}
		o.ejecucion, o.estado = c.Ejecucion, Verificando
	case "declarar_ensayo":
		if o.estado != Verificando || c.Evidencia == nil || c.ManifiestoSHA256 != "" || c.Ejecucion != "" {
			return ErrTransicion
		}
		e := *c.Evidencia
		if e.Conjunto != o.solicitud.Conjunto || e.ManifiestoSHA256 != o.manifiesto || e.Ejecucion != o.ejecucion {
			return ErrVinculo
		}
		if !referencia(e.Referencia) || !huella(e.SHA256) || (e.Modo != "fisico" && e.Modo != "logico") || (e.Resultado != "satisfactorio" && e.Resultado != "fallido") {
			return ErrEntrada
		}
		if (e.Modo == "fisico" && o.fisica) || (e.Modo == "logico" && o.logica) {
			return ErrTransicion
		}
		if e.Resultado == "fallido" {
			o.estado = NoValidaDeclarada
			return nil
		}
		if e.Modo == "fisico" {
			o.fisica = true
		} else {
			o.logica = true
		}
		if o.fisica && o.logica {
			o.estado = VerificadaDeclarada
		}
	default:
		return ErrTransicion
	}
	return nil
}

// Reconstruir verifies event order and exact links. It never sorts or repairs an
// incomplete history and never trusts a state supplied independently of events.
func Reconstruir(s Solicitud, historia []Evento) (Operacion, error) {
	o, err := Nueva(s)
	if err != nil {
		return Operacion{}, err
	}
	for _, esperado := range historia {
		next, obtenido, replay, err := o.Aplicar(esperado.Comando)
		if err != nil || replay || !reflect.DeepEqual(obtenido, esperado) {
			return Operacion{}, ErrHistoria
		}
		o = next
	}
	return o, nil
}

// Reconciliar recommends observation/revalidation only. It does not certify
// platform state or suggest repeating uncertain effects after an interruption.
func (o Operacion) Reconciliar() string {
	switch o.estado {
	case Solicitada:
		return "revalidar_antes_de_captura"
	case Capturando:
		return "conciliar_captura_sin_repetir"
	case Capturada:
		return "revalidar_antes_de_ensayos"
	case Verificando:
		return "conciliar_ensayos_pendientes"
	case VerificadaDeclarada:
		return "autenticar_evidencias_antes_de_uso"
	case NoValidaDeclarada:
		return "revisar_ensayo_fallido"
	case AbandonadaDeclarada:
		return "revisar_abandono_confirmado"
	default:
		return "historia_invalida"
	}
}

func sinDatos(c Comando) bool {
	return c.ManifiestoSHA256 == "" && c.Ejecucion == "" && c.Evidencia == nil
}
func referencia(s string) bool { return refPattern.MatchString(s) }
func huella(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func sellar(c Comando) string {
	// This closed DTO has no unsupported JSON fields; Marshal cannot fail.
	b, _ := json.Marshal(c)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func clonarEvento(e Evento) Evento {
	if e.Comando.Evidencia != nil {
		v := *e.Comando.Evidencia
		e.Comando.Evidencia = &v
	}
	if e.Comando.Abandono != nil {
		v := *e.Comando.Abandono
		e.Comando.Abandono = &v
	}
	return e
}
