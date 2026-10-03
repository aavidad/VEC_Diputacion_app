package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strconv"

	ctadapters "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	almacenvec "vec-diputacion-granada/internal/vec/adapters/almacen"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var errAutoridadOriginalDocumentosCTNoDisponible = errors.New("bootstrap: autoridad CT para el original de Documentos no disponible")

type operacionAutorizacionOriginalDocumentosCT struct {
	rutas           []string
	perfil          *perfilFijoCTDesarrollo
	motivo          core.ReferenciaEntradaCatalogo
	finalidad, tipo string
	campos          []string
	exportador      exportadorMaterialFirmaR5Desarrollo
}

type configuracionAutorizacionesOriginalDocumentosCT struct {
	soporte        *soporteAltaContratacionTemporalDesarrollo
	pdp            *autorizadorComunDesarrollo
	fronteras      catalogoFronterasComunDesarrollo
	reloj          relojContratacionTemporalDesarrollo
	catalogo       catalogoMaterialAutorizacionComunDesarrollo
	politicas      *conservacion.Catalogo
	seudonimizador *seudonimizadorAlmacenDesarrollo
	operaciones    map[string]operacionAutorizacionOriginalDocumentosCT
}

type autorizacionesOriginalDocumentosCT struct {
	soporte        *soporteAltaContratacionTemporalDesarrollo
	pdp            vecports.AutorizadorSolicitudLigadaV3
	reloj          vecports.Reloj
	politicas      *conservacion.Catalogo
	tipos          *ctadapters.TiposOriginalFirmableRRHH
	seudonimizador *seudonimizadorAlmacenDesarrollo
	operaciones    map[string]operacionAutorizacionOriginalDocumentosCT
}

// El montaje consume perfiles publicados. Nunca los prepara ni los publica.
func nuevasAutorizacionesOriginalDocumentosCTDesarrollo(configuraciones ...configuracionAutorizacionesOriginalDocumentosCT) (*autorizacionesOriginalDocumentosCT, error) {
	if len(configuraciones) != 1 {
		return nil, errAutoridadOriginalDocumentosCTNoDisponible
	}
	c := configuraciones[0]
	if c.soporte == nil || c.soporte.sello == nil || c.pdp == nil || c.pdp.servicio == nil ||
		!c.pdp.selector.catalogo.aceptaCatalogoFronteras(c.fronteras) || c.politicas == nil ||
		!c.seudonimizador.valido() || dependenciaEsNulaContratacionTemporalDesarrollo(c.reloj) || len(c.operaciones) != 5 {
		return nil, errAutoridadOriginalDocumentosCTNoDisponible
	}
	descriptor, ok := c.catalogo.descriptorPara(docports.AudienciaV3)
	if !ok || !slices.Contains(descriptoresMaterialDocumentosDesarrollo(), descriptor) {
		return nil, errAutoridadOriginalDocumentosCTNoDisponible
	}
	tipos, err := ctadapters.NuevosTiposOriginalFirmableRRHH(c.politicas)
	if err != nil {
		return nil, errAutoridadOriginalDocumentosCTNoDisponible
	}
	operaciones := make(map[string]operacionAutorizacionOriginalDocumentosCT, len(c.operaciones))
	for _, accion := range []string{docports.AccionDescargar, docports.AccionReservarOriginalFirmable, docports.AccionConfirmarOriginalFirmable,
		vecports.AccionNegocioEscribirOriginalFirmable, vecports.AccionNegocioLeerOriginalDocumentoGenerado} {
		o, ok := c.operaciones[accion]
		p := o.perfil
		if !ok || p == nil || p.plantilla.Validar() != nil || p.contexto.Resultado.Validar() != nil ||
			p.contexto.Vinculo.ValidarPara(p.contexto.Resultado) != nil || p.contextoEsperadoRegistrado.Validar() != nil ||
			dependenciaEsNulaContratacionTemporalDesarrollo(p.sesionOperativa) || p.perfilRef() != p.contexto.Resultado.Contexto.PerfilActivoRef ||
			p.contexto.Resultado.Contexto.Principal.ID != c.soporte.contexto.Resultado.Contexto.Principal.ID ||
			p.contexto.Resultado.Contexto.PersonaRef != c.soporte.contexto.Resultado.Contexto.PersonaRef ||
			!core.ReferenciaMotivoAutorizacionV2Valida(o.motivo) || len(o.rutas) == 0 || !perfilOriginalDocumentosCTConcede(p, accion, o) ||
			(!accionSoloAlmacenOriginalCT(accion) && dependenciaEsNulaContratacionTemporalDesarrollo(o.exportador)) {
			return nil, errAutoridadOriginalDocumentosCTNoDisponible
		}
		for _, ruta := range o.rutas {
			if !rutaContextoAutorizacionContratacionTemporalDesarrollo(ruta) || !rutaSesionOperativaCTDesarrollo(ruta) ||
				!p.atiendeMetodo(ruta, http.MethodPost) || c.soporte.perfilFijoParaRutaYMetodo(ruta, http.MethodPost) != p ||
				!fronteraFirmaR5Compuesta(c.pdp, c.fronteras, ruta, accion, p.perfilRef()) ||
				!fuenteOriginalDocumentosCTCompuesta(c.soporte, p, ruta, accion, o) {
				return nil, errAutoridadOriginalDocumentosCTNoDisponible
			}
		}
		o.rutas, o.campos = slices.Clone(o.rutas), slices.Clone(o.campos)
		operaciones[accion] = o
	}
	return &autorizacionesOriginalDocumentosCT{soporte: c.soporte, pdp: c.pdp, reloj: c.reloj, politicas: c.politicas,
		seudonimizador: c.seudonimizador, operaciones: operaciones, tipos: tipos}, nil
}

func accionSoloAlmacenOriginalCT(accion string) bool {
	return accion == vecports.AccionNegocioEscribirOriginalFirmable || accion == vecports.AccionNegocioLeerOriginalDocumentoGenerado
}

func perfilOriginalDocumentosCTConcede(p *perfilFijoCTDesarrollo, accion string, o operacionAutorizacionOriginalDocumentosCT) bool {
	if p == nil || o.finalidad == "" || o.tipo == "" || len(p.plantilla.AsignacionPerfil.Ambitos) != 1 {
		return false
	}
	ambito := p.plantilla.AsignacionPerfil.Ambitos[0]
	if ambito.Clave != "organizacion_ref" || !slices.Equal(ambito.Valores, []string{docports.OrganizacionRefV3}) {
		return false
	}
	for _, c := range p.plantilla.VersionRol.Concesiones {
		if c.Accion == accion && c.ModuloID == "documentos" && c.TipoRecurso == o.tipo &&
			slices.Equal(c.Finalidades, []string{o.finalidad}) && c.GarantiaMinima == core.AuthAssuranceHigh &&
			slices.Equal(c.CamposPermitidos, o.campos) && len(c.Obligaciones) == 0 {
			return true
		}
	}
	return false
}

func fuenteOriginalDocumentosCTCompuesta(s *soporteAltaContratacionTemporalDesarrollo, p *perfilFijoCTDesarrollo,
	ruta, accion string, o operacionAutorizacionOriginalDocumentosCT) bool {
	recurso := core.RecursoAutorizable{Referencia: "ref:" + huellaAltaContratacionTemporalDesarrollo("original-ct-probe"), ModuloID: "documentos", Tipo: o.tipo,
		Ambitos: map[string]string{"organizacion_ref": docports.OrganizacionRefV3}, Atributos: map[string]string{"preimagen_sha256": huellaAltaContratacionTemporalDesarrollo("original-ct-probe")}}
	d := core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: p.contexto.Vinculo, Accion: accion,
		Finalidad: o.finalidad, ReferenciaMotivo: o.motivo, Recurso: recurso}
	ctx := context.WithValue(context.Background(), claveSolicitudOriginalDocumentosCT{}, solicitudOriginalDocumentosCT{datos: d})
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, d)
	i, ok := s.instantaneaPerfilFijoParaContexto(ctx, ruta, p)
	if !ok || i.Validar() != nil || i.AsignacionPerfil.PerfilActivoRef != p.perfilRef() {
		return false
	}
	d.Recurso.Atributos = map[string]string{"preimagen_sha256": huellaAltaContratacionTemporalDesarrollo("original-ct-alterado")}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, d)
	_, ok = s.instantaneaPerfilFijoParaContexto(ctx, ruta, p)
	return !ok
}

type claveSolicitudOriginalDocumentosCT struct{}
type solicitudOriginalDocumentosCT struct {
	datos core.DatosSolicitudAutorizacionLigadaV3
}

// Lo invoca la fuente central antes de resolver la instantánea del perfil.
func solicitudAutorizacionOriginalDocumentosCTValida(ctx context.Context, d core.DatosSolicitudAutorizacionLigadaV3) bool {
	if ctx == nil {
		return false
	}
	e, ok := ctx.Value(claveSolicitudOriginalDocumentosCT{}).(solicitudOriginalDocumentosCT)
	return ok && d.Recurso.ModuloID == "documentos" &&
		maps.Equal(d.Recurso.Ambitos, map[string]string{"organizacion_ref": docports.OrganizacionRefV3}) &&
		d.Accion == e.datos.Accion && d.Finalidad == e.datos.Finalidad && d.ReferenciaMotivo == e.datos.ReferenciaMotivo &&
		d.Correlacion == e.datos.Correlacion &&
		d.Recurso.Referencia == e.datos.Recurso.Referencia && d.Recurso.Tipo == e.datos.Recurso.Tipo &&
		maps.Equal(d.Recurso.Atributos, e.datos.Recurso.Atributos) &&
		d.VinculoAutenticacionActor.CoincideExactamenteCon(e.datos.VinculoAutenticacionActor)
}

func (a *autorizacionesOriginalDocumentosCT) contextoVigente(ctx context.Context, accion string) (operacionAutorizacionOriginalDocumentosCT, contextoSeguridadComunDesarrollo, error) {
	vacio := contextoSeguridadComunDesarrollo{}
	if a == nil || a.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.pdp) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(a.reloj) || ctx == nil || ctx.Err() != nil {
		return operacionAutorizacionOriginalDocumentosCT{}, vacio, errAutoridadOriginalDocumentosCTNoDisponible
	}
	o, ok := a.operaciones[accion]
	cap, valida := a.soporte.capacidadValida(ctx)
	ahora := a.reloj.Ahora()
	if !ok || o.perfil == nil || !valida || cap.metodo != http.MethodPost || !slices.Contains(o.rutas, cap.ruta) ||
		cap.certificadoVerificadoEn.IsZero() || cap.certificadoValidoHasta.IsZero() ||
		cap.certificadoVerificadoEn.After(ahora) || !ahora.Before(cap.certificadoValidoHasta) ||
		!o.perfil.atiendeMetodo(cap.ruta, cap.metodo) || a.soporte.perfilFijoParaRutaYMetodo(cap.ruta, cap.metodo) != o.perfil {
		return o, vacio, docports.ErrAccesoDenegado
	}
	if _, estado := a.soporte.consumirPerfilFijoCTDesarrolloConEstado(ctx, o.perfil); estado != perfilFijoConsumoVigente {
		if estado == perfilFijoConsumoFuenteNoDisponible {
			return o, vacio, errAutoridadOriginalDocumentosCTNoDisponible
		}
		return o, vacio, docports.ErrAccesoDenegado
	}
	operativo, err := a.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil || operativo.Resultado.Contexto.PerfilActivoRef != o.perfil.perfilRef() ||
		operativo.Resultado.Contexto.Principal.ID != o.perfil.contexto.Resultado.Contexto.Principal.ID ||
		operativo.Resultado.Contexto.PersonaRef != o.perfil.contexto.Resultado.Contexto.PersonaRef ||
		!operativo.Vinculo.VigenteEn(ahora, operativo.Resultado) {
		return o, vacio, docports.ErrAccesoDenegado
	}
	return o, contextoSeguridadComunDesarrollo{Vinculo: operativo.Vinculo, Resultado: operativo.Resultado}, nil
}

func (a *autorizacionesOriginalDocumentosCT) solicitar(ctx context.Context, accion string, recurso core.RecursoAutorizable) (docautorizacion.ConcesionAlmacenV3, core.DatosVinculoAutenticacionActorV2, error) {
	var vacia docautorizacion.ConcesionAlmacenV3
	var actor core.DatosVinculoAutenticacionActorV2
	o, operativo, err := a.contextoVigente(ctx, accion)
	if err != nil {
		return vacia, actor, err
	}
	if recurso.Validar() != nil || recurso.ModuloID != "documentos" || recurso.Tipo != o.tipo ||
		!maps.Equal(recurso.Ambitos, map[string]string{"organizacion_ref": docports.OrganizacionRefV3}) {
		return vacia, actor, docports.ErrAccesoDenegado
	}
	actor, err = operativo.Vinculo.Datos()
	if err != nil {
		return vacia, actor, docports.ErrAccesoDenegado
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, actor, errAutoridadOriginalDocumentosCTNoDisponible
	}
	datos := core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: operativo.Vinculo, Accion: accion,
		Recurso: recurso, Finalidad: o.finalidad, ReferenciaMotivo: o.motivo, Correlacion: correlacion}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return vacia, actor, docports.ErrAccesoDenegado
	}
	datos.Recurso.Ambitos, datos.Recurso.Atributos = maps.Clone(recurso.Ambitos), maps.Clone(recurso.Atributos)
	ctx = context.WithValue(ctx, claveSolicitudOriginalDocumentosCT{}, solicitudOriginalDocumentosCT{datos: datos})
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := a.pdp.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if err != nil {
		return vacia, actor, errors.Join(docports.ErrAccesoDenegado, err)
	}
	orden, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, o.motivo, operativo.Resultado)
	if err != nil || confirmacion.ValidarPara(orden) != nil {
		return vacia, actor, docports.ErrAccesoDenegado
	}
	restricciones, err := decision.RestriccionesProyeccionPara(solicitud)
	if err != nil || !slices.Equal(restricciones.CamposPermitidos, o.campos) || len(restricciones.Obligaciones) != 0 {
		return vacia, actor, docports.ErrAccesoDenegado
	}
	return docautorizacion.ConcesionAlmacenV3{Solicitud: solicitud, Decision: decision, Confirmacion: confirmacion}, actor, nil
}

func (a *autorizacionesOriginalDocumentosCT) autorizar(ctx context.Context, accion string, preimagen []byte, id, expediente string) (docports.AutorizacionV3, error) {
	if a == nil || len(preimagen) == 0 || len(preimagen) > 16384 || !docdomain.ReferenciaOpacaValida(id) || !docdomain.ReferenciaOpacaValida(expediente) {
		return docports.AutorizacionV3{}, docports.ErrAccesoDenegado
	}
	o, ok := a.operaciones[accion]
	if !ok || accionSoloAlmacenOriginalCT(accion) || dependenciaEsNulaContratacionTemporalDesarrollo(o.exportador) {
		return docports.AutorizacionV3{}, docports.ErrAccesoDenegado
	}
	suma := sha256.Sum256(preimagen)
	recurso := core.RecursoAutorizable{Referencia: id, ModuloID: "documentos", Tipo: o.tipo,
		Ambitos: map[string]string{"organizacion_ref": docports.OrganizacionRefV3}, Atributos: map[string]string{"preimagen_sha256": hex.EncodeToString(suma[:])}}
	c, actor, err := a.solicitar(ctx, accion, recurso)
	if err != nil {
		return docports.AutorizacionV3{}, err
	}
	_, operativo, err := a.contextoVigente(ctx, accion)
	if err != nil {
		return docports.AutorizacionV3{}, err
	}
	material, err := o.exportador.proveerMaterialConfirmacion(ctx, c.Solicitud, c.Decision, c.Confirmacion, o.motivo, operativo.Resultado)
	if err != nil || !vecports.MaterialAtestadoLigadoV3(c.Solicitud, c.Decision, c.Confirmacion, operativo.Resultado, o.motivo, material, docports.AudienciaV3) {
		return docports.AutorizacionV3{}, errAutoridadOriginalDocumentosCTNoDisponible
	}
	resumen := material.ResumenCapacidad()
	ahora := a.reloj.Ahora()
	if resumen.Operacion() != accion || resumen.EfectoRef() != id || resumen.EfectoHuellaSHA256() != docports.HuellaEfectoV3(preimagen) ||
		resumen.AudienciaConsumo() != docports.AudienciaV3 || ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return docports.AutorizacionV3{}, docports.ErrAccesoDenegado
	}
	datos, err := c.Solicitud.Datos()
	if err != nil {
		return docports.AutorizacionV3{}, docports.ErrAccesoDenegado
	}
	correlacionRef, err := datos.Correlacion.ValorCanonico()
	if err != nil {
		return docports.AutorizacionV3{}, docports.ErrAccesoDenegado
	}
	return docports.AutorizacionV3{Material: material, Accion: accion, Finalidad: o.finalidad, RecursoRef: id, AmbitoRef: expediente,
		PrincipalID: actor.PrincipalID, PerfilActivoRef: actor.PerfilActivoRef, CorrelacionRef: correlacionRef}, nil
}

func identidadOriginalDocumentosCT(s vecports.SolicitudOriginalFirmableCT, ref string) (almacencanonico.IdentidadOriginalCT, string, error) {
	i := almacencanonico.IdentidadOriginalCT{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, Documento: s.Documento, Version: s.OriginalVersion}
	exp, err := ctapp.ReferenciaExpedienteDocumentalFormalizacion(s.ExpedienteRef)
	if err != nil || !i.Valida() || s.OrganizacionRef != docports.OrganizacionRefV3 || ref != i.Referencia() ||
		(s.OriginalRef != "" && s.OriginalRef != ref) {
		return i, "", docports.ErrAccesoDenegado
	}
	return i, exp, nil
}

func (a *autorizacionesOriginalDocumentosCT) AutorizarLecturaOriginalCT(ctx context.Context, s vecports.SolicitudOriginalFirmableCT, ref string) (docports.ConsultaDocumento, error) {
	if a == nil || a.tipos == nil {
		return docports.ConsultaDocumento{}, docports.ErrAccesoDenegado
	}
	if _, err := a.tipos.ResolverTipoOriginalCT(ctx, s.Documento); err != nil {
		return docports.ConsultaDocumento{}, docports.ErrAccesoDenegado
	}
	_, exp, err := identidadOriginalDocumentosCT(s, ref)
	if err != nil {
		return docports.ConsultaDocumento{}, err
	}
	q := docports.ConsultaDocumento{DocumentoID: ref, Version: s.OriginalVersion}
	preimagen, err := q.PreimagenDescargar()
	if err != nil {
		return docports.ConsultaDocumento{}, err
	}
	q.Autorizacion, err = a.autorizar(ctx, docports.AccionDescargar, preimagen, ref, exp)
	if err != nil {
		return docports.ConsultaDocumento{}, err
	}
	return q, nil
}

// La fuente CT obtiene q de su historia nominal autorizada y coteja, tras la
// descarga, el eco, la versión y la huella de los bytes. Este emisor concede la
// descarga exacta desde el perfil de la petición; no interpreta una cadena de firmas.
func (a *autorizacionesOriginalDocumentosCT) AutorizarLecturaPDFFirmaAnteriorCT(ctx context.Context, q ctports.SolicitudPDFFirmaAnterior) (docports.ConsultaDocumento, error) {
	if q.Validar() != nil || q.OrganizacionRef != docports.OrganizacionRefV3 {
		return docports.ConsultaDocumento{}, docports.ErrAccesoDenegado
	}
	expediente, err := ctapp.ReferenciaExpedienteDocumentalFormalizacion(q.ExpedienteRef)
	if err != nil {
		return docports.ConsultaDocumento{}, docports.ErrAccesoDenegado
	}
	consulta := docports.ConsultaDocumento{DocumentoID: q.DocumentoRef, Version: q.DocumentoVersion}
	preimagen, err := consulta.PreimagenDescargar()
	if err != nil {
		return docports.ConsultaDocumento{}, docports.ErrAccesoDenegado
	}
	consulta.Autorizacion, err = a.autorizar(ctx, docports.AccionDescargar, preimagen, q.DocumentoRef, expediente)
	if err != nil {
		return docports.ConsultaDocumento{}, err
	}
	return consulta, nil
}

func (a *autorizacionesOriginalDocumentosCT) PrepararCustodiaOriginalCT(ctx context.Context, s vecports.SolicitudOriginalFirmableCT, pdf vecports.PDFOriginalCT, ref string) (docports.OrdenCustodiarOriginalFirmable, docports.AutorizarOriginalFirmable, error) {
	i, exp, err := identidadOriginalDocumentosCT(s, ref)
	if err != nil || a == nil || a.politicas == nil || !a.politicas.CustodiaOriginalCTReservada(pdf.TipoRef) ||
		len(pdf.Contenido) < 8 || len(pdf.Contenido) > vecports.LimiteOriginalFirmableCT || !bytes.HasPrefix(pdf.Contenido, []byte("%PDF-")) {
		return docports.OrdenCustodiarOriginalFirmable{}, nil, docports.ErrAccesoDenegado
	}
	if a.tipos == nil {
		return docports.OrdenCustodiarOriginalFirmable{}, nil, docports.ErrAccesoDenegado
	}
	tipoEsperado, err := a.tipos.ResolverTipoOriginalCT(ctx, s.Documento)
	if err != nil || tipoEsperado != pdf.TipoRef {
		return docports.OrdenCustodiarOriginalFirmable{}, nil, docports.ErrAccesoDenegado
	}
	if _, _, err := a.contextoVigente(ctx, docports.AccionReservarOriginalFirmable); err != nil {
		return docports.OrdenCustodiarOriginalFirmable{}, nil, err
	}
	tipo, ok := a.politicas.ClaveTipo(pdf.TipoRef)
	if !ok {
		return docports.OrdenCustodiarOriginalFirmable{}, nil, docports.ErrAccesoDenegado
	}
	politica, err := a.politicas.SolicitudPara(tipo, exp)
	if err != nil {
		return docports.OrdenCustodiarOriginalFirmable{}, nil, err
	}
	suma := sha256.Sum256(pdf.Contenido)
	ligado := &emisorOriginalDocumentosCTLigado{base: a, peticion: ctx, id: ref, expediente: exp, tipo: pdf.TipoRef, clave: i.ClaveLogica(),
		version: s.OriginalVersion, huella: hex.EncodeToString(suma[:]), tamano: int64(len(pdf.Contenido))}
	autoridad, err := docautorizacion.NuevaAutoridadOriginalFirmableV3(ligado, ligado, a.reloj)
	if err != nil {
		return docports.OrdenCustodiarOriginalFirmable{}, nil, err
	}
	return docports.OrdenCustodiarOriginalFirmable{ID: ref, ClaveIdempotencia: i.ClaveLogica(), ModuloID: moduloProductorCustodiaCT,
		ExpedienteRef: exp, TipoRef: pdf.TipoRef, Version: s.OriginalVersion, MIME: "application/pdf", Contenido: bytes.Clone(pdf.Contenido), SolicitudPolitica: politica}, autoridad, nil
}

type emisorOriginalDocumentosCTLigado struct {
	peticion                            context.Context
	base                                *autorizacionesOriginalDocumentosCT
	id, expediente, tipo, clave, huella string
	version                             uint64
	tamano                              int64
}

// La autoridad ligada no se reutiliza desde otra petición, aunque sea del mismo actor.
func (e *emisorOriginalDocumentosCTLigado) mismaPeticion(ctx context.Context) bool {
	return e != nil && e.base != nil && ctx != nil && e.peticion != nil &&
		reflect.TypeOf(ctx).Comparable() && reflect.TypeOf(e.peticion).Comparable() && ctx == e.peticion
}

func (e *emisorOriginalDocumentosCTLigado) AutorizarOperacionOriginalFirmable(ctx context.Context, accion string, preimagen []byte, id, exp string) (docports.AutorizacionV3, error) {
	if !e.mismaPeticion(ctx) || id != e.id || exp != e.expediente || len(preimagen) == 0 || len(preimagen) > 16384 || validarClavesJSONUnicas(preimagen) != nil {
		return docports.AutorizacionV3{}, docports.ErrAccesoDenegado
	}
	var p struct {
		Accion     string `json:"accion"`
		ID         string `json:"id"`
		Expediente string `json:"expediente_ref"`
		Modulo     string `json:"modulo_id"`
		Tipo       string `json:"tipo_ref"`
		Clave      string `json:"clave_idempotencia"`
		Version    uint64 `json:"version"`
		Huella     string `json:"huella_sha256"`
		Tamano     int64  `json:"tamano"`
	}
	if json.Unmarshal(preimagen, &p) != nil || p.Accion != accion || p.ID != e.id || p.Huella != e.huella {
		return docports.AutorizacionV3{}, docports.ErrAccesoDenegado
	}
	switch accion {
	case docports.AccionReservarOriginalFirmable:
		if p.Expediente != e.expediente || p.Modulo != moduloProductorCustodiaCT || p.Tipo != e.tipo || p.Clave != e.clave || p.Version != e.version || p.Tamano != e.tamano {
			return docports.AutorizacionV3{}, docports.ErrAccesoDenegado
		}
	case docports.AccionConfirmarOriginalFirmable:
	default:
		return docports.AutorizacionV3{}, docports.ErrAccesoDenegado
	}
	return e.base.autorizar(ctx, accion, preimagen, id, exp)
}

func (a *autorizacionesOriginalDocumentosCT) SeudonimosLecturaOriginal(ctx context.Context, aut docports.AutorizacionV3) (docautorizacion.DatosSeudonimosLectura, error) {
	_, operativo, err := a.contextoVigente(ctx, aut.Accion)
	if err != nil {
		return docautorizacion.DatosSeudonimosLectura{}, err
	}
	actor, err := operativo.Vinculo.Datos()
	if err != nil || !a.seudonimizador.valido() || actor.PrincipalID != aut.PrincipalID || actor.PerfilActivoRef != aut.PerfilActivoRef {
		return docautorizacion.DatosSeudonimosLectura{}, docports.ErrAccesoDenegado
	}
	if aut.Accion == docports.AccionDescargar {
		err = aut.ValidarPara(aut.Accion, a.reloj.Ahora())
	} else {
		err = docapp.ValidarAutorizacionOriginalFirmable(aut, aut.Accion, a.reloj.Ahora())
	}
	if err != nil {
		return docautorizacion.DatosSeudonimosLectura{}, docports.ErrAccesoDenegado
	}
	return (&emisorConcesionAlmacenDocumentosDesarrollo{seudonimizador: a.seudonimizador}).SeudonimosLecturaOriginal(ctx, aut)
}

func (e *emisorOriginalDocumentosCTLigado) SeudonimosLecturaOriginal(ctx context.Context, a docports.AutorizacionV3) (docautorizacion.DatosSeudonimosLectura, error) {
	if !e.mismaPeticion(ctx) || a.RecursoRef != e.id || a.AmbitoRef != e.expediente || a.Accion != docports.AccionReservarOriginalFirmable {
		return docautorizacion.DatosSeudonimosLectura{}, docports.ErrAccesoDenegado
	}
	return e.base.SeudonimosLecturaOriginal(ctx, a)
}

func (a *autorizacionesOriginalDocumentosCT) EmitirConcesionAlmacenV3(ctx context.Context, s docautorizacion.SolicitudConcesionAlmacenV3) (docautorizacion.ConcesionAlmacenV3, error) {
	if s.Accion != vecports.AccionNegocioLeerOriginalDocumentoGenerado || s.Recurso.Atributos["documento_modulo_productor"] != moduloProductorCustodiaCT {
		return docautorizacion.ConcesionAlmacenV3{}, docports.ErrAccesoDenegado
	}
	return a.emitirAlmacen(ctx, s)
}

func (e *emisorOriginalDocumentosCTLigado) EmitirConcesionAlmacenV3(ctx context.Context, s docautorizacion.SolicitudConcesionAlmacenV3) (docautorizacion.ConcesionAlmacenV3, error) {
	if !e.mismaPeticion(ctx) || s.Accion != vecports.AccionNegocioEscribirOriginalFirmable || s.Recurso.Referencia != e.id ||
		s.Recurso.Atributos["documentos_original_expediente_ref"] != e.expediente || s.Recurso.Atributos["documentos_original_tipo_ref"] != e.tipo ||
		s.Recurso.Atributos["documentos_original_huella_sha256"] != e.huella || s.Recurso.Atributos["documentos_original_version"] != strconv.FormatUint(e.version, 10) {
		return docautorizacion.ConcesionAlmacenV3{}, docports.ErrAccesoDenegado
	}
	return e.base.emitirAlmacen(ctx, s)
}

func (a *autorizacionesOriginalDocumentosCT) emitirAlmacen(ctx context.Context, s docautorizacion.SolicitudConcesionAlmacenV3) (docautorizacion.ConcesionAlmacenV3, error) {
	o, operativo, err := a.contextoVigente(ctx, s.Accion)
	if err != nil {
		return docautorizacion.ConcesionAlmacenV3{}, err
	}
	actor, err := operativo.Vinculo.Datos()
	if err != nil || s.PrincipalID != actor.PrincipalID || s.PerfilActivoRef != actor.PerfilActivoRef || s.Finalidad != o.finalidad ||
		s.CorrelacionRef == "" || s.Recurso.Atributos[vecports.AtributoAlmacenOperacionRef] != s.CorrelacionRef {
		return docautorizacion.ConcesionAlmacenV3{}, docports.ErrAccesoDenegado
	}
	c, _, err := a.solicitar(ctx, s.Accion, s.Recurso)
	return c, err
}

func (a *autorizacionesOriginalDocumentosCT) ContextoLecturaOriginal(ctx context.Context, d docdomain.Documento, aut docports.AutorizacionV3) (vecports.ContextoOperacionAlmacen, error) {
	if a == nil {
		return vecports.ContextoOperacionAlmacen{}, docports.ErrAccesoDenegado
	}
	f, err := docautorizacion.NuevaFabricaContextoLecturaOriginalV3(a, a.reloj)
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, err
	}
	return f.ContextoLecturaOriginal(ctx, d, aut)
}

var (
	_ almacenvec.AutorizacionesDocumentosOriginalCT   = (*autorizacionesOriginalDocumentosCT)(nil)
	_ docports.FabricaContextoLectura                 = (*autorizacionesOriginalDocumentosCT)(nil)
	_ docautorizacion.EmisorOperacionOriginalFirmable = (*emisorOriginalDocumentosCTLigado)(nil)
	_ docautorizacion.EmisorConcesionAlmacenV3        = (*emisorOriginalDocumentosCTLigado)(nil)
)
