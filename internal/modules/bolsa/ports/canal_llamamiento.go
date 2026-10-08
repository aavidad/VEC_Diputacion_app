package ports

import (
	"context"
	"errors"
	"time"
)

var (
	ErrCanalAvisoNoDisponible = errors.New("bolsa: canal de aviso no disponible")
	ErrMensajeAvisoInvalido   = errors.New("bolsa: mensaje de aviso invalido")
)

// CanalAvisoLlamamiento es el adaptador de un canal de aviso. Disponible dice
// si su infraestructura real existe (función SQL instalada, proveedor
// contratado…); nunca hay un proveedor ficticio detrás de un canal disponible.
type CanalAvisoLlamamiento interface {
	Canal() string
	Disponible(context.Context) bool
}

// MensajeAvisoLlamamiento es el aviso corto de los canales de mensaje (SMS,
// Telegram). Lleva lo mínimo: un texto de catálogo sin datos personales y el
// enlace al portal donde la persona consulta el llamamiento. Un aviso no es
// una notificación administrativa (Ley 39/2015).
type MensajeAvisoLlamamiento struct {
	BolsaRef, LlamamientoRef, ParticipacionRef string
	Texto, EnlacePortal                        string
	Instante                                   time.Time
}

// ResultadoAvisoCanal es lo que el canal devuelve para registrarlo como
// contacto: «enviado», «no_enviado» o «no_entregado», y la referencia del
// proveedor para cruzar un acuse posterior.
type ResultadoAvisoCanal struct {
	Resultado           string
	ReferenciaProveedor string
}

// EnviadorAvisoLlamamiento lo implementan los canales automáticos de
// seguimiento (SMS, Telegram). El correo conserva su emisión propia.
type EnviadorAvisoLlamamiento interface {
	CanalAvisoLlamamiento
	EnviarAviso(context.Context, MensajeAvisoLlamamiento) (ResultadoAvisoCanal, error)
}

// FuenteMovilParticipacion entrega el móvil vigente de una participación con
// la lectura nominal auditada de los datos de contacto (B78). Sin móvil
// válido devuelve "" y el aviso queda «no_enviado».
type FuenteMovilParticipacion interface {
	MovilParticipacion(context.Context, string) (string, error)
}

// ProveedorSMS es el servicio corporativo de SMS (pendiente de Informática).
// Coste y límites los fija su contrato; la referencia sirve para el acuse.
type ProveedorSMS interface {
	EnviarSMS(ctx context.Context, movil, texto string) (referencia string, err error)
}

// VinculacionesTelegram resuelve el chat de una persona que se dio de alta
// voluntariamente en Telegram desde su área personal, con un código de un
// solo uso, y que no se ha dado de baja. Sin vínculo vigente no se envía.
type VinculacionesTelegram interface {
	ChatVinculado(ctx context.Context, participacionRef string) (chat string, vigente bool, err error)
}

// ProveedorTelegram es el bot corporativo; su token vive en configuración
// privada, nunca en Git ni en el código.
type ProveedorTelegram interface {
	EnviarMensaje(ctx context.Context, chat, texto string) (referencia string, err error)
}
