package postgres

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Se llama cuando consumir ya ha terminado y su rollback diferido ha salido.
// Conserva el resultado y vínculo V2 originales; la configuración privada
// aporta proceso, canal y motivos de catálogo.
func (f *Fuente) registrarFallo(ctx context.Context, evidencia domain.EvidenciaSesionAdministracionPerfiles, accion, recursoRef, correlacion string, causa error) error {
	if f == nil || evidencia.Vinculo.ValidarPara(evidencia.ResultadoContexto) != nil {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	resultado := domain.ResultadoIntentoAuditoriaError
	motivo := f.config.MotivoError
	if errors.Is(causa, domain.ErrAutorizacionDenegada) {
		resultado = domain.ResultadoIntentoAuditoriaDenegado
		motivo = f.config.MotivoDenegado
	}
	datos := domain.DatosIntentoAuditoria{Accion: accion, ModuloID: "administracion", RecursoRef: recursoRef,
		FinalidadRef: "gestion_usuarios", Resultado: resultado, Motivo: motivo,
		Proceso: f.config.Proceso, Canal: f.config.Canal, CorrelacionRef: correlacion}
	if datos.Validar() != nil {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	ref, err := ports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	orden, err := ports.NuevaOrdenIntentoAuditoria(ref, evidencia.ResultadoContexto, evidencia.Vinculo, datos)
	if err != nil {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	base := context.WithoutCancel(ctx)
	registroCtx, cancel := context.WithTimeout(base, 2*time.Second)
	defer cancel()
	acuse, err := f.intentos.AppendIntentoAuditoria(registroCtx, orden)
	if errors.Is(err, ports.ErrIntentoAuditoriaNoDisponible) {
		// Misma orden ante COMMIT incierto; nunca otra referencia de intento.
		acuse, err = f.intentos.AppendIntentoAuditoria(registroCtx, orden)
	}
	if err != nil || acuse.ValidarPara(orden) != nil {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return nil
}
