package ports

import (
	"context"
	"errors"
	"regexp"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const prefijoReferenciaBorradorLlamamiento = "borrador-llamamiento:alta:"

var (
	patronHuellaBorradorLlamamiento       = regexp.MustCompile(`^[0-9a-f]{64}$`)
	patronReciboBorradorLlamamiento       = regexp.MustCompile(`^recibo:[a-p]{64}$`)
	patronActorIntentoBorradorLlamamiento = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
)

const (
	AccionCrearBorradorLlamamientoInterno        = "bolsa.llamamiento.borrador_interno.crear"
	AccionConsultarBorradorLlamamientoInterno    = "bolsa.llamamiento.borrador_interno.consultar"
	FinalidadCrearBorradorLlamamientoInterno     = "gestion_borradores_llamamiento_interno"
	FinalidadConsultarBorradorLlamamientoInterno = "consulta_borrador_llamamiento_interno"
	AudienciaCrearBorradorLlamamientoInterno     = "vec_bolsa_llamamientos.borrador_llamamiento_interno.crear.v1"
	AudienciaConsultarBorradorLlamamientoInterno = "vec_bolsa_llamamientos.borrador_llamamiento_interno.consultar.v1"
	ModuloBorradorLlamamiento                    = "bolsa"
	TipoRecursoBorradorLlamamiento               = "borrador_llamamiento_interno"
)

var (
	ErrSolicitudBorradorLlamamientoInvalida  = errors.New("bolsa: solicitud de borrador de llamamiento invalida")
	ErrClaveBorradorLlamamientoReutilizada   = errors.New("bolsa: clave de borrador de llamamiento reutilizada con otro comando")
	ErrBorradorLlamamientoNoEncontrado       = errors.New("bolsa: borrador de llamamiento no encontrado")
	ErrFuenteBorradorLlamamientoNoDisponible = errors.New("bolsa: fuente de borrador de llamamiento no disponible")
)

type ContextoBorradorLlamamientoResuelto struct {
	UnidadRef string
	AmbitoRef string
}

func (c ContextoBorradorLlamamientoResuelto) Validar() error {
	if c.UnidadRef == "" || c.AmbitoRef == "" {
		return ErrSolicitudBorradorLlamamientoInvalida
	}
	return nil
}

// SolicitudCrearBorradorLlamamiento solo se construye detrás de la frontera
// interna: actor, vínculo V3 y correlación jamás proceden del cuerpo HTTP.
type SolicitudCrearBorradorLlamamiento struct {
	Vinculo           dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto dominiovec.ResultadoContextoActorRegistradoV2
	ClaveIdempotencia string
	Contenido         dominiobolsa.ContenidoBorradorLlamamiento
	Correlacion       dominiovec.ReferenciaCorrelacionAutorizacionV2
	Motivo            dominiovec.ReferenciaEntradaCatalogo
}

func (s SolicitudCrearBorradorLlamamiento) Validar() error {
	if s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil || s.ClaveIdempotencia == "" || s.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.Motivo) {
		return ErrSolicitudBorradorLlamamientoInvalida
	}
	return nil
}

type SolicitudConsultarBorradorLlamamiento struct {
	Vinculo           dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto dominiovec.ResultadoContextoActorRegistradoV2
	BorradorRef       string
	Correlacion       dominiovec.ReferenciaCorrelacionAutorizacionV2
	Motivo            dominiovec.ReferenciaEntradaCatalogo
}

func (s SolicitudConsultarBorradorLlamamiento) Validar() error {
	if s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil || s.BorradorRef == "" || s.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.Motivo) {
		return ErrSolicitudBorradorLlamamientoInvalida
	}
	return nil
}

type ResolutorContextoBorradorLlamamiento interface {
	ResolverContextoBorradorLlamamiento(context.Context, dominiovec.ContextoActor) (ContextoBorradorLlamamientoResuelto, error)
}

type AutorizadorBorradorLlamamientoV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type ComandoCrearBorradorLlamamiento struct {
	Borrador              dominiobolsa.BorradorLlamamiento
	ClaveIdempotencia     string
	HuellaComandoSHA256   string
	SolicitudAutorizacion dominiovec.SolicitudAutorizacionLigadaV3
	Decision              dominiovec.DecisionAutorizacionLigadaV3
	Confirmacion          puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material              puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (c ComandoCrearBorradorLlamamiento) Validar() error {
	if _, err := c.SolicitudAutorizacion.Datos(); c.Borrador.Validar() != nil || c.ClaveIdempotencia == "" || len(c.HuellaComandoSHA256) != 64 || err != nil || c.Decision.ValidarPara(c.SolicitudAutorizacion) != nil || c.Confirmacion.Validar() != nil || c.Material.ValidarEstructura() != nil {
		return ErrSolicitudBorradorLlamamientoInvalida
	}
	return nil
}

type ReciboBorradorLlamamiento struct {
	Referencia           string
	Borrador             dominiobolsa.BorradorLlamamiento
	HuellaComandoSHA256  string
	ReintentoIdempotente bool
	RegistradoEn         time.Time
}

func (r ReciboBorradorLlamamiento) Validar() error {
	if !patronReciboBorradorLlamamiento.MatchString(r.Referencia) || r.Borrador.Validar() != nil || !patronHuellaBorradorLlamamiento.MatchString(r.HuellaComandoSHA256) || r.Borrador.Referencia() != prefijoReferenciaBorradorLlamamiento+r.HuellaComandoSHA256 || r.RegistradoEn.IsZero() {
		return ErrSolicitudBorradorLlamamientoInvalida
	}
	return nil
}

type TransaccionBorradorLlamamiento interface {
	CrearBorradorLlamamiento(context.Context, ComandoCrearBorradorLlamamiento) (ReciboBorradorLlamamiento, error)
}
type LectorBorradorLlamamiento interface {
	ObtenerBorradorLlamamiento(context.Context, string, string, string, string, dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboBorradorLlamamiento, error)
}

// AccionIntentoBorradorLlamamiento y los tipos asociados forman el contrato
// mínimo de la bitácora de frontera. No son estados del agregado ni contienen
// su referencia, cuerpo, material de autorización o clave de idempotencia.
type AccionIntentoBorradorLlamamiento string

const (
	AccionIntentoCrearBorradorLlamamiento       AccionIntentoBorradorLlamamiento = "crear"
	AccionIntentoConsultarBorradorLlamamiento   AccionIntentoBorradorLlamamiento = "consultar"
	AccionIntentoCambiarSituacionParticipacion  AccionIntentoBorradorLlamamiento = "cambiar_situacion"
	AccionIntentoRegistrarContactoParticipacion AccionIntentoBorradorLlamamiento = "registrar_contacto"
)

type ClaseRutaIntentoBorradorLlamamiento string

const (
	ClaseRutaColeccionBorradorLlamamiento ClaseRutaIntentoBorradorLlamamiento = "coleccion"
	ClaseRutaDetalleBorradorLlamamiento   ClaseRutaIntentoBorradorLlamamiento = "detalle"
	ClaseRutaSituacionParticipacion       ClaseRutaIntentoBorradorLlamamiento = "situacion"
	ClaseRutaContactosParticipacion       ClaseRutaIntentoBorradorLlamamiento = "contactos"
)

type ResultadoIntentoBorradorLlamamiento string

const (
	ResultadoIntentoAutenticacionRequeridaBorradorLlamamiento      ResultadoIntentoBorradorLlamamiento = "autenticacion_requerida"
	ResultadoIntentoAccesoDenegadoBorradorLlamamiento              ResultadoIntentoBorradorLlamamiento = "acceso_denegado"
	ResultadoIntentoRecursoNoDisponibleBorradorLlamamiento         ResultadoIntentoBorradorLlamamiento = "recurso_no_disponible"
	ResultadoIntentoInfraestructuraNoDisponibleBorradorLlamamiento ResultadoIntentoBorradorLlamamiento = "infraestructura_no_disponible"
	ResultadoIntentoIndeterminadoBorradorLlamamiento               ResultadoIntentoBorradorLlamamiento = "resultado_indeterminado"
)

type IntentoBorradorLlamamiento struct {
	Correlacion     dominiovec.ReferenciaCorrelacionAutorizacionV2
	Accion          AccionIntentoBorradorLlamamiento
	ClaseRuta       ClaseRutaIntentoBorradorLlamamiento
	ActorVerificado string
	Resultado       ResultadoIntentoBorradorLlamamiento
}

func (i IntentoBorradorLlamamiento) Validar() error {
	if i.Correlacion.Validar() != nil ||
		(i.Accion != AccionIntentoCrearBorradorLlamamiento && i.Accion != AccionIntentoConsultarBorradorLlamamiento && i.Accion != AccionIntentoCambiarSituacionParticipacion && i.Accion != AccionIntentoRegistrarContactoParticipacion) ||
		(i.ClaseRuta != ClaseRutaColeccionBorradorLlamamiento && i.ClaseRuta != ClaseRutaDetalleBorradorLlamamiento && i.ClaseRuta != ClaseRutaSituacionParticipacion && i.ClaseRuta != ClaseRutaContactosParticipacion) ||
		(i.ActorVerificado != "" && !patronActorIntentoBorradorLlamamiento.MatchString(i.ActorVerificado)) ||
		(i.Resultado != ResultadoIntentoAutenticacionRequeridaBorradorLlamamiento &&
			i.Resultado != ResultadoIntentoAccesoDenegadoBorradorLlamamiento &&
			i.Resultado != ResultadoIntentoRecursoNoDisponibleBorradorLlamamiento &&
			i.Resultado != ResultadoIntentoInfraestructuraNoDisponibleBorradorLlamamiento &&
			i.Resultado != ResultadoIntentoIndeterminadoBorradorLlamamiento) {
		return ErrSolicitudBorradorLlamamientoInvalida
	}
	return nil
}

// RegistradorIntentoBorradorLlamamiento se invoca fuera de la transacción del
// caso principal tras un fallo o una denegación. Su éxito no altera el
// agregado, el recibo ni el outbox del borrador.
type RegistradorIntentoBorradorLlamamiento interface {
	RegistrarIntentoBorradorLlamamiento(context.Context, IntentoBorradorLlamamiento) error
}
