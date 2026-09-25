package domain

import "time"

// TratamientoRespuestaFueraDePlazo es el mecanismo técnico con el que se trata
// una respuesta recibida tras el vencimiento. Cuál se aplica lo decide el
// catálogo de reglas (atributo «tratamiento»), nunca este paquete.
type TratamientoRespuestaFueraDePlazo string

const (
	TratamientoFueraDePlazoAdmitir               TratamientoRespuestaFueraDePlazo = "admitir"
	TratamientoFueraDePlazoExigeCausaJustificada TratamientoRespuestaFueraDePlazo = "exige_causa_justificada"
	TratamientoFueraDePlazoNoAdmitir             TratamientoRespuestaFueraDePlazo = "no_admitir"
)

func (t TratamientoRespuestaFueraDePlazo) Valido() bool {
	return t == TratamientoFueraDePlazoAdmitir || t == TratamientoFueraDePlazoExigeCausaJustificada ||
		t == TratamientoFueraDePlazoNoAdmitir
}

// ConfirmacionExpiracionLlamamiento indica quién confirma la propuesta de no
// aceptación y siguiente candidato al vencer sin respuesta. Solo existe la
// confirmación expresa de RRHH: VEC propone y no ejecuta nada por sí mismo.
type ConfirmacionExpiracionLlamamiento string

const ConfirmacionExpiracionRRHH ConfirmacionExpiracionLlamamiento = "rrhh"

func (c ConfirmacionExpiracionLlamamiento) Valida() bool { return c == ConfirmacionExpiracionRRHH }

// SituacionPlazoRespuesta es la lectura del plazo en un instante dado.
type SituacionPlazoRespuesta string

const (
	PlazoRespuestaEnPlazo SituacionPlazoRespuesta = "en_plazo"
	PlazoRespuestaVencido SituacionPlazoRespuesta = "vencido"
)

// SituacionPlazo compara un instante con el primer instante vencido. El plazo
// incluye todo el último día: vence exactamente en respuestaHasta.
func SituacionPlazo(respuestaHasta, instante time.Time) SituacionPlazoRespuesta {
	if instante.Before(respuestaHasta) {
		return PlazoRespuestaEnPlazo
	}
	return PlazoRespuestaVencido
}

// AdmisionRespuesta es el resultado de aplicar la regla a una respuesta.
type AdmisionRespuesta string

const (
	RespuestaAdmitidaEnPlazo          AdmisionRespuesta = "en_plazo"
	RespuestaAdmitidaFueraDePlazo     AdmisionRespuesta = "fuera_de_plazo_admitida"
	RespuestaRequiereCausaJustificada AdmisionRespuesta = "requiere_causa_justificada"
	RespuestaNoAdmitida               AdmisionRespuesta = "no_admitida"
)

// EvaluarAdmisionRespuesta aplica el tratamiento capturado al abrir el plazo.
// La persistencia repite esta comprobación con sus datos durables.
func EvaluarAdmisionRespuesta(
	tratamiento TratamientoRespuestaFueraDePlazo,
	respuestaHasta, recibidaEn time.Time,
	causaAcreditada bool,
) AdmisionRespuesta {
	if SituacionPlazo(respuestaHasta, recibidaEn) == PlazoRespuestaEnPlazo {
		return RespuestaAdmitidaEnPlazo
	}
	switch tratamiento {
	case TratamientoFueraDePlazoAdmitir:
		return RespuestaAdmitidaFueraDePlazo
	case TratamientoFueraDePlazoExigeCausaJustificada:
		if causaAcreditada {
			return RespuestaAdmitidaFueraDePlazo
		}
		return RespuestaRequiereCausaJustificada
	default:
		return RespuestaNoAdmitida
	}
}

// ProcedePropuestaExpiracion indica si VEC debe proponer la no aceptación y
// el siguiente candidato: plazo vencido y ninguna respuesta registrada.
func ProcedePropuestaExpiracion(respuestaHasta, instante time.Time, respuestaRegistrada bool) bool {
	return !respuestaRegistrada && SituacionPlazo(respuestaHasta, instante) == PlazoRespuestaVencido
}
