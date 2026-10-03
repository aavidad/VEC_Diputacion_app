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

// RegistradorIntentosFichaPropia es el destino común con su preflight real.
// No concede permiso de lectura ni mantiene un registro propio de Personal.
type RegistradorIntentosFichaPropia interface {
	vecports.RegistradorIntentosAuditoria
	PreflightIntentoAuditoria(context.Context) error
}
type ConfiguracionIntentosFichaPropia struct {
	Proceso                string                              `json:"proceso"`
	Canal                  string                              `json:"canal"`
	RecursoEntradaInvalida string                              `json:"recurso_entrada_invalida"`
	MotivoDenegado         vecdomain.ReferenciaEntradaCatalogo `json:"motivo_denegado"`
	MotivoEntradaInvalida  vecdomain.ReferenciaEntradaCatalogo `json:"motivo_entrada_invalida"`
	MotivoNoDisponible     vecdomain.ReferenciaEntradaCatalogo `json:"motivo_no_disponible"`
}
type RegistroIntentosFichaPropia struct {
	destino       RegistradorIntentosFichaPropia
	configuracion ConfiguracionIntentosFichaPropia
}

func NuevoRegistroIntentosFichaPropia(d RegistradorIntentosFichaPropia, c ConfiguracionIntentosFichaPropia) (*RegistroIntentosFichaPropia, error) {
	if dependenciaNula(d) {
		return nil, domain.ErrFichaPropiaNoDisponible
	}
	for _, motivo := range []vecdomain.ReferenciaEntradaCatalogo{c.MotivoDenegado, c.MotivoEntradaInvalida, c.MotivoNoDisponible} {
		datos := vecdomain.DatosIntentoAuditoria{Accion: domain.AccionFichaPropia, ModuloID: "personal", RecursoRef: c.RecursoEntradaInvalida, FinalidadRef: domain.FinalidadFichaPropia, Resultado: vecdomain.ResultadoIntentoAuditoriaError, Motivo: motivo, Proceso: c.Proceso, Canal: c.Canal, CorrelacionRef: "correlacion_00000000000000000000000000000000"}
		if datos.Validar() != nil {
			return nil, domain.ErrFichaPropiaNoDisponible
		}
	}
	return &RegistroIntentosFichaPropia{d, c}, nil
}
func (r *RegistroIntentosFichaPropia) VerificarRegistroFichaPropia(ctx context.Context) error {
	if r == nil || ctx == nil || ctx.Err() != nil || dependenciaNula(r.destino) || r.destino.PreflightIntentoAuditoria(ctx) != nil {
		return domain.ErrFichaPropiaNoDisponible
	}
	return nil
}
func (r *RegistroIntentosFichaPropia) RegistrarIntentoFichaPropia(ctx context.Context, in ports.IntentoFichaPropia) error {
	if r == nil || ctx == nil || dependenciaNula(r.destino) || (in.Motivo != "entrada_invalida" && in.Motivo != "denegado" && in.Motivo != "no_disponible") {
		return domain.ErrFichaPropiaNoDisponible
	}
	intento, err := estadoIntentoFichaPropia(ctx)
	if err != nil {
		return domain.ErrFichaPropiaNoDisponible
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return domain.ErrFichaPropiaNoDisponible
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil {
		return domain.ErrFichaPropiaNoDisponible
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
			recurso = "personal:ficha_propia:sha256:" + hex.EncodeToString(h[:])
		}
	}
	datos := vecdomain.DatosIntentoAuditoria{Accion: domain.AccionFichaPropia, ModuloID: "personal", RecursoRef: recurso, FinalidadRef: domain.FinalidadFichaPropia, Resultado: resultado, Motivo: motivo, Proceso: r.configuracion.Proceso, Canal: r.configuracion.Canal, CorrelacionRef: ref}
	intento.mu.Lock()
	defer intento.mu.Unlock()
	if intento.orden == nil {
		orden, err := vecports.NuevaOrdenIntentoAuditoria(intento.referencia, intento.identidad.Resultado, intento.identidad.Vinculo, datos)
		if err != nil {
			return domain.ErrFichaPropiaNoDisponible
		}
		intento.orden = &orden
	} else {
		original, err := intento.orden.Datos()
		if err != nil || original.Datos != datos {
			return domain.ErrFichaPropiaNoDisponible
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
	return domain.ErrFichaPropiaNoDisponible
}

var _ ports.RegistroIntentosFichaPropia = (*RegistroIntentosFichaPropia)(nil)
