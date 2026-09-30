package autorizacion

import (
	"context"
	"strconv"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Custodia del documento firmado con una concesión V3 registrada.
//
// La autorización V3 documentos.firmado.custodiar que consumirá el SQL
// (custodiar_firmado_v1) está ligada a la preimagen exacta de la custodia.
// Para escribir el objeto antes hace falta una concesión de almacén de la
// misma acción, para un recurso con los vínculos de almacén y ligado a esa
// decisión (referencia y huella), a las huellas firmada y original y a la
// operación de firma. Esa concesión no se consume por sí misma: la custodia
// solo queda hecha cuando el SQL consume la decisión de la preimagen.

// Atributos que ligan el recurso de almacén a la decisión que consume el SQL.
const (
	AtributoDecisionCustodia       = "documentos_custodia_decision_ref"
	AtributoHuellaDecisionCustodia = "documentos_custodia_decision_sha256"
)

// FabricaContextoCustodiaFirmadoV3 implementa docports.FabricaContextoCustodia.
type FabricaContextoCustodiaFirmadoV3 struct {
	emisor EmisorConcesionAlmacenV3
	reloj  vecports.Reloj
}

func NuevaFabricaContextoCustodiaFirmadoV3(emisor EmisorConcesionAlmacenV3, reloj vecports.Reloj) (*FabricaContextoCustodiaFirmadoV3, error) {
	if nulo(emisor) || nulo(reloj) {
		return nil, vecports.ErrAutorizacionAlmacenInvalida
	}
	return &FabricaContextoCustodiaFirmadoV3{emisor: emisor, reloj: reloj}, nil
}

var _ docports.FabricaContextoCustodia = (*FabricaContextoCustodiaFirmadoV3)(nil)

func (f *FabricaContextoCustodiaFirmadoV3) ContextoCustodiaFirmado(
	ctx context.Context, c docports.CustodiaFirmadoPersistente,
) (vecports.ContextoOperacionAlmacen, error) {
	a := c.Autorizacion
	if f == nil || nulo(f.emisor) || nulo(f.reloj) || ctx == nil || ctx.Err() != nil ||
		a.RecursoRef != c.ID || a.AmbitoRef != c.ExpedienteRef {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(contextoCancelado(ctx))
	}
	preimagen, err := c.PreimagenCustodia()
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	if err := a.ValidarPara(docports.AccionCustodiarFirmado, f.reloj.Ahora()); err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	resumen := a.Material.ResumenCapacidad()
	if resumen.Operacion() != docports.AccionCustodiarFirmado ||
		resumen.EfectoHuellaSHA256() != docports.HuellaEfectoV3(preimagen) {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(nil)
	}
	seudonimos, err := f.emisor.SeudonimosLecturaOriginal(ctx, a)
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	vinculos := vecports.VinculosOperacionAlmacen{
		OperacionRef: a.CorrelacionRef, CargaRef: c.ID,
		Clasificacion:       string(c.Politica.Politica().Proteccion()),
		SujetoSeudonimoHMAC: seudonimos.SujetoHMAC, HuellaSolicitudHMAC: seudonimos.SolicitudHMAC,
		EfectoRef: c.ID,
	}
	recurso := RecursoCustodiaFirmadoV3(c, vinculos)
	if err := recurso.Validar(); err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	concesion, err := f.emisor.EmitirConcesionAlmacenV3(ctx, SolicitudConcesionAlmacenV3{
		PrincipalID: a.PrincipalID, PerfilActivoRef: a.PerfilActivoRef, CorrelacionRef: a.CorrelacionRef,
		Accion: vecports.AccionNegocioCustodiarDocumentoFirmadoExpediente, Finalidad: a.Finalidad, Recurso: recurso,
	})
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	if err := ctx.Err(); err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	if err := concesionDelActor(concesion, a, recurso, vecports.AccionNegocioCustodiarDocumentoFirmadoExpediente); err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	return vecports.NuevoContextoCustodiarDocumentoFirmadoExpedienteAlmacenV3(
		concesion.Solicitud, concesion.Decision, concesion.Confirmacion, vinculos, f.reloj.Ahora())
}

// RecursoCustodiaFirmadoV3 tiene el mismo módulo, tipo y ámbito que la
// decisión SQL (la concesión del perfil los cubre) y, como atributos, los
// vínculos de almacén, la decisión que consumirá el SQL y el documento.
func RecursoCustodiaFirmadoV3(c docports.CustodiaFirmadoPersistente, vinculos vecports.VinculosOperacionAlmacen) vecdomain.RecursoAutorizable {
	resumen := c.Autorizacion.Material.ResumenCapacidad()
	return vecdomain.RecursoAutorizable{
		Referencia: c.ID, ModuloID: "documentos", Tipo: "documento_firmado",
		Ambitos: map[string]string{"organizacion_ref": docports.OrganizacionRefV3},
		Atributos: map[string]string{
			vecports.AtributoAlmacenOperacionRef:        vinculos.OperacionRef,
			vecports.AtributoAlmacenCargaRef:            vinculos.CargaRef,
			vecports.AtributoAlmacenClasificacion:       vinculos.Clasificacion,
			vecports.AtributoAlmacenSujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
			vecports.AtributoAlmacenHuellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
			vecports.AtributoAlmacenEfectoRef:           vinculos.EfectoRef,
			AtributoDecisionCustodia:                    resumen.DecisionRef(),
			AtributoHuellaDecisionCustodia:              resumen.DecisionHuellaSHA256(),
			"documento_expediente_ref":                  c.ExpedienteRef,
			"documento_modulo_productor":                c.ModuloID,
			"documento_version":                         strconv.FormatUint(c.Version, 10),
			"documento_tipo_ref":                        c.TipoRef,
			"documento_huella_sha256":                   c.HuellaSHA256,
			"documento_huella_original_sha256":          c.HuellaOriginalSHA256,
			"documento_firma_operacion_ref":             c.FirmaOperacionRef,
		},
	}
}
