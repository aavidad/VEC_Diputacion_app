package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// ServicioOfertaSAE prepara y registra actuaciones de selección manual. La
// autoridad V3 es la existente; el repositorio debe consumirla en el COMMIT.
type ServicioOfertaSAE struct {
	ambitos     puertosbolsa.ResolutorAmbitoOfertaSAE
	catalogos   puertosbolsa.LectorCatalogoOfertaSAE
	autorizador puertosbolsa.AutorizadorSituacionParticipacionV3
	repositorio puertosbolsa.RepositorioOfertaSAE
	personas    puertosbolsa.AcreditadorPersonaOfertaSAE
	reloj       func() time.Time
}

// personas puede quedar nil mientras no exista un adaptador a la autoridad
// canónica: preparación, registro y desierta siguen; conciliación y resolución
// devuelven un pendiente nominal antes de cualquier efecto.
func NuevoServicioOfertaSAE(a puertosbolsa.ResolutorAmbitoOfertaSAE, c puertosbolsa.LectorCatalogoOfertaSAE, v3 puertosbolsa.AutorizadorSituacionParticipacionV3, r puertosbolsa.RepositorioOfertaSAE, personas puertosbolsa.AcreditadorPersonaOfertaSAE, reloj func() time.Time) (*ServicioOfertaSAE, error) {
	if nulo(a) || nulo(c) || nulo(v3) || nulo(r) || reloj == nil {
		return nil, puertosbolsa.ErrOfertaSAENoDisponible
	}
	return &ServicioOfertaSAE{ambitos: a, catalogos: c, autorizador: v3, repositorio: r, personas: personas, reloj: reloj}, nil
}

func (s *ServicioOfertaSAE) Preparar(ctx context.Context, q puertosbolsa.SolicitudPrepararOfertaSAE) (puertosbolsa.ReciboOfertaSAE, error) {
	if !s.solicitudValida(ctx, q.Vinculo, q.ResultadoContexto, q.Correlacion, q.MotivoAutorizacion) || !claveSAEValida(q.ClaveIdempotencia) {
		return puertosbolsa.ReciboOfertaSAE{}, dominiobolsa.ErrOfertaSAEInvalida
	}
	catalogo, err := s.catalogos.CatalogoOfertaSAEVigente(ctx)
	if err != nil || catalogo.Validar() != nil {
		return puertosbolsa.ReciboOfertaSAE{}, puertosbolsa.ErrOfertaSAENoDisponible
	}
	if q.Datos.Validar(catalogo) != nil {
		return puertosbolsa.ReciboOfertaSAE{}, dominiobolsa.ErrOfertaSAEInvalida
	}
	actor := q.ResultadoContexto.Contexto
	ambito, err := s.ambitos.ResolverAmbitoOfertaSAE(ctx, actor)
	if err != nil || !ambito.Validar() {
		return puertosbolsa.ReciboOfertaSAE{}, dependenciaSAE(err)
	}
	sufijo := sufijoSAE(actor.PersonaRef, q.ClaveIdempotencia)
	ref := "oferta-sae:" + sufijo
	oferta, err := dominiobolsa.NuevaOfertaSAE(ref, q.Datos, catalogo)
	if err != nil {
		return puertosbolsa.ReciboOfertaSAE{}, err
	}
	materialHash, err := hashSAE(q.Datos)
	if err != nil {
		return puertosbolsa.ReciboOfertaSAE{}, puertosbolsa.ErrOfertaSAENoDisponible
	}
	autorizacion, err := s.autorizar(ctx, q.Vinculo, q.ResultadoContexto, q.Correlacion, q.MotivoAutorizacion,
		puertosbolsa.AccionPrepararOfertaSAE, puertosbolsa.AudienciaPrepararOfertaSAE, ref, ambito, materialHash)
	if err != nil {
		return puertosbolsa.ReciboOfertaSAE{}, err
	}
	recibo := "recibo:oferta-sae:" + sufijo
	resultado, err := s.repositorio.Preparar(ctx, puertosbolsa.OrdenPrepararOfertaSAE{Oferta: oferta, ActorRef: actor.PersonaRef,
		ClaveIdempotencia: q.ClaveIdempotencia, ReciboRef: recibo, Ambito: ambito, Autorizacion: autorizacion})
	if err != nil {
		return puertosbolsa.ReciboOfertaSAE{}, err
	}
	if resultado.ReciboRef != recibo || resultado.Oferta.Referencia != ref || resultado.Oferta.Datos != q.Datos {
		return puertosbolsa.ReciboOfertaSAE{}, puertosbolsa.ErrOfertaSAENoDisponible
	}
	return resultado, nil
}

func (s *ServicioOfertaSAE) Actuar(ctx context.Context, q puertosbolsa.SolicitudActuarOfertaSAE) (puertosbolsa.ReciboOfertaSAE, error) {
	if !s.solicitudValida(ctx, q.Vinculo, q.ResultadoContexto, q.Correlacion, q.MotivoAutorizacion) ||
		!refOfertaSAEValida(q.OfertaRef) || !claveSAEValida(q.Cambio.Clave) || q.Cambio.VersionEsperada < 1 ||
		!accionSAEValida(q.Cambio.Accion) || q.Cambio.Acreditacion != nil ||
		(q.Cambio.Accion == dominiobolsa.AccionSAERegistrarCandidato &&
			(q.Cambio.Candidato == nil || q.Cambio.Candidato.PersonaRef != "" || q.Cambio.Candidato.Acreditacion != nil || q.Cambio.Candidato.EstadoConciliacion != "")) {
		return puertosbolsa.ReciboOfertaSAE{}, dominiobolsa.ErrOfertaSAEInvalida
	}
	cambio := q.Cambio
	if cambio.Accion == dominiobolsa.AccionSAEConciliarPersona || cambio.Accion == dominiobolsa.AccionSAEResolver {
		if nulo(s.personas) {
			return puertosbolsa.ReciboOfertaSAE{}, dominiobolsa.ErrOfertaSAEConciliacionPendiente
		}
		acreditacion, err := s.acreditarCandidato(ctx, q)
		if err != nil {
			return puertosbolsa.ReciboOfertaSAE{}, err
		}
		cambio.Acreditacion = &acreditacion
	}
	actor := q.ResultadoContexto.Contexto
	ambito, err := s.ambitos.ResolverAmbitoOfertaSAE(ctx, actor)
	if err != nil || !ambito.Validar() {
		return puertosbolsa.ReciboOfertaSAE{}, dependenciaSAE(err)
	}
	cambio.ActorRef = actor.PersonaRef
	cambio.ReciboRef = "recibo:oferta-sae:" + sufijoSAE(q.OfertaRef, cambio.Clave)
	cambio.Instante = s.reloj().UTC().Truncate(time.Microsecond)
	// El instante de recepción no forma parte de la huella semántica: repetir
	// la misma petición tras reiniciar debe conservar el mismo material.
	materialHash, err := hashSAE(struct {
		OfertaRef, Accion, Clave, NumeroSAE, FechaEnvio, CandidatoElegidoRef string
		VersionEsperada                                                      int64
		Candidato                                                            *dominiobolsa.CandidatoOfertaSAE
		Valoracion                                                           *dominiobolsa.ValoracionOfertaSAE
		Acreditacion                                                         *dominiobolsa.AcreditacionPersonaSAE
	}{q.OfertaRef, cambio.Accion, cambio.Clave, cambio.NumeroSAE, cambio.FechaEnvio, cambio.CandidatoElegidoRef,
		cambio.VersionEsperada, cambio.Candidato, cambio.Valoracion, cambio.Acreditacion})
	if err != nil {
		return puertosbolsa.ReciboOfertaSAE{}, puertosbolsa.ErrOfertaSAENoDisponible
	}
	autorizacion, err := s.autorizar(ctx, q.Vinculo, q.ResultadoContexto, q.Correlacion, q.MotivoAutorizacion,
		"bolsa.oferta_sae."+cambio.Accion, puertosbolsa.AudienciaActuarOfertaSAE, q.OfertaRef, ambito, materialHash)
	if err != nil {
		return puertosbolsa.ReciboOfertaSAE{}, err
	}
	resultado, err := s.repositorio.Actuar(ctx, puertosbolsa.OrdenActuarOfertaSAE{OfertaRef: q.OfertaRef, Cambio: cambio,
		Ambito: ambito, Autorizacion: autorizacion})
	if err != nil {
		return puertosbolsa.ReciboOfertaSAE{}, err
	}
	if resultado.ReciboRef != cambio.ReciboRef || resultado.Oferta.Referencia != q.OfertaRef {
		return puertosbolsa.ReciboOfertaSAE{}, puertosbolsa.ErrOfertaSAENoDisponible
	}
	return resultado, nil
}

func (s *ServicioOfertaSAE) acreditarCandidato(ctx context.Context, q puertosbolsa.SolicitudActuarOfertaSAE) (dominiobolsa.AcreditacionPersonaSAE, error) {
	if !strings.HasPrefix(q.Cambio.CandidatoElegidoRef, "candidato:") || len(q.Cambio.CandidatoElegidoRef) > 256 {
		return dominiobolsa.AcreditacionPersonaSAE{}, dominiobolsa.ErrOfertaSAEInvalida
	}
	oferta, err := s.Consultar(ctx, puertosbolsa.SolicitudConsultarOfertaSAE{Vinculo: q.Vinculo,
		ResultadoContexto: q.ResultadoContexto, OfertaRef: q.OfertaRef, Correlacion: q.Correlacion, MotivoAutorizacion: q.MotivoAutorizacion})
	if err != nil {
		return dominiobolsa.AcreditacionPersonaSAE{}, err
	}
	var candidato *dominiobolsa.CandidatoOfertaSAE
	for i := range oferta.Candidatos {
		if oferta.Candidatos[i].Referencia == q.Cambio.CandidatoElegidoRef {
			candidato = &oferta.Candidatos[i]
			break
		}
	}
	if candidato == nil {
		return dominiobolsa.AcreditacionPersonaSAE{}, dominiobolsa.ErrOfertaSAEInvalida
	}
	if q.Cambio.Accion == dominiobolsa.AccionSAEResolver && candidato.EstadoConciliacion != dominiobolsa.ConciliacionSAEAcreditada {
		return dominiobolsa.AcreditacionPersonaSAE{}, dominiobolsa.ErrOfertaSAEConciliacionPendiente
	}
	acreditacion, err := s.personas.AcreditarVinculoPersonaSAE(ctx, q.ResultadoContexto.Contexto, q.OfertaRef, *candidato)
	if err != nil {
		if errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
			return dominiobolsa.AcreditacionPersonaSAE{}, err
		}
		return dominiobolsa.AcreditacionPersonaSAE{}, dominiobolsa.ErrOfertaSAEConciliacionPendiente
	}
	if !acreditacion.VigenteEn(s.reloj().UTC().Truncate(time.Microsecond)) ||
		(q.Cambio.Accion == dominiobolsa.AccionSAEResolver && candidato.PersonaRef != acreditacion.PersonaRef) {
		return dominiobolsa.AcreditacionPersonaSAE{}, dominiobolsa.ErrOfertaSAEConciliacionPendiente
	}
	return acreditacion, nil
}

func (s *ServicioOfertaSAE) Consultar(ctx context.Context, q puertosbolsa.SolicitudConsultarOfertaSAE) (dominiobolsa.OfertaSAE, error) {
	if !s.solicitudValida(ctx, q.Vinculo, q.ResultadoContexto, q.Correlacion, q.MotivoAutorizacion) || !refOfertaSAEValida(q.OfertaRef) {
		return dominiobolsa.OfertaSAE{}, dominiobolsa.ErrOfertaSAEInvalida
	}
	ambito, err := s.ambitos.ResolverAmbitoOfertaSAE(ctx, q.ResultadoContexto.Contexto)
	if err != nil || !ambito.Validar() {
		return dominiobolsa.OfertaSAE{}, dependenciaSAE(err)
	}
	autorizacion, err := s.autorizar(ctx, q.Vinculo, q.ResultadoContexto, q.Correlacion, q.MotivoAutorizacion,
		puertosbolsa.AccionConsultarOfertaSAE, puertosbolsa.AudienciaConsultarOfertaSAE, q.OfertaRef, ambito, "")
	if err != nil {
		return dominiobolsa.OfertaSAE{}, err
	}
	oferta, err := s.repositorio.Consultar(ctx, puertosbolsa.OrdenConsultarOfertaSAE{OfertaRef: q.OfertaRef, Ambito: ambito, Autorizacion: autorizacion})
	if err != nil {
		return dominiobolsa.OfertaSAE{}, err
	}
	if oferta.Referencia != q.OfertaRef {
		return dominiobolsa.OfertaSAE{}, puertosbolsa.ErrOfertaSAENoDisponible
	}
	return oferta, nil
}

func (s *ServicioOfertaSAE) solicitudValida(ctx context.Context, v dominiovec.VinculoAutenticacionActorV2, r dominiovec.ResultadoContextoActorRegistradoV2, c dominiovec.ReferenciaCorrelacionAutorizacionV2, m dominiovec.ReferenciaEntradaCatalogo) bool {
	return ctx != nil && s != nil && r.Validar() == nil && v.ValidarPara(r) == nil && c.Validar() == nil && dominiovec.ReferenciaMotivoAutorizacionV2Valida(m)
}

func (s *ServicioOfertaSAE) autorizar(ctx context.Context, vinculo dominiovec.VinculoAutenticacionActorV2, resultado dominiovec.ResultadoContextoActorRegistradoV2, correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2, motivo dominiovec.ReferenciaEntradaCatalogo, accion, audiencia, ref string, ambito puertosbolsa.AmbitoOfertaSAE, huella string) (puertosbolsa.AutorizacionOfertaSAE, error) {
	recurso := dominiovec.RecursoAutorizable{Referencia: ref, ModuloID: puertosbolsa.ModuloOfertaSAE, Tipo: puertosbolsa.TipoRecursoOfertaSAE,
		Ambitos: map[string]string{"unidad_ref": ambito.UnidadRef, "ambito_ref": ambito.AmbitoRef}}
	if huella != "" {
		recurso.Atributos = map[string]string{"material_sha256": huella}
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: vinculo,
		ReferenciaMotivo: motivo, Accion: accion, Recurso: recurso, Finalidad: puertosbolsa.FinalidadOfertaSAE, Correlacion: correlacion})
	if err != nil {
		return puertosbolsa.AutorizacionOfertaSAE{}, dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil || nulo(exportador) || decision.ValidarPara(solicitud) != nil {
		return puertosbolsa.AutorizacionOfertaSAE{}, dependenciaSAE(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(solicitud, decision, confirmacion, resultado, motivo, material, audiencia) {
		return puertosbolsa.AutorizacionOfertaSAE{}, puertosbolsa.ErrOfertaSAENoDisponible
	}
	return puertosbolsa.AutorizacionOfertaSAE{Solicitud: solicitud, Decision: decision, Confirmacion: confirmacion, Material: material}, nil
}

func dependenciaSAE(err error) error {
	if errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) {
		return err
	}
	return puertosbolsa.ErrOfertaSAENoDisponible
}
func claveSAEValida(s string) bool {
	return len(s) >= 8 && len(s) <= 256 && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\x00\r\n")
}
func refOfertaSAEValida(s string) bool {
	return strings.HasPrefix(s, "oferta-sae:") && len(s) > len("oferta-sae:") &&
		len(s) <= 256 && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\x00\r\n")
}
func accionSAEValida(s string) bool {
	switch s {
	case dominiobolsa.AccionSAEEnviar, dominiobolsa.AccionSAERegistrarCandidato, dominiobolsa.AccionSAEConciliarPersona, dominiobolsa.AccionSAERecibirCandidatos,
		dominiobolsa.AccionSAEIniciarSeleccion, dominiobolsa.AccionSAEValorar, dominiobolsa.AccionSAEResolver, dominiobolsa.AccionSAEDeclararDesierta:
		return true
	}
	return false
}
func sufijoSAE(base, clave string) string {
	h := sha256.Sum256([]byte(base + "\x1f" + clave))
	return hex.EncodeToString(h[:])
}
func hashSAE(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
