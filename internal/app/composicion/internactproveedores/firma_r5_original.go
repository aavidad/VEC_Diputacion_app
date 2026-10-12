package internactproveedores

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"

	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/vec/adapters/almacen"
	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const finalidadDescargaOriginalR5 = "descargar_documento_original"

type claveOriginalR5 struct{}

type originalR5Esperado struct {
	id, expediente, tipo, clave, huella string
	version                             uint64
	tamano                              int64
	reserva, almacen                    string
	intento                             uint64
}

// AutoridadOriginalR5 no guarda permisos. Las cuatro decisiones V3 se emiten
// en la misma petición F1 y se cotejan con el documento concreto de CT.
type AutoridadOriginalR5 struct {
	proveedor  *ProveedorFirmaR5
	seudonimos [sha256.Size]byte
	politicas  interface {
		ClaveTipo(string) (string, bool)
		CustodiaOriginalCTReservada(string) bool
		SolicitudPara(string, string) (vecports.SolicitudPoliticaConservacionDocumental, error)
	}
	mapear almacen.MapeadorExpedienteOriginalCT
	reloj  vecports.Reloj
}

func NuevaAutoridadOriginalR5(proveedor *ProveedorFirmaR5, clave []byte,
	politicas interface {
		ClaveTipo(string) (string, bool)
		CustodiaOriginalCTReservada(string) bool
		SolicitudPara(string, string) (vecports.SolicitudPoliticaConservacionDocumental, error)
	}, reloj vecports.Reloj) (*AutoridadOriginalR5, error) {
	if proveedor == nil || len(clave) != sha256.Size || politicas == nil || reloj == nil {
		return nil, ErrProveedorFirmaR5NoDisponible
	}
	var k [sha256.Size]byte
	copy(k[:], clave)
	return &AutoridadOriginalR5{proveedor: proveedor, seudonimos: k, politicas: politicas,
		mapear: almacen.FuncionMapeoExpedienteOriginalCT(ctapp.ReferenciaExpedienteDocumentalFormalizacion), reloj: reloj}, nil
}

func (a *AutoridadOriginalR5) Cerrar() {
	if a != nil {
		clear(a.seudonimos[:])
		a.proveedor = nil
	}
}

func (a *AutoridadOriginalR5) identidad(s vecports.SolicitudOriginalFirmableCT, ref string) (almacencanonico.IdentidadOriginalCT, string, error) {
	i := almacencanonico.IdentidadOriginalCT{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, Documento: s.Documento, Version: s.OriginalVersion}
	if a == nil || a.mapear == nil || !i.Valida() || ref != i.Referencia() || s.OrganizacionRef != docports.OrganizacionRefV3 ||
		(s.OriginalRef != "" && s.OriginalRef != ref) {
		return i, "", vecports.ErrOriginalFirmableCTInvalido
	}
	exp, err := a.mapear.ReferenciaDocumentalCT(s.ExpedienteRef)
	if err != nil || !docdomain.ReferenciaOpacaValida(exp) {
		return i, "", vecports.ErrOriginalFirmableCTInvalido
	}
	return i, exp, nil
}

func (a *AutoridadOriginalR5) AutorizarLecturaOriginalCT(ctx context.Context, s vecports.SolicitudOriginalFirmableCT, ref string) (docports.ConsultaDocumento, error) {
	var vacio docports.ConsultaDocumento
	if a == nil || a.proveedor == nil || ctx == nil || ctx.Err() != nil {
		return vacio, ErrProveedorFirmaR5NoDisponible
	}
	_, exp, err := a.identidad(s, ref)
	if err != nil {
		return vacio, err
	}
	q := docports.ConsultaDocumento{DocumentoID: ref, Version: s.OriginalVersion}
	pre, err := q.PreimagenDescargar()
	if err != nil {
		return vacio, vecports.ErrOriginalFirmableCTInvalido
	}
	recurso, err := docports.RecursoV3(docports.AccionDescargar, ref, pre)
	if err != nil {
		return vacio, vecports.ErrOriginalFirmableCTInvalido
	}
	q.Autorizacion, err = a.emitirAutorizacion(ctx, docports.AccionDescargar, finalidadDescargaOriginalR5, recurso, ref, exp)
	if err != nil {
		return vacio, err
	}
	if q.Autorizacion.Material.ResumenCapacidad().EfectoHuellaSHA256() != docports.HuellaEfectoV3(pre) {
		return vacio, ErrProveedorFirmaR5NoDisponible
	}
	return q, nil
}

func (a *AutoridadOriginalR5) PrepararCustodiaOriginalCT(ctx context.Context, s vecports.SolicitudOriginalFirmableCT,
	pdf vecports.PDFOriginalCT, ref string) (docports.OrdenCustodiarOriginalFirmable, docports.AutorizarOriginalFirmable, error) {
	var vacio docports.OrdenCustodiarOriginalFirmable
	if a == nil || a.proveedor == nil || a.politicas == nil || ctx == nil || ctx.Err() != nil {
		return vacio, nil, ErrProveedorFirmaR5NoDisponible
	}
	i, exp, err := a.identidad(s, ref)
	if err != nil {
		return vacio, nil, err
	}
	if len(pdf.Contenido) < 8 || len(pdf.Contenido) > vecports.LimiteOriginalFirmableCT || !bytes.HasPrefix(pdf.Contenido, []byte("%PDF-")) {
		return vacio, nil, vecports.ErrOriginalFirmableCTInvalido
	}
	clave, ok := a.politicas.ClaveTipo(pdf.TipoRef)
	if !ok || !a.politicas.CustodiaOriginalCTReservada(pdf.TipoRef) {
		return vacio, nil, ErrProveedorFirmaR5NoDisponible
	}
	politica, err := a.politicas.SolicitudPara(clave, exp)
	if err != nil {
		return vacio, nil, ErrProveedorFirmaR5NoDisponible
	}
	contenido := bytes.Clone(pdf.Contenido)
	suma := sha256.Sum256(contenido)
	esperado := &originalR5Esperado{id: ref, expediente: exp, tipo: pdf.TipoRef, clave: i.ClaveLogica(),
		huella: hex.EncodeToString(suma[:]), version: s.OriginalVersion, tamano: int64(len(contenido))}
	orden := docports.OrdenCustodiarOriginalFirmable{ID: ref, ClaveIdempotencia: esperado.clave, ModuloID: "contratacion_temporal",
		ExpedienteRef: exp, TipoRef: pdf.TipoRef, Version: s.OriginalVersion, MIME: "application/pdf", Contenido: contenido, SolicitudPolitica: politica}
	delegada, err := docautorizacion.NuevaAutoridadOriginalFirmableV3(a, a, a.reloj)
	if err != nil {
		return vacio, nil, ErrProveedorFirmaR5NoDisponible
	}
	return orden, autoridadPeticionOriginalR5{esperado: esperado, delegada: delegada}, nil
}

type autoridadPeticionOriginalR5 struct {
	esperado *originalR5Esperado
	delegada *docautorizacion.AutoridadOriginalFirmableV3
}

func (a autoridadPeticionOriginalR5) AutorizarReservaOriginal(ctx context.Context, pre []byte, id, exp string) (docports.AutorizacionV3, error) {
	if a.esperado == nil || a.delegada == nil {
		return docports.AutorizacionV3{}, docports.ErrAccesoDenegado
	}
	return a.delegada.AutorizarReservaOriginal(context.WithValue(ctx, claveOriginalR5{}, a.esperado), pre, id, exp)
}
func (a autoridadPeticionOriginalR5) AutorizarConfirmacionOriginal(ctx context.Context, pre []byte, id, exp string) (docports.AutorizacionV3, error) {
	if a.esperado == nil || a.delegada == nil {
		return docports.AutorizacionV3{}, docports.ErrAccesoDenegado
	}
	return a.delegada.AutorizarConfirmacionOriginal(context.WithValue(ctx, claveOriginalR5{}, a.esperado), pre, id, exp)
}
func (a autoridadPeticionOriginalR5) ContextoEscrituraOriginal(ctx context.Context, r docports.ReservaOriginalFirmable, i docports.IntentoOriginalFirmable) (vecports.ContextoOperacionAlmacen, error) {
	if a.esperado == nil || a.delegada == nil {
		return vecports.ContextoOperacionAlmacen{}, docports.ErrAccesoDenegado
	}
	return a.delegada.ContextoEscrituraOriginal(context.WithValue(ctx, claveOriginalR5{}, a.esperado), r, i)
}

func (a *AutoridadOriginalR5) AutorizarOperacionOriginalFirmable(ctx context.Context, accion string, pre []byte, id, exp string) (docports.AutorizacionV3, error) {
	e, ok := ctx.Value(claveOriginalR5{}).(*originalR5Esperado)
	if a == nil || a.proveedor == nil || !ok || e == nil || id != e.id || exp != e.expediente ||
		!preimagenOriginalR5Valida(accion, pre, e) {
		return docports.AutorizacionV3{}, docports.ErrAccesoDenegado
	}
	suma := sha256.Sum256(pre)
	recurso := core.RecursoAutorizable{Referencia: id, ModuloID: "documentos", Tipo: "documento_original_firmable",
		Ambitos:   map[string]string{"organizacion_ref": docports.OrganizacionRefV3},
		Atributos: map[string]string{"preimagen_sha256": hex.EncodeToString(suma[:])}}
	return a.emitirAutorizacion(ctx, accion, docports.FinalidadOriginalFirmable, recurso, id, exp)
}

func preimagenOriginalR5Valida(accion string, pre []byte, e *originalR5Esperado) bool {
	if len(pre) == 0 || len(pre) > 16384 || e == nil {
		return false
	}
	var v struct {
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
	if json.Unmarshal(pre, &v) != nil || v.Accion != accion || v.ID != e.id || v.Huella != e.huella {
		return false
	}
	switch accion {
	case docports.AccionReservarOriginalFirmable:
		return v.Clave == e.clave && v.Modulo == "contratacion_temporal" && v.Expediente == e.expediente &&
			v.Tipo == e.tipo && v.Version == e.version && v.MIME == "application/pdf" && v.Tamano == e.tamano
	case docports.AccionConfirmarOriginalFirmable:
		return e.reserva != "" && v.Reserva == e.reserva && v.Intento == e.intento && v.Almacen == e.almacen && v.Objeto.Huella == e.huella
	}
	return false
}

func (a *AutoridadOriginalR5) emitirAutorizacion(ctx context.Context, accion, finalidad string, recurso core.RecursoAutorizable, id, exp string) (docports.AutorizacionV3, error) {
	var vacio docports.AutorizacionV3
	s, d, c, m, err := a.proveedor.EmitirMaterialOriginal(ctx, accion, finalidad, recurso)
	_ = d
	_ = c
	if err != nil {
		return vacio, err
	}
	datos, err := s.Datos()
	if err != nil {
		return vacio, ErrProveedorFirmaR5NoDisponible
	}
	actor, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		return vacio, ErrProveedorFirmaR5NoDisponible
	}
	correlacion, err := datos.Correlacion.ValorCanonico()
	if err != nil {
		return vacio, ErrProveedorFirmaR5NoDisponible
	}
	return docports.AutorizacionV3{Material: m, Accion: accion, Finalidad: finalidad, RecursoRef: id, AmbitoRef: exp,
		PrincipalID: actor.PrincipalID, PerfilActivoRef: actor.PerfilActivoRef, CorrelacionRef: correlacion}, nil
}

func (a *AutoridadOriginalR5) SeudonimosLecturaOriginal(ctx context.Context, aut docports.AutorizacionV3) (docautorizacion.DatosSeudonimosLectura, error) {
	if a == nil || a.seudonimos == [sha256.Size]byte{} || ctx == nil || ctx.Err() != nil ||
		aut.PrincipalID == "" || aut.CorrelacionRef == "" || aut.RecursoRef == "" {
		return docautorizacion.DatosSeudonimosLectura{}, docports.ErrAccesoDenegado
	}
	firma := func(etiqueta string, partes ...string) string {
		mac := hmac.New(sha256.New, a.seudonimos[:])
		_, _ = mac.Write([]byte("vec.almacen.seudonimo.ct-firma-r5.v1\x00" + etiqueta))
		for _, parte := range partes {
			_, _ = mac.Write([]byte{0})
			_, _ = mac.Write([]byte(parte))
		}
		return hex.EncodeToString(mac.Sum(nil))
	}
	return docautorizacion.DatosSeudonimosLectura{
		SujetoHMAC:    "hmac-sha256:almacen_sujeto_desarrollo_v1:" + firma("sujeto", aut.PrincipalID),
		SolicitudHMAC: "hmac-sha256:almacen_solicitud_desarrollo_v1:" + firma("solicitud", aut.CorrelacionRef, aut.RecursoRef)}, nil
}

func (a *AutoridadOriginalR5) EmitirConcesionAlmacenV3(ctx context.Context, q docautorizacion.SolicitudConcesionAlmacenV3) (docautorizacion.ConcesionAlmacenV3, error) {
	var vacia docautorizacion.ConcesionAlmacenV3
	if a == nil || a.proveedor == nil || ctx == nil || ctx.Err() != nil || q.Recurso.Validar() != nil ||
		q.Recurso.ModuloID != "documentos" || q.PrincipalID == "" || q.PerfilActivoRef == "" || q.CorrelacionRef == "" {
		return vacia, docports.ErrAccesoDenegado
	}
	if e, ok := ctx.Value(claveOriginalR5{}).(*originalR5Esperado); ok {
		if e == nil || q.Accion != vecports.AccionNegocioEscribirOriginalFirmable || q.Recurso.Referencia != e.id ||
			q.Recurso.Tipo != "documento_original_firmable" || q.Finalidad != docports.FinalidadOriginalFirmable ||
			q.Recurso.Atributos["documentos_original_huella_sha256"] != e.huella ||
			q.Recurso.Atributos["documentos_original_tipo_ref"] != e.tipo ||
			q.Recurso.Atributos["documentos_original_version"] != strconv.FormatUint(e.version, 10) {
			return vacia, docports.ErrAccesoDenegado
		}
		reserva := q.Recurso.Atributos["documentos_original_reserva_ref"]
		intento, err := strconv.ParseUint(q.Recurso.Atributos["documentos_original_intento_num"], 10, 53)
		almacen := q.Recurso.Atributos["documentos_original_clave_almacen_ref"]
		if err != nil || !docdomain.ReferenciaOpacaValida(reserva) || !docdomain.ReferenciaOpacaValida(almacen) {
			return vacia, docports.ErrAccesoDenegado
		}
		e.reserva, e.intento, e.almacen = reserva, intento, almacen
	} else if q.Accion != vecports.AccionNegocioLeerOriginalDocumentoGenerado || q.Recurso.Tipo != "documento_original" ||
		q.Finalidad != finalidadDescargaOriginalR5 {
		return vacia, docports.ErrAccesoDenegado
	}
	s, d, c, _, err := a.proveedor.EmitirMaterialOriginal(ctx, q.Accion, q.Finalidad, q.Recurso)
	if err != nil {
		return vacia, err
	}
	datos, err := s.Datos()
	if err != nil {
		return vacia, ErrProveedorFirmaR5NoDisponible
	}
	actor, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil || actor.PrincipalID != q.PrincipalID || actor.PerfilActivoRef != q.PerfilActivoRef {
		return vacia, docports.ErrAccesoDenegado
	}
	return docautorizacion.ConcesionAlmacenV3{Solicitud: s, Decision: d, Confirmacion: c}, nil
}

var (
	_ almacen.AutorizacionesDocumentosOriginalCT      = (*AutoridadOriginalR5)(nil)
	_ docautorizacion.EmisorOperacionOriginalFirmable = (*AutoridadOriginalR5)(nil)
	_ docautorizacion.EmisorConcesionAlmacenV3        = (*AutoridadOriginalR5)(nil)
)
