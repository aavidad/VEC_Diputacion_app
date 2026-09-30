package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrLecturaRPTInvalida     = errors.New("vec: consulta de categorias RPT invalida")
	ErrLecturaRPTDenegada     = errors.New("vec: consulta de categorias RPT denegada")
	ErrLecturaRPTNoDisponible = errors.New("vec: consulta de categorias RPT no disponible")
	ErrLecturaRPTNoConfiable  = errors.New("vec: respuesta de categorias RPT no confiable")
)

// EvidenciaLecturaRPT conserva el recibo de cada consumo AD3. Cada lectura
// requiere una decision vigente y deja su propia auditoria, incluso si no hay
// datos para el recurso consultado.
type EvidenciaLecturaRPT struct {
	DecisionRef         string
	EfectoRef           string
	HuellaEfectoSHA256  string
	ConsumoHuellaSHA256 string
	AuditoriaRef        string
	ConsumidaEn         time.Time
	ConsumoNuevo        bool
}

// ReferenciaPublicacionRPT identifica la version y huella exactas de una
// publicacion. Una categoria habilitada puede proceder de una version anterior
// a la ultima del catalogo.
type ReferenciaPublicacionRPT struct {
	CatalogoID   string
	Version      int
	HuellaSHA256 string
}

type PublicacionRPT struct {
	Referencia        ReferenciaPublicacionRPT
	DocumentoCanonico string
	PublicadaEn       time.Time
}

type ControlCategoriaRPT struct {
	Publicacion ReferenciaPublicacionRPT
	Revision    int64
	Estado      string
}

type CategoriaHabilitadaRPT struct {
	CategoriaID string
	Publicacion ReferenciaPublicacionRPT
	Revision    int64
	Estado      string
	Etiqueta    string
	Definicion  domain.EntradaCatalogoConfigurable
}

type ConsultaCategoriasHabilitadasRPT struct {
	CatalogoID        string
	CursorCategoriaID string
	Limite            int
}

type OrdenCategoriasHabilitadasRPT struct {
	Consulta     ConsultaCategoriasHabilitadasRPT
	Solicitud    domain.SolicitudAutorizacionLigadaV3
	Autorizacion ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ResultadoCategoriasHabilitadasRPT struct {
	Encontrado      bool
	Categorias      []CategoriaHabilitadaRPT
	Publicaciones   []PublicacionRPT
	HayMas          bool
	SiguienteCursor *string
	Evidencia       EvidenciaLecturaRPT
}

type ConsultaPublicacionCategoriaRPT struct {
	Referencia  ReferenciaPublicacionRPT
	CategoriaID string
}

type OrdenPublicacionCategoriaRPT struct {
	Consulta     ConsultaPublicacionCategoriaRPT
	Solicitud    domain.SolicitudAutorizacionLigadaV3
	Autorizacion ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// La publicacion historica no se reescribe con el control de hoy. Este queda
// aparte y puede apuntar a otra version o estar deshabilitado o retirado.
type ResultadoPublicacionCategoriaRPT struct {
	Encontrado    bool
	Publicacion   *PublicacionRPT
	Entrada       *domain.EntradaCatalogoConfigurable
	ControlActual *ControlCategoriaRPT
	Evidencia     EvidenciaLecturaRPT
}

type ConsultaUsoCategoriaRPT struct {
	Consumidor       string
	UsoRef           string
	ReservaReciboRef string
}

type OrdenUsoCategoriaRPT struct {
	Consulta     ConsultaUsoCategoriaRPT
	Solicitud    domain.SolicitudAutorizacionLigadaV3
	Autorizacion ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type UsoCategoriaRPT struct {
	Consumidor        string
	UsoRef            string
	CategoriaID       string
	Publicacion       ReferenciaPublicacionRPT
	Estado            string
	Revision          int64
	ReservaReciboRef  string
	TerminalReciboRef *string
	ReservadoEn       time.Time
	TerminalEn        *time.Time
}

type ResultadoUsoCategoriaRPT struct {
	Encontrado bool
	Uso        *UsoCategoriaRPT
	Evidencia  EvidenciaLecturaRPT
}

// LectorCategoriasRPT es la unica frontera de lectura de la autoridad comun.
// Sus implementadores consumen una autorizacion nominal nueva por consulta;
// los modulos consumidores no leen sus tablas.
type LectorCategoriasRPT interface {
	ListarCategoriasHabilitadasRPT(context.Context, OrdenCategoriasHabilitadasRPT) (ResultadoCategoriasHabilitadasRPT, error)
	LeerPublicacionCategoriaRPT(context.Context, OrdenPublicacionCategoriaRPT) (ResultadoPublicacionCategoriaRPT, error)
	ConsultarUsoCategoriaRPT(context.Context, OrdenUsoCategoriaRPT) (ResultadoUsoCategoriaRPT, error)
}
