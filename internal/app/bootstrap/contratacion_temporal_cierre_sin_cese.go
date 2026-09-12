package bootstrap

import (
	"github.com/jackc/pgx/v5/pgxpool"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

// ConfiguracionCierreAdministrativoSinCeseDesarrollo contiene exclusivamente
// fronteras ya operativas. El proveedor obtiene una concesión V3 actual; no
// deriva permisos del canal HTTP ni del alta.
type ConfiguracionCierreAdministrativoSinCeseDesarrollo struct {
	EjecutorCT *pgxpool.Pool
	Proveedor  postgresct.ProveedorAutorizacionCierreAdministrativo
}

// NuevoServicioCierreAdministrativoSinCeseDesarrollo reutiliza el pool CT
// existente. Si no hay proveedor V3, publicación sucesora o libro durable, la
// transacción rechaza la preparación y la raíz no debe publicar la ruta.
func NuevoServicioCierreAdministrativoSinCeseDesarrollo(c ConfiguracionCierreAdministrativoSinCeseDesarrollo) (*appct.ServicioCierreAdministrativo, error) {
	if c.EjecutorCT == nil || dependenciaBootstrapNula(c.Proveedor) {
		return nil, appct.ErrServicioCierreAdministrativoInvalido
	}
	transaccion, err := postgresct.NuevaTransaccionCierreAdministrativoPostgreSQL(c.EjecutorCT, c.Proveedor)
	if err != nil {
		return nil, appct.ErrServicioCierreAdministrativoInvalido
	}
	return appct.NuevoServicioCierreAdministrativo(transaccion)
}
