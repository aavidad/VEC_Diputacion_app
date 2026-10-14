package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	cronoscomp "vec-diputacion-granada/internal/modules/cronos/adapters/composicion"
	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronospg "vec-diputacion-granada/internal/modules/cronos/adapters/postgres"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
	personalcomp "vec-diputacion-granada/internal/modules/personal/adapters/composicion"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	core "vec-diputacion-granada/internal/vec/domain"
)

// La configuración de C5a vive aparte: un error suyo no afecta el arranque
// ni las rutas ya montadas de Cronos.
type configuracionRecuperacionCorreccionCronos struct {
	redaccionMaterialRutasDietas
	Version        int                            `json:"version"`
	Autoridad      string                         `json:"autoridad"`
	DSNPersonal    string                         `json:"dsn_personal"`
	MotivoPersonal core.ReferenciaEntradaCatalogo `json:"motivo_personal"`
	MotivoCronos   core.ReferenciaEntradaCatalogo `json:"motivo_cronos"`
}

func leerRecuperacionCorreccionCronos(cfg config.Config) (configuracionRecuperacionCorreccionCronos, error) {
	var vacia configuracionRecuperacionCorreccionCronos
	contenido, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "cronos-recuperacion-correccion.json"), 32<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	defer borrarBytes(contenido)
	var c configuracionRecuperacionCorreccionCronos
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(new(any)), io.EOF) || c.Version != 1 ||
		c.Autoridad != AutoridadNoAutoritativa || c.DSNPersonal == "" ||
		!core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoPersonal) ||
		!core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoCronos) ||
		c.MotivoPersonal.CatalogoID != c.MotivoCronos.CatalogoID {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	return c, nil
}

// Comprueba los consumidores reales y ambos motivos antes de publicar
// cualquiera de las dos audiencias.
func prepararRecuperacionCorreccionCronosParaPublicacion(ctx context.Context, cfg config.Config) error {
	c, err := leerRecuperacionCorreccionCronos(cfg)
	if err != nil || ctx == nil || ctx.Err() != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	base, err := leerConfigBaseCronosParaRecuperacion(cfg)
	if err != nil || c.MotivoPersonal.CatalogoID != base.Motivos.Saldo.CatalogoID ||
		c.MotivoCronos.CatalogoID != base.Motivos.Saldo.CatalogoID {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	personal, _, err := abrirPoolRutasDietas(ctx, c.DSNPersonal, "vec_personal_ejecutor")
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	defer personal.Close()
	cronos, _, err := abrirPoolRutasDietas(ctx, base.DSNCronos, "vec_cronos_v1_ejecutor")
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	defer cronos.Close()
	motivos, _, err := abrirPoolRutasDietas(ctx, base.DSNMotivos, "vec_autorizacion_motivos_evaluador")
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	defer motivos.Close()
	if err := preflightRecuperacionCorreccionCronos(ctx, cronos, personal); err != nil {
		return err
	}
	validador, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivos, c.MotivoPersonal.CatalogoID)
	ahora := time.Now().UTC()
	if err != nil || validador.ValidarReferenciaMotivoAutorizacionV2(ctx, c.MotivoPersonal, ahora) != nil ||
		validador.ValidarReferenciaMotivoAutorizacionV2(ctx, c.MotivoCronos, ahora) != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	return nil
}

func leerConfigBaseCronosParaRecuperacion(cfg config.Config) (configuracionCronosEmpleadoDesarrollo, error) {
	var vacia configuracionCronosEmpleadoDesarrollo
	contenido, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "cronos-empleado.json"), 256<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	defer borrarBytes(contenido)
	var c configuracionCronosEmpleadoDesarrollo
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || !errors.Is(d.Decode(new(any)), io.EOF) || c.Version != 1 ||
		c.DSNCronos == "" || c.DSNMotivos == "" {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	return c, nil
}

// La identidad de Personal sale del holder que la frontera Cronos guardó
// para esta misma petición. No se construye desde cabeceras libres.
type identidadVinculoRecuperacionCronos struct {
	autoridad *autoridadCronosEmpleadoDesarrollo
}

func (i identidadVinculoRecuperacionCronos) ResolverIdentidadVinculoCRN11(ctx context.Context) (personalcomp.IdentidadRegistradaVinculoCRN11, error) {
	var vacia personalcomp.IdentidadRegistradaVinculoCRN11
	if ctx == nil || i.autoridad == nil || i.autoridad.reloj == nil {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	holder, ok := ctx.Value(claveContextoCronosEmpleado{}).(contextoCronosEmpleado)
	if !ok || holder.autoridad != i.autoridad || holder.seguridad.Resultado.Validar() != nil ||
		holder.seguridad.Vinculo.ValidarPara(holder.seguridad.Resultado) != nil ||
		!holder.seguridad.Vinculo.VigenteEn(i.autoridad.reloj.Ahora(), holder.seguridad.Resultado) {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	resultado, err := holder.seguridad.Resultado.Clonar()
	if err != nil {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	return personalcomp.IdentidadRegistradaVinculoCRN11{Vinculo: holder.seguridad.Vinculo, Resultado: resultado}, nil
}

func preflightRecuperacionCorreccionCronos(ctx context.Context, cronos, personal *pgxpool.Pool) error {
	if ctx == nil || cronos == nil || personal == nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	var ok bool
	if cronos.QueryRow(ctx, `SELECT has_function_privilege('vec_cronos_v1.recuperar_recibo_correccion_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')`).Scan(&ok) != nil || !ok {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	if personal.QueryRow(ctx, `SELECT has_function_privilege('vec_personal.consultar_vinculo_propio_historico_crn11_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')`).Scan(&ok) != nil || !ok {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	for _, consumo := range []struct {
		pool        *pgxpool.Pool
		funcion     string
		propietario string
	}{
		{cronos, "consumir_cronos_correccion_recibo_propio_v3_atestada", "vec_cronos_v1_propietario"},
		{personal, "consumir_vinculo_propio_crn11_v3_atestada", "vec_personal_propietario"},
	} {
		// El LOGIN nominal carece de USAGE sobre el esquema V3. La firma se
		// encuentra por catálogo y se comprueba por OID, sin abrir el esquema.
		if consumo.pool.QueryRow(ctx, consultaPermisoConsumoReciboCorreccionCronos, consumo.funcion, consumo.propietario).Scan(&ok) != nil || !ok {
			return ErrComposicionCronosEmpleadoNoDisponible
		}
	}
	return nil
}

const consultaPermisoConsumoReciboCorreccionCronos = `SELECT EXISTS (
 SELECT 1 FROM pg_catalog.pg_proc p
 JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 JOIN pg_catalog.pg_roles propietario ON propietario.rolname=$2
 JOIN pg_catalog.pg_roles propietario_v3 ON propietario_v3.rolname='vec_autorizacion_atestada_v3_propietario'
 WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname=$1
  AND p.pronargs=10 AND p.provariadic=0 AND p.prokind='f'
  AND p.proowner=propietario_v3.oid AND p.prosecdef
  AND p.proargtypes[0]='pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[1]='pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[2]='pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[3]='pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[4]='pg_catalog.numeric'::pg_catalog.regtype
  AND p.proargtypes[5]='pg_catalog.numeric'::pg_catalog.regtype
  AND p.proargtypes[6]='pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[7]='pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[8]='pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[9]='pg_catalog.bytea'::pg_catalog.regtype
  AND pg_catalog.has_function_privilege(propietario.oid,p.oid,'EXECUTE')
)`

// publicarMaterialRecuperacionCorreccionCronosEnLote añade sólo las dos
// audiencias C5a y confirma ambas en una transacción propia. Su error se
// limita al GET opcional; el lote anterior de Cronos ya está confirmado.
func publicarMaterialRecuperacionCorreccionCronosEnLote(ctx context.Context, gobierno *pgxpool.Pool, material materialAtestacionContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo, catalogo catalogoMaterialAutorizacionComunDesarrollo) ([2]*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	var vacio [2]*proveedorMaterialAltaContratacionTemporalDesarrollo
	if ctx == nil || gobierno == nil {
		return vacio, ErrComposicionCronosEmpleadoNoDisponible
	}
	descriptores := descriptoresMaterialRecuperacionCorreccionCronosDesarrollo()
	var derivados [2]materialAtestacionContratacionTemporalDesarrollo
	defer func() {
		for i := range derivados {
			borrarBytes(derivados[i].claveHMAC)
		}
	}()
	for i, descriptor := range descriptores {
		if _, ok := catalogo.descriptorPara(descriptor.Audiencia); !ok {
			return vacio, ErrComposicionCronosEmpleadoNoDisponible
		}
		var err error
		derivados[i], err = derivarMaterialConsumidorV3Desarrollo(material, descriptor)
		if err != nil {
			return vacio, ErrComposicionCronosEmpleadoNoDisponible
		}
	}
	var proveedores [2]*proveedorMaterialAltaContratacionTemporalDesarrollo
	err := ejecutarTransaccionGobiernoCTDesarrolloUnaVez(ctx, gobierno, func(tx pgx.Tx) error {
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

func montarRecuperacionCorreccionCronos(ctx context.Context, a *autoridadCronosEmpleadoDesarrollo, rutas map[string]http.Handler,
	cronos, personal *pgxpool.Pool, zona *time.Location, emisor emisorCronosEmpleadoDesarrollo,
	autorizador *cronoscomp.AutorizadorCronos, motivoPersonal core.ReferenciaEntradaCatalogo, canal cronosdomain.AcreditacionCanalMarcaje,
) error {
	if err := preflightRecuperacionCorreccionCronos(ctx, cronos, personal); err != nil || a == nil || rutas == nil || zona == nil || autorizador == nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	lector, err := personalcomp.ComponerLectorVinculoPropioCRN11(personalcomp.DependenciasLectorVinculoCRN11{
		Identidad: identidadVinculoRecuperacionCronos{autoridad: a}, Emisor: emisor, Motivo: motivoPersonal,
		Personal: personal, Ahora: func() time.Time { return a.reloj.Ahora() },
	})
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	repositorio, err := cronospg.NuevoRepositorioCorreccionesPropias(cronos, zona.String())
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	servicio, err := cronosapp.NuevoServicioCorreccionesConVinculoHistorico(repositorio, relojCronosEmpleado{}, lector)
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	resolvedor, err := cronoscomp.NuevoResolutorPeticionCronos(seguridadCronosEmpleadoDesarrollo{autoridad: a}, autorizador, canal)
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	manejador, err := cronoshttp.NuevoManejadorRecuperacionCorreccionPropia(servicio, resolvedor)
	if err != nil {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	rutas[cronoshttp.RutaRecuperarReciboCorreccionPropia] = manejador
	return nil
}

// Cierra sólo el GET cuando C5a no se puede componer.
func reciboCorreccionNoDisponible() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		responderDenegacionCronosEmpleado(w, http.StatusServiceUnavailable, "no_disponible")
	})
}
