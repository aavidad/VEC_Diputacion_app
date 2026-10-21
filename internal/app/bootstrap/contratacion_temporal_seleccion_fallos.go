package bootstrap

import (
	"context"
	"errors"
	"log/slog"

	appbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Solo se registra el texto fijo del centinela, nunca Error de la causa ni
// de su envoltorio. Las causas SQL y desconocidas usan el clasificador común.
func registrarFalloSeleccionNativaCT(ctx context.Context, err error) {
	causa := causaFalloBolsaRRHHDesarrollo(err)
	for _, centinela := range []error{
		ports.ErrPeticionIntegracionBolsaInvalida, ports.ErrRespuestaBolsaNoConfiable,
		ports.ErrIntegracionBolsaNoDisponible, ports.ErrAutorizacionDenegada,
		ports.ErrEvidenciaBolsaNoAutenticada,
		puertosbolsa.ErrOrigenLlamamientoCTNoDisponible, puertosbolsa.ErrOrigenLlamamientoCTNoEncontrado,
		puertosbolsa.ErrOrigenLlamamientoCTAmbiguo, puertosbolsa.ErrOrigenLlamamientoCTConflicto,
		puertosbolsa.ErrEmisionLlamamientoInvalida, puertosbolsa.ErrEmisionLlamamientoConflicto,
		puertosbolsa.ErrLlamamientoFueraDeOrden, puertosbolsa.ErrMotivoSaltoNoAdmitido,
		puertosbolsa.ErrDatosContactoParticipacionNoDisponibles, puertosbolsa.ErrDatosContactoParticipacionNoEncontrados,
		puertosbolsa.ErrOrigenDatosContactoNoConfigurado,
		appbolsa.ErrCambioSituacionParticipacionNoDisponible,
		dominiobolsa.ErrDatosCorreoLlamamientoIncompletos, dominiobolsa.ErrCorreoLlamamientoExcedeLimite,
		dominiobolsa.ErrPlantillaCorreoLlamamiento,
	} {
		if errors.Is(err, centinela) {
			causa = centinela.Error()
			break
		}
	}
	slog.ErrorContext(ctx, "contratación: selección nativa no completada",
		"operacion", "seleccion_llamamiento", "etapa", "solicitar_llamamiento",
		"causa_interna", causa)
}
