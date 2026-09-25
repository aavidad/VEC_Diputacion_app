package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/app/bootstrap"
)

const (
	errGobiernoConexion = errorPropio("gobierno V3: conexión de solo lectura no disponible o rol sin lectura")
	errClaveBaseNoPub   = errorPropio("clave base: no coincide con ninguna clave base CT publicada (material privado divergente)")
	errEmisorDistinto   = errorPropio("clave base: el emisor publicado no coincide con el de ct_v3.json")
	errDerivacion       = errorPropio("claves B2: derivación rechazada")
	errClaveB2          = errorPropio("claves B2: lo derivado no coincide con el gobierno publicado vigente")
	errRaiz             = errorPropio("gobierno V3: la raíz vigente no coincide con la de ct_v3.json o no está vigente")

	// audienciaClaveBaseCT es la audiencia de la clave base del publicador
	// de desarrollo (vec-server); de ella derivan todas las capacidades.
	audienciaClaveBaseCT = "vec_contratacion_temporal.confirmar_alta_atestada.v1"
	prefijoClaveBaseCT   = "clave:capacidad:ct:"
	rolPropietarioAD3    = "vec_autorizacion_atestada_v3_propietario"
)

var errSinFila = errors.New("sin fila")

// filaClave son coordenadas públicas de una clave de capacidad publicada y
// los hechos de vigencia calculados en la misma instantánea. Nunca incluye
// el secreto.
type filaClave struct {
	ClaveID, HuellaGobierno, EmisorID, Audiencia string
	Version, Revision                            uint64
	Desde, Hasta                                 time.Time
	ActoPropio, Vigente, Revocada                bool
	DentroCheckpoint, PunteroVigente             bool
}

type filaRaiz struct {
	ClaveID, Audiencia string
	Version            uint64
	Vigente            bool
}

// fuenteGobierno es una instantánea de solo lectura del gobierno V3.
type fuenteGobierno interface {
	clavePorHuellaSecreto(context.Context, string) (filaClave, error)
	raizVigente(context.Context) (filaRaiz, error)
	cerrar()
}

// datosCT son las coordenadas no secretas que aporta ct_v3.json.
type datosCT struct {
	catalogo, emisor, raizID, audiencia string
	raizVersion                         uint64
}

// capacidadCotejada reúne las coordenadas publicadas de una capacidad B2 y
// su secreto derivado, que solo vive hasta escribirse en la salida.
type capacidadCotejada struct {
	bootstrap.CapacidadPublicadaPersonalB2V3
	secreto []byte
}

func borrarCapacidades(c *[8]capacidadCotejada) {
	for i := range c {
		clear(c[i].secreto)
		c[i].secreto = nil
	}
}

// cotejar localiza la clave base publicada por la huella de su secreto,
// deriva las ocho claves B2 con la derivación única del publicador y exige
// que cada una esté publicada, vigente, sin revocar, apuntada como clave de
// emisión de su audiencia y dentro del checkpoint. La raíz vigente debe ser
// la misma que usa ct_v3.json.
func cotejar(ctx context.Context, g fuenteGobierno, ct datosCT, base []byte) ([8]capacidadCotejada, error) {
	var salida [8]capacidadCotejada
	huella := sha256.Sum256(base)
	fb, err := g.clavePorHuellaSecreto(ctx, hex.EncodeToString(huella[:]))
	if errors.Is(err, errSinFila) {
		return salida, errClaveBaseNoPub
	}
	if err != nil {
		return salida, errGobiernoConexion
	}
	if fb.Audiencia != audienciaClaveBaseCT || !strings.HasPrefix(fb.ClaveID, prefijoClaveBaseCT) || !fb.ActoPropio || fb.Revocada {
		return salida, errClaveBaseNoPub
	}
	if fb.EmisorID != ct.emisor {
		return salida, errEmisorDistinto
	}
	claves, err := bootstrap.DerivarClavesPersonalB2V3Desarrollo(base, fb.ClaveID, fb.EmisorID, fb.Desde, fb.Hasta)
	defer func() {
		for i := range claves {
			claves[i].Borrar()
		}
	}()
	if err != nil {
		return salida, errDerivacion
	}
	for i := range claves {
		d := claves[i]
		f, err := g.clavePorHuellaSecreto(ctx, d.SHA256)
		if err != nil && !errors.Is(err, errSinFila) {
			borrarCapacidades(&salida)
			return [8]capacidadCotejada{}, errGobiernoConexion
		}
		if err != nil || f.ClaveID != d.ClaveID || f.Audiencia != d.Audiencia || f.HuellaGobierno != d.HuellaGobierno ||
			f.EmisorID != d.EmisorID || !f.Desde.Equal(d.Desde) || !f.Hasta.Equal(d.Hasta) || f.Version == 0 || f.Revision == 0 ||
			!f.ActoPropio || !f.Vigente || f.Revocada || !f.DentroCheckpoint || !f.PunteroVigente {
			borrarCapacidades(&salida)
			return [8]capacidadCotejada{}, errClaveB2
		}
		publicada := d.CapacidadPublicadaPersonalB2V3
		publicada.Version, publicada.RevisionGobierno = f.Version, f.Revision
		salida[i] = capacidadCotejada{CapacidadPublicadaPersonalB2V3: publicada, secreto: claves[i].CopiarSecreto()}
		comprobada := sha256.Sum256(salida[i].secreto)
		if hex.EncodeToString(comprobada[:]) != publicada.SHA256 {
			borrarCapacidades(&salida)
			return [8]capacidadCotejada{}, errDerivacion
		}
	}
	r, err := g.raizVigente(ctx)
	if err != nil || !r.Vigente || r.ClaveID != ct.raizID || r.Version != ct.raizVersion || r.Audiencia != ct.audiencia {
		borrarCapacidades(&salida)
		return [8]capacidadCotejada{}, errRaiz
	}
	return salida, nil
}

// gobiernoPostgreSQL mantiene una transacción REPEATABLE READ READ ONLY: una
// sola instantánea para todas las lecturas y ninguna escritura posible. Las
// tablas de gobierno tienen RLS forzada para el propietario AD3, así que el
// LOGIN debe poder asumir ese rol (SET ROLE, como el publicador de
// vec-server). Ninguna consulta menciona la columna secreto_hmac.
type gobiernoPostgreSQL struct {
	con *pgx.Conn
	tx  pgx.Tx
}

func abrirGobiernoPostgreSQL(ctx context.Context, dsn string) (fuenteGobierno, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, errGobiernoConexion
	}
	if cfg.ConnectTimeout == 0 || cfg.ConnectTimeout > 15*time.Second {
		cfg.ConnectTimeout = 15 * time.Second
	}
	cfg.RuntimeParams["application_name"] = "vec-preparar-material-interno"
	cfg.RuntimeParams["default_transaction_read_only"] = "on"
	con, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, errGobiernoConexion
	}
	tx, err := con.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		_ = con.Close(context.Background())
		return nil, errGobiernoConexion
	}
	g := &gobiernoPostgreSQL{con: con, tx: tx}
	var soloLectura bool
	for _, orden := range []string{
		`SET LOCAL search_path = pg_catalog`,
		`SET LOCAL statement_timeout = '10s'`,
		`SET LOCAL lock_timeout = '2s'`,
		`SET LOCAL ROLE ` + rolPropietarioAD3,
	} {
		if _, err := tx.Exec(ctx, orden); err != nil {
			g.cerrar()
			return nil, errGobiernoConexion
		}
	}
	if err := tx.QueryRow(ctx, `SELECT pg_catalog.current_setting('transaction_read_only') = 'on'
	   AND pg_catalog.current_setting('server_version_num')::integer >= 180000`).Scan(&soloLectura); err != nil || !soloLectura {
		g.cerrar()
		return nil, errGobiernoConexion
	}
	return g, nil
}

func (g *gobiernoPostgreSQL) cerrar() {
	if g == nil || g.con == nil {
		return
	}
	fin, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	_ = g.tx.Rollback(fin)
	_ = g.con.Close(fin)
	g.con = nil
}

// sqlClavePorHuella calcula vigencia, revocación (programada o efectiva),
// checkpoint y puntero de emisión vigente de la audiencia con el mismo reloj
// clock_timestamp() que la sonda AD3-69.
const sqlClavePorHuella = `
SELECT k.clave_id, k.version::bigint, k.revision_gobierno::bigint, k.huella_gobierno_sha256,
       k.emisor_id, k.audiencia_consumo, k.valida_desde, k.valida_hasta,
       pg_catalog.left(k.acto_ref, pg_catalog.length('acto:ct:desarrollo:clave-capacidad:')) = 'acto:ct:desarrollo:clave-capacidad:',
       k.valida_desde <= a.ahora AND a.ahora < k.valida_hasta,
       EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad r
                WHERE r.clave_id = k.clave_id AND r.version = k.version),
       k.revision_gobierno <= (SELECT c.revision FROM vec_autorizacion_atestada_v3.checkpoint_gobierno c WHERE c.control_id),
       (SELECT ROW(p.clave_id, p.version)
          FROM vec_autorizacion_atestada_v3.puntero_clave_emision p
          JOIN vec_autorizacion_atestada_v3.clave_capacidad_version q
            ON q.clave_id = p.clave_id AND q.version = p.version
         WHERE q.audiencia_consumo = k.audiencia_consumo AND p.establecida_en <= a.ahora
         ORDER BY p.orden DESC LIMIT 1) IS NOT DISTINCT FROM ROW(k.clave_id, k.version)
  FROM vec_autorizacion_atestada_v3.clave_capacidad_version k
 CROSS JOIN (SELECT pg_catalog.clock_timestamp() AS ahora) a
 WHERE k.huella_secreto_sha256 = $1`

// sqlRaizVigente sigue el puntero de configuración vigente hasta su raíz;
// devuelve una fila por raíz asociada para detectar conjuntos ampliados.
const sqlRaizVigente = `
WITH a AS (SELECT pg_catalog.clock_timestamp() AS ahora),
     ck AS (SELECT * FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id),
     cfg AS (
       SELECT c.revision, c.secuencia, c.publicada_en, c.expira_en
         FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
         JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c
           ON c.revision = p.configuracion_revision
        CROSS JOIN a
        WHERE p.establecida_en <= a.ahora
        ORDER BY p.orden DESC LIMIT 1)
SELECT r.clave_id, r.version::bigint, r.audiencia_despliegue,
       cfg.publicada_en <= a.ahora AND a.ahora < cfg.expira_en
       AND cfg.secuencia >= ck.configuracion_secuencia_minima
       AND NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion x
                        WHERE x.configuracion_revision = cfg.revision)
       AND r.version >= ck.raiz_version_minima
       AND r.valida_desde <= a.ahora AND a.ahora < r.valida_hasta
       AND NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz y
                        WHERE y.raiz_clave_id = r.clave_id AND y.raiz_version = r.version)
  FROM cfg
  JOIN vec_autorizacion_atestada_v3.configuracion_raiz cr ON cr.configuracion_revision = cfg.revision
  JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r
    ON r.clave_id = cr.raiz_clave_id AND r.version = cr.raiz_version
 CROSS JOIN a CROSS JOIN ck`

func (g *gobiernoPostgreSQL) clavePorHuellaSecreto(ctx context.Context, huella string) (filaClave, error) {
	var f filaClave
	var version, revision int64
	var dentro, puntero *bool
	err := g.tx.QueryRow(ctx, sqlClavePorHuella, huella).Scan(&f.ClaveID, &version, &revision, &f.HuellaGobierno,
		&f.EmisorID, &f.Audiencia, &f.Desde, &f.Hasta, &f.ActoPropio, &f.Vigente, &f.Revocada, &dentro, &puntero)
	if errors.Is(err, pgx.ErrNoRows) {
		return filaClave{}, errSinFila
	}
	if err != nil || version < 1 || revision < 1 {
		return filaClave{}, errGobiernoConexion
	}
	f.Version, f.Revision = uint64(version), uint64(revision)
	f.DentroCheckpoint = dentro != nil && *dentro
	f.PunteroVigente = puntero != nil && *puntero
	f.Desde, f.Hasta = f.Desde.UTC(), f.Hasta.UTC()
	return f, nil
}

func (g *gobiernoPostgreSQL) raizVigente(ctx context.Context) (filaRaiz, error) {
	filas, err := g.tx.Query(ctx, sqlRaizVigente)
	if err != nil {
		return filaRaiz{}, errGobiernoConexion
	}
	defer filas.Close()
	var r filaRaiz
	n := 0
	for filas.Next() {
		var version int64
		var vigente *bool
		if err := filas.Scan(&r.ClaveID, &version, &r.Audiencia, &vigente); err != nil || version < 1 {
			return filaRaiz{}, errGobiernoConexion
		}
		r.Version, r.Vigente = uint64(version), vigente != nil && *vigente
		n++
	}
	if filas.Err() != nil || n != 1 {
		return filaRaiz{}, errRaiz
	}
	return r, nil
}
