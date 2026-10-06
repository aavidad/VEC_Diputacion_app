package auditoriafirma

import (
	"context"
	"errors"
	"time"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Recuperacion registra los fallos del lector sólo después de que éste cierre
// su transacción. Las lecturas autorizadas se auditan en esa transacción.
type Recuperacion struct {
	lector   ct.LectorRecuperacionFirmasV2
	intentos ports.RegistradorIntentosAuditoria
	fabrica  FabricaOrden
}

var _ ct.LectorRecuperacionFirmasV2 = (*Recuperacion)(nil)

func NuevaRecuperacion(lector ct.LectorRecuperacionFirmasV2, intentos ports.RegistradorIntentosAuditoria, fabrica FabricaOrden) (*Recuperacion, error) {
	if nulo(lector) || nulo(intentos) || nulo(fabrica) {
		return nil, ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	return &Recuperacion{lector: lector, intentos: intentos, fabrica: fabrica}, nil
}

func (r *Recuperacion) RecuperarFirmasAutorizadasV2(ctx context.Context, m ct.MaterialConsultaFirmasR5V2, c ct.CapacidadRecuperacionFirmasV2) (ct.LecturaRecuperacionFirmasV2, error) {
	var cero ct.LecturaRecuperacionFirmasV2
	if r == nil || ctx == nil || nulo(r.lector) || nulo(r.intentos) || nulo(r.fabrica) {
		return cero, ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	if _, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx); err != nil {
		return cero, ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	intento, err := ports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return cero, ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	lectura, err := r.lector.RecuperarFirmasAutorizadasV2(ctx, m, c)
	if err == nil {
		return lectura, nil
	}
	if lecturaAuditadaSinResultado(err) {
		return cero, err
	}
	return cero, r.auditar(ctx, intento, m.ExpedienteRef, err)
}

func (r *Recuperacion) auditar(ctx context.Context, intento, recurso string, fallo error) error {
	ref, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	correlacion, err := ref.ValorCanonico()
	if err != nil {
		return ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	resultado := domain.ResultadoIntentoAuditoriaError
	if errors.Is(fallo, ct.ErrFirmaDocumentoDenegada) || errors.Is(fallo, ct.ErrAutorizacionDenegada) {
		resultado = domain.ResultadoIntentoAuditoriaDenegado
	}
	auditCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancelar()
	orden, err := r.fabrica.CrearOrdenIntentoFirma(auditCtx, intento, ct.AccionRecuperarFirmasR5V2, recurso, resultado)
	if err != nil {
		return ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	d, err := orden.Datos()
	if err != nil || d.IntentoRef != intento || d.Datos.Accion != ct.AccionRecuperarFirmasR5V2 ||
		d.Datos.ModuloID != ct.ModuloContratacion || d.Datos.Resultado != resultado ||
		!ctdomain.ReferenciaOpacaValida(d.Datos.RecursoRef) || d.Datos.RecursoRef != recurso ||
		d.Datos.CorrelacionRef != correlacion {
		return ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	acuse, err := r.intentos.AppendIntentoAuditoria(auditCtx, orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return ct.ErrRegistroFirmaDocumentoNoDisponible
	}
	return fallo
}
