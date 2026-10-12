package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// AppendIntentoOrigenCTEnTx es la operación común AD169/AD252 que I expone.
// La transacción pertenece a este lector; el callback no la confirma ni
// serializa de nuevo la orden nominal.
type AppendIntentoOrigenCTEnTx func(context.Context, pgx.Tx, vecports.OrdenIntentoAuditoria, string, string) (vecports.AcuseIntentoAuditoria, error)

type LectorOrigenLlamamientoCTAuditadoPostgreSQL struct {
	pool    *pgxpool.Pool
	append  AppendIntentoOrigenCTEnTx
	proceso string
	canal   string
}

var _ ports.FuenteOrigenLlamamientoCTAuditada = (*LectorOrigenLlamamientoCTAuditadoPostgreSQL)(nil)

func NuevoLectorOrigenLlamamientoCTAuditadoPostgreSQL(pool *pgxpool.Pool,
	appendEnTx AppendIntentoOrigenCTEnTx, proceso, canal string,
) (*LectorOrigenLlamamientoCTAuditadoPostgreSQL, error) {
	if pool == nil || appendEnTx == nil || proceso == "" || canal == "" ||
		strings.TrimSpace(proceso) != proceso || strings.TrimSpace(canal) != canal {
		return nil, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	return &LectorOrigenLlamamientoCTAuditadoPostgreSQL{
		pool: pool, append: appendEnTx, proceso: proceso, canal: canal,
	}, nil
}

// Preflight exige el LOGIN exclusivo del registrador y sólo EXECUTE de las
// funciones lectoras B110. No solicita rol ejecutor de Bolsa.
func (r *LectorOrigenLlamamientoCTAuditadoPostgreSQL) Preflight(ctx context.Context) error {
	if r == nil || r.pool == nil || ctx == nil {
		return ports.ErrOrigenLlamamientoCTNoDisponible
	}
	var valido bool
	err := r.pool.QueryRow(ctx, `SELECT session_user=current_user
 AND vec_autorizacion_atestada_v3.preflight_registrador_intentos_v1($1::text,$2::text)
 AND has_schema_privilege(session_user,'vec_bolsa_llamamientos','USAGE')
 AND has_function_privilege(session_user,'vec_bolsa_llamamientos.leer_origen_llamamiento_ct_v1(text)','EXECUTE')
 AND has_function_privilege(session_user,'vec_bolsa_llamamientos.recuperar_reserva_llamamiento_ct_v1(text,text)','EXECUTE')`, r.proceso, r.canal).Scan(&valido)
	if err != nil || !valido {
		return ports.ErrOrigenLlamamientoCTNoDisponible
	}
	return nil
}

// ConsultarOrigenCTAuditado realiza una lectura nominal por petición. Si ya
// existe reserva CT recupera su snapshot y comando originales; si no existe,
// obtiene la única constitución vigente. En ambos casos confirma el éxito de
// lectura en la cadena común antes de exponer datos o acuse al consumidor.
func (r *LectorOrigenLlamamientoCTAuditadoPostgreSQL) ConsultarOrigenCTAuditado(
	ctx context.Context, expedienteRef, claveCT, categoriaRef string, crearOrden ports.CrearOrdenOrigenCT,
) (ports.OrigenLlamamientoCT, *ports.EmisionLlamamiento, vecports.AcuseIntentoAuditoria, error) {
	vacio, sinAcuse := ports.OrigenLlamamientoCT{}, vecports.AcuseIntentoAuditoria{}
	if r == nil || r.pool == nil || r.append == nil || ctx == nil || crearOrden == nil ||
		expedienteRef == "" || claveCT == "" || categoriaRef == "" ||
		categoriaRef != strings.TrimSpace(categoriaRef) || len(categoriaRef) > 512 {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL TIME ZONE 'UTC'`); err != nil {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	var reserva *ports.EmisionLlamamiento
	var origen ports.OrigenLlamamientoCT
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT reserva FROM (SELECT vec_bolsa_llamamientos.recuperar_reserva_llamamiento_ct_v1($1::text,$2::text) AS reserva) x WHERE reserva IS NOT NULL`, expedienteRef, claveCT).Scan(&raw)
	switch {
	case err == nil:
		emision, e := decodificarReservaLlamamientoCT(raw, expedienteRef, claveCT)
		if e != nil || emision.OrigenCT == nil || emision.OrigenCT.Lectura.CategoriaRef != categoriaRef {
			return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTConflicto
		}
		origen = emision.OrigenCT.Lectura
		reserva = &emision
	case errors.Is(err, pgx.ErrNoRows):
		origen, err = decodificarOrigenLlamamientoCT(ctx,
			tx.QueryRow(ctx, consultaOrigenLlamamientoCT, categoriaRef), categoriaRef)
		if err != nil {
			return vacio, nil, sinAcuse, err
		}
	default:
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	orden, err := crearOrden(origen, reserva)
	if err != nil {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	datosOrden, err := orden.Datos()
	if err != nil || string(datosOrden.Datos.Resultado) != "exito" ||
		datosOrden.Datos.ModuloID != "bolsa" || datosOrden.Datos.RecursoRef != origen.BolsaRef ||
		datosOrden.Datos.Accion != ports.AccionConsultarOrigenCT ||
		datosOrden.Datos.FinalidadRef != ports.FinalidadConsultarOrigenCT ||
		datosOrden.Datos.Proceso != r.proceso || datosOrden.Datos.Canal != r.canal {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	acuse, err := r.append(ctx, tx, orden, r.proceso, r.canal)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	return origen, reserva, acuse, nil
}
