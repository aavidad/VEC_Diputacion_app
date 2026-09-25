package ports

import (
	"context"
	"errors"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Acciones propias del candidato en «Mi bolsa» (AD3-84). El recurso es
// siempre el propio 'mi-bolsa:<candidato>' y la bolsa va en sus atributos.
const (
	AccionSolicitarPausaPropia        = "bolsa.participaciones_propias.solicitar_pausa"
	AccionSolicitarReactivacionPropia = "bolsa.participaciones_propias.solicitar_reactivacion"
	AccionResponderLlamamientoPropio  = "bolsa.participaciones_propias.responder_llamamiento"
	// AccionManifestarDisposicionPropia actúa sobre la oferta publicada
	// (TipoRecursoOfertaBolsa); la consumirá la función de ofertas de Bolsa.
	AccionManifestarDisposicionPropia = "bolsa.participaciones_propias.manifestar_disposicion"

	AudienciaSolicitarPausaPropia        = "vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1"
	AudienciaSolicitarReactivacionPropia = "vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1"
	AudienciaResponderLlamamientoPropio  = "vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1"
	AudienciaManifestarDisposicionPropia = "vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1"

	FinalidadPortalCandidato = "gestion_participaciones_propias"
	TipoRecursoOfertaBolsa   = "oferta_bolsa"
)

// AccionesPortalCandidato enumera las acciones propias con su audiencia, en
// el mismo orden que AD3-84.
func AccionesPortalCandidato() [][2]string {
	return [][2]string{
		{AccionSolicitarPausaPropia, AudienciaSolicitarPausaPropia},
		{AccionSolicitarReactivacionPropia, AudienciaSolicitarReactivacionPropia},
		{AccionResponderLlamamientoPropio, AudienciaResponderLlamamientoPropio},
		{AccionManifestarDisposicionPropia, AudienciaManifestarDisposicionPropia},
	}
}

// Tipos de solicitud y de respuesta del portal.
const (
	SolicitudPortalPausa        = "pausa"
	SolicitudPortalReactivacion = "reactivacion"

	RespuestaPortalAcepta              = "acepta"
	RespuestaPortalRenuncia            = "renuncia"
	RespuestaPortalRenunciaJustificada = "renuncia_justificada"

	ModoRespuestaPortalFirme     = "firme"
	ModoRespuestaPortalPropuesta = "propuesta_rrhh"
)

var (
	ErrPortalCandidatoNoDisponible = errors.New("bolsa: portal del candidato no disponible")
	ErrPortalCandidatoInvalido     = errors.New("bolsa: petición del portal del candidato no válida")
	// Conflictos de negocio: el portal los explica a la persona.
	ErrPortalClaveReutilizada       = errors.New("bolsa: clave del portal reutilizada con otro contenido")
	ErrPortalSolicitudPendiente     = errors.New("bolsa: ya hay una solicitud pendiente de RRHH")
	ErrPortalSituacionNoAdmite      = errors.New("bolsa: la situación actual no admite la solicitud")
	ErrPortalSinLlamamientoAbierto  = errors.New("bolsa: no hay un llamamiento abierto")
	ErrPortalRespuestaFueraDePlazo  = errors.New("bolsa: respuesta fuera de plazo")
	ErrPortalCausaNoAdmitida        = errors.New("bolsa: causa de renuncia no admitida")
	ErrPortalPausaFueraDeLimite     = errors.New("bolsa: fin de la pausa fuera del límite")
	ErrReglasPortalCandidatoAusente = errors.New("bolsa: sin reglas del portal del candidato")
)

// SolicitudPortalCandidato llega a PostgreSQL con el material ya emitido y
// las reglas resueltas; SQL comprueba y conserva, no decide.
type SolicitudPortalCandidato struct {
	SolicitudRef, ReciboRef string
	CandidatoRef, Bolsa     string
	Tipo                    string
	PausaHasta, PausaMaxima *time.Time
	SituacionesAdmitidas    []string
	ReglaRef, Clave         string
	RegistradaEn            time.Time
	Material                puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ReciboSolicitudPortal struct {
	Reutilizada             bool
	SolicitudRef, ReciboRef string
	RegistradaEn            time.Time
}

// RespuestaPortalCandidato conserva el contacto que abrió el plazo y su
// vencimiento; SQL exige que el contacto sea el vigente y la hora anterior.
type RespuestaPortalCandidato struct {
	RespuestaRef, ReciboRef string
	CandidatoRef, Bolsa     string
	Respuesta               string
	Causa                   string
	JustificanteRef         string
	JustificanteSHA256      string
	Modo                    string
	ResultadosEfectivos     []string
	ReglaRef, Clave         string
	RespondidaEn            time.Time
	Material                puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ReciboRespuestaPortal struct {
	Reutilizada              bool
	RespuestaRef, ReciboRef  string
	RespondidaEn             time.Time
	Modo                     string
	ContactoEn, VenceAntesDe time.Time
}

// PlazoRespuestaPortal calcula el vencimiento desde el contacto efectivo. Se
// invoca dentro de la escritura, tras leer el contacto vigente.
type PlazoRespuestaPortal interface {
	VencimientoRespuesta(ctx context.Context, contacto time.Time) (time.Time, error)
}

// EstadoPortalCandidato resume por bolsa lo que la persona puede hacer. No
// contiene referencias de participación.
type EstadoPortalCandidato struct {
	Bolsa              string
	LlamamientoAbierto *LlamamientoAbiertoPortal
	SolicitudPendiente *SolicitudPendientePortal
	UltimaRespuesta    *UltimaRespuestaPortal
}

type LlamamientoAbiertoPortal struct {
	ContactoEn   time.Time
	VenceAntesDe *time.Time
}

type SolicitudPendientePortal struct {
	Tipo, Recibo string
	RegistradaEn time.Time
	PausaHasta   *time.Time
}

type UltimaRespuestaPortal struct {
	Respuesta, Modo, Recibo string
	RespondidaEn            time.Time
}

// RegistroPortalCandidato es el contrato exacto para PostgreSQL: cada
// escritura consume la decisión dentro de su transacción.
type RegistroPortalCandidato interface {
	SolicitarPortal(context.Context, SolicitudPortalCandidato) (ReciboSolicitudPortal, error)
	ResponderPortal(context.Context, RespuestaPortalCandidato, PlazoRespuestaPortal) (ReciboRespuestaPortal, error)
}

// ReglasPortalCandidato traduce el catálogo de reglas a lo que necesita el
// portal. Cada método devuelve la referencia de la regla que aplica.
type ReglasPortalCandidato interface {
	ModoRespuesta(context.Context) (modo, reglaRef string, err error)
	ResultadosContactoEfectivo(context.Context) ([]string, error)
	SituacionesAdmitidas(ctx context.Context, tipo string) (situaciones []string, reglaRef string, err error)
	PausaMaxima(ctx context.Context, desde time.Time) (hasta time.Time, reglaRef string, err error)
	VencimientoRespuesta(ctx context.Context, contacto time.Time) (vence time.Time, reglaRef string, err error)
	CausasRenunciaJustificada(context.Context) ([]string, error)
}

// ProveedorMaterialPortalCandidato emite el material de la acción ya
// decidida, con la audiencia que corresponde a esa acción.
type ProveedorMaterialPortalCandidato interface {
	EmitirMaterialPortalCandidato(ctx context.Context, accion string, solicitud dominiovec.SolicitudAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2, decision dominiovec.DecisionAutorizacionLigadaV3, confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}
