package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const politicaInformeC12SQL = `SELECT vec_cronos_v1.politica_informe_c12_v1($1,$2,$3)`
const prepararInformeC12SQL = `SELECT vec_cronos_v1.preparar_informe_c12_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const confirmarInformeC12SQL = `SELECT vec_cronos_v1.confirmar_informe_c12_v1($1,$2,$3::bigint,$4,$5,$6,$7,$8::numeric,$9::numeric,$10,$11,$12,$13)`

type RepositorioInformeC12 struct {
	db     *pgxpool.Pool
	equipo *RepositorioEquipoInformesC10
}

func NuevoRepositorioInformeC12(pool *pgxpool.Pool) (*RepositorioInformeC12, error) {
	if pool == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioInformeC12{db: pool, equipo: &RepositorioEquipoInformesC10{db: pool}}, nil
}
func (r *RepositorioInformeC12) FotoEquipo(ctx context.Context, ref string) (domain.FotoEquipoAutoritativa, error) {
	if r == nil || r.equipo == nil {
		return domain.FotoEquipoAutoritativa{}, ports.ErrDependenciaNoDisponible
	}
	return r.equipo.FotoEquipo(ctx, ref)
}
func (r *RepositorioInformeC12) PoliticaInforme(ctx context.Context, audiencia string, tipo domain.TipoInformeCronos, formato domain.FormatoInformeCronos) (ports.PoliticaInformeC12, error) {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil {
		return ports.PoliticaInformeC12{}, ports.ErrDependenciaNoDisponible
	}
	if audiencia != string(ports.InformeC12Empleado) && audiencia != string(ports.InformeC12Jefatura) && audiencia != string(ports.InformeC12RRHH) {
		return ports.PoliticaInformeC12{}, ports.ErrDependenciaNoDisponible
	}
	var bruto []byte
	if err := r.db.QueryRow(ctx, politicaInformeC12SQL, audiencia, string(tipo), string(formato)).Scan(&bruto); err != nil {
		return ports.PoliticaInformeC12{}, errorSeguro(ctx, err)
	}
	defer clear(bruto)
	if len(bruto) == 0 || len(bruto) > 4096 {
		return ports.PoliticaInformeC12{}, ports.ErrDependenciaNoDisponible
	}
	var p ports.PoliticaInformeC12
	if decodificarEstricto(bruto, &p) != nil || p.Referencia == "" || p.Version < 1 || len(p.SHA256) != 64 {
		return ports.PoliticaInformeC12{}, ports.ErrDependenciaNoDisponible
	}
	return p, nil
}
func validarOrdenInformeC12(o ports.OrdenInformeC12, m domain.MaterialExportacionCronos) bool {
	actor, err := o.ContextoActor()
	if err != nil || o.Proveedor() == nil || string(o.Alcance()) != m.Alcance || actor.PersonaRef != m.ActorRef || actor.PerfilActivoRef != m.PerfilRef {
		return false
	}
	refs, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	return err == nil && len(refs) == 1 && refs[0] == m.EmpleadoRef
}
func (r *RepositorioInformeC12) ejecutar(ctx context.Context, o ports.OrdenInformeC12, m domain.MaterialExportacionCronos, paso ports.PasoInformeC12, sql string, extra ...any) ([]byte, error) {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil || !validarOrdenInformeC12(o, m) || string(paso) != m.Fase {
		return nil, ports.ErrDependenciaNoDisponible
	}
	canon, err := m.Canonico()
	if err != nil {
		return nil, ports.ErrInformeC12Invalido
	}
	recurso, err := m.Recurso()
	if err != nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	accion, _, audiencia, ok := application.ContratoInformeC12(o.Alcance())
	if !ok {
		return nil, ports.ErrDependenciaNoDisponible
	}
	v3, err := o.Proveedor().ProveerMaterialInformeC12(ctx, m, paso)
	if err != nil {
		return nil, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, audiencia, accion, recurso) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	// El ejecutor compartido conserva la transacción, el COMMIT y el borrado de
	// piezas V3. La confirmación necesita dos parámetros antes de ellas.
	return ejecutarInformeV3(ctx, r.db, sql, canon, v3, extra...)
}
func (r *RepositorioInformeC12) PrepararInforme(ctx context.Context, o ports.OrdenInformeC12, m domain.MaterialExportacionCronos) (ports.PreparacionInformeC12, error) {
	bruto, err := r.ejecutar(ctx, o, m, ports.PasoPrepararInformeC12, prepararInformeC12SQL)
	if err != nil {
		return ports.PreparacionInformeC12{}, err
	}
	defer clear(bruto)
	if len(bruto) == 0 || len(bruto) > 2*1024*1024 {
		return ports.PreparacionInformeC12{}, ports.ErrDependenciaNoDisponible
	}
	var p ports.PreparacionInformeC12
	if decodificarEstricto(bruto, &p) != nil || p.Campos == nil || p.Datos == nil {
		return ports.PreparacionInformeC12{}, ports.ErrDependenciaNoDisponible
	}
	return p, nil
}
func (r *RepositorioInformeC12) ConfirmarInforme(ctx context.Context, o ports.OrdenInformeC12, m domain.MaterialExportacionCronos, sha string, tamano int64) (ports.ReciboInformeC12, error) {
	if len(sha) != 64 || tamano <= 0 || tamano > 2*1024*1024 || m.DocumentoSHA256 != sha || m.Tamano != tamano {
		return ports.ReciboInformeC12{}, ports.ErrInformeC12Invalido
	}
	bruto, err := r.ejecutar(ctx, o, m, ports.PasoConfirmarInformeC12, confirmarInformeC12SQL, sha, tamano)
	if err != nil {
		return ports.ReciboInformeC12{}, err
	}
	defer clear(bruto)
	if len(bruto) == 0 || len(bruto) > 4096 {
		return ports.ReciboInformeC12{}, ports.ErrDependenciaNoDisponible
	}
	var recibo ports.ReciboInformeC12
	if decodificarEstricto(bruto, &recibo) != nil {
		return ports.ReciboInformeC12{}, ports.ErrDependenciaNoDisponible
	}
	return recibo, nil
}

var _ ports.RepositorioInformeC12 = (*RepositorioInformeC12)(nil)

func ejecutarInformeV3(ctx context.Context, db *pgxpool.Pool, sql string, material []byte, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, extra ...any) ([]byte, error) {
	if ctx == nil || db == nil || v3.ValidarEstructura() != nil || len(material) == 0 || len(extra) != 0 && len(extra) != 2 {
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
		c, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		_ = tx.Rollback(c)
	}()
	if _, err := tx.Exec(ctx, "SELECT set_config('search_path','pg_catalog',true),set_config('timezone','UTC',true),set_config('row_security','on',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','10s',true)"); err != nil {
		return nil, errorSeguro(ctx, err)
	}
	args := []any{string(material)}
	args = append(args, extra...)
	args = append(args, secretos[0], secretos[1], secretos[2], secretos[3], v3.PersonaVersion(), v3.PerfilVersion(), secretos[4], secretos[5], secretos[6], secretos[7])
	var bruto []byte
	if err := tx.QueryRow(ctx, sql, args...).Scan(&bruto); err != nil {
		return nil, errorSeguro(ctx, err)
	}
	if tx.Commit(ctx) != nil {
		clear(bruto)
		return nil, ports.ErrDependenciaNoDisponible
	}
	return bruto, nil
}
