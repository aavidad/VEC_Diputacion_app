package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const MaximosCandidatosRaizIncorporacionV2 = 16
const MaximoBytesRaicesIncorporacionV2 = 32 << 20
const consultaRaicesIncorporacionV2 = `SELECT seguimiento_ref,organizacion_ref,expediente_ref,relacion_ref,version_expediente_observada,definicion_ref,definicion_version,definicion_sha256,publicacion_json,raiz_canonica,raiz_sha256,estado_inicial_json,estado_inicial_canonico FROM vec_contratacion_temporal.leer_raices_incorporacion_ejercicio_v2($1::text,$2::text,$3::text)`
const ajustesRaicesIncorporacionV2 = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','5s',true),set_config('idle_in_transaction_session_timeout','10s',true)`

var ErrResolucionRaizIncorporacionV2 = errors.New("contratacion temporal: raiz durable no disponible")

type iniciadorRaicesIncorporacionV2 interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// ResolverRaizIncorporacionV2PostgreSQL requiere pool dedicado al LOGIN simple
// de vec_contratacion_temporal_lector_raices_historicas. No es CAS, permiso ni
// publicación de raíz. No tiene configuración opcional, caché o selección latest.
type ResolverRaizIncorporacionV2PostgreSQL struct {
	pool  iniciadorRaicesIncorporacionV2
	reloj ct.Reloj
}

var _ ResolverRaizIncorporacionV2 = (*ResolverRaizIncorporacionV2PostgreSQL)(nil)

func NuevoResolverRaizIncorporacionV2PostgreSQL(p *pgxpool.Pool, r ct.Reloj) (*ResolverRaizIncorporacionV2PostgreSQL, error) {
	return nuevoResolverRaizIncorporacionV2(p, r)
}
func nuevoResolverRaizIncorporacionV2(p iniciadorRaicesIncorporacionV2, r ct.Reloj) (*ResolverRaizIncorporacionV2PostgreSQL, error) {
	if nuloLectorHistoriaV2(p) || nuloLectorHistoriaV2(r) {
		return nil, ErrResolucionRaizIncorporacionV2
	}
	return &ResolverRaizIncorporacionV2PostgreSQL{p, r}, nil
}
func errorRaicesV2(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ErrResolucionRaizIncorporacionV2
}

type filaRaizIncorporacionV2 struct {
	referencia, org, exp, rel, versionExp, defRef, defVersion, defHash string
	publicacion, raiz                                                  []byte
	raizHash                                                           string
	estado, canon                                                      []byte
}

func (f filaRaizIncorporacionV2) bytes() int {
	return len(f.referencia) + len(f.org) + len(f.exp) + len(f.rel) + len(f.versionExp) + len(f.defRef) + len(f.defVersion) + len(f.defHash) + len(f.publicacion) + len(f.raiz) + len(f.raizHash) + len(f.estado) + len(f.canon)
}
func (f filaRaizIncorporacionV2) clonar() filaRaizIncorporacionV2 {
	f.publicacion = bytes.Clone(f.publicacion)
	f.raiz = bytes.Clone(f.raiz)
	f.estado = bytes.Clone(f.estado)
	f.canon = bytes.Clone(f.canon)
	return f
}
func decimalRaicesV2(s string) (uint64, error) {
	if len(s) == 0 || len(s) > 20 {
		return 0, ErrResolucionRaizIncorporacionV2
	}
	n, e := strconv.ParseUint(s, 10, 64)
	// Ambas columnas de CT72/78 tienen el rango entero seguro 1..2^53-1.
	if e != nil || n == 0 || n > 9007199254740991 || strconv.FormatUint(n, 10) != s {
		return 0, ErrResolucionRaizIncorporacionV2
	}
	return n, nil
}
func shaRaicesV2(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// Vigencia.Hasta admite exclusivamente el cero canónico como extremo abierto.
// Derivamos el esquema del DTO público; no relajamos otros time.Time ni
// sustituimos el original por una fecha inventada. El dominio revalida después.
func normalizarPublicacionRaicesV2(arbol any) (any, error) {
	tipo := reflect.TypeOf(dom.PublicacionDefinicionSeguimiento{})
	campos := make([]reflect.StructField, tipo.NumField())
	for i := range campos {
		campos[i] = tipo.Field(i)
		if campos[i].Name == "Vigencia" {
			campos[i].Type = reflect.TypeOf(struct {
				Desde time.Time `json:"desde"`
				Hasta string    `json:"hasta"`
			}{})
		}
	}
	forma, err := normalizarFormaRegistroV2(arbol, reflect.StructOf(campos))
	if err != nil {
		return nil, err
	}
	vigencia := forma.(map[string]any)["vigencia"].(map[string]any)
	hasta := vigencia["hasta"].(string)
	if hasta != "0001-01-01T00:00:00Z" {
		normal, e := normalizarFormaRegistroV2(hasta, reflect.TypeOf(time.Time{}))
		if e != nil {
			return nil, e
		}
		vigencia["hasta"] = normal
	}
	return forma, nil
}

func validarFilaRaicesV2(ctx context.Context, f filaRaizIncorporacionV2, triple [3]string) error {
	if f.org != triple[0] || f.exp != triple[1] || f.rel != triple[2] || !dom.ReferenciaOpacaValida(f.referencia) {
		return ErrResolucionRaizIncorporacionV2
	}
	if _, e := decimalRaicesV2(f.versionExp); e != nil {
		return e
	}
	version, e := decimalRaicesV2(f.defVersion)
	if e != nil {
		return e
	}
	terna := dom.ReferenciaDefinicionSeguimiento{Referencia: f.defRef, Version: version, HuellaSHA256: f.defHash}
	if terna.Validar() != nil || !huellaSnapshotSeguimientoValida(f.raizHash) || len(f.raiz) == 0 || shaRaicesV2(f.raiz) != f.raizHash {
		return ErrResolucionRaizIncorporacionV2
	}
	// Validadores existentes: JSON cerrado (incluye duplicados/alias/null/tipos y
	// UTC microsegundos) y reconstrucción semántica de la publicación original.
	if validarJSONRegistroV2(ctx, f.publicacion) != nil {
		return errorRaicesV2(ctx)
	}
	arbol, e := arbolJSONRegistroV2(f.publicacion)
	if e != nil {
		return ErrResolucionRaizIncorporacionV2
	}
	forma, e := normalizarPublicacionRaicesV2(arbol)
	if e != nil {
		return ErrResolucionRaizIncorporacionV2
	}
	b, e := json.Marshal(forma)
	if e != nil {
		return ErrResolucionRaizIncorporacionV2
	}
	var pub dom.PublicacionDefinicionSeguimiento
	if json.Unmarshal(b, &pub) != nil {
		return ErrResolucionRaizIncorporacionV2
	}
	def, e := dom.RestaurarDefinicionSeguimiento(pub)
	if e != nil || !def.Referencia().Coincide(terna) {
		return ErrResolucionRaizIncorporacionV2
	}
	s, e := RestaurarSeguimientoPersistido(def, ExpectativaSeguimientoPersistido{Referencia: f.referencia, OrganizacionRef: f.org, ExpedienteRef: f.exp, RelacionRef: f.rel, Definicion: terna, Version: 0},
		SnapshotSeguimientoPersistido{EstadoJSON: f.estado, EstadoCanonico: f.canon, HuellaCanonicaSHA256: shaRaicesV2(f.canon)})
	// Rehidratar recalcula la raíz desde la identidad/definición/período/fecha
	// original; su huella liga los bytes recibidos sin inventar otro codec.
	if e != nil || s.Estado().HuellaRaizSHA256 != f.raizHash {
		return ErrResolucionRaizIncorporacionV2
	}
	return ctx.Err()
}

func (l *ResolverRaizIncorporacionV2PostgreSQL) ResolverSeguimientoIncorporacionV2(ctx context.Context, o ct.OrdenConfirmacionIncorporacionV2) (string, error) {
	if ctx == nil || l == nil || nuloLectorHistoriaV2(l.pool) || nuloLectorHistoriaV2(l.reloj) {
		return "", errorRaicesV2(ctx)
	}
	var ultimo time.Time
	validar := func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ahora := l.reloj.Ahora()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !dom.InstanteUTCCanonico(ahora) || ahora.Before(ultimo) || o.ValidarEn(ahora) != nil {
			return ErrResolucionRaizIncorporacionV2
		}
		ultimo = ahora
		return ctx.Err()
	}
	if e := validar(); e != nil {
		return "", e
	}
	d, e := o.Material().Datos()
	if e != nil {
		return "", errorRaicesV2(ctx)
	}
	triple := [3]string{d.Preparacion.OrganizacionRef, d.Confirmacion.SolicitudPersonal.ExpedienteRef, d.Confirmacion.ResultadoPersonal.RelacionRef}
	tx, e := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	confirmado := false
	if !nuloLectorHistoriaV2(tx) {
		defer func() {
			if !confirmado {
				c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = tx.Rollback(c)
			}
		}()
	}
	if e != nil || nuloLectorHistoriaV2(tx) {
		return "", errorRaicesV2(ctx)
	}
	if e = validar(); e != nil {
		return "", e
	}
	if _, e = tx.Exec(ctx, ajustesRaicesIncorporacionV2); e != nil {
		return "", errorRaicesV2(ctx)
	}
	if e = validar(); e != nil {
		return "", e
	}
	filas, e := tx.Query(ctx, consultaRaicesIncorporacionV2, triple[0], triple[1], triple[2])
	if !nuloLectorHistoriaV2(filas) {
		defer filas.Close()
	}
	if e != nil || nuloLectorHistoriaV2(filas) {
		return "", errorRaicesV2(ctx)
	}
	if e = validar(); e != nil {
		return "", e
	}
	var elegida string
	total, n := 0, 0
	for filas.Next() {
		n++
		if n > MaximosCandidatosRaizIncorporacionV2 {
			return "", errorRaicesV2(ctx)
		}
		if e = validar(); e != nil {
			return "", e
		}
		var f filaRaizIncorporacionV2
		if e = filas.Scan(&f.referencia, &f.org, &f.exp, &f.rel, &f.versionExp, &f.defRef, &f.defVersion, &f.defHash, &f.publicacion, &f.raiz, &f.raizHash, &f.estado, &f.canon); e != nil {
			return "", errorRaicesV2(ctx)
		}
		// Presupuesto sobre los trece campos, antes de clonar o decodificar.
		if f.bytes() > MaximoBytesRaicesIncorporacionV2-total {
			return "", errorRaicesV2(ctx)
		}
		total += f.bytes()
		f = f.clonar()
		if e = validar(); e != nil {
			return "", e
		}
		if e = validarFilaRaicesV2(ctx, f, triple); e != nil {
			return "", errorRaicesV2(ctx)
		}
		elegida = f.referencia
	}
	if filas.Err() != nil {
		return "", errorRaicesV2(ctx)
	}
	filas.Close()
	if e = validar(); e != nil {
		return "", e
	}
	if n != 1 {
		return "", errorRaicesV2(ctx)
	}
	if e = tx.Commit(ctx); e != nil {
		return "", errorRaicesV2(ctx)
	}
	confirmado = true
	if e = validar(); e != nil {
		return "", e
	}
	return elegida, nil
}

// LeerPreparacionInicial recupera la única raíz técnica y su estado cero.
// La versión devuelta es la observada al crearla, no la versión actual del
// expediente. Esta lectura no fabrica una orden ni confirma incorporación.
func (l *ResolverRaizIncorporacionV2PostgreSQL) LeerPreparacionInicial(ctx context.Context, org, exp, rel string) (dom.PublicacionDefinicionSeguimiento, dom.EstadoPersistidoSeguimiento, uint64, error) {
	fallar := func() (dom.PublicacionDefinicionSeguimiento, dom.EstadoPersistidoSeguimiento, uint64, error) {
		return dom.PublicacionDefinicionSeguimiento{}, dom.EstadoPersistidoSeguimiento{}, 0, errorRaicesV2(ctx)
	}
	if ctx == nil || l == nil || nuloLectorHistoriaV2(l.pool) {
		return fallar()
	}
	if ctx.Err() != nil || !dom.ReferenciaOpacaValida(org) || !dom.ReferenciaOpacaValida(exp) || !dom.ReferenciaOpacaValida(rel) {
		return fallar()
	}
	triple := [3]string{org, exp, rel}
	tx, e := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	confirmado := false
	if !nuloLectorHistoriaV2(tx) {
		defer func() {
			if !confirmado {
				c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = tx.Rollback(c)
			}
		}()
	}
	if e != nil || nuloLectorHistoriaV2(tx) || ctx.Err() != nil {
		return fallar()
	}
	if _, e = tx.Exec(ctx, ajustesRaicesIncorporacionV2); e != nil || ctx.Err() != nil {
		return fallar()
	}
	filas, e := tx.Query(ctx, consultaRaicesIncorporacionV2, org, exp, rel)
	if !nuloLectorHistoriaV2(filas) {
		defer filas.Close()
	}
	if e != nil || nuloLectorHistoriaV2(filas) || ctx.Err() != nil {
		return fallar()
	}
	var elegida filaRaizIncorporacionV2
	total, n := 0, 0
	for filas.Next() {
		n++
		if ctx.Err() != nil || n > MaximosCandidatosRaizIncorporacionV2 {
			return fallar()
		}
		var f filaRaizIncorporacionV2
		if e = filas.Scan(&f.referencia, &f.org, &f.exp, &f.rel, &f.versionExp, &f.defRef, &f.defVersion, &f.defHash, &f.publicacion, &f.raiz, &f.raizHash, &f.estado, &f.canon); e != nil || ctx.Err() != nil {
			return fallar()
		}
		if f.bytes() > MaximoBytesRaicesIncorporacionV2-total {
			return fallar()
		}
		total += f.bytes()
		f = f.clonar()
		if validarFilaRaicesV2(ctx, f, triple) != nil {
			return fallar()
		}
		elegida = f
	}
	if filas.Err() != nil || ctx.Err() != nil || n != 1 {
		return fallar()
	}
	filas.Close()
	// Los bytes propios ya han pasado la validación cerrada, la restauración
	// de dominio y el cotejo del canon de estado cero y la huella de raíz.
	var publicacion dom.PublicacionDefinicionSeguimiento
	var estado dom.EstadoPersistidoSeguimiento
	if json.Unmarshal(elegida.publicacion, &publicacion) != nil || json.Unmarshal(elegida.estado, &estado) != nil {
		return fallar()
	}
	version, e := decimalRaicesV2(elegida.versionExp)
	if e != nil || ctx.Err() != nil {
		return fallar()
	}
	if e = tx.Commit(ctx); e != nil {
		return fallar()
	}
	confirmado = true
	if ctx.Err() != nil {
		return fallar()
	}
	return publicacion, estado, version, nil
}
