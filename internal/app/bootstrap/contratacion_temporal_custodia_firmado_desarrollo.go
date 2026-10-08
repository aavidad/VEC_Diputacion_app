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
	"sync"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Custodia del PDF firmado de CT en Documentos (5.06), perfil de desarrollo.
// Se compone solo con el registro de firmas de CT, Documentos activo, el
// seudonimizador del almacén y la sección custodia_firmado del material de
// Documentos (qué documento del circuito va a qué tipo documental). Las dos
// autorizaciones de Documentos (la V3 que consume la SQL y la concesión del
// almacén) las da el PDP de CT a la identidad de la MISMA petición de firma,
// con el perfil publicado de la ruta de firma: ninguna otra ruta las obtiene,
// y solo para el documento y el PDF exactos que CT está custodiando.

const moduloProductorCustodiaCT = "contratacion_temporal"

var errCustodiaFirmadoCTDenegada = errors.New("bootstrap: custodia del documento firmado denegada")

// claveCustodiaFirmadoCTDesarrollo lleva en el contexto de la petición la
// custodia que CT ha pedido; el PDP solo concede para ella.
type claveCustodiaFirmadoCTDesarrollo struct{}

type esperadoCustodiaFirmadoCTDesarrollo struct {
	documentoRef, expedienteRef, huellaSHA256 string

	mu sync.Mutex
	// preimagenSHA256 y decisionRef los fija el autorizador al pedir la V3:
	// la concesión del almacén debe ligarse a esa decisión.
	preimagenSHA256, decisionRef string
	motivo                       dominiovec.ReferenciaEntradaCatalogo
}

func (e *esperadoCustodiaFirmadoCTDesarrollo) fijarMotivo(m dominiovec.ReferenciaEntradaCatalogo) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.motivo != (dominiovec.ReferenciaEntradaCatalogo{}) && e.motivo != m {
		return false
	}
	e.motivo = m
	return true
}

func (e *esperadoCustodiaFirmadoCTDesarrollo) motivoEsperado() dominiovec.ReferenciaEntradaCatalogo {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.motivo
}

func (e *esperadoCustodiaFirmadoCTDesarrollo) fijar(preimagen, decision string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if preimagen != "" {
		e.preimagenSHA256 = preimagen
	}
	if decision != "" {
		e.decisionRef = decision
	}
}

func (e *esperadoCustodiaFirmadoCTDesarrollo) leer() (string, string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.preimagenSHA256, e.decisionRef
}

func esperadoCustodiaDe(ctx context.Context) (*esperadoCustodiaFirmadoCTDesarrollo, bool) {
	if ctx == nil {
		return nil, false
	}
	e, ok := ctx.Value(claveCustodiaFirmadoCTDesarrollo{}).(*esperadoCustodiaFirmadoCTDesarrollo)
	return e, ok && e != nil
}

// solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida es el predicado del
// PDP de CT para documentos.firmado.custodiar en la ruta de firma: el recurso
// es el documento esperado y, o bien la preimagen que ya se validó (V3 de la
// SQL), o bien el PDF y la decisión esperados (concesión del almacén).
func solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx context.Context, datos dominiovec.DatosSolicitudAutorizacionLigadaV3) bool {
	e, ok := esperadoCustodiaDe(ctx)
	r := datos.Recurso
	if !ok || datos.Accion != docports.AccionCustodiarFirmado || datos.Finalidad != docports.FinalidadCustodiarFirmado ||
		e.motivoEsperado() == (dominiovec.ReferenciaEntradaCatalogo{}) || datos.ReferenciaMotivo != e.motivoEsperado() || r.Referencia != e.documentoRef ||
		r.ModuloID != "documentos" || r.Tipo != "documento_firmado" ||
		!maps.Equal(r.Ambitos, map[string]string{"organizacion_ref": docports.OrganizacionRefV3}) {
		return false
	}
	preimagen, decision := e.leer()
	if len(r.Atributos) == 1 {
		return preimagen != "" && r.Atributos["preimagen_sha256"] == preimagen
	}
	return decision != "" && r.Atributos[docautorizacion.AtributoDecisionCustodia] == decision &&
		r.Atributos["documento_huella_sha256"] == e.huellaSHA256 &&
		r.Atributos["documento_expediente_ref"] == e.expedienteRef &&
		r.Atributos["documento_modulo_productor"] == moduloProductorCustodiaCT
}

// concesionCustodiaFirmadoCTDesarrollo es el rol del perfil de firma de CT en
// Documentos: custodiar el PDF firmado, nada más.
func concesionCustodiaFirmadoCTDesarrollo() dominiovec.ConcesionRol {
	return dominiovec.ConcesionRol{Accion: docports.AccionCustodiarFirmado, ModuloID: "documentos", TipoRecurso: "documento_firmado",
		Finalidades: []string{docports.FinalidadCustodiarFirmado}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
		CamposPermitidos: []string{"documento_firmado.custodia", "evidencia_custodia"}}
}

// pdpCustodiaCTDesarrollo es lo que la custodia necesita del PDP de CT: la
// decisión registrada para la identidad de la petición de firma y el material
// V3 de la audiencia de Documentos. Lo implementa firmaDocumentoCTDesarrollo.
type pdpCustodiaCTDesarrollo interface {
	solicitarCustodiaV3(context.Context, dominiovec.RecursoAutorizable) (solicitudCustodiaCTDesarrollo, error)
	materialCustodiaV3(context.Context, solicitudCustodiaCTDesarrollo) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// custodiaFirmadoCTDesarrollo implementa el puerto de CT hacia Documentos y,
// para Documentos, el autorizador de la V3 y el emisor de la concesión del
// almacén, ambos con la identidad de la petición de firma de CT.
type custodiaFirmadoCTDesarrollo struct {
	pdp        pdpCustodiaCTDesarrollo
	documentos *custodiaDocumentosDesarrollo
	servicio   *docapp.Servicio
}

var (
	_ ports.CustodioDocumentoFirmado           = (*custodiaFirmadoCTDesarrollo)(nil)
	_ docports.AutorizadorCustodiaFirmado      = (*custodiaFirmadoCTDesarrollo)(nil)
	_ docautorizacion.EmisorConcesionAlmacenV3 = emisorConcesionCustodiaCTDesarrollo{}
)

// CustodiarFirmado traduce la orden de CT a Documentos y sus errores a los
// centinelas del puerto de CT, sin detalles internos.
func (c *custodiaFirmadoCTDesarrollo) CustodiarFirmado(ctx context.Context, o ports.OrdenCustodiaFirmado) (ports.DocumentoCustodiado, error) {
	var cero ports.DocumentoCustodiado
	if c == nil || c.servicio == nil || c.documentos == nil || ctx == nil {
		return cero, ports.ErrCustodiaFirmadoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	tipo, err := c.documentos.politicas.TipoDocumentalRef(o.TipoDocumental)
	if err != nil {
		return cero, ports.ErrCustodiaFirmadoInvalida
	}
	solicitud, err := c.documentos.politicas.SolicitudPara(o.TipoDocumental, o.ExpedienteRef)
	if err != nil {
		return cero, ports.ErrCustodiaFirmadoInvalida
	}
	suma := sha256.Sum256(o.Contenido)
	esperado := &esperadoCustodiaFirmadoCTDesarrollo{documentoRef: o.DocumentoRef, expedienteRef: o.ExpedienteRef,
		huellaSHA256: hex.EncodeToString(suma[:])}
	ctx = context.WithValue(ctx, claveCustodiaFirmadoCTDesarrollo{}, esperado)
	d, err := c.servicio.CustodiarFirmado(ctx, docports.CustodiaFirmado{
		ID: o.DocumentoRef, ClaveIdempotencia: o.ClaveIdempotencia, ModuloID: moduloProductorCustodiaCT,
		ExpedienteRef: o.ExpedienteRef, TipoRef: tipo, Version: o.Version, Contenido: o.Contenido,
		HuellaOriginalSHA256: o.HuellaOriginalSHA256, FirmaOperacionRef: o.FirmaOperacionRef, SolicitudPolitica: solicitud,
	}, c)
	if err != nil && ctx.Err() != nil {
		return cero, ctx.Err()
	}
	switch {
	case err == nil:
		return ports.DocumentoCustodiado{Ref: d.ID, Version: d.Version, HuellaSHA256: d.HuellaSHA256}, nil
	case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		return cero, err
	case errors.Is(err, docports.ErrSolicitudInvalida):
		return cero, ports.ErrCustodiaFirmadoInvalida
	case errors.Is(err, docports.ErrConflicto):
		return cero, ports.ErrCustodiaFirmadoEnConflicto
	case errors.Is(err, docports.ErrCapacidadNoDisponible):
		// La fábrica del almacén envuelve también los fallos de fuente en
		// autorización inválida: conservar la indisponibilidad explícita.
		return cero, ports.ErrCustodiaFirmadoNoDisponible
	case errors.Is(err, errCustodiaFirmadoCTDenegada), errors.Is(err, docports.ErrAccesoDenegado),
		errors.Is(err, puertosvec.ErrAutorizacionAlmacenInvalida):
		return cero, ports.ErrCustodiaFirmadoDenegada
	default:
		return cero, ports.ErrCustodiaFirmadoNoDisponible
	}
}

// AutorizarCustodiaFirmado pide al PDP de CT la V3 de documentos.firmado.
// custodiar para la preimagen que ha construido Documentos, tras comprobar
// que describe exactamente el documento y el PDF que CT pidió custodiar.
func (c *custodiaFirmadoCTDesarrollo) AutorizarCustodiaFirmado(ctx context.Context, preimagen []byte, documentoID, expedienteRef string) (docports.AutorizacionV3, error) {
	vacia := docports.AutorizacionV3{}
	e, ok := esperadoCustodiaDe(ctx)
	if c == nil || dependenciaEsNulaContratacionTemporalDesarrollo(c.pdp) || !ok ||
		documentoID != e.documentoRef || expedienteRef != e.expedienteRef || !preimagenCustodiaEsperada(preimagen, e) {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	recurso, err := docports.RecursoV3(docports.AccionCustodiarFirmado, documentoID, preimagen)
	if err != nil {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	suma := sha256.Sum256(preimagen)
	e.fijar(hex.EncodeToString(suma[:]), "")
	pedida, err := c.pdp.solicitarCustodiaV3(ctx, recurso)
	if err != nil {
		return vacia, err
	}
	material, err := c.pdp.materialCustodiaV3(ctx, pedida)
	if err != nil {
		return vacia, err
	}
	e.fijar("", material.ResumenCapacidad().DecisionRef())
	return docports.AutorizacionV3{Material: material, Accion: docports.AccionCustodiarFirmado, Finalidad: docports.FinalidadCustodiarFirmado,
		RecursoRef: documentoID, AmbitoRef: expedienteRef, PrincipalID: pedida.actor.PrincipalID,
		PerfilActivoRef: pedida.actor.PerfilActivoRef, CorrelacionRef: pedida.correlacion}, nil
}

// preimagenCustodiaEsperada comprueba en la preimagen de Documentos el
// documento, el expediente, el módulo y la huella del PDF pedidos por CT.
func preimagenCustodiaEsperada(preimagen []byte, e *esperadoCustodiaFirmadoCTDesarrollo) bool {
	var p struct {
		Accion     string `json:"accion"`
		ID         string `json:"id"`
		Modulo     string `json:"modulo_id"`
		Expediente string `json:"expediente_ref"`
		Huella     string `json:"huella_sha256"`
	}
	d := json.NewDecoder(bytes.NewReader(preimagen))
	return len(preimagen) > 0 && len(preimagen) <= 8192 && d.Decode(&p) == nil &&
		p.Accion == docports.AccionCustodiarFirmado && p.ID == e.documentoRef && p.Modulo == moduloProductorCustodiaCT &&
		p.Expediente == e.expedienteRef && p.Huella == e.huellaSHA256
}

type solicitudCustodiaCTDesarrollo struct {
	solicitud    dominiovec.SolicitudAutorizacionLigadaV3
	decision     dominiovec.DecisionAutorizacionLigadaV3
	confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	operativo    ports.ContextoAutorizacionAltaV3
	actor        dominiovec.DatosVinculoAutenticacionActorV2
	correlacion  string
}

// solicitarCustodiaV3 obtiene y registra en el PDP de CT una decisión para
// documentos.firmado.custodiar con la identidad de la petición de firma.
func (f *firmaDocumentoCTDesarrollo) solicitarCustodiaV3(ctx context.Context, recurso dominiovec.RecursoAutorizable) (solicitudCustodiaCTDesarrollo, error) {
	var vacia solicitudCustodiaCTDesarrollo
	if ctx == nil || f == nil || f.alta == nil || f.alta.soporte == nil || f.alta.autorizador == nil {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	s := f.alta.soporte
	capacidad, valida := s.capacidadValida(ctx)
	perfil := s.perfilFijoParaContexto(ctx, capacidad.ruta)
	if !valida || perfil == nil ||
		(capacidad.ruta == httpinterno.RutaFirmaDocumento && perfil.clave != clavePerfilFijoFirmaCTDesarrollo) ||
		(capacidad.ruta == httpinterno.RutaRegistroFirmaExterna &&
			(capacidad.metodo != http.MethodPost || perfil.clave != clavePerfilFijoFirmaExternaV2CTDesarrollo)) ||
		(capacidad.ruta != httpinterno.RutaFirmaDocumento && capacidad.ruta != httpinterno.RutaRegistroFirmaExterna) {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	motivo, motivoValido := s.motivoAutorizacionParaContexto(ctx, capacidad.ruta)
	if !motivoValido || (capacidad.ruta == httpinterno.RutaFirmaDocumento && motivo != motivoFirmaDocumentoCTDesarrollo()) ||
		(capacidad.ruta == httpinterno.RutaRegistroFirmaExterna && motivo != motivoFirmaV2CTDesarrollo()) {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	_, estadoPerfil := s.consumirPerfilFijoCTDesarrolloConEstado(ctx, perfil)
	if estadoPerfil == perfilFijoConsumoFuenteNoDisponible {
		return vacia, docports.ErrCapacidadNoDisponible
	}
	if estadoPerfil != perfilFijoConsumoVigente {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	operativo, err := s.contextoOperativoDesarrollo(ctx)
	if err != nil {
		if errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
			return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
		}
		return vacia, errCustodiaFirmadoCTDenegada
	}
	actor, err := operativo.Vinculo.Datos()
	if err != nil || !contextoRegistradoPerfilFijoCTDesarrollo(operativo.Resultado.Contexto, perfil) {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	e, ok := esperadoCustodiaDe(ctx)
	if !ok || !e.fijarMotivo(motivo) {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: motivo,
		Accion: docports.AccionCustodiarFirmado, Recurso: recurso, Finalidad: docports.FinalidadCustodiarFirmado, Correlacion: correlacion,
	}
	if !solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx, datos) {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := f.alta.autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if err != nil {
		if ctx.Err() != nil {
			return vacia, ctx.Err()
		}
		if errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) {
			_, estado := s.consumirPerfilFijoCTDesarrolloConEstado(ctx, perfil)
			if estado == perfilFijoConsumoDenegado {
				return vacia, errCustodiaFirmadoCTDenegada
			}
			return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
		}
		if errors.Is(err, errAutorizacionComunDesarrolloNoDisponible) ||
			errors.Is(err, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
			errors.Is(err, puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) {
			return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
		}
		if errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
			return vacia, errCustodiaFirmadoCTDenegada
		}
		return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	return solicitudCustodiaCTDesarrollo{solicitud: solicitud, decision: decision, confirmacion: confirmacion,
		operativo: operativo, actor: actor, correlacion: correlacionRef}, nil
}

// materialCustodiaV3 exporta el material V3 de la audiencia de Documentos
// para una decisión del PDP de CT y comprueba que queda ligado a ella.
func (f *firmaDocumentoCTDesarrollo) materialCustodiaV3(ctx context.Context, p solicitudCustodiaCTDesarrollo) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if f == nil || f.alta == nil || f.alta.postgresql.materialDocumentos == nil {
		return vacio, docports.ErrCapacidadNoDisponible
	}
	datos, err := p.solicitud.Datos()
	if err != nil {
		return vacio, docports.ErrCapacidadNoDisponible
	}
	motivo := datos.ReferenciaMotivo
	if motivo != motivoFirmaDocumentoCTDesarrollo() && motivo != motivoFirmaV2CTDesarrollo() {
		return vacio, errCustodiaFirmadoCTDenegada
	}
	material, err := f.alta.postgresql.materialDocumentos.proveerMaterialConfirmacion(ctx, p.solicitud, p.decision,
		p.confirmacion, motivo, p.operativo.Resultado)
	if err != nil || !puertosvec.MaterialAtestadoLigadoV3(p.solicitud, p.decision, p.confirmacion,
		p.operativo.Resultado, motivo, material, docports.AudienciaV3) {
		return vacio, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	return material, nil
}

// emisorConcesionCustodiaCTDesarrollo da a la fábrica de Documentos la
// concesión del almacén para custodiar, con la identidad de la petición.
type emisorConcesionCustodiaCTDesarrollo struct {
	pdp            pdpCustodiaCTDesarrollo
	seudonimizador *seudonimizadorAlmacenDesarrollo
}

func (e emisorConcesionCustodiaCTDesarrollo) SeudonimosLecturaOriginal(ctx context.Context, a docports.AutorizacionV3) (docautorizacion.DatosSeudonimosLectura, error) {
	if !e.seudonimizador.valido() || ctx == nil || ctx.Err() != nil || a.PrincipalID == "" || a.CorrelacionRef == "" || a.RecursoRef == "" {
		return docautorizacion.DatosSeudonimosLectura{}, errCustodiaFirmadoCTDenegada
	}
	return docautorizacion.DatosSeudonimosLectura{
		SujetoHMAC:    prefijoHMACSujetoAlmacenDesarrollo + e.seudonimizador.hmac("sujeto", a.PrincipalID),
		SolicitudHMAC: prefijoHMACSolicitudAlmacenDesarrollo + e.seudonimizador.hmac("solicitud", a.CorrelacionRef, a.RecursoRef),
	}, nil
}

func (e emisorConcesionCustodiaCTDesarrollo) EmitirConcesionAlmacenV3(ctx context.Context, s docautorizacion.SolicitudConcesionAlmacenV3) (docautorizacion.ConcesionAlmacenV3, error) {
	var vacia docautorizacion.ConcesionAlmacenV3
	if dependenciaEsNulaContratacionTemporalDesarrollo(e.pdp) || s.Accion != docports.AccionCustodiarFirmado || s.Finalidad != docports.FinalidadCustodiarFirmado {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	pedida, err := e.pdp.solicitarCustodiaV3(ctx, s.Recurso)
	if err != nil {
		return vacia, err
	}
	if pedida.actor.PrincipalID != s.PrincipalID || pedida.actor.PerfilActivoRef != s.PerfilActivoRef {
		return vacia, errCustodiaFirmadoCTDenegada
	}
	return docautorizacion.ConcesionAlmacenV3{Solicitud: pedida.solicitud, Decision: pedida.decision, Confirmacion: pedida.confirmacion}, nil
}

// componerCustodia enlaza la firma de CT con Documentos cuando ambos están
// compuestos y el material de Documentos declara qué documentos se custodian.
func (f *firmaDocumentoCTDesarrollo) componerCustodia(d *autoridadDocumentosDesarrollo) error {
	if f == nil {
		return nil
	}
	if d == nil || d.custodia == nil {
		if f.alta != nil && f.alta.soporte != nil && len(f.alta.soporte.custodiaFirmaDocumentos) != 0 {
			return errFirmaDocumentoCTDesarrolloNoDisponible
		}
		return nil
	}
	if f.servicio == nil || f.alta == nil || f.alta.soporte == nil ||
		!maps.Equal(f.alta.soporte.custodiaFirmaDocumentos, d.custodia.documentos) || len(f.alta.soporte.custodiaFirmaDocumentos) == 0 {
		return errFirmaDocumentoCTDesarrolloNoDisponible
	}
	custodia, err := nuevaCustodiaFirmadoCTDesarrollo(f, d.custodia, d.reloj)
	if err != nil {
		return errFirmaDocumentoCTDesarrolloNoDisponible
	}
	if err := f.servicio.ComponerCustodia(custodia, d.custodia.documentos); err != nil {
		return errFirmaDocumentoCTDesarrolloNoDisponible
	}
	f.custodiaR5Compuesta = true
	return nil
}

// nuevaCustodiaFirmadoCTDesarrollo une el PDP de CT con el servicio de
// custodia de Documentos (fábrica de concesiones con el mismo PDP).
func nuevaCustodiaFirmadoCTDesarrollo(pdp pdpCustodiaCTDesarrollo, d *custodiaDocumentosDesarrollo, reloj puertosvec.Reloj) (*custodiaFirmadoCTDesarrollo, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(pdp) || d == nil || !d.seudonimizador.valido() {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	fabrica, err := docautorizacion.NuevaFabricaContextoCustodiaFirmadoV3(
		emisorConcesionCustodiaCTDesarrollo{pdp: pdp, seudonimizador: d.seudonimizador}, reloj)
	if err != nil {
		return nil, errFirmaDocumentoCTDesarrolloNoDisponible
	}
	return &custodiaFirmadoCTDesarrollo{pdp: pdp, documentos: d, servicio: d.servicio(fabrica)}, nil
}
