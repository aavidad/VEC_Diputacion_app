package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var ErrContactoParticipacionInvalido = errors.New("bolsa: contacto de participacion invalido")

// ErrAnotacionContactoDatoPersonal distingue, dentro de un contacto inválido,
// la nota que nombra un documento o un medio de contacto (DNI, teléfono, @…):
// la persona puede corregirla, a diferencia del resto de casos. Sigue siendo
// un ErrContactoParticipacionInvalido para quien compruebe solo ese error.
var ErrAnotacionContactoDatoPersonal = fmt.Errorf("%w: la anotacion nombra un dato personal", ErrContactoParticipacionInvalido)

// ErrContactoClaveDivergente: la clave de idempotencia ya registró un
// contacto con otros datos (SQL VBC01).
var ErrContactoClaveDivergente = fmt.Errorf("%w: clave idempotente con otros datos", ErrContactoParticipacionInvalido)

const (
	CanalContactoTelefono       = "telefono"
	CanalContactoCorreo         = "correo"
	CanalContactoSMS            = "sms"
	CanalContactoPresencial     = "presencial"
	CanalContactoOtro           = "otro"
	ResultadoContactoContactado = "contactado"
	ResultadoContactoComunica   = "comunica"
	ResultadoContactoNoContesta = "no_contesta"
	ResultadoContactoBuzon      = "buzon"
	ResultadoContactoAcepta     = "acepta"
	ResultadoContactoRechaza    = "rechaza"
	ResultadoContactoAplazado   = "aplazado"
	ResultadoContactoOtro       = "otro"
	ResultadoContactoEnviado    = "enviado"
	ResultadoContactoNoEnviado  = "no_enviado"
	// ResultadoContactoNumeroErroneo: el teléfono no corresponde a la persona.
	ResultadoContactoNumeroErroneo = "numero_erroneo"
	// ResultadoContactoNoEntregado: el correo de aviso rebotó; lo anota RRHH.
	ResultadoContactoNoEntregado = "no_entregado"
	// ResultadoContactoEntregaDeclarada exige referencia y huella de la
	// constancia externa comunicada por RRHH; no acredita entrega SMTP.
	ResultadoContactoEntregaDeclarada = "entrega_declarada"
)

var canalesContacto = map[string]struct{}{CanalContactoTelefono: {}, CanalContactoCorreo: {}, CanalContactoSMS: {}, CanalContactoPresencial: {}, CanalContactoOtro: {}}
var resultadosContacto = map[string]struct{}{ResultadoContactoContactado: {}, ResultadoContactoComunica: {}, ResultadoContactoNoContesta: {}, ResultadoContactoBuzon: {}, ResultadoContactoAcepta: {}, ResultadoContactoRechaza: {}, ResultadoContactoAplazado: {}, ResultadoContactoOtro: {}, ResultadoContactoEnviado: {}, ResultadoContactoNoEnviado: {}, ResultadoContactoNumeroErroneo: {}, ResultadoContactoNoEntregado: {}, ResultadoContactoEntregaDeclarada: {}}
var patronEvidenciaOferta = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9:_-]*$`)

type ContactoParticipacion struct {
	ContactoRef, BolsaRef, ParticipacionRef, LlamamientoRef string
	OfertaRef, EvidenciaRef, EvidenciaHuellaSHA256          string
	Canal, Actor, Resultado, Anotacion                      string
	Instante                                                time.Time
	// InstanteServidor solo está presente durante el registro nuevo. El lector
	// histórico B78 no necesita distinguir el origen del instante ya conservado.
	InstanteServidor bool
}

func (c ContactoParticipacion) Validar() error {
	_, canal := canalesContacto[c.Canal]
	_, resultado := resultadosContacto[c.Resultado]
	if !referenciaBorradorLlamamientoValida(c.ContactoRef) || !referenciaLlamamientoOpacaValida(c.BolsaRef) || !referenciaLlamamientoOpacaValida(c.ParticipacionRef) ||
		(c.LlamamientoRef != "" && !referenciaLlamamientoOpacaValida(c.LlamamientoRef)) || !canal || !resultado ||
		(c.OfertaRef != "" && (len(c.OfertaRef) != len("oferta:")+64 || !strings.HasPrefix(c.OfertaRef, "oferta:") || !huellaHexMinuscula(strings.TrimPrefix(c.OfertaRef, "oferta:")))) ||
		(c.OfertaRef != "" && c.LlamamientoRef != "") ||
		(c.Resultado == ResultadoContactoComunica && (c.Canal != CanalContactoTelefono || c.LlamamientoRef == "")) ||
		(c.OfertaRef == "" && (c.EvidenciaRef != "" || c.EvidenciaHuellaSHA256 != "" || c.Resultado == ResultadoContactoEntregaDeclarada)) ||
		(c.OfertaRef != "" && (c.Canal != CanalContactoCorreo || (c.Resultado != ResultadoContactoEnviado && c.Resultado != ResultadoContactoNoEnviado && c.Resultado != ResultadoContactoNoEntregado && c.Resultado != ResultadoContactoEntregaDeclarada))) ||
		(c.Resultado == ResultadoContactoEntregaDeclarada && (c.EvidenciaRef == "" || len(c.EvidenciaRef) > 256 || c.EvidenciaRef != strings.TrimSpace(c.EvidenciaRef) || !patronEvidenciaOferta.MatchString(c.EvidenciaRef) || !referenciaLlamamientoOpacaValida(c.EvidenciaRef) || len(c.EvidenciaHuellaSHA256) != 64 || !huellaHexMinuscula(c.EvidenciaHuellaSHA256))) ||
		(c.Resultado != ResultadoContactoEntregaDeclarada && (c.EvidenciaRef != "" || c.EvidenciaHuellaSHA256 != "")) ||
		!patronPersonaBorradorLlamamiento.MatchString(c.Actor) || !(instanteLlamamientoCanonico(c.Instante) || (c.InstanteServidor && c.Instante.IsZero() && c.Canal == CanalContactoTelefono && c.LlamamientoRef != "" && c.OfertaRef == "")) ||
		c.Anotacion != strings.TrimSpace(c.Anotacion) || (len(c.Anotacion) == 0 && (c.Canal != CanalContactoTelefono || c.LlamamientoRef == "")) || len(c.Anotacion) > 1000 || strings.ContainsAny(c.Anotacion, "\x00\u2028\u2029") {
		return ErrContactoParticipacionInvalido
	}
	if contieneDatoPersonalEvidente(c.Anotacion) {
		return ErrAnotacionContactoDatoPersonal
	}
	return nil
}

func CanalesContactoParticipacion() []string {
	return []string{CanalContactoTelefono, CanalContactoCorreo, CanalContactoSMS, CanalContactoPresencial, CanalContactoOtro}
}
func ResultadosContactoParticipacion() []string {
	return []string{ResultadoContactoContactado, ResultadoContactoComunica, ResultadoContactoNoContesta, ResultadoContactoBuzon, ResultadoContactoAcepta, ResultadoContactoRechaza, ResultadoContactoAplazado, ResultadoContactoOtro, ResultadoContactoEnviado, ResultadoContactoNoEnviado, ResultadoContactoNumeroErroneo, ResultadoContactoNoEntregado, ResultadoContactoEntregaDeclarada}
}

// ResultadoTelefonoActualValido limita el nuevo registro de llamada a los
// resultados visibles de ese modo; no convierte ninguno en resolución.
func ResultadoTelefonoActualValido(resultado string) bool {
	switch resultado {
	case ResultadoContactoContactado, ResultadoContactoComunica, ResultadoContactoNoContesta,
		ResultadoContactoBuzon, ResultadoContactoNumeroErroneo, ResultadoContactoAcepta,
		ResultadoContactoRechaza, ResultadoContactoAplazado:
		return true
	}
	return false
}

func huellaHexMinuscula(v string) bool {
	for _, r := range v {
		if r < '0' || r > '9' {
			if r < 'a' || r > 'f' {
				return false
			}
		}
	}
	return true
}
