package bootstrap

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/reglas"
)

// La ruta común /Reglas conserva su único resolutor. Antes de abrir HTTP se
// conecta a la lectura CT148 para que la vista y todos los consumidores vean
// las mismas reglas efectivas. El puntero se comparte desde composición raíz;
// sólo se sustituye aquí durante el arranque, cuando no hay peticiones.
func conectarAjustesReglasCTAlResolutor(cfg config.Config, destino *reglas.Resolutor,
	pool *pgxpool.Pool, calendarios calendariosports.ConsultaCalendarios,
	reloj reglas.Reloj) error {
	if destino == nil || pool == nil || reloj == nil || destino.CatalogoID() != reglas.CatalogoContratacionTemporal {
		return errMontajeAjustesReglasCT
	}
	rutas, activas, err := cfg.ReglasEjemploDesarrollo()
	if err != nil || !activas || rutas.CTSourcePath == "" {
		return errMontajeAjustesReglasCT
	}
	base, err := fichero.NuevaConsultaCatalogos(rutas.CTSourcePath)
	if err != nil {
		return errMontajeAjustesReglasCT
	}
	ajustes, err := postgresct.NuevaConsultaAjustesReglasPostgreSQL(pool)
	if err != nil {
		return errMontajeAjustesReglasCT
	}
	compuesto, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: base, Metadatos: base, CatalogoID: reglas.CatalogoContratacionTemporal,
		ModuloID: reglas.ModuloContratacionTemporal, Reloj: reloj,
		Calculadora:   calculadoraPlazosCalendarios{consulta: calendarios},
		MunicipioSede: reglas.MunicipioSedeDiputacion, Ajustes: ajustes,
	})
	if err != nil {
		return errMontajeAjustesReglasCT
	}
	*destino = *compuesto
	return nil
}
