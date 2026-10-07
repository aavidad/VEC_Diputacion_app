package contrastecopias

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	domain "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

func validarReferencias(config Configuracion) error {
	bad := errors.New("configuracion_contraste_postgresql_no_admitida")
	if (!config.ObjetosGrandesSemanticos && len(config.ReferenciasObjetosGrandes) != 0) || len(config.ReferenciasObjetosGrandes) > config.MaxObjetos {
		return bad
	}
	seen := map[ReferenciaObjetoGrande]bool{}
	for _, r := range config.ReferenciasObjetosGrandes {
		if len(config.BasesInventariadas) == 0 {
			if r.Base != "" {
				return bad
			}
		} else if !contieneBase(config.BasesInventariadas, r.Base) {
			return bad
		}
		for _, name := range []string{r.Esquema, r.Tabla, r.Columna} {
			if name == "" || len(name) > 63 || !utf8.ValidString(name) || strings.ContainsRune(name, 0) {
				return bad
			}
		}
		if seen[r] {
			return bad
		}
		seen[r] = true
	}
	return nil
}

// La comprobación genérica conserva el cierre original salvo oid expresamente
// habilitado. La admisión de cada columna oid se comprueba aparte por identidad.
func consultaComprobacion(check comprobacion, lo bool) string {
	if lo && check.motivo == "objetos_grandes_sin_vinculos_comprobables" {
		return `SELECT false`
	}
	if lo && check.motivo == "tipo_columna_no_admitido" {
		return strings.Replace(check.sql, "'bool','int2'", "'oid','bool','int2'", 1)
	}
	return check.sql
}
func (c *captura) referenciaDeclarada(esquema, tabla, columna string) bool {
	for _, r := range c.l.limites.ReferenciasObjetosGrandes {
		if r.Esquema == esquema && r.Tabla == tabla && r.Columna == columna {
			return true
		}
	}
	return false
}
func (c *captura) validarReferencias(ctx context.Context) error {
	if !c.l.limites.ObjetosGrandesSemanticos {
		return nil
	}
	for _, r := range c.l.limites.ReferenciasObjetosGrandes {
		var found bool
		if e := c.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_attribute a JOIN pg_class r ON r.oid=a.attrelid JOIN pg_namespace n ON n.oid=r.relnamespace JOIN pg_type t ON t.oid=a.atttypid JOIN pg_namespace tn ON tn.oid=t.typnamespace JOIN pg_am am ON am.oid=r.relam WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema' AND n.nspname=$1 AND r.relname=$2 AND a.attname=$3 AND a.attnum>0 AND NOT a.attisdropped AND a.attgenerated<>'v' AND tn.nspname='pg_catalog' AND t.typname='oid' AND r.relkind='r' AND am.amname='heap')`, r.Esquema, r.Tabla, r.Columna).Scan(&found); e != nil {
			return errCaptura
		}
		if !found {
			c.motivo("referencia_objeto_grande_no_admitida")
			continue
		}
		var dangling bool
		col := pgx.Identifier{r.Columna}.Sanitize()
		sql := `SELECT EXISTS(SELECT 1 FROM ` + pgx.Identifier{r.Esquema, r.Tabla}.Sanitize() + ` AS fila WHERE ` + col + ` IS NOT NULL AND ` + col + `<>0::oid AND NOT EXISTS(SELECT 1 FROM pg_largeobject_metadata l WHERE l.oid=fila.` + col + `))`
		if e := c.tx.QueryRow(ctx, sql).Scan(&dangling); e != nil {
			return errCaptura
		}
		if dangling {
			c.motivo("referencia_objeto_grande_ausente")
		}
	}
	return nil
}

const metadataGrandeSQL = `SELECT jsonb_build_array(l.oid::text,pg_get_userbyid(l.lomowner),coalesce((SELECT jsonb_agg(jsonb_build_array(pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable) ORDER BY pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable) FROM aclexplode(coalesce(l.lomacl,acldefault('L',l.lomowner))) a),'[]'::jsonb),coalesce((SELECT max(p.pageno::bigint*(current_setting('block_size')::bigint/4)+octet_length(p.data)) FROM pg_largeobject p WHERE p.loid=l.oid),0)::text)::text FROM pg_largeobject_metadata l`

func nuevoSelloGrande(meta string) hash.Hash {
	h := sha256.New()
	h.Write([]byte("vec:cs06:lo:bytes:v1\n"))
	var frame [8]byte
	binary.BigEndian.PutUint64(frame[:], uint64(len(meta)))
	h.Write(frame[:])
	h.Write([]byte(meta))
	return h
}

// grandesSemanticos conserva el loid que pg_dump restaura como identidad lógica.
// El tamaño consulta páginas solo para acotar; el sello usa exclusivamente lo_get
// y conserva huecos como bytes cero, con independencia de su distribución física.
func (c *captura) grandesSemanticos(ctx context.Context) error {
	var count int64
	if e := c.tx.QueryRow(ctx, `SELECT count(*) FROM pg_largeobject_metadata`).Scan(&count); e != nil {
		return errCaptura
	}
	if count > int64(c.l.limites.MaxObjetos) {
		return errLimite
	}
	rows, e := c.leer(ctx, metadataGrandeSQL)
	if e != nil {
		return e
	}
	for _, meta := range rows {
		var fields []json.RawMessage
		if json.Unmarshal([]byte(meta), &fields) != nil || len(fields) != 4 {
			return errCaptura
		}
		var idText, sizeText string
		if json.Unmarshal(fields[0], &idText) != nil || json.Unmarshal(fields[3], &sizeText) != nil {
			return errCaptura
		}
		id, e := strconv.ParseUint(idText, 10, 32)
		if e != nil {
			return errCaptura
		}
		size, e := strconv.ParseInt(sizeText, 10, 64)
		if e != nil || size < 0 {
			return errCaptura
		}
		// Hex transfiere dos bytes por cada byte lógico; se reserva el presupuesto
		// antes de llamar lo_get, incluidos huecos que todavía no ocupan páginas.
		if size > (c.l.limites.MaxBytes-c.bytes)/2 {
			return errLimite
		}
		h := nuevoSelloGrande(meta)
		for offset := int64(0); offset < size; {
			remaining := c.l.limites.MaxBytes - c.bytes
			chunk := int64(64 << 10)
			if chunk > size-offset {
				chunk = size - offset
			}
			if chunk > (remaining-16)/2 {
				chunk = (remaining - 16) / 2
			}
			if chunk < 1 {
				return errLimite
			}
			rs, e := c.leer(ctx, `SELECT to_json(encode(lo_get($1::oid,$2::bigint,$3::int),'hex'))::text`, uint32(id), offset, chunk)
			if e != nil {
				return e
			}
			if len(rs) != 1 {
				return errCaptura
			}
			var encoded string
			if json.Unmarshal([]byte(rs[0]), &encoded) != nil {
				return errCaptura
			}
			b, e := hex.DecodeString(encoded)
			if e != nil || int64(len(b)) != chunk {
				return errCaptura
			}
			h.Write(b)
			offset += chunk
		}
		if e = c.agregar(domain.Objeto{Clase: "objetos_grandes", Clave: "loid:" + idText, Cantidad: size, SHA256: hex.EncodeToString(h.Sum(nil))}); e != nil {
			return e
		}
	}
	return nil
}
