package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type ServicioCatalogoCausasParticipacion struct {
	autorizador puertosbolsa.AutorizadorCatalogoCausasParticipacionV3
	repositorio puertosbolsa.RepositorioCatalogoCausasParticipacion
}

func NuevoServicioCatalogoCausasParticipacion(a puertosbolsa.AutorizadorCatalogoCausasParticipacionV3, r puertosbolsa.RepositorioCatalogoCausasParticipacion) (*ServicioCatalogoCausasParticipacion, error) {
	if a == nil || r == nil {
		return nil, puertosbolsa.ErrCatalogoCausasParticipacionNoDisponible
	}
	return &ServicioCatalogoCausasParticipacion{a, r}, nil
}
func huellaCausaParticipacion(c puertosbolsa.CausaParticipacionCatalogada) string {
	v := []string{"bolsa.causa_participacion.v1", c.Codigo, strconv.FormatInt(c.Version, 10), c.Etiqueta, strconv.FormatBool(c.AplicaSituacion), strconv.FormatBool(c.AplicaContacto), strconv.FormatBool(c.Publicable), strconv.FormatBool(c.Activa)}
	h := sha256.Sum256([]byte(strings.Join(v, "\n")))
	return hex.EncodeToString(h[:])
}
func reciboCausaParticipacion(c puertosbolsa.CausaParticipacionCatalogada) string {
	h := sha256.Sum256([]byte("bolsa.causa_participacion.recibo.v1\n" + huellaCausaParticipacion(c)))
	return "recibo:causa:" + hex.EncodeToString(h[:])
}
func propuestaCausaParticipacion(c puertosbolsa.CausaParticipacionCatalogada, actor string) string {
	h := sha256.Sum256([]byte("bolsa.causa_participacion.propuesta.v1\n" + huellaCausaParticipacion(c) + "\n" + actor))
	return "propuesta:causa:" + hex.EncodeToString(h[:])
}
func reciboPropuestaCausaParticipacion(ref string) string {
	h := sha256.Sum256([]byte("bolsa.causa_participacion.propuesta.recibo.v1\n" + ref))
	return "recibo:propuesta:causa:" + hex.EncodeToString(h[:])
}
func (s *ServicioCatalogoCausasParticipacion) Proponer(ctx context.Context, q puertosbolsa.SolicitudProponerCausaParticipacion) (puertosbolsa.PropuestaCausaParticipacion, string, error) {
	if s == nil || ctx == nil || q.Validar() != nil || q.ResultadoContexto.Contexto.Principal.ID == "" {
		return puertosbolsa.PropuestaCausaParticipacion{}, "", puertosbolsa.ErrCatalogoCausasParticipacionNoDisponible
	}
	c := q.Causa
	c.HuellaSHA256 = huellaCausaParticipacion(c)
	actor := q.ResultadoContexto.Contexto.Principal.ID
	ref := propuestaCausaParticipacion(c, actor)
	recibo := reciboPropuestaCausaParticipacion(ref)
	recurso := dominiovec.RecursoAutorizable{Referencia: puertosbolsa.ReferenciaCatalogoCausasParticipacion, ModuloID: "bolsa", Tipo: puertosbolsa.TipoRecursoCatalogoCausasParticipacion, Atributos: map[string]string{"causa_sha256": c.HuellaSHA256, "propuesta_ref": ref, "recibo_ref": recibo}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion, Accion: puertosbolsa.AccionProponerCausasParticipacion, Recurso: recurso, Finalidad: puertosbolsa.FinalidadProponerCausasParticipacion, Correlacion: q.Correlacion})
	if err != nil {
		return puertosbolsa.PropuestaCausaParticipacion{}, "", dominiovec.ErrAutorizacionDenegada
	}
	d, cf, e, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, q.ResultadoContexto)
	if err != nil || e == nil || d.ValidarPara(auth) != nil {
		return puertosbolsa.PropuestaCausaParticipacion{}, "", dependenciaCatalogo(err)
	}
	m, err := e.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, d, cf, q.ResultadoContexto, q.MotivoAutorizacion, m, puertosbolsa.AudienciaProponerCausasParticipacion) {
		return puertosbolsa.PropuestaCausaParticipacion{}, "", dependenciaCatalogo(err)
	}
	return s.repositorio.ProponerCausaParticipacion(ctx, puertosbolsa.ComandoProponerCausaParticipacion{Causa: c, Actor: actor, PropuestaRef: ref, ReciboRef: recibo, SolicitudAutorizacion: auth, Decision: d, Confirmacion: cf, Material: m})
}
func (s *ServicioCatalogoCausasParticipacion) Publicar(ctx context.Context, q puertosbolsa.SolicitudPublicarCausaParticipacion) (puertosbolsa.CausaParticipacionCatalogada, string, error) {
	if s == nil || ctx == nil || q.Validar() != nil || q.ResultadoContexto.Contexto.PersonaRef == "" {
		return puertosbolsa.CausaParticipacionCatalogada{}, "", puertosbolsa.ErrCatalogoCausasParticipacionNoDisponible
	}
	c := q.Causa
	c.HuellaSHA256 = huellaCausaParticipacion(c)
	recibo := reciboCausaParticipacion(c)
	recurso := dominiovec.RecursoAutorizable{Referencia: puertosbolsa.ReferenciaCatalogoCausasParticipacion, ModuloID: "bolsa", Tipo: puertosbolsa.TipoRecursoCatalogoCausasParticipacion, Atributos: map[string]string{"causa_sha256": c.HuellaSHA256, "propuesta_ref": q.PropuestaRef, "recibo_ref": recibo}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion, Accion: puertosbolsa.AccionPublicarCausasParticipacion, Recurso: recurso, Finalidad: puertosbolsa.FinalidadPublicarCausasParticipacion, Correlacion: q.Correlacion})
	if err != nil {
		return c, "", dominiovec.ErrAutorizacionDenegada
	}
	d, cf, e, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, q.ResultadoContexto)
	if err != nil || e == nil || d.ValidarPara(auth) != nil {
		return c, "", dependenciaCatalogo(err)
	}
	m, err := e.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, d, cf, q.ResultadoContexto, q.MotivoAutorizacion, m, puertosbolsa.AudienciaPublicarCausasParticipacion) {
		return c, "", dependenciaCatalogo(err)
	}
	return s.repositorio.PublicarCausaParticipacion(ctx, puertosbolsa.ComandoPublicarCausaParticipacion{Causa: c, PropuestaRef: q.PropuestaRef, Actor: q.ResultadoContexto.Contexto.Principal.ID, ReciboRef: recibo, SolicitudAutorizacion: auth, Decision: d, Confirmacion: cf, Material: m})
}
func (s *ServicioCatalogoCausasParticipacion) Consultar(ctx context.Context, q puertosbolsa.SolicitudConsultarCausasParticipacion) ([]puertosbolsa.CausaParticipacionCatalogada, error) {
	if s == nil || ctx == nil || q.Validar() != nil || q.ResultadoContexto.Contexto.PersonaRef == "" {
		return nil, puertosbolsa.ErrCatalogoCausasParticipacionNoDisponible
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: puertosbolsa.ReferenciaCatalogoCausasParticipacion, ModuloID: "bolsa", Tipo: puertosbolsa.TipoRecursoCatalogoCausasParticipacion}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion, Accion: puertosbolsa.AccionConsultarCausasParticipacion, Recurso: recurso, Finalidad: puertosbolsa.FinalidadConsultarCausasParticipacion, Correlacion: q.Correlacion})
	if err != nil {
		return nil, dominiovec.ErrAutorizacionDenegada
	}
	d, cf, e, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, q.ResultadoContexto)
	if err != nil || e == nil || d.ValidarPara(auth) != nil {
		return nil, dependenciaCatalogo(err)
	}
	m, err := e.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, d, cf, q.ResultadoContexto, q.MotivoAutorizacion, m, puertosbolsa.AudienciaConsultarCausasParticipacion) {
		return nil, dependenciaCatalogo(err)
	}
	return s.repositorio.ListarCausasParticipacion(ctx, puertosbolsa.ConsultaCausasParticipacionAutorizada{Actor: q.ResultadoContexto.Contexto.Principal.ID, SolicitudAutorizacion: auth, Decision: d, Confirmacion: cf, Material: m})
}

func (s *ServicioCatalogoCausasParticipacion) ConsultarPropuesta(ctx context.Context, q puertosbolsa.SolicitudConsultarPropuestaCausaParticipacion) (puertosbolsa.PropuestaCausaParticipacionLeida, error) {
	if s == nil || ctx == nil || q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		q.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) ||
		!propuestaCausaParticipacionRefValida(q.PropuestaRef) || q.ResultadoContexto.Contexto.Principal.ID == "" {
		return puertosbolsa.PropuestaCausaParticipacionLeida{}, puertosbolsa.ErrCatalogoCausasParticipacionNoDisponible
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: q.PropuestaRef, ModuloID: "bolsa", Tipo: puertosbolsa.TipoRecursoPropuestaCatalogoCausasParticipacion,
		Atributos: map[string]string{"propuesta_ref": q.PropuestaRef},
	}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion,
		Accion: puertosbolsa.AccionConsultarPropuestaCausasParticipacion, Recurso: recurso,
		Finalidad: puertosbolsa.FinalidadConsultarPropuestaCausasParticipacion, Correlacion: q.Correlacion,
	})
	if err != nil {
		return puertosbolsa.PropuestaCausaParticipacionLeida{}, dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, q.ResultadoContexto)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		return puertosbolsa.PropuestaCausaParticipacionLeida{}, dependenciaCatalogo(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, q.ResultadoContexto, q.MotivoAutorizacion, material, puertosbolsa.AudienciaConsultarPropuestaCausasParticipacion) {
		return puertosbolsa.PropuestaCausaParticipacionLeida{}, dependenciaCatalogo(err)
	}
	return s.repositorio.ConsultarPropuestaCausaParticipacion(ctx, puertosbolsa.ConsultaPropuestaCausaParticipacionAutorizada{
		PropuestaRef: q.PropuestaRef, Actor: q.ResultadoContexto.Contexto.Principal.ID,
		SolicitudAutorizacion: auth, Decision: decision, Confirmacion: confirmacion, Material: material,
	})
}

func propuestaCausaParticipacionRefValida(ref string) bool {
	const prefijo = "propuesta:causa:"
	if !strings.HasPrefix(ref, prefijo) || len(ref) != len(prefijo)+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(ref, prefijo))
	return err == nil && strings.ToLower(ref) == ref
}

func dependenciaCatalogo(err error) error {
	if errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) {
		return err
	}
	return puertosbolsa.ErrCatalogoCausasParticipacionNoDisponible
}
