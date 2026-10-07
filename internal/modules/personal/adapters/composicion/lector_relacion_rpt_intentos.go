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

// RegistradorIntentosLectorRPT exige el puerto común y su preflight real.
// No concede permiso de negocio ni permite omitir la configuración DBA.
type RegistradorIntentosLectorRPT interface {
	vecports.RegistradorIntentosAuditoria
	PreflightIntentoAuditoria(context.Context) error
}

type ConfiguracionIntentosLectorRPT struct {
	Proceso                string
	Canal                  string
	MotivoDenegado         vecdomain.ReferenciaEntradaCatalogo
	MotivoEntradaInvalida  vecdomain.ReferenciaEntradaCatalogo
	MotivoNoDisponible     vecdomain.ReferenciaEntradaCatalogo
	RecursoEntradaInvalida string
}

// RegistroIntentosLectorRelacionRPT enlaza fallos a la cadena común después de
// cerrar la transacción de lectura. Nunca consulta otra identidad al auditar.
type RegistroIntentosLectorRelacionRPT struct {
	destino       RegistradorIntentosLectorRPT
	configuracion ConfiguracionIntentosLectorRPT
}

func NuevoRegistroIntentosLectorRelacionRPT(d RegistradorIntentosLectorRPT, c ConfiguracionIntentosLectorRPT) (*RegistroIntentosLectorRelacionRPT, error) {
	if dependenciaNula(d) {
		return nil, domain.ErrLectorRelacionRPTNoDisponible
	}
	for _, motivo := range []vecdomain.ReferenciaEntradaCatalogo{c.MotivoDenegado, c.MotivoEntradaInvalida, c.MotivoNoDisponible} {
		prueba := vecdomain.DatosIntentoAuditoria{Accion: ports.AccionRelacionParaRPTV1, ModuloID: "personal", RecursoRef: c.RecursoEntradaInvalida, FinalidadRef: domain.FinalidadLectorRelacionRPT, Resultado: vecdomain.ResultadoIntentoAuditoriaError, Motivo: motivo, Proceso: c.Proceso, Canal: c.Canal, CorrelacionRef: "correlacion_00000000000000000000000000000000"}
		if prueba.Validar() != nil {
			return nil, domain.ErrLectorRelacionRPTNoDisponible
		}
	}
	return &RegistroIntentosLectorRelacionRPT{destino: d, configuracion: c}, nil
}

func (r *RegistroIntentosLectorRelacionRPT) VerificarRegistroRelacionRPT(ctx context.Context) error {
	if r == nil || ctx == nil || ctx.Err() != nil || dependenciaNula(r.destino) {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	if err := r.destino.PreflightIntentoAuditoria(ctx); err != nil {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	return nil
}

func (r *RegistroIntentosLectorRelacionRPT) RegistrarIntentoRelacionRPT(ctx context.Context, in ports.IntentoLectorRelacionRPT) error {
	if r == nil || ctx == nil || dependenciaNula(r.destino) || (in.Motivo != "entrada_invalida" && in.Motivo != "denegado" && in.Motivo != "no_disponible") {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	intento, err := estadoIntentoLectorRelacionRPT(ctx)
	if err != nil {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	// Actor declarado inválido se conserva como error de entrada, nunca como una
	// identidad alternativa. La orden siempre nombra a la frontera original.
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	motivo := r.configuracion.MotivoNoDisponible
	resultado := vecdomain.ResultadoIntentoAuditoriaError
	if in.Motivo == "entrada_invalida" {
		motivo = r.configuracion.MotivoEntradaInvalida
	}
	if in.Motivo == "denegado" {
		motivo = r.configuracion.MotivoDenegado
		resultado = vecdomain.ResultadoIntentoAuditoriaDenegado
	}
	recurso := r.configuracion.RecursoEntradaInvalida
	if domain.ReferenciaRelacionValida(in.RelacionRef) {
		// El contrato común sólo admite claves minúsculas. El hash conserva la
		// referencia exacta, sin normalizarla ni registrar entrada libre.
		h := sha256.Sum256([]byte(in.RelacionRef))
		recurso = "personal:relacion_rpt:sha256:" + hex.EncodeToString(h[:])
	}
	datos := vecdomain.DatosIntentoAuditoria{Accion: ports.AccionRelacionParaRPTV1, ModuloID: "personal", RecursoRef: recurso, FinalidadRef: domain.FinalidadLectorRelacionRPT, Resultado: resultado, Motivo: motivo, Proceso: r.configuracion.Proceso, Canal: r.configuracion.Canal, CorrelacionRef: ref}
	intento.mu.Lock()
	defer intento.mu.Unlock()
	if intento.orden == nil {
		orden, err := vecports.NuevaOrdenIntentoAuditoria(intento.referencia, intento.identidad.Resultado, intento.identidad.Vinculo, datos)
		if err != nil {
			return domain.ErrLectorRelacionRPTNoDisponible
		}
		intento.orden = &orden
	} else {
		original, err := intento.orden.Datos()
		if err != nil || original.Datos != datos {
			return domain.ErrLectorRelacionRPTNoDisponible
		}
	}
	if intento.acuse != nil {
		return nil
	}
	// Ante COMMIT ambiguo se recupera la misma orden, con la misma referencia e
	// identidad. Nunca se repite la lectura ni se crea un segundo intento.
	for range 2 {
		acuse, err := r.destino.AppendIntentoAuditoria(ctx, *intento.orden)
		if err == nil && acuse.ValidarPara(*intento.orden) == nil {
			intento.acuse = &acuse
			return nil
		}
	}
	return domain.ErrLectorRelacionRPTNoDisponible
}

var _ ports.RegistroIntentosLectorRelacionRPT = (*RegistroIntentosLectorRelacionRPT)(nil)
