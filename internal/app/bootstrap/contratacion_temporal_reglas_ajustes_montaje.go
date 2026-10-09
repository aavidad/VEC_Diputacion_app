package bootstrap

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	ajusteshttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/ajustesreglas"
	ajustespg "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ajustesapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/ajustesreglas"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

const (
	envCTAjustesReglasEnabled = "VEC_CT_REGLAS_AJUSTES_ENABLED"
	envCTAjustesReglasMotivos = "VEC_CT_REGLAS_AJUSTES_MOTIVOS_PATH"
)

var errMontajeAjustesReglasCT = errors.New("bootstrap: ajustes de reglas CT no disponibles")

func ajustesReglasCTSolicitados(cfg config.Config) (bool, error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envCTAjustesReglasEnabled)
	if err != nil {
		return false, err
	}
	if !activo {
		return false, nil
	}
	if cfg.Normalize().ReglasEjemplo.CTSourcePath == "" ||
		strings.TrimSpace(os.Getenv(envCTAjustesReglasMotivos)) == "" {
		return false, errMontajeAjustesReglasCT
	}
	return true, nil
}

func nuevaRutaAjustesReglasCT(ctx context.Context, cfg config.Config,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	fronteras catalogoFronterasComunDesarrollo,
	fuentePool, motivosPool *pgxpool.Pool,
	material *proveedorMaterialAltaContratacionTemporalDesarrollo,
	fuenteReglas *reglas.Resolutor, reloj relojContratacionTemporalDesarrollo,
) (vechttp.RutaExacta, error) {
	vacia := vechttp.RutaExacta{}
	activo, err := ajustesReglasCTSolicitados(cfg)
	if err != nil || !activo || ctx == nil || ctx.Err() != nil || alta == nil || alta.soporte == nil ||
		alta.postgresql.gobierno == nil || alta.postgresql.ejecucion == nil || alta.postgresql.registroAutorizacion == nil ||
		fuentePool == nil || motivosPool == nil || material == nil ||
		fuenteReglas == nil || fronteras.identidad == nil || fuentePool == motivosPool {
		return vacia, errMontajeAjustesReglasCT
	}
	soporte := alta.soporte
	fijo := soporte.perfilFijoParaRutaYMetodo(ajusteshttp.Ruta, http.MethodPost)
	lector := soporte.perfilFijoParaRutaYMetodo(ajusteshttp.Ruta, http.MethodGet)
	if fijo == nil || lector != fijo || fijo.clave != clavePerfilFijoLectorEntregaCTDesarrollo || fijo.plantilla.Validar() != nil {
		return vacia, errMontajeAjustesReglasCT
	}
	perfil := fijo.perfilRef()
	for _, par := range []struct{ metodo, accion string }{{http.MethodGet, accionConsultarAjustesCT}, {http.MethodPost, accionPublicarAjustesCT}} {
		d, ok := fronteras.resolver(par.metodo, ajusteshttp.Ruta)
		if !ok || d.ClaveCapacidad != par.accion || !d.admitePerfil(perfil) {
			return vacia, errMontajeAjustesReglasCT
		}
	}
	var sqlDisponible bool
	if err := alta.postgresql.ejecucion.QueryRow(ctx, `SELECT
		pg_catalog.to_regprocedure('vec_contratacion_temporal.operar_ajustes_reglas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
		AND pg_catalog.to_regprocedure('vec_contratacion_temporal.leer_activacion_regla_base_v1()') IS NOT NULL
		AND pg_catalog.has_function_privilege(current_user,'vec_contratacion_temporal.operar_ajustes_reglas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')`).Scan(&sqlDisponible); err != nil || !sqlDisponible {
		return vacia, errMontajeAjustesReglasCT
	}
	fuente, err := postgresvec.NuevoAlmacenAutorizacion(fuentePool)
	if err != nil {
		return vacia, errMontajeAjustesReglasCT
	}
	registro, err := postgresvec.NuevoAlmacenAutorizacion(alta.postgresql.registroAutorizacion)
	if err != nil {
		return vacia, errMontajeAjustesReglasCT
	}
	motivo := motivoAutorizacionAjustesCT()
	validador, err := postgresvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivosPool, motivo.CatalogoID)
	if err != nil {
		return vacia, errMontajeAjustesReglasCT
	}
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	if !vigente || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno,
		[]vecdomain.ReferenciaEntradaCatalogo{motivo}, desde) != nil {
		return vacia, errMontajeAjustesReglasCT
	}
	publicada, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, alta.postgresql.gobierno, perfil)
	if err != nil || !encontrada {
		return vacia, errMontajeAjustesReglasCT
	}
	if _, exacta := instantaneaConsumible(publicada, fijo.plantilla, reloj.Ahora()); !exacta {
		return vacia, errMontajeAjustesReglasCT
	}
	pdp, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registro, registro,
		validador, reloj, seguridadvec.GeneradorReferenciasCriptograficas{},
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		return vacia, errMontajeAjustesReglasCT
	}
	material.soporte, material.motivo = soporte, motivo
	proveedor := &proveedorAjustesReglasCT{soporte: soporte, pdp: pdp, material: material,
		motivo: motivo, reloj: reloj, poolPerfiles: alta.postgresql.gobierno, plantilla: fijo.plantilla}
	repo, err := ajustespg.NuevoRepositorioAjustesReglasCT(alta.postgresql.ejecucion, proveedor,
		organizacionAltaContratacionTemporalDesarrollo)
	if err != nil {
		return vacia, err
	}
	base, huella, _, err := fuenteReglas.CatalogoVigente(ctx)
	if err != nil {
		return vacia, errMontajeAjustesReglasCT
	}
	activacion, err := repo.LeerActivacion(ctx)
	calculada, errHuella := base.HuellaSHA256()
	if err != nil || errHuella != nil || calculada != huella || base.Estado != vecdomain.EstadoCatalogoPublicado ||
		activacion.Estado != "activa" ||
		activacion.CatalogoID != base.ID || activacion.Version != base.Version ||
		activacion.HuellaSHA256 != huella || activacion.AprobacionRef == "" ||
		activacion.AprobacionRef != base.AprobacionRef {
		return vacia, errMontajeAjustesReglasCT
	}
	fichero, err := os.Open(strings.TrimSpace(os.Getenv(envCTAjustesReglasMotivos)))
	if err != nil {
		return vacia, errMontajeAjustesReglasCT
	}
	defer fichero.Close()
	contenido, err := io.ReadAll(io.LimitReader(fichero, 16*1024+1))
	if err != nil || len(contenido) > 16*1024 {
		return vacia, errMontajeAjustesReglasCT
	}
	motivos, err := ajustesapp.LeerCatalogoMotivos(contenido)
	if err != nil {
		return vacia, err
	}
	servicio, err := ajustesapp.NuevoServicio(repo, fuenteReglas, motivos, reloj)
	if err != nil {
		return vacia, err
	}
	h, err := ajusteshttp.NuevoManejador(proveedor, servicio)
	if err != nil {
		return vacia, err
	}
	return vechttp.RutaExacta{Ruta: ajusteshttp.Ruta, Manejador: h}, nil
}

func sondaAjustesReglasCT() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), plazoarranque.Ampliar(60*time.Second))
}
