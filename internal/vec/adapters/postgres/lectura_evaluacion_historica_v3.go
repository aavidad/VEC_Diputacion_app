package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"reflect"
	"time"
	"unicode/utf8"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const consultaEvaluacionOriginalV3 = "SELECT documento_asignacion,documento_rol,documento_control_rol,revision_catalogo,huella_catalogo,documentos_politicas FROM vec_autorizacion.leer_evaluacion_original_contexto_actor_v3($1,$2,$3)"

// Cota del transporte de esta fachada, no del dominio ni de exportaciones V3.
const maximoBytesEvaluacionOriginalV3 = 32 << 20

type RelojEvaluacionOriginalV3 interface{ Ahora() time.Time }
type LectorEvaluacionOriginalPostgreSQLV3 struct {
	pool  iniciadorTransacciones
	reloj RelojEvaluacionOriginalV3
}

var _ ports.LectorEvaluacionOriginalV3 = (*LectorEvaluacionOriginalPostgreSQLV3)(nil)

func NuevoLectorEvaluacionOriginalPostgreSQLV3(p *pgxpool.Pool, r RelojEvaluacionOriginalV3) (*LectorEvaluacionOriginalPostgreSQLV3, error) {
	return nuevoLectorEvaluacionOriginalPostgreSQLV3(p, r)
}
func nuevoLectorEvaluacionOriginalPostgreSQLV3(p iniciadorTransacciones, r RelojEvaluacionOriginalV3) (*LectorEvaluacionOriginalPostgreSQLV3, error) {
	if valorNuloPostgreSQL(p) || valorNuloPostgreSQL(r) {
		return nil, ports.ErrLecturaEvaluacionOriginalV3
	}
	return &LectorEvaluacionOriginalPostgreSQLV3{p, r}, nil
}
func errorEvaluacionOriginal(ctx context.Context) error {
	if !valorNuloPostgreSQL(ctx) && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrLecturaEvaluacionOriginalV3
}
func (l *LectorEvaluacionOriginalPostgreSQLV3) LeerEvaluacionOriginalV3(ctx context.Context, s ports.SolicitudLecturaEvaluacionOriginalV3) (domain.InstantaneaAutorizacion, error) {
	var cero domain.InstantaneaAutorizacion
	if valorNuloPostgreSQL(ctx) || l == nil || valorNuloPostgreSQL(l.pool) || valorNuloPostgreSQL(l.reloj) || s.Validar() != nil {
		return cero, errorEvaluacionOriginal(ctx)
	}
	var ultimo time.Time
	validar := func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		t := l.reloj.Ahora()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if t.IsZero() || t.Location() != time.UTC || t.Year() < 1 || t.Year() > 9999 || t.Nanosecond()%1000 != 0 || t.Before(ultimo) {
			return ports.ErrLecturaEvaluacionOriginalV3
		}
		ultimo = t
		return ctx.Err()
	}
	if e := validar(); e != nil {
		return cero, e
	}
	tx, e := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	confirmado := false
	if !valorNuloPostgreSQL(tx) {
		defer func() {
			if !confirmado {
				c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = tx.Rollback(c)
			}
		}()
	}
	if e != nil || valorNuloPostgreSQL(tx) {
		return cero, errorEvaluacionOriginal(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	if e = configurarTransaccionAutorizacion(ctx, tx); e != nil {
		return cero, errorEvaluacionOriginal(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	rows, e := tx.Query(ctx, consultaEvaluacionOriginalV3, s.DecisionRef, s.DecisionSHA256, s.SolicitudSHA256)
	if !valorNuloPostgreSQL(rows) {
		defer rows.Close()
	}
	if e != nil || valorNuloPostgreSQL(rows) {
		return cero, errorEvaluacionOriginal(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	if !rows.Next() {
		return cero, errorEvaluacionOriginal(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	var docs [4][]byte
	var rev, hash string
	if e = rows.Scan(&docs[0], &docs[1], &docs[2], &rev, &hash, &docs[3]); e != nil {
		return cero, errorEvaluacionOriginal(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	var i domain.InstantaneaAutorizacion
	if len(rev) < 1 || len(rev) > 20 || rev[0] < '1' || rev[0] > '9' || len(hash) != 64 {
		return cero, errorEvaluacionOriginal(ctx)
	}
	i.RevisionCatalogoPoliticas, e = parsearRevisionAutorizacion(rev)
	if e != nil {
		return cero, errorEvaluacionOriginal(ctx)
	}
	i.CatalogoPoliticasHuellaSHA256 = hash
	budget := maximoBytesEvaluacionOriginalV3
	for _, d := range docs {
		if len(d) == 0 || len(d) > budget {
			return cero, errorEvaluacionOriginal(ctx)
		}
		budget -= len(d)
	}
	dests := []any{&i.AsignacionPerfil, &i.VersionRol, &i.ControlVigenciaVersionRol, &i.Politicas}
	for n, d := range docs {
		if decodificarEvaluacionOriginal(d, dests[n]) != nil {
			return cero, errorEvaluacionOriginal(ctx)
		}
	}
	if i.Validar() != nil {
		return cero, errorEvaluacionOriginal(ctx)
	}
	if rows.Next() || rows.Err() != nil {
		return cero, errorEvaluacionOriginal(ctx)
	}
	rows.Close()
	if e = validar(); e != nil {
		return cero, e
	}
	// El decoder crea datos propios. Una segunda copia impide alias entre capas.
	salida, e := ports.CopiarInstantaneaEvaluacionOriginalV3(i)
	if e != nil {
		return cero, errorEvaluacionOriginal(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	if e = tx.Commit(ctx); e != nil {
		return cero, errorEvaluacionOriginal(ctx)
	}
	confirmado = true
	if e = validar(); e != nil {
		return cero, e
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	return salida, nil
}

// JSON de structs públicos originales, no un codec de decisión nominal. Exige
// claves/tipos exactos y conserva fechas/números: no corrige alias, null o escala.
func decodificarEvaluacionOriginal(b []byte, dest any) error {
	if len(b) == 0 || len(b) > maximoBytesEvaluacionOriginalV3 || !utf8.Valid(b) || valorNuloPostgreSQL(dest) {
		return ports.ErrLecturaEvaluacionOriginalV3
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	nodos := 0
	if nodoEvaluacionOriginal(d, 0, &nodos) != nil {
		return ports.ErrLecturaEvaluacionOriginalV3
	}
	if _, e := d.Token(); e != io.EOF {
		return ports.ErrLecturaEvaluacionOriginalV3
	}
	if decodificarDocumentoPostgreSQL(b, dest) != nil {
		return ports.ErrLecturaEvaluacionOriginalV3
	}
	c, e := json.Marshal(dest)
	if e != nil {
		return ports.ErrLecturaEvaluacionOriginalV3
	}
	var x, y any
	d = json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if d.Decode(&x) != nil {
		return ports.ErrLecturaEvaluacionOriginalV3
	}
	d = json.NewDecoder(bytes.NewReader(c))
	d.UseNumber()
	if d.Decode(&y) != nil || !reflect.DeepEqual(x, y) {
		return ports.ErrLecturaEvaluacionOriginalV3
	}
	return nil
}
func nodoEvaluacionOriginal(d *json.Decoder, depth int, nodes *int) error {
	*nodes++
	if depth > 32 || *nodes > 1000000 {
		return ports.ErrLecturaEvaluacionOriginalV3
	}
	t, e := d.Token()
	if e != nil {
		return e
	}
	switch t {
	case json.Delim('{'):
		seen := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			if e != nil {
				return e
			}
			s, ok := k.(string)
			if !ok || seen[s] {
				return ports.ErrLecturaEvaluacionOriginalV3
			}
			seen[s] = true
			if e = nodoEvaluacionOriginal(d, depth+1, nodes); e != nil {
				return e
			}
		}
		t, e = d.Token()
		if e != nil || t != json.Delim('}') {
			return ports.ErrLecturaEvaluacionOriginalV3
		}
	case json.Delim('['):
		for d.More() {
			if e = nodoEvaluacionOriginal(d, depth+1, nodes); e != nil {
				return e
			}
		}
		t, e = d.Token()
		if e != nil || t != json.Delim(']') {
			return ports.ErrLecturaEvaluacionOriginalV3
		}
	}
	return nil
}
