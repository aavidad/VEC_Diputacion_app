package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	maximoOfertasConsulta = 100
	// maximaSecuenciaPlaza acota la historia de una plaza (la migración
	// B58 admite secuencias 1..10000).
	maximaSecuenciaPlaza = 10000
)

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
		!dominiobolsa.NumeroPlazasValido(q.NumeroPlazas) || q.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	if q.Notificacion.ValidarPara(ahora) != nil {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
	}
	q.Notificacion.NotificadaEn = q.Notificacion.NotificadaEn.UTC()
	var plazo puertosbolsa.PlazoOferta
	var vence time.Time
	var err error
	if porBolsa, ok := s.plazos.(puertosbolsa.CalculadoraPlazoOfertaPorBolsa); ok {
		plazo, vence, err = porBolsa.PlazoDisposicionBolsa(ctx, q.BolsaRef, q.Notificacion.NotificadaEn)
	} else {
		plazo, vence, err = s.plazos.PlazoDisposicion(ctx, q.Notificacion.NotificadaEn)
	}
	if err != nil {
		if errors.Is(err, puertosbolsa.ErrPlazoOfertaNoConfigurado) {
			return puertosbolsa.OfertaPublicada{}, err
		}
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	if !vence.After(q.Notificacion.NotificadaEn) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	vence = vence.UTC().Truncate(time.Microsecond)
	notificacion := q.Notificacion
	plazo.Notificacion = &notificacion
	materialHash := huellaMaterialPlazoOfertaConPlazas(q.BolsaRef, ahora, vence, plazo, q.NumeroPlazas)
	emision, err := s.materialEmision(ctx, q.Vinculo, q.ResultadoContexto, q.BolsaRef, q.Correlacion, q.MotivoAutorizacion, materialHash)
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, err
	}
	sufijo := huellaOferta(q.BolsaRef, q.ClaveIdempotencia)
	oferta, err := s.repositorio.Publicar(ctx, puertosbolsa.ComandoPublicarOferta{
		OfertaRef: "oferta:" + sufijo, ReciboRef: "recibo:oferta:" + sufijo, BolsaRef: q.BolsaRef, ActorRef: emision.ActorRef,
		UnidadRef: emision.UnidadRef, AmbitoRef: emision.AmbitoRef,
		ClaveIdempotencia: q.ClaveIdempotencia, Datos: q.Datos, NumeroPlazas: q.NumeroPlazas, Plazo: plazo, PublicadaEn: ahora,
		VenceAntesDe: vence, Material: emision.Material,
	})
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, err
	}
	if oferta.BolsaRef != q.BolsaRef || oferta.Datos != q.Datos || oferta.NumeroPlazas != q.NumeroPlazas ||
		oferta.Plazo.Notificacion == nil || *oferta.Plazo.Notificacion != q.Notificacion {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaConflicto
	}
	return oferta, nil
}

// ResolverOferta registra un acto sobre una plaza. El almacén recalcula en ese
// mismo instante la propuesta y el estado de la plaza; si difieren de lo que
// RRHH vio (persona, tipo o secuencia), lo rechaza.
func (s *ServicioOfertasPublicadas) ResolverOferta(ctx context.Context, q puertosbolsa.SolicitudResolverOferta) (puertosbolsa.OfertaPublicada, error) {
	if ctx == nil || s == nil || q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		!referenciaOfertaValida(q.BolsaRef) || !strings.HasPrefix(q.OfertaRef, "oferta:") || !referenciaOfertaValida(q.OfertaRef) ||
		!dominiobolsa.NumeroPlazasValido(q.NumeroDePlaza) || !dominiobolsa.ActoPlazaValido(q.Tipo, q.ParticipacionRef != "") ||
		q.SecuenciaEsperada < 0 || q.SecuenciaEsperada >= maximaSecuenciaPlaza ||
		(q.ParticipacionRef != "" && !referenciaOfertaValida(q.ParticipacionRef)) || !claveOfertaValida(q.ClaveIdempotencia) ||
		q.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
	}
	emision, err := s.materialEmision(ctx, q.Vinculo, q.ResultadoContexto, q.BolsaRef, q.Correlacion, q.MotivoAutorizacion, "")
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, err
	}
	return s.repositorio.Resolver(ctx, puertosbolsa.ComandoResolverOferta{
		OfertaRef: q.OfertaRef, ReciboRef: "recibo:plaza-oferta:" + huellaOferta(q.OfertaRef, q.ClaveIdempotencia),
		BolsaRef: q.BolsaRef, NumeroDePlaza: q.NumeroDePlaza, Tipo: q.Tipo, SecuenciaEsperada: q.SecuenciaEsperada,
		ParticipacionRef: q.ParticipacionRef, ActorRef: emision.ActorRef,
		ClaveIdempotencia: q.ClaveIdempotencia, Material: emision.Material,
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

type emisionOfertaAtestada struct {
	Material                       puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	ActorRef, UnidadRef, AmbitoRef string
}

func (s *ServicioOfertasPublicadas) materialEmision(ctx context.Context, vinculo dominiovec.VinculoAutenticacionActorV2, resultado dominiovec.ResultadoContextoActorRegistradoV2, bolsa string, correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2, motivo dominiovec.ReferenciaEntradaCatalogo, materialHash string) (emisionOfertaAtestada, error) {
	actor := resultado.Contexto
	resuelto, err := s.contextoBolsa.ResolverContextoContactosBolsa(ctx, actor, bolsa)
	if err != nil || resuelto.Validar() != nil {
		return emisionOfertaAtestada{}, errorDependenciaOferta(err)
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: bolsa, ModuloID: puertosbolsa.ModuloBorradorLlamamiento, Tipo: puertosbolsa.TipoRecursoEmision, Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	if materialHash != "" {
		recurso.Atributos = map[string]string{"material_sha256": materialHash}
	}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo, Accion: puertosbolsa.AccionEmitirLlamamiento, Recurso: recurso, Finalidad: puertosbolsa.FinalidadEmitirLlamamiento, Correlacion: correlacion})
	if err != nil {
		return emisionOfertaAtestada{}, dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, resultado)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		return emisionOfertaAtestada{}, errorDependenciaOferta(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, resultado, motivo, material, puertosbolsa.AudienciaEmitirLlamamiento) {
		return emisionOfertaAtestada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	return emisionOfertaAtestada{Material: material, ActorRef: actor.PersonaRef, UnidadRef: resuelto.UnidadRef, AmbitoRef: resuelto.AmbitoRef}, nil
}

const formatoInstanteMaterialOferta = "2006-01-02T15:04:05.000000Z"

// Esta preimagen no incluye textos personales y tiene los mismos campos y
// separadores que publicar_oferta_v2 en Bolsa B47.
func huellaMaterialPlazoOferta(bolsa string, publicada, vence time.Time, p puertosbolsa.PlazoOferta) string {
	h := sha256.Sum256([]byte(strings.Join(camposMaterialPlazoOferta(bolsa, publicada, vence, p), "\x1f")))
	return hex.EncodeToString(h[:])
}

func camposMaterialPlazoOferta(bolsa string, publicada, vence time.Time, p puertosbolsa.PlazoOferta) []string {
	campos := []string{bolsa, publicada.UTC().Format(formatoInstanteMaterialOferta),
		vence.UTC().Format(formatoInstanteMaterialOferta), p.ReglaRef, p.HuellaCatalogo,
		p.Unidad, fmt.Sprint(p.Cantidad), p.Computo, p.MunicipioSede, p.UltimoDia,
		fmt.Sprint(p.PoliticaVersion), strings.Join(p.Calendarios, "\x1e")}
	if p.Unidad == "horas_naturales" {
		campos = append(campos, p.AperturaEn, p.VenceEn)
	}
	return campos
}

// huellaMaterialPlazoOfertaConPlazas añade al final el número de plazas; es la
// preimagen de publicar_oferta_v3 en Bolsa B58, de modo que la autorización de
// publicación queda ligada a cuántas plazas se ofrecen.
func huellaMaterialPlazoOfertaConPlazas(bolsa string, publicada, vence time.Time, p puertosbolsa.PlazoOferta, numeroPlazas int) string {
	campos := camposMaterialPlazoOferta(bolsa, publicada, vence, p)
	campos = append(campos, fmt.Sprint(numeroPlazas))
	if n := p.Notificacion; n != nil {
		campos = append(campos, n.NotificadaEn.UTC().Format(formatoInstanteMaterialOferta), n.ReferenciaCorreo, n.HuellaCorreoSHA256, n.Fuente)
	}
	h := sha256.Sum256([]byte(strings.Join(campos, "\x1f")))
	return hex.EncodeToString(h[:])
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
