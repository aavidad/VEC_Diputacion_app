package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	importacionpg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgresimportacionconvoca"
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// maximoCausaFalloBolsaRRHH acota la línea de registro.
const maximoCausaFalloBolsaRRHH = 240

// centinelasFalloBolsaRRHH son los errores de texto fijo (sin datos) que
// pueden llegar a estas rutas. Solo su texto va al registro.
var centinelasFalloBolsaRRHH = []error{
	dominiovec.ErrAutorizacionDenegada, dominiovec.ErrPermissionDenied, ErrSeguridadComunDesarrolloDenegada,
	ErrComposicionDesarrolloIncompleta,
	puertosbolsa.ErrContactoParticipacionNoDisponible, puertosbolsa.ErrContactoParticipacionNoEncontrado,
	puertosbolsa.ErrSituacionParticipacionNoDisponible, puertosbolsa.ErrSituacionParticipacionNoEncontrada,
	puertosbolsa.ErrConsultaEstadoCeseNoDisponible, puertosbolsa.ErrConsultaOrdenVigenteNoDisponible,
	puertosbolsa.ErrConstitucionBolsaNoDisponible, puertosbolsa.ErrConstitucionBolsaInvalida,
	puertosbolsa.ErrEmisionLlamamientoNoDisponible, puertosbolsa.ErrPoliticaAvisosNoDisponible,
	importacionpg.ErrRepositorioNoDisponible, importacionpg.ErrResultadoNoConfiable, importacionapp.ErrStagingExpurgado,
}

// causaFalloBolsaRRHHDesarrollo describe un fallo de lectura de Bolsa RRHH
// para el registro del servidor, nunca para la respuesta. De PostgreSQL y de
// la conexión solo se da el código (sus mensajes pueden llevar valores o
// datos de conexión). Del resto, solo el texto de un centinela conocido de
// la lista cerrada; el rechazo del montaje B-BACK añade fichero y línea
// (errBorradorNoDisponibleEn). Cualquier otro error queda en su tipo, sin
// su texto, para que un error futuro con valores no llegue al registro.
func causaFalloBolsaRRHHDesarrollo(err error) string {
	var pg *pgconn.PgError
	var conexion *pgconn.ConnectError
	switch {
	case err == nil:
		return "sin_error"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled),
		errors.As(err, &pg), errors.As(err, &conexion), errors.Is(err, pgx.ErrNoRows):
		return causaFalloPostgreSQLCTDesarrollo(err)
	case errors.Is(err, errBorradorLlamamientoDesarrolloNoDisponible):
		return acotarCausaFalloBolsaRRHH(err.Error())
	}
	for _, centinela := range centinelasFalloBolsaRRHH {
		if errors.Is(err, centinela) {
			return centinela.Error()
		}
	}
	return causaFalloPostgreSQLCTDesarrollo(err)
}

func acotarCausaFalloBolsaRRHH(texto string) string {
	texto = strings.Join(strings.Fields(texto), " ")
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
