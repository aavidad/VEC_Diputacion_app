package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// falloPostgreSQLCTDesarrollo registra en qué punto y por qué se detiene la
// composición PostgreSQL de contratación temporal (o una operación que la
// usa) y devuelve el error centinela de siempre, para que quien llama siga
// comparándolo con errors.Is. La etapa es la función y la línea que falla;
// la causa, una clasificación sin datos: SQLSTATE, falta de filas, tiempo
// agotado, conexión o el tipo del error. Nunca se registra el texto del error,
// que puede llevar DSN, nombres de rol o contenido de filas.
func falloPostgreSQLCTDesarrollo(causa error) error {
	registrarFalloPostgreSQLContratacionTemporalDesarrollo(
		etapaLlamadorPostgreSQLCTDesarrollo(2), causaFalloPostgreSQLCTDesarrollo(causa),
	)
	return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
}

// etapaLlamadorPostgreSQLCTDesarrollo devuelve «función:línea» del llamador
// indicado, sin ruta de paquete.
func etapaLlamadorPostgreSQLCTDesarrollo(salto int) string {
	pc, _, linea, ok := runtime.Caller(salto)
	if !ok {
		return "desconocida"
	}
	nombre := "desconocida"
	if f := runtime.FuncForPC(pc); f != nil {
		nombre = f.Name()
		if i := strings.LastIndex(nombre, "/"); i >= 0 {
			nombre = nombre[i+1:]
		}
		nombre = strings.TrimPrefix(nombre, "bootstrap.")
	}
	return fmt.Sprintf("%s:%d", nombre, linea)
}

// causaFalloPostgreSQLCTDesarrollo clasifica el error sin exponer su texto.
func causaFalloPostgreSQLCTDesarrollo(err error) string {
	if err == nil {
		return "comprobacion_rechazada"
	}
	var pg *pgconn.PgError
	var conexion *pgconn.ConnectError
	switch {
	case errors.As(err, &pg) && codigoSQLStateValidoCTDesarrollo(pg.Code):
		return "sqlstate_" + pg.Code
	case errors.Is(err, pgx.ErrNoRows):
		return "sin_filas"
	case errors.Is(err, context.DeadlineExceeded):
		return "tiempo_agotado"
	case errors.Is(err, context.Canceled):
		return "cancelado"
	case errors.As(err, &conexion):
		return "conexion_no_disponible"
	case errors.Is(err, errGobiernoPostgreSQLContratacionTemporalDesarrolloAjeno),
		errors.Is(err, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente),
		errors.Is(err, errGobiernoPostgreSQLContratacionTemporalDesarrolloAgotado):
		return codigoFalloGobiernoPostgreSQLContratacionTemporalDesarrollo(err)
	case errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible):
		return "dependencia_no_disponible"
	default:
		return fmt.Sprintf("error_%T", err)
	}
}

func codigoSQLStateValidoCTDesarrollo(codigo string) bool {
	if len(codigo) != 5 {
		return false
	}
	for _, c := range codigo {
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
