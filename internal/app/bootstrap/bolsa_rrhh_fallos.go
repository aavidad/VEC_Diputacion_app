package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// maximoCausaFalloBolsaRRHH acota la línea de registro.
const maximoCausaFalloBolsaRRHH = 240

// causaFalloBolsaRRHHDesarrollo describe un fallo de lectura de Bolsa RRHH
// para el registro del servidor, nunca para la respuesta. De PostgreSQL y de
// la conexión solo se da el código (sus mensajes pueden llevar valores o
// datos de conexión); del resto, el texto del error: en estas rutas son
// centinelas de texto fijo, a veces con fichero y línea del rechazo
// (errBorradorNoDisponibleEn), sin datos de personas.
func causaFalloBolsaRRHHDesarrollo(err error) string {
	var pg *pgconn.PgError
	var conexion *pgconn.ConnectError
	switch {
	case err == nil:
		return "sin_error"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled),
		errors.As(err, &pg), errors.As(err, &conexion), errors.Is(err, pgx.ErrNoRows):
		return causaFalloPostgreSQLCTDesarrollo(err)
	}
	texto := strings.Join(strings.Fields(err.Error()), " ")
	if len(texto) > maximoCausaFalloBolsaRRHH {
		texto = texto[:maximoCausaFalloBolsaRRHH]
	}
	return texto
}

func autorizacionDenegadaBolsaRRHH(err error) bool {
	return errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) ||
		errors.Is(err, ErrSeguridadComunDesarrolloDenegada)
}

// responderFalloBolsaRRHHDesarrollo deja la causa interna en el registro y
// responde con el estado que corresponde: 403 si la autorización se denegó,
// 504 si se agotó el plazo y 503 en el resto. Una petición cancelada por el
// cliente no se registra como fallo del servicio.
func responderFalloBolsaRRHHDesarrollo(w http.ResponseWriter, r *http.Request, etapa string, err error) {
	ruta := ""
	if r != nil && r.URL != nil {
		ruta = r.URL.Path
	}
	switch {
	case errors.Is(err, context.Canceled):
		responderBolsaRRHHDesarrollo(w, http.StatusServiceUnavailable, map[string]string{"codigo": "servicio_no_disponible"})
		return
	case autorizacionDenegadaBolsaRRHH(err):
		slog.Warn("bolsa rrhh: lectura denegada", "ruta", ruta, "etapa", etapa, "causa", causaFalloBolsaRRHHDesarrollo(err))
		responderBolsaRRHHDesarrollo(w, http.StatusForbidden, map[string]string{"codigo": "acceso_denegado"})
	case errors.Is(err, context.DeadlineExceeded):
		slog.Error("bolsa rrhh: lectura no disponible", "ruta", ruta, "etapa", etapa, "causa", causaFalloBolsaRRHHDesarrollo(err))
		responderBolsaRRHHDesarrollo(w, http.StatusGatewayTimeout, map[string]string{"codigo": "tiempo_agotado"})
	default:
		slog.Error("bolsa rrhh: lectura no disponible", "ruta", ruta, "etapa", etapa, "causa", causaFalloBolsaRRHHDesarrollo(err))
		responderBolsaRRHHDesarrollo(w, http.StatusServiceUnavailable, map[string]string{"codigo": "servicio_no_disponible"})
	}
}

// errContactosBolsaRRHHIncoherentes: el repositorio repitió el cursor.
var errContactosBolsaRRHHIncoherentes = errors.Join(puertosbolsa.ErrContactoParticipacionNoDisponible, errors.New("bolsa rrhh: paginación de contactos sin avance"))
