package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrOrdenGobiernoCategoriaRPTInvalida = errors.New("vec: orden interna de gobierno RPT invalida")

// CredencialesGobiernoCategoriaRPT solo se construye con capacidades de las
// fronteras de identidad y motivo. Nunca procede de un cuerpo HTTP.
type CredencialesGobiernoCategoriaRPT struct {
	Actor             domain.ContextoActor
	Vinculo           domain.VinculoAutenticacionActorV2
	ResultadoContexto domain.ResultadoContextoActorRegistradoV2
	Motivo            domain.ReferenciaEntradaCatalogo
	Correlacion       domain.ReferenciaCorrelacionAutorizacionV2
}

func (CredencialesGobiernoCategoriaRPT) MarshalJSON() ([]byte, error) {
	return nil, ErrOrdenGobiernoCategoriaRPTInvalida
}
func (*CredencialesGobiernoCategoriaRPT) UnmarshalJSON([]byte) error {
	return ErrOrdenGobiernoCategoriaRPTInvalida
}

type OrdenProponerGobiernoCategoriaRPT struct {
	Credenciales CredencialesGobiernoCategoriaRPT
	Borrador     ports.BorradorPropuestaGobiernoCategoriaRPT
}

type OrdenAvanzarGobiernoCategoriaRPT struct {
	Credenciales CredencialesGobiernoCategoriaRPT
	Material     ports.MaterialAvanceGobiernoCategoriaRPT
}

func (OrdenProponerGobiernoCategoriaRPT) MarshalJSON() ([]byte, error) {
	return nil, ErrOrdenGobiernoCategoriaRPTInvalida
}
func (*OrdenProponerGobiernoCategoriaRPT) UnmarshalJSON([]byte) error {
	return ErrOrdenGobiernoCategoriaRPTInvalida
}
func (OrdenAvanzarGobiernoCategoriaRPT) MarshalJSON() ([]byte, error) {
	return nil, ErrOrdenGobiernoCategoriaRPTInvalida
}
func (*OrdenAvanzarGobiernoCategoriaRPT) UnmarshalJSON([]byte) error {
	return ErrOrdenGobiernoCategoriaRPTInvalida
}

type ServicioGobiernoCategoriaRPT struct {
	preparador    ports.PreparadorGobiernoCategoriaRPT
	autorizador   ports.AutorizadorGobiernoCategoriaRPT
	gestor        ports.GestorGobiernoCategoriaRPT
	reloj         ports.Reloj
	versionRolRef string
}

func NuevoServicioGobiernoCategoriaRPT(
	preparador ports.PreparadorGobiernoCategoriaRPT,
	autorizador ports.AutorizadorGobiernoCategoriaRPT,
	gestor ports.GestorGobiernoCategoriaRPT,
	reloj ports.Reloj,
	versionRolRef string,
) (*ServicioGobiernoCategoriaRPT, error) {
	if nuloGobiernoCategoriaRPT(preparador) || nuloGobiernoCategoriaRPT(autorizador) ||
		nuloGobiernoCategoriaRPT(gestor) || nuloGobiernoCategoriaRPT(reloj) ||
		!referenciaGobiernoCategoriaRPTValida(versionRolRef) {
		return nil, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	return &ServicioGobiernoCategoriaRPT{preparador, autorizador, gestor, reloj, versionRolRef}, nil
}

func (s *ServicioGobiernoCategoriaRPT) Proponer(ctx context.Context, o OrdenProponerGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	var cero ports.ResultadoGobiernoCategoriaRPT
	b := o.Borrador
	if s == nil || ctx == nil || ctx.Err() != nil ||
		!b.Contenido.TamanoBorradorValido() ||
		!referenciaGobiernoCategoriaRPTValida(b.PropuestaRef) ||
		!referenciaGobiernoCategoriaRPTValida(b.ReciboRef) ||
		b.Contenido.MotivoRef != o.Credenciales.Motivo.Referencia() {
		return cero, ErrOrdenGobiernoCategoriaRPTInvalida
	}
	b.Contenido = clonarContenidoGobiernoCategoriaRPT(b.Contenido)
	contenido, err := b.Contenido.PrepararBorradorParaEditor(o.Credenciales.Actor.Principal.ID)
	if err != nil {
		return cero, ErrOrdenGobiernoCategoriaRPTInvalida
	}
	b.Contenido = contenido
	original := clonarContenidoGobiernoCategoriaRPT(contenido)
	p, err := s.preparador.PrepararPropuestaGobiernoCategoriaRPT(ctx, b)
	if err != nil {
		return cero, errorDependenciaGobiernoCategoriaRPT(ctx, err)
	}
	m := p.Material
	comparacion := m.Contenido
	comparacion.PreimagenesHuellaSHA256 = ""
	if m.PropuestaRef != b.PropuestaRef || m.ReciboRef != b.ReciboRef ||
		!reflect.DeepEqual(comparacion, original) ||
		!huellaGobiernoCategoriaRPTValida(m.HuellaSHA256) ||
		m.Contenido.ValidarParaEditor(o.Credenciales.Actor.Principal.ID) != nil {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	solicitud, material, err := s.autorizar(ctx, o.Credenciales, p.Autorizable,
		ports.AccionProponerGobiernoCategoriaRPT, m.PropuestaRef,
		m.Contenido.CatalogoID, m.Contenido.ModuloID, m.HuellaSHA256)
	if err != nil {
		return cero, errorDependenciaGobiernoCategoriaRPT(ctx, err)
	}
	r, err := s.gestor.ProponerGobiernoCategoriaRPT(ctx, ports.OrdenPropuestaGobiernoCategoriaRPT{Material: m, Solicitud: solicitud, Autorizacion: material})
	if err != nil {
		return cero, errorDependenciaGobiernoCategoriaRPT(ctx, err)
	}
	if !resultadoGobiernoCategoriaRPTValido(r, m.PropuestaRef, m.HuellaSHA256,
		m.ReciboRef, 1, domain.EstadoGobiernoCategoriaRPTPropuesta, material.ResumenCapacidad()) {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	return r, nil
}

func (s *ServicioGobiernoCategoriaRPT) Aprobar(ctx context.Context, o OrdenAvanzarGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	return s.avanzar(ctx, o, ports.AccionAprobarGobiernoCategoriaRPT)
}

func (s *ServicioGobiernoCategoriaRPT) Confirmar(ctx context.Context, o OrdenAvanzarGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	return s.avanzar(ctx, o, ports.AccionConfirmarGobiernoCategoriaRPT)
}

func (s *ServicioGobiernoCategoriaRPT) avanzar(ctx context.Context, o OrdenAvanzarGobiernoCategoriaRPT, accion string) (ports.ResultadoGobiernoCategoriaRPT, error) {
	var cero ports.ResultadoGobiernoCategoriaRPT
	m := o.Material
	if s == nil || ctx == nil || ctx.Err() != nil ||
		!referenciaGobiernoCategoriaRPTValida(m.PropuestaRef) ||
		!referenciaGobiernoCategoriaRPTValida(m.ReciboRef) ||
		!huellaGobiernoCategoriaRPTValida(m.HuellaSHA256) ||
		m.CatalogoID == "" || m.ModuloID == "" {
		return cero, ErrOrdenGobiernoCategoriaRPTInvalida
	}
	var esperado int64
	var estado string
	var p ports.PreparacionGobiernoCategoriaRPT
	var err error
	switch accion {
	case ports.AccionAprobarGobiernoCategoriaRPT:
		if m.RevisionEsperada != 1 {
			return cero, ErrOrdenGobiernoCategoriaRPTInvalida
		}
		esperado, estado = 2, domain.EstadoGobiernoCategoriaRPTAprobada
		p, err = s.preparador.PrepararAprobacionGobiernoCategoriaRPT(ctx, m)
	case ports.AccionConfirmarGobiernoCategoriaRPT:
		if m.RevisionEsperada != 2 {
			return cero, ErrOrdenGobiernoCategoriaRPTInvalida
		}
		esperado, estado = 3, domain.EstadoGobiernoCategoriaRPTConfirmada
		p, err = s.preparador.PrepararConfirmacionGobiernoCategoriaRPT(ctx, m)
	default:
		return cero, ErrOrdenGobiernoCategoriaRPTInvalida
	}
	if err != nil {
		return cero, errorDependenciaGobiernoCategoriaRPT(ctx, err)
	}
	solicitud, material, err := s.autorizar(ctx, o.Credenciales, p, accion, m.PropuestaRef, m.CatalogoID, m.ModuloID, m.HuellaSHA256)
	if err != nil {
		return cero, errorDependenciaGobiernoCategoriaRPT(ctx, err)
	}
	orden := ports.OrdenAvanceGobiernoCategoriaRPT{Material: m, Solicitud: solicitud, Autorizacion: material}
	var r ports.ResultadoGobiernoCategoriaRPT
	if accion == ports.AccionAprobarGobiernoCategoriaRPT {
		r, err = s.gestor.AprobarGobiernoCategoriaRPT(ctx, orden)
	} else {
		r, err = s.gestor.ConfirmarGobiernoCategoriaRPT(ctx, orden)
	}
	if err != nil {
		return cero, errorDependenciaGobiernoCategoriaRPT(ctx, err)
	}
	if !resultadoGobiernoCategoriaRPTValido(r, m.PropuestaRef, m.HuellaSHA256, m.ReciboRef,
		esperado, estado, material.ResumenCapacidad()) {
		return cero, ports.ErrGobiernoCategoriaRPTNoConfiable
	}
	return r, nil
}

func (s *ServicioGobiernoCategoriaRPT) autorizar(ctx context.Context, c CredencialesGobiernoCategoriaRPT,
	p ports.PreparacionGobiernoCategoriaRPT, accion, propuestaRef, catalogoID, moduloID, huella string,
) (domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var solicitud domain.SolicitudAutorizacionLigadaV3
	var cero ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	instanteInicial := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	v, err := c.Vinculo.Datos()
	h, errActor := c.Actor.HuellaSHA256VinculadaV2()
	if err != nil || errActor != nil || c.Actor.Validar() != nil ||
		c.ResultadoContexto.Validar() != nil || c.ResultadoContexto.HuellaSHA256 != h ||
		c.Vinculo.ValidarPara(c.ResultadoContexto) != nil || !c.Vinculo.VigenteEn(instanteInicial, c.ResultadoContexto) ||
		c.Actor.Principal.ID != c.Actor.PersonaRef || c.Actor.Principal.ID != v.PrincipalID || c.Actor.PerfilActivoRef != v.PerfilActivoRef ||
		v.CuentaPrivilegiada || v.Superficie != domain.SuperficieAutenticacionInternaCorporativaV1 ||
		!v.GarantiaObservada.Cumple(domain.AuthAssuranceHigh) ||
		!referenciaGobiernoCategoriaRPTValida(s.versionRolRef) ||
		!c.Actor.Principal.AuthAssurance.Cumple(domain.AuthAssuranceHigh) ||
		!domain.ReferenciaMotivoAutorizacionV2Valida(c.Motivo) || c.Correlacion.Validar() != nil ||
		p.Accion != accion || p.Finalidad != ports.FinalidadGobiernoCategoriaRPT ||
		p.Audiencia != ports.AudienciaGobiernoCategoriaRPT || p.HuellaPropuesta != huella ||
		p.Recurso.Referencia != propuestaRef || p.Recurso.ModuloID != moduloID ||
		p.Recurso.Tipo != ports.TipoRecursoGobiernoCategoriaRPT ||
		len(p.Recurso.Ambitos) != 2 || p.Recurso.Ambitos["catalogo_id"] != catalogoID ||
		p.Recurso.Ambitos["modulo_id"] != moduloID || len(p.Recurso.Atributos) != 1 ||
		!huellaGobiernoCategoriaRPTValida(p.Recurso.Atributos["material_sha256"]) ||
		p.Recurso.Validar() != nil {
		return solicitud, cero, ports.ErrGobiernoCategoriaRPTDenegado
	}
	solicitud, err = domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: c.Vinculo, ReferenciaMotivo: c.Motivo,
		Accion: accion, Recurso: p.Recurso, Finalidad: ports.FinalidadGobiernoCategoriaRPT,
		Correlacion: c.Correlacion,
	})
	if err != nil {
		return domain.SolicitudAutorizacionLigadaV3{}, cero, ports.ErrGobiernoCategoriaRPTDenegado
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, c.ResultadoContexto)
	if ctx.Err() != nil {
		return domain.SolicitudAutorizacionLigadaV3{}, cero, ctx.Err()
	}
	if errors.Is(err, ports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
		return domain.SolicitudAutorizacionLigadaV3{}, cero, ports.ErrGobiernoCategoriaRPTDenegado
	}
	if err != nil || nuloGobiernoCategoriaRPT(exportador) {
		return domain.SolicitudAutorizacionLigadaV3{}, cero, ports.ErrGobiernoCategoriaRPTNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	instanteConsumo := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if err != nil {
		return domain.SolicitudAutorizacionLigadaV3{}, cero, denegacionValidacionGobiernoCategoriaRPT(err)
	}
	if err := concesionGobiernoCategoriaRPTValida(material, solicitud, decision,
		confirmacion, c.ResultadoContexto, c.Actor, accion, p.Recurso, instanteConsumo, s.versionRolRef); err != nil {
		return domain.SolicitudAutorizacionLigadaV3{}, cero, ports.ErrGobiernoCategoriaRPTDenegado
	}
	return solicitud, material, nil
}

func concesionGobiernoCategoriaRPTValida(m ports.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	s domain.SolicitudAutorizacionLigadaV3, d domain.DecisionAutorizacionLigadaV3,
	c ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	resultado domain.ResultadoContextoActorRegistradoV2, actor domain.ContextoActor,
	accion string, recurso domain.RecursoAutorizable, ahora time.Time, versionRolRef string,
) error {
	if err := m.ValidarEstructura(); err != nil {
		return denegacionValidacionGobiernoCategoriaRPT(err)
	}
	if err := d.ValidarPara(s); err != nil {
		return denegacionValidacionGobiernoCategoriaRPT(err)
	}
	datos, err := s.Datos()
	if err != nil {
		return denegacionValidacionGobiernoCategoriaRPT(err)
	}
	if datos.Accion != accion || datos.Finalidad != ports.FinalidadGobiernoCategoriaRPT ||
		!reflect.DeepEqual(datos.Recurso, recurso) {
		return ports.ErrGobiernoCategoriaRPTDenegado
	}
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, datos.ReferenciaMotivo, resultado)
	if err != nil {
		return denegacionValidacionGobiernoCategoriaRPT(err)
	}
	if err := c.ValidarPara(orden); err != nil {
		return denegacionValidacionGobiernoCategoriaRPT(err)
	}
	cd, err := c.Datos()
	if err != nil {
		return denegacionValidacionGobiernoCategoriaRPT(err)
	}
	if !c.DentroDeVentanaEn(cd.RegistradaEn) {
		return ports.ErrGobiernoCategoriaRPTDenegado
	}
	dc, errD := domain.RepresentacionCanonicaDecisionAutorizacionV3(d)
	mc, errM := domain.RepresentacionCanonicaMotivoAutorizacionV2(datos.ReferenciaMotivo)
	h, errR := recurso.HuellaContextoAutorizacionSHA256()
	for _, causa := range []error{errD, errM, errR} {
		if causa != nil {
			return denegacionValidacionGobiernoCategoriaRPT(causa)
		}
	}
	hd, hm := sha256.Sum256(dc), sha256.Sum256(mc)
	r := m.ResumenCapacidad()
	proyeccion, err := domain.ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(m.PayloadVECAD3())
	if err != nil {
		return denegacionValidacionGobiernoCategoriaRPT(err)
	}
	// La proyeccion no concede autoridad: se contrasta el rol y despues el
	// mensaje completo contra la decision nominal y el material del consumidor.
	rolDecision, err := proyeccion.VersionRolRef()
	if err != nil || !referenciaGobiernoCategoriaRPTValida(versionRolRef) || rolDecision != versionRolRef {
		return ports.ErrGobiernoCategoriaRPTDenegado
	}
	cabecera, err := proyeccion.Cabecera()
	if err != nil {
		return denegacionValidacionGobiernoCategoriaRPT(err)
	}
	mensaje, err := domain.SerializarMensajeAtestacionAutorizacionV3(cabecera, d, datos.ReferenciaMotivo, resultado)
	if err != nil {
		return denegacionValidacionGobiernoCategoriaRPT(err)
	}
	if bytes.Equal(mensaje, m.PayloadVECAD3()) &&
		r.DecisionRef() == cd.DecisionRef && r.DecisionHuellaSHA256() == cd.DecisionHuellaSHA256 &&
		bytes.Equal(dc, m.DecisionCanonica()) && bytes.Equal(mc, m.MotivoCanonico()) &&
		bytes.Equal(resultado.RepresentacionCanonica, m.ContextoActorCanonico()) &&
		r.DecisionHuellaSHA256() == hex.EncodeToString(hd[:]) &&
		r.MotivoHuellaSHA256() == hex.EncodeToString(hm[:]) &&
		r.Operacion() == accion && r.EfectoRef() == recurso.Referencia &&
		r.EfectoHuellaSHA256() == h && r.AudienciaConsumo() == ports.AudienciaGobiernoCategoriaRPT &&
		r.ContextoRef() == resultado.RegistroContextoRef &&
		r.ContextoHuellaSHA256() == resultado.HuellaSHA256 &&
		m.PersonaVersion() == actor.Instantanea.PersonaVersion &&
		m.PerfilVersion() == actor.Instantanea.PerfilVersion &&
		!ahora.Before(r.EmitidaEn()) && ahora.Before(r.ExpiraEn()) {
		return nil
	}
	return ports.ErrGobiernoCategoriaRPTDenegado
}

// Solo transporta el tipo de la causa interna. El mensaje del proveedor puede
// contener datos privados y no debe atravesar esta frontera.
func denegacionValidacionGobiernoCategoriaRPT(causa error) error {
	return fmt.Errorf("%w: %T", ports.ErrGobiernoCategoriaRPTDenegado, causa)
}

func resultadoGobiernoCategoriaRPTValido(r ports.ResultadoGobiernoCategoriaRPT,
	propuestaRef, huella, recibo string, revision int64, estado string,
	resumen ports.ResumenCapacidadAtestacionAutorizacionV3,
) bool {
	e := r.Evidencia
	return r.PropuestaRef == propuestaRef && r.HuellaSHA256 == huella &&
		r.ReciboRef == recibo && r.Revision == revision && r.Estado == estado &&
		e.DecisionRef == resumen.DecisionRef() && e.EfectoRef == propuestaRef &&
		e.HuellaEfectoSHA256 == resumen.EfectoHuellaSHA256() &&
		huellaGobiernoCategoriaRPTValida(e.ConsumoHuellaSHA256) &&
		e.AuditoriaRef != "" && !e.ConsumidaEn.IsZero() && e.ConsumoNuevo
}

func referenciaGobiernoCategoriaRPTValida(s string) bool {
	if len(s) < 3 || len(s) > 160 || strings.ContainsRune(s, '*') {
		return false
	}
	for _, c := range s {
		if c < '!' || c > '~' {
			return false
		}
	}
	return true
}

func huellaGobiernoCategoriaRPTValida(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil && strings.ToLower(s) == s
}

func nuloGobiernoCategoriaRPT(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}

func clonarContenidoGobiernoCategoriaRPT(c domain.ContenidoGobiernoCategoriaRPT) domain.ContenidoGobiernoCategoriaRPT {
	if c.DocumentoCanonico != nil {
		s := *c.DocumentoCanonico
		c.DocumentoCanonico = &s
	}
	if c.DocumentoHuellaSHA256 != nil {
		s := *c.DocumentoHuellaSHA256
		c.DocumentoHuellaSHA256 = &s
	}
	if c.CategoriaID != nil {
		s := *c.CategoriaID
		c.CategoriaID = &s
	}
	if c.RevisionEsperada != nil {
		r := *c.RevisionEsperada
		c.RevisionEsperada = &r
	}
	if c.PreimagenesControl != nil {
		m := make(map[string]domain.PreimagenControlGobiernoCategoriaRPT, len(c.PreimagenesControl))
		for id, p := range c.PreimagenesControl {
			m[id] = p
		}
		c.PreimagenesControl = m
	}
	return c
}

func errorDependenciaGobiernoCategoriaRPT(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, conocido := range []error{
		ports.ErrGobiernoCategoriaRPTInvalido,
		ports.ErrGobiernoCategoriaRPTDenegado,
		ports.ErrGobiernoCategoriaRPTConflicto,
		ports.ErrGobiernoCategoriaRPTNoDisponible,
		ports.ErrGobiernoCategoriaRPTNoConfiable,
	} {
		if errors.Is(err, conocido) {
			return conocido
		}
	}
	return ports.ErrGobiernoCategoriaRPTNoDisponible
}
