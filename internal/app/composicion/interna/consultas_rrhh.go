package interna

import (
	ctinterna "vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// dependenciasLecturasRRHH conserva las autoridades de aplicación y el pool
// nominal. El pool y el emisor deben proceder de proveedores productivos; la
// raíz no lee configuraciones de desarrollo ni fabrica material criptográfico.
type dependenciasLecturasRRHH struct {
	autoridad ctports.AutoridadContextoConsultaRRHH
	emisor    *ctports.EmisorMaterialConsultaRRHH
	pool      *postgresct.PoolConsultasRRHHPostgreSQL
	reloj     ctports.Reloj
}

func nuevasRutasConsultasRRHH(d dependenciasLecturasRRHH) ([]httpapi.RutaExacta, error) {
	sesion, err := postgresct.NuevaSesionConsultaRRHHPostgreSQL(d.pool)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	cuadro, err := ctapp.NuevoServicioConsultaCuadroRRHH(d.autoridad, d.emisor, sesion, d.reloj)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	detalle, err := ctapp.NuevoServicioConsultaDetalleRRHH(d.autoridad, d.emisor, sesion, d.reloj)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	rutas, err := ctinterna.NuevasRutasConsultasRRHH(ctinterna.DependenciasConsultasRRHH{
		Cuadro: cuadro, Detalle: detalle,
	})
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	return rutas, nil
}
