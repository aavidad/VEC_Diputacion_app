package bootstrap

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pgpersonal "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	apppersonal "vec-diputacion-granada/internal/modules/personal/application"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
)

// componerCesePersonalB2 monta el fin de la relación en Personal tras el cese
// de CT solo si la configuración privada fija cese_fecha_efecto. Usa la
// autoridad de actos B2 de siempre con su repositorio general (no el del plan
// de incorporación, que sólo admite las claves de ese plan).
func componerCesePersonalB2(c *archivoIncorporacionPersonalB2, actosPool *pgxpool.Pool, autoridad *autoridadIncorporacionPersonalB2,
	contratos inc.FuenteContratoPlanNominal, origen inc.LectorOrigenCesePersonalB2, ficha pp.FuenteFichaIncorporacionCT, reloj ct.Reloj) (*inc.CesePersonalB2, error) {
	if c == nil || c.CeseFechaEfecto == "" {
		return nil, nil
	}
	repositorio, e := pgpersonal.NuevoRepositorioRegistroEmpleadoB2PostgreSQL(actosPool)
	if e != nil {
		return nil, e
	}
	actos, e := apppersonal.NuevoServicioActosRegistroEmpleadoB2(autoridad, repositorio)
	if e != nil {
		return nil, e
	}
	return inc.NuevoCesePersonalB2(inc.ConfiguracionCesePersonalB2{Contratos: contratos, Origen: origen, Ficha: ficha,
		Actos: actos, Actores: autoridad, Reloj: reloj, Fecha: inc.ReglaFechaCesePersonalB2(c.CeseFechaEfecto)})
}

// finCesePersonalB2Desarrollo engancha el paso de Personal detrás del éxito
// del cese de CT, con los perfiles nominales B2 y en otra transacción.
type finCesePersonalB2Desarrollo struct {
	cese      *inc.CesePersonalB2
	soporte   *soporteAltaContratacionTemporalDesarrollo
	fronteras catalogoFronterasComunDesarrollo
}

// finCesePersonalB2 toma el enganche del único montaje de incorporación, si lo hay.
func finCesePersonalB2(incorporacion []ConfiguracionIncorporacionDesarrollo, soporte *soporteAltaContratacionTemporalDesarrollo, fronteras catalogoFronterasComunDesarrollo) *finCesePersonalB2Desarrollo {
	if len(incorporacion) != 1 || incorporacion[0].nominales == nil {
		return nil
	}
	return incorporacion[0].nominales.montajeB2.finCese(soporte, fronteras)
}

// finCese devuelve nil si el montaje B2 no existe o no compone el cese.
func (m *montajeIncorporacionPersonalB2) finCese(soporte *soporteAltaContratacionTemporalDesarrollo, fronteras catalogoFronterasComunDesarrollo) *finCesePersonalB2Desarrollo {
	if m == nil || m.cese == nil || soporte == nil {
		return nil
	}
	return &finCesePersonalB2Desarrollo{cese: m.cese, soporte: soporte, fronteras: fronteras}
}

// envolver devuelve el ejecutor de seguimiento con el cese enganchado. Los
// demás métodos (cierre, modificación, GINPIX, no incorporación) son los del
// servicio original, por lo que las rutas opcionales no cambian.
func (f *finCesePersonalB2Desarrollo) envolver(s *appct.ServicioOperacionesSeguimiento) httpct.EjecutorOperacionesSeguimiento {
	if f == nil || s == nil {
		return s
	}
	return &ejecutorCeseConPersonalB2{ServicioOperacionesSeguimiento: s, fin: f}
}

type ejecutorCeseConPersonalB2 struct {
	*appct.ServicioOperacionesSeguimiento
	fin *finCesePersonalB2Desarrollo
}

// errFinPersonalB2Pendiente: el cese consta en CT pero Personal no ha dado su
// recibo. La respuesta es 503 para que el cliente repita con la misma clave:
// CT devuelve su recibo original y Personal se reintenta sin duplicar.
var errFinPersonalB2Pendiente = errors.New("contratacion temporal: cese registrado; fin de la relacion en Personal pendiente")

func (e *ejecutorCeseConPersonalB2) RegistrarCese(ctx context.Context, s appct.SolicitudRegistrarCese) (ct.ReciboOperacionSeguimiento, error) {
	recibo, err := e.ServicioOperacionesSeguimiento.RegistrarCese(ctx, s)
	if err != nil {
		return recibo, err
	}
	if err := e.fin.finalizar(ctx, s, recibo); err != nil {
		return ct.ReciboOperacionSeguimiento{}, err
	}
	return recibo, nil
}

func (f *finCesePersonalB2Desarrollo) finalizar(ctx context.Context, s appct.SolicitudRegistrarCese, recibo ct.ReciboOperacionSeguimiento) error {
	if ctx == nil {
		return errFinPersonalB2Pendiente
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	nominal, ok := contextoNominalIncorporacionPersonalB2(ctx, f.soporte, f.fronteras, rutaPeticionIncorporacionB2{metodo: "POST", ruta: httpct.RutaCesesNombramiento})
	if !ok {
		return errors.Join(ct.ErrOperacionSeguimientoNoDisponible, errFinPersonalB2Pendiente)
	}
	_, err := f.cese.FinalizarRelacionPersonalB2(nominal, inc.SolicitudCesePersonalB2{Recibo: recibo,
		JustificanteRef: s.JustificanteRef, JustificanteSHA256: s.JustificanteSHA256})
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	// También una denegación: el cese de CT ya es firme y la clave debe poder
	// repetirse cuando se corrija el permiso; no se devuelve 403 aquí.
	return errors.Join(ct.ErrOperacionSeguimientoNoDisponible, errFinPersonalB2Pendiente, err)
}
