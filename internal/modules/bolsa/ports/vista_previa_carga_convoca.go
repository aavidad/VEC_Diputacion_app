package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const EsquemaVistaPreviaCargaConvocaV1 = "vec.bolsa.rrhh.carga_convoca.vista_previa.v1"

var ErrVistaPreviaCargaConvocaInvalida = errors.New("bolsa: vista previa CONVOCA invalida")
var ErrVistaPreviaCargaConvocaNoDisponible = errors.New("bolsa: vista previa CONVOCA no disponible")

var referenciaActaVistaConvoca = regexp.MustCompile(`^acta:importacion-convoca:[0-9a-f]{64}$`)
var huellaVistaConvoca = regexp.MustCompile(`^[0-9a-f]{64}$`)
var auditoriaVistaConvoca = regexp.MustCompile(`^aud_v3_[0-9a-f]{32}$`)
var categoriaVistaConvoca = regexp.MustCompile(`^categoria:rpt:[a-z0-9][a-z0-9_.-]{0,63}$`)

type PaginaVistaPreviaCargaConvoca struct {
	Filtro         string
	Limite         int
	Desplazamiento int
}

func (p PaginaVistaPreviaCargaConvoca) Validar() error {
	switch p.Filtro {
	case "todas", "aceptadas", "rechazadas", "con_avisos":
	default:
		return ErrVistaPreviaCargaConvocaInvalida
	}
	if p.Limite < 1 || p.Limite > 100 || p.Desplazamiento < 0 || p.Desplazamiento > 20000 {
		return ErrVistaPreviaCargaConvocaInvalida
	}
	return nil
}

// El navegador sólo aporta fichero, clave de categoría y página; identidad,
// motivo y correlación proceden de la frontera corporativa verificada.
type SolicitudVistaPreviaCargaConvoca struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
	CategoriaRef       string
	NombreFichero      string
	Contenido          []byte
	Pagina             PaginaVistaPreviaCargaConvoca
}

func (s SolicitudVistaPreviaCargaConvoca) Validar() error {
	if s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil ||
		s.ResultadoContexto.Contexto.PersonaRef == "" || s.Correlacion.Validar() != nil ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) ||
		!categoriaVistaConvoca.MatchString(s.CategoriaRef) ||
		s.NombreFichero == "" || len(s.Contenido) == 0 || s.Pagina.Validar() != nil {
		return ErrVistaPreviaCargaConvocaInvalida
	}
	return nil
}

// La orden sólo contiene referencias y el contexto de recurso que V3 selló.
// Ninguna fila del Excel cruza la frontera PostgreSQL de la vista previa.
type OrdenVistaPreviaCargaConvoca struct {
	ActaRef                 string
	ActorRef                string
	CategoriaRef            string
	HuellaFicheroSHA256     string
	ContextoRecursoCanonico []byte
}

func (o OrdenVistaPreviaCargaConvoca) Validar() error {
	if !referenciaActaVistaConvoca.MatchString(o.ActaRef) || o.ActorRef == "" ||
		len(o.ActorRef) > 256 || strings.TrimSpace(o.ActorRef) != o.ActorRef ||
		!categoriaVistaConvoca.MatchString(o.CategoriaRef) ||
		!huellaVistaConvoca.MatchString(o.HuellaFicheroSHA256) ||
		len(o.ContextoRecursoCanonico) == 0 || len(o.ContextoRecursoCanonico) > 2048 {
		return ErrVistaPreviaCargaConvocaInvalida
	}
	acta := sha256.Sum256([]byte(o.HuellaFicheroSHA256 + "\x1f" + o.CategoriaRef))
	if o.ActaRef != "acta:importacion-convoca:"+hex.EncodeToString(acta[:]) {
		return ErrVistaPreviaCargaConvocaInvalida
	}
	return nil
}

type AcuseVistaPreviaCargaConvoca struct {
	DecisionRef          string
	ActaRef              string
	HuellaContextoSHA256 string
	AuditoriaRef         string
	ConsumidaEn          time.Time
}

func (a AcuseVistaPreviaCargaConvoca) Validar() error {
	if a.DecisionRef == "" || !referenciaActaVistaConvoca.MatchString(a.ActaRef) ||
		!huellaVistaConvoca.MatchString(a.HuellaContextoSHA256) ||
		!auditoriaVistaConvoca.MatchString(a.AuditoriaRef) || a.ConsumidaEn.IsZero() ||
		!a.ConsumidaEn.Equal(a.ConsumidaEn.Truncate(time.Microsecond)) {
		return ErrVistaPreviaCargaConvocaInvalida
	}
	return nil
}

func (a AcuseVistaPreviaCargaConvoca) ValidarPara(o OrdenVistaPreviaCargaConvoca,
	m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) error {
	if a.Validar() != nil || o.Validar() != nil || m.ValidarEstructura() != nil {
		return ErrVistaPreviaCargaConvocaInvalida
	}
	resumen := m.ResumenCapacidad()
	huella := sha256.Sum256(o.ContextoRecursoCanonico)
	if a.DecisionRef == "" || a.DecisionRef != resumen.DecisionRef() || a.ActaRef != o.ActaRef ||
		a.HuellaContextoSHA256 != hex.EncodeToString(huella[:]) ||
		a.HuellaContextoSHA256 != resumen.EfectoHuellaSHA256() ||
		resumen.EfectoRef() != o.ActaRef || resumen.Operacion() != AccionConfirmarCargaConvoca ||
		resumen.AudienciaConsumo() != AudienciaConfirmarCargaConvoca {
		return ErrVistaPreviaCargaConvocaInvalida
	}
	return nil
}

// El adaptador PostgreSQL consume la decisión AD218 y devuelve el acuse
// confirmado en una transacción SERIALIZABLE; nunca recibe filas ni fichero.
type ConsumidorVistaPreviaCargaConvoca interface {
	ConsumirVistaPreviaCargaConvoca(context.Context, OrdenVistaPreviaCargaConvoca,
		puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (AcuseVistaPreviaCargaConvoca, error)
}
