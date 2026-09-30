//go:build ignore

// Testigo sintético de las tres lecturas RPT con emisión V3 y consumo PostgreSQL reales.
// Se ejecuta solo dentro del clon efímero preparado por lecturas_v3_pg18.sh.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/veraison/go-cose"
	postgrescontexto "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	catalogo        = "rpt:testigo:ct"
	modulo          = "contratacion_temporal"
	consumidor      = "contratacion_temporal"
	rolRuntime      = "vec_rpt_testigo_ct_ejecutor"
	rolContexto     = "vec_rpt_testigo_contexto"
	rolRevalidacion = "vec_rpt_testigo_revalidador"
)

type dsnArchivo struct {
	Admin        string `json:"admin"`
	Contexto     string `json:"contexto"`
	Runtime      string `json:"runtime"`
	Revalidacion string `json:"revalidacion"`
}

var etapa = "inicio"

func fallo(err error) {
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			fmt.Fprintf(os.Stderr, "RPT-V3-FALLO: %s (SQLSTATE %s)\n", etapa, pg.Code)
		} else {
			fmt.Fprintf(os.Stderr, "RPT-V3-FALLO: %s\n", etapa)
		}
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) != 1 {
		fallo(errors.New("el testigo no acepta argumentos"))
	}
	b, err := os.ReadFile("/tmp/dsn.json")
	fallo(err)
	var cfg dsnArchivo
	fallo(json.Unmarshal(b, &cfg))
	if cfg.Admin == "" || cfg.Contexto == "" || cfg.Runtime == "" || cfg.Revalidacion == "" {
		fallo(errors.New("DSN incompletas"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, cfg.Admin)
	fallo(err)
	defer admin.Close(context.Background())
	fallo(ejecutar(ctx, admin, cfg))
	fmt.Println("RPT-V3-TESTIGO-OK")
}

func shaTexto(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func shaBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

const maximoEnteroExacto = 9007199254740991

func versionNumerica(v uint64) (pgtype.Numeric, error) {
	if v == 0 || v > maximoEnteroExacto {
		return pgtype.Numeric{}, errors.New("versión fuera de rango SQL")
	}
	return pgtype.Numeric{Int: new(big.Int).SetUint64(v), Exp: 0, Valid: true}, nil
}

func siguienteVersion(v int64) (uint64, error) {
	if v < 0 || v >= maximoEnteroExacto {
		return 0, errors.New("contador de gobierno fuera de rango")
	}
	return strconv.ParseUint(strconv.FormatInt(v+1, 10), 10, 64)
}

func id32() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	fallo(err)
	return hex.EncodeToString(b[:])
}

type reloj struct{ ahora time.Time }

func (r reloj) Ahora() time.Time { return r.ahora }

type relojActual struct{}

func (relojActual) Ahora() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

type generadorCorrelacion struct{ ref string }

func (g generadorCorrelacion) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return g.ref, nil
}

type candidato struct {
	cuenta, perfil, persona, sesion, autenticacion string
	metodo, garantia                               string
}

func bytes32() []byte {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	fallo(err)
	return b
}

func crearProyeccionSintetica(ctx context.Context, db *pgx.Conn, c candidato) error {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	desde, hasta := ahora.Add(-time.Minute), ahora.Add(time.Hour)
	procedencia := "prc_rpt_testigo_" + id32()
	vinculo := "vca_sintetico_" + id32()
	huella := shaTexto(procedencia)
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE vec_contexto_actor_v1_propietario`); err != nil {
		return err
	}
	consultas := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO vec_contexto_actor_v1.procedencias VALUES($1,1,$2,'autoridad_maestra_acreditada')`, []any{procedencia, huella}},
		{`INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones VALUES($1,1,$2,1,$3,'autoridad_maestra_acreditada','activo',$4,$5)`, []any{c.cuenta, procedencia, huella, desde, hasta}},
		{`INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual VALUES($1,1)`, []any{c.cuenta}},
		{`INSERT INTO vec_contexto_actor_v1.persona_versiones VALUES($1,1,$2,1,$3,'autoridad_maestra_acreditada','activo',$4,$5)`, []any{c.persona, procedencia, huella, desde, hasta}},
		{`INSERT INTO vec_contexto_actor_v1.persona_actual VALUES($1,1)`, []any{c.persona}},
		{`INSERT INTO vec_contexto_actor_v1.perfil_versiones VALUES($1,1,$2,$3,1,$4,'autoridad_maestra_acreditada','activo',$5,$6)`, []any{c.perfil, c.persona, procedencia, huella, desde, hasta}},
		{`INSERT INTO vec_contexto_actor_v1.perfil_actual VALUES($1,1)`, []any{c.perfil}},
		{`INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones VALUES($1,1,$2,$3,$4,$5,1,$6,'autoridad_maestra_acreditada','activo',$7,$8)`, []any{vinculo, c.cuenta, c.perfil, c.persona, procedencia, huella, desde, hasta}},
		{`INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual VALUES($1,1)`, []any{vinculo}},
	}
	for _, q := range consultas {
		if _, err = tx.Exec(ctx, q.sql, q.args...); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func crearCandidatoSintetico(ctx context.Context, db *pgx.Conn) (candidato, error) {
	var c candidato
	c.perfil = "prf_sintetico_" + id32()
	c.persona = "per_sintetica_" + id32()
	c.metodo = "kerberos_ad"
	c.garantia = "alto"
	cuentaHMAC, sujetoHMAC, sesionHMAC, asercionHMAC := bytes32(), bytes32(), bytes32(), bytes32()
	dominio, clave := "idh_"+id32(), "clave-hsm-rpt-testigo"
	tx, err := db.Begin(ctx)
	if err != nil {
		return c, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE vec_identidad_sesiones_v1_provisionador`); err != nil {
		return c, err
	}
	err = tx.QueryRow(ctx, `SELECT cuenta_ref FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
	 $1,'vec.identidad.hmac-sha256.v1',$2,$3,1,$4,$5,false,NULL)`, "opr_"+id32(), dominio, clave, cuentaHMAC, sujetoHMAC).Scan(&c.cuenta)
	if err != nil {
		return c, err
	}
	if err = tx.Commit(ctx); err != nil {
		return c, err
	}
	if err = crearProyeccionSintetica(ctx, db, c); err != nil {
		return c, err
	}
	verificada := time.Now().UTC().Truncate(time.Microsecond).Add(-2 * time.Second)
	emitida := verificada.Add(time.Second)
	expira := verificada.Add(4 * time.Minute)
	autenticacionHuella := shaTexto("aut:rpt:testigo:" + id32())
	politicaHuella := shaTexto("politica:rpt:testigo:" + id32())
	politicaRef := "pga_" + id32()
	tx, err = db.Begin(ctx)
	if err != nil {
		return c, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE vec_identidad_sesiones_v1_registrador`); err != nil {
		return c, err
	}
	var asercion, control, revision, estado, controlHuella, cuenta, ordinaria string
	var revalidada, valida time.Time
	err = tx.QueryRow(ctx, `SELECT * FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
	 $1,'vec.identidad.hmac-sha256.v1',$2,$3,1,$4,$5,$6,$7,NULL,false,
	 'interna_corporativa','kerberos_ad','alto',$8,$9,$10,$11,$12,$13)`,
		"opr_"+id32(), dominio, clave, asercionHMAC, sesionHMAC, sujetoHMAC, cuentaHMAC,
		autenticacionHuella, verificada, emitida, expira, politicaRef, politicaHuella).Scan(
		&c.autenticacion, &asercion, &c.sesion, &control, &revision, &estado, &controlHuella, &cuenta, &ordinaria, &revalidada, &valida)
	if err != nil {
		return c, err
	}
	if revision != "1" || estado != "activa" || cuenta != c.cuenta || ordinaria != c.cuenta || !valida.After(revalidada) {
		return c, errors.New("sesión sintética no confirmada")
	}
	if err = tx.Commit(ctx); err != nil {
		return c, err
	}
	return c, nil
}

func candidatos(ctx context.Context, db *pgx.Conn) ([]candidato, error) {
	result := make([]candidato, 0, 2)
	for i := 0; i < 2; i++ {
		c, err := crearCandidatoSintetico(ctx, db)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func crearLogins(ctx context.Context, db *pgx.Conn) error {
	_, err := db.Exec(ctx, `CREATE ROLE vec_rpt_testigo_contexto LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD NULL`)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `GRANT vec_contexto_actor_v1_runtime TO vec_rpt_testigo_contexto WITH ADMIN FALSE, INHERIT TRUE, SET FALSE`)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `CREATE ROLE vec_rpt_testigo_ct_ejecutor LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD NULL`)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `GRANT vec_contratacion_temporal_ejecutor TO vec_rpt_testigo_ct_ejecutor WITH ADMIN FALSE, INHERIT TRUE, SET FALSE`)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `CREATE ROLE vec_rpt_testigo_revalidador LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD NULL`)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `GRANT vec_identidad_sesiones_v1_revalidador TO vec_rpt_testigo_revalidador WITH ADMIN FALSE, INHERIT TRUE, SET FALSE`)
	if err != nil {
		return err
	}
	// CONNECT llega exclusivamente por el grupo nominal. Un GRANT directo al
	// LOGIN crearía pg_shdepend y la frontera de Contexto Actor lo rechazaría.
	return nil
}

type perfilPrueba struct {
	actor       candidato
	instantanea domain.InstantaneaAutorizacion
}

func concesionesRPT() []domain.ConcesionRol {
	return []domain.ConcesionRol{
		{Accion: "vec.catalogos.categorias.listar_habilitadas", ModuloID: modulo, TipoRecurso: "catalogo_configurable",
			Finalidades: []string{"consultar_categorias_rpt"}, GarantiaMinima: domain.AuthAssuranceHigh,
			CamposPermitidos: []string{"categorias", "paginacion", "publicaciones"}},
		{Accion: "vec.catalogos.categorias.consultar_historica", ModuloID: modulo, TipoRecurso: "catalogo_configurable",
			Finalidades: []string{"consultar_categorias_rpt"}, GarantiaMinima: domain.AuthAssuranceHigh,
			CamposPermitidos: []string{"publicacion", "entrada", "control_actual"}},
		{Accion: "vec.catalogos.categorias.consultar_uso", ModuloID: modulo, TipoRecurso: "uso_categoria",
			Finalidades: []string{"consultar_categorias_rpt"}, GarantiaMinima: domain.AuthAssuranceHigh,
			CamposPermitidos: []string{"uso"}},
	}
}

func provisionarPerfiles(ctx context.Context, admin *pgx.Conn, actores []candidato) ([]perfilPrueba, error) {
	if len(actores) != 2 {
		return nil, errors.New("dos perfiles requeridos")
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	rol := domain.VersionRol{RolID: "rpt_testigo_lecturas_" + id32(), Version: 1, Nombre: "RPT testigo sintético", Estado: domain.EstadoVersionRolPublicada,
		Concesiones: concesionesRPT(), PublicadaPor: "autoridad_sintetica_rpt", PublicadaEn: ahora.Add(-30 * time.Minute)}
	rolHash, err := rol.HuellaSHA256()
	if err != nil {
		return nil, err
	}
	rolJSON, err := json.Marshal(rol)
	if err != nil {
		return nil, err
	}
	control := domain.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1,
		Estado: domain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "autoridad_sintetica_rpt", ActualizadoEn: ahora.Add(-20 * time.Minute)}
	controlHash, err := control.HuellaSHA256()
	if err != nil {
		return nil, err
	}
	controlJSON, err := json.Marshal(control)
	if err != nil {
		return nil, err
	}
	var revision uint64
	var catalogoHash string
	err = admin.QueryRow(ctx, `SELECT revision,huella_sha256 FROM vec_autorizacion.control_catalogo_politicas WHERE control_id`).Scan(&revision, &catalogoHash)
	if err != nil {
		return nil, err
	}
	hashVacio, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return nil, err
	}
	if catalogoHash != hashVacio {
		return nil, errors.New("políticas activas no previstas")
	}
	tx, err := admin.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_propietario`)
	if err != nil {
		return nil, err
	}
	etapa = "RBAC: version rol"
	_, err = tx.Exec(ctx, `INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
	 VALUES($1,$2,$3,$4,$5,$6::jsonb)`, rol.Referencia(), rol.RolID, rol.Version, rolHash, rol.PublicadaEn, rolJSON)
	if err != nil {
		return nil, err
	}
	etapa = "RBAC: control rol"
	_, err = tx.Exec(ctx, `INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
	 VALUES($1,$2,$3,$4,$5,$6::jsonb)`, control.VersionRolRef, control.Revision, control.Estado, controlHash, control.ActualizadoEn, controlJSON)
	if err != nil {
		return nil, err
	}
	etapa = "RBAC: puntero control"
	_, err = tx.Exec(ctx, `INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
	 (version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref) VALUES($1,1,$2,$3,$4)`,
		rol.Referencia(), ahora, "autoridad_sintetica_rpt", "acto:rpt:testigo:control")
	if err != nil {
		return nil, err
	}
	result := make([]perfilPrueba, 0, 2)
	for i, c := range actores {
		var anterior, asignacionID string
		var versionAnterior int64
		etapa = "RBAC: preimagen asignacion"
		e := tx.QueryRow(ctx, `SELECT a.asignacion_ref,a.asignacion_id,a.version
		 FROM vec_autorizacion.asignacion_perfil_actual actual
		 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=actual.asignacion_ref
		 WHERE actual.perfil_activo_ref=$1 FOR UPDATE OF actual`, c.perfil).Scan(&anterior, &asignacionID, &versionAnterior)
		if errors.Is(e, pgx.ErrNoRows) {
			asignacionID = "rpt_testigo_" + id32()
			versionAnterior = 0
		} else if e != nil {
			return nil, e
		}
		ambitos := []domain.AmbitoPerfil{{Clave: "catalogo_id", Valores: []string{catalogo}}, {Clave: "modulo_id", Valores: []string{modulo}}}
		if i == 1 {
			ambitos = []domain.AmbitoPerfil{{Clave: "catalogo_id", Valores: []string{catalogo}}, {Clave: "consumidor", Valores: []string{consumidor}}, {Clave: "modulo_id", Valores: []string{modulo}}}
		}
		asig := domain.AsignacionPerfil{AsignacionID: asignacionID, Version: int(versionAnterior + 1), PerfilActivoRef: c.perfil, PrincipalID: c.persona,
			VersionRolRef: rol.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva, Ambitos: ambitos,
			EmitidaPor: "autoridad_sintetica_rpt", EmitidaEn: ahora.Add(-19 * time.Minute),
			VigenteDesde: ahora.Add(-18 * time.Minute), VigenteHasta: ahora.Add(2 * time.Hour)}
		asigHash, e := asig.HuellaSHA256()
		if e != nil {
			return nil, e
		}
		asigJSON, e := json.Marshal(asig)
		if e != nil {
			return nil, e
		}
		etapa = "RBAC: asignacion"
		_, e = tx.Exec(ctx, `INSERT INTO vec_autorizacion.asignacion_perfil
		 (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)`, asig.Referencia(), asig.AsignacionID, asig.Version,
			asig.PerfilActivoRef, asig.PrincipalID, asig.VersionRolRef, asigHash, asig.EmitidaEn, asigJSON)
		if e != nil {
			return nil, e
		}
		etapa = "RBAC: puntero asignacion"
		if anterior == "" {
			_, e = tx.Exec(ctx, `INSERT INTO vec_autorizacion.asignacion_perfil_actual
			 (perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref) VALUES($1,$2,$3,$4,$5)`,
				c.perfil, asig.Referencia(), ahora, "autoridad_sintetica_rpt", "acto:rpt:testigo:asignacion:"+id32())
		} else {
			var tag pgconn.CommandTag
			tag, e = tx.Exec(ctx, `UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref=$3,actualizada_en=$4,
			 actualizada_por=$5,acto_ref=$6 WHERE perfil_activo_ref=$1 AND asignacion_ref=$2`,
				c.perfil, anterior, asig.Referencia(), ahora, "autoridad_sintetica_rpt", "acto:rpt:testigo:asignacion:"+id32())
			if e == nil && tag.RowsAffected() != 1 {
				e = errors.New("CAS de asignación no afectó una fila")
			}
		}
		if e != nil {
			return nil, e
		}
		inst := domain.InstantaneaAutorizacion{AsignacionPerfil: asig, VersionRol: rol, ControlVigenciaVersionRol: control,
			RevisionCatalogoPoliticas: revision, CatalogoPoliticasHuellaSHA256: catalogoHash}
		if e = inst.Validar(); e != nil {
			return nil, e
		}
		result = append(result, perfilPrueba{actor: c, instantanea: inst})
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

type gobiernoPrueba struct {
	privada                                             ed25519.PrivateKey
	raiz                                                confianza.RaizPublicaAtestacionAutorizacionV3
	config                                              confianza.ConfiguracionConfianzaAtestacionAutorizacionV3
	clave                                               confianza.ClaveHMACCapacidadAtestacionV3
	secreto                                             []byte
	claveID, raizID, audiencia, revisionConfig          string
	secuencia, revisionClave, versionRaiz, versionClave uint64
	huellaGobierno                                      string
	publicada, expira, validaDesde, validaHasta         time.Time
}

func publicarYReservar(ctx context.Context, admin *pgx.Conn) (string, error) {
	var doc struct {
		ID       string              `json:"id"`
		ModuloID string              `json:"modulo_id"`
		Version  int                 `json:"version"`
		Estado   string              `json:"estado"`
		Entradas []map[string]string `json:"entradas"`
	}
	doc.ID = catalogo
	doc.ModuloID = modulo
	doc.Version = 1
	doc.Estado = "publicado"
	doc.Entradas = []map[string]string{{"clave": "cat-rpt-habilitada", "etiqueta": "Categoría habilitada"}, {"clave": "cat-rpt-reservada", "etiqueta": "Categoría reservada"}}
	can, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	huella := shaBytes(can)
	tx, err := admin.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario`)
	if err != nil {
		return "", err
	}
	var recibo string
	err = tx.QueryRow(ctx, `SELECT vec_catalogos_configurables.publicar($1,1,$2,$3,'{}'::jsonb,$4,
	 $5,$6,$7,$8,$9)`, catalogo, huella, string(can), shaTexto("{}"),
		"aprobacion:rpt:a", "aprobacion:rpt:b", "actor:rpt:testigo", "decision:rpt:publicar", "recibo:rpt:publicar").Scan(&recibo)
	if err != nil {
		return "", err
	}
	if recibo != "recibo:rpt:publicar" {
		return "", errors.New("publicación divergente")
	}
	err = tx.QueryRow(ctx, `SELECT vec_catalogos_configurables.reservar($1,$2,$3,$4,1,$5,$6,$7,$8)`,
		consumidor, "uso:rpt:testigo", "cat-rpt-reservada", catalogo, huella, "actor:rpt:testigo", "decision:rpt:reservar", "recibo:rpt:reserva").Scan(&recibo)
	if err != nil {
		return "", err
	}
	if recibo != "recibo:rpt:reserva" {
		return "", errors.New("reserva divergente")
	}
	var revision int64
	err = tx.QueryRow(ctx, `SELECT vec_catalogos_configurables.cambiar_proyeccion($1,1,'deshabilitar',NULL,NULL,$2,$3,$4)`,
		"cat-rpt-reservada", "actor:rpt:testigo", "decision:rpt:deshabilitar", "recibo:rpt:deshabilitar").Scan(&revision)
	if err != nil {
		return "", err
	}
	if revision != 2 {
		return "", errors.New("revision de proyección divergente")
	}
	return huella, tx.Commit(ctx)
}

func provisionarGobierno(ctx context.Context, admin *pgx.Conn) (g gobiernoPrueba, err error) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	acto := "acto:rpt:testigo:" + id32()
	g.publicada = ahora.Add(-time.Minute)
	g.expira = ahora.Add(time.Hour)
	g.validaDesde = ahora.Add(-time.Hour)
	g.validaHasta = ahora.Add(time.Hour)
	g.claveID = "clave:rpt:testigo:" + id32()
	g.raizID = "raiz:rpt:testigo:" + id32()
	g.audiencia = "vec/rpt/testigo"
	var checkpointSec, checkpointRaiz, maxRev, maxPuntero, maxConfig, maxVersionClave int64
	err = admin.QueryRow(ctx, `SELECT configuracion_secuencia_minima,raiz_version_minima FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id`).Scan(&checkpointSec, &checkpointRaiz)
	if err != nil {
		return g, err
	}
	err = admin.QueryRow(ctx, `SELECT coalesce(max(revision_gobierno),0) FROM vec_autorizacion_atestada_v3.clave_capacidad_version`).Scan(&maxRev)
	if err != nil {
		return g, err
	}
	err = admin.QueryRow(ctx, `SELECT coalesce(max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision`).Scan(&maxPuntero)
	if err != nil {
		return g, err
	}
	err = admin.QueryRow(ctx, `SELECT coalesce(max(version),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision`).Scan(&maxVersionClave)
	if err != nil {
		return g, err
	}
	err = admin.QueryRow(ctx, `SELECT coalesce(max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual`).Scan(&maxConfig)
	if err != nil {
		return g, err
	}
	if maxPuntero < 0 || maxPuntero >= maximoEnteroExacto || maxConfig < 0 || maxConfig >= maximoEnteroExacto {
		return g, errors.New("puntero de gobierno fuera de rango")
	}
	if g.secuencia, err = siguienteVersion(checkpointSec); err != nil {
		return g, err
	}
	if g.versionRaiz, err = siguienteVersion(checkpointRaiz); err != nil {
		return g, err
	}
	if g.revisionClave, err = siguienteVersion(maxRev); err != nil {
		return g, err
	}
	if g.versionClave, err = siguienteVersion(maxVersionClave); err != nil {
		return g, err
	}
	g.huellaGobierno = shaTexto("rpt:testigo:gobierno:" + g.claveID)
	publica, privada, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		return g, e
	}
	g.privada = privada
	g.raiz, e = confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(g.raizID, g.versionRaiz, publica, g.audiencia,
		confianza.EstadoClaveAtestacionAutorizacionV3Activa, g.validaDesde, g.validaHasta, time.Time{})
	if e != nil {
		return g, e
	}
	g.revisionConfig = "confianza:rpt:testigo:" + id32()
	g.config, e = confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(g.revisionConfig, g.secuencia, g.publicada, g.expira, g.raiz)
	if e != nil {
		return g, e
	}
	hashConfig, e := g.config.HuellaSHA256ParaGobierno()
	if e != nil {
		return g, e
	}
	g.secreto = make([]byte, 32)
	_, e = rand.Read(g.secreto)
	if e != nil {
		return g, e
	}
	g.clave, e = confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(g.claveID, g.versionClave, g.secreto, "emisor:rpt:testigo",
		"vec_catalogos_configurables.lectura_categorias.v1", confianza.EstadoClaveHMACCapacidadAtestacionV3Emision,
		g.validaDesde, g.validaHasta, time.Time{}, g.revisionClave, g.huellaGobierno)
	if e != nil {
		return g, e
	}
	spki, e := x509.MarshalPKIXPublicKey(publica)
	if e != nil {
		return g, e
	}
	tx, e := admin.Begin(ctx)
	if e != nil {
		return g, e
	}
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario`)
	if e != nil {
		return g, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
	 (clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
	 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, g.claveID, g.versionClave, g.revisionClave, g.huellaGobierno, g.secreto,
		shaBytes(g.secreto), "emisor:rpt:testigo", "vec_catalogos_configurables.lectura_categorias.v1", g.validaDesde, g.validaHasta, acto+":clave")
	if e != nil {
		return g, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision(orden,clave_id,version,establecida_en,acto_ref)
	 VALUES($1,$2,$3,$4,$5)`, maxPuntero+1, g.claveID, g.versionClave, ahora, acto+":puntero-clave")
	if e != nil {
		return g, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
	 (revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
	 VALUES($1,$2,$3,$4,$5,$6)`, g.revisionConfig, g.secuencia, hashConfig, g.publicada, g.expira, acto+":configuracion")
	if e != nil {
		return g, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
	 (clave_id,version,clave_publica_spki,huella_spki_sha256,valida_desde,valida_hasta,suite,audiencia_despliegue,acto_ref)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, g.raizID, g.versionRaiz, spki, shaBytes(spki), g.validaDesde, g.validaHasta,
		confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, g.audiencia, acto+":raiz")
	if e != nil {
		return g, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz(configuracion_revision,raiz_clave_id,raiz_version) VALUES($1,$2,$3)`,
		g.revisionConfig, g.raizID, g.versionRaiz)
	if e != nil {
		return g, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
	 (orden,configuracion_revision,establecida_en,acto_ref) VALUES($1,$2,$3,$4)`, maxConfig+1, g.revisionConfig, ahora, acto+":puntero-configuracion")
	if e != nil {
		return g, e
	}
	return g, tx.Commit(ctx)
}

func motivoSintetico(ctx context.Context, admin *pgx.Conn) (domain.ReferenciaEntradaCatalogo, error) {
	var secuencia int64
	err := admin.QueryRow(ctx, `SELECT ultima_secuencia FROM vec_autorizacion.motivo_v2_checkpoint_origen WHERE control_id`).Scan(&secuencia)
	if err != nil {
		return domain.ReferenciaEntradaCatalogo{}, err
	}
	m := domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_rpt_testigo_" + id32(), CatalogoVersion: 1,
		CatalogoHuellaSHA256: shaTexto("rpt:testigo:motivos:" + id32()), EntradaClave: "motivo_" + id32()}
	if err = m.Validar(); err != nil {
		return m, err
	}
	publicado := time.Now().UTC().Truncate(time.Microsecond).Add(-time.Minute)
	entradas, err := json.Marshal([]map[string]any{{"clave": m.EntradaClave,
		"vigente_desde": publicado.Format("2006-01-02T15:04:05.000000Z"), "vigente_hasta": nil}})
	if err != nil {
		return m, err
	}
	tx, err := admin.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return m, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_motivos_proyector`); err != nil {
		return m, err
	}
	var publicada bool
	err = tx.QueryRow(ctx, `SELECT vec_autorizacion.publicar_motivos_autorizacion_v2(
	 $1,$2,$3,$4,1,$5,$6,$7::jsonb)`, "evento_"+id32(), secuencia+1,
		shaTexto("evento:rpt:testigo:"+id32()), m.CatalogoID, m.CatalogoHuellaSHA256, publicado, string(entradas)).Scan(&publicada)
	if err != nil {
		return m, err
	}
	if !publicada {
		return m, errors.New("motivo sintético no publicado")
	}
	return m, tx.Commit(ctx)
}

type autoridadesPrueba struct {
	revalidador domain.RevalidadorAutenticacionActorV1
	contexto    domain.ResolutorContextoActorRegistradoV2
}

func vinculada(ctx context.Context, p perfilPrueba, autoridades autoridadesPrueba) (
	domain.VinculoAutenticacionActorV2, domain.ResultadoContextoActorRegistradoV2, error) {
	a := p.actor
	cuenta := domain.CuentaAutenticadaContextoActor{CuentaRef: a.cuenta,
		Metodo: domain.AuthMethod(a.metodo), Garantia: domain.AuthAssurance(a.garantia)}
	return domain.CrearVinculoAutenticacionActorV2ConResultado(ctx, autoridades.revalidador,
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: a.autenticacion, SesionRef: a.sesion},
		autoridades.contexto, domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: a.perfil}, relojActual{})
}

func materialHash(ctx context.Context, db *pgx.Conn, material string) (string, error) {
	var h string
	err := db.QueryRow(ctx, `SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to($1::jsonb::text,'UTF8')),'hex')`, material).Scan(&h)
	return h, err
}

type solicitudLectura struct {
	accion     string
	referencia string
	tipo       string
	campos     []string
	ambitos    map[string]string
	material   string
}

func emitir(ctx context.Context, admin *pgx.Conn, p perfilPrueba, autoridades autoridadesPrueba, g gobiernoPrueba, motivo domain.ReferenciaEntradaCatalogo,
	s solicitudLectura) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, string, error) {
	var vacia ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	etapa = "emisión: vínculo"
	vinculo, contexto, err := vinculada(ctx, p, autoridades)
	if err != nil {
		return vacia, "", err
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	etapa = "emisión: material"
	h, err := materialHash(ctx, admin, s.material)
	if err != nil {
		return vacia, "", err
	}
	cor, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, generadorCorrelacion{"correlacion_" + id32()})
	if err != nil {
		return vacia, "", err
	}
	recurso := domain.RecursoAutorizable{Referencia: s.referencia, ModuloID: modulo, Tipo: s.tipo,
		Ambitos: s.ambitos, Atributos: map[string]string{"material_sha256": h}}
	etapa = "emisión: solicitud"
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo, Accion: s.accion, Recurso: recurso,
		Finalidad: "consultar_categorias_rpt", Correlacion: cor})
	if err != nil {
		return vacia, "", err
	}
	decisionRef := "dec_" + id32()
	etapa = "emisión: evidencia"
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(solicitud, p.instantanea, decisionRef, ahora, ahora.Add(90*time.Second))
	if err != nil {
		return vacia, "", err
	}
	etapa = "emisión: decisión"
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		return vacia, "", err
	}
	concedida, _, err := decision.Resultado()
	if err != nil || !concedida {
		return vacia, "", errors.New("RBAC RPT sintética no concedida")
	}
	cabecera := domain.CabeceraAtestacionAutorizacionV3{FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: g.raizID, Audiencia: g.audiencia}
	etapa = "emisión: firma solicitud"
	firmaSolicitud, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV3(cabecera, decision, motivo, contexto)
	if err != nil {
		return vacia, "", err
	}
	payload, err := firmaSolicitud.Mensaje()
	if err != nil {
		return vacia, "", err
	}
	aad, err := confianza.AADExternoAtestacionAutorizacionV3(g.audiencia)
	if err != nil {
		return vacia, "", err
	}
	sobre := cose.NewSign1Message()
	sobre.Headers.Protected.SetAlgorithm(cose.AlgorithmEdDSA)
	sobre.Headers.Protected[cose.HeaderLabelKeyID] = []byte(g.raizID)
	sobre.Payload = payload
	firmante, err := cose.NewSigner(cose.AlgorithmEdDSA, g.privada)
	if err != nil {
		return vacia, "", err
	}
	if err = sobre.Sign(rand.Reader, aad, firmante); err != nil {
		return vacia, "", err
	}
	sobre.Payload = nil
	sobre.Headers.RawProtected = nil
	sobre.Headers.RawUnprotected = nil
	sobreBytes, err := sobre.MarshalCBOR()
	if err != nil {
		return vacia, "", err
	}
	etapa = "emisión: firma COSE"
	resultadoFirma, err := ports.NuevoResultadoFirmaAtestacionAutorizacionV3(firmaSolicitud, sobreBytes,
		"evidencia:rpt:testigo:"+id32(), ahora)
	if err != nil {
		return vacia, "", err
	}
	atestacion, err := ports.NuevaAtestacionAutorizacionV3(firmaSolicitud, resultadoFirma)
	if err != nil {
		return vacia, "", err
	}
	verificador, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(g.config, reloj{ahora})
	if err != nil {
		return vacia, "", err
	}
	etapa = "emisión: verificar COSE"
	prueba, err := verificador.Verificar(ctx, solicitud, decision, motivo, contexto, atestacion)
	if err != nil {
		return vacia, "", err
	}
	emisor, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(g.clave, reloj{ahora})
	if err != nil {
		return vacia, "", err
	}
	etapa = "emisión: HMAC"
	capacidad, err := emisor.Emitir(ctx, solicitud, decision, motivo, contexto, atestacion, prueba)
	if err != nil {
		return vacia, "", err
	}
	etapa = "emisión: exportación"
	material, err := confianza.NuevoMaterialConsumoAutorizacionAtestadaV3(solicitud, decision, motivo, contexto, atestacion, prueba, capacidad, g.raiz)
	if err != nil {
		return vacia, "", err
	}
	exportacion, err := material.ExportarMaterialParaConsumidor()
	if err != nil {
		return vacia, "", err
	}
	return exportacion, decisionRef, nil
}

func registrarDecision(ctx context.Context, admin *pgx.Conn, e ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) error {
	personaVersion, err := versionNumerica(e.PersonaVersion())
	if err != nil {
		return err
	}
	perfilVersion, err := versionNumerica(e.PerfilVersion())
	if err != nil {
		return err
	}
	tx, err := admin.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var concedida bool
	var codigo, huella string
	var fecha time.Time
	err = tx.QueryRow(ctx, `SELECT * FROM vec_autorizacion.registrar_decision_contexto_actor_v3($1,$2,$3::numeric,$4::numeric)`,
		e.DecisionCanonica(), e.MotivoCanonico(), personaVersion, perfilVersion).Scan(&concedida, &codigo, &huella, &fecha)
	if err != nil {
		return err
	}
	if !concedida || codigo != "concedida" || len(huella) != 64 || fecha.IsZero() {
		return errors.New("decisión V3 no registrada")
	}
	return tx.Commit(ctx)
}

func invocar(ctx context.Context, runtime *pgx.Conn, nombre string, material string,
	e ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (map[string]json.RawMessage, error) {
	personaVersion, err := versionNumerica(e.PersonaVersion())
	if err != nil {
		return nil, err
	}
	perfilVersion, err := versionNumerica(e.PerfilVersion())
	if err != nil {
		return nil, err
	}
	var consulta string
	switch nombre {
	case "listar":
		consulta = `SELECT vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada(
	 $1::jsonb,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	case "historica":
		consulta = `SELECT vec_autorizacion_atestada_v3.leer_publicacion_categoria_rpt_v3_atestada(
	 $1::jsonb,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	case "uso":
		consulta = `SELECT vec_autorizacion_atestada_v3.consultar_uso_categoria_rpt_v3_atestada(
	 $1::jsonb,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	default:
		return nil, errors.New("fachada no prevista")
	}
	tx, err := runtime.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL TIME ZONE 'UTC'`); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `SET LOCAL statement_timeout='5s'`); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `SET LOCAL idle_in_transaction_session_timeout='7s'`); err != nil {
		return nil, err
	}
	var statement, idle int
	if err = tx.QueryRow(ctx, `SELECT
	 (SELECT setting::int FROM pg_catalog.pg_settings WHERE name='statement_timeout' AND unit='ms'),
	 (SELECT setting::int FROM pg_catalog.pg_settings WHERE name='idle_in_transaction_session_timeout' AND unit='ms')`).Scan(&statement, &idle); err != nil {
		return nil, err
	}
	if statement != 5000 || idle != 7000 {
		return nil, errors.New("límites de transacción no fijados")
	}
	var contenido []byte
	err = tx.QueryRow(ctx, consulta, material, e.CapacidadCanonica(), e.DecisionCanonica(), e.MotivoCanonico(),
		e.ContextoActorCanonico(), personaVersion, perfilVersion, e.PayloadVECAD3(),
		e.SobreCOSESign1(), e.EvidenciaVerificacion(), e.RaizPublicaSPKI()).Scan(&contenido)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	var r map[string]json.RawMessage
	if err = json.Unmarshal(contenido, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func comprobarRecibo(ctx context.Context, admin *pgx.Conn, ref string, respuesta map[string]json.RawMessage, encontrado bool) error {
	var decision string
	if err := json.Unmarshal(respuesta["decision_ref"], &decision); err != nil {
		return err
	}
	if decision != ref {
		return errors.New("decisión ajena en recibo")
	}
	var consumo bool
	if err := json.Unmarshal(respuesta["consumo_nuevo"], &consumo); err != nil {
		return err
	}
	if !consumo {
		return errors.New("consumo V3 repetido")
	}
	var found bool
	if err := json.Unmarshal(respuesta["encontrado"], &found); err != nil {
		return err
	}
	if found != encontrado {
		return errors.New("estado de lectura inesperado")
	}
	if !found && string(respuesta["datos"]) != "null" {
		return errors.New("ausencia filtró datos")
	}
	for _, clave := range []string{"efecto_ref", "huella_efecto_sha256", "consumo_huella_sha256", "auditoria_ref", "consumida_en"} {
		if len(respuesta[clave]) == 0 || string(respuesta[clave]) == "null" {
			return errors.New("recibo incompleto")
		}
	}
	var nConsumo, nAuditoria int
	err := admin.QueryRow(ctx, `SELECT (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref=$1),
	 (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE decision_ref=$1)`, ref).Scan(&nConsumo, &nAuditoria)
	if err != nil {
		return err
	}
	if nConsumo != 1 || nAuditoria != 1 {
		return errors.New("consumo o auditoría no durable")
	}
	return nil
}

func lectura(ctx context.Context, admin, runtime *pgx.Conn, p perfilPrueba, autoridades autoridadesPrueba, g gobiernoPrueba, motivo domain.ReferenciaEntradaCatalogo,
	nombre string, s solicitudLectura, encontrado bool) (map[string]json.RawMessage, error) {
	exportacion, ref, err := emitir(ctx, admin, p, autoridades, g, motivo, s)
	if err != nil {
		return nil, err
	}
	etapa = "registro decisión"
	if err = registrarDecision(ctx, admin, exportacion); err != nil {
		return nil, err
	}
	etapa = "fachada " + nombre
	r, err := invocar(ctx, runtime, nombre, s.material, exportacion)
	if err != nil {
		return nil, err
	}
	etapa = "recibo " + nombre
	if err = comprobarRecibo(ctx, admin, ref, r, encontrado); err != nil {
		return nil, err
	}
	return r, nil
}

// Cada negativa tiene una decisión nueva y válida para el material original.
// La alteración se limita al descriptor presentado a la fachada; una
// denegación conserva consumo y auditoría ausentes.
func negativa(ctx context.Context, admin, runtime *pgx.Conn, p perfilPrueba, autoridades autoridadesPrueba,
	g gobiernoPrueba, motivo domain.ReferenciaEntradaCatalogo, nombre string, s solicitudLectura,
	materialAjeno string) error {
	exportacion, ref, err := emitir(ctx, admin, p, autoridades, g, motivo, s)
	if err != nil {
		return err
	}
	if err = registrarDecision(ctx, admin, exportacion); err != nil {
		return err
	}
	_, err = invocar(ctx, runtime, nombre, materialAjeno, exportacion)
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "42501" {
		return errors.New("descriptor ajeno no denegado con 42501")
	}
	return sinConsumo(ctx, admin, ref)
}

func sinConsumo(ctx context.Context, admin *pgx.Conn, ref string) error {
	var consumo, auditoria int
	err := admin.QueryRow(ctx, `SELECT
	 (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3 WHERE decision_ref=$1),
	 (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE decision_ref=$1)`, ref).Scan(&consumo, &auditoria)
	if err != nil {
		return err
	}
	if consumo != 0 || auditoria != 0 {
		return errors.New("denegación dejó consumo o auditoría")
	}
	return nil
}

func negativaSinLimite(ctx context.Context, admin, runtime *pgx.Conn, p perfilPrueba,
	autoridades autoridadesPrueba, g gobiernoPrueba, motivo domain.ReferenciaEntradaCatalogo,
	s solicitudLectura) error {
	exportacion, ref, err := emitir(ctx, admin, p, autoridades, g, motivo, s)
	if err != nil {
		return err
	}
	if err = registrarDecision(ctx, admin, exportacion); err != nil {
		return err
	}
	personaVersion, err := versionNumerica(exportacion.PersonaVersion())
	if err != nil {
		return err
	}
	perfilVersion, err := versionNumerica(exportacion.PerfilVersion())
	if err != nil {
		return err
	}
	tx, err := runtime.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, ajuste := range []string{`SET LOCAL TIME ZONE 'UTC'`, `SET LOCAL statement_timeout=0`, `SET LOCAL idle_in_transaction_session_timeout='7s'`} {
		if _, err = tx.Exec(ctx, ajuste); err != nil {
			return err
		}
	}
	var contenido []byte
	err = tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.listar_categorias_habilitadas_rpt_v3_atestada(
	 $1::jsonb,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`,
		s.material, exportacion.CapacidadCanonica(), exportacion.DecisionCanonica(), exportacion.MotivoCanonico(),
		exportacion.ContextoActorCanonico(), personaVersion, perfilVersion,
		exportacion.PayloadVECAD3(), exportacion.SobreCOSESign1(), exportacion.EvidenciaVerificacion(),
		exportacion.RaizPublicaSPKI()).Scan(&contenido)
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "22023" || pg.Message != "límites VEC-AD-3 ausentes" {
		return errors.New("fachada sustituyó el límite ausente")
	}
	if err = tx.Rollback(ctx); err != nil {
		return err
	}
	return sinConsumo(ctx, admin, ref)
}

func material(v any) string {
	b, err := json.Marshal(v)
	fallo(err)
	return string(b)
}

func ejecutar(ctx context.Context, admin *pgx.Conn, cfg dsnArchivo) error {
	etapa = "roles técnicos"
	if err := crearLogins(ctx, admin); err != nil {
		return err
	}
	etapa = "candidatos sintéticos"
	actores, err := candidatos(ctx, admin)
	if err != nil {
		return err
	}
	etapa = "pool contexto"
	contextoPool, err := pgxpool.New(ctx, cfg.Contexto)
	if err != nil {
		return err
	}
	defer contextoPool.Close()
	etapa = "pool revalidación"
	revalidacionPool, err := pgxpool.New(ctx, cfg.Revalidacion)
	if err != nil {
		return err
	}
	defer revalidacionPool.Close()
	etapa = "adaptador contexto productivo"
	resolutor, err := postgrescontexto.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, contextoPool)
	if err != nil {
		return err
	}
	servicioContexto, err := aplicacionvec.NuevoServicioContextoActorProductivoV2(
		resolutor, postgrescontexto.NuevoGeneradorOperacionContextoActorV2Criptografico(), relojActual{})
	if err != nil {
		return err
	}
	autoridadContexto, err := aplicacionvec.NuevaAutoridadContextoActorRegistradoV2(servicioContexto)
	if err != nil {
		return err
	}
	etapa = "adaptador revalidación productivo"
	revalidador, err := postgresidentidad.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, revalidacionPool)
	if err != nil {
		return err
	}
	autoridades := autoridadesPrueba{revalidador: revalidador, contexto: autoridadContexto}
	etapa = "provisión RBAC"
	perfiles, err := provisionarPerfiles(ctx, admin, actores)
	if err != nil {
		return err
	}
	etapa = "publicación RPT"
	huella, err := publicarYReservar(ctx, admin)
	if err != nil {
		return err
	}
	etapa = "gobierno V3"
	g, err := provisionarGobierno(ctx, admin)
	if err != nil {
		return err
	}
	defer clear(g.privada)
	defer clear(g.secreto)
	etapa = "motivo sintético"
	motivo, err := motivoSintetico(ctx, admin)
	if err != nil {
		return err
	}
	etapa = "conexión ejecutor"
	runtime, err := pgx.Connect(ctx, cfg.Runtime)
	if err != nil {
		return err
	}
	defer runtime.Close(context.Background())
	ambitosLista := map[string]string{"catalogo_id": catalogo, "modulo_id": modulo}
	ambitosUso := map[string]string{"catalogo_id": catalogo, "consumidor": consumidor, "modulo_id": modulo}
	listaMaterial := material(map[string]any{"catalogo_id": catalogo, "modulo_id": modulo, "cursor_categoria_id": nil, "limite": 100})
	lista := solicitudLectura{accion: "vec.catalogos.categorias.listar_habilitadas", referencia: catalogo, tipo: "catalogo_configurable",
		campos: []string{"categorias", "paginacion", "publicaciones"}, ambitos: ambitosLista, material: listaMaterial}
	etapa = "lista atestada"
	r, err := lectura(ctx, admin, runtime, perfiles[0], autoridades, g, motivo, "listar", lista, true)
	if err != nil {
		return err
	}
	var datosLista struct {
		Items   []map[string]any `json:"items"`
		Anclaje map[string]any   `json:"anclaje_publicacion"`
	}
	if err = json.Unmarshal(r["datos"], &datosLista); err != nil {
		return err
	}
	if len(datosLista.Items) != 1 || datosLista.Items[0]["categoria_id"] != "cat-rpt-habilitada" || datosLista.Anclaje["huella_sha256"] != huella {
		return errors.New("lista habilitada o anclaje incorrectos")
	}
	histMaterial := material(map[string]any{"catalogo_id": catalogo, "modulo_id": modulo, "version": 1,
		"huella_sha256": huella, "categoria_id": "cat-rpt-reservada"})
	historica := solicitudLectura{accion: "vec.catalogos.categorias.consultar_historica", referencia: catalogo,
		tipo: "catalogo_configurable", campos: []string{"publicacion", "entrada", "control_actual"}, ambitos: ambitosLista, material: histMaterial}
	etapa = "historia atestada"
	r, err = lectura(ctx, admin, runtime, perfiles[0], autoridades, g, motivo, "historica", historica, true)
	if err != nil {
		return err
	}
	var datosHistoricos struct {
		Control     map[string]any `json:"control_actual"`
		Publicacion map[string]any `json:"publicacion"`
	}
	if err = json.Unmarshal(r["datos"], &datosHistoricos); err != nil {
		return err
	}
	if datosHistoricos.Control["estado"] != "deshabilitada" || datosHistoricos.Publicacion["huella_sha256"] != huella {
		return errors.New("historia alterada por control actual")
	}
	usoMaterial := material(map[string]any{"catalogo_id": catalogo, "modulo_id": modulo, "consumidor": consumidor,
		"uso_ref": "uso:rpt:testigo", "reserva_recibo_ref": "recibo:rpt:reserva"})
	uso := solicitudLectura{accion: "vec.catalogos.categorias.consultar_uso", referencia: "uso:rpt:testigo", tipo: "uso_categoria",
		campos: []string{"uso"}, ambitos: ambitosUso, material: usoMaterial}
	etapa = "uso atestado"
	r, err = lectura(ctx, admin, runtime, perfiles[1], autoridades, g, motivo, "uso", uso, true)
	if err != nil {
		return err
	}
	var datosUso struct {
		Estado string `json:"estado"`
		Recibo string `json:"reserva_recibo_ref"`
		Huella string `json:"huella_sha256"`
	}
	if err = json.Unmarshal(r["datos"], &datosUso); err != nil {
		return err
	}
	if datosUso.Estado != "reservado" || datosUso.Recibo != "recibo:rpt:reserva" || datosUso.Huella != huella {
		return errors.New("uso previo no conservado")
	}
	// Un recibo ajeno produce ausencia con una decisión y auditoría nuevas.
	uso.material = material(map[string]any{"catalogo_id": catalogo, "modulo_id": modulo, "consumidor": consumidor,
		"uso_ref": "uso:rpt:testigo", "reserva_recibo_ref": "recibo:rpt:ajeno"})
	etapa = "ausencia auditada"
	if _, err = lectura(ctx, admin, runtime, perfiles[1], autoridades, g, motivo, "uso", uso, false); err != nil {
		return err
	}
	etapa = "negativa descriptor"
	if err = negativa(ctx, admin, runtime, perfiles[0], autoridades, g, motivo, "listar", lista,
		material(map[string]any{"catalogo_id": "rpt:testigo:ajeno", "modulo_id": modulo,
			"cursor_categoria_id": nil, "limite": 100})); err != nil {
		return err
	}
	etapa = "negativa módulo"
	if err = negativa(ctx, admin, runtime, perfiles[0], autoridades, g, motivo, "listar", lista,
		material(map[string]any{"catalogo_id": catalogo, "modulo_id": "bolsa",
			"cursor_categoria_id": nil, "limite": 100})); err != nil {
		return err
	}
	etapa = "negativa consumidor"
	uso.material = usoMaterial
	if err = negativa(ctx, admin, runtime, perfiles[1], autoridades, g, motivo, "uso", uso,
		material(map[string]any{"catalogo_id": catalogo, "modulo_id": modulo, "consumidor": "bolsa",
			"uso_ref": "uso:rpt:testigo", "reserva_recibo_ref": "recibo:rpt:reserva"})); err != nil {
		return err
	}
	etapa = "negativa límite de sesión"
	if err = negativaSinLimite(ctx, admin, runtime, perfiles[0], autoridades, g, motivo, lista); err != nil {
		return err
	}
	return nil
}
