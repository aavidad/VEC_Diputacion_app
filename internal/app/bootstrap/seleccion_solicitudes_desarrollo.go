package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	seleccioncatalogo "vec-diputacion-granada/internal/modules/seleccion/adapters/catalogo"
	seleccionpg "vec-diputacion-granada/internal/modules/seleccion/adapters/postgres"
	seleccionapp "vec-diputacion-granada/internal/modules/seleccion/application"
	seleccionports "vec-diputacion-granada/internal/modules/seleccion/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Selección («Convoca integrado», fase 1) en vec-server: solicitud de
// participación de la persona en la superficie personal y consulta de RRHH en
// la interna. Solo se compone con VEC_SELECCION_SOLICITUDES_ENABLED=true
// (doble llave y catálogo de convocatorias); encendido, el arranque comprueba
// sus migraciones y se detiene nombrando la primera que falte.

// ErrSeleccionMigracionesNoDisponibles detiene el arranque si falta alguna
// migración de Selección; los errores concretos la envuelven.
var ErrSeleccionMigracionesNoDisponibles = errors.New("bootstrap: solicitudes de Selección sin sus migraciones")

var (
	ErrSeleccionFaltaAD389       = fmt.Errorf("%w: falta AD3-89 (consumidor de la solicitud propia; requiere antes deploy/postgresql/seleccion/roles_up.sql)", ErrSeleccionMigracionesNoDisponibles)
	ErrSeleccionFaltaAD390       = fmt.Errorf("%w: falta AD3-90 (consumidor de la consulta de RRHH)", ErrSeleccionMigracionesNoDisponibles)
	ErrSeleccionFaltaSeleccion1  = fmt.Errorf("%w: falta Selección 000001 o el LOGIN de Bolsa no es miembro de vec_seleccion_ejecutor", ErrSeleccionMigracionesNoDisponibles)
	errSeleccionComprobacionRota = fmt.Errorf("%w: no se pudo comprobar el catálogo", ErrSeleccionMigracionesNoDisponibles)
	errSeleccionNoDisponible     = errors.New("bootstrap: solicitudes de Selección no disponibles")
)

// seleccionSolicitudesSolicitada dice si el selector está encendido y es
// válido. Un valor inválido se rechaza antes, en la validación de selectores.
func seleccionSolicitudesSolicitada(cfg config.Config) bool {
	activo, err := cfg.SeleccionSolicitudesDesarrolloActivo()
	return err == nil && activo
}

// comprobarMigracionesSeleccionDesarrollo se ejecuta con el LOGIN ejecutor de
// Bolsa, que usa Selección: nombra la primera migración ausente en el orden
// de instalación (AD3-89, AD3-90, Selección 000001).
func comprobarMigracionesSeleccionDesarrollo(ctx context.Context, pool *pgxpool.Pool) error {
	estado, err := seleccionpg.ComprobarMigraciones(ctx, pool)
	if err != nil {
		return errors.Join(errSeleccionComprobacionRota, err)
	}
	switch {
	case !estado.AD389:
		return ErrSeleccionFaltaAD389
	case !estado.AD390:
		return ErrSeleccionFaltaAD390
	case !estado.Seleccion1:
		return ErrSeleccionFaltaSeleccion1
	}
	return nil
}

// prepararSeleccionDesarrollo comprueba las migraciones y publica las
// convocatorias del catálogo. La publicación es idempotente: el mismo
// catálogo reutiliza las versiones vigentes y un segundo arranque no cambia
// nada.
func prepararSeleccionDesarrollo(ctx context.Context, cfg config.Config, bolsa *pgxpool.Pool) (*seleccionpg.Repositorio, error) {
	if err := comprobarMigracionesSeleccionDesarrollo(ctx, bolsa); err != nil {
		slog.Error("solicitudes de Selección encendidas sin sus migraciones", "causa", err)
		return nil, err
	}
	fuente, err := seleccioncatalogo.NuevoFichero(cfg.Normalize().SeleccionConvocatoriasSourcePath)
	if err != nil {
		slog.Error("catálogo de convocatorias de Selección no válido", "causa", err)
		return nil, err
	}
	repositorio, err := seleccionpg.NuevoRepositorio(bolsa)
	if err != nil {
		return nil, err
	}
	publicadas, err := seleccionapp.PublicarConvocatorias(ctx, fuente, repositorio)
	if err != nil {
		slog.Error("convocatorias de Selección no publicadas", "causa", err)
		return nil, err
	}
	nuevas := 0
	for _, p := range publicadas {
		if p.Nueva {
			nuevas++
		}
	}
	slog.Info("convocatorias de Selección publicadas", "vigentes", len(publicadas), "versiones_nuevas", nuevas)
	return repositorio, nil
}

// descriptoresMaterialSeleccionDesarrollo declara las cinco audiencias de
// AD3-89 y AD3-90 en el catálogo común de material.
func descriptoresMaterialSeleccionDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	pares := append(seleccionports.AccionesPropias(), seleccionports.AccionesRRHH()...)
	descriptores := make([]descriptorMaterialConsumidorV3Desarrollo, 0, len(pares))
	for _, par := range pares {
		nombre := strings.ReplaceAll(strings.TrimPrefix(par[0], "seleccion."), "_", "-")
		nombre = strings.ReplaceAll(nombre, ".", "-")
		descriptores = append(descriptores, descriptorMaterialConsumidorV3Desarrollo{
			Audiencia:        par[1],
			Dominio:          "vec.seleccion." + strings.ReplaceAll(nombre, "-", "_") + ".desarrollo.capacidad-v3",
			Prefijo:          "clave:capacidad:seleccion-" + nombre + ":",
			ProveedorNominal: "proveedor-material-seleccion-" + nombre,
		})
	}
	return descriptores
}

// motivoSeleccionPropiaDesarrollo motiva las acciones propias de la persona
// (AD3-89) en la política de su perfil.
func motivoSeleccionPropiaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_seleccion_solicitudes_propias", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-seleccion-solicitudes-propias-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "seleccion-solicitud-propia"),
	}
}

// motivoSeleccionRRHHDesarrollo motiva la consulta de RRHH (AD3-90).
func motivoSeleccionRRHHDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_seleccion_consulta_rrhh", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-seleccion-consulta-rrhh-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "seleccion-consulta-rrhh"),
	}
}

// concesionesSeleccionPropiaDesarrollo concede las tres acciones propias
// sobre 'mis-solicitudes:<persona>': sin campos ni obligaciones.
func concesionesSeleccionPropiaDesarrollo() []dominiovec.ConcesionRol {
	concesiones := make([]dominiovec.ConcesionRol, 0, 3)
	for _, par := range seleccionports.AccionesPropias() {
		concesiones = append(concesiones, dominiovec.ConcesionRol{
			Accion: par[0], ModuloID: seleccionports.ModuloSeleccion, TipoRecurso: seleccionports.TipoRecursoSolicitudesPropias,
			Finalidades: []string{seleccionports.FinalidadSolicitudesPropias}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
		})
	}
	return concesiones
}

// esAccionSeleccionPropia dice si la acción es una de las de AD3-89.
func esAccionSeleccionPropia(accion string) bool {
	for _, par := range seleccionports.AccionesPropias() {
		if par[0] == accion {
			return true
		}
	}
	return false
}

// audienciaSeleccion devuelve la audiencia de una acción de Selección.
func audienciaSeleccion(accion string) (string, bool) {
	for _, par := range append(seleccionports.AccionesPropias(), seleccionports.AccionesRRHH()...) {
		if par[0] == accion {
			return par[1], true
		}
	}
	return "", false
}
