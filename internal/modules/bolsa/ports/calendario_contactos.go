package ports

import (
	"context"
	"strconv"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionEntregarCalendarioContactos    = "calendarios.calendario_contactos.entregar"
	AudienciaEntregarCalendarioContactos = "vec_bolsa_llamamientos.calendario_contactos.importar.v1"
	FinalidadEntregarCalendarioContactos = "entregar_calendario_contactos"
)

func RecursoCalendarioContactos(sedeRef string, anio int) string {
	return "calendario-contactos:" + sedeRef + ":" + strconv.Itoa(anio)
}

// FuenteCalendarioContactos es una proyección publicada por Calendarios.
// Ningún valor de este contrato se obtiene del cuerpo HTTP de un contacto.
// HuellaFuenteSHA256 resume el material anual con versión Bolsa fija a 1;
// así puede comprobarse antes de asignar la versión local por CAS.
type FuenteCalendarioContactos struct {
	Tipo               string
	SedeRef            string
	Anio               int
	Fuentes            []dominiobolsa.VersionFuenteCalendario
	Dias               []dominiobolsa.DiaCalendarioContactos
	HuellaFuenteSHA256 string
}

type ConsultaCalendarioContactosPublicado interface {
	ObtenerPublicado(context.Context, string, string, int) (FuenteCalendarioContactos, error)
}

// PublicacionCalendarioContactos exige CAS e historia de solo adición.
// La implementación PostgreSQL consume autorización V3 e inserta el recibo
// en la misma transacción. Reutilizado no crea otra versión.
type PublicacionCalendarioContactos interface {
	VersionActual(context.Context, string, string, int) (dominiobolsa.CalendarioContactos, bool, error)
	PublicarSiVersion(context.Context, OrdenPublicarCalendarioContactos) (ReciboCalendarioContactos, error)
}

type OrdenPublicarCalendarioContactos struct {
	Calendario        dominiobolsa.CalendarioContactos
	VersionEsperada   uint64
	HuellaFuente      string
	ActorRef          string
	ClaveIdempotencia string
	ReciboRef         string
	Material          puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ReciboCalendarioContactos struct {
	ReciboRef    string
	Tipo         string
	SedeRef      string
	Anio         int
	Version      uint64
	HuellaSHA256 string
	Reutilizado  bool
}
