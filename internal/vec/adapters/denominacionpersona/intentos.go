package denominacionpersona

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Configuración del proceso y motivos gobernados; no procede del cliente.
type ConfiguracionIntentos struct {
	Proceso, Canal string
	MotivoError    domain.ReferenciaEntradaCatalogo
}

func (l *Lector) datosError(a ports.AccesoDenominacionPersona, accion string) domain.DatosIntentoAuditoria {
	return domain.DatosIntentoAuditoria{Accion: accion, ModuloID: a.Recurso.ModuloID, RecursoRef: a.Recurso.Referencia, FinalidadRef: a.FinalidadRef, Resultado: domain.ResultadoIntentoAuditoriaError, Motivo: l.configuracion.MotivoError, Proceso: l.configuracion.Proceso, Canal: l.configuracion.Canal, CorrelacionRef: a.Auditoria.CorrelationRef}
}

func (l *Lector) evidenciaValida(a ports.AccesoDenominacionPersona, accion string) bool {
	h, err := a.Contexto.HuellaSHA256VinculadaV2()
	v, errV := a.Vinculo.Datos()
	return err == nil && errV == nil && a.ResultadoContexto.Validar() == nil && a.ResultadoContexto.HuellaSHA256 == h && a.Vinculo.ValidarPara(a.ResultadoContexto) == nil && string(v.Superficie) == l.configuracion.Canal && l.datosError(a, accion).Validar() == nil
}

// El intento original ya terminó. Este error observado conserva el consumo
// permitido previo; no afirma rollback ni reescribe su auditoría. Una orden
// común única se conserva ante COMMIT ambiguo del registrador de intentos.
func (l *Lector) registrarError(ctx context.Context, a ports.AccesoDenominacionPersona, accion string) error {
	if ctx == nil {
		return ErrNoDisponible
	}
	ref, err := ports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return ErrNoDisponible
	}
	orden, err := ports.NuevaOrdenIntentoAuditoria(ref, a.ResultadoContexto, a.Vinculo, l.datosError(a, accion))
	if err != nil {
		return ErrNoDisponible
	}
	acuse, err := l.intentos.AppendIntentoAuditoria(ctx, orden)
	if errors.Is(err, ports.ErrIntentoAuditoriaNoDisponible) {
		acuse, err = l.intentos.AppendIntentoAuditoria(ctx, orden)
	}
	if err != nil || acuse.ValidarPara(orden) != nil {
		return ErrNoDisponible
	}
	return ErrNoDisponible
}
