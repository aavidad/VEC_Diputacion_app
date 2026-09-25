package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

// ErrOrigenDatosContactoNoConfigurado: sin la regla del catálogo no se puede
// dar vigencia a un contacto de origen CONVOCA; el alta se rechaza en lugar de
// inventar un plazo.
var ErrOrigenDatosContactoNoConfigurado = errors.New("bolsa: origen de datos de contacto no configurado")

// PoliticaOrigenDatosContacto calcula la marca de un contacto de origen
// CONVOCA registrado en ese instante, con la regla vigente del catálogo.
type PoliticaOrigenDatosContacto interface {
	MarcaOrigenConvoca(context.Context, time.Time) (dominiobolsa.MarcaOrigenDatosContacto, error)
}

// Avisos a RRHH al emitir un llamamiento por correo. No bloquean el envío:
// RRHH decide y, si hace falta, llama por teléfono.
const (
	// AvisoContactoNoConfirmado: el contacto vigente es de origen CONVOCA y su
	// vigencia ha vencido sin que la persona lo confirme.
	AvisoContactoNoConfirmado = "contacto_origen_convoca_no_confirmado"
	// AvisoEstadoContactoNoDisponible: no se pudo comprobar el origen; nunca
	// se da por confirmado.
	AvisoEstadoContactoNoDisponible = "estado_contacto_no_disponible"
)

type AvisoContactoEmision struct {
	ParticipacionRef string `json:"participacion_ref"`
	Aviso            string `json:"aviso"`
	UltimoDia        string `json:"ultimo_dia,omitempty"`
}

// FuenteOrigenContactoParticipacion devuelve la marca de origen del contacto
// vigente de una participación, o nil si es propio o no hay contacto.
type FuenteOrigenContactoParticipacion interface {
	OrigenContactoParticipacion(context.Context, string) (*dominiobolsa.MarcaOrigenDatosContacto, error)
}
