package contrastecopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	domain "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
	ports "vec-diputacion-granada/internal/modules/administracion/ports/contrastecopias"
)

// FuenteBaseNoConectable es una autoridad de infraestructura. Antes de responder
// debe observar, autenticar y verificar el material lógico conservado del initdb
// y su inmutabilidad para la misma base/runtime. El lector no acepta una bandera
// de configuración, no activa la base y no fabrica huellas de fábrica.
type FuenteBaseNoConectable interface {
	CapturarBase(context.Context, SolicitudBaseNoConectable) (EvidenciaBaseNoConectable, error)
}
type SolicitudBaseNoConectable struct {
	Nombre, VersionPostgreSQL, PropiedadesSHA256, SelloExclusion string
	MaxFilas, MaxBytes                                           int64
	MaxObjetos                                                   int
}
type EvidenciaBaseNoConectable struct {
	Snapshot                                                                        domain.Snapshot
	Nombre, PropiedadesSHA256, SelloExclusion, InicializacionSHA256, SnapshotSHA256 string
	FilasLeidas, BytesLeidos                                                        int64
}
type presupuestoCaptura struct {
	bytes, filas int64
	objetos      int
}
type baseObservada struct {
	Nombre      string          `json:"nombre"`
	Conectable  bool            `json:"conectable"`
	Propiedades json.RawMessage `json:"propiedades"`
}
type baseCapturada struct {
	base, propiedades string
	snapshot          domain.Snapshot
}

func contieneBase(bases []string, base string) bool {
	for _, v := range bases {
		if v == base {
			return true
		}
	}
	return false
}
func validarBases(c Configuracion) error {
	if len(c.BasesInventariadas) > c.MaxObjetos {
		return errors.New("configuracion_contraste_postgresql_no_admitida")
	}
	seen := map[string]bool{}
	for _, base := range c.BasesInventariadas {
		if !baseAdmitida.MatchString(base) || seen[base] {
			return errors.New("configuracion_contraste_postgresql_no_admitida")
		}
		seen[base] = true
	}
	return nil
}
func shaBytes(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func (c *captura) consultaAgregado(q consulta) string {
	if c.baseMetadatos != "" && (q.clase == "esquema" || q.clase == "acl") {
		return strings.Replace(q.sql, "d.datname=current_database()", "d.datname="+literal(c.baseMetadatos), 1)
	}
	return q.sql
}

// Propiedades y ACL de TODAS las bases: también plantilla y no conectable. Los
// identificadores físicos, xid y counters de almacenamiento no son su semántica.
const basesSQL = `SELECT jsonb_build_object('nombre',d.datname,'conectable',d.datallowconn,'propiedades',jsonb_build_array(d.datname,pg_get_userbyid(d.datdba),d.datistemplate,d.datallowconn,d.datconnlimit,pg_encoding_to_char(d.encoding),d.datlocprovider,d.datcollate,d.datctype,d.datlocale,d.daticurules,d.datcollversion,pg_database_collation_actual_version(d.oid),t.spcname,shobj_description(d.oid,'pg_database'),coalesce((SELECT jsonb_agg(jsonb_build_array(pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable) ORDER BY pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable) FROM aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a),'[]'::jsonb)))::text FROM pg_database d JOIN pg_tablespace t ON t.oid=d.dattablespace`

func (l *Lector) leerBases(ctx context.Context, tx consultaSQL, p *presupuestoCaptura) ([]baseObservada, error) {
	c := &captura{tx: tx, l: l, presupuesto: p, bytes: p.bytes, filas: p.filas}
	rows, e := c.leer(ctx, basesSQL)
	p.bytes = c.bytes
	p.filas = c.filas
	if e != nil {
		return nil, e
	}
	out := make([]baseObservada, 0, len(rows))
	for _, row := range rows {
		var base baseObservada
		if json.Unmarshal([]byte(row), &base) != nil || base.Nombre == "" || len(base.Propiedades) == 0 {
			return nil, errCaptura
		}
		out = append(out, base)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Nombre < out[j].Nombre })
	return out, nil
}
func (l *Lector) coincideAmbito(bases []baseObservada) bool {
	if len(bases) != len(l.limites.BasesInventariadas) {
		return false
	}
	for i, b := range bases {
		if b.Nombre != l.limites.BasesInventariadas[i] {
			return false
		}
	}
	return true
}
func observarSello(ctx context.Context, guard ports.ExclusionObservada, esperado string) (string, error) {
	s, e := guard.ComprobarExclusion(ctx)
	if e != nil || !selloValido.MatchString(s) || esperado != "" && s != esperado {
		return "", errCaptura
	}
	return s, nil
}

func (l *Lector) capturarBases(ctx context.Context, exec ports.EjecutorPostgreSQL, base string, guard ports.ExclusionObservada, fuente FuenteBaseNoConectable) (domain.Snapshot, error) {
	incomplete := domain.Snapshot{Version: 1, PostgreSQL: l.limites.VersionPostgreSQL, Completo: false, Objetos: []domain.Objeto{}, Motivos: []string{"bases_no_inventariadas"}}
	if !contieneBase(l.limites.BasesInventariadas, base) {
		return incomplete, nil
	}
	sello, e := observarSello(ctx, guard, "")
	if e != nil {
		return domain.Snapshot{}, e
	}
	p := &presupuestoCaptura{objetos: 8}
	tx := &transporteEjecutor{exec: exec, base: base, maxBytes: l.limites.MaxBytes, timeout: l.limites.TiempoMaximo.Milliseconds()}
	before, e := l.leerBases(ctx, tx, p)
	if e != nil {
		return domain.Snapshot{}, e
	}
	if !l.coincideAmbito(before) {
		return incomplete, nil
	}
	capturadas := make([]baseCapturada, 0, len(before))
	for _, b := range before {
		if _, e = observarSello(ctx, guard, sello); e != nil {
			return domain.Snapshot{}, e
		}
		props := shaBytes(b.Propiedades)
		var s domain.Snapshot
		if !b.Conectable {
			s, e = l.capturarNoConectable(ctx, b.Nombre, props, sello, p, fuente)
			if e != nil {
				return domain.Snapshot{}, e
			}
		} else {
			cfg := l.limites
			cfg.BasesInventariadas = nil
			cfg.ReferenciasObjetosGrandes = nil
			for _, r := range l.limites.ReferenciasObjetosGrandes {
				if r.Base == b.Nombre {
					r.Base = ""
					cfg.ReferenciasObjetosGrandes = append(cfg.ReferenciasObjetosGrandes, r)
				}
			}
			child := &Lector{limites: cfg}
			reader := &transporteEjecutor{exec: exec, base: b.Nombre, maxBytes: l.limites.MaxBytes, timeout: l.limites.TiempoMaximo.Milliseconds()}
			s, e = child.capturarSQLAcotada(ctx, reader, guard, true, p, "")
			if e != nil {
				return domain.Snapshot{}, e
			}
		}
		capturadas = append(capturadas, baseCapturada{b.Nombre, props, s})
	}
	after, e := l.leerBases(ctx, tx, p)
	if e != nil {
		return domain.Snapshot{}, e
	}
	if _, e = observarSello(ctx, guard, sello); e != nil {
		return domain.Snapshot{}, e
	}
	same := l.coincideAmbito(after)
	if same {
		a, _ := json.Marshal(before)
		b, _ := json.Marshal(after)
		same = string(a) == string(b)
	}
	s, e := l.combinarBases(capturadas)
	if e != nil {
		return domain.Snapshot{}, e
	}
	if !same {
		s.Completo = false
		s.Motivos = append(s.Motivos, "bases_inestables")
	}
	return s, nil
}

func (l *Lector) capturarNoConectable(ctx context.Context, base, props, sello string, p *presupuestoCaptura, fuente FuenteBaseNoConectable) (domain.Snapshot, error) {
	unknown := domain.Snapshot{Version: 1, PostgreSQL: l.limites.VersionPostgreSQL, Completo: false, Objetos: []domain.Objeto{}, Motivos: []string{"base_no_conectable_sin_evidencia"}}
	if fuente == nil {
		return unknown, nil
	}
	req := SolicitudBaseNoConectable{Nombre: base, VersionPostgreSQL: l.limites.VersionPostgreSQL, PropiedadesSHA256: props, SelloExclusion: sello, MaxFilas: l.limites.MaxFilas - p.filas, MaxBytes: l.limites.MaxBytes - p.bytes, MaxObjetos: l.limites.MaxObjetos - p.objetos + 8}
	evidence, e := fuente.CapturarBase(ctx, req)
	if e != nil {
		return unknown, nil
	}
	if evidence.Nombre != base || evidence.PropiedadesSHA256 != props || evidence.SelloExclusion != sello || !selloValido.MatchString(evidence.InicializacionSHA256) || evidence.Snapshot.PostgreSQL != l.limites.VersionPostgreSQL || len(domain.Validar(evidence.Snapshot)) != 0 {
		return unknown, nil
	}
	encoded, e := json.Marshal(evidence.Snapshot)
	if e != nil {
		return domain.Snapshot{}, errCaptura
	}
	if len(encoded) > 32<<20 {
		return domain.Snapshot{}, errLimite
	}
	if evidence.SnapshotSHA256 != shaBytes(encoded) {
		return unknown, nil
	}
	if evidence.BytesLeidos < int64(len(encoded)) || evidence.FilasLeidas < 1 || evidence.BytesLeidos > req.MaxBytes || evidence.FilasLeidas > req.MaxFilas {
		return domain.Snapshot{}, errLimite
	}
	p.bytes += evidence.BytesLeidos
	p.filas += evidence.FilasLeidas
	for _, o := range evidence.Snapshot.Objetos {
		if o.Clave != "inventario" {
			if p.objetos >= l.limites.MaxObjetos {
				return domain.Snapshot{}, errLimite
			}
			p.objetos++
		}
	}
	return evidence.Snapshot, nil
}

func (l *Lector) combinarBases(bases []baseCapturada) (domain.Snapshot, error) {
	s := domain.Snapshot{Version: 1, PostgreSQL: l.limites.VersionPostgreSQL, Completo: true, Objetos: []domain.Objeto{}, Motivos: []string{}}
	globals := map[string][]string{}
	cantidades := map[string]int64{}
	sort.Slice(bases, func(i, j int) bool { return bases[i].base < bases[j].base })
	motivos := map[string]bool{}
	for _, b := range bases {
		if !b.snapshot.Completo {
			s.Completo = false
		}
		for _, m := range b.snapshot.Motivos {
			motivos[m] = true
		}
		for _, o := range b.snapshot.Objetos {
			if o.Clave == "inventario" {
				if o.Clase == "tablas" || o.Clase == "secuencias" || o.Clase == "objetos_grandes" {
					continue
				}
				v, _ := json.Marshal(struct {
					Base   string
					Objeto domain.Objeto
				}{b.base, o})
				globals[o.Clase] = append(globals[o.Clase], string(v))
				cantidades[o.Clase] += o.Cantidad
			} else {
				o.Clave = pgx.Identifier{b.base}.Sanitize() + "." + o.Clave
				if len(o.Clave) > 1024 {
					return domain.Snapshot{}, errLimite
				}
				s.Objetos = append(s.Objetos, o)
			}
		}
		// Incluye las propiedades/ACL observadas de toda base, incluso sin evidencia
		// de contenido. Así la ausencia no borra propietarios ni permiso de conexión.
		meta, _ := json.Marshal([]string{b.base, b.propiedades})
		globals["esquema"] = append(globals["esquema"], string(meta))
		cantidades["esquema"]++
	}
	for _, clase := range []string{"esquema", "roles", "acl", "extensiones", "privilegios_defecto"} {
		s.Objetos = append(s.Objetos, domain.Objeto{Clase: clase, Clave: "inventario", Cantidad: cantidades[clase], SHA256: huella(globals[clase])})
	}
	for _, clase := range []string{"tablas", "secuencias", "objetos_grandes"} {
		s.Objetos = append(s.Objetos, domain.Resumir(clase, s.Objetos))
	}
	for m := range motivos {
		s.Motivos = append(s.Motivos, m)
	}
	sort.Strings(s.Motivos)
	sort.Slice(s.Objetos, func(i, j int) bool {
		a, b := s.Objetos[i], s.Objetos[j]
		if a.Clase != b.Clase {
			return a.Clase < b.Clase
		}
		return a.Clave < b.Clave
	})
	if len(s.Objetos) > l.limites.MaxObjetos {
		return domain.Snapshot{}, errLimite
	}
	encoded, e := json.Marshal(s)
	if e != nil {
		return domain.Snapshot{}, errCaptura
	}
	if len(encoded) > 32<<20 {
		return domain.Snapshot{}, errLimite
	}
	return s, nil
}
