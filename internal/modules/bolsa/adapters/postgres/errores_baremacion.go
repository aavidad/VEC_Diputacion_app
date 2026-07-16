package postgres

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func accionReserva(clase puertosbolsa.ClaseCambioBaremacion) puertosbolsa.AccionOperacionBaremacion {
	if clase == puertosbolsa.ClaseCambioAltaBaremacion {
		return puertosbolsa.AccionReservarAltaBaremacion
	}
	return puertosbolsa.AccionReservarDecisionBaremacion
}

func accionConfirmacion(clase puertosbolsa.ClaseCambioBaremacion) puertosbolsa.AccionOperacionBaremacion {
	if clase == puertosbolsa.ClaseCambioAltaBaremacion {
		return puertosbolsa.AccionConfirmarAltaBaremacion
	}
	return puertosbolsa.AccionConfirmarDecisionBaremacion
}

func accionAbandono(clase puertosbolsa.ClaseCambioBaremacion) puertosbolsa.AccionOperacionBaremacion {
	if clase == puertosbolsa.ClaseCambioAltaBaremacion {
		return puertosbolsa.AccionAbandonarAltaBaremacion
	}
	return puertosbolsa.AccionAbandonarDecisionBaremacion
}

func errorEstadoReserva(estado string) (error, bool) {
	switch estado {
	case "en_curso":
		return puertosbolsa.ErrCambioBaremacionEnCurso, true
	case "idempotencia_reutilizada":
		return puertosbolsa.ErrClaveIdempotenciaBaremacionReutilizada, true
	case "autorizacion_reutilizada":
		return puertosbolsa.ErrAutorizacionBaremacionReutilizada, true
	case "autorizacion_obsoleta":
		return puertosbolsa.ErrAutorizacionBaremacionInvalida, true
	case "ya_existe":
		return puertosbolsa.ErrBaremacionYaExiste, true
	case "conflicto_version":
		return puertosbolsa.ErrVersionBaremacionConflicto, true
	case "no_encontrada":
		return puertosbolsa.ErrBaremacionNoEncontrada, true
	case "evidencia_no_confiable":
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, true
	case "colision":
		return puertosbolsa.ErrCambioBaremacionEnCurso, true
	case "rechazada":
		return puertosbolsa.ErrSolicitudBaremacionInvalida, true
	default:
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, false
	}
}

func errorEstadoConfirmacion(estado string) (error, bool) {
	switch estado {
	case "reserva_invalida":
		return puertosbolsa.ErrReservaBaremacionNoValida, true
	case "idempotencia_reutilizada":
		return puertosbolsa.ErrClaveIdempotenciaBaremacionReutilizada, true
	case "autorizacion_reutilizada":
		return puertosbolsa.ErrAutorizacionBaremacionReutilizada, true
	case "autorizacion_obsoleta":
		return puertosbolsa.ErrAutorizacionBaremacionInvalida, true
	case "ya_existe":
		return puertosbolsa.ErrBaremacionYaExiste, true
	case "conflicto_version":
		return puertosbolsa.ErrVersionBaremacionConflicto, true
	case "no_encontrada":
		return puertosbolsa.ErrBaremacionNoEncontrada, true
	case "historial_no_anexable":
		return puertosbolsa.ErrHistorialBaremacionNoAnexable, true
	case "evidencia_no_confiable":
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, true
	case "colision":
		return puertosbolsa.ErrCambioBaremacionEnCurso, true
	case "rechazada":
		return puertosbolsa.ErrSolicitudBaremacionInvalida, true
	default:
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, false
	}
}

func errorEstadoAbandono(estado string) (error, bool) {
	switch estado {
	case "reserva_invalida":
		return puertosbolsa.ErrReservaBaremacionNoValida, true
	case "idempotencia_reutilizada":
		return puertosbolsa.ErrClaveIdempotenciaBaremacionReutilizada, true
	case "autorizacion_reutilizada":
		return puertosbolsa.ErrAutorizacionBaremacionReutilizada, true
	case "autorizacion_obsoleta":
		return puertosbolsa.ErrAutorizacionBaremacionInvalida, true
	case "colision":
		return puertosbolsa.ErrCambioBaremacionEnCurso, true
	case "rechazada":
		return puertosbolsa.ErrSolicitudBaremacionInvalida, true
	default:
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, false
	}
}

func errorEstadoLectura(estado string) (error, bool) {
	switch estado {
	case "autorizacion_reutilizada":
		return puertosbolsa.ErrAutorizacionBaremacionReutilizada, true
	case "autorizacion_obsoleta":
		return puertosbolsa.ErrAutorizacionBaremacionInvalida, true
	case "evidencia_no_confiable":
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, true
	case "colision":
		return puertosbolsa.ErrCambioBaremacionEnCurso, true
	case "rechazada":
		return puertosbolsa.ErrSolicitudBaremacionInvalida, true
	default:
		return puertosbolsa.ErrEvidenciaBaremacionNoConfiable, false
	}
}

func errorPostgreSQL(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		switch pgError.Code {
		case "40001", "40P01", "55P03", "57014":
			return puertosbolsa.ErrCambioBaremacionEnCurso
		case "22000", "22023", "23503", "23514", "55000":
			return puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
	}
	return puertosbolsa.ErrFuenteBaremacionNoDisponible
}

func validarContexto(ctx context.Context) error {
	if ctx == nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	return ctx.Err()
}

func revertir(tx pgx.Tx) {
	if tx == nil {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelar()
	_ = tx.Rollback(ctx)
}

func valorNulo(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
