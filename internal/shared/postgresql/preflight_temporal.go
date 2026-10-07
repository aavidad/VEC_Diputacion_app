package postgresql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrTEMPArranque impide entregar un pool cuando la base permite tablas
// temporales a PUBLIC o a un LOGIN técnico de la aplicación.
var ErrTEMPArranque = errors.New("postgresql: temp_preflight_rejected")

const DuracionMaximaPreflightTEMP = 2 * time.Second

// NuevoPoolConPreflightTEMP abre una sola conexión en el pool que recibirá el
// consumidor. Si la lectura o la política falla, cierra el pool antes de
// devolver el error. Las comprobaciones AfterConnect ya configuradas se
// conservan y suceden antes de la lectura de catálogos.
func NuevoPoolConPreflightTEMP(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := ComprobarTEMPArranque(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// consultaTEMPArranque usa el ACL efectivo de la base actual. acldefault cubre
// las bases sin datacl explícita; has_database_privilege incluye PUBLIC, las
// membresías heredadas y la propiedad de la base. El prefijo vec_ identifica
// los LOGIN técnicos del producto, incluidos los que posean la base o tengan
// superusuario por error. session_user cubre otra credencial configurada.
// postgres no entra en el barrido salvo que sea el LOGIN del pool.
const consultaTEMPArranque = `
SELECT EXISTS (
         SELECT 1
           FROM pg_catalog.aclexplode(coalesce(base.datacl,
               pg_catalog.acldefault('d', base.datdba))) AS acl
          WHERE acl.grantee = 0 AND acl.privilege_type = 'TEMPORARY'
       ),
       EXISTS (
         SELECT 1
           FROM pg_catalog.pg_roles AS rol
          WHERE rol.rolcanlogin
            AND (pg_catalog.left(rol.rolname::text, 4) = 'vec_'
                 OR rol.rolname = session_user)
            AND pg_catalog.has_database_privilege(rol.oid, base.oid, 'TEMPORARY')
       )
  FROM pg_catalog.pg_database AS base
 WHERE base.datname = pg_catalog.current_database()`

// ComprobarTEMPArranque se invoca una vez durante la construcción de cada pool,
// después de conectar y antes de entregar el pool a un consumidor. Sólo lee
// catálogos; nunca se llama por petición ni cambia privilegios.
func ComprobarTEMPArranque(ctx context.Context, consulta interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) error {
	if ctx == nil || consulta == nil {
		return falloTEMPArranque("metadatos_base", "presente", "ausente")
	}
	if ctx.Err() != nil {
		return falloTEMPArranque("lectura_acl", "disponible", "fallo")
	}
	sonda, cancelar := context.WithTimeout(ctx, DuracionMaximaPreflightTEMP)
	defer cancelar()
	var publico, login bool
	if err := consulta.QueryRow(sonda, consultaTEMPArranque).Scan(&publico, &login); err != nil {
		return falloTEMPArranque("lectura_acl", "disponible", "fallo")
	}
	if publico {
		return falloTEMPArranque("public_temp", "false", "true")
	}
	if login {
		return falloTEMPArranque("login_app_temp", "false", "true")
	}
	return nil
}

func falloTEMPArranque(clave, esperado, observado string) error {
	slog.Error("postgresql_temp_preflight_rejected", "clave", clave,
		"esperado", esperado, "observado", observado)
	return fmt.Errorf("%w: clave=%s esperado=%s observado=%s",
		ErrTEMPArranque, clave, esperado, observado)
}
