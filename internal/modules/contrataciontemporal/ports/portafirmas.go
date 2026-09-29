package ports

import (
	"context"
	"errors"
	"regexp"
)

// Portafirmas corporativo (Firmadoc). VEC aún no conoce su interfaz (duda 74
// de RRHH y anexo Firmadoc de la petición a Informática), así que la única
// implementación compuesta es la apagada: dice «no conectado» y rechaza todo
// envío. Un conector real solo podrá informar de un envío aceptado; que un
// documento quede firmado se sabrá por su estado en el portafirmas y por la
// verificación del documento devuelto, nunca por la respuesta del envío.

// ErrPortafirmasNoDisponible: el portafirmas no está conectado o no responde.
// Nunca equivale a envío, firma ni devolución.
var ErrPortafirmasNoDisponible = errors.New("contratacion temporal: portafirmas no disponible")

// MotivoPortafirmasConexionPendiente: falta la conexión con Firmadoc.
const MotivoPortafirmasConexionPendiente = "conexion_pendiente"

var motivoPortafirmasValido = regexp.MustCompile(`^[a-z][a-z0-9_]{2,63}$`)

// EstadoConexionPortafirmas dice si se puede enviar y, si no, por qué.
type EstadoConexionPortafirmas struct {
	Conectado bool
	Motivo    string
}

// Validar exige un motivo cerrado cuando no hay conexión y ninguno si la hay.
func (e EstadoConexionPortafirmas) Validar() error {
	if e.Conectado == (e.Motivo != "") || (!e.Conectado && !motivoPortafirmasValido.MatchString(e.Motivo)) {
		return ErrPortafirmasNoDisponible
	}
	return nil
}

// SolicitudEnvioPortafirmas es el documento exacto que se enviaría a firma.
type SolicitudEnvioPortafirmas struct {
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	Documento         string
	OriginalHuella    string
	Original          []byte
	ClaveIdempotencia string
}

// ReciboEnvioPortafirmas acredita solo que el portafirmas aceptó el envío.
type ReciboEnvioPortafirmas struct {
	EnvioRef       string
	OriginalHuella string
}

// ConectorPortafirmas es el adaptador del portafirmas corporativo.
type ConectorPortafirmas interface {
	EstadoConexion(context.Context) (EstadoConexionPortafirmas, error)
	Enviar(context.Context, SolicitudEnvioPortafirmas) (ReciboEnvioPortafirmas, error)
}
