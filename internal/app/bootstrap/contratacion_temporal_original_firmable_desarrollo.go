package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"sync"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/almacen"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Original firmable de CT en Documentos (corte 4c-6a), perfil de desarrollo.
//
// La custodia común (almacen.CustodiaDocumentosOriginalCT) necesita, para
// cada lectura o alta del original, decisiones V3 de Documentos para el
// recurso exacto. Aquí las emite el PDP de CT con la identidad de la MISMA
// petición y el perfil fijo propio de la ruta del original, como la custodia
// del PDF firmado: nadie aporta actor, perfil ni capacidad desde la petición.
//
// Cuatro decisiones, todas de la audiencia vec_documentos.operacion.v1 o del
// almacén, y cada una ligada a lo que CT pidió en esta petición:
//   - documentos.original.descargar (SQL y concesión de lectura del objeto),
//   - documentos.original_firmable.reservar y .confirmar (AD158, Documentos13),
//   - documentos.original_firmable.almacen.escribir (objeto del intento).
//
// El predicado del PDP solo concede si el contexto lleva el original que se
// está leyendo o custodiando (esperadoOriginalFirmableCTDesarrollo) y el
// recurso coincide con él: documento, versión, expediente documental, huella
// del PDF, preimagen y decisión previa. Sin ese contexto, deniega.

const (
	clavePerfilFijoOriginalFirmableCTDesarrollo = "original_firmable_ct"
	tipoRecursoOriginalFirmableCT               = "documento_original_firmable"
	tipoRecursoDescargaOriginalCT               = "documento_original"
	moduloRecursoDocumentosCT                   = "documentos"
	maximoPreimagenOriginalFirmableCT           = 16384
)

var errOriginalFirmableCTDenegado = fmt.Errorf("%w: original firmable de contratación temporal", docports.ErrAccesoDenegado)

// rutaOriginalFirmableCTDesarrollo enumera las rutas que pueden pedir estas
// decisiones. Hoy solo la del original; la firma R5 (4c-5/4c-8) la ampliará
// con su propio perfil cuando sus rutas existan.
func rutaOriginalFirmableCTDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaOriginalFirmableCT
}

func motivoOriginalFirmableCTDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_original_firmable_ct", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("original-firmable-ct-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "original-firmable-ct"),
	}
}

// motivoOriginalFirmableCTAdmitido: el PDP valida además que el motivo sea el
// de la ruta sellada. Las rutas de firma R5 añadirán aquí el suyo.
func motivoOriginalFirmableCTAdmitido(m dominiovec.ReferenciaEntradaCatalogo) bool {
	return m == motivoOriginalFirmableCTDesarrollo()
}

// concesionesOriginalFirmableCTDesarrollo son las del rol del perfil fijo.
// Los campos son los que exigen AD3-60 (descarga), AD3-158 (reserva y
// confirmación) y el plan de escritura del almacén; sin obligaciones.
func concesionesOriginalFirmableCTDesarrollo() []dominiovec.ConcesionRol {
	documentos := func(accion, tipo, finalidad string, campos ...string) dominiovec.ConcesionRol {
		return dominiovec.ConcesionRol{Accion: accion, ModuloID: moduloRecursoDocumentosCT, TipoRecurso: tipo,
			Finalidades: []string{finalidad}, GarantiaMinima: dominiovec.AuthAssuranceHigh, CamposPermitidos: campos}
	}
	return []dominiovec.ConcesionRol{
		documentos(docports.AccionDescargar, tipoRecursoDescargaOriginalCT, finalidadDescargaDocumento, "contenido", "documento"),
		documentos(docports.AccionReservarOriginalFirmable, tipoRecursoOriginalFirmableCT, docports.FinalidadOriginalFirmable, "intento", "reserva"),
		documentos(docports.AccionConfirmarOriginalFirmable, tipoRecursoOriginalFirmableCT, docports.FinalidadOriginalFirmable, "documento", "recibo"),
		documentos(puertosvec.AccionNegocioEscribirOriginalFirmable, tipoRecursoOriginalFirmableCT, docports.FinalidadOriginalFirmable,
			"evidencia_almacen", "original_firmable.contenido"),
	}
}

func instantaneaOriginalFirmableCTDesarrollo(principalID, perfilRef string, ahora time.Time) (dominiovec.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principalID, perfilRef, ahora,
		"original_firmable_ct_desarrollo", "Lectura y custodia del original firmable CT de desarrollo",
		"original-firmable-ct-desarrollo-no-autoritativa", concesionesOriginalFirmableCTDesarrollo(),
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
}

// claveOriginalFirmableCTDesarrollo lleva en el contexto el original que CT
// lee o custodia en esta petición; el PDP solo concede para él.
type claveOriginalFirmableCTDesarrollo struct{}

type esperadoOriginalFirmableCTDesarrollo struct {
	documentoRef, expedienteRef string
	version                     uint64
	// Solo en una custodia: lo que CT renderizó y pidió guardar.
	escritura                                  bool
	claveIdempotencia, tipoRef, huellaSHA256   string
	tamano                                     int64
	decisionDescarga, correlacionDescarga      string // lectura del objeto: la descarga ya consumida
	mu                                         sync.Mutex
	preimagenes                                map[string]string // acción -> SHA-256 de la preimagen admitida
	decisionReserva                            string
	reservaRef, claveAlmacenRef, intentoNumero string
}

func (e *esperadoOriginalFirmableCTDesarrollo) fijarPreimagen(accion string, preimagen []byte) {
	suma := sha256.Sum256(preimagen)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.preimagenes == nil {
		e.preimagenes = map[string]string{}
	}
	e.preimagenes[accion] = hex.EncodeToString(suma[:])
}

func (e *esperadoOriginalFirmableCTDesarrollo) preimagen(accion string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.preimagenes[accion]
}

func (e *esperadoOriginalFirmableCTDesarrollo) fijarDecisionReserva(decision string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.decisionReserva = decision
}

func (e *esperadoOriginalFirmableCTDesarrollo) fijarIntento(reserva, numero, clave string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.reservaRef, e.intentoNumero, e.claveAlmacenRef = reserva, numero, clave
}

func (e *esperadoOriginalFirmableCTDesarrollo) leerReserva() (decision, reserva, numero, clave string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.decisionReserva, e.reservaRef, e.intentoNumero, e.claveAlmacenRef
}

func esperadoOriginalFirmableDe(ctx context.Context) (*esperadoOriginalFirmableCTDesarrollo, bool) {
	if ctx == nil {
		return nil, false
	}
	e, ok := ctx.Value(claveOriginalFirmableCTDesarrollo{}).(*esperadoOriginalFirmableCTDesarrollo)
	return e, ok && e != nil
}

// solicitudAutorizacionOriginalFirmableCTDesarrolloValida es el predicado del
// PDP de CT para las rutas del original: acción, finalidad, motivo, módulo,
// tipo, documento, ámbito y atributos exactos del original en curso.
func solicitudAutorizacionOriginalFirmableCTDesarrolloValida(ctx context.Context, datos dominiovec.DatosSolicitudAutorizacionLigadaV3) bool {
	e, ok := esperadoOriginalFirmableDe(ctx)
	r := datos.Recurso
	if !ok || !motivoOriginalFirmableCTAdmitido(datos.ReferenciaMotivo) || r.ModuloID != moduloRecursoDocumentosCT ||
		r.Referencia != e.documentoRef || !maps.Equal(r.Ambitos, map[string]string{"organizacion_ref": docports.OrganizacionRefV3}) {
		return false
	}
	a := r.Atributos
	switch datos.Accion {
	case docports.AccionDescargar:
		if datos.Finalidad != finalidadDescargaDocumento || r.Tipo != tipoRecursoDescargaOriginalCT {
			return false
		}
		if len(a) == 1 {
			p := e.preimagen(docports.AccionDescargar)
			return p != "" && a["preimagen_sha256"] == p
		}
		// Concesión de lectura del objeto: la descarga SQL de esta petición.
		return e.decisionDescarga != "" && e.correlacionDescarga != "" &&
			a[docautorizacion.AtributoDecisionDescarga] == e.decisionDescarga &&
			a[docautorizacion.AtributoCorrelacionDescarga] == e.correlacionDescarga &&
			a["documento_expediente_ref"] == e.expedienteRef &&
			a["documento_modulo_productor"] == moduloProductorCustodiaCT &&
			a["documento_version"] == strconv.FormatUint(e.version, 10)
	case docports.AccionReservarOriginalFirmable, docports.AccionConfirmarOriginalFirmable:
		p := e.preimagen(datos.Accion)
		return e.escritura && datos.Finalidad == docports.FinalidadOriginalFirmable && r.Tipo == tipoRecursoOriginalFirmableCT &&
			len(a) == 1 && p != "" && a["preimagen_sha256"] == p
	case puertosvec.AccionNegocioEscribirOriginalFirmable:
		decision, _, _, _ := e.leerReserva()
		return e.escritura && datos.Finalidad == docports.FinalidadOriginalFirmable && r.Tipo == tipoRecursoOriginalFirmableCT &&
			decision != "" && a["documentos_original_decision_reserva_ref"] == decision &&
			a["documentos_original_huella_sha256"] == e.huellaSHA256 &&
			a["documentos_original_expediente_ref"] == e.expedienteRef &&
			a["documentos_original_tipo_ref"] == e.tipoRef &&
			a["documentos_original_version"] == strconv.FormatUint(e.version, 10) &&
			a[puertosvec.AtributoAlmacenEfectoRef] == e.documentoRef
	default:
		return false
	}
}

// pdpOriginalFirmableCTDesarrollo es lo que el original necesita del PDP de
// CT: una decisión registrada para la identidad de la petición y el material
// V3 de la audiencia de Documentos.
type pdpOriginalFirmableCTDesarrollo interface {
	solicitarOriginalV3(context.Context, string, string, dominiovec.RecursoAutorizable) (solicitudCustodiaCTDesarrollo, error)
	materialOriginalV3(context.Context, solicitudCustodiaCTDesarrollo) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// pdpCTOriginalFirmableDesarrollo implementa el puerto anterior con el PDP y
// el material de Documentos que ya compone el alta de CT.
type pdpCTOriginalFirmableDesarrollo struct {
	alta *dependenciasAltaContratacionTemporalDesarrollo
}

// solicitarOriginalV3 consume el perfil fijo de la ruta sellada y pide al PDP
// la decisión con la sesión registrada de ese perfil. Una caída de la fuente
// es indisponibilidad; una revocación o un recurso ajeno, denegación.
func (p pdpCTOriginalFirmableDesarrollo) solicitarOriginalV3(ctx context.Context, accion, finalidad string, recurso dominiovec.RecursoAutorizable) (solicitudCustodiaCTDesarrollo, error) {
	var vacia solicitudCustodiaCTDesarrollo
	if ctx == nil || p.alta == nil || p.alta.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(p.alta.autorizador) {
		return vacia, errOriginalFirmableCTDenegado
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	s := p.alta.soporte
	capacidad, valida := s.capacidadValida(ctx)
	perfil := s.perfilFijoParaContexto(ctx, capacidad.ruta)
	if !valida || !rutaOriginalFirmableCTDesarrollo(capacidad.ruta) || perfil == nil {
		return vacia, errOriginalFirmableCTDenegado
	}
	motivo, motivoValido := s.motivoAutorizacionParaContexto(ctx, capacidad.ruta)
	if !motivoValido || !motivoOriginalFirmableCTAdmitido(motivo) {
		return vacia, errOriginalFirmableCTDenegado
	}
	_, estadoPerfil := s.consumirPerfilFijoCTDesarrolloConEstado(ctx, perfil)
	if estadoPerfil == perfilFijoConsumoFuenteNoDisponible {
		return vacia, docports.ErrCapacidadNoDisponible
	}
	if estadoPerfil != perfilFijoConsumoVigente {
		return vacia, errOriginalFirmableCTDenegado
	}
	operativo, err := s.contextoOperativoDesarrollo(ctx)
	if err != nil {
		if errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
			return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
		}
		return vacia, errOriginalFirmableCTDenegado
	}
	actor, err := operativo.Vinculo.Datos()
	if err != nil || !contextoRegistradoPerfilFijoCTDesarrollo(operativo.Resultado.Contexto, perfil) {
		return vacia, errOriginalFirmableCTDenegado
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: motivo,
		Accion: accion, Recurso: recurso, Finalidad: finalidad, Correlacion: correlacion,
	}
	if !solicitudAutorizacionOriginalFirmableCTDesarrolloValida(ctx, datos) {
		return vacia, errOriginalFirmableCTDenegado
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return vacia, errOriginalFirmableCTDenegado
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := p.alta.autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if err != nil {
		if ctx.Err() != nil {
			return vacia, ctx.Err()
		}
		if errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) {
			// La fuente cae justo cuando el perfil se revoca: manda la revocación.
			if _, estado := s.consumirPerfilFijoCTDesarrolloConEstado(ctx, perfil); estado == perfilFijoConsumoDenegado {
				return vacia, errOriginalFirmableCTDenegado
			}
			return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
		}
		if errors.Is(err, errAutorizacionComunDesarrolloNoDisponible) ||
			errors.Is(err, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
			errors.Is(err, puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) {
			return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
		}
		if errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
			return vacia, errOriginalFirmableCTDenegado
		}
		return vacia, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	return solicitudCustodiaCTDesarrollo{solicitud: solicitud, decision: decision, confirmacion: confirmacion,
		operativo: operativo, actor: actor, correlacion: correlacionRef}, nil
}

func (p pdpCTOriginalFirmableDesarrollo) materialOriginalV3(ctx context.Context, s solicitudCustodiaCTDesarrollo) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p.alta == nil || p.alta.postgresql.materialDocumentos == nil {
		return vacio, docports.ErrCapacidadNoDisponible
	}
	datos, err := s.solicitud.Datos()
	if err != nil {
		return vacio, docports.ErrCapacidadNoDisponible
	}
	motivo := datos.ReferenciaMotivo
	material, err := p.alta.postgresql.materialDocumentos.proveerMaterialConfirmacion(ctx, s.solicitud, s.decision,
		s.confirmacion, motivo, s.operativo.Resultado)
	if err != nil || !puertosvec.MaterialAtestadoLigadoV3(s.solicitud, s.decision, s.confirmacion,
		s.operativo.Resultado, motivo, material, docports.AudienciaV3) {
		return vacio, errors.Join(docports.ErrCapacidadNoDisponible, err)
	}
	return material, nil
}

// originalFirmableCTDesarrollo implementa las autorizaciones que pide la
// custodia común del original y los dos emisores de Documentos (operación y
// concesión del almacén), todos con el mismo PDP de CT.
type originalFirmableCTDesarrollo struct {
	pdp            pdpOriginalFirmableCTDesarrollo
	seudonimizador *seudonimizadorAlmacenDesarrollo
	politicas      *conservacion.Catalogo
	autoridad      *docautorizacion.AutoridadOriginalFirmableV3
	// mapear es la única traducción del expediente CT al documental; la
	// composición entrega este mismo mapeador a la custodia común.
	mapear almacen.MapeadorExpedienteOriginalCT
}

var (
	_ almacen.AutorizacionesDocumentosOriginalCT      = (*originalFirmableCTDesarrollo)(nil)
	_ docautorizacion.EmisorOperacionOriginalFirmable = (*originalFirmableCTDesarrollo)(nil)
	_ docautorizacion.EmisorConcesionAlmacenV3        = (*originalFirmableCTDesarrollo)(nil)
	_ docports.AutorizarOriginalFirmable              = autoridadOriginalFirmablePeticionCTDesarrollo{}
	_ almacen.ServicioDocumentosOriginalCT            = servicioDocumentosOriginalCTDesarrollo{}
)

func nuevoOriginalFirmableCTDesarrollo(pdp pdpOriginalFirmableCTDesarrollo, seudonimizador *seudonimizadorAlmacenDesarrollo,
	politicas *conservacion.Catalogo, reloj puertosvec.Reloj,
) (*originalFirmableCTDesarrollo, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(pdp) || !seudonimizador.valido() || politicas == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(reloj) {
		return nil, puertosvec.ErrOriginalFirmableCTNoDisponible
	}
	o := &originalFirmableCTDesarrollo{pdp: pdp, seudonimizador: seudonimizador, politicas: politicas,
		mapear: almacen.FuncionMapeoExpedienteOriginalCT(ctapplication.ReferenciaExpedienteDocumentalFormalizacion)}
	autoridad, err := docautorizacion.NuevaAutoridadOriginalFirmableV3(o, o, reloj)
	if err != nil {
		return nil, puertosvec.ErrOriginalFirmableCTNoDisponible
	}
	o.autoridad = autoridad
	return o, nil
}

// identidadYExpedienteOriginalCT valida la solicitud CT y deriva la
// referencia del original y el expediente documental (opaco) de CT.
func (o *originalFirmableCTDesarrollo) identidadYExpedienteOriginalCT(s puertosvec.SolicitudOriginalFirmableCT, ref string) (almacencanonico.IdentidadOriginalCT, string, error) {
	identidad := almacencanonico.IdentidadOriginalCT{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		Documento: s.Documento, Version: s.OriginalVersion}
	if !identidad.Valida() || ref != identidad.Referencia() || (s.OriginalRef != "" && s.OriginalRef != ref) ||
		s.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo || o.mapear == nil {
		return identidad, "", puertosvec.ErrOriginalFirmableCTInvalido
	}
	expediente, err := o.mapear.ReferenciaDocumentalCT(s.ExpedienteRef)
	if err != nil {
		return identidad, "", errors.Join(puertosvec.ErrOriginalFirmableCTInvalido, err)
	}
	if !docdomain.ReferenciaOpacaValida(expediente) {
		return identidad, "", puertosvec.ErrOriginalFirmableCTInvalido
	}
	return identidad, expediente, nil
}

// AutorizarLecturaOriginalCT pide la V3 de documentos.original.descargar para
// el original exacto de la solicitud CT y su versión.
func (o *originalFirmableCTDesarrollo) AutorizarLecturaOriginalCT(ctx context.Context, s puertosvec.SolicitudOriginalFirmableCT, ref string) (docports.ConsultaDocumento, error) {
	var vacia docports.ConsultaDocumento
	if o == nil || dependenciaEsNulaContratacionTemporalDesarrollo(o.pdp) || ctx == nil {
		return vacia, puertosvec.ErrOriginalFirmableCTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	_, expediente, err := o.identidadYExpedienteOriginalCT(s, ref)
	if err != nil {
		return vacia, err
	}
	consulta := docports.ConsultaDocumento{DocumentoID: ref, Version: s.OriginalVersion}
	preimagen, err := consulta.PreimagenDescargar()
	if err != nil {
		return vacia, puertosvec.ErrOriginalFirmableCTInvalido
	}
	recurso, err := docports.RecursoV3(docports.AccionDescargar, ref, preimagen)
	if err != nil {
		return vacia, puertosvec.ErrOriginalFirmableCTInvalido
	}
	e := &esperadoOriginalFirmableCTDesarrollo{documentoRef: ref, expedienteRef: expediente, version: s.OriginalVersion}
	e.fijarPreimagen(docports.AccionDescargar, preimagen)
	ctx = context.WithValue(ctx, claveOriginalFirmableCTDesarrollo{}, e)
	pedida, err := o.pdp.solicitarOriginalV3(ctx, docports.AccionDescargar, finalidadDescargaDocumento, recurso)
	if err != nil {
		return vacia, err
	}
	material, err := o.pdp.materialOriginalV3(ctx, pedida)
	if err != nil {
		return vacia, err
	}
	consulta.Autorizacion = docports.AutorizacionV3{Material: material, Accion: docports.AccionDescargar, Finalidad: finalidadDescargaDocumento,
		RecursoRef: ref, AmbitoRef: expediente, PrincipalID: pedida.actor.PrincipalID,
		PerfilActivoRef: pedida.actor.PerfilActivoRef, CorrelacionRef: pedida.correlacion}
	return consulta, nil
}

// PrepararCustodiaOriginalCT construye la orden de alta con el tipo y la
// política del catálogo gobernado y una autoridad ligada a este PDF: las
// decisiones posteriores solo se conceden para él.
func (o *originalFirmableCTDesarrollo) PrepararCustodiaOriginalCT(ctx context.Context, s puertosvec.SolicitudOriginalFirmableCT, pdf puertosvec.PDFOriginalCT, ref string) (docports.OrdenCustodiarOriginalFirmable, docports.AutorizarOriginalFirmable, error) {
	var vacia docports.OrdenCustodiarOriginalFirmable
	if o == nil || o.autoridad == nil || o.politicas == nil || ctx == nil {
		return vacia, nil, puertosvec.ErrOriginalFirmableCTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, nil, err
	}
	identidad, expediente, err := o.identidadYExpedienteOriginalCT(s, ref)
	if err != nil {
		return vacia, nil, err
	}
	if len(pdf.Contenido) < 8 || len(pdf.Contenido) > puertosvec.LimiteOriginalFirmableCT ||
		!bytes.HasPrefix(pdf.Contenido, []byte("%PDF-")) {
		return vacia, nil, puertosvec.ErrOriginalFirmableCTInvalido
	}
	clave, existe := o.politicas.ClaveTipo(pdf.TipoRef)
	if !existe || !o.politicas.CustodiaOriginalCTReservada(pdf.TipoRef) {
		return vacia, nil, puertosvec.ErrOriginalFirmableCTNoDisponible
	}
	politica, err := o.politicas.SolicitudPara(clave, expediente)
	if err != nil {
		return vacia, nil, puertosvec.ErrOriginalFirmableCTNoDisponible
	}
	contenido := bytes.Clone(pdf.Contenido)
	suma := sha256.Sum256(contenido)
	e := &esperadoOriginalFirmableCTDesarrollo{documentoRef: ref, expedienteRef: expediente, version: s.OriginalVersion,
		escritura: true, claveIdempotencia: identidad.ClaveLogica(), tipoRef: pdf.TipoRef,
		huellaSHA256: hex.EncodeToString(suma[:]), tamano: int64(len(contenido))}
	orden := docports.OrdenCustodiarOriginalFirmable{ID: ref, ClaveIdempotencia: e.claveIdempotencia, ModuloID: moduloProductorCustodiaCT,
		ExpedienteRef: expediente, TipoRef: pdf.TipoRef, Version: s.OriginalVersion, MIME: "application/pdf",
		Contenido: contenido, SolicitudPolitica: politica}
	return orden, autoridadOriginalFirmablePeticionCTDesarrollo{esperado: e, delegada: o.autoridad}, nil
}

// autoridadOriginalFirmablePeticionCTDesarrollo lleva al contexto el original
// de esta custodia antes de cada paso de la autoridad V3 de Documentos.
type autoridadOriginalFirmablePeticionCTDesarrollo struct {
	esperado *esperadoOriginalFirmableCTDesarrollo
	delegada *docautorizacion.AutoridadOriginalFirmableV3
}

func (a autoridadOriginalFirmablePeticionCTDesarrollo) contexto(ctx context.Context) (context.Context, bool) {
	if ctx == nil || a.esperado == nil || a.delegada == nil {
		return ctx, false
	}
	return context.WithValue(ctx, claveOriginalFirmableCTDesarrollo{}, a.esperado), true
}

func (a autoridadOriginalFirmablePeticionCTDesarrollo) AutorizarReservaOriginal(ctx context.Context, preimagen []byte, id, expediente string) (docports.AutorizacionV3, error) {
	ctx, ok := a.contexto(ctx)
	if !ok {
		return docports.AutorizacionV3{}, errOriginalFirmableCTDenegado
	}
	return a.delegada.AutorizarReservaOriginal(ctx, preimagen, id, expediente)
}

func (a autoridadOriginalFirmablePeticionCTDesarrollo) AutorizarConfirmacionOriginal(ctx context.Context, preimagen []byte, id, expediente string) (docports.AutorizacionV3, error) {
	ctx, ok := a.contexto(ctx)
	if !ok {
		return docports.AutorizacionV3{}, errOriginalFirmableCTDenegado
	}
	return a.delegada.AutorizarConfirmacionOriginal(ctx, preimagen, id, expediente)
}

func (a autoridadOriginalFirmablePeticionCTDesarrollo) ContextoEscrituraOriginal(ctx context.Context, r docports.ReservaOriginalFirmable, i docports.IntentoOriginalFirmable) (puertosvec.ContextoOperacionAlmacen, error) {
	ctx, ok := a.contexto(ctx)
	if !ok {
		return puertosvec.ContextoOperacionAlmacen{}, errOriginalFirmableCTDenegado
	}
	return a.delegada.ContextoEscrituraOriginal(ctx, r, i)
}

// AutorizarOperacionOriginalFirmable emite reserva o confirmación tras
// comprobar que la preimagen de Documentos describe el original en curso.
func (o *originalFirmableCTDesarrollo) AutorizarOperacionOriginalFirmable(ctx context.Context, accion string, preimagen []byte, id, expediente string) (docports.AutorizacionV3, error) {
	vacia := docports.AutorizacionV3{}
	e, ok := esperadoOriginalFirmableDe(ctx)
	if o == nil || dependenciaEsNulaContratacionTemporalDesarrollo(o.pdp) || !ok || !e.escritura ||
		id != e.documentoRef || expediente != e.expedienteRef || !preimagenOriginalFirmableEsperada(accion, preimagen, e) {
		return vacia, errOriginalFirmableCTDenegado
	}
	suma := sha256.Sum256(preimagen)
	recurso := dominiovec.RecursoAutorizable{Referencia: id, ModuloID: moduloRecursoDocumentosCT, Tipo: tipoRecursoOriginalFirmableCT,
		Ambitos:   map[string]string{"organizacion_ref": docports.OrganizacionRefV3},
		Atributos: map[string]string{"preimagen_sha256": hex.EncodeToString(suma[:])}}
	if recurso.Validar() != nil {
		return vacia, errOriginalFirmableCTDenegado
	}
	e.fijarPreimagen(accion, preimagen)
	pedida, err := o.pdp.solicitarOriginalV3(ctx, accion, docports.FinalidadOriginalFirmable, recurso)
	if err != nil {
		return vacia, err
	}
	material, err := o.pdp.materialOriginalV3(ctx, pedida)
	if err != nil {
		return vacia, err
	}
	if accion == docports.AccionReservarOriginalFirmable {
		e.fijarDecisionReserva(material.ResumenCapacidad().DecisionRef())
	}
	return docports.AutorizacionV3{Material: material, Accion: accion, Finalidad: docports.FinalidadOriginalFirmable,
		RecursoRef: id, AmbitoRef: expediente, PrincipalID: pedida.actor.PrincipalID,
		PerfilActivoRef: pedida.actor.PerfilActivoRef, CorrelacionRef: pedida.correlacion}, nil
}

// preimagenOriginalFirmableEsperada coteja la preimagen canónica de
// Documentos con el original en curso: en la reserva, identidad, tipo, versión
// y PDF; en la confirmación, además, el intento cuya escritura se concedió.
func preimagenOriginalFirmableEsperada(accion string, preimagen []byte, e *esperadoOriginalFirmableCTDesarrollo) bool {
	if len(preimagen) == 0 || len(preimagen) > maximoPreimagenOriginalFirmableCT {
		return false
	}
	var p struct {
		Accion     string `json:"accion"`
		ID         string `json:"id"`
		Clave      string `json:"clave_idempotencia"`
		Modulo     string `json:"modulo_id"`
		Expediente string `json:"expediente_ref"`
		Tipo       string `json:"tipo_ref"`
		Version    uint64 `json:"version"`
		MIME       string `json:"mime"`
		Tamano     int64  `json:"tamano"`
		Huella     string `json:"huella_sha256"`
		Reserva    string `json:"reserva_ref"`
		Intento    uint64 `json:"intento_num"`
		Almacen    string `json:"clave_almacen_ref"`
		Objeto     struct {
			Huella string `json:"huella_sha256"`
		} `json:"objeto"`
	}
	if json.NewDecoder(bytes.NewReader(preimagen)).Decode(&p) != nil || p.Accion != accion || p.ID != e.documentoRef || p.Huella != e.huellaSHA256 {
		return false
	}
	switch accion {
	case docports.AccionReservarOriginalFirmable:
		return p.Clave == e.claveIdempotencia && p.Modulo == moduloProductorCustodiaCT && p.Expediente == e.expedienteRef &&
			p.Tipo == e.tipoRef && p.Version == e.version && p.MIME == "application/pdf" && p.Tamano == e.tamano
	case docports.AccionConfirmarOriginalFirmable:
		_, reserva, numero, clave := e.leerReserva()
		return reserva != "" && p.Reserva == reserva && strconv.FormatUint(p.Intento, 10) == numero &&
			p.Almacen == clave && p.Objeto.Huella == e.huellaSHA256
	default:
		return false
	}
}

func (o *originalFirmableCTDesarrollo) SeudonimosLecturaOriginal(ctx context.Context, a docports.AutorizacionV3) (docautorizacion.DatosSeudonimosLectura, error) {
	if o == nil || !o.seudonimizador.valido() || ctx == nil || ctx.Err() != nil || a.PrincipalID == "" || a.CorrelacionRef == "" || a.RecursoRef == "" {
		return docautorizacion.DatosSeudonimosLectura{}, errOriginalFirmableCTDenegado
	}
	return docautorizacion.DatosSeudonimosLectura{
		SujetoHMAC:    prefijoHMACSujetoAlmacenDesarrollo + o.seudonimizador.hmac("sujeto", a.PrincipalID),
		SolicitudHMAC: prefijoHMACSolicitudAlmacenDesarrollo + o.seudonimizador.hmac("solicitud", a.CorrelacionRef, a.RecursoRef),
	}, nil
}

// EmitirConcesionAlmacenV3 da la concesión del almacén para leer el objeto de
// una descarga ya consumida o escribir el del intento reservado, con la
// identidad de la misma petición.
func (o *originalFirmableCTDesarrollo) EmitirConcesionAlmacenV3(ctx context.Context, s docautorizacion.SolicitudConcesionAlmacenV3) (docautorizacion.ConcesionAlmacenV3, error) {
	var vacia docautorizacion.ConcesionAlmacenV3
	e, ok := esperadoOriginalFirmableDe(ctx)
	if o == nil || dependenciaEsNulaContratacionTemporalDesarrollo(o.pdp) || !ok {
		return vacia, errOriginalFirmableCTDenegado
	}
	switch {
	case s.Accion == puertosvec.AccionNegocioLeerOriginalDocumentoGenerado && s.Finalidad == finalidadDescargaDocumento && !e.escritura:
	case s.Accion == puertosvec.AccionNegocioEscribirOriginalFirmable && s.Finalidad == docports.FinalidadOriginalFirmable && e.escritura:
		a := s.Recurso.Atributos
		if _, err := strconv.ParseUint(a["documentos_original_intento_num"], 10, 53); err != nil ||
			!docdomain.ReferenciaOpacaValida(a["documentos_original_reserva_ref"]) ||
			!docdomain.ReferenciaOpacaValida(a["documentos_original_clave_almacen_ref"]) {
			return vacia, errOriginalFirmableCTDenegado
		}
	default:
		return vacia, errOriginalFirmableCTDenegado
	}
	pedida, err := o.pdp.solicitarOriginalV3(ctx, s.Accion, s.Finalidad, s.Recurso)
	if err != nil {
		return vacia, err
	}
	if pedida.actor.PrincipalID != s.PrincipalID || pedida.actor.PerfilActivoRef != s.PerfilActivoRef {
		return vacia, errOriginalFirmableCTDenegado
	}
	if e.escritura {
		// La confirmación posterior solo se admite para el intento concedido.
		a := s.Recurso.Atributos
		e.fijarIntento(a["documentos_original_reserva_ref"], a["documentos_original_intento_num"], a["documentos_original_clave_almacen_ref"])
	}
	return docautorizacion.ConcesionAlmacenV3{Solicitud: pedida.solicitud, Decision: pedida.decision, Confirmacion: pedida.confirmacion}, nil
}

// servicioDocumentosOriginalCTDesarrollo es el servicio de Documentos que ve
// la custodia común. Para leer, lleva al contexto la descarga autorizada,
// que es lo único que la concesión de lectura del objeto puede cubrir.
type servicioDocumentosOriginalCTDesarrollo struct {
	servicio *docapp.Servicio
}

func (s servicioDocumentosOriginalCTDesarrollo) CustodiarOriginalFirmable(ctx context.Context, orden docports.OrdenCustodiarOriginalFirmable, autoridad docports.AutorizarOriginalFirmable) (docports.IntentoOriginalFirmable, error) {
	if s.servicio == nil {
		return docports.IntentoOriginalFirmable{}, docports.ErrCapacidadNoDisponible
	}
	return s.servicio.CustodiarOriginalFirmable(ctx, orden, autoridad)
}

func (s servicioDocumentosOriginalCTDesarrollo) DescargarOriginalConDocumento(ctx context.Context, c docports.ConsultaDocumento) (docports.Original, docdomain.Documento, error) {
	if s.servicio == nil || ctx == nil {
		return docports.Original{}, docdomain.Documento{}, docports.ErrCapacidadNoDisponible
	}
	e := &esperadoOriginalFirmableCTDesarrollo{documentoRef: c.DocumentoID, expedienteRef: c.Autorizacion.AmbitoRef, version: c.Version,
		decisionDescarga: c.Autorizacion.Material.ResumenCapacidad().DecisionRef(), correlacionDescarga: c.Autorizacion.CorrelacionRef}
	return s.servicio.DescargarOriginalConDocumento(context.WithValue(ctx, claveOriginalFirmableCTDesarrollo{}, e), c)
}
