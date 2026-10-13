package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresqlcompartido "vec-diputacion-granada/internal/shared/postgresql"
	"vec-diputacion-granada/internal/shared/telemetria"
	confianzaatestacion "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

var errPublicacionConfianzaExterna = errors.New("bootstrap: publicacion de confianza externa no disponible")

const (
	rolPublicadorConfianzaExterna     = "vec_autorizacion_atestada_v3_publicador_externo"
	funcionPublicadorConfianzaExterna = "vec_autorizacion_atestada_v3.gestionar_publicacion_externa_v1(jsonb)"
)

// El publicador tiene un único grupo heredado y sólo la fachada nominal.
// Esta sonda se ejecuta al abrir cada conexión física, además del preflight
// común que comprueba que el login carece de TEMP.
const sondaPublicadorConfianzaExterna = `SELECT session_user::text,
 session_user=current_user AND pg_catalog.starts_with(session_user,'vec_confianza_externa_publicador_')
 AND l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb
 AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls
 AND NOT g.rolcanlogin AND NOT g.rolbypassrls
 AND pg_catalog.pg_has_role(session_user,g.oid,'USAGE')
 AND (SELECT pg_catalog.count(*)=1 AND pg_catalog.bool_and(m.roleid=g.oid
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
   FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=g.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.roleid=l.oid)
 AND pg_catalog.has_schema_privilege(session_user,'vec_autorizacion_atestada_v3','USAGE')
 AND coalesce(pg_catalog.has_function_privilege(session_user,
   pg_catalog.to_regprocedure($2)::oid,'EXECUTE'),false)
 AND NOT pg_catalog.has_database_privilege(session_user,pg_catalog.current_database(),'TEMPORARY')
 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_roles g ON g.rolname=$1
 WHERE l.rolname=session_user`

func comprobarPoolPublicadorConfianzaExterna(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) error {
	if ctx == nil || q == nil {
		return falloPublicacionConfianzaExterna(nil)
	}
	var usuario string
	var valido bool
	if q.QueryRow(ctx, sondaPublicadorConfianzaExterna, rolPublicadorConfianzaExterna,
		funcionPublicadorConfianzaExterna).Scan(&usuario, &valido) != nil || !valido || usuario == "" {
		return falloPublicacionConfianzaExterna(nil)
	}
	return nil
}

func abrirPoolPublicadorConfianzaExterna(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if ctx == nil || ctx.Err() != nil || strings.TrimSpace(dsn) == "" {
		return nil, errPublicacionConfianzaExterna
	}
	c, err := pgxpool.ParseConfig(strings.TrimSpace(dsn))
	if err != nil || c == nil || c.ConnConfig == nil || c.ConnConfig.User == "" ||
		len(c.ConnConfig.Fallbacks) != 0 || validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, false) != nil {
		return nil, errPublicacionConfianzaExterna
	}
	c.MaxConns, c.MinConns = 2, 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = make(map[string]string)
	}
	p := c.ConnConfig.RuntimeParams
	p["application_name"], p["timezone"], p["search_path"] = "vec-publicador-confianza-externa", "UTC", "pg_catalog,pg_temp"
	p["default_transaction_isolation"], p["default_transaction_read_only"] = "serializable", "off"
	p["statement_timeout"], p["lock_timeout"], p["idle_in_transaction_session_timeout"] = "15s", "2s", "20s"
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return comprobarPoolPublicadorConfianzaExterna(ctx, conn)
	}
	telemetria.Instrumentar(c)
	pool, err := postgresqlcompartido.NuevoPoolConPreflightTEMP(ctx, c)
	if err != nil {
		return nil, errPublicacionConfianzaExterna
	}
	if err := comprobarPoolPublicadorConfianzaExterna(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

type estadoPublicacionExternaV3 struct {
	Preimagen          string          `json:"preimagen_sha256"`
	SecuenciaSiguiente uint64          `json:"configuracion_secuencia_siguiente"`
	VersionSiguiente   uint64          `json:"clave_version_siguiente"`
	RevisionSiguiente  uint64          `json:"revision_gobierno_siguiente"`
	OrdenSiguiente     uint64          `json:"clave_orden_siguiente"`
	Actual             json.RawMessage `json:"publicacion_externa_actual"`
}

type materialPublicoExternoV3 struct {
	Configuracion struct {
		Revision    string    `json:"revision"`
		Secuencia   uint64    `json:"secuencia"`
		PublicadaEn time.Time `json:"publicada_en"`
		ExpiraEn    time.Time `json:"expira_en"`
	} `json:"configuracion"`
	Raiz struct {
		ClaveID     string    `json:"clave_id"`
		Version     uint64    `json:"version"`
		HuellaSPKI  string    `json:"huella_spki_sha256"`
		ValidaDesde time.Time `json:"valida_desde"`
		ValidaHasta time.Time `json:"valida_hasta"`
	} `json:"raiz"`
	Claves []coordenadaClaveExternaV3 `json:"claves"`
}

// La aplicación entrega material completo y compara la respuesta pública
// antes del COMMIT. Un replay sólo vale si el material derivado de los secretos
// actuales coincide con todos los identificadores y huellas publicados.
func publicarConfianzaExternaV3(ctx context.Context, pool *pgxpool.Pool,
	materiales map[string][]materialAtestacionContratacionTemporalDesarrollo, ahora time.Time) error {
	if ctx == nil || pool == nil || len(materiales) != len(consumidoresPortalExternoV3) {
		return falloPublicacionConfianzaExterna(nil)
	}
	var esperada []byte
	err := reintentarSerializacionCTDesarrollo(ctx, func() error {
		esperada = nil
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		var estadoBytes []byte
		if err := tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.gestionar_publicacion_externa_v1(NULL::pg_catalog.jsonb)`).Scan(&estadoBytes); err != nil {
			return err
		}
		var estado estadoPublicacionExternaV3
		if json.Unmarshal(estadoBytes, &estado) != nil || estado.Preimagen == "" {
			return falloPublicacionConfianzaExterna(nil)
		}
		lista := ordenarMaterialExternoV3(materiales)
		if len(lista) != 23 {
			return falloPublicacionConfianzaExterna(nil)
		}
		var actual materialPublicoExternoV3
		if len(estado.Actual) != 0 && string(estado.Actual) != "null" && json.Unmarshal(estado.Actual, &actual) == nil &&
			actual.Raiz.HuellaSPKI == lista[0].spkiHuella && actual.Raiz.ClaveID == "clave:atestacion:externo:"+lista[0].spkiHuella &&
			ahora.UTC().Before(actual.Configuracion.ExpiraEn) && len(actual.Claves) == len(lista) {
			sort.Slice(actual.Claves, func(i, j int) bool { return actual.Claves[i].Audiencia < actual.Claves[j].Audiencia })
			if err := coordinarMaterialExternoV3(lista, actual.Configuracion.Revision, actual.Configuracion.Secuencia,
				actual.Configuracion.PublicadaEn, actual.Configuracion.ExpiraEn, actual.Raiz.ValidaDesde, actual.Raiz.ValidaHasta,
				actual.Claves); err != nil {
				return err
			}
			propuesta, err := serializarMaterialExternoV3(lista)
			if err != nil {
				return err
			}
			if publicoIgualExternoV3(propuesta, estado.Actual) {
				esperada = propuesta
				var confirmado []byte
				if err := tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.gestionar_publicacion_externa_v1($1::pg_catalog.jsonb)`, propuesta).Scan(&confirmado); err != nil {
					return err
				}
				if !publicoIgualExternoV3(propuesta, confirmado) {
					return falloPublicacionConfianzaExterna(nil)
				}
				return tx.Commit(ctx)
			}
		}
		if estado.SecuenciaSiguiente == 0 || estado.VersionSiguiente == 0 || estado.RevisionSiguiente == 0 || estado.OrdenSiguiente == 0 ||
			estado.SecuenciaSiguiente > uint64(maximoVersionGobiernoPostgreSQLContratacionTemporalDesarrollo) ||
			estado.VersionSiguiente+uint64(len(lista)) > uint64(maximoVersionGobiernoPostgreSQLContratacionTemporalDesarrollo) ||
			estado.RevisionSiguiente+uint64(len(lista)) > uint64(maximoVersionGobiernoPostgreSQLContratacionTemporalDesarrollo) ||
			estado.OrdenSiguiente+uint64(len(lista)) > uint64(maximoVersionGobiernoPostgreSQLContratacionTemporalDesarrollo) {
			return falloPublicacionConfianzaExterna(nil)
		}
		versiones := make([]coordenadaClaveExternaV3, len(lista))
		for i, m := range lista {
			versiones[i].Audiencia = m.audienciaConsumo
			versiones[i].Version = estado.VersionSiguiente + uint64(i)
			versiones[i].Revision = estado.RevisionSiguiente + uint64(i)
			versiones[i].Orden = estado.OrdenSiguiente + uint64(i)
		}
		publicada := time.Date(ahora.UTC().Year(), ahora.UTC().Month(), ahora.UTC().Day(), 0, 0, 0, 0, time.UTC)
		if err := coordinarMaterialExternoV3(lista, "confianza:atestacion:externo:"+numeroDecimal64(estado.SecuenciaSiguiente),
			estado.SecuenciaSiguiente, publicada, publicada.Add(24*time.Hour), lista[0].validaDesde, lista[0].validaHasta, versiones); err != nil {
			return err
		}
		propuesta, err := serializarMaterialExternoV3(lista)
		if err != nil {
			return err
		}
		esperada = propuesta
		var confirmado []byte
		if err := tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.gestionar_publicacion_externa_v1($1::pg_catalog.jsonb)`, propuesta).Scan(&confirmado); err != nil {
			return err
		}
		if !publicoIgualExternoV3(propuesta, confirmado) {
			return falloPublicacionConfianzaExterna(nil)
		}
		return tx.Commit(ctx)
	})
	if err != nil {
		// Un COMMIT con respuesta perdida puede haber dejado la publicación
		// durable. Releer por la misma fachada antes de informar fallo evita
		// que el operador publique otra propuesta para una operación ya hecha.
		if len(esperada) != 0 && confirmarPublicacionExternaV3(ctx, pool, esperada) == nil {
			return nil
		}
		return falloPublicacionConfianzaExterna(err)
	}
	return nil
}

func confirmarPublicacionExternaV3(ctx context.Context, pool *pgxpool.Pool, esperada []byte) error {
	if ctx == nil || ctx.Err() != nil || pool == nil || len(esperada) == 0 {
		return falloPublicacionConfianzaExterna(nil)
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return falloPublicacionConfianzaExterna(nil)
	}
	defer tx.Rollback(ctx)
	var respuesta []byte
	if err := tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.gestionar_publicacion_externa_v1(NULL::pg_catalog.jsonb)`).Scan(&respuesta); err != nil {
		return falloPublicacionConfianzaExterna(nil)
	}
	var estado estadoPublicacionExternaV3
	if json.Unmarshal(respuesta, &estado) != nil || !publicoIgualExternoV3(esperada, estado.Actual) {
		return falloPublicacionConfianzaExterna(nil)
	}
	return nil
}

func ordenarMaterialExternoV3(materiales map[string][]materialAtestacionContratacionTemporalDesarrollo) []*materialAtestacionContratacionTemporalDesarrollo {
	var lista []*materialAtestacionContratacionTemporalDesarrollo
	for _, c := range consumidoresPortalExternoV3 {
		for i := range materiales[c] {
			lista = append(lista, &materiales[c][i])
		}
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].audienciaConsumo < lista[j].audienciaConsumo })
	return lista
}

type coordenadaClaveExternaV3 struct {
	Audiencia string `json:"audiencia_consumo"`
	Version   uint64 `json:"version"`
	Revision  uint64 `json:"revision_gobierno"`
	Orden     uint64 `json:"orden"`
}

func coordinarMaterialExternoV3(lista []*materialAtestacionContratacionTemporalDesarrollo, revision string, secuencia uint64,
	publicada, expira, desde, hasta time.Time, coordenadas []coordenadaClaveExternaV3) error {
	if len(lista) == 0 || len(lista) != len(coordenadas) || revision == "" || secuencia == 0 ||
		!publicada.Before(expira) || !desde.Before(hasta) {
		return falloPublicacionConfianzaExterna(nil)
	}
	for i, m := range lista {
		if coordenadas[i].Audiencia != m.audienciaConsumo || coordenadas[i].Version == 0 || coordenadas[i].Revision == 0 || coordenadas[i].Orden == 0 {
			return falloPublicacionConfianzaExterna(nil)
		}
		m.claveID = "clave:atestacion:externo:" + m.spkiHuella
		m.claveVersion = 1
		m.emisorID = "emisor:externo:" + m.spkiHuella
		huellaAudiencia := sha256.Sum256([]byte(m.audienciaConsumo))
		m.claveHMACID = "clave:capacidad:externo:" + hex.EncodeToString(huellaAudiencia[:])
		m.claveHMACVersion = coordenadas[i].Version
		m.claveHMACRevision = coordenadas[i].Revision
		m.claveHMACOrden = coordenadas[i].Orden
		m.validaDesde, m.validaHasta = desde.UTC(), hasta.UTC()
		m.publicadaEn, m.expiraEn = publicada.UTC(), expira.UTC()
		m.configuracionRef, m.configuracionOrden = revision, secuencia
		if err := reconstruirClavesGobiernoPostgreSQLContratacionTemporalDesarrollo(m); err != nil {
			return falloPublicacionConfianzaExterna(nil)
		}
		c, err := confianzaatestacion.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(revision, secuencia, m.publicadaEn, m.expiraEn, m.raiz)
		if err != nil {
			return falloPublicacionConfianzaExterna(nil)
		}
		m.configuracion = c
		m.configuracionHuella, err = c.HuellaSHA256ParaGobierno()
		if err != nil {
			return falloPublicacionConfianzaExterna(nil)
		}
	}
	return nil
}

func serializarMaterialExternoV3(lista []*materialAtestacionContratacionTemporalDesarrollo) ([]byte, error) {
	if len(lista) == 0 {
		return nil, errPublicacionConfianzaExterna
	}
	m := lista[0]
	claves := make([]map[string]any, 0, len(lista))
	for _, x := range lista {
		if x.configuracionRef != m.configuracionRef || x.claveID != m.claveID {
			return nil, errPublicacionConfianzaExterna
		}
		claves = append(claves, map[string]any{
			"clave_id": x.claveHMACID, "version": x.claveHMACVersion, "revision_gobierno": x.claveHMACRevision,
			"huella_gobierno_sha256": x.claveHMACHuella, "secreto_hmac_hex": hex.EncodeToString(x.claveHMAC),
			"huella_secreto_sha256": x.claveHMACSecreto, "emisor_id": x.emisorID, "audiencia_consumo": x.audienciaConsumo,
			"valida_desde": x.validaDesde, "valida_hasta": x.validaHasta, "orden": x.claveHMACOrden,
		})
	}
	return json.Marshal(map[string]any{
		"configuracion": map[string]any{"revision": m.configuracionRef, "secuencia": m.configuracionOrden,
			"huella_configuracion_sha256": m.configuracionHuella, "publicada_en": m.publicadaEn, "expira_en": m.expiraEn},
		"raiz": map[string]any{"clave_id": m.claveID, "version": m.claveVersion,
			"clave_publica_spki_hex": hex.EncodeToString(m.spki), "huella_spki_sha256": m.spkiHuella,
			"valida_desde": m.validaDesde, "valida_hasta": m.validaHasta,
			"suite":                confianzaatestacion.SuiteAtestacionAutorizacionV3COSEEdDSA,
			"audiencia_despliegue": audienciaAtestacionContratacionTemporalDesarrollo},
		"claves": claves,
	})
}

func publicoIgualExternoV3(propuesta, publicado []byte) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(propuesta, &m) != nil {
		return false
	}
	delete(m, "claves")
	// La respuesta SQL omite solo los secretos de las claves. Reconstruir esa
	// proyección localmente evita aceptar una respuesta parcial o divergente.
	var entrada struct {
		Claves []map[string]any `json:"claves"`
	}
	if json.Unmarshal(propuesta, &entrada) != nil {
		return false
	}
	for _, clave := range entrada.Claves {
		delete(clave, "secreto_hmac_hex")
	}
	claves, err := json.Marshal(entrada.Claves)
	if err != nil {
		return false
	}
	m["claves"] = claves
	esperado, err := json.Marshal(m)
	if err != nil {
		return false
	}
	var a, b map[string]any
	if json.Unmarshal(esperado, &a) != nil || json.Unmarshal(publicado, &b) != nil {
		return false
	}
	// SQL ordena por la colación de la base; el orden no forma parte del
	// material de confianza. Cotejar por audiencia sin admitir duplicados.
	normalizar := func(m map[string]any) bool {
		lista, ok := m["claves"].([]any)
		if !ok {
			return false
		}
		claves := make(map[string]any, len(lista))
		for _, entrada := range lista {
			clave, ok := entrada.(map[string]any)
			if !ok {
				return false
			}
			audiencia, ok := clave["audiencia_consumo"].(string)
			if !ok || audiencia == "" {
				return false
			}
			if _, duplicada := claves[audiencia]; duplicada {
				return false
			}
			claves[audiencia] = clave
		}
		m["claves"] = claves
		return true
	}
	return normalizar(a) && normalizar(b) && reflect.DeepEqual(a, b)
}

func falloPublicacionConfianzaExterna(causa error) error {
	registrarFalloPostgreSQLContratacionTemporalDesarrollo(etapaLlamadorPostgreSQLCTDesarrollo(2), causaFalloPostgreSQLCTDesarrollo(causa))
	return errPublicacionConfianzaExterna
}
