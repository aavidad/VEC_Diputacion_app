package contrastecopias

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	domain "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
	ports "vec-diputacion-granada/internal/modules/administracion/ports/contrastecopias"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

var selloValido = regexp.MustCompile(`^[0-9a-f]{64}$`)
var baseAdmitida = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$-]{0,62}$`)

// CapturarEjecutor usa exclusivamente consultas fijas del lector. Cada consulta
// abre REPEATABLE READ READ ONLY; con la ventana externa exclusiva observada,
// todas pertenecen al mismo estado. Sin el guard no existe evidencia completa.
func (l *Lector) CapturarEjecutor(ctx context.Context, exec ports.EjecutorPostgreSQL, base string, exclusion ports.ExclusionObservada) (domain.Snapshot, error) {
	return l.CapturarEjecutorConFuente(ctx, exec, base, exclusion, nil)
}

// La fuente opcional aporta evidencia observada de bases no conectables.
func (l *Lector) CapturarEjecutorConFuente(ctx context.Context, exec ports.EjecutorPostgreSQL, base string, exclusion ports.ExclusionObservada, fuente FuenteBaseNoConectable) (domain.Snapshot, error) {
	if exec == nil || exclusion == nil || !baseAdmitida.MatchString(base) {
		return domain.Snapshot{}, errCaptura
	}
	ctx, cancel := context.WithTimeout(ctx, plazoarranque.Ampliar(l.limites.TiempoMaximo))
	defer cancel()
	if len(l.limites.BasesInventariadas) != 0 {
		return l.capturarBases(ctx, exec, base, exclusion, fuente)
	}
	tx := &transporteEjecutor{exec: exec, base: base, maxBytes: l.limites.MaxBytes, timeout: l.limites.TiempoMaximo.Milliseconds()}
	return l.capturarSQL(ctx, tx, exclusion)
}

type transporteEjecutor struct {
	exec              ports.EjecutorPostgreSQL
	base              string
	maxBytes, timeout int64
}

// bind reconoce tokens SQL; nunca sustituye $n dentro de nombres, literales,
// comentarios ni dollar quotes obtenidos del catálogo.
func bind(sql string, args []any) (string, error) {
	var out strings.Builder
	for i := 0; i < len(sql); {
		start := i
		if sql[i] == '\'' || sql[i] == '"' {
			quote := sql[i]
			i++
			for i < len(sql) {
				if sql[i] == quote {
					i++
					if i < len(sql) && sql[i] == quote {
						i++
						continue
					}
					break
				}
				i++
			}
			out.WriteString(sql[start:i])
			continue
		}
		if strings.HasPrefix(sql[i:], "--") {
			i += 2
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			out.WriteString(sql[start:i])
			continue
		}
		if strings.HasPrefix(sql[i:], "/*") {
			depth := 1
			i += 2
			for i < len(sql) && depth > 0 {
				if strings.HasPrefix(sql[i:], "/*") {
					depth++
					i += 2
				} else if strings.HasPrefix(sql[i:], "*/") {
					depth--
					i += 2
				} else {
					i++
				}
			}
			out.WriteString(sql[start:i])
			continue
		}
		if sql[i] == '$' {
			j := i + 1
			for j < len(sql) && ((sql[j] >= 'a' && sql[j] <= 'z') || (sql[j] >= 'A' && sql[j] <= 'Z') || sql[j] == '_' || (j > i+1 && sql[j] >= '0' && sql[j] <= '9')) {
				j++
			}
			if j < len(sql) && sql[j] == '$' {
				tag := sql[i : j+1]
				end := strings.Index(sql[j+1:], tag)
				if end < 0 {
					return "", errCaptura
				}
				i = j + 1 + end + len(tag)
				out.WriteString(sql[start:i])
				continue
			}
			j = i + 1
			for j < len(sql) && sql[j] >= '0' && sql[j] <= '9' {
				j++
			}
			if j > i+1 {
				n, e := strconv.Atoi(sql[i+1 : j])
				if e != nil || n < 1 || n > len(args) {
					return "", errCaptura
				}
				var v string
				switch x := args[n-1].(type) {
				case string:
					v = literal(x)
				case int64:
					v = strconv.FormatInt(x, 10)
				case int:
					v = strconv.Itoa(x)
				case uint32:
					v = strconv.FormatUint(uint64(x), 10)
				default:
					return "", errCaptura
				}
				out.WriteString(v)
				i = j
				continue
			}
		}
		out.WriteByte(sql[i])
		i++
	}
	return out.String(), nil
}

func (t *transporteEjecutor) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	_, e := t.Query(ctx, sql, args...)
	return pgconn.NewCommandTag("SELECT"), e
}
func (t *transporteEjecutor) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	r, e := t.Query(ctx, sql, args...)
	return filaEjecutor{r, e}
}
func (t *transporteEjecutor) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	sql, e := bind(sql, args)
	if e != nil {
		return nil, e
	}
	// -q elimina tags BEGIN/SET/COMMIT; row_to_json conserva orden de columnas y
	// escapa contenido. No se interpretan nombres/celdas como comandos psql.
	input := `BEGIN TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL client_encoding='UTF8';
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL datestyle='ISO, YMD';
SET LOCAL intervalstyle='postgres';
SET LOCAL extra_float_digits=3;
SET LOCAL bytea_output='hex';
SET LOCAL row_security=off;
SET LOCAL standard_conforming_strings=on;
SET LOCAL statement_timeout=` + strconv.FormatInt(t.timeout, 10) + `;
SELECT row_to_json(_vec)::text FROM (` + sql + `) _vec;
COMMIT;
`
	out, e := t.exec.EjecutarPostgreSQL(ctx, "psql", []string{"-X", "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1", "--dbname=" + t.base}, []byte(input), int(t.maxBytes)+1<<20)
	if e != nil {
		return nil, errCaptura
	}
	if int64(len(out)) > t.maxBytes+1<<20 {
		return nil, errLimite
	}
	if !utf8.Valid(out) {
		return nil, errCaptura
	}
	rows := &filasEjecutor{at: -1}
	for _, line := range bytes.Split(out, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		d := json.NewDecoder(bytes.NewReader(line))
		token, e := d.Token()
		if e != nil || token != json.Delim('{') {
			return nil, errCaptura
		}
		values := []json.RawMessage{}
		for d.More() {
			if _, e = d.Token(); e != nil {
				return nil, errCaptura
			}
			var v json.RawMessage
			if d.Decode(&v) != nil {
				return nil, errCaptura
			}
			values = append(values, v)
		}
		if _, e = d.Token(); e != nil {
			return nil, errCaptura
		}
		if _, e = d.Token(); e != io.EOF {
			return nil, errCaptura
		}
		rows.rows = append(rows.rows, values)
	}
	return rows, nil
}

type filaEjecutor struct {
	r   pgx.Rows
	err error
}

func (f filaEjecutor) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	defer f.r.Close()
	if !f.r.Next() {
		return pgx.ErrNoRows
	}
	return f.r.Scan(dest...)
}

type filasEjecutor struct {
	rows   [][]json.RawMessage
	at     int
	closed bool
	err    error
}

func (r *filasEjecutor) Close()                                       { r.closed = true }
func (r *filasEjecutor) Err() error                                   { return r.err }
func (r *filasEjecutor) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT") }
func (r *filasEjecutor) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *filasEjecutor) Conn() *pgx.Conn                              { return nil }
func (r *filasEjecutor) Next() bool {
	if r.closed {
		return false
	}
	r.at++
	return r.at < len(r.rows)
}
func (r *filasEjecutor) Scan(dest ...any) error {
	if r.at < 0 || r.at >= len(r.rows) || len(dest) != len(r.rows[r.at]) {
		return errCaptura
	}
	for i, v := range r.rows[r.at] {
		if json.Unmarshal(v, dest[i]) != nil {
			return errCaptura
		}
	}
	return nil
}
func (r *filasEjecutor) Values() ([]any, error) {
	if r.at < 0 || r.at >= len(r.rows) {
		return nil, errCaptura
	}
	out := make([]any, len(r.rows[r.at]))
	for i, v := range r.rows[r.at] {
		if json.Unmarshal(v, &out[i]) != nil {
			return nil, errCaptura
		}
	}
	return out, nil
}
func (r *filasEjecutor) RawValues() [][]byte {
	if r.at < 0 || r.at >= len(r.rows) {
		return nil
	}
	out := make([][]byte, len(r.rows[r.at]))
	for i, v := range r.rows[r.at] {
		out[i] = v
	}
	return out
}
