package bootstrap

import (
	"github.com/jackc/pgx/v5/pgxpool"

	ctpostgres "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

type ConfiguracionComposicionVinculoCategoriaRPT struct {
	Proveedor                          ConfiguracionProveedorVinculoCategoriaRPT
	Sesiones                           FuenteSesionVinculoCategoriaRPT
	Contextos                          ctports.ResolutorContextoAutorizacionAltaV3
	ConsultaCT, RegistroCT, LecturaRPT *confianza.EmisorMaterialAutorizacionAtestadaV3
	Pool                               *pgxpool.Pool
	Reloj                              ctports.Reloj
}

// ComponerVinculoCategoriaRPTNominal devuelve el proveedor verificable y el
// servicio de aplicación. La fuente de sesión debe provenir del canal VEC
// acreditado; la configuración y las concesiones se fijan antes de servir.
func ComponerVinculoCategoriaRPTNominal(c ConfiguracionComposicionVinculoCategoriaRPT) (*ProveedorVinculoCategoriaRPT, *ctapp.ServicioVinculoCategoriaRPT, error) {
	a, e := NuevoProveedorVinculoCategoriaRPT(c.Proveedor, c.Sesiones, c.Contextos, c.ConsultaCT, c.RegistroCT, c.LecturaRPT, c.Pool, c.Reloj)
	if e != nil {
		return nil, nil, e
	}
	s, e := ComponerVinculoCategoriaRPT(c.Pool, a, c.Reloj)
	if e != nil {
		return nil, nil, e
	}
	return a, s, nil
}

// ComponerVinculoCategoriaRPT conecta los dos consumidores nominales al mismo
// principal. La autoridad se inyecta desde la composicion de identidad V3
// con sus perfiles fijos; esta funcion no publica concesiones ni crea sesiones.
func ComponerVinculoCategoriaRPT(
	pool *pgxpool.Pool,
	autoridad ctports.AutoridadVinculoCategoriaRPT,
	reloj ctports.Reloj,
) (*ctapp.ServicioVinculoCategoriaRPT, error) {
	if pool == nil || autoridad == nil || reloj == nil {
		return nil, ctports.ErrVinculoCategoriaRPTNoDisponible
	}
	ct, e := ctpostgres.NuevaFuenteVinculoCategoriaRPTPostgreSQL(pool)
	if e != nil {
		return nil, e
	}
	rpt, e := ctpostgres.NuevaFuentePublicacionCategoriaRPTPostgreSQL(pool)
	if e != nil {
		return nil, e
	}
	return ctapp.NuevoServicioVinculoCategoriaRPT(autoridad, ct, rpt, reloj)
}
