package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	importacionpg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgresimportacionconvoca"
	protector "vec-diputacion-granada/internal/modules/bolsa/adapters/protectorstagingdesarrollo"
	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

var ErrConstitucionBolsaNoDisponible = errors.New("bootstrap: constitucion de bolsa no disponible")

// Actor de la confirmación en el perfil de desarrollo: el subcomando se ejecuta
// por el operador de RRHH dentro del contenedor. En producción el actor vendrá
// de la identidad mTLS de la ruta de confirmación.
const actorConstitucionBolsaDesarrollo = "actor:rrhh:constitucion-bolsa"
const rolConstitucionBolsaDesarrollo = "vec_bolsa_llamamientos_constituidor"

// SolicitudConstitucionBolsa identifica el acta por el fichero importado (su
// huella SHA-256) y la categoría RPT con la que se importó.
type SolicitudConstitucionBolsa struct {
	Fichero   string
	Categoria string
}

// EjecutarConstitucionBolsa compone el recuperador del acta (pool de
// importación + protector de desarrollo) y el repositorio de constitución
// (pool de Bolsa) y confirma el acta como bolsa constituida.
func EjecutarConstitucionBolsa(ctx context.Context, cfg config.Config, s SolicitudConstitucionBolsa) (ports.ReciboConstitucion, error) {
	if ctx == nil || !cfg.DevelopmentEnabledByDoubleKey() {
		return ports.ReciboConstitucion{}, ErrConstitucionBolsaNoDisponible
	}
	categoria, err := validarCategoriaImportacionConvoca(cfg, s.Categoria)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	contenido, err := os.ReadFile(s.Fichero)
	if err != nil || len(contenido) == 0 {
		return ports.ReciboConstitucion{}, ErrConstitucionBolsaNoDisponible
	}
	suma := sha256.Sum256(contenido)
	borrarBytes(contenido)
	huella := hex.EncodeToString(suma[:])
	material, err := cargarMaterialSeguridadDesarrollo(cfg)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	defer borrarMaterialImportacionConvoca(material)
	p, err := protector.Nuevo(material.claveKMS)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	derivador, err := protector.NuevoDerivadorCandidato(material.claveKMS)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	poolImportacion, err := abrirPoolImportacionConvoca(ctx, cfg)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	defer poolImportacion.Close()
	recuperador, err := importacionpg.NuevoRepositorioRecuperacionPostgreSQL(poolImportacion, p)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	poolBolsa, err := abrirPoolConstitucionBolsaDesarrollo(ctx, cfg)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	defer poolBolsa.Close()
	repositorio, err := postgresbolsa.NuevoRepositorioConstitucionPostgreSQL(poolBolsa)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	servicio, err := constitucion.NuevoServicio(recuperador, repositorio, derivador, time.Now)
	if err != nil {
		return ports.ReciboConstitucion{}, err
	}
	return servicio.Constituir(ctx, constitucion.Solicitud{
		HuellaFicheroSHA256: huella, CategoriaRef: categoria, ActorRef: actorConstitucionBolsaDesarrollo,
	})
}

// abrirPoolConstitucionBolsaDesarrollo conserva el privilegio B7/B8 fuera del
// LOGIN usado por las rutas web de llamamientos. No acepta su DSN ni su rol.
func abrirPoolConstitucionBolsaDesarrollo(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	if ctx == nil {
		return nil, ErrConstitucionBolsaNoDisponible
	}
	dsn, err := cfg.DSNBolsaConstitucionSeparado()
	if err != nil {
		return nil, ErrConstitucionBolsaNoDisponible
	}
	pc, err := pgxpool.ParseConfig(dsn)
	if err != nil || pc == nil || pc.ConnConfig == nil ||
		validarTLSPostgreSQLBorradores(&pc.ConnConfig.Config, true) != nil {
		return nil, ErrConstitucionBolsaNoDisponible
	}
	pc.MaxConns = 4
	pc.MinConns = 0
	pc.ConnConfig.ConnectTimeout = 5 * time.Second
	if pc.ConnConfig.RuntimeParams == nil {
		pc.ConnConfig.RuntimeParams = make(map[string]string)
	}
	p := pc.ConnConfig.RuntimeParams
	p["application_name"] = "vec-bolsa-constitucion-desarrollo"
	p["timezone"] = "UTC"
	p["search_path"] = "pg_catalog,pg_temp"
	p["default_transaction_isolation"] = "serializable"
	p["default_transaction_read_only"] = "off"
	p["statement_timeout"] = "15s"
	p["lock_timeout"] = "3s"
	p["idle_in_transaction_session_timeout"] = "20s"
	pc.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return comprobarIdentidadConstitucionBolsaDesarrollo(ctx, conn)
	}
	pc.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		return comprobarIdentidadConstitucionBolsaDesarrollo(ctx, conn) == nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, ErrConstitucionBolsaNoDisponible
	}
	if err := comprobarIdentidadConstitucionBolsaDesarrollo(ctx, pool); err != nil {
		pool.Close()
		return nil, ErrConstitucionBolsaNoDisponible
	}
	return pool, nil
}

func comprobarIdentidadConstitucionBolsaDesarrollo(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) error {
	if ctx == nil || q == nil {
		return ErrConstitucionBolsaNoDisponible
	}
	var valida bool
	err := q.QueryRow(ctx, `
		WITH RECURSIVE membresias(rol_id, admin_option) AS (
			SELECT a.roleid, a.admin_option
			  FROM pg_catalog.pg_auth_members AS a WHERE a.member = session_user::regrole
			UNION
			SELECT a.roleid, m.admin_option OR a.admin_option
			  FROM pg_catalog.pg_auth_members AS a JOIN membresias AS m ON a.member = m.rol_id
		)
		SELECT session_user = current_user
		   AND identidad.rolcanlogin AND identidad.rolinherit
		   AND NOT identidad.rolsuper AND NOT identidad.rolcreatedb
		   AND NOT identidad.rolcreaterole AND NOT identidad.rolreplication
		   AND NOT identidad.rolbypassrls
		   AND NOT grupo.rolcanlogin AND NOT grupo.rolsuper
		   AND NOT grupo.rolcreatedb AND NOT grupo.rolcreaterole
		   AND NOT grupo.rolreplication AND NOT grupo.rolbypassrls
		   AND pg_catalog.pg_has_role(session_user, grupo.oid, 'USAGE')
		   AND NOT EXISTS (
		       SELECT 1 FROM membresias WHERE rol_id <> grupo.oid OR admin_option
		   )
		  FROM pg_catalog.pg_roles AS identidad
		  JOIN pg_catalog.pg_roles AS grupo ON grupo.rolname = $1
		 WHERE identidad.rolname = session_user`, rolConstitucionBolsaDesarrollo).Scan(&valida)
	if err != nil || !valida {
		return ErrConstitucionBolsaNoDisponible
	}
	return nil
}
