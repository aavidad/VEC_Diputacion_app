package interna

import (
	httpinterno "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// dependenciasLecturasRRHH conserva las autoridades de aplicación y el pool
// nominal. El pool y el emisor deben proceder de proveedores productivos; la
// raíz no lee configuraciones de desarrollo ni fabrica material criptográfico.
type dependenciasLecturasRRHH struct {
	emisor *ctports.EmisorMaterialConsultaRRHH
	pool   *postgresct.PoolConsultasRRHHPostgreSQL
}

func nuevasRutasConsultasRRHH(d dependenciasLecturasRRHH, autoridad autoridadContextoConsultaRRHH) ([]httpapi.RutaExacta, error) {
	sesion, err := postgresct.NuevaSesionConsultaRRHHPostgreSQL(d.pool)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	cuadro, err := ctapp.NuevoServicioConsultaCuadroRRHH(autoridad, d.emisor, sesion, autoridad.actor.reloj)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	detalle, err := ctapp.NuevoServicioConsultaDetalleRRHH(autoridad, d.emisor, sesion, autoridad.actor.reloj)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	return rutasConsultoresRRHH(cuadro, detalle)
}

func rutasConsultoresRRHH(
	cuadro httpinterno.ConsultorCuadroRRHH,
	detalle httpinterno.ConsultorDetalleRRHH,
) ([]httpapi.RutaExacta, error) {
	manejadorCuadro, err := httpinterno.NuevoManejadorConsultaCuadroRRHH(cuadro)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	manejadorDetalle, err := httpinterno.NuevoManejadorConsultaDetalleRRHH(detalle)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	return []httpapi.RutaExacta{
		{Ruta: httpinterno.RutaConsultaCuadroRRHH, Manejador: manejadorCuadro},
		{Ruta: httpinterno.RutaConsultaDetalleRRHH, Manejador: manejadorDetalle},
	}, nil
}
