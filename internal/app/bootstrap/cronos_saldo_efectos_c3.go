package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	cronoscomp "vec-diputacion-granada/internal/modules/cronos/adapters/composicion"
	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronospg "vec-diputacion-granada/internal/modules/cronos/adapters/postgres"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
)

var ErrMontajeSaldoEfectosC3 = errors.New("bootstrap: saldo con efectos C3 no disponible")

// Reemplaza únicamente la lectura propia de saldo después de verificar CRN25.
// El resolutor vuelve a obtener la misma identidad y concesión V3 de C3.
func montarSaldoEfectosC3(ctx context.Context, rutas map[string]http.Handler, a *autoridadCronosEmpleadoDesarrollo,
	ejecutor *pgxpool.Pool, autorizador *cronoscomp.AutorizadorCronos,
	canal cronosdomain.AcreditacionCanalMarcaje, zona *time.Location) error {
	if ctx == nil || rutas == nil || a == nil || a.base == nil || ejecutor == nil || autorizador == nil || zona == nil ||
		rutas[cronoshttp.RutaConsultarSaldoPropio] == nil || cronospg.PreflightSaldoEfectosC3(ctx, ejecutor) != nil {
		return ErrMontajeSaldoEfectosC3
	}
	repo, err := cronospg.NuevoRepositorioConsultaSaldoConEfectosC3(ejecutor)
	if err != nil {
		return ErrMontajeSaldoEfectosC3
	}
	servicio, err := cronosapp.NuevoServicioConsultaSaldo(repo, relojCronosEmpleado{}, zona)
	if err != nil {
		return ErrMontajeSaldoEfectosC3
	}
	resolutor, err := cronoscomp.NuevoResolutorPeticionCronos(seguridadCronosEmpleadoDesarrollo{autoridad: a}, autorizador, canal)
	if err != nil {
		return ErrMontajeSaldoEfectosC3
	}
	manejador, err := cronoshttp.NuevoManejadorSaldoPropio(servicio, resolutor)
	if err != nil {
		return ErrMontajeSaldoEfectosC3
	}
	rutas[cronoshttp.RutaConsultarSaldoPropio] = manejador
	return nil
}
