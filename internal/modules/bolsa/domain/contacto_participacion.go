package domain

import (
	"errors"
	"strings"
	"time"
)

var ErrContactoParticipacionInvalido = errors.New("bolsa: contacto de participacion invalido")

const (
	CanalContactoTelefono       = "telefono"
	CanalContactoCorreo         = "correo"
	CanalContactoSMS            = "sms"
	CanalContactoPresencial     = "presencial"
	CanalContactoOtro           = "otro"
	ResultadoContactoContactado = "contactado"
	ResultadoContactoNoContesta = "no_contesta"
	ResultadoContactoBuzon      = "buzon"
	ResultadoContactoAcepta     = "acepta"
	ResultadoContactoRechaza    = "rechaza"
	ResultadoContactoAplazado   = "aplazado"
	ResultadoContactoOtro       = "otro"
	ResultadoContactoEnviado    = "enviado"
	ResultadoContactoNoEnviado  = "no_enviado"
)

var canalesContacto = map[string]struct{}{CanalContactoTelefono: {}, CanalContactoCorreo: {}, CanalContactoSMS: {}, CanalContactoPresencial: {}, CanalContactoOtro: {}}
var resultadosContacto = map[string]struct{}{ResultadoContactoContactado: {}, ResultadoContactoNoContesta: {}, ResultadoContactoBuzon: {}, ResultadoContactoAcepta: {}, ResultadoContactoRechaza: {}, ResultadoContactoAplazado: {}, ResultadoContactoOtro: {}, ResultadoContactoEnviado: {}, ResultadoContactoNoEnviado: {}}

type ContactoParticipacion struct {
	ContactoRef, BolsaRef, ParticipacionRef, LlamamientoRef string
	Canal, Actor, Resultado, Anotacion                      string
	Instante                                                time.Time
}

func (c ContactoParticipacion) Validar() error {
	_, canal := canalesContacto[c.Canal]
	_, resultado := resultadosContacto[c.Resultado]
	if !referenciaBorradorLlamamientoValida(c.ContactoRef) || !referenciaLlamamientoOpacaValida(c.BolsaRef) || !referenciaLlamamientoOpacaValida(c.ParticipacionRef) ||
		(c.LlamamientoRef != "" && !referenciaLlamamientoOpacaValida(c.LlamamientoRef)) || !canal || !resultado ||
		!patronPersonaBorradorLlamamiento.MatchString(c.Actor) || !instanteLlamamientoCanonico(c.Instante) ||
		c.Anotacion != strings.TrimSpace(c.Anotacion) || len(c.Anotacion) == 0 || len(c.Anotacion) > 1000 || strings.ContainsAny(c.Anotacion, "\x00\u2028\u2029") || contieneDatoPersonalEvidente(c.Anotacion) {
		return ErrContactoParticipacionInvalido
	}
	return nil
}

func CanalesContactoParticipacion() []string {
	return []string{CanalContactoTelefono, CanalContactoCorreo, CanalContactoSMS, CanalContactoPresencial, CanalContactoOtro}
}
func ResultadosContactoParticipacion() []string {
	return []string{ResultadoContactoContactado, ResultadoContactoNoContesta, ResultadoContactoBuzon, ResultadoContactoAcepta, ResultadoContactoRechaza, ResultadoContactoAplazado, ResultadoContactoOtro, ResultadoContactoEnviado, ResultadoContactoNoEnviado}
}
