package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

const consultaCompetenciasAsignacionDietasSQL = `SELECT recibo_ref,decision_ref,efecto_ref,consumo_huella_sha256,auditoria_ad3_ref,consultada_en,cardinalidad,competencias FROM vec_personal.consultar_competencias_asignacion_dietas_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

var reciboCompetenciasAsignacionPostgres = regexp.MustCompile(`^rca_[0-9a-f]{32}$`)

type RepositorioCompetenciasAsignacionDietasPostgreSQL struct{ pool iniciadorConsultaRelacion }

var _ personalports.RepositorioCompetenciasAsignacionDietas = (*RepositorioCompetenciasAsignacionDietasPostgreSQL)(nil)

func NuevoRepositorioCompetenciasAsignacionDietasPostgreSQL(pool *pgxpool.Pool) (*RepositorioCompetenciasAsignacionDietasPostgreSQL, error) {
	return nuevoRepositorioCompetenciasAsignacionDietasPostgreSQL(pool)
}

func nuevoRepositorioCompetenciasAsignacionDietasPostgreSQL(pool iniciadorConsultaRelacion) (*RepositorioCompetenciasAsignacionDietasPostgreSQL, error) {
	if nuloRelacion(pool) {
		return nil, personalports.ErrCompetenciasAsignacionDietasNoDisponibles
	}
	return &RepositorioCompetenciasAsignacionDietasPostgreSQL{pool: pool}, nil
}

func (r *RepositorioCompetenciasAsignacionDietasPostgreSQL) ConsultarCompetenciasAsignacionDietas(ctx context.Context, orden personalports.OrdenConsultaCompetenciasAsignacionDietas) (personalports.ResultadoConsultaCompetenciasAsignacionDietas, error) {
	var vacio personalports.ResultadoConsultaCompetenciasAsignacionDietas
	if r == nil || ctx == nil || nuloRelacion(r.pool) {
		return vacio, personalports.ErrCompetenciasAsignacionDietasNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material := orden.Material.Canonico()
	if len(material) == 0 || orden.Autorizacion.ValidarEstructura() != nil || orden.Autorizacion.PersonaVersion() > math.MaxInt64 || orden.Autorizacion.PerfilVersion() > math.MaxInt64 {
		return vacio, personalports.ErrConsultaCompetenciasAsignacionDietasInvalida
	}
	parametros, secretos, err := parametrosConsultaRelacionPropia(material, orden.Autorizacion)
	if err != nil {
		return vacio, personalports.ErrConsultaCompetenciasAsignacionDietasInvalida
	}
	defer borrarRelacion(secretos[:])
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, normalizarErrorCompetenciasAsignacion(ctx, err)
	}
	if tx == nil {
		return vacio, personalports.ErrCompetenciasAsignacionDietasNoDisponibles
	}
	confirmada := false
	defer func() {
		if !confirmada {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err = tx.Exec(ctx, ajustesAsignacionDietas); err != nil {
		return vacio, normalizarErrorCompetenciasAsignacion(ctx, err)
	}
	var e personalports.EvidenciaConsultaCompetenciasAsignacionDietas
	var n int
	var bruto []byte
	err = tx.QueryRow(ctx, consultaCompetenciasAsignacionDietasSQL, parametros...).Scan(
		&e.ReciboRef, &e.DecisionRef, &e.EfectoRef, &e.ConsumoHuellaSHA256, &e.AuditoriaRef, &e.ConsultadaEn, &n, &bruto,
	)
	if err != nil {
		return vacio, normalizarErrorCompetenciasAsignacion(ctx, err)
	}
	resumen := orden.Autorizacion.ResumenCapacidad()
	rpta, err := decodificarCompetenciasAsignacion(bruto, n, orden.Material.Solicitud(), resumen.DecisionRef(), resumen.EfectoRef(), resumen.EmitidaEn(), resumen.ExpiraEn(), e)
	if err != nil {
		return vacio, personalports.ErrCompetenciasAsignacionDietasNoDisponibles
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, normalizarErrorCompetenciasAsignacion(ctx, err)
	}
	confirmada = true
	return rpta, nil
}

type competenciaAsignacionWire struct {
	AsignacionRef string `json:"asignacion_ref"`
	RelacionRef   string `json:"relacion_ref"`
	UnidadRef     string `json:"unidad_ref"`
	Rol           string `json:"rol"`
	VigenteDesde  string `json:"vigente_desde"`
	Version       int64  `json:"version"`
}

func decodificarCompetenciasAsignacion(b []byte, n int, solicitud personaldomain.SolicitudCompetenciasAsignacionDietas, decision, efecto string, emitida, expira time.Time, e personalports.EvidenciaConsultaCompetenciasAsignacionDietas) (personalports.ResultadoConsultaCompetenciasAsignacionDietas, error) {
	var vacio personalports.ResultadoConsultaCompetenciasAsignacionDietas
	if n < 0 || n > 100 || len(b) == 0 || len(b) > 128<<10 || bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		return vacio, errors.New("wire")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	var objetos []json.RawMessage
	if d.Decode(&objetos) != nil || d.Decode(new(any)) != io.EOF || len(objetos) != n || objetos == nil {
		return vacio, errors.New("wire")
	}
	competencias := make([]personaldomain.CompetenciaAsignacionDietas, 0, n)
	vistas := make(map[string]bool, n)
	for _, bruto := range objetos {
		var campos map[string]json.RawMessage
		if json.Unmarshal(bruto, &campos) != nil || len(campos) != 6 {
			return vacio, errors.New("wire")
		}
		for _, clave := range []string{"asignacion_ref", "relacion_ref", "unidad_ref", "rol", "vigente_desde", "version"} {
			if _, ok := campos[clave]; !ok {
				return vacio, errors.New("wire")
			}
		}
		dec := json.NewDecoder(bytes.NewReader(bruto))
		dec.DisallowUnknownFields()
		var w competenciaAsignacionWire
		if dec.Decode(&w) != nil || dec.Decode(new(any)) != io.EOF {
			return vacio, errors.New("wire")
		}
		fecha, err := personaldomain.NuevaFechaCivil(w.VigenteDesde)
		if err != nil {
			return vacio, err
		}
		c := personaldomain.CompetenciaAsignacionDietas{AsignacionRef: w.AsignacionRef, RelacionRef: w.RelacionRef, UnidadRef: w.UnidadRef, Rol: w.Rol, VigenteDesde: fecha, Version: w.Version}
		if c.Validar(solicitud.FechaReferencia) != nil || vistas[c.RelacionRef] {
			return vacio, errors.New("wire")
		}
		vistas[c.RelacionRef] = true
		competencias = append(competencias, c)
	}
	_, offset := e.ConsultadaEn.Zone()
	if !reciboCompetenciasAsignacionPostgres.MatchString(e.ReciboRef) || !huellaAsignacionPostgres.MatchString(e.ConsumoHuellaSHA256) ||
		e.DecisionRef != decision || e.EfectoRef != efecto || e.AuditoriaRef == "" || e.ConsultadaEn.IsZero() || offset != 0 ||
		e.ConsultadaEn.Nanosecond()%1000 != 0 || e.ConsultadaEn.Before(emitida) || !e.ConsultadaEn.Before(expira) {
		return vacio, errors.New("wire")
	}
	e.ConsultadaEn = e.ConsultadaEn.UTC()
	return personalports.ResultadoConsultaCompetenciasAsignacionDietas{Competencias: competencias, Evidencia: e}, nil
}

func normalizarErrorCompetenciasAsignacion(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && (pg.Code == "42501" || pg.Code == "P7201") {
		return personalports.ErrCompetenciasAsignacionDietasDenegadas
	}
	return personalports.ErrCompetenciasAsignacionDietasNoDisponibles
}
