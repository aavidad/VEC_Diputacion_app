package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Las cuatro lecturas y escrituras de la persona empleada llaman a una sola
// función de cronos_v1 que consume la decisión V3 en su transacción. Estas
// constantes son las únicas consultas admitidas por el ejecutor.
const (
	consultaSaldoPropio        = `SELECT vec_cronos_v1.consultar_saldo_propio_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaEstadoRemotoPropio = `SELECT vec_cronos_v1.consultar_estado_remoto_propio_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaRecuperarRemoto    = `SELECT vec_cronos_v1.recuperar_marcaje_remoto_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
)

// resumenV3Ligado comprueba que el material exportado autoriza exactamente la
// audiencia, operación, recurso y huella esperados antes de tocar la base.
func resumenV3Ligado(v vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, audiencia, operacion string, recurso vecdomain.RecursoAutorizable) bool {
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	r := v.ResumenCapacidad()
	return err == nil && v.ValidarEstructura() == nil && r.AudienciaConsumo() == audiencia &&
		r.Operacion() == operacion && r.EfectoRef() == recurso.Referencia && r.EfectoHuellaSHA256() == huella
}

// errorProveedorV3 conserva únicamente la denegación nominal del PDP. Cualquier
// otro fallo del emisor es dependencia no disponible, nunca autorización.
func errorProveedorV3(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, vecdomain.ErrPermissionDenied) {
		return vecdomain.ErrPermissionDenied
	}
	return ports.ErrDependenciaNoDisponible
}

// ejecutarLecturaV3 abre una transacción SERIALIZABLE propia, invoca la función
// durable con el material y las diez piezas V3 y confirma. Los bytes secretos
// se borran al terminar. Sin COMMIT confirmado no se devuelve nada.
func ejecutarLecturaV3(ctx context.Context, db iniciadorMarcaje, consulta string, material []byte, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]byte, error) {
	if db == nil || ctx == nil || len(material) == 0 || v3.ValidarEstructura() != nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	secretos := [][]byte{v3.CapacidadCanonica(), v3.DecisionCanonica(), v3.MotivoCanonico(), v3.ContextoActorCanonico(), v3.PayloadVECAD3(), v3.SobreCOSESign1(), v3.EvidenciaVerificacion(), v3.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			clear(b)
		}
	}()
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, errorSeguro(ctx, err)
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	if _, err := tx.Exec(ctx, "SELECT set_config('search_path','pg_catalog',true),set_config('timezone','UTC',true),set_config('row_security','on',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','10s',true)"); err != nil {
		return nil, errorSeguro(ctx, err)
	}
	var bruto []byte
	if err := tx.QueryRow(ctx, consulta, string(material), secretos[0], secretos[1], secretos[2], secretos[3], v3.PersonaVersion(), v3.PerfilVersion(), secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&bruto); err != nil {
		return nil, errorSeguro(ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		clear(bruto)
		return nil, ports.ErrDependenciaNoDisponible
	}
	return bruto, nil
}
