package composicion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// RegistradorIntentosHistoriaServiciosPropia es el destino común con su preflight real.
// No concede permiso de lectura ni mantiene un registro propio de Personal.
type RegistradorIntentosHistoriaServiciosPropia interface {
	vecports.RegistradorIntentosAuditoria
	PreflightIntentoAuditoria(context.Context) error
}
type ConfiguracionIntentosHistoriaServiciosPropia struct {
	Proceso                string                              `json:"proceso"`
	Canal                  string                              `json:"canal"`
	RecursoEntradaInvalida string                              `json:"recurso_entrada_invalida"`
	MotivoDenegado         vecdomain.ReferenciaEntradaCatalogo `json:"motivo_denegado"`
	MotivoEntradaInvalida  vecdomain.ReferenciaEntradaCatalogo `json:"motivo_entrada_invalida"`
	MotivoNoDisponible     vecdomain.ReferenciaEntradaCatalogo `json:"motivo_no_disponible"`
}
type RegistroIntentosHistoriaServiciosPropia struct {
	destino       RegistradorIntentosHistoriaServiciosPropia
	configuracion ConfiguracionIntentosHistoriaServiciosPropia
}

func NuevoRegistroIntentosHistoriaServiciosPropia(d RegistradorIntentosHistoriaServiciosPropia, c ConfiguracionIntentosHistoriaServiciosPropia) (*RegistroIntentosHistoriaServiciosPropia, error) {
	if dependenciaNula(d) {
		return nil, domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	for _, motivo := range []vecdomain.ReferenciaEntradaCatalogo{c.MotivoDenegado, c.MotivoEntradaInvalida, c.MotivoNoDisponible} {
		datos := vecdomain.DatosIntentoAuditoria{Accion: domain.AccionHistoriaServiciosPropia, ModuloID: "personal", RecursoRef: c.RecursoEntradaInvalida, FinalidadRef: domain.FinalidadHistoriaServiciosPropia, Resultado: vecdomain.ResultadoIntentoAuditoriaError, Motivo: motivo, Proceso: c.Proceso, Canal: c.Canal, CorrelacionRef: "correlacion_00000000000000000000000000000000"}
		if datos.Validar() != nil {
			return nil, domain.ErrHistoriaServiciosPropiaNoDisponible
		}
	}
	return &RegistroIntentosHistoriaServiciosPropia{d, c}, nil
}
func (r *RegistroIntentosHistoriaServiciosPropia) VerificarRegistroHistoriaServiciosPropia(ctx context.Context) error {
	if r == nil || ctx == nil || ctx.Err() != nil || dependenciaNula(r.destino) || r.destino.PreflightIntentoAuditoria(ctx) != nil {
		return domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	return nil
}
func (r *RegistroIntentosHistoriaServiciosPropia) RegistrarIntentoHistoriaServiciosPropia(ctx context.Context, in ports.IntentoHistoriaServiciosPropia) error {
	if r == nil || ctx == nil || dependenciaNula(r.destino) || (in.Motivo != "entrada_invalida" && in.Motivo != "denegado" && in.Motivo != "no_disponible") {
		return domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	intento, err := estadoIntentoFichaPropia(ctx)
	if err != nil {
		return domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil {
		return domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	motivo := r.configuracion.MotivoNoDisponible
	resultado := vecdomain.ResultadoIntentoAuditoriaError
	if in.Motivo == "entrada_invalida" {
		motivo = r.configuracion.MotivoEntradaInvalida
	}
	if in.Motivo == "denegado" {
		motivo, resultado = r.configuracion.MotivoDenegado, vecdomain.ResultadoIntentoAuditoriaDenegado
	}
	recurso := r.configuracion.RecursoEntradaInvalida
	if in.Motivo != "entrada_invalida" {
		empleados, err := intento.identidad.Resultado.Contexto.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
		if err == nil && len(empleados) == 1 && domain.ReferenciaEmpleadoValida(empleados[0]) && intento.identidad.Resultado.Contexto.AlcanceProyecciones().IncluyeEmpleado() {
			h := sha256.Sum256([]byte(empleados[0]))
			recurso = "personal:historia_servicios_propios:sha256:" + hex.EncodeToString(h[:])
		}
	}
	datos := vecdomain.DatosIntentoAuditoria{Accion: domain.AccionHistoriaServiciosPropia, ModuloID: "personal", RecursoRef: recurso, FinalidadRef: domain.FinalidadHistoriaServiciosPropia, Resultado: resultado, Motivo: motivo, Proceso: r.configuracion.Proceso, Canal: r.configuracion.Canal, CorrelacionRef: ref}
	intento.mu.Lock()
	defer intento.mu.Unlock()
	if intento.orden == nil {
		orden, err := vecports.NuevaOrdenIntentoAuditoria(intento.referencia, intento.identidad.Resultado, intento.identidad.Vinculo, datos)
		if err != nil {
			return domain.ErrHistoriaServiciosPropiaNoDisponible
		}
		intento.orden = &orden
	} else {
		original, err := intento.orden.Datos()
		if err != nil || original.Datos != datos {
			return domain.ErrHistoriaServiciosPropiaNoDisponible
		}
	}
	if intento.acuse != nil {
		return nil
	}
	// COMMIT ambiguo: la recuperación mantiene orden, identidad y referencia.
	// Sólo se repite el append común, nunca la consulta de negocio.
	for range 2 {
		acuse, err := r.destino.AppendIntentoAuditoria(ctx, *intento.orden)
		if err == nil && acuse.ValidarPara(*intento.orden) == nil {
			intento.acuse = &acuse
			return nil
		}
	}
	return domain.ErrHistoriaServiciosPropiaNoDisponible
}

var _ ports.RegistroIntentosHistoriaServiciosPropia = (*RegistroIntentosHistoriaServiciosPropia)(nil)
