package postgres

import (
	"context"
	"errors"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ConfiguracionIntentosCapacidades procede de la composición privada. Los
// motivos son entradas del catálogo gobernado; no se sustituyen por errores
// SQL ni por texto de la petición. No hay valores por defecto.
type ConfiguracionIntentosCapacidades struct {
	Proceso        string
	Canal          string
	FinalidadRef   string
	MotivoDenegado domain.ReferenciaEntradaCatalogo
	MotivoError    domain.ReferenciaEntradaCatalogo
}

func (c ConfiguracionIntentosCapacidades) datos(actor domain.ContextoActor, correlacion string, resultado domain.ResultadoIntentoAuditoria) domain.DatosIntentoAuditoria {
	motivo := c.MotivoError
	if resultado == domain.ResultadoIntentoAuditoriaDenegado {
		motivo = c.MotivoDenegado
	}
	return domain.DatosIntentoAuditoria{
		Accion: "administracion.perfiles.consultar", ModuloID: "administracion",
		RecursoRef: actor.PerfilActivoRef, FinalidadRef: c.FinalidadRef,
		Resultado: resultado, Motivo: motivo, Proceso: c.Proceso, Canal: c.Canal,
		CorrelacionRef: correlacion,
	}
}

func (f *FuenteCapacidades) Capacidades(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles) (api.Capacidades, error) {
	if ctx == nil || f == nil || ausente(f.intentos) || evidencia.ValidarPara(actor) != nil {
		return api.Capacidades{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	correlacion, ok := ports.CorrelacionIncidenciasPeticion(ctx)
	if !ok {
		return api.Capacidades{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	correlacion = "correlacion_" + correlacion
	vinculo, err := evidencia.Vinculo.Datos()
	if err != nil || f.configuracionIntentos.Canal != string(vinculo.Superficie) {
		return api.Capacidades{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	for _, resultado := range []domain.ResultadoIntentoAuditoria{domain.ResultadoIntentoAuditoriaDenegado, domain.ResultadoIntentoAuditoriaError} {
		if f.configuracionIntentos.datos(actor, correlacion, resultado).Validar() != nil {
			return api.Capacidades{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
	}

	salida, err := f.consultarCapacidades(ctx, actor, evidencia)
	if err == nil {
		// AD168 ya confirmó el consumo y la auditoría permitida juntos.
		return salida, nil
	}
	// consultarCapacidades ya retornó: ejecutarConsumoADMIN ejecutó su
	// rollback diferido. La orden acredita identidad histórica, incluso si
	// durante el intento caducó o se canceló el contexto de la petición.
	resultado := domain.ResultadoIntentoAuditoriaError
	if errors.Is(err, domain.ErrAutorizacionDenegada) {
		resultado = domain.ResultadoIntentoAuditoriaDenegado
	}
	referencia, fallo := ports.NuevaReferenciaIntentoAuditoria()
	if fallo != nil {
		return api.Capacidades{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	orden, fallo := ports.NuevaOrdenIntentoAuditoria(referencia, evidencia.ResultadoContexto, evidencia.Vinculo,
		f.configuracionIntentos.datos(actor, correlacion, resultado))
	if fallo != nil {
		return api.Capacidades{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	acuse, fallo := f.intentos.AppendIntentoAuditoria(ctx, orden)
	if errors.Is(fallo, ports.ErrIntentoAuditoriaNoDisponible) {
		// Un COMMIT incierto puede haber escrito. L coteja la misma orden y
		// recupera el acuse original; nunca creamos otra clave de intento.
		acuse, fallo = f.intentos.AppendIntentoAuditoria(ctx, orden)
	}
	if fallo != nil || acuse.ValidarPara(orden) != nil {
		return api.Capacidades{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return api.Capacidades{}, err
}
