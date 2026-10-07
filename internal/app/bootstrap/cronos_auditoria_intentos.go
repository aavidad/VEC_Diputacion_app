package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type contratoIntentoCronos struct{ accion, finalidad string }

func validarMotivoDenegadoCronos(ctx context.Context, validador vecports.ValidadorReferenciaMotivoAutorizacionV2,
	motivo core.ReferenciaEntradaCatalogo, catalogoID string, ahora time.Time,
) error {
	if ctx == nil || dependenciaBootstrapNula(validador) || motivo.CatalogoID != catalogoID ||
		!core.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return errCronosEmpleadoEn()
	}
	if err := validador.ValidarReferenciaMotivoAutorizacionV2(ctx, motivo, ahora); err != nil {
		return errors.Join(errCronosEmpleadoEn(), err)
	}
	return nil
}

func contratoDenegacionVinculoCronos(ruta string) (contratoIntentoCronos, bool) {
	contratos := map[string]contratoIntentoCronos{
		cronoshttp.RutaConsultarMovimientosPropios: {cronosapp.AccionConsultarMovimientosPropios, cronosapp.FinalidadConsultarMovimientosPropios},
		cronoshttp.RutaSolicitarCorreccionPropia:   {cronosapp.AccionSolicitarCorreccion, cronosapp.FinalidadSolicitarCorreccion},
		cronoshttp.RutaConsultarPermisosPropios:    {cronosapp.AccionConsultarPermisosPropios, cronosapp.FinalidadConsultarPermisosPropios},
		cronoshttp.RutaSolicitarPermisoPropio:      {cronosapp.AccionSolicitarPermisoPropio, cronosapp.FinalidadSolicitarPermisoPropio},
		cronoshttp.RutaBandejaPermisos:             {cronosapp.AccionConsultarBandeja, cronosapp.FinalidadConsultarBandeja},
		cronoshttp.RutaResolverPermiso:             {cronosapp.AccionResolverPermiso, cronosapp.FinalidadResolverPermiso},
		cronoshttp.RutaAvisosPropios:               {cronosapp.AccionConsultarAvisosPropios, cronosapp.FinalidadConsultarAvisosPropios},
		cronoshttp.RutaArchivarAviso:               {cronosapp.AccionArchivarAvisoPropio, cronosapp.FinalidadArchivarAvisoPropio},
		cronoshttp.RutaNotificacionesPropias:       {cronosapp.AccionConsultarNotificacionesPropias, cronosapp.FinalidadConsultarNotificacionesPropias},
		cronoshttp.RutaRegistrarNotificacion:       {cronosapp.AccionRegistrarNotificacion, cronosapp.FinalidadRegistrarNotificacion},
		cronoshttp.RutaBandejaNotificaciones:       {cronosapp.AccionConsultarBandejaNotif, cronosapp.FinalidadConsultarBandejaNotif},
		cronoshttp.RutaAtenderNotificacion:         {cronosapp.AccionAtenderNotificacion, cronosapp.FinalidadAtenderNotificacion},
	}
	c, ok := contratos[ruta]
	return c, ok
}

// registroDenegacionVinculoCronos conserva la identidad histórica acreditada
// por la frontera. La referencia de empleado se reduce a una huella y nunca
// procede del cuerpo de la petición.
type registroDenegacionVinculoCronos struct {
	mu          sync.Mutex
	destino     vecports.RegistradorIntentosAuditoria
	resultado   core.ResultadoContextoActorRegistradoV2
	vinculo     core.VinculoAutenticacionActorV2
	accion      string
	finalidad   string
	recurso     string
	empleadoRef string
	proceso     string
	canal       string
	motivo      core.ReferenciaEntradaCatalogo
	orden       *vecports.OrdenIntentoAuditoria
	confirmado  bool
}

func nuevoRegistroDenegacionVinculoCronos(a *autoridadCronosEmpleadoDesarrollo, holder contextoSeguridadComunDesarrollo, ruta string) (*registroDenegacionVinculoCronos, error) {
	contrato, ok := contratoDenegacionVinculoCronos(ruta)
	if !ok {
		return nil, nil
	}
	if a == nil || dependenciaBootstrapNula(a.auditoriaIntentos) || a.procesoIntentos == "" || !core.ReferenciaMotivoAutorizacionV2Valida(a.motivoDenegado) ||
		holder.Resultado.Validar() != nil || holder.Vinculo.ValidarPara(holder.Resultado) != nil {
		return nil, errCronosEmpleadoEn()
	}
	datosVinculo, err := holder.Vinculo.Datos()
	if err != nil || datosVinculo.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 {
		return nil, errCronosEmpleadoEn()
	}
	empleados, err := holder.Resultado.Contexto.Referencias(core.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 {
		return nil, errCronosEmpleadoEn()
	}
	resultado, err := holder.Resultado.Clonar()
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	huella := sha256.Sum256([]byte(empleados[0]))
	return &registroDenegacionVinculoCronos{destino: a.auditoriaIntentos, resultado: resultado, vinculo: holder.Vinculo,
		accion: contrato.accion, finalidad: contrato.finalidad, recurso: "cronos:empleado:sha256:" + hex.EncodeToString(huella[:]),
		empleadoRef: empleados[0],
		proceso:     a.procesoIntentos, canal: string(datosVinculo.Superficie), motivo: a.motivoDenegado}, nil
}

func (r *registroDenegacionVinculoCronos) RegistrarDenegacionVinculo(ctx context.Context, actor core.ContextoActor) error {
	if r == nil || ctx == nil || actor.Validar() != nil || r.resultado.Validar() != nil ||
		actor.PersonaRef != r.resultado.Contexto.PersonaRef || actor.PerfilActivoRef != r.resultado.Contexto.PerfilActivoRef ||
		actor.Instantanea.VinculoRef != r.resultado.Contexto.Instantanea.VinculoRef || !actor.ResueltoEn.Equal(r.resultado.Contexto.ResueltoEn) {
		return errCronosEmpleadoEn()
	}
	empleados, err := actor.Referencias(core.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 || empleados[0] != r.empleadoRef {
		return errors.Join(errCronosEmpleadoEn(), err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.confirmado {
		return nil
	}
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
	defer cancelar()
	if r.orden == nil {
		correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
		if err != nil {
			return errors.Join(errCronosEmpleadoEn(), err)
		}
		refCorrelacion, err := correlacion.ValorCanonico()
		if err != nil {
			return errors.Join(errCronosEmpleadoEn(), err)
		}
		refIntento, err := vecports.NuevaReferenciaIntentoAuditoria()
		if err != nil {
			return errors.Join(errCronosEmpleadoEn(), err)
		}
		orden, err := vecports.NuevaOrdenIntentoAuditoria(refIntento, r.resultado, r.vinculo, core.DatosIntentoAuditoria{
			Accion: r.accion, ModuloID: "cronos", RecursoRef: r.recurso, FinalidadRef: r.finalidad,
			Resultado: core.ResultadoIntentoAuditoriaDenegado, Motivo: r.motivo,
			Proceso: r.proceso, Canal: r.canal, CorrelacionRef: refCorrelacion,
		})
		if err != nil {
			return errors.Join(errCronosEmpleadoEn(), err)
		}
		r.orden = &orden
	}
	fallos := []error{errCronosEmpleadoEn()}
	for range 2 {
		acuse, err := r.destino.AppendIntentoAuditoria(ctx, *r.orden)
		if err == nil {
			err = acuse.ValidarPara(*r.orden)
		}
		if err == nil {
			r.confirmado = true
			return nil
		}
		fallos = append(fallos, err)
	}
	return errors.Join(fallos...)
}
