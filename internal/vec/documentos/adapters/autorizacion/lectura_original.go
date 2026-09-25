package autorizacion

import (
	"context"
	"reflect"
	"strconv"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// DatosSeudonimosLectura procede de la autoridad corporativa, que calcula
// HMAC con claves fuera del modulo. Ninguna huella sin clave se presenta como HMAC.
type DatosSeudonimosLectura struct {
	SujetoHMAC    string
	SolicitudHMAC string
}

// ResolutorDecisionLecturaOriginal es la frontera PDP V1. La composicion
// productiva debe inyectar la autoridad corporativa y su fuente de seudonimos;
// el adaptador no fabrica decisiones ni concede por ausencia del proveedor.
type ResolutorDecisionLecturaOriginal interface {
	SeudonimosLecturaOriginal(context.Context, docports.AutorizacionV3) (DatosSeudonimosLectura, error)
	ExigirLecturaOriginal(context.Context, SolicitudDecisionLecturaOriginal) (vecdomain.DecisionAutorizacion, error)
}

// SolicitudDecisionLecturaOriginal es interna al proceso. Recurso se construye
// exclusivamente a partir del documento recuperado bajo SQL V3 y sus vinculos.
type SolicitudDecisionLecturaOriginal struct {
	Autorizacion docports.AutorizacionV3
	Accion       string
	Finalidad    string
	Recurso      vecdomain.RecursoAutorizable
}

type FabricaContextoLecturaOriginal struct {
	resolutor ResolutorDecisionLecturaOriginal
	reloj     vecports.Reloj
}

func NuevaFabricaContextoLecturaOriginal(
	resolutor ResolutorDecisionLecturaOriginal,
	reloj vecports.Reloj,
) (*FabricaContextoLecturaOriginal, error) {
	if nulo(resolutor) || nulo(reloj) {
		return nil, vecports.ErrAutorizacionAlmacenInvalida
	}
	return &FabricaContextoLecturaOriginal{resolutor: resolutor, reloj: reloj}, nil
}

var _ docports.FabricaContextoLectura = (*FabricaContextoLecturaOriginal)(nil)

func (f *FabricaContextoLecturaOriginal) ContextoLecturaOriginal(
	ctx context.Context, d domain.Documento, a docports.AutorizacionV3,
) (vecports.ContextoOperacionAlmacen, error) {
	denegado := vecports.ErrAutorizacionAlmacenInvalida
	if f == nil || nulo(f.resolutor) || nulo(f.reloj) || ctx == nil || ctx.Err() != nil ||
		d.Validar() != nil || !d.Descargable() || d.ID != a.RecursoRef || d.ExpedienteRef != a.AmbitoRef {
		return vecports.ContextoOperacionAlmacen{}, denegado
	}
	instante := f.reloj.Ahora()
	preimagen, err := (docports.ConsultaDocumento{DocumentoID: d.ID, Version: d.Version}).PreimagenDescargar()
	if err != nil || a.ValidarPara(docports.AccionDescargar, instante) != nil ||
		a.Material.ResumenCapacidad().Operacion() != docports.AccionDescargar ||
		a.Material.ResumenCapacidad().EfectoHuellaSHA256() != docports.HuellaPreimagen(preimagen) {
		return vecports.ContextoOperacionAlmacen{}, denegado
	}
	seudonimos, err := f.resolutor.SeudonimosLecturaOriginal(ctx, a)
	if err != nil || ctx.Err() != nil {
		return vecports.ContextoOperacionAlmacen{}, denegado
	}
	objeto := vecports.ReferenciaObjetoAlmacen{Referencia: d.ObjetoRef, Version: d.ObjetoVersion}
	vinculos := vecports.VinculosOperacionAlmacen{
		OperacionRef: a.CorrelacionRef, CargaRef: d.ID,
		Clasificacion: d.Proteccion, SujetoSeudonimoHMAC: seudonimos.SujetoHMAC,
		HuellaSolicitudHMAC: seudonimos.SolicitudHMAC,
		EfectoRef:           d.ID, ObjetoVinculado: objeto,
	}
	recurso := recursoLecturaOriginal(d, a, vinculos)
	if recurso.Validar() != nil || objeto.Validar() != nil {
		return vecports.ContextoOperacionAlmacen{}, denegado
	}
	decision, err := f.resolutor.ExigirLecturaOriginal(ctx, SolicitudDecisionLecturaOriginal{
		Autorizacion: a, Accion: vecports.AccionNegocioLeerOriginalDocumentoGenerado,
		Finalidad: a.Finalidad, Recurso: recurso,
	})
	if err != nil || ctx.Err() != nil || decision.PrincipalID != a.PrincipalID ||
		decision.PerfilActivoRef != a.PerfilActivoRef || decision.CorrelacionRef != a.CorrelacionRef ||
		decision.Finalidad != a.Finalidad {
		return vecports.ContextoOperacionAlmacen{}, denegado
	}
	return vecports.NuevoContextoLeerDocumentoGeneradoAlmacen(decision, recurso, vinculos, f.reloj.Ahora())
}

func recursoLecturaOriginal(
	d domain.Documento, a docports.AutorizacionV3, vinculos vecports.VinculosOperacionAlmacen,
) vecdomain.RecursoAutorizable {
	return vecdomain.RecursoAutorizable{
		Referencia: d.ID, ModuloID: "documentos", Tipo: "documento_original",
		Ambitos: map[string]string{
			"expediente_ref": d.ExpedienteRef, "modulo_productor": d.ModuloID,
			"ambito_v3": a.AmbitoRef,
		},
		Atributos: map[string]string{
			vecports.AtributoAlmacenOperacionRef:        vinculos.OperacionRef,
			vecports.AtributoAlmacenCargaRef:            vinculos.CargaRef,
			vecports.AtributoAlmacenClasificacion:       vinculos.Clasificacion,
			vecports.AtributoAlmacenSujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
			vecports.AtributoAlmacenHuellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
			vecports.AtributoAlmacenEfectoRef:           vinculos.EfectoRef,
			vecports.AtributoAlmacenObjetoRef:           d.ObjetoRef,
			vecports.AtributoAlmacenObjetoVersion:       d.ObjetoVersion,
			"documento_version":                         strconv.FormatUint(d.Version, 10),
			"documento_tipo_ref":                        d.TipoRef,
			"documento_huella_sha256":                   d.HuellaSHA256,
			"documento_politica_ref":                    d.PoliticaRef,
			"documento_politica_version":                strconv.FormatUint(d.VersionPolitica, 10),
			"documento_politica_huella_sha256":          d.HuellaPoliticaSHA256,
		},
	}
}

func nulo(v any) bool {
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
