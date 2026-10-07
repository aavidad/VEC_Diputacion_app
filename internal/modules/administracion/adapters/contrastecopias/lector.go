// Package contrastecopias captura evidencia lógica privada, nunca modifica PostgreSQL.
package contrastecopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	domain "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
	ports "vec-diputacion-granada/internal/modules/administracion/ports/contrastecopias"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

type Configuracion struct {
	DSN                       string
	VersionPostgreSQL         string
	TiempoMaximo              time.Duration
	MaxFilas                  int64
	MaxBytes                  int64
	MaxObjetos                int
	ObjetosGrandesSemanticos  bool
	ReferenciasObjetosGrandes []ReferenciaObjetoGrande
	BasesInventariadas        []string
}

// ReferenciaObjetoGrande declara una columna cuyo oid es identidad lógica LO.
type ReferenciaObjetoGrande struct{ Esquema, Tabla, Columna, Base string }

type Lector struct {
	config  *pgx.ConnConfig
	limites Configuracion
}

var version = regexp.MustCompile(`^18\.[0-9]+$`)
var errCaptura = errors.New("captura_postgresql_no_disponible")
var errLimite = errors.New("limite_captura_postgresql")

func Nuevo(c Configuracion) (*Lector, error) {
	if !version.MatchString(c.VersionPostgreSQL) || c.TiempoMaximo <= 0 || c.TiempoMaximo > 10*time.Minute || c.MaxFilas < 1 || c.MaxFilas > 10000000 || c.MaxBytes < 1 || c.MaxBytes > 1<<30 || c.MaxObjetos < 8 || c.MaxObjetos > 100000 {
		return nil, errors.New("configuracion_contraste_postgresql_no_admitida")
	}
	if e := validarBases(c); e != nil {
		return nil, e
	}
	if e := validarReferencias(c); e != nil {
		return nil, e
	}
	c.ReferenciasObjetosGrandes = append([]ReferenciaObjetoGrande(nil), c.ReferenciasObjetosGrandes...)
	c.BasesInventariadas = append([]string(nil), c.BasesInventariadas...)
	sort.Strings(c.BasesInventariadas)
	var config *pgx.ConnConfig
	if c.DSN == "" {
		return &Lector{limites: c}, nil
	}
	config, e := pgx.ParseConfig(c.DSN)
	if e != nil {
		return nil, errors.New("configuracion_contraste_postgresql_no_admitida")
	}
	// No heredamos search_path ni GUC aportados por la cadena de conexión.
	config.RuntimeParams = map[string]string{"application_name": "vec_cs06_lector", "default_transaction_read_only": "on", "search_path": "pg_catalog", "client_encoding": "UTF8"}
	config.ConnectTimeout = c.TiempoMaximo
	return &Lector{config: config, limites: c}, nil
}

type consultaSQL interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type captura struct {
	tx            consultaSQL
	l             *Lector
	bytes, filas  int64
	s             domain.Snapshot
	guard         ports.ExclusionObservada
	sello         string
	presupuesto   *presupuestoCaptura
	baseMetadatos string
}

func (l *Lector) Capturar(ctx context.Context) (domain.Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, plazoarranque.Ampliar(l.limites.TiempoMaximo))
	defer cancel()
	if len(l.limites.BasesInventariadas) != 0 {
		return domain.Snapshot{}, errors.New("captura_multibase_requiere_ejecutor")
	}
	if l.config == nil {
		return domain.Snapshot{}, errCaptura
	}
	conn, e := pgx.ConnectConfig(ctx, l.config.Copy())
	if e != nil {
		return domain.Snapshot{}, errCaptura
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(time.Second))
		defer cancel()
		_ = conn.Close(cleanup)
	}()
	tx, e := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return domain.Snapshot{}, errCaptura
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(time.Second))
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	for _, sql := range []string{`SET LOCAL client_encoding = 'UTF8'`, `SET LOCAL search_path = pg_catalog`, `SET LOCAL timezone = 'UTC'`, `SET LOCAL datestyle = 'ISO, YMD'`, `SET LOCAL intervalstyle = 'postgres'`, `SET LOCAL extra_float_digits = 3`, `SET LOCAL bytea_output = 'hex'`, `SET LOCAL row_security = off`, `SET LOCAL standard_conforming_strings = on`} {
		if _, e = tx.Exec(ctx, sql); e != nil {
			return domain.Snapshot{}, errCaptura
		}
	}
	if _, e = tx.Exec(ctx, `SELECT pg_catalog.set_config('statement_timeout',$1,true)`, strconv.FormatInt(l.limites.TiempoMaximo.Milliseconds(), 10)); e != nil {
		return domain.Snapshot{}, errCaptura
	}
	snapshot, e := l.capturarSQL(ctx, tx, nil)
	if e != nil {
		return domain.Snapshot{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return domain.Snapshot{}, errCaptura
	}
	return snapshot, nil
}

func (l *Lector) capturarSQL(ctx context.Context, tx consultaSQL, guard ports.ExclusionObservada) (domain.Snapshot, error) {
	return l.capturarSQLAcotada(ctx, tx, guard, false, nil, "")
}

// El coordinador valida el ámbito completo antes de suprimir la guarda local.
// Todas las bases consumen el mismo presupuesto de filas, bytes y objetos.
func (l *Lector) capturarSQLAcotada(ctx context.Context, tx consultaSQL, guard ports.ExclusionObservada, ambitoCompleto bool, presupuesto *presupuestoCaptura, baseMetadatos string) (domain.Snapshot, error) {
	if presupuesto == nil {
		presupuesto = &presupuestoCaptura{objetos: 8}
	}
	var e error
	c := &captura{tx: tx, l: l, guard: guard, presupuesto: presupuesto, baseMetadatos: baseMetadatos, bytes: presupuesto.bytes, filas: presupuesto.filas, s: domain.Snapshot{Version: 1, PostgreSQL: l.limites.VersionPostgreSQL, Completo: true, Motivos: []string{}, Objetos: []domain.Objeto{}}}
	defer func() { presupuesto.bytes = c.bytes; presupuesto.filas = c.filas }()
	paused, e := c.exclusion(ctx)
	if e != nil {
		return domain.Snapshot{}, errCaptura
	}
	if !paused {
		c.motivo("escritores_no_excluidos")
	}
	var observed int
	if e = tx.QueryRow(ctx, `SELECT current_setting('server_version_num')::int`).Scan(&observed); e != nil {
		return domain.Snapshot{}, errCaptura
	}
	actual := fmt.Sprintf("%d.%d", observed/10000, observed%10000)
	if actual != l.limites.VersionPostgreSQL {
		c.motivo("version_postgresql_no_admitida")
		c.s.PostgreSQL = actual
		return c.s, nil
	}
	for _, check := range comprobaciones {
		if ambitoCompleto && check.motivo == "otras_bases_no_inventariadas" {
			continue
		}
		var existe bool
		if e = tx.QueryRow(ctx, consultaComprobacion(check, l.limites.ObjetosGrandesSemanticos)).Scan(&existe); e != nil {
			return domain.Snapshot{}, errCaptura
		}
		if existe {
			c.motivo(check.motivo)
		}
	}
	for _, q := range agregados {
		rows, e := c.leer(ctx, c.consultaAgregado(q))
		if e != nil {
			return domain.Snapshot{}, e
		}
		if e = c.agregar(domain.Objeto{Clase: q.clase, Clave: "inventario", Cantidad: int64(len(rows)), SHA256: huella(rows)}); e != nil {
			return domain.Snapshot{}, e
		}
	}
	if e = c.validarReferencias(ctx); e != nil {
		return domain.Snapshot{}, e
	}
	if e = c.tablas(ctx); e != nil {
		return domain.Snapshot{}, e
	}
	seq, e := c.secuencias(ctx)
	if e != nil {
		return domain.Snapshot{}, e
	}
	if e = c.grandes(ctx); e != nil {
		return domain.Snapshot{}, e
	}
	// Las secuencias no son MVCC: dos lecturas y exclusión continua son obligatorias.
	again, e := c.secuenciasValores(ctx)
	if e != nil {
		return domain.Snapshot{}, e
	}
	if huella(seq) != huella(again) {
		c.motivo("secuencias_inestables")
	}
	// Los catálogos compartidos (roles/globals) y el esquema se vuelven a leer:
	// la estabilidad de contenido no se infiere del sello de aislamiento solo.
	for _, q := range agregados {
		rs, e := c.leer(ctx, c.consultaAgregado(q))
		if e != nil {
			return domain.Snapshot{}, e
		}
		for _, o := range c.s.Objetos {
			if o.Clase == q.clase && o.Clave == "inventario" && (o.SHA256 != huella(rs) || o.Cantidad != int64(len(rs))) {
				c.motivo("catalogo_inestable")
			}
		}
	}
	after, e := c.exclusion(ctx)
	if e != nil {
		return domain.Snapshot{}, errCaptura
	}
	if !after {
		c.motivo("escritores_no_excluidos")
	}
	for _, clase := range []string{"tablas", "secuencias", "objetos_grandes"} {
		if e = c.agregar(domain.Resumir(clase, c.s.Objetos)); e != nil {
			return domain.Snapshot{}, e
		}
	}

	sort.Slice(c.s.Objetos, func(i, j int) bool {
		a, b := c.s.Objetos[i], c.s.Objetos[j]
		if a.Clase != b.Clase {
			return a.Clase < b.Clase
		}
		return a.Clave < b.Clave
	})
	sort.Strings(c.s.Motivos)
	encoded, e := json.Marshal(c.s)
	if e != nil {
		return domain.Snapshot{}, errCaptura
	}
	if len(encoded) > 32<<20 {
		return domain.Snapshot{}, errLimite
	}
	return c.s, nil
}

func (c *captura) motivo(m string) {
	c.s.Completo = false
	for _, v := range c.s.Motivos {
		if v == m {
			return
		}
	}
	c.s.Motivos = append(c.s.Motivos, m)
}
func (c *captura) agregar(o domain.Objeto) error {
	if len(c.s.Objetos) >= c.l.limites.MaxObjetos || len(o.Clave) > 1024 {
		return errLimite
	}
	if o.Clave != "inventario" && c.presupuesto != nil {
		if c.presupuesto.objetos >= c.l.limites.MaxObjetos {
			return errLimite
		}
		c.presupuesto.objetos++
	}
	c.s.Objetos = append(c.s.Objetos, o)
	return nil
}
func huella(rows []string) string {
	copyRows := append([]string{}, rows...)
	sort.Strings(copyRows)
	h := sha256.New()
	for _, row := range copyRows {
		b, _ := json.Marshal(row)
		h.Write(b)
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// leer recibe una sola columna JSON textual. El presupuesto se aplica en el servidor
// antes de transferir cada registro; toda la captura comparte filas y bytes máximos.
func (c *captura) leer(ctx context.Context, sql string, args ...any) ([]string, error) {
	remaining := c.l.limites.MaxBytes - c.bytes
	if remaining < 0 {
		return nil, errLimite
	}
	n := len(args)
	args = append(args, remaining, c.l.limites.MaxFilas-c.filas+1)
	wrapped := fmt.Sprintf(`SELECT CASE WHEN octet_length(v)>$%d THEN NULL ELSE v END FROM (%s) entrada(v) LIMIT $%d`, n+1, sql, n+2)
	rows, e := c.tx.Query(ctx, wrapped, args...)
	if e != nil {
		return nil, errCaptura
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var v *string
		if e = rows.Scan(&v); e != nil {
			return nil, errCaptura
		}
		if v == nil {
			return nil, errLimite
		}
		if !utf8.ValidString(*v) {
			return nil, errCaptura
		}
		c.bytes += int64(len(*v))
		c.filas++
		if c.bytes > c.l.limites.MaxBytes || c.filas > c.l.limites.MaxFilas {
			return nil, errLimite
		}
		result = append(result, *v)
	}
	if rows.Err() != nil {
		return nil, errCaptura
	}
	sort.Strings(result)
	return result, nil
}

// default_transaction_read_only o ausencia momentánea de sesiones no prueban
// exclusión: solo una réplica realmente pausada puede cerrar este lector V1.
func (c *captura) exclusion(ctx context.Context) (bool, error) {
	if c.guard != nil {
		sello, e := c.guard.ComprobarExclusion(ctx)
		if e != nil || !selloValido.MatchString(sello) {
			return false, errCaptura
		}
		if c.sello == "" {
			c.sello = sello
		}
		return sello == c.sello, nil
	}
	var standby bool
	if e := c.tx.QueryRow(ctx, `SELECT pg_catalog.pg_is_in_recovery()`).Scan(&standby); e != nil {
		return false, e
	}
	if !standby {
		return false, nil
	}
	var paused string
	var receive, replay *string
	e := c.tx.QueryRow(ctx, `SELECT pg_catalog.pg_get_wal_replay_pause_state(),pg_catalog.pg_last_wal_receive_lsn()::text,pg_catalog.pg_last_wal_replay_lsn()::text`).Scan(&paused, &receive, &replay)
	// receive puede avanzar mientras la aplicación permanece pausada. replay es el
	// punto semántico, no se publica ni se contrasta con otra restauración.
	if e != nil {
		return false, e
	}
	if replay == nil {
		return false, nil
	}
	if c.sello == "" {
		c.sello = *replay
	}
	return paused == "paused" && *replay == c.sello, nil
}

func (c *captura) tablas(ctx context.Context) error {
	names, e := c.leer(ctx, `SELECT jsonb_build_array(n.nspname,r.relname)::text FROM pg_class r JOIN pg_namespace n ON n.oid=r.relnamespace JOIN pg_am am ON am.oid=r.relam WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema' AND r.relkind='r' AND am.amname='heap' ORDER BY n.nspname,r.relname`)
	if e != nil {
		return e
	}
	for _, name := range names {
		var parts []string
		if json.Unmarshal([]byte(name), &parts) != nil || len(parts) != 2 {
			return errCaptura
		}
		cols, e := c.leer(ctx, `SELECT jsonb_build_array(a.attname,t.typname,tn.nspname,a.attgenerated)::text FROM pg_attribute a JOIN pg_type t ON t.oid=a.atttypid JOIN pg_namespace tn ON tn.oid=t.typnamespace WHERE a.attrelid=to_regclass($1) AND a.attnum>0 AND NOT a.attisdropped ORDER BY a.attnum`, pgx.Identifier(parts).Sanitize())
		if e != nil {
			return e
		}
		// El orden de columnas es explícito y estable; leer ordena sus registros, por
		// eso reconstruimos por nombre (el esquema conserva attnum por separado).
		expr := []string{}
		unsupported := false
		for _, col := range cols {
			var p []string
			if json.Unmarshal([]byte(col), &p) != nil || len(p) != 4 {
				return errCaptura
			}
			oidAdmitido := p[1] == "oid" && c.l.limites.ObjetosGrandesSemanticos && c.referenciaDeclarada(parts[0], parts[1], p[0])
			if p[2] != "pg_catalog" || p[3] == "v" || (!tipos[p[1]] && !oidAdmitido) {
				unsupported = true
				continue
			}
			expr = append(expr, `jsonb_build_array(`+literal(p[0])+`,`+literal(p[1])+`,`+pgx.Identifier{p[0]}.Sanitize()+`::text)`)
		}
		if unsupported {
			c.motivo("tipo_columna_no_admitido")
			continue
		}
		sql := `SELECT jsonb_build_array(` + strings.Join(expr, ",") + `)::text FROM ` + pgx.Identifier(parts).Sanitize()
		rows, e := c.leer(ctx, sql)
		if e != nil {
			return e
		}
		if e = c.agregar(domain.Objeto{Clase: "tablas", Clave: pgx.Identifier(parts).Sanitize(), Cantidad: int64(len(rows)), SHA256: huella(rows)}); e != nil {
			return e
		}
	}
	return nil
}
func literal(v string) string { return "'" + strings.ReplaceAll(v, "'", "''") + "'" }

var tipos = map[string]bool{"bool": true, "int2": true, "int4": true, "int8": true, "float4": true, "float8": true, "numeric": true, "text": true, "varchar": true, "bpchar": true, "bytea": true, "uuid": true, "date": true, "time": true, "timetz": true, "timestamp": true, "timestamptz": true, "interval": true, "json": true, "jsonb": true}

func (c *captura) secuenciasValores(ctx context.Context) ([]string, error) {
	names, e := c.leer(ctx, `SELECT jsonb_build_array(n.nspname,r.relname)::text FROM pg_class r JOIN pg_namespace n ON n.oid=r.relnamespace WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema' AND r.relkind='S' ORDER BY n.nspname,r.relname`)
	if e != nil {
		return nil, e
	}
	values := []string{}
	for _, name := range names {
		var p []string
		if json.Unmarshal([]byte(name), &p) != nil || len(p) != 2 {
			return nil, errCaptura
		}
		rs, e := c.leer(ctx, `SELECT jsonb_build_array(`+literal(pgx.Identifier(p).Sanitize())+`,last_value::text,is_called)::text FROM `+pgx.Identifier(p).Sanitize())
		if e != nil {
			return nil, e
		}
		values = append(values, rs...)
	}
	return values, nil
}
func (c *captura) secuencias(ctx context.Context) ([]string, error) {
	rows, e := c.secuenciasValores(ctx)
	if e != nil {
		return nil, e
	}
	for _, r := range rows {
		var fields []json.RawMessage
		if json.Unmarshal([]byte(r), &fields) != nil || len(fields) != 3 {
			return nil, errCaptura
		}
		var key string
		if json.Unmarshal(fields[0], &key) != nil {
			return nil, errCaptura
		}
		if e = c.agregar(domain.Objeto{Clase: "secuencias", Clave: key, Cantidad: 1, SHA256: huella([]string{r})}); e != nil {
			return nil, e
		}
	}
	return rows, nil
}

func (c *captura) grandes(ctx context.Context) error {
	if c.l.limites.ObjetosGrandesSemanticos {
		return c.grandesSemanticos(ctx)
	}
	// OID solo se usa como localizador efímero. La identidad publicada sella dueño,
	// ACL y páginas (conserva huecos), más ordinal para objetos idénticos duplicados.
	var count int64
	if e := c.tx.QueryRow(ctx, `SELECT count(*) FROM pg_largeobject_metadata`).Scan(&count); e != nil {
		return errCaptura
	}
	if count > int64(c.l.limites.MaxObjetos) {
		return errLimite
	}
	rows, e := c.tx.Query(ctx, `SELECT oid::bigint FROM pg_largeobject_metadata ORDER BY oid`)
	if e != nil {
		return errCaptura
	}
	ids := []uint32{}
	for rows.Next() {
		var id uint32
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return errCaptura
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return errCaptura
	}
	items := []string{}
	for _, id := range ids {
		meta, e := c.leer(ctx, `SELECT jsonb_build_array(pg_get_userbyid(l.lomowner),(SELECT jsonb_agg(jsonb_build_array(pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable) ORDER BY a.grantor::regrole::text,a.grantee::regrole::text,a.privilege_type,a.is_grantable) FROM aclexplode(coalesce(l.lomacl,acldefault('L',l.lomowner))) a))::text FROM pg_largeobject_metadata l WHERE oid=$1`, id)
		if e != nil {
			return e
		}
		content, e := c.leer(ctx, `SELECT jsonb_build_array(pageno,encode(data,'hex'))::text FROM pg_largeobject WHERE loid=$1`, id)
		if e != nil {
			return e
		}
		b, _ := json.Marshal([]any{meta, huella(content), len(content)})
		items = append(items, string(b))
	}
	sort.Strings(items)
	ord := map[string]int{}
	for _, r := range items {
		h := huella([]string{r})
		ord[h]++
		key := fmt.Sprintf("contenido:%s:%d", h, ord[h])
		if e = c.agregar(domain.Objeto{Clase: "objetos_grandes", Clave: key, Cantidad: 1, SHA256: h}); e != nil {
			return e
		}
	}
	return nil
}
