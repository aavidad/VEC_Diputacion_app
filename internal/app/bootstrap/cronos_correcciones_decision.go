package bootstrap

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	cronoscomp "vec-diputacion-granada/internal/modules/cronos/adapters/composicion"
	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronospg "vec-diputacion-granada/internal/modules/cronos/adapters/postgres"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
)

// Cada acción C5b conserva una audiencia, clave derivada y proveedor propios.
func descriptoresMaterialDecisionCorreccionesCronos() [6]descriptorMaterialConsumidorV3Desarrollo {
	return [6]descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: cronosapp.AudienciaBandejaCorrecciones, Dominio: "vec.cronos.correccion-bandeja.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-correccion-bandeja:", ProveedorNominal: "proveedor-material-cronos-correccion-bandeja"},
		{Audiencia: cronosapp.AudienciaDecisionResponsableCorreccion, Dominio: "vec.cronos.correccion-responsable.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-correccion-responsable:", ProveedorNominal: "proveedor-material-cronos-correccion-responsable"},
		{Audiencia: cronosapp.AudienciaResolucionRRHHCorreccion, Dominio: "vec.cronos.correccion-rrhh.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-correccion-rrhh:", ProveedorNominal: "proveedor-material-cronos-correccion-rrhh"},
		{Audiencia: cronosapp.AudienciaAplicacionCorreccion, Dominio: "vec.cronos.correccion-aplicar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-correccion-aplicar:", ProveedorNominal: "proveedor-material-cronos-correccion-aplicar"},
		{Audiencia: cronosapp.AudienciaHistoriaCorreccionPropia, Dominio: "vec.cronos.correccion-historia-propia.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-correccion-historia-propia:", ProveedorNominal: "proveedor-material-cronos-correccion-historia-propia"},
		{Audiencia: cronosapp.AudienciaHistoriaCorreccionEquipo, Dominio: "vec.cronos.correccion-historia-equipo.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:cronos-correccion-historia-equipo:", ProveedorNominal: "proveedor-material-cronos-correccion-historia-equipo"},
	}
}

var consumidoresDecisionCorreccionesCronos = [6]string{
	"consumir_cronos_bandeja_correcciones_v3_atestada",
	"registrar_y_consumir_cronos_decision_responsable_v3_atestada",
	"registrar_y_consumir_cronos_resolucion_rrhh_v3_atestada",
	"registrar_y_consumir_cronos_aplicacion_correccion_v3_atestada",
	"consumir_cronos_historia_correccion_propia_v3_atestada",
	"consumir_cronos_historia_correccion_equipo_v3_atestada",
}

func preflightDecisionCorreccionesCronos(ctx context.Context, ejecutor *pgxpool.Pool) error {
	if ctx == nil || ctx.Err() != nil || ejecutor == nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	var ok bool
	if ejecutor.QueryRow(ctx, `SELECT bool_and(pg_catalog.has_function_privilege(f,'EXECUTE'))
 FROM pg_catalog.unnest(ARRAY[
  'vec_cronos_v1.consultar_bandeja_correcciones_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.registrar_actuacion_correccion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.consultar_historia_correccion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)']) f`).Scan(&ok) != nil || !ok {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	// El LOGIN de Cronos no recibe USAGE del esquema V3. La consulta común
	// localiza cada firma exacta en pg_proc y verifica EXECUTE por OID.
	for _, consumidor := range consumidoresDecisionCorreccionesCronos {
		if ejecutor.QueryRow(ctx, consultaPermisoConsumoReciboCorreccionCronos, consumidor, "vec_cronos_v1_propietario").Scan(&ok) != nil || !ok {
			return ErrComposicionCronosEmpleadoNoDisponible
		}
	}
	return nil
}

func prepararDecisionCorreccionesCronosParaPublicacion(ctx context.Context, cfg config.Config) error {
	if !cfg.CronosCorreccionesDecisionDesarrolloActiva() || ctx == nil || ctx.Err() != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	base, err := leerConfigBaseCronosParaRecuperacion(cfg)
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	ejecutor, _, err := abrirPoolRutasDietas(ctx, base.DSNCronos, "vec_cronos_v1_ejecutor")
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	defer ejecutor.Close()
	return preflightDecisionCorreccionesCronos(ctx, ejecutor)
}

// Publica las seis audiencias después del lote base y C5a. Un fallo revierte
// sólo este lote: no altera los proveedores anteriores ni el plazo de CT.
func publicarMaterialDecisionCorreccionesCronosEnLote(ctx context.Context, gobierno *pgxpool.Pool,
	material materialAtestacionContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo,
) ([6]*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	var vacio [6]*proveedorMaterialAltaContratacionTemporalDesarrollo
	if ctx == nil || ctx.Err() != nil || gobierno == nil {
		return vacio, ErrComposicionCronosEmpleadoNoDisponible
	}
	descriptores := descriptoresMaterialDecisionCorreccionesCronos()
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptores[:])
	if err != nil {
		return vacio, ErrComposicionCronosEmpleadoNoDisponible
	}
	var derivados [6]materialAtestacionContratacionTemporalDesarrollo
	defer func() {
		for i := range derivados {
			borrarBytes(derivados[i].claveHMAC)
		}
	}()
	for i, descriptor := range descriptores {
		if _, ok := catalogo.descriptorPara(descriptor.Audiencia); !ok {
			return vacio, ErrComposicionCronosEmpleadoNoDisponible
		}
		derivados[i], err = derivarMaterialConsumidorV3Desarrollo(material, descriptor)
		if err != nil {
			return vacio, ErrComposicionCronosEmpleadoNoDisponible
		}
	}
	var proveedores [6]*proveedorMaterialAltaContratacionTemporalDesarrollo
	err = ejecutarTransaccionGobiernoCTDesarrolloUnaVez(ctx, gobierno, func(tx pgx.Tx) error {
		for i := range derivados {
			if err := publicarGobiernoAtestacionCTEnTxDesarrollo(ctx, tx, &derivados[i]); err != nil {
				return err
			}
			var err error
			proveedores[i], err = nuevoProveedorMaterialAutorizacionBaseDesarrollo(derivados[i], reloj)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return vacio, ErrComposicionCronosEmpleadoNoDisponible
	}
	return proveedores, nil
}

func montarDecisionCorreccionesCronos(ctx context.Context, rutas map[string]http.Handler, d dependenciasCronosEmpleado) error {
	if rutas == nil || d.ejecutor == nil || d.zona == nil || d.autorizador == nil ||
		preflightDecisionCorreccionesCronos(ctx, d.ejecutor) != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	repositorio, err := cronospg.NuevoRepositorioCorreccionesDecisiones(d.ejecutor, d.zona.String())
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	servicio, err := cronosapp.NuevoServicioDecisionCorrecciones(repositorio, relojCronosEmpleado{}, d.zona)
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	resolvedor, err := cronoscomp.NuevoResolutorPeticionCronos(d.identidad, d.autorizador, d.canal)
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	manejador, err := cronoshttp.NuevoManejadorDecisionCorrecciones(servicio, resolvedor)
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	rutas[cronoshttp.RutaBandejaCorrecciones] = manejador
	rutas[cronoshttp.RutaActuacionCorrecciones] = manejador
	rutas[cronoshttp.RutaHistoriaCorrecciones] = manejador
	return nil
}

func rutasDecisionCorreccionesNoDisponibles(rutas map[string]http.Handler) {
	cerrado := reciboCorreccionNoDisponible()
	rutas[cronoshttp.RutaBandejaCorrecciones] = cerrado
	rutas[cronoshttp.RutaActuacionCorrecciones] = cerrado
	rutas[cronoshttp.RutaHistoriaCorrecciones] = cerrado
}
