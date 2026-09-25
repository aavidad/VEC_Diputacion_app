package interna

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type componentesPersonalB2 struct {
	proveedor                               *internactproveedores.ProveedorAutorizacionPersonalB2
	ficha, vacantes, alta, hecho, catalogos http.Handler
}

// montarPersonalB2Gobernado activa todas las capacidades B2 o ninguna. Su
// material y SQL son opcionales para CT; una carencia deja B2 fuera de la
// lista de rutas servidas sin abrir una sustitución de desarrollo.
func montarPersonalB2Gobernado(
	ctx context.Context, directorio, loginPersonal string, pool *pgxpool.Pool,
	base *internactproveedores.Proveedores, fuente *internagobierno.FuenteF1,
	auditoria vecports.RegistradorAuditoriaFronteraRutaExacta, reloj relojGobiernoInterno,
) (componentesPersonalB2, bool) {
	var vacio componentesPersonalB2
	if ctx == nil || ctx.Err() != nil || directorio == "" || pool == nil || base == nil || fuente == nil ||
		interfazNulaIdentidadOffline(auditoria) {
		return vacio, false
	}
	material, err := internactproveedores.CargarMaterialPersonalB2(directorio)
	if err != nil {
		return vacio, false
	}
	defer material.Cerrar()
	if acreditarPoolPersonalB2(ctx, pool, loginPersonal) != nil {
		return vacio, false
	}
	proveedor, err := internactproveedores.ConstruirPersonalB2(ctx, internactproveedores.ConfiguracionPersonalB2{
		Material: material, Base: base, Fuente: fuente, Reloj: reloj,
	})
	if err != nil {
		return vacio, false
	}
	transferido := false
	defer func() {
		if !transferido {
			proveedor.Cerrar()
		}
	}()
	repositorio, err := personalpg.NuevoRepositorioRegistroEmpleadoB2PostgreSQL(pool)
	if err != nil {
		return vacio, false
	}
	consulta, err := personalapp.NuevoServicioRegistroEmpleadoB2(proveedor, repositorio)
	if err != nil {
		return vacio, false
	}
	actos, err := personalapp.NuevoServicioActosRegistroEmpleadoB2(proveedor, repositorio)
	if err != nil {
		return vacio, false
	}
	catalogos, err := personalapp.NuevoServicioCatalogosRegistroEmpleadoB2(proveedor, repositorio)
	if err != nil {
		return vacio, false
	}
	autoridad := internactproveedores.AutoridadContextoRegistroEmpleadoB2{Fuente: fuente}
	auditor := internactproveedores.AuditorDenegacionRegistroEmpleadoB2{Registrador: auditoria}
	ficha, err := httpapi.NewHandlerFichaEmpleadoB2(autoridad, consulta, auditor)
	if err != nil {
		return vacio, false
	}
	vacantes, err := httpapi.NewHandlerVacantesEmpleadoB2(autoridad, consulta, auditor)
	if err != nil {
		return vacio, false
	}
	alta, err := httpapi.NewHandlerAltaEmpleadoB2(autoridad, actos, auditor)
	if err != nil {
		return vacio, false
	}
	hecho, err := httpapi.NewHandlerHechosEmpleadoB2(autoridad, actos, auditor)
	if err != nil {
		return vacio, false
	}
	handlerCatalogos, err := httpapi.NewHandlerCatalogosRegistroEmpleadoB2(autoridad, catalogos, auditor)
	if err != nil {
		return vacio, false
	}
	transferido = true
	return componentesPersonalB2{proveedor, ficha, vacantes, alta, hecho, handlerCatalogos}, true
}
