package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"reflect"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const consultaRelacionesPropiasDietas = `SELECT recibo_ref,decision_ref,efecto_ref,consumo_huella_sha256,auditoria_ad3_ref,consultada_en,cardinalidad,relaciones FROM vec_personal.consultar_relaciones_propias_dietas_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const ajustesConsultaRelacionesPropiasDietas = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`

type iniciadorConsultaRelacion interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}
type RepositorioRelacionEmpleadoPostgreSQL struct{ pool iniciadorConsultaRelacion }

var _ personalports.RepositorioRelacionesEmpleado = (*RepositorioRelacionEmpleadoPostgreSQL)(nil)

func NuevoRepositorioRelacionEmpleadoPostgreSQL(pool *pgxpool.Pool) (*RepositorioRelacionEmpleadoPostgreSQL, error) {
	return nuevoRepositorioRelacionEmpleadoPostgreSQL(pool)
}
func nuevoRepositorioRelacionEmpleadoPostgreSQL(pool iniciadorConsultaRelacion) (*RepositorioRelacionEmpleadoPostgreSQL, error) {
	if nuloRelacion(pool) {
		return nil, personalports.ErrRelacionEmpleadoNoDisponible
	}
	return &RepositorioRelacionEmpleadoPostgreSQL{pool: pool}, nil
}

func (r *RepositorioRelacionEmpleadoPostgreSQL) ConsultarRelacionesPropiasDietas(ctx context.Context, o personalports.OrdenConsultaRelacionPropia) (personalports.ResultadoConsultaRelacionPropia, error) {
	var cero personalports.ResultadoConsultaRelacionPropia
	if r == nil || ctx == nil || nuloRelacion(r.pool) {
		return cero, personalports.ErrRelacionEmpleadoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	c := o.Material.Solicitud()
	material := o.Material.Canonico()
	if len(material) == 0 || o.Autorizacion.ValidarEstructura() != nil {
		return cero, personalports.ErrConsultaRelacionEmpleadoInvalida
	}
	parametros, piezas, err := parametrosConsultaRelacionPropia(material, o.Autorizacion)
	if err != nil {
		return cero, personalports.ErrConsultaRelacionEmpleadoInvalida
	}
	defer borrarRelacion(piezas[:])
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, normalizarErrorRelacionEmpleado(ctx, err)
	}
	if tx == nil {
		return cero, personalports.ErrRelacionEmpleadoNoDisponible
	}
	confirmada := false
	defer func() {
		if !confirmada {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err = tx.Exec(ctx, ajustesConsultaRelacionesPropiasDietas); err != nil {
		return cero, normalizarErrorRelacionEmpleado(ctx, err)
	}
	var recibo, decision, efecto, huella, auditoria string
	var consultadaEn time.Time
	var n int
	var relacionesJSON []byte
	if err = tx.QueryRow(ctx, consultaRelacionesPropiasDietas, parametros...).Scan(&recibo, &decision, &efecto, &huella, &auditoria, &consultadaEn, &n, &relacionesJSON); err != nil {
		return cero, normalizarErrorRelacionEmpleado(ctx, err)
	}
	resultado, err := decodificarRelacionesPropias(relacionesJSON, n, c, o.Autorizacion, recibo, decision, efecto, huella, auditoria, consultadaEn)
	if err != nil {
		return cero, personalports.ErrRelacionEmpleadoNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, personalports.ErrRelacionEmpleadoNoDisponible
	}
	confirmada = true
	return resultado, nil
}

func parametrosConsultaRelacionPropia(material []byte, x vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]any, [8][]byte, error) {
	var p [8][]byte
	if x.ValidarEstructura() != nil || x.PersonaVersion() > math.MaxInt64 || x.PerfilVersion() > math.MaxInt64 {
		return nil, p, personalports.ErrConsultaRelacionEmpleadoInvalida
	}
	p = [8][]byte{x.CapacidadCanonica(), x.DecisionCanonica(), x.MotivoCanonico(), x.ContextoActorCanonico(), x.PayloadVECAD3(), x.SobreCOSESign1(), x.EvidenciaVerificacion(), x.RaizPublicaSPKI()}
	return []any{string(material), p[0], p[1], p[2], p[3], int64(x.PersonaVersion()), int64(x.PerfilVersion()), p[4], p[5], p[6], p[7]}, p, nil
}

type relacionWire struct {
	Desde              string  `json:"desde"`
	EmpleadoRef        string  `json:"empleado_ref"`
	Estado             string  `json:"estado"`
	FuenteRef          string  `json:"fuente_ref"`
	FuenteVersion      int64   `json:"fuente_version"`
	Hasta              *string `json:"hasta"`
	PersonaRef         string  `json:"persona_ref"`
	ProcedenciaActoRef string  `json:"procedencia_acto_ref"`
	RelacionRef        string  `json:"relacion_ref"`
	UnidadRef          string  `json:"unidad_ref"`
	Version            int64   `json:"version"`
}

func decodificarRelacionesPropias(b []byte, n int, c personaldomain.SolicitudConsultaRelacionPropia, autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, recibo, decision, efecto, huella, auditoria string, consultadaEn time.Time) (personalports.ResultadoConsultaRelacionPropia, error) {
	var cero personalports.ResultadoConsultaRelacionPropia
	if n < 0 || n > 100 || len(b) == 0 || len(b) > 128<<10 || bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		return cero, errors.New("wire")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	var bruto []json.RawMessage
	if d.Decode(&bruto) != nil || d.Decode(new(any)) != io.EOF || len(bruto) != n {
		return cero, errors.New("wire")
	}
	relaciones := make([]personaldomain.RelacionEmpleado, 0, len(bruto))
	for _, objeto := range bruto {
		w, err := decodificarRelacionWire(objeto)
		if err != nil {
			return cero, err
		}
		desde, e := personaldomain.NuevaFechaCivil(w.Desde)
		if e != nil {
			return cero, e
		}
		var hasta personaldomain.FechaCivil
		if w.Hasta != nil {
			hasta, e = personaldomain.NuevaFechaCivil(*w.Hasta)
			if e != nil {
				return cero, e
			}
		}
		r := personaldomain.RelacionEmpleado{PersonaRef: w.PersonaRef, EmpleadoRef: w.EmpleadoRef, RelacionRef: w.RelacionRef, UnidadRef: w.UnidadRef, Estado: w.Estado, Desde: desde, Hasta: hasta, Version: w.Version, ProcedenciaActoRef: w.ProcedenciaActoRef, FuenteRef: w.FuenteRef, FuenteVersion: w.FuenteVersion}
		empleados, empleadoErr := c.Actor.Referencias("empleado")
		if empleadoErr != nil || len(empleados) != 1 || r.Validar() != nil || r.PersonaRef != c.Actor.PersonaRef || r.EmpleadoRef != empleados[0] || !r.VigenteEn(c.FechaReferencia) || (c.Operacion == personaldomain.OperacionDetalleRelacionPropia && r.RelacionRef != c.RelacionRef) {
			return cero, errors.New("wire")
		}
		relaciones = append(relaciones, r)
	}
	_, offset := consultadaEn.Zone()
	resumen := autorizacion.ResumenCapacidad()
	if !reciboRelacionValido.MatchString(recibo) || decision != resumen.DecisionRef() || efecto != resumen.EfectoRef() || !huellaRelacionHex.MatchString(huella) || auditoria == "" || consultadaEn.IsZero() || offset != 0 || consultadaEn.Nanosecond()%1000 != 0 || consultadaEn.Before(resumen.EmitidaEn()) || !consultadaEn.Before(resumen.ExpiraEn()) {
		return cero, errors.New("wire")
	}
	return personalports.ResultadoConsultaRelacionPropia{Relaciones: relaciones, Evidencia: personalports.EvidenciaConsultaRelacionPropia{ReciboRef: recibo, DecisionRef: decision, EfectoRef: efecto, ConsumoHuellaSHA256: huella, AuditoriaRef: auditoria, ConsultadaEn: consultadaEn.UTC()}}, nil
}

// PostgreSQL entrega un JSON contractual, no un objeto parcial cómodo para Go.
// En particular, hasta:null expresa explícitamente una relación abierta; una
// omisión no puede convertirse silenciosamente en esa misma semántica.
func decodificarRelacionWire(bruto json.RawMessage) (relacionWire, error) {
	var campos map[string]json.RawMessage
	if json.Unmarshal(bruto, &campos) != nil || len(campos) != 11 {
		return relacionWire{}, errors.New("wire")
	}
	for _, nombre := range []string{"desde", "empleado_ref", "estado", "fuente_ref", "fuente_version", "hasta", "persona_ref", "procedencia_acto_ref", "relacion_ref", "unidad_ref", "version"} {
		if _, existe := campos[nombre]; !existe {
			return relacionWire{}, errors.New("wire")
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(bruto))
	decoder.DisallowUnknownFields()
	var wire relacionWire
	if decoder.Decode(&wire) != nil || decoder.Decode(new(any)) != io.EOF {
		return relacionWire{}, errors.New("wire")
	}
	return wire, nil
}

var huellaRelacionHex = regexp.MustCompile(`^[a-f0-9]{64}$`)
var reciboRelacionValido = regexp.MustCompile(`^rpd_[0-9a-f]{32}$`)

func normalizarErrorRelacionEmpleado(ctx context.Context, err error) error {
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
	if errors.As(err, &pg) && pg.Code == "P7202" {
		return personalports.ErrRelacionEmpleadoAmbigua
	}
	if errors.As(err, &pg) && (pg.Code == "42501" || pg.Code == "22023" || pg.Code == "P0573") {
		return personalports.ErrRelacionEmpleadoNoDisponible
	}
	return personalports.ErrRelacionEmpleadoNoDisponible
}
func borrarRelacion(p [][]byte) {
	for _, b := range p {
		for i := range b {
			b[i] = 0
		}
	}
}
func nuloRelacion(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	return (x.Kind() == reflect.Ptr || x.Kind() == reflect.Interface || x.Kind() == reflect.Func || x.Kind() == reflect.Map || x.Kind() == reflect.Slice) && x.IsNil()
}
