package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var referenciaActaVistaConvoca = regexp.MustCompile(`^acta:importacion-convoca:[0-9a-f]{64}$`)
var huellaVistaConvoca = regexp.MustCompile(`^[0-9a-f]{64}$`)
var auditoriaVistaConvoca = regexp.MustCompile(`^aud_v3_[0-9a-f]{32}$`)
var categoriaVistaConvoca = regexp.MustCompile(`^categoria:rpt:[a-z0-9][a-z0-9_.-]{0,63}$`)

func ValidarPaginaVistaPreviaCargaConvoca(p ports.PaginaVistaPreviaCargaConvoca) error {
	switch p.Filtro {
	case "todas", "aceptadas", "rechazadas", "con_avisos":
	default:
		return ports.ErrVistaPreviaCargaConvocaInvalida
	}
	if p.Limite < 1 || p.Limite > 100 || p.Desplazamiento < 0 || p.Desplazamiento > MaximoFilasCargaConvoca {
		return ports.ErrVistaPreviaCargaConvocaInvalida
	}
	return nil
}

func ValidarSolicitudVistaPreviaCargaConvoca(s ports.SolicitudVistaPreviaCargaConvoca) error {
	if s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil ||
		s.ResultadoContexto.Contexto.PersonaRef == "" || s.Correlacion.Validar() != nil ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) ||
		!categoriaVistaConvoca.MatchString(s.CategoriaRef) ||
		s.NombreFichero == "" || len(s.Contenido) == 0 || ValidarPaginaVistaPreviaCargaConvoca(s.Pagina) != nil {
		return ports.ErrVistaPreviaCargaConvocaInvalida
	}
	return nil
}

func ValidarOrdenVistaPreviaCargaConvoca(o ports.OrdenVistaPreviaCargaConvoca) error {
	if !referenciaActaVistaConvoca.MatchString(o.ActaRef) || o.ActorRef == "" ||
		len(o.ActorRef) > 256 || strings.TrimSpace(o.ActorRef) != o.ActorRef ||
		!categoriaVistaConvoca.MatchString(o.CategoriaRef) ||
		!huellaVistaConvoca.MatchString(o.HuellaFicheroSHA256) ||
		len(o.ContextoRecursoCanonico) == 0 || len(o.ContextoRecursoCanonico) > 2048 {
		return ports.ErrVistaPreviaCargaConvocaInvalida
	}
	acta := sha256.Sum256([]byte(o.HuellaFicheroSHA256 + "\x1f" + o.CategoriaRef))
	if o.ActaRef != "acta:importacion-convoca:"+hex.EncodeToString(acta[:]) {
		return ports.ErrVistaPreviaCargaConvocaInvalida
	}
	return nil
}

func ValidarAcuseVistaPreviaCargaConvoca(a ports.AcuseVistaPreviaCargaConvoca) error {
	if a.DecisionRef == "" || !referenciaActaVistaConvoca.MatchString(a.ActaRef) ||
		!huellaVistaConvoca.MatchString(a.HuellaContextoSHA256) ||
		!auditoriaVistaConvoca.MatchString(a.AuditoriaRef) || a.ConsumidaEn.IsZero() ||
		!a.ConsumidaEn.Equal(a.ConsumidaEn.Truncate(time.Microsecond)) {
		return ports.ErrVistaPreviaCargaConvocaInvalida
	}
	return nil
}

func ValidarAcuseVistaPreviaCargaConvocaPara(a ports.AcuseVistaPreviaCargaConvoca,
	o ports.OrdenVistaPreviaCargaConvoca, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) error {
	if ValidarAcuseVistaPreviaCargaConvoca(a) != nil || ValidarOrdenVistaPreviaCargaConvoca(o) != nil ||
		m.ValidarEstructura() != nil {
		return ports.ErrVistaPreviaCargaConvocaInvalida
	}
	resumen := m.ResumenCapacidad()
	huella := sha256.Sum256(o.ContextoRecursoCanonico)
	if a.DecisionRef != resumen.DecisionRef() || a.ActaRef != o.ActaRef ||
		a.HuellaContextoSHA256 != hex.EncodeToString(huella[:]) ||
		a.HuellaContextoSHA256 != resumen.EfectoHuellaSHA256() ||
		resumen.EfectoRef() != o.ActaRef || resumen.Operacion() != ports.AccionConfirmarCargaConvoca ||
		resumen.AudienciaConsumo() != ports.AudienciaConfirmarCargaConvoca {
		return ports.ErrVistaPreviaCargaConvocaInvalida
	}
	return nil
}

// VistaPreviaCargaConvocaPreparada sólo expone la página que HTTP puede
// serializar en memoria. La solicitud V3 y la orden SQL permanecen privadas.
type VistaPreviaCargaConvocaPreparada struct {
	Vista         VistaPreviaCargaConvoca
	Pagina        ports.PaginaVistaPreviaCargaConvoca
	TotalFiltrado int
	recurso       dominiovec.RecursoAutorizable
	orden         ports.OrdenVistaPreviaCargaConvoca
	vinculo       dominiovec.VinculoAutenticacionActorV2
	resultado     dominiovec.ResultadoContextoActorRegistradoV2
	correlacion   dominiovec.ReferenciaCorrelacionAutorizacionV2
	motivo        dominiovec.ReferenciaEntradaCatalogo
}

type ServicioVistaPreviaCargaConvocaAutorizada struct {
	vista       *PrevisualizadorCargaConvoca
	contexto    ports.ResolutorContextoBorradorLlamamiento
	autorizador ports.AutorizadorBorradorLlamamientoV3
	consumidor  ports.ConsumidorVistaPreviaCargaConvoca
}

func NuevoServicioVistaPreviaCargaConvocaAutorizada(vista *PrevisualizadorCargaConvoca,
	contexto ports.ResolutorContextoBorradorLlamamiento,
	autorizador ports.AutorizadorBorradorLlamamientoV3,
	consumidor ports.ConsumidorVistaPreviaCargaConvoca,
) (*ServicioVistaPreviaCargaConvocaAutorizada, error) {
	if vista == nil || dependenciaLlamamientoNula(contexto) ||
		dependenciaLlamamientoNula(autorizador) || dependenciaLlamamientoNula(consumidor) {
		return nil, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	return &ServicioVistaPreviaCargaConvocaAutorizada{vista: vista, contexto: contexto,
		autorizador: autorizador, consumidor: consumidor}, nil
}

// Preparar interpreta el libro una vez, filtra y deja sólo la página en la
// respuesta. Ningún material V3 se consume hasta que HTTP haya serializado
// esa página y llame a Consumir sobre la misma preparación.
func (s *ServicioVistaPreviaCargaConvocaAutorizada) Preparar(ctx context.Context,
	q ports.SolicitudVistaPreviaCargaConvoca,
) (VistaPreviaCargaConvocaPreparada, error) {
	if s == nil || s.vista == nil || s.contexto == nil || ctx == nil || ValidarSolicitudVistaPreviaCargaConvoca(q) != nil {
		return VistaPreviaCargaConvocaPreparada{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return VistaPreviaCargaConvocaPreparada{}, err
	}
	actor := q.ResultadoContexto.Contexto
	resuelto, err := s.contexto.ResolverContextoBorradorLlamamiento(ctx, actor)
	if err != nil || resuelto.Validar() != nil {
		return VistaPreviaCargaConvocaPreparada{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	vista, err := s.vista.Previsualizar(ctx, q.NombreFichero, q.Contenido)
	if err != nil {
		return VistaPreviaCargaConvocaPreparada{}, err
	}
	huella := sha256.Sum256(q.Contenido)
	huellaFichero := hex.EncodeToString(huella[:])
	if vista.HuellaSHA256 != huellaFichero {
		return VistaPreviaCargaConvocaPreparada{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	acta := importacionapp.ReferenciaActa(huellaFichero, q.CategoriaRef)
	recurso := dominiovec.RecursoAutorizable{Referencia: acta, ModuloID: ports.ModuloCargaConvoca,
		Tipo:    ports.TipoRecursoCargaConvoca,
		Ambitos: map[string]string{"ambito_ref": resuelto.AmbitoRef, "unidad_ref": resuelto.UnidadRef},
		Atributos: map[string]string{
			"desplazamiento": strconv.Itoa(q.Pagina.Desplazamiento),
			"esquema":        ports.EsquemaVistaPreviaCargaConvocaV1,
			"fase":           "vista_previa", "filtro": q.Pagina.Filtro,
			"limite": strconv.Itoa(q.Pagina.Limite),
		},
	}
	canon, err := contextoRecursoVistaPreviaCargaConvocaCanonico(recurso)
	if err != nil {
		return VistaPreviaCargaConvocaPreparada{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	orden := ports.OrdenVistaPreviaCargaConvoca{ActaRef: acta, ActorRef: actor.Principal.ID,
		CategoriaRef: q.CategoriaRef, HuellaFicheroSHA256: huellaFichero, ContextoRecursoCanonico: canon}
	if ValidarOrdenVistaPreviaCargaConvoca(orden) != nil {
		return VistaPreviaCargaConvocaPreparada{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	preparada, err := PaginarVistaPreviaCargaConvoca(vista, q.Pagina)
	if err != nil {
		return VistaPreviaCargaConvocaPreparada{}, err
	}
	preparada.recurso, preparada.orden, preparada.vinculo = recurso, orden, q.Vinculo
	preparada.resultado, preparada.correlacion, preparada.motivo = q.ResultadoContexto, q.Correlacion, q.MotivoAutorizacion
	return preparada, nil
}

// Paginar aplica el orden ya calculado por Bolsa y conserva los contadores
// globales. El servidor sólo serializa las filas seleccionadas.
func PaginarVistaPreviaCargaConvoca(vista VistaPreviaCargaConvoca,
	pagina ports.PaginaVistaPreviaCargaConvoca,
) (VistaPreviaCargaConvocaPreparada, error) {
	if ValidarPaginaVistaPreviaCargaConvoca(pagina) != nil {
		return VistaPreviaCargaConvocaPreparada{}, ports.ErrVistaPreviaCargaConvocaInvalida
	}
	total := 0
	filas := make([]FilaVistaPreviaCargaConvoca, 0, pagina.Limite)
	for _, fila := range vista.Filas {
		if !filaCumpleFiltroVistaPreviaCargaConvoca(fila, pagina.Filtro) {
			continue
		}
		if total >= pagina.Desplazamiento && len(filas) < pagina.Limite {
			filas = append(filas, fila)
		}
		total++
	}
	vista.Filas = filas
	return VistaPreviaCargaConvocaPreparada{Vista: vista, Pagina: pagina, TotalFiltrado: total}, nil
}

func filaCumpleFiltroVistaPreviaCargaConvoca(f FilaVistaPreviaCargaConvoca, filtro string) bool {
	switch filtro {
	case "todas":
		return true
	case "aceptadas":
		return f.Estado == EstadoFilaCargaAceptada
	case "rechazadas":
		return f.Estado == EstadoFilaCargaRechazada
	case "con_avisos":
		return len(f.Avisos) > 0
	default:
		return false
	}
}

// CanonGo es la misma preimagen que V3 sella: no se calcula sobre jsonb::text.
func contextoRecursoVistaPreviaCargaConvocaCanonico(r dominiovec.RecursoAutorizable) ([]byte, error) {
	if r.Validar() != nil {
		return nil, ports.ErrVistaPreviaCargaConvocaInvalida
	}
	canon, err := json.Marshal(struct {
		Ambitos   map[string]string `json:"ambitos"`
		Atributos map[string]string `json:"atributos"`
	}{Ambitos: r.Ambitos, Atributos: r.Atributos})
	if err != nil {
		return nil, ports.ErrVistaPreviaCargaConvocaInvalida
	}
	huellaV3, err := r.HuellaContextoAutorizacionSHA256()
	huella := sha256.Sum256(canon)
	if err != nil || hex.EncodeToString(huella[:]) != huellaV3 {
		return nil, ports.ErrVistaPreviaCargaConvocaInvalida
	}
	return canon, nil
}

// Consumir emite material V3 sobre el acta y la página exactas y exige el
// acuse B93 confirmado. El error nunca entrega la página preparada.
func (s *ServicioVistaPreviaCargaConvocaAutorizada) Consumir(ctx context.Context,
	p VistaPreviaCargaConvocaPreparada,
) (ports.AcuseVistaPreviaCargaConvoca, error) {
	if s == nil || s.autorizador == nil || s.consumidor == nil || ctx == nil ||
		ValidarOrdenVistaPreviaCargaConvoca(p.orden) != nil || ValidarPaginaVistaPreviaCargaConvoca(p.Pagina) != nil || p.resultado.Validar() != nil ||
		p.vinculo.ValidarPara(p.resultado) != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, err
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: p.vinculo, ReferenciaMotivo: p.motivo,
		Accion: ports.AccionConfirmarCargaConvoca, Recurso: p.recurso,
		Finalidad: ports.FinalidadConfirmarCargaConvoca, Correlacion: p.correlacion,
	})
	if err != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, p.resultado)
	if errors.Is(err, puertosvec.ErrDenegacionExplicitaAutorizacionLigadaV3) || errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		return ports.AcuseVistaPreviaCargaConvoca{}, dominiovec.ErrAutorizacionDenegada
	}
	if err != nil || exportador == nil || decision.ValidarPara(solicitud) != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(solicitud, decision, confirmacion,
		p.resultado, p.motivo, material, ports.AudienciaConfirmarCargaConvoca) {
		return ports.AcuseVistaPreviaCargaConvoca{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	huella := sha256.Sum256(p.orden.ContextoRecursoCanonico)
	huellaV3, err := p.recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || hex.EncodeToString(huella[:]) != huellaV3 ||
		material.ResumenCapacidad().EfectoHuellaSHA256() != huellaV3 {
		return ports.AcuseVistaPreviaCargaConvoca{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	acuse, err := s.consumidor.ConsumirVistaPreviaCargaConvoca(ctx, p.orden, material)
	if errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		return ports.AcuseVistaPreviaCargaConvoca{}, err
	}
	if err != nil || ValidarAcuseVistaPreviaCargaConvocaPara(acuse, p.orden, material) != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	return acuse, nil
}
