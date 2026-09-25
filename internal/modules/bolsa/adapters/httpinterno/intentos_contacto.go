package httpinterno

import (
	"context"
	"errors"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// limiteHistoricoIntentos es la página que se lee cuando la ficha pide el
// control de intentos de un llamamiento: el máximo de la consulta.
const limiteHistoricoIntentos = 100

// EvaluadorIntentosContacto es opcional: si el operador lo implementa, la
// consulta de contactos con ?llamamiento_ref= añade el control de intentos.
type EvaluadorIntentosContacto interface {
	EstadoIntentosTelefonicos(context.Context, string, []dominiobolsa.ContactoParticipacion, bool) (puertosbolsa.EstadoIntentosContacto, error)
}

func referenciaLlamamientoConsultaValida(valores []string) bool {
	if len(valores) != 1 {
		return false
	}
	v := valores[0]
	return v != "" && len(v) <= 256 && strings.TrimSpace(v) == v && !strings.ContainsAny(v, "\x00\r\n\t ")
}

func (h *HandlerContactoParticipacion) estadoIntentos(ctx context.Context, llamamiento string, contactos []dominiobolsa.ContactoParticipacion, completo bool) (map[string]any, error) {
	evaluador, ok := h.operador.(EvaluadorIntentosContacto)
	if !ok {
		return map[string]any{"configurado": false, "llamamiento_ref": llamamiento}, nil
	}
	e, err := evaluador.EstadoIntentosTelefonicos(ctx, llamamiento, contactos, completo)
	if err != nil {
		return nil, err
	}
	if !e.Configurada {
		return map[string]any{"configurado": false, "llamamiento_ref": llamamiento}, nil
	}
	reglas := make([]map[string]any, 0, len(e.Reglas))
	for _, r := range e.Reglas {
		reglas = append(reglas, map[string]any{"clave": r.Clave, "etiqueta": r.Etiqueta, "descripcion": r.Descripcion, "referencia": r.Referencia, "ejemplo": r.Ejemplo})
	}
	salida := salidaEstadoIntentos(e.Estado)
	salida["configurado"] = true
	salida["llamamiento_ref"] = e.LlamamientoRef
	salida["completo"] = e.Completo
	salida["ahora"] = e.Ahora.UTC().Format(time.RFC3339)
	salida["intentos_por_proceso"] = e.Politica.IntentosPorProceso
	salida["procesos"] = e.Politica.Procesos
	salida["separacion_minutos"] = int(e.Politica.SeparacionMinima / time.Minute)
	salida["control_separacion"] = e.Politica.ControlSeparacion
	salida["resultados_sin_contacto"] = append([]string(nil), e.Politica.ResultadosSinContacto...)
	if e.Politica.Franja.Zona != nil {
		salida["franja"] = map[string]any{"valor": e.Politica.Franja.Texto, "solo_dias_habiles": e.Politica.Franja.SoloDiasHabiles, "control": e.Politica.Franja.Control}
	} else {
		salida["franja"] = nil
	}
	salida["reglas"] = reglas
	return salida, nil
}

func salidaEstadoIntentos(e dominiobolsa.EstadoIntentosTelefonicos) map[string]any {
	avisos := append([]string{}, e.Avisos...)
	return map[string]any{
		"sin_contacto": e.SinContacto, "maximo": e.Maximo, "proceso": e.Proceso, "intento": e.Intento,
		"contactado": e.Contactado, "ultimo_intento": instanteIntento(e.UltimoIntento),
		"siguiente_permitido_desde": instanteIntento(e.SiguientePermitidoDesde),
		"baja_propuesta":            e.BajaPropuesta, "avisos": avisos,
	}
}

func instanteIntento(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

// codigoErrorIntento traduce los rechazos de las reglas de intentos.
func codigoErrorIntento(err error) string {
	switch {
	case errors.Is(err, dominiobolsa.ErrIntentoAntesDeSeparacion):
		return "intento_antes_de_separacion"
	case errors.Is(err, dominiobolsa.ErrIntentoFueraDeFranja):
		return "intento_fuera_de_franja"
	case errors.Is(err, dominiobolsa.ErrIntentosContactoAgotados):
		return "intentos_agotados"
	}
	return ""
}
