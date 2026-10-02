package postgres

import (
	"context"
	"crypto/subtle"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type ProvisionadorBootstrap struct {
	pool       conexion
	aprobacion string
	reloj      ports.Reloj
}

var _ ports.ProvisionadorBootstrapAdministracionPerfiles = (*ProvisionadorBootstrap)(nil)

// NuevoProvisionadorBootstrap se usa exclusivamente en el canal de operador.
// La aprobación procede de configuración privada, nunca del cuerpo HTTP.
func NuevoProvisionadorBootstrap(ctx context.Context, pool *pgxpool.Pool, huellaAprobada string, reloj ports.Reloj) (*ProvisionadorBootstrap, error) {
	if pool == nil {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nuevoProvisionadorBootstrap(ctx, pool, huellaAprobada, reloj)
}

func nuevoProvisionadorBootstrap(ctx context.Context, pool conexion, huellaAprobada string, reloj ports.Reloj) (*ProvisionadorBootstrap, error) {
	if ctx == nil || ausente(pool) || ausente(reloj) || !domain.HuellaAdministracionPerfilesValida(huellaAprobada) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var acreditado bool
	if err := pool.QueryRow(ctx, acreditarBootstrapSQL).Scan(&acreditado); err != nil || !acreditado {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return &ProvisionadorBootstrap{pool: pool, aprobacion: huellaAprobada, reloj: reloj}, nil
}

const provisionarBootstrapSQL = `SELECT vec_autorizacion.provisionar_dos_administradores_iniciales_v2($1::text,$2::text)`

func (p *ProvisionadorBootstrap) ProvisionarDosAdministradoresIniciales(ctx context.Context, preimagen ports.PreimagenBootstrapAdministracionPerfiles) (ports.ReciboBootstrapAdministracionPerfiles, error) {
	var cero ports.ReciboBootstrapAdministracionPerfiles
	if ctx == nil || p == nil || ausente(p.pool) || ausente(p.reloj) || preimagen.PlanV2 == nil || preimagen.Validar() != nil {
		return cero, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	plan, huella, err := preimagen.PlanV2.CanonicoYHuella()
	if err != nil || subtle.ConstantTimeCompare([]byte(p.aprobacion), []byte(huella)) != 1 || huella != preimagen.HuellaPlanSHA256 {
		return cero, domain.ErrControlAdministracionPerfilesInvalido
	}
	defer clear(plan)
	// SQL distingue el primer efecto de la recuperación de su recibo. La
	// caducidad cierra nuevas altas; no impide recuperar una alta ya confirmada
	// con la misma aprobación tras revalidar sus fuentes actuales.
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || ausente(tx) {
		return cero, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(c)
	}()
	var b []byte
	if err := tx.QueryRow(ctx, provisionarBootstrapSQL, string(plan), p.aprobacion).Scan(&b); err != nil {
		return cero, traducir(ctx, err)
	}
	var resultado reciboBootstrapJSON
	if decodificar(b, &resultado) != nil {
		return cero, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	r := resultado.Dominio()
	if r.ValidarPara(preimagen) != nil || !instantePersistible(r.ConfirmadoEn) {
		return cero, domain.ErrControlAdministracionPerfilesInvalido
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return cero, traducir(ctx, err)
	}
	return r, nil
}

type reciboBootstrapJSON struct {
	ActoRef           string    `json:"acto_ref"`
	ReciboRef         string    `json:"recibo_ref"`
	HuellaPlanSHA256  string    `json:"huella_plan_sha256"`
	PrimeraPersonaRef string    `json:"primera_persona_ref"`
	SegundaPersonaRef string    `json:"segunda_persona_ref"`
	ConfirmadoEn      time.Time `json:"confirmado_en"`
}

func (r reciboBootstrapJSON) Dominio() domain.ReciboBootstrapAdministracionPerfiles {
	return domain.ReciboBootstrapAdministracionPerfiles{
		ActoRef: r.ActoRef, ReciboRef: r.ReciboRef, HuellaPlanSHA256: r.HuellaPlanSHA256, PrimeraPersonaRef: r.PrimeraPersonaRef, SegundaPersonaRef: r.SegundaPersonaRef, ConfirmadoEn: r.ConfirmadoEn}
}

const acreditarBootstrapSQL = `SELECT current_user=session_user AND r.rolcanlogin AND r.rolinherit
 AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)=1
 AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
   WHERE m.member=r.oid AND g.rolname='vec_admin_perfiles_bootstrap_ejecutor'
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
   AND NOT (g.rolsuper OR g.rolcanlogin OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
   AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members n WHERE n.member=g.oid))
 AND NOT pg_catalog.has_database_privilege(current_user,current_database(),'CREATE')
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_database db WHERE db.datname=current_database()
   AND (db.datdba=r.oid OR db.datdba=pg_catalog.to_regrole('vec_admin_perfiles_bootstrap_ejecutor')))
 AND pg_catalog.to_regprocedure('vec_autorizacion.provisionar_dos_administradores_iniciales_v2(text,text)') IS NOT NULL
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
   AND (n.nspowner=r.oid OR n.nspowner=pg_catalog.to_regrole('vec_admin_perfiles_bootstrap_ejecutor')
     OR pg_catalog.has_schema_privilege(current_user,n.oid,'CREATE')))
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
   AND (c.relowner=r.oid OR c.relowner=pg_catalog.to_regrole('vec_admin_perfiles_bootstrap_ejecutor')
    OR (c.relkind IN ('r','p','v','m','f') AND (pg_catalog.has_table_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
      OR pg_catalog.has_any_column_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))
    OR (c.relkind='S' AND pg_catalog.has_sequence_privilege(current_user,c.oid,'USAGE,SELECT,UPDATE'))))
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_type t JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND (t.typowner=r.oid OR t.typowner=pg_catalog.to_regrole('vec_admin_perfiles_bootstrap_ejecutor')))
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
   AND (p.proowner=r.oid OR p.proowner=pg_catalog.to_regrole('vec_admin_perfiles_bootstrap_ejecutor')
     OR (pg_catalog.has_function_privilege(current_user,p.oid,'EXECUTE')
       AND p.oid<>pg_catalog.to_regprocedure('vec_autorizacion.provisionar_dos_administradores_iniciales_v2(text,text)')
       AND (p.prosecdef OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) acl
         WHERE acl.privilege_type='EXECUTE' AND acl.grantee IN (r.oid,pg_catalog.to_regrole('vec_admin_perfiles_bootstrap_ejecutor')))))))
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`
