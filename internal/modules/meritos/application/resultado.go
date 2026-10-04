package application

import (
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

// ValidarResultadoOperacion es un cotejo puro, compartido por servicio y
// adaptador transaccional. El adaptador lo ejecuta antes de confirmar COMMIT.
// La referencia de auditoría solo afirma persistencia cuando el puerto devuelve
// este resultado después de COMMIT; un DTO de entrada no puede afirmarla.
func ValidarResultadoOperacion(orden ports.OrdenOperacion, resultado ports.ResultadoOperacion) error {
	if !domain.ReferenciaValida(resultado.AuditoriaRef) {
		return ports.ErrRegistroNoDisponible
	}
	if resultado.Codigo != "confirmada" {
		if resultado.Recibo != nil || resultado.Anterior != nil || errorResultadoOperacion(resultado.Codigo) == ports.ErrRegistroNoDisponible {
			return ports.ErrRegistroNoDisponible
		}
		return nil
	}
	if resultado.Recibo == nil || !reciboCoincide(*resultado.Recibo, orden) {
		return ports.ErrRegistroNoDisponible
	}
	cambio, err := prepararCambio(orden, resultado.Anterior, resultado.Recibo.RegistradoEn)
	if err == nil && cambio.Nuevo.Hecho.Revision != nil {
		fecha, fechaErr := time.Parse(time.RFC3339Nano, resultado.Recibo.Registro.Hecho.Revision.Fecha)
		if fechaErr != nil || !fecha.Equal(resultado.Recibo.RegistradoEn) {
			return ports.ErrRegistroNoDisponible
		}
		// PostgreSQL conserva precisión de microsegundos con seis decimales;
		// una representación equivalente no cambia la revisión original.
		cambio.Nuevo.Hecho.Revision.Fecha = resultado.Recibo.Registro.Hecho.Revision.Fecha
	}
	if err != nil || !reflect.DeepEqual(resultado.Recibo.Registro, cambio.Nuevo) {
		return ports.ErrRegistroNoDisponible
	}
	return nil
}

func errorResultadoOperacion(codigo string) error {
	switch codigo {
	case "conflicto_version":
		return ports.ErrConflictoVersion
	case "clave_reutilizada":
		return ports.ErrClaveReutilizada
	case "denegada":
		return vec.ErrAutorizacionDenegada
	default:
		return ports.ErrRegistroNoDisponible
	}
}
