package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	"vec-diputacion-granada/internal/shared/postgresql"
	vec "vec-diputacion-granada/internal/vec/domain"
)

const consultarRevisionRRHHSQL = `SELECT vec_meritos.consultar_revision_rrhh_v1($1::bytea,$2::bytea,$3::jsonb,$4::jsonb)`

type RevisionRRHH struct{ pool iniciador }

var _ ports.LectorRevisionRRHH = (*RevisionRRHH)(nil)

func NuevoRevisionRRHH(pool *pgxpool.Pool) (*RevisionRRHH, error) {
	if nulo(pool) {
		return nil, ports.ErrRevisionRRHHNoDisponible
	}
	return &RevisionRRHH{pool: pool}, nil
}

func (r *RevisionRRHH) ConsultarRevisionRRHH(ctx context.Context,
	s ports.SolicitudRevisionRRHH) (ports.PaginaRevisionRRHH, error) {
	if ctx == nil || r == nil || nulo(r.pool) || s.Validar() != nil {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHDenegada
	}
	datos, err := s.Vinculo.Datos()
	if err != nil {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHDenegada
	}
	correlacion, err := s.Correlacion.ValorCanonico()
	if err != nil {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHDenegada
	}
	contexto := append([]byte(nil), s.Contexto.RepresentacionCanonica...)
	capturaBolsa := append([]byte(nil), s.CapturaDetalleBolsa...)
	vinculo, err := json.Marshal(s.Vinculo)
	if err != nil {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
	}
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
	}
	huella := sha256.Sum256(contexto)
	orden := map[string]any{
		"intento_ref":                "lectura_" + hex.EncodeToString(aleatorio[:]),
		"registro_contexto_ref":      datos.RegistroContextoRef,
		"contexto_sha256":            hex.EncodeToString(huella[:]),
		"procedencia_sha256":         datos.ManifiestoProcedenciaHuellaSHA256,
		"autenticacion_ref":          datos.AutenticacionRef,
		"sesion_ref":                 datos.SesionRef,
		"autenticacion_sha256":       datos.AutenticacionHuellaSHA256,
		"accion":                     ports.AccionRevisionRRHH,
		"modulo_id":                  "meritos",
		"recurso_ref":                s.SolicitudRef,
		"finalidad_ref":              ports.FinalidadRevisionRRHH,
		"proceso":                    "vec-server",
		"canal":                      "interna_corporativa",
		"correlacion_ref":            correlacion,
		"lectura_revision_permisos":  strconv.FormatUint(s.RevisionPermisos, 10),
		"lectura_instantanea_sha256": s.HuellaInstantaneaSHA256,
		"persona_ref":                s.PersonaRef,
		"limite":                     s.Limite,
		"cursor":                     s.Cursor,
	}
	b, err := json.Marshal(orden)
	if err != nil || len(b) > 16384 {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
	}
	var resultado ports.PaginaRevisionRRHH
	err = postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		var e error
		resultado, e = r.consultar(ctx, s, contexto, vinculo, b, capturaBolsa, correlacion)
		return e
	})
	if err != nil {
		return ports.PaginaRevisionRRHH{}, err
	}
	return resultado, nil
}

type respuestaRevisionRRHHSQL struct {
	Codigo          string                   `json:"codigo"`
	Esquema         string                   `json:"esquema"`
	Hechos          []domain.Hecho           `json:"hechos"`
	Total           int                      `json:"total"`
	CursorSiguiente string                   `json:"cursor_siguiente"`
	ReciboAcceso    ports.ReciboRevisionRRHH `json:"recibo_acceso"`
}

func (r *RevisionRRHH) consultar(ctx context.Context, s ports.SolicitudRevisionRRHH,
	contexto, vinculo, orden, capturaBolsa []byte, correlacion string) (ports.PaginaRevisionRRHH, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
	}
	if nulo(tx) {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
	}
	defer revertir(tx)
	if _, err := tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
set_config('row_security','on',true),set_config('timezone','UTC',true),
set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
	}
	var bruto []byte
	if err := tx.QueryRow(ctx, consultarRevisionRRHHSQL, contexto, vinculo, orden, capturaBolsa).Scan(&bruto); err != nil {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
	}
	var salida respuestaRevisionRRHHSQL
	if json.Unmarshal(bruto, &salida) != nil {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
	}
	if salida.Codigo == "denegada" {
		if err := tx.Commit(ctx); err != nil {
			return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
		}
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHDenegada
	}
	if salida.Codigo != "obtenida" || salida.Esquema != ports.EsquemaRevisionRRHH ||
		salida.Hechos == nil || salida.Total < len(salida.Hechos) ||
		len(salida.Hechos) > s.Limite ||
		salida.ReciboAcceso.AuditoriaRef == "" ||
		salida.ReciboAcceso.CorrelacionRef != correlacion ||
		!huellaSHA256CarteraSQL(salida.ReciboAcceso.HuellaSHA256) ||
		salida.ReciboAcceso.RegistradaEn.IsZero() {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
	}
	if _, err := strconv.ParseUint(salida.ReciboAcceso.Secuencia, 10, 64); err != nil {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
	}
	pagina := ports.PaginaRevisionRRHH{Esquema: salida.Esquema,
		Hechos: make([]ports.HechoRevisionRRHH, 0, len(salida.Hechos)),
		Total:  salida.Total, CursorSiguiente: salida.CursorSiguiente,
		ReciboAcceso: salida.ReciboAcceso}
	ultimo := s.Cursor
	for _, hecho := range salida.Hechos {
		if hecho.Validar() != nil || hecho.PersonaRef != s.PersonaRef ||
			hecho.Referencia <= ultimo {
			return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
		}
		ultimo = hecho.Referencia
		evidencias := append([]vec.ReferenciaDocumento(nil), hecho.Evidencias...)
		if evidencias == nil {
			evidencias = []vec.ReferenciaDocumento{}
		}
		pagina.Hechos = append(pagina.Hechos, ports.HechoRevisionRRHH{
			HechoRef: hecho.Referencia, Version: hecho.Version, Tipo: hecho.Tipo,
			Denominacion: hecho.Denominacion, Estado: hecho.Estado,
			Vigencia: hecho.Vigencia, Horas: hecho.Horas,
			Evidencias: evidencias, Revision: hecho.Revision,
		})
	}
	if pagina.CursorSiguiente != "" && (len(pagina.Hechos) != s.Limite ||
		pagina.CursorSiguiente != ultimo) {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.PaginaRevisionRRHH{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.PaginaRevisionRRHH{}, ports.ErrRevisionRRHHNoDisponible
	}
	return pagina, nil
}
