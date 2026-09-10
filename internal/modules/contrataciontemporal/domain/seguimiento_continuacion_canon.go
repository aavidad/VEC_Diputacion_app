package domain

import "bytes"

const (
	dominioContinuacionSeguimientoV1     = "vec.dipgra.contratacion-temporal.seguimiento.continuacion"
	dominioEstadoSeguimientoContinuadoV1 = "vec.dipgra.contratacion-temporal.seguimiento.estado-continuado"
)

// SerializarEstadoSeguimientoConContinuacionCanonico usa un dominio binario
// nuevo, version de esquema 1. El canon V1 original permanece cerrado e identico.
// Ningun hash de este codec sustituye la huella de la raiz fundacional.
func SerializarEstadoSeguimientoConContinuacionCanonico(
	original, sucesora DefinicionSeguimiento,
	estado EstadoPersistidoSeguimiento,
) ([]byte, error) {
	validado, err := RehidratarSeguimientoConContinuacion(original, sucesora, estado)
	if err != nil {
		return nil, ErrContinuacionSeguimientoInvalida
	}
	return materialCanonicoEstadoSeguimientoContinuado(validado.estado)
}

// Este material privado se compara con el resultado reconstruido, no acredita
// por si mismo las transiciones. Incluye cada campo del estado y de su adopcion.
func materialCanonicoEstadoSeguimientoContinuado(estado EstadoPersistidoSeguimiento) ([]byte, error) {
	if estado.Continuacion == nil || estado.Version != uint64(len(estado.Actuaciones)) ||
		len(estado.Actuaciones) < 2 || len(estado.Actuaciones) > maximoActuacionesSeguimiento ||
		len(estado.PeriodosResultantes) > maximoActuacionesSeguimiento ||
		!instanteSeguimientoValido(estado.ActualizadoEn) || !estado.EstadoActual.Valida() {
		return nil, ErrContinuacionSeguimientoInvalida
	}
	var material bytes.Buffer
	e := nuevoEscritorCanonSeguimiento(&material, dominioEstadoSeguimientoContinuadoV1)
	e.cadena(estado.Referencia)
	e.cadena(estado.OrganizacionRef)
	e.cadena(estado.ExpedienteRef)
	e.cadena(estado.RelacionRef)
	e.referenciaDefinicion(estado.Definicion)
	e.entero64(estado.Version)
	e.clave(estado.EstadoActual)
	e.intervalo(estado.PeriodoPrevisto)
	e.instante(estado.CreadoEn)
	e.instante(estado.ActualizadoEn)
	e.huella(estado.HuellaRaizSHA256)
	e.entero32(uint32(len(estado.PeriodosResultantes)))
	for _, periodo := range estado.PeriodosResultantes {
		e.intervalo(periodo.Intervalo)
		e.cadena(periodo.ActuacionRef)
	}
	e.booleano(estado.CeseEfectivo != nil)
	if estado.CeseEfectivo != nil {
		e.instante(estado.CeseEfectivo.EfectivoEn)
		e.cadena(estado.CeseEfectivo.ActuacionRef)
	}
	e.entero32(uint32(len(estado.Actuaciones)))
	for _, actuacion := range estado.Actuaciones {
		canonActuacion, err := materialCanonicoActuacionSeguimiento(actuacion)
		if err != nil {
			return nil, ErrContinuacionSeguimientoInvalida
		}
		e.bytes(canonActuacion)
		e.huella(actuacion.HuellaActuacionSHA256)
	}
	e.bytes(materialCanonicoContinuacionSeguimiento(*estado.Continuacion))
	if e.err != nil {
		return nil, ErrContinuacionSeguimientoInvalida
	}
	return material.Bytes(), nil
}

func materialCanonicoContinuacionSeguimiento(c ContinuacionSeguimiento) []byte {
	var material bytes.Buffer
	e := nuevoEscritorCanonSeguimiento(&material, dominioContinuacionSeguimientoV1)
	e.cadena(c.SeguimientoRef)
	e.cadena(c.OrganizacionRef)
	e.cadena(c.ExpedienteRef)
	e.cadena(c.RelacionRef)
	e.referenciaDefinicion(c.DefinicionOriginal)
	e.referenciaDefinicion(c.DefinicionSucesora)
	e.huella(c.HuellaRaizSHA256)
	e.entero64(c.VersionAnterior)
	e.huella(c.HuellaEstadoAnteriorSHA256)
	e.huella(c.HuellaActuacionAnteriorSHA256)
	e.entero64(c.PrimeraSecuencia)
	e.cadena(c.ActuacionRef)
	e.cadena(c.ReciboRef)
	e.cadena(c.CorrelacionRef)
	e.instante(c.RegistradaEn)
	e.huella(c.HuellaPeticionSHA256)
	return material.Bytes()
}
