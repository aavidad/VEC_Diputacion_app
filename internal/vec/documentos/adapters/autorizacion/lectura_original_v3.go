package autorizacion

import (
	"context"
	"errors"
	"strconv"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Lectura del original con una concesión V3 registrada.
//
// La descarga ya consumió en SQL su autorización V3 (documentos.original.
// descargar, ligada a la preimagen de la consulta). Para abrir el objeto en el
// almacén hace falta además una concesión de almacén: esta fábrica la pide al
// PDP V3 para el recurso de lectura exacto (documento, objeto y versión,
// seudónimos y la correlación de la descarga consumida) y deriva el contexto
// con vecports.NuevoContextoLeerDocumentoGeneradoAlmacenV3. La concesión de
// almacén no se consume de forma única: vale dentro de su ventana registrada y
// solo se pide después de la descarga SQL, que sí se consume.

// AtributoCorrelacionDescarga liga el recurso de lectura del almacén a la
// autorización de descarga ya consumida en SQL.
const AtributoCorrelacionDescarga = "documentos_descarga_correlacion_ref"

// SolicitudConcesionAlmacenV3 es lo que la fábrica pide al PDP: la acción,
// la finalidad y el recurso exactos, para el actor de la descarga consumida.
type SolicitudConcesionAlmacenV3 struct {
	Autorizacion docports.AutorizacionV3
	Accion       string
	Finalidad    string
	Recurso      vecdomain.RecursoAutorizable
}

// ConcesionAlmacenV3 es el trío que devuelve el PDP V3 tras registrar la
// decisión: solicitud evaluada, decisión y confirmación de registro.
type ConcesionAlmacenV3 struct {
	Solicitud    vecdomain.SolicitudAutorizacionLigadaV3
	Decision     vecdomain.DecisionAutorizacionLigadaV3
	Confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
}

// EmisorConcesionAlmacenV3 es la frontera con el PDP V3 y con la
// seudonimización. La composición debe construir la solicitud con el vínculo
// del actor de la petición y un motivo del catálogo; nunca concede por
// ausencia del proveedor.
type EmisorConcesionAlmacenV3 interface {
	SeudonimosLecturaOriginal(context.Context, docports.AutorizacionV3) (DatosSeudonimosLectura, error)
	EmitirConcesionAlmacenV3(context.Context, SolicitudConcesionAlmacenV3) (ConcesionAlmacenV3, error)
}

// FabricaContextoLecturaOriginalV3 implementa FabricaContextoLectura con V3.
type FabricaContextoLecturaOriginalV3 struct {
	emisor EmisorConcesionAlmacenV3
	reloj  vecports.Reloj
}

func NuevaFabricaContextoLecturaOriginalV3(emisor EmisorConcesionAlmacenV3, reloj vecports.Reloj) (*FabricaContextoLecturaOriginalV3, error) {
	if nulo(emisor) || nulo(reloj) {
		return nil, vecports.ErrAutorizacionAlmacenInvalida
	}
	return &FabricaContextoLecturaOriginalV3{emisor: emisor, reloj: reloj}, nil
}

var _ docports.FabricaContextoLectura = (*FabricaContextoLecturaOriginalV3)(nil)

func (f *FabricaContextoLecturaOriginalV3) ContextoLecturaOriginal(
	ctx context.Context, d domain.Documento, a docports.AutorizacionV3,
) (vecports.ContextoOperacionAlmacen, error) {
	if f == nil || nulo(f.emisor) || nulo(f.reloj) || ctx == nil || ctx.Err() != nil ||
		d.Validar() != nil || !d.Descargable() || d.ID != a.RecursoRef || d.ExpedienteRef != a.AmbitoRef {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(contextoCancelado(ctx))
	}
	preimagen, err := (docports.ConsultaDocumento{DocumentoID: d.ID, Version: d.Version}).PreimagenDescargar()
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	if err := a.ValidarPara(docports.AccionDescargar, f.reloj.Ahora()); err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	if a.Material.ResumenCapacidad().Operacion() != docports.AccionDescargar ||
		a.Material.ResumenCapacidad().EfectoHuellaSHA256() != docports.HuellaEfectoV3(preimagen) {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(nil)
	}
	seudonimos, err := f.emisor.SeudonimosLecturaOriginal(ctx, a)
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	if err := ctx.Err(); err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	objeto := vecports.ReferenciaObjetoAlmacen{Referencia: d.ObjetoRef, Version: d.ObjetoVersion}
	vinculos := vecports.VinculosOperacionAlmacen{
		OperacionRef: a.CorrelacionRef, CargaRef: d.ID,
		Clasificacion: d.Proteccion, SujetoSeudonimoHMAC: seudonimos.SujetoHMAC,
		HuellaSolicitudHMAC: seudonimos.SolicitudHMAC,
		EfectoRef:           d.ID, ObjetoVinculado: objeto,
	}
	recurso := RecursoLecturaOriginalV3(d, a, vinculos)
	if err := errors.Join(recurso.Validar(), objeto.Validar()); err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	concesion, err := f.emisor.EmitirConcesionAlmacenV3(ctx, SolicitudConcesionAlmacenV3{
		Autorizacion: a, Accion: vecports.AccionNegocioLeerOriginalDocumentoGenerado,
		Finalidad: a.Finalidad, Recurso: recurso,
	})
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	if err := ctx.Err(); err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	if err := concesionDelActor(concesion, a, recurso); err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	return vecports.NuevoContextoLeerDocumentoGeneradoAlmacenV3(
		concesion.Solicitud, concesion.Decision, concesion.Confirmacion, vinculos, f.reloj.Ahora())
}

// concesionDelActor exige que la solicitud evaluada sea la pedida y del mismo
// actor, perfil y finalidad que la descarga consumida. La decisión y su
// registro los coteja el núcleo al derivar el contexto.
func concesionDelActor(c ConcesionAlmacenV3, a docports.AutorizacionV3, recurso vecdomain.RecursoAutorizable) error {
	datos, err := c.Solicitud.Datos()
	if err != nil {
		return err
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		return err
	}
	esperada, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return err
	}
	obtenida, err := datos.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || obtenida != esperada || datos.Recurso.Referencia != recurso.Referencia ||
		datos.Recurso.ModuloID != recurso.ModuloID || datos.Recurso.Tipo != recurso.Tipo ||
		datos.Accion != vecports.AccionNegocioLeerOriginalDocumentoGenerado || datos.Finalidad != a.Finalidad ||
		vinculo.PrincipalID != a.PrincipalID || vinculo.PerfilActivoRef != a.PerfilActivoRef {
		return vecports.ErrAutorizacionAlmacenInvalida
	}
	return nil
}

// RecursoLecturaOriginalV3 es el recurso que evalúa el PDP V3 para abrir el
// objeto: el mismo módulo, tipo y ámbito que la descarga SQL (la concesión
// del perfil los cubre) y, como atributos, los vínculos de almacén, el objeto
// y la versión exactos, los datos del documento y la correlación de la
// descarga consumida.
func RecursoLecturaOriginalV3(
	d domain.Documento, a docports.AutorizacionV3, vinculos vecports.VinculosOperacionAlmacen,
) vecdomain.RecursoAutorizable {
	recurso := vecdomain.RecursoAutorizable{
		Referencia: d.ID, ModuloID: "documentos", Tipo: "documento_original",
		Ambitos: map[string]string{"organizacion_ref": docports.OrganizacionRefV3},
		Atributos: map[string]string{
			vecports.AtributoAlmacenOperacionRef:        vinculos.OperacionRef,
			vecports.AtributoAlmacenCargaRef:            vinculos.CargaRef,
			vecports.AtributoAlmacenClasificacion:       vinculos.Clasificacion,
			vecports.AtributoAlmacenSujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
			vecports.AtributoAlmacenHuellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
			vecports.AtributoAlmacenEfectoRef:           vinculos.EfectoRef,
			vecports.AtributoAlmacenObjetoRef:           d.ObjetoRef,
			vecports.AtributoAlmacenObjetoVersion:       d.ObjetoVersion,
			AtributoCorrelacionDescarga:                 a.CorrelacionRef,
			"documento_expediente_ref":                  d.ExpedienteRef,
			"documento_modulo_productor":                d.ModuloID,
			"documento_version":                         strconv.FormatUint(d.Version, 10),
			"documento_tipo_ref":                        d.TipoRef,
			"documento_huella_sha256":                   d.HuellaSHA256,
		},
	}
	return recurso
}
