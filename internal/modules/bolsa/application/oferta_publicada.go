package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	var plazo puertosbolsa.PlazoOferta
	var vence time.Time
	var err error
	if porBolsa, ok := s.plazos.(puertosbolsa.CalculadoraPlazoOfertaPorBolsa); ok {
		plazo, vence, err = porBolsa.PlazoDisposicionBolsa(ctx, q.BolsaRef, ahora)
	} else {
		plazo, vence, err = s.plazos.PlazoDisposicion(ctx, ahora)
	}
	if err != nil {
		if errors.Is(err, puertosbolsa.ErrPlazoOfertaNoConfigurado) {
			return puertosbolsa.OfertaPublicada{}, err
		}
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	if !vence.After(ahora) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	vence = vence.UTC().Truncate(time.Microsecond)
	materialHash := huellaMaterialPlazoOferta(q.BolsaRef, ahora, vence, plazo, q.Datos.NumeroPlazasEfectivas())
	emision, err := s.materialEmision(ctx, q.Vinculo, q.ResultadoContexto, q.BolsaRef, q.Correlacion, q.MotivoAutorizacion, materialHash)
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, err
	}
	sufijo := huellaOferta(q.BolsaRef, q.ClaveIdempotencia)
	oferta, err := s.repositorio.Publicar(ctx, puertosbolsa.ComandoPublicarOferta{
		OfertaRef: "oferta:" + sufijo, ReciboRef: "recibo:oferta:" + sufijo, BolsaRef: q.BolsaRef, ActorRef: emision.ActorRef,
		UnidadRef: emision.UnidadRef, AmbitoRef: emision.AmbitoRef,
		ClaveIdempotencia: q.ClaveIdempotencia, Datos: q.Datos, Plazo: plazo, PublicadaEn: ahora,
		VenceAntesDe: vence, Material: emision.Material,
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
	numeroDePlaza := q.NumeroDePlaza
	if numeroDePlaza == 0 {
		numeroDePlaza = 1
	}
	if ctx == nil || s == nil || q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		!referenciaOfertaValida(q.BolsaRef) || !strings.HasPrefix(q.OfertaRef, "oferta:") || !referenciaOfertaValida(q.OfertaRef) ||
		(q.ParticipacionRef != "" && !referenciaOfertaValida(q.ParticipacionRef)) || numeroDePlaza < 1 || numeroDePlaza > 100 || !claveOfertaValida(q.ClaveIdempotencia) ||
		q.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
	}
	emision, err := s.materialEmision(ctx, q.Vinculo, q.ResultadoContexto, q.BolsaRef, q.Correlacion, q.MotivoAutorizacion, "")
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, err
	}
	return s.repositorio.Resolver(ctx, puertosbolsa.ComandoResolverOferta{
		OfertaRef: q.OfertaRef, ReciboRef: "recibo:resolucion-oferta:" + huellaResolucionOferta(q.OfertaRef, numeroDePlaza, q.ClaveIdempotencia),
		BolsaRef: q.BolsaRef, ParticipacionRef: q.ParticipacionRef, ActorRef: emision.ActorRef,
		ClaveIdempotencia: q.ClaveIdempotencia, NumeroDePlaza: numeroDePlaza, UnidadRef: emision.UnidadRef, AmbitoRef: emision.AmbitoRef, Material: emision.Material,
	})
}

// ConfirmarAdjudicacionOferta consume una autorización distinta, ligada a la
// preparación exacta. SQL verifica actor distinto, plaza, orden y replay.
func (s *ServicioOfertasPublicadas) ConfirmarAdjudicacionOferta(ctx context.Context, q puertosbolsa.SolicitudConfirmarAdjudicacionOferta) (puertosbolsa.OfertaPublicada, error) {
	if ctx == nil || s == nil || q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		!referenciaOfertaValida(q.BolsaRef) || !strings.HasPrefix(q.OfertaRef, "oferta:") || !referenciaOfertaValida(q.OfertaRef) ||
		!strings.HasPrefix(q.PreparacionRef, "recibo:preparacion-oferta:") || !referenciaOfertaValida(q.PreparacionRef) ||
		q.NumeroDePlaza < 1 || q.NumeroDePlaza > 100 || !claveOfertaValida(q.ClaveIdempotencia) ||
		q.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaInvalida
	}
	confirmador, ok := s.repositorio.(puertosbolsa.RepositorioConfirmacionAdjudicacionOferta)
	if !ok {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	actor := q.ResultadoContexto.Contexto
	if (actor.Principal.AuthMethod != dominiovec.AuthMethodCertificate && actor.Principal.AuthMethod != dominiovec.AuthMethodDNIe) || actor.Principal.AuthAssurance != dominiovec.AuthAssuranceHigh {
		return puertosbolsa.OfertaPublicada{}, dominiovec.ErrAutorizacionDenegada
	}
	resuelto, err := s.contextoBolsa.ResolverContextoContactosBolsa(ctx, actor, q.BolsaRef)
	if err != nil || resuelto.Validar() != nil {
		return puertosbolsa.OfertaPublicada{}, errorDependenciaOferta(err)
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: q.PreparacionRef, ModuloID: "bolsa", Tipo: "preparacion_adjudicacion_oferta",
		Ambitos:   map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef},
		Atributos: map[string]string{"bolsa_ref": q.BolsaRef, "oferta_ref": q.OfertaRef, "numero_de_plaza": fmt.Sprint(q.NumeroDePlaza)}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion,
		Accion: puertosbolsa.AccionConfirmarAdjudicacionOferta, Recurso: recurso,
		Finalidad: puertosbolsa.FinalidadConfirmarAdjudicacionOferta, Correlacion: q.Correlacion})
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, q.ResultadoContexto)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		return puertosbolsa.OfertaPublicada{}, errorDependenciaOferta(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, q.ResultadoContexto, q.MotivoAutorizacion, material, puertosbolsa.AudienciaConfirmarAdjudicacionOferta) {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	oferta, err := confirmador.ConfirmarAdjudicacion(ctx, puertosbolsa.ComandoConfirmarAdjudicacionOferta{
		OfertaRef: q.OfertaRef, BolsaRef: q.BolsaRef, PreparacionRef: q.PreparacionRef, NumeroDePlaza: q.NumeroDePlaza,
		ActorRef: actor.PersonaRef, ClaveIdempotencia: q.ClaveIdempotencia, Material: material})
	if err != nil {
		return puertosbolsa.OfertaPublicada{}, err
	}
	if oferta.OfertaRef != q.OfertaRef || oferta.BolsaRef != q.BolsaRef || len(oferta.Adjudicaciones) < q.NumeroDePlaza ||
		oferta.Adjudicaciones[q.NumeroDePlaza-1].NumeroDePlaza != q.NumeroDePlaza || oferta.Adjudicaciones[q.NumeroDePlaza-1].ReciboRef == "" {
		return puertosbolsa.OfertaPublicada{}, puertosbolsa.ErrOfertaNoDisponible
	}
	return oferta, nil
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

// La variante B57 añade el número de plazas al final del material B54; así la
// autorización de publicación no puede ampliarse a más plazas desde SQL.
func huellaMaterialPlazoOferta(bolsa string, publicada, vence time.Time, p puertosbolsa.PlazoOferta, numeroPlazas ...int) string {
	campos := []string{bolsa, publicada.UTC().Format(formatoInstanteMaterialOferta),
		vence.UTC().Format(formatoInstanteMaterialOferta), p.ReglaRef, p.HuellaCatalogo,
		p.Unidad, fmt.Sprint(p.Cantidad), p.Computo, p.MunicipioSede, p.UltimoDia,
		fmt.Sprint(p.PoliticaVersion), strings.Join(p.Calendarios, "\x1e")}
	if p.Unidad == "horas_naturales" {
		campos = append(campos, p.AperturaEn, p.VenceEn)
	}
	if len(numeroPlazas) == 1 {
		campos = append(campos, fmt.Sprint(numeroPlazas[0]))
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

func huellaResolucionOferta(oferta string, numeroDePlaza int, clave string) string {
	h := sha256.Sum256([]byte(oferta + "\x1f" + fmt.Sprint(numeroDePlaza) + "\x1f" + clave))
	return hex.EncodeToString(h[:])
}

func referenciaOfertaValida(ref string) bool {
	return ref != "" && len(ref) <= 256 && strings.TrimSpace(ref) == ref
}

func claveOfertaValida(clave string) bool {
	return len(clave) >= 8 && len(clave) <= 256 && strings.TrimSpace(clave) == clave
}
