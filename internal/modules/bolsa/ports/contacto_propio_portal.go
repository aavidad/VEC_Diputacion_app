package ports

import (
	"context"
	"errors"
	"time"

	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Confirmación del contacto propio desde «Mi bolsa» (duda 45; AD3-86 y
// Bolsa 000040). La persona confirma que la versión vigente de su contacto
// sigue siendo suya; actúa sobre su propio 'mi-bolsa:<candidato>' con la
// bolsa en los atributos.
const (
	AccionConfirmarContactoPropio    = "bolsa.participaciones_propias.confirmar_contacto"
	AudienciaConfirmarContactoPropio = "vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1"
)

// AccionesContactoPropio enumera la acción de AD3-86 con su audiencia.
func AccionesContactoPropio() [][2]string {
	return [][2]string{{AccionConfirmarContactoPropio, AudienciaConfirmarContactoPropio}}
}

var (
	// ErrPortalContactoCambiado: RRHH registró otra versión después de la que
	// vio la persona.
	ErrPortalContactoCambiado = errors.New("bolsa: el contacto ha cambiado")
	// ErrPortalContactoYaConfirmado: esa versión ya estaba confirmada.
	ErrPortalContactoYaConfirmado = errors.New("bolsa: contacto ya confirmado")
	// ErrPortalSinContacto: la participación no tiene contacto registrado.
	ErrPortalSinContacto = errors.New("bolsa: sin contacto que confirmar")
)

// ContactoPortalCandidato es el estado del contacto de una bolsa de la
// persona: versión vigente, marca de origen CONVOCA y confirmación. Nunca
// lleva el claro ni la participación.
type ContactoPortalCandidato struct {
	Bolsa        string
	Version      int64
	Origen       *OrigenContactoPortal
	ConfirmadaEn *time.Time
}

type OrigenContactoPortal struct {
	VigenteHasta time.Time
	UltimoDia    string
}

// ConfirmacionContactoPortal llega a PostgreSQL con el material ya emitido.
type ConfirmacionContactoPortal struct {
	CandidatoRef, Bolsa, Clave, ReciboRef string
	Version                               int64
	ConfirmadaEn                          time.Time
	Material                              puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ReciboConfirmacionContacto struct {
	Reutilizada  bool
	ReciboRef    string
	Version      int64
	ConfirmadaEn time.Time
}

// RegistroConfirmacionContacto consume la decisión y registra la
// confirmación en una sola transacción.
type RegistroConfirmacionContacto interface {
	ConfirmarContacto(context.Context, ConfirmacionContactoPortal) (ReciboConfirmacionContacto, error)
}
