package contrastecopias

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Los identificadores de este grafo son localizadores efímeros. Nunca forman
// parte de la expresión canónica ni de las evidencias publicadas.
type tipoCanon struct {
	OID               uint32 `json:"oid"`
	Esquema           string `json:"schema"`
	Nombre            string `json:"name"`
	Clase             string `json:"kind"`
	Base              uint32 `json:"base"`
	Elemento          uint32 `json:"element"`
	Relacion          uint32 `json:"relation"`
	Categoria         string `json:"category"`
	SalidaSegura      bool   `json:"safe_output"`
	ModificadorSeguro bool   `json:"safe_typmod"`
	ArraySeguro       bool   `json:"safe_array"`
	campos            []campoCanon
}

type campoCanon struct {
	Relacion uint32 `json:"relation"`
	Posicion int    `json:"position"`
	Nombre   string `json:"name"`
	Tipo     uint32 `json:"type"`
	Generada string `json:"generated"`
}

type canonTipos struct {
	c         *captura
	tipos     map[uint32]*tipoCanon
	noDeparse bool
}

// NoDeparseSeguro impide deparsear constantes de tipos cuya salida podría
// ejecutar código del destino. Se consulta antes de pg_get_expr/format_type.
func (t *canonTipos) NoDeparseSeguro() bool { return t.noDeparse }

const grafoTiposSQL = `SELECT pg_catalog.jsonb_build_object(
 'oid',t.oid::bigint,'schema',n.nspname,'name',t.typname,'kind',t.typtype,
 'base',t.typbasetype::bigint,'element',t.typelem::bigint,'relation',t.typrelid::bigint,'category',t.typcategory,
 'safe_output',coalesce(p.oid<16384 AND pn.nspname='pg_catalog' AND pl.lanname='internal' AND p.probin IS NULL
   AND (p.prosrc=p.proname OR (n.nspname='pg_catalog' AND t.oid<16384 AND t.typname='txid_snapshot'
     AND p.oid=2940 AND p.proname='txid_snapshot_out' AND p.prosrc='pg_snapshot_out')),false),
 'safe_typmod',t.typmodout=0 OR coalesce(m.oid<16384 AND mn.nspname='pg_catalog' AND ml.lanname='internal'
   AND m.probin IS NULL AND m.prosrc=m.proname,false),
 'safe_array',coalesce(s.oid<16384 AND sn.nspname='pg_catalog' AND sl.lanname='internal' AND s.probin IS NULL
   AND s.prosrc=s.proname AND s.proname='array_subscript_handler',false)
)::text FROM pg_catalog.pg_type t
 JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
 LEFT JOIN pg_catalog.pg_proc p ON p.oid=t.typoutput
 LEFT JOIN pg_catalog.pg_namespace pn ON pn.oid=p.pronamespace
 LEFT JOIN pg_catalog.pg_language pl ON pl.oid=p.prolang
 LEFT JOIN pg_catalog.pg_proc m ON m.oid=t.typmodout
 LEFT JOIN pg_catalog.pg_namespace mn ON mn.oid=m.pronamespace
 LEFT JOIN pg_catalog.pg_language ml ON ml.oid=m.prolang
 LEFT JOIN pg_catalog.pg_proc s ON s.oid=t.typsubscript
 LEFT JOIN pg_catalog.pg_namespace sn ON sn.oid=s.pronamespace
 LEFT JOIN pg_catalog.pg_language sl ON sl.oid=s.prolang`

const camposTiposSQL = `SELECT pg_catalog.jsonb_build_object(
 'relation',a.attrelid::bigint,'position',a.attnum,'name',a.attname,'type',a.atttypid::bigint,'generated',a.attgenerated
)::text FROM pg_catalog.pg_attribute a
 JOIN pg_catalog.pg_type t ON t.typrelid=a.attrelid AND t.typtype='c'
 WHERE a.attnum>0 AND NOT a.attisdropped`

func (c *captura) prepararTipos(ctx context.Context) (*canonTipos, error) {
	rs, err := c.leer(ctx, grafoTiposSQL)
	if err != nil {
		return nil, err
	}
	ct := &canonTipos{c: c, tipos: make(map[uint32]*tipoCanon, len(rs))}
	rels := map[uint32]*tipoCanon{}
	for _, row := range rs {
		var t tipoCanon
		if json.Unmarshal([]byte(row), &t) != nil || t.OID == 0 || t.Nombre == "" || t.Esquema == "" || ct.tipos[t.OID] != nil {
			return nil, errCaptura
		}
		ct.tipos[t.OID] = &t
		if t.Clase == "c" && t.Relacion != 0 {
			rels[t.Relacion] = &t
		}
		// Shell/pseudo types do not have an output function and cannot contain
		// stored values. All defined outputs and typmod outputs must be builtins.
		if (t.Clase != "p" && !t.SalidaSegura) || !t.ModificadorSeguro ||
			(t.OID >= 16384 && t.Clase == "b" && !t.ArraySeguro) {
			ct.noDeparse = true
		}
	}
	rs, err = c.leer(ctx, camposTiposSQL)
	if err != nil {
		return nil, err
	}
	for _, row := range rs {
		var f campoCanon
		if json.Unmarshal([]byte(row), &f) != nil || f.Posicion <= 0 || f.Nombre == "" || f.Tipo == 0 || rels[f.Relacion] == nil {
			return nil, errCaptura
		}
		t := rels[f.Relacion]
		t.campos = append(t.campos, f)
	}
	for _, t := range rels {
		sort.Slice(t.campos, func(i, j int) bool { return t.campos[i].Posicion < t.campos[j].Posicion })
		for i, f := range t.campos {
			if i > 0 && t.campos[i-1].Posicion == f.Posicion {
				return nil, errCaptura
			}
		}
	}
	return ct, nil
}

const maxProfundidadCanon = 32
const maxNodosCanon = 4096
const maxSQLCanon = 1 << 20

type compilacionCanon struct {
	tipos               *canonTipos
	activos             map[uint32]bool
	nodos, bytes, alias int
	reservados          map[string]bool
}

// canonColumna returns the complete legacy column frame. A false admission
// never returns an expression that the caller could accidentally execute.
func (t *canonTipos) canonColumna(esquema, tabla, columna string, oid uint32, generada string) (string, bool, error) {
	if generada == "v" {
		return "", false, nil
	}
	tipo := t.tipos[oid]
	if tipo == nil {
		return "", false, nil
	}
	oidLO := t.c != nil && t.c.l != nil && t.c.l.limites.ObjetosGrandesSemanticos && t.c.referenciaDeclarada(esquema, tabla, columna)
	c := &compilacionCanon{tipos: t, activos: map[uint32]bool{}, reservados: map[string]bool{columna: true, tabla: true}}
	v, ok, err := c.valor(oid, pgx.Identifier{columna}.Sanitize(), 0, oidLO)
	if err != nil || !ok {
		return "", false, err
	}
	nombreTipo := tipo.Nombre
	if tipo.Esquema != "pg_catalog" {
		nombreTipo = pgx.Identifier{tipo.Esquema, tipo.Nombre}.Sanitize()
	}
	frame := `jsonb_build_array(` + literal(columna) + `,` + literal(nombreTipo) + `,` + v + `)`
	if len(frame) > maxSQLCanon {
		return "", false, errLimite
	}
	return frame, true, nil
}

func (c *compilacionCanon) acotar(s string) (string, bool, error) {
	c.bytes += len(s)
	if c.bytes > maxSQLCanon {
		return "", false, errLimite
	}
	return s, true, nil
}

func (c *compilacionCanon) valor(oid uint32, v string, depth int, oidLO bool) (string, bool, error) {
	c.nodos++
	if depth > maxProfundidadCanon || c.nodos > maxNodosCanon {
		return "", false, errLimite
	}
	t := c.tipos.tipos[oid]
	if t == nil || c.activos[oid] || !t.SalidaSegura || !t.ModificadorSeguro {
		return "", false, nil
	}
	c.activos[oid] = true
	defer delete(c.activos, oid)
	if t.Esquema == "pg_catalog" && t.OID < 16384 && t.Clase == "b" && (tipos[t.Nombre] || (t.Nombre == "oid" && oidLO && depth == 0)) {
		return c.acotar(v + `::text`)
	}
	tag := literal(t.Esquema) + `,` + literal(t.Nombre)
	nullGuard := func(body string) string {
		return `CASE WHEN pg_catalog.pg_column_size(` + v + `) IS NULL THEN NULL ELSE ` + body + ` END`
	}
	switch t.Clase {
	case "e":
		return c.acotar(nullGuard(`pg_catalog.jsonb_build_array('enum',` + tag + `,pg_catalog.enum_out(` + v + `)::text)`))
	case "d":
		base := c.tipos.tipos[t.Base]
		if base == nil {
			return "", false, nil
		}
		inner, ok, err := c.valor(t.Base, `(`+v+`)::`+pgx.Identifier{base.Esquema, base.Nombre}.Sanitize(), depth+1, false)
		if !ok || err != nil {
			return "", false, err
		}
		return c.acotar(nullGuard(`pg_catalog.jsonb_build_array('domain',` + tag + `,` + inner + `)`))
	case "c":
		if t.Relacion == 0 {
			return "", false, nil
		}
		fields := make([]string, 0, len(t.campos))
		for _, f := range t.campos {
			if f.Generada == "v" {
				return "", false, nil
			}
			ft := c.tipos.tipos[f.Tipo]
			if ft == nil {
				return "", false, nil
			}
			inner, ok, err := c.valor(f.Tipo, `(`+v+`).`+pgx.Identifier{f.Nombre}.Sanitize(), depth+1, false)
			if !ok || err != nil {
				return "", false, err
			}
			fields = append(fields, `pg_catalog.jsonb_build_array(`+literal(f.Nombre)+`,`+literal(ft.Esquema)+`,`+literal(ft.Nombre)+`,`+inner+`)`)
		}
		// No IS NULL record predicate: a non-null record of all null fields
		// has a different canonical value from a null composite datum.
		return c.acotar(nullGuard(`pg_catalog.jsonb_build_array('composite',` + tag + `,pg_catalog.jsonb_build_array(` + strings.Join(fields, ",") + `))`))
	case "b":
		if t.Elemento == 0 || t.Categoria != "A" || !t.ArraySeguro {
			return "", false, nil
		}
		branches := []string{`WHEN 0 THEN '[]'::pg_catalog.jsonb`}
		for dims := 1; dims <= 6; dims++ {
			from := []string{}
			order := []string{}
			element := `(` + v + `)`
			for dimension := 1; dimension <= dims; dimension++ {
				var a string
				for {
					c.alias++
					a = fmt.Sprintf("vec_cs06_a%d", c.alias)
					if !c.reservados[a] {
						break
					}
				}
				idx := a + `.i`
				from = append(from, fmt.Sprintf(`pg_catalog.generate_subscripts(%s,%d) AS %s(i)`, v, dimension, a))
				order = append(order, idx)
				element += `[` + idx + `]`
			}
			inner, ok, err := c.valor(t.Elemento, element, depth+1, false)
			if !ok || err != nil {
				return "", false, err
			}
			branches = append(branches, fmt.Sprintf(`WHEN %d THEN (SELECT pg_catalog.jsonb_agg(%s ORDER BY %s) FROM %s)`, dims, inner, strings.Join(order, ","), strings.Join(from, " CROSS JOIN ")))
		}
		shape := `coalesce(pg_catalog.array_ndims(` + v + `),0)`
		// Arrays have at most six dimensions in PG18. array_dims preserves
		// lower/upper bounds; explicit ordered subscripts preserve nulls and
		// composite elements without calling array_out or record_out.
		body := `pg_catalog.jsonb_build_array('array',` + tag + `,` + shape + `,pg_catalog.array_dims(` + v + `),CASE ` + shape + ` ` + strings.Join(branches, " ") + ` END)`
		return c.acotar(nullGuard(body))
	}
	return "", false, nil
}
