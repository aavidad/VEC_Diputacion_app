package autorizacion

import (
	"context"
	"strconv"

	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// EmisorOperacionOriginalFirmable obtiene la decisión V3 desde el contexto
// autenticado de la misma petición. El contrato no permite que un DTO CT
// aporte actor, perfil, organización ni una capacidad ya emitida.
type EmisorOperacionOriginalFirmable interface {
	AutorizarOperacionOriginalFirmable(context.Context, string, []byte, string, string) (docports.AutorizacionV3, error)
}

// AutoridadOriginalFirmableV3 separa las decisiones de reserva y confirmación
// consumidas por SQL de la tercera concesión registrada para escribir objeto.
type AutoridadOriginalFirmableV3 struct {
	operaciones EmisorOperacionOriginalFirmable
	almacen     EmisorConcesionAlmacenV3
	reloj       vecports.Reloj
}

func NuevaAutoridadOriginalFirmableV3(operaciones EmisorOperacionOriginalFirmable, almacen EmisorConcesionAlmacenV3, reloj vecports.Reloj) (*AutoridadOriginalFirmableV3, error) {
	if nulo(operaciones) || nulo(almacen) || nulo(reloj) {
		return nil, vecports.ErrAutorizacionAlmacenInvalida
	}
	return &AutoridadOriginalFirmableV3{operaciones: operaciones, almacen: almacen, reloj: reloj}, nil
}

func (a *AutoridadOriginalFirmableV3) AutorizarReservaOriginal(ctx context.Context, preimagen []byte, id, expedienteRef string) (docports.AutorizacionV3, error) {
	return a.autorizarOperacion(ctx, docports.AccionReservarOriginalFirmable, preimagen, id, expedienteRef)
}

func (a *AutoridadOriginalFirmableV3) AutorizarConfirmacionOriginal(ctx context.Context, preimagen []byte, id, expedienteRef string) (docports.AutorizacionV3, error) {
	return a.autorizarOperacion(ctx, docports.AccionConfirmarOriginalFirmable, preimagen, id, expedienteRef)
}

func (a *AutoridadOriginalFirmableV3) autorizarOperacion(ctx context.Context, accion string, preimagen []byte, id, expedienteRef string) (docports.AutorizacionV3, error) {
	if a == nil || nulo(a.operaciones) || nulo(a.reloj) || ctx == nil || ctx.Err() != nil ||
		len(preimagen) == 0 || len(preimagen) > 16384 || id == "" || expedienteRef == "" {
		return docports.AutorizacionV3{}, denegadoPor(contextoCancelado(ctx))
	}
	concesion, err := a.operaciones.AutorizarOperacionOriginalFirmable(ctx, accion, preimagen, id, expedienteRef)
	if err != nil {
		return docports.AutorizacionV3{}, denegadoPor(err)
	}
	if concesion.RecursoRef != id || concesion.AmbitoRef != expedienteRef ||
		docapp.ValidarAutorizacionOriginalFirmable(concesion, accion, a.reloj.Ahora()) != nil ||
		concesion.Material.ResumenCapacidad().EfectoHuellaSHA256() != docports.HuellaEfectoV3(preimagen) {
		return docports.AutorizacionV3{}, denegadoPor(nil)
	}
	return concesion, nil
}

func (a *AutoridadOriginalFirmableV3) ContextoEscrituraOriginal(
	ctx context.Context, r docports.ReservaOriginalFirmable, intento docports.IntentoOriginalFirmable,
) (vecports.ContextoOperacionAlmacen, error) {
	if a == nil || nulo(a.almacen) || nulo(a.reloj) || ctx == nil || ctx.Err() != nil ||
		docapp.ValidarIntentoOriginalFirmable(intento, r) != nil || intento.Estado != "pendiente" {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(contextoCancelado(ctx))
	}
	preimagen, err := docapp.PreimagenReservaOriginalFirmable(r)
	if err != nil || r.Autorizacion.RecursoRef != r.ID || r.Autorizacion.AmbitoRef != r.ExpedienteRef ||
		docapp.ValidarAutorizacionOriginalFirmable(r.Autorizacion, docports.AccionReservarOriginalFirmable, a.reloj.Ahora()) != nil ||
		r.Autorizacion.Material.ResumenCapacidad().EfectoHuellaSHA256() != docports.HuellaEfectoV3(preimagen) {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	seudonimos, err := a.almacen.SeudonimosLecturaOriginal(ctx, r.Autorizacion)
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	vinculos := vecports.VinculosOperacionAlmacen{
		OperacionRef: r.Autorizacion.CorrelacionRef, CargaRef: intento.ClaveAlmacenRef,
		Clasificacion:       string(r.Politica.Politica().Proteccion()),
		SujetoSeudonimoHMAC: seudonimos.SujetoHMAC, HuellaSolicitudHMAC: seudonimos.SolicitudHMAC,
		EfectoRef: r.ID,
	}
	recurso := RecursoEscrituraOriginalFirmableV3(r, intento, vinculos)
	if recurso.Validar() != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(nil)
	}
	concesion, err := a.almacen.EmitirConcesionAlmacenV3(ctx, SolicitudConcesionAlmacenV3{
		PrincipalID: r.Autorizacion.PrincipalID, PerfilActivoRef: r.Autorizacion.PerfilActivoRef,
		CorrelacionRef: r.Autorizacion.CorrelacionRef,
		Accion:         vecports.AccionNegocioEscribirOriginalFirmable,
		Finalidad:      r.Autorizacion.Finalidad, Recurso: recurso,
	})
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(err)
	}
	if ctx.Err() != nil || concesionDelActor(concesion, r.Autorizacion, recurso, vecports.AccionNegocioEscribirOriginalFirmable) != nil {
		return vecports.ContextoOperacionAlmacen{}, denegadoPor(contextoCancelado(ctx))
	}
	return vecports.NuevoContextoEscribirOriginalFirmableAlmacenV3(
		concesion.Solicitud, concesion.Decision, concesion.Confirmacion, vinculos, a.reloj.Ahora())
}

// RecursoEscrituraOriginalFirmableV3 compromete el intento y la huella.
// La referencia y el tipo siguen siendo de Documentos; ningún campo CT se
// interpreta aquí. La política central debe conceder explícitamente la acción.
func RecursoEscrituraOriginalFirmableV3(
	r docports.ReservaOriginalFirmable, intento docports.IntentoOriginalFirmable,
	v vecports.VinculosOperacionAlmacen,
) vecdomain.RecursoAutorizable {
	resumen := r.Autorizacion.Material.ResumenCapacidad()
	return vecdomain.RecursoAutorizable{
		Referencia: r.ID, ModuloID: "documentos", Tipo: "documento_original_firmable",
		Ambitos: map[string]string{"organizacion_ref": docports.OrganizacionRefV3},
		Atributos: map[string]string{
			vecports.AtributoAlmacenOperacionRef:        v.OperacionRef,
			vecports.AtributoAlmacenCargaRef:            v.CargaRef,
			vecports.AtributoAlmacenClasificacion:       v.Clasificacion,
			vecports.AtributoAlmacenSujetoSeudonimoHMAC: v.SujetoSeudonimoHMAC,
			vecports.AtributoAlmacenHuellaSolicitudHMAC: v.HuellaSolicitudHMAC,
			vecports.AtributoAlmacenEfectoRef:           v.EfectoRef,
			"documentos_original_reserva_ref":           intento.ReservaRef,
			"documentos_original_intento_num":           strconv.FormatUint(intento.Numero, 10),
			"documentos_original_clave_almacen_ref":     intento.ClaveAlmacenRef,
			"documentos_original_huella_sha256":         r.HuellaSHA256,
			"documentos_original_expediente_ref":        r.ExpedienteRef,
			"documentos_original_tipo_ref":              r.TipoRef,
			"documentos_original_version":               strconv.FormatUint(r.Version, 10),
			"documentos_original_decision_reserva_ref":  resumen.DecisionRef(),
			"documentos_original_decision_reserva_sha":  resumen.DecisionHuellaSHA256(),
		},
	}
}

var _ docports.AutorizarOriginalFirmable = (*AutoridadOriginalFirmableV3)(nil)
