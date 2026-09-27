package ports

import (
	"context"
	"errors"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	EsquemaHistorialMiBolsa        = "vec.bolsa.mi-bolsa.historial.v1"
	AccionConsultarHistorialPropio = "bolsa.historial_propio.consultar"
	FinalidadHistorialMiBolsa      = "consulta_historial_propio"
	AudienciaHistorialMiBolsa      = EsquemaHistorialMiBolsa
	CampoContratosPropios          = "contratos_propios"
	CampoLlamamientosPropios       = "llamamientos_propios"
	CampoRenunciasPropias          = "renuncias_propias"
	TamanoPaginaHistorialMiBolsa   = 20
	MaximaPaginaHistorialMiBolsa   = 10000
)

var (
	ErrConsultaHistorialMiBolsaInvalida  = errors.New("bolsa: consulta de historial propio invalida")
	ErrHistorialMiBolsaNoDisponible      = errors.New("bolsa: historial propio no disponible")
	ErrResultadoHistorialMiBolsaInvalido = errors.New("bolsa: resultado de historial propio invalido")
)

func CamposHistorialMiBolsa() []string {
	return []string{CampoContratosPropios, CampoLlamamientosPropios, CampoRenunciasPropias}
}

// HechoHistorialMiBolsa contiene exclusivamente datos publicables al titular.
// Los contratos son eventos recibidos en Bolsa; no prueban firma ni alta laboral.
type HechoHistorialMiBolsa struct {
	Clase, Bolsa, Categoria string
	OcurridoEn              time.Time
	Tipo                    string
	Inicio, FinPrevisto     *time.Time
	ModalidadClave          *string
	Procedencia             string
	Canal, Resultado        string
	Respuesta, Modo, Estado string
}

type PaginaHistorialMiBolsa struct {
	ConsultadaEn time.Time
	Pagina       int
	Tamano       int
	HayMas       bool
	Items        []HechoHistorialMiBolsa
}

type SolicitudConsultaHistorialMiBolsa struct {
	CandidatoRef string
	Material     puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	ConsultadaEn time.Time
	Pagina       int
}

// La implementación PostgreSQL consume autorización y audita la lectura en
// la misma transacción que proyecta los eventos propios de Bolsa.
type ConsultaHistorialMiBolsa interface {
	ConsultarHistorialMiBolsa(context.Context, SolicitudConsultaHistorialMiBolsa) (PaginaHistorialMiBolsa, error)
}

type ProveedorMaterialHistorialMiBolsa interface {
	EmitirMaterialHistorialMiBolsa(context.Context, dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.ResultadoContextoActorRegistradoV2, dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}
