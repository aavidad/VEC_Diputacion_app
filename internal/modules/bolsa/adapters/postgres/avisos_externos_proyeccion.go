package postgres

import (
	"crypto/sha256"
	"encoding/hex"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// La emisión reúne todos los participantes aunque solo parte del lote tenga
// resultado terminal. Un terminal referencia su contacto B7; el pendiente
// conserva el recibo de cola y no acredita un intento SMTP confirmado.
func proyectarContactosAvisosExternos(e ports.EmisionLlamamiento, pendientes []ports.AvisoExternoPendiente, bolsa, clave string) ([]ports.ResultadoContactoEmision, error) {
	if bolsa == "" || clave == "" || len(pendientes) != len(e.Participaciones) {
		return nil, ports.ErrEmisionLlamamientoNoDisponible
	}
	contactos := make([]ports.ResultadoContactoEmision, 0, len(e.Participaciones))
	eventos := make(map[string]ports.AvisoExternoPendiente, len(pendientes))
	for _, p := range pendientes {
		if !patronReciboOutboxAvisoExterno.MatchString(p.ReciboOutboxRef) {
			return nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		hRecibo := sha256.Sum256([]byte(p.Evento.ProductorRef + "\x1f" + p.Evento.EventoRef + "\x1f" + p.Huella))
		if p.ReciboOutboxRef != "recibo_outbox:"+hex.EncodeToString(hRecibo[:]) {
			return nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		eventos[p.Evento.EventoRef] = p
	}
	for _, participacion := range e.Participaciones {
		eventHash := sha256.Sum256([]byte(e.LlamamientoRef + "\x1f" + participacion))
		pendiente, exists := eventos["evento_aviso:"+hex.EncodeToString(eventHash[:])]
		if !exists {
			return nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		resultado := "aviso_pendiente"
		switch pendiente.EstadoDespacho {
		case "", "reservado_incierto":
		case "aceptado":
			resultado = "enviado"
		case "no_aceptado", "sin_destino":
			resultado = "no_enviado"
		default:
			return nil, ports.ErrEmisionLlamamientoNoDisponible
		}
		recibo := pendiente.ReciboOutboxRef
		if resultado != "aviso_pendiente" {
			hContacto := sha256.Sum256([]byte(bolsa + "\x1f" + clave + "\x1f" + participacion))
			recibo = "recibo:contacto:" + hex.EncodeToString(hContacto[:])
		}
		contactos = append(contactos, ports.ResultadoContactoEmision{ParticipacionRef: participacion, Resultado: resultado, ReciboRef: recibo})
	}
	return contactos, nil
}
