package ports

import (
	"context"
	"errors"
	"time"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultarCatalogoRutasDietas = "dietas.ruta.catalogo.consultar"
	AccionSolicitarCalculoRutasDietas  = "dietas.ruta.calculo.solicitar"
	AudienciaAccesoRutasDietas         = "vec_dietas_rutas_v1.acceso.v1"
	FinalidadConsultarItinerarioDietas = "consultar_itinerario_dietas"
	TipoCatalogoRutasDietas            = "catalogo_rutas_dietas"
	TipoCalculoRutasDietas             = "calculo_rutas_dietas"
)

var (
	ErrAccesoRutasDietasDenegado     = errors.New("dietas: acceso a rutas denegado")
	ErrAccesoRutasDietasNoDisponible = errors.New("dietas: acceso a rutas no disponible")
)

// SolicitudAccesoRutasDietas nace en frontera autenticada. Nunca transporta
// coordenadas: Recurso ata ámbito, versión, huella y canal observados.
type SolicitudAccesoRutasDietas struct {
	ResultadoContexto vecdomain.ResultadoContextoActorRegistradoV2
	Vinculo           vecdomain.VinculoAutenticacionActorV2
	ReferenciaMotivo  vecdomain.ReferenciaEntradaCatalogo
	Correlacion       vecdomain.ReferenciaCorrelacionAutorizacionV2
	Accion            string
	Recurso           vecdomain.RecursoAutorizable
	Audiencia         string
	Finalidad         string
}

// ProveedorMaterialAccesoRutas recibe sólo artefactos V3 ya verificados; no
// reconstruye autoridad, identidad, claves ni material desde HTTP.
type ProveedorMaterialAccesoRutas interface {
	ProveerMaterialAccesoRutas(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenConsumoAccesoRutasDietas struct {
	Solicitud vecdomain.SolicitudAutorizacionLigadaV3
	Material  vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type ReciboAccesoRutasDietas struct {
	DecisionRef         string
	EfectoRef           string
	HuellaEfectoSHA256  string
	ConsumoHuellaSHA256 string
	AuditoriaRef        string
	ConsumidaEn         time.Time
	ConsumoNuevo        bool
}
type ConsumidorAccesoRutasDietas interface {
	ConsumirAccesoRutasDietas(context.Context, OrdenConsumoAccesoRutasDietas) (ReciboAccesoRutasDietas, error)
}
