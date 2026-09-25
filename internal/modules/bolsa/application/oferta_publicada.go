package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const maximoOfertasConsulta = 100

// ServicioOfertasPublicadas coordina la publicación de ofertas (art. 8.1 del
// Reglamento), su consulta por RRHH y la confirmación de la propuesta de
// adjudicación. El plazo lo fija la regla b10 del catálogo; la propuesta y
// todas las guardas de estado viven en el almacén de Bolsa.
type ServicioOfertasPublicadas struct {
	contextoBolsa puertosbolsa.ResolutorContextoContactoParticipacion
	autorizador   puertosbolsa.AutorizadorSituacionParticipacionV3
	repositorio   puertosbolsa.RepositorioOfertasPublicadas
	plazos        puertosbolsa.CalculadoraPlazoOferta
	reloj         func() time.Time
}

func NuevoServicioOfertasPublicadas(cb puertosbolsa.ResolutorContextoContactoParticipacion, a puertosbolsa.AutorizadorSituacionParticipacionV3, r puertosbolsa.RepositorioOfertasPublicadas, p puertosbolsa.CalculadoraPlazoOferta, reloj func() time.Time) (*ServicioOfertasPublicadas, error) {
	if cb == nil || a == nil || r == nil || p == nil || reloj == nil {
		return nil, puertosbolsa.ErrOfertaNoDisponible
	}
	return &ServicioOfertasPublicadas{cb, a, r, p, reloj}, nil
}

// PublicarOferta calcula el plazo con la regla vigente y reserva la oferta
// consumiendo una autorización de emisión de llamamiento sobre la bolsa.
func (s *ServicioOfertasPublicadas) PublicarOferta(ctx context.Context, q puertosbolsa.SolicitudPublicarOferta) (puertosbolsa.OfertaPublicada, error) {
	if ctx == nil || s == nil || q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		!referenciaOfertaValida(q.BolsaRef) || !claveOfertaValida(q.ClaveIdempotencia) || q.Datos.Validar() != nil ||
		q.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
	}
	material, actor, err := s.materialEmision(ctx, q.Vinculo, q.ResultadoContexto, q.BolsaRef, q.Correlacion, q.MotivoAutorizacion)
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, err
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	plazo, vence, err := s.plazos.PlazoDisposicion(ctx, ahora)
	if err != nil {
		if errors.Is(err, puertosbolsa.ErrPlazoOfertaNoConfigurado) {
			return puertosbolsa.OfertaPublicada{}, err
		}
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	if !vence.After(ahora) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	sufijo := huellaOferta(q.BolsaRef, q.ClaveIdempotencia)
	oferta, err := s.repositorio.Publicar(ctx, puertosbolsa.ComandoPublicarOferta{
		OfertaRef: "oferta:" + sufijo, ReciboRef: "recibo:oferta:" + sufijo, BolsaRef: q.BolsaRef, ActorRef: actor,
		ClaveIdempotencia: q.ClaveIdempotencia, Datos: q.Datos, Plazo: plazo, PublicadaEn: ahora,
		VenceAntesDe: vence.UTC().Truncate(time.Microsecond), Material: material,
	})
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, err
	}
	if oferta.BolsaRef != q.BolsaRef || oferta.Datos != q.Datos {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaConflicto
	}
	return oferta, nil
}

// ResolverOferta confirma la propuesta que el almacén recalcula en ese mismo
// instante; si difiere de la que RRHH vio, el almacén la rechaza.
func (s *ServicioOfertasPublicadas) ResolverOferta(ctx context.Context, q puertosbolsa.SolicitudResolverOferta) (puertosbolsa.OfertaPublicada, error) {
	if ctx == nil || s == nil || q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		!referenciaOfertaValida(q.BolsaRef) || !strings.HasPrefix(q.OfertaRef, "oferta:") || !referenciaOfertaValida(q.OfertaRef) ||
		(q.ParticipacionRef != "" && !referenciaOfertaValida(q.ParticipacionRef)) || !claveOfertaValida(q.ClaveIdempotencia) ||
		q.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
	}
	material, actor, err := s.materialEmision(ctx, q.Vinculo, q.ResultadoContexto, q.BolsaRef, q.Correlacion, q.MotivoAutorizacion)
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, err
	}
	return s.repositorio.Resolver(ctx, puertosbolsa.ComandoResolverOferta{
		OfertaRef: q.OfertaRef, ReciboRef: "recibo:resolucion-oferta:" + huellaOferta(q.OfertaRef, q.ClaveIdempotencia),
		BolsaRef: q.BolsaRef, ParticipacionRef: q.ParticipacionRef, ActorRef: actor,
		ClaveIdempotencia: q.ClaveIdempotencia, Material: material,
	})
}

// ConsultarOfertas devuelve las ofertas de una bolsa admitida para la sesión
// de RRHH, con disposiciones, propuesta y resolución en el instante actual.
func (s *ServicioOfertasPublicadas) ConsultarOfertas(ctx context.Context, q puertosbolsa.SolicitudConsultarOfertas) ([]puertosbolsa.OfertaPublicada, error) {
	if ctx == nil || s == nil || q.ContextoActor.PersonaRef == "" || !referenciaOfertaValida(q.BolsaRef) || q.Limite < 1 || q.Limite > maximoOfertasConsulta {
		return nil, puertosbolsa.ErrOfertaInvalida
	}
	resuelto, err := s.contextoBolsa.ResolverContextoContactosBolsa(ctx, q.ContextoActor, q.BolsaRef)
	if err != nil || resuelto.Validar() != nil {
		return nil, errorDependenciaOferta(err)
	}
	return s.repositorio.Listar(ctx, q.BolsaRef, s.reloj().UTC().Truncate(time.Microsecond), q.Limite)
}

func (s *ServicioOfertasPublicadas) materialEmision(ctx context.Context, vinculo dominiovec.VinculoAutenticacionActorV2, resultado dominiovec.ResultadoContextoActorRegistradoV2, bolsa string, correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2, motivo dominiovec.ReferenciaEntradaCatalogo) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, string, error) {
	actor := resultado.Contexto
	resuelto, err := s.contextoBolsa.ResolverContextoContactosBolsa(ctx, actor, bolsa)
	if err != nil || resuelto.Validar() != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, "", errorDependenciaOferta(err)
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: bolsa, ModuloID: puertosbolsa.ModuloBorradorLlamamiento, Tipo: puertosbolsa.TipoRecursoEmision, Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo, Accion: puertosbolsa.AccionEmitirLlamamiento, Recurso: recurso, Finalidad: puertosbolsa.FinalidadEmitirLlamamiento, Correlacion: correlacion})
	if err != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, "", dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, resultado)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, "", errorDependenciaOferta(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, resultado, motivo, material, puertosbolsa.AudienciaEmitirLlamamiento) {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, "", puertosbolsa.ErrOfertaNoDisponible
	}
	return material, actor.PersonaRef, nil
}

func errorDependenciaOferta(err error) error {
	if errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) {
		return err
	}
	return puertosbolsa.ErrOfertaNoDisponible
}

func huellaOferta(base, clave string) string {
	h := sha256.Sum256([]byte(base + "\x1f" + clave))
	return hex.EncodeToString(h[:])
}

func referenciaOfertaValida(ref string) bool {
	return ref != "" && len(ref) <= 256 && strings.TrimSpace(ref) == ref
}

func claveOfertaValida(clave string) bool {
	return len(clave) >= 8 && len(clave) <= 256 && strings.TrimSpace(clave) == clave
}
