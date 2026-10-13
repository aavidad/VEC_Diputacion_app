package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El LOGIN técnico sólo puede invocar AD261. Las primitivas de Bolsa quedan
// reservadas al propietario NOLOGIN del wrapper.
type LectorOrigenLlamamientoCTAuditadoPostgreSQL struct {
	pool           *pgxpool.Pool
	proceso, canal string
}

var _ ports.FuenteOrigenLlamamientoCTAuditada = (*LectorOrigenLlamamientoCTAuditadoPostgreSQL)(nil)

func NuevoLectorOrigenLlamamientoCTAuditadoPostgreSQL(pool *pgxpool.Pool,
	proceso, canal string,
) (*LectorOrigenLlamamientoCTAuditadoPostgreSQL, error) {
	if pool == nil || proceso != "vec-rrhh" || canal != "interna_corporativa" {
		return nil, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	return &LectorOrigenLlamamientoCTAuditadoPostgreSQL{pool: pool, proceso: proceso, canal: canal}, nil
}

func (r *LectorOrigenLlamamientoCTAuditadoPostgreSQL) Preflight(ctx context.Context) error {
	if r == nil || r.pool == nil || ctx == nil {
		return ports.ErrOrigenLlamamientoCTNoDisponible
	}
	var valido bool
	err := r.pool.QueryRow(ctx, `SELECT session_user=current_user
      AND vec_autorizacion_atestada_v3.preflight_registrador_intentos_v1($1::text,$2::text)
      AND has_function_privilege(session_user,'vec_autorizacion_atestada_v3.resolver_recurso_origen_bolsa_ct_v1(bytea,bytea,text,text,text,text)','EXECUTE')
      AND has_function_privilege(session_user,'vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1(bytea,bytea,jsonb,jsonb,text,text,text)','EXECUTE')
      AND NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc p
        JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
        WHERE n.nspname='vec_bolsa_llamamientos'
          AND p.proname IN ('leer_origen_llamamiento_ct_v1','recuperar_reserva_llamamiento_ct_v1')
          AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE'))`,
		r.proceso, r.canal).Scan(&valido)
	if err != nil || !valido {
		return ports.ErrOrigenLlamamientoCTNoDisponible
	}
	return nil
}

// COMMIT precede a la entrega de canónicos, reserva y acuse. Cualquier fallo
// descarta el asiento y el dato; un cliente HTTP nunca controla esta transacción.
func (r *LectorOrigenLlamamientoCTAuditadoPostgreSQL) ConsultarOrigenCTAuditado(
	ctx context.Context, expedienteRef, claveCT, categoriaRef, unidadRef string,
	contexto, vinculo []byte, crearCaptura ports.CrearCapturaOrigenCT,
) (ports.OrigenLlamamientoCT, *ports.ReservaOrigenCT, vecports.AcuseIntentoAuditoria, error) {
	vacio, sinAcuse := ports.OrigenLlamamientoCT{}, vecports.AcuseIntentoAuditoria{}
	if r == nil || r.pool == nil || ctx == nil || crearCaptura == nil ||
		expedienteRef == "" || claveCT == "" || categoriaRef == "" || unidadRef == "" ||
		categoriaRef != strings.TrimSpace(categoriaRef) || len(categoriaRef) > 512 ||
		len(contexto) == 0 || len(vinculo) == 0 {
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
	var bolsaRef string
	err = tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.resolver_recurso_origen_bolsa_ct_v1($1,$2,$3,$4,$5,$6)`,
		contexto, vinculo, expedienteRef, claveCT, categoriaRef, unidadRef).Scan(&bolsaRef)
	if err != nil || bolsaRef == "" {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	creada, err := crearCaptura(bolsaRef)
	captura := creada.Permiso
	datosOrden, errOrden := creada.Auditoria.Datos()
	if err != nil || errOrden != nil || captura.BolsaRef != bolsaRef || captura.RecursoRef != bolsaRef ||
		len(captura.ContextoCanonico) == 0 || len(captura.VinculoCanonico) == 0 || len(captura.Orden) == 0 ||
		!captura.ValidaHasta.After(time.Now().UTC()) ||
		datosOrden.Datos.Accion != ports.AccionConsultarOrigenCT ||
		datosOrden.Datos.RecursoRef != bolsaRef ||
		datosOrden.Datos.Proceso != r.proceso || datosOrden.Datos.Canal != r.canal {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	_, _, ordenAuditoria, err := postgresvec.SerializarOrdenIntentoAuditoria(creada.Auditoria)
	if err != nil {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.leer_origen_bolsa_ct_auditado_v1($1,$2,$3::jsonb,$4::jsonb,$5,$6,$7)`,
		captura.ContextoCanonico, captura.VinculoCanonico, []byte(captura.Orden), ordenAuditoria,
		expedienteRef, claveCT, categoriaRef).Scan(&raw)
	if err != nil {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	var salida struct {
		Origen  ports.OrigenLlamamientoCT `json:"origen"`
		Reserva *ports.ReservaOrigenCT    `json:"reserva"`
		Acuse   struct {
			AuditoriaRef   string    `json:"auditoria_ref"`
			Secuencia      int64     `json:"secuencia"`
			HuellaSHA256   string    `json:"huella_sha256"`
			CorrelacionRef string    `json:"correlacion_ref"`
			RegistradaEn   time.Time `json:"registrada_en"`
		} `json:"acuse"`
	}
	if json.Unmarshal(raw, &salida) != nil || salida.Origen.Corte.IsZero() {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	salida.Origen.Corte = salida.Origen.Corte.UTC()
	if salida.Origen.BolsaRef != bolsaRef ||
		salida.Origen.CategoriaRef != categoriaRef || validarLecturaOrigenCT(salida.Origen) != nil ||
		salida.Acuse.AuditoriaRef == "" || salida.Acuse.Secuencia < 1 ||
		salida.Acuse.HuellaSHA256 == "" || salida.Acuse.CorrelacionRef == "" {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	if salida.Reserva != nil && (salida.Reserva.Configuracion.Referencia == "" ||
		salida.Reserva.Configuracion.Descripcion == "" ||
		salida.Reserva.Configuracion.Categoria == "" ||
		salida.Reserva.Configuracion.Centro == "" ||
		salida.Reserva.Configuracion.Modalidad == "" ||
		salida.Reserva.Configuracion.FechaInicio == "" ||
		salida.Reserva.Configuracion.Plazo == "" ||
		salida.Reserva.Configuracion.PlantillaVersion == "" ||
		salida.Reserva.Configuracion.Asunto == "" ||
		salida.Reserva.Configuracion.Cuerpo == "" ||
		salida.Reserva.ExpedienteRef != expedienteRef ||
		salida.Reserva.ClaveCT != claveCT ||
		salida.Reserva.UnidadRef != unidadRef ||
		salida.Reserva.ParticipacionRef == "" ||
		salida.Reserva.CandidatoRef == "" ||
		salida.Reserva.OrdenSeleccionado == 0) {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTConflicto
	}
	if salida.Reserva != nil {
		coincide := false
		for _, entrada := range salida.Origen.Entradas {
			if entrada.ParticipacionRef == salida.Reserva.ParticipacionRef &&
				entrada.CandidatoRef == salida.Reserva.CandidatoRef &&
				entrada.OrdenVigente != nil &&
				*entrada.OrdenVigente == salida.Reserva.OrdenSeleccionado {
				coincide = true
				break
			}
		}
		if !coincide {
			return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTConflicto
		}
	}
	acuse := vecports.AcuseIntentoAuditoria{
		AuditoriaRef: salida.Acuse.AuditoriaRef, Secuencia: salida.Acuse.Secuencia,
		HuellaSHA256: salida.Acuse.HuellaSHA256, CorrelacionRef: salida.Acuse.CorrelacionRef,
		RegistradaEn: salida.Acuse.RegistradaEn.UTC(),
	}
	if acuse.ValidarPara(creada.Auditoria) != nil {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, nil, sinAcuse, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	return salida.Origen, salida.Reserva, acuse, nil
}
