package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	vd "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	FinalidadRRHHBolsasConsultar       = "consulta_rrhh_bolsas"
	FinalidadRRHHEstadisticasConsultar = "consulta_rrhh_estadisticas"
	FinalidadRRHHCandidatosConsultar   = "consulta_rrhh_candidatos"
	TipoRecursoRRHHBolsas              = "coleccion_bolsas_rrhh"
	TipoRecursoRRHHEstadisticas        = "estadisticas_bolsas_rrhh"
	TipoRecursoRRHHCandidatos          = "consulta_candidatos_bolsa"
	RecursoRRHHBolsas                  = "coleccion:bolsa:rrhh:bolsas"
	RecursoRRHHEstadisticas            = "coleccion:bolsa:rrhh:estadisticas"
)

// La identidad procede de la frontera interna registrada y el ámbito se
// configura al montar Bolsa; ninguno llega del navegador.
type OrdenConsultaResumenRRHH struct {
	Accion               string
	UnidadRef, AmbitoRef string
	Resultado            vd.ResultadoContextoActorRegistradoV2
	Vinculo              vd.VinculoAutenticacionActorV2
	Motivo               vd.ReferenciaEntradaCatalogo
	Correlacion          vd.ReferenciaCorrelacionAutorizacionV2
}

type AutorizadorConsultaRRHHV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vd.SolicitudAutorizacionLigadaV3, vd.ResultadoContextoActorRegistradoV2) (
		vd.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
		vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

const (
	ModoPaginaCandidatosRRHH       = "pagina"
	ModoBarridoTextoCandidatosRRHH = "barrido_texto"
	ModoPaginaTextoCandidatosRRHH  = "pagina_texto"
)

type ConsultaCandidatosRRHHNominal struct {
	BolsaRef, Estado, Texto, CursorToken, CursorRef, SnapshotSHA256 string
	Limite                                                          int
	Modo                                                            string
	SeleccionRefs                                                   []string
	CeseActivo                                                      bool
}

type CandidatoRRHHNominal struct {
	ParticipacionRef     string
	Orden                *int
	OrdenActa            int
	RazonOrden           string
	NombreVisible        string
	DocumentoEnmascarado string
	Estado               string
	EstadoDesde          time.Time
	DisponibleDesde      *time.Time
}

type TurnoCandidatoRRHHNominal struct {
	Candidato CandidatoRRHHNominal
	Contacto  *domain.ContactoParticipacion
}

type PaginaCandidatosRRHHNominal struct {
	GeneradoEn                        time.Time
	BolsaRef, CategoriaRef, TipoLista string
	ConfirmadaEn, VigenteDesde        time.Time
	VigenteHasta                      *time.Time
	Politica                          domain.PoliticaOrdenBolsa
	LlamamientosEnCurso               int
	SnapshotSHA256                    string
	Total, TotalPrefiltrado           int
	TotalFiltrado                     int
	PorEstado                         map[string]int
	Candidatos                        []CandidatoRRHHNominal
	Contactos                         []domain.ContactoParticipacion
	Marcas                            map[string]domain.MarcasParticipacion
	TurnoSiguiente                    *CandidatoRRHHNominal
	TurnoUltimo                       *TurnoCandidatoRRHHNominal
	HayMas                            bool
	CursorRefSiguiente                string
	FilasDescifradas                  int
	ConsumoHuellaSHA256               string
	SeleccionRefs                     []string
}

type LectorCandidatosRRHHNominal interface {
	LeerCandidatosNominal(context.Context, ConsultaCandidatosRRHHNominal, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (PaginaCandidatosRRHHNominal, error)
}
