package interna

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	ports "vec-diputacion-granada/internal/vec/ports"
)

type componentesOrganizacionHistorica struct {
	proveedor    *internactproveedores.ProveedorAutorizacionOrganizacionHistorica
	consulta     http.Handler
	vincular     func(context.Context) (context.Context, error)
	seleccionada bool
}

// Usa el pool nominal existente de Personal. Una carencia deja únicamente
// Organización fuera; no depende de que el montaje B2 esté activo.
func montarOrganizacionHistoricaGobernada(ctx context.Context, directorio, login string, pool *pgxpool.Pool, base *internactproveedores.Proveedores, fuente *internagobierno.FuenteF1, auditoria ports.RegistradorAuditoriaFronteraRutaExacta, reloj relojGobiernoInterno, intentos ...internactproveedores.DependenciasIntentosOrganizacionHistorica) (componentesOrganizacionHistorica, bool) {
	var vacio componentesOrganizacionHistorica
	vacio.seleccionada = materialOrganizacionHistoricaSeleccionado(directorio)
	if !vacio.seleccionada {
		return vacio, false
	}
	if ctx == nil || ctx.Err() != nil || directorio == "" || pool == nil || base == nil || fuente == nil || interfazNulaIdentidadOffline(auditoria) {
		return vacio, false
	}
	if len(intentos) != 1 {
		return vacio, false
	}
	registro, err := internactproveedores.NuevoRegistroIntentosOrganizacionHistorica(intentos[0])
	if err != nil || registro.PreflightIntentosOrganizacionHistorica(ctx) != nil {
		log.Print("composicion interna: organizacion_historica_no_disponible")
		return vacio, false
	}
	m, err := internactproveedores.CargarMaterialOrganizacionHistorica(directorio)
	if err != nil {
		log.Print("composicion interna: organizacion_historica_no_disponible")
		return vacio, false
	}
	defer m.Cerrar()
	if acreditarPoolOrganizacionHistorica(ctx, pool, login) != nil {
		return vacio, false
	}
	proveedor, err := internactproveedores.ConstruirOrganizacionHistorica(ctx, m, base, fuente, reloj)
	if err != nil {
		log.Print("composicion interna: organizacion_historica_no_disponible")
		return vacio, false
	}
	transferido := false
	defer func() {
		if !transferido {
			proveedor.Cerrar()
		}
	}()
	repositorio, err := personalpg.NuevoRepositorioOrganizacionHistoricaPostgreSQL(pool)
	if err != nil {
		log.Print("composicion interna: organizacion_historica_no_disponible")
		return vacio, false
	}
	servicio, err := personalapp.NuevoServicioConsultaOrganizacionHistorica(proveedor, repositorio, registro)
	if err != nil {
		log.Print("composicion interna: organizacion_historica_no_disponible")
		return vacio, false
	}
	consulta, err := internactproveedores.NuevaConsultaOrganizacionHistoricaConIntentos(servicio, fuente, registro)
	if err != nil {
		log.Print("composicion interna: organizacion_historica_no_disponible")
		return vacio, false
	}
	handler, err := httpapi.NewHandlerOrganizacionHistoricaPersonal(internactproveedores.AutoridadContextoOrganizacionHistorica{Fuente: fuente}, consulta, internactproveedores.AuditorDenegacionOrganizacionHistorica{Registrador: auditoria})
	if err != nil {
		log.Print("composicion interna: organizacion_historica_no_disponible")
		return vacio, false
	}
	ambitos := make(map[string]internagobierno.AmbitoOrganizacionHistorica, len(m.Contextos))
	for cuenta, scope := range m.Contextos {
		ambitos[cuenta] = scope
	}
	vincular := func(ctx context.Context) (context.Context, error) {
		return fuente.VincularContextoOrganizacionHistorica(ctx, ambitos)
	}
	transferido = true
	return componentesOrganizacionHistorica{proveedor: proveedor, consulta: handler, vincular: vincular, seleccionada: true}, true
}

func acreditarPoolOrganizacionHistorica(ctx context.Context, pool *pgxpool.Pool, login string) error {
	if ctx == nil || ctx.Err() != nil || pool == nil || login == "" {
		return ErrPoolsSeguimientoNoDisponibles
	}
	perfil := perfilPoolSeguimiento{rol: "vec_personal_ejecutor", material: MaterialPoolSeguimiento{Login: login}, funcion: "vec_personal.consultar_organizacion_historica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"}
	if acreditarPoolSeguimiento(ctx, pool, perfil) != nil {
		return ErrPoolsSeguimientoNoDisponibles
	}
	ctxSonda, cancelar := context.WithTimeout(ctx, plazoarranque.Ampliar(5*time.Second))
	defer cancelar()
	tablas := []string{"org_nodo_historia", "version_rpt_historia", "version_plantilla_historia", "puesto_tipo_historia", "dotacion_rpt_historia", "plaza_plantilla_historia", "puesto_rpt_historia", "vinculo_plaza_puesto_historia", "recibo_consulta_organizacion"}
	const sql = `SELECT count(*)=9 AND NOT COALESCE(bool_or(pg_catalog.has_table_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')),true) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_personal' AND c.relname=ANY($1::text[]) AND c.relkind IN ('r','p')`
	var sinDML bool
	if err := pool.QueryRow(ctxSonda, sql, tablas).Scan(&sinDML); err != nil || !sinDML {
		return ErrPoolsSeguimientoNoDisponibles
	}
	return nil
}

func materialOrganizacionHistoricaSeleccionado(directorio string) bool {
	if directorio == "" {
		return false
	}
	raiz, err := os.OpenRoot(directorio)
	if err != nil {
		return !errors.Is(err, os.ErrNotExist)
	}
	defer raiz.Close()
	_, err = raiz.Lstat("organizacion_historica_v3.json")
	return !errors.Is(err, os.ErrNotExist)
}
