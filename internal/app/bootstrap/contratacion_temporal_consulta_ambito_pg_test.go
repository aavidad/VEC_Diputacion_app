package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	ca "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	sesiones "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	pg "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridad "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Prueba optativa exclusiva del runner PG18 desechable. El preparador siembra
// datos gobernados; cada autoridad de ejecución usa su LOGIN nominal. No carga
// claves privadas de desarrollo ni altera cuerpos SQL.
func TestCT109PositivoPostgreSQL(t *testing.T) {
	if os.Getenv("VEC_CT109_PG_DESECHABLE") != "1" {
		t.Skip("requiere runner PostgreSQL 18 desechable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	dsn := os.Getenv("VEC_CT109_PG_DSN")
	cfg, err := pgxpool.ParseConfig(dsn)
	ct109OK(t, "configuración", err)
	if cfg.ConnConfig.Host != "/var/run/postgresql" || cfg.ConnConfig.User != "postgres" {
		t.Fatal("el ensayo exige socket del contenedor y preparador nominal")
	}
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	ct109OK(t, "preparador", err)
	defer admin.Close()
	var version int
	ct109OK(t, "PG18", admin.QueryRow(ctx, "SELECT current_setting('server_version_num')::int").Scan(&version))
	if version < 180000 || version >= 190000 {
		t.Fatal("requiere PostgreSQL 18")
	}
	var metadata, proof []byte
	ct109OK(t, "metadatos del fixture", admin.QueryRow(ctx, "SELECT datos FROM public.ct109_positivo_metadatos").Scan(&metadata))
	ct109OK(t, "comprobante CA6", admin.QueryRow(ctx, "SELECT comprobante FROM public.ct109_positivo_comprobante").Scan(&proof))
	var refs map[string]json.RawMessage
	ct109OK(t, "metadatos JSON", json.Unmarshal(metadata, &refs))
	ref := func(k string) string {
		var s string
		ct109OK(t, k, json.Unmarshal(refs[k], &s))
		if s == "" {
			t.Fatal("referencia ausente: " + k)
		}
		return s
	}
	reloj := relojContratacionTemporalDesarrollo{}
	pool := func(login, group string) *pgxpool.Pool {
		_, e := admin.Exec(ctx, "CREATE ROLE "+pgx.Identifier{login}.Sanitize()+" LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS")
		ct109OK(t, "login "+login, e)
		_, e = admin.Exec(ctx, "GRANT "+pgx.Identifier{group}.Sanitize()+" TO "+pgx.Identifier{login}.Sanitize()+" WITH ADMIN FALSE, INHERIT TRUE, SET FALSE")
		ct109OK(t, "membresía "+login, e)
		c := cfg.Copy()
		c.ConnConfig.User = login
		p, e := pgxpool.NewWithConfig(ctx, c)
		ct109OK(t, "pool "+login, e)
		t.Cleanup(p.Close)
		return p
	}
	runtime := pool("ct109_runtime_contexto", "vec_contexto_actor_v1_runtime")
	resolver, e := ca.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, runtime)
	if e != nil {
		ct109DiagnosticarRuntime(t, ctx, admin, runtime)
	}
	ct109OK(t, "resolutor contexto real", e)
	servicio, e := app.NuevoServicioContextoActorProductivoV2(resolver, ca.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj)
	ct109OK(t, "servicio contexto", e)
	autoridad, e := app.NuevaAutoridadContextoActorRegistradoV2(servicio)
	ct109OK(t, "autoridad contexto", e)
	revalidador, e := sesiones.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, pool("ct109_runtime_sesion", "vec_identidad_sesiones_v1_revalidador"))
	ct109OK(t, "revalidador sesión real", e)
	vinculo, resultado, e := core.CrearVinculoAutenticacionActorV2ConResultado(ctx, revalidador, core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: ref("autenticacion_ref"), SesionRef: ref("sesion_ref")}, autoridad, core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: ref("cuenta_ref"), Metodo: core.AuthMethodKerberos, Garantia: core.AuthAssuranceHigh}, PerfilActivoRef: ref("perfil_ref")}, reloj)
	ct109OK(t, "vínculo durable", e)
	datos, e := vinculo.Datos()
	ct109OK(t, "datos vínculo", e)
	org := ref("organizacion_ref")
	instantanea, e := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(datos.PrincipalID, datos.PerfilActivoRef, reloj.Ahora(), "ct109_lectura_sintetica", "Lectura sintética CT109", "ct109_lectura_sintetica", []core.ConcesionRol{
		{Accion: ct.AccionConsultarCuadroRRHH, ModuloID: ct.ModuloContratacion, TipoRecurso: ct.TipoRecursoCuadroRRHH, Finalidades: []string{ct.FinalidadConsultarCuadroRRHH}, GarantiaMinima: core.AuthAssuranceHigh},
		{Accion: ct.AccionConsultarDetalleRRHH, ModuloID: ct.ModuloContratacion, TipoRecurso: ct.TipoRecursoExpediente, Finalidades: []string{ct.FinalidadConsultarDetalleRRHH}, GarantiaMinima: core.AuthAssuranceHigh},
	}, []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{org}}, {Clave: "clase_ambito", Valores: []string{"organizacion"}}, {Clave: "ambito_ref", Valores: []string{org}}})
	ct109OK(t, "RBAC exacto", e)
	publicador := autoridadPostgreSQLDesarrollo{pool: admin, vinculo: vinculo, prefijoBloqueo: "ct109:fixture:", actoControlRol: "acto:ct109:rol", actoAsignacion: "acto:ct109:asignacion", actoSesion: "opr_ct109_positivo_00000000000001"}
	ct109OK(t, "publicación RBAC", publicador.PublicarInstantanea(ctx, instantanea))
	fuente, e := pg.NuevoAlmacenAutorizacion(pool("ct109_runtime_fuente", "vec_autorizacion_fuente"))
	ct109OK(t, "fuente RBAC", e)
	registro, e := pg.NuevoAlmacenAutorizacion(pool("ct109_runtime_registro", "vec_autorizacion_registro"))
	ct109OK(t, "registro V3", e)
	motivos := ct109Motivos{pool: pool("ct109_runtime_motivos", "vec_autorizacion_motivos_rrhh_resolutor")}
	m, e := motivos.ResolverMotivoCuadroRRHH(ctx, reloj.Ahora())
	ct109OK(t, "motivo cuadro", e)
	validador, e := pg.NuevoValidadorReferenciaMotivoPostgreSQLV2(pool("ct109_runtime_motivo_historico", "vec_autorizacion_motivos_evaluador"), m.CatalogoID)
	ct109OK(t, "validador motivos", e)
	pdp, e := app.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registro, registro, validador, reloj, seguridad.GeneradorReferenciasCriptograficas{}, app.ConfiguracionServicioAutorizacion{})
	ct109OK(t, "PDP V3", e)
	cuadro := &ct109EmisorNominal{t: t, admin: admin, pdp: pdp, audiencia: ct.AudienciaConsumoConsultaCuadroRRHHV3, sufijo: "cuadro"}
	detalle := &ct109EmisorNominal{t: t, admin: admin, pdp: pdp, audiencia: ct.AudienciaConsumoConsultaDetalleRRHHV3, sufijo: "detalle"}
	emisor, e := ct.NuevoEmisorMaterialConsultaRRHH(motivos, seguridad.GeneradorReferenciasCriptograficas{}, reloj, cuadro, detalle)
	ct109OK(t, "emisor RRHH", e)
	contexto, e := ct.NuevoContextoConsultaRRHH(ct.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}, org, reloj.Ahora())
	ct109OK(t, "contexto RRHH", e)
	lector := pool("ct109_runtime_lector", "vec_contratacion_temporal_consultor_rrhh_ambito")
	q, e := ct.NuevaSolicitudCuadroRRHH("", "", "", 100, "")
	ct109OK(t, "consulta cuadro", e)
	material, e := emisor.EmitirMaterialCuadroRRHH(ctx, contexto, q)
	ct109OK(t, "material auténtico cuadro", e)
	capacidad, e := ct.NuevaCapacidadConsultaCuadroRRHH(contexto, material, q, reloj.Ahora())
	ct109OK(t, "capacidad cuadro", e)
	orden, e := ct.NuevaOrdenConsultaCuadroRRHH(contexto, capacidad, q, reloj.Ahora())
	ct109OK(t, "orden cuadro", e)
	exportacion, e := orden.ExportacionParaSQL()
	ct109OK(t, "exportación cuadro", e)
	ct109Consultar(t, ctx, lector, org, proof, exportacion, "cuadro", ref("expediente_ref"))
	qd, e := ct.NuevaSolicitudDetalleRRHH(ref("expediente_ref"), 1)
	ct109OK(t, "consulta detalle", e)
	material, e = emisor.EmitirMaterialDetalleRRHH(ctx, contexto, qd)
	ct109OK(t, "material auténtico detalle", e)
	capacidad, e = ct.NuevaCapacidadConsultaDetalleRRHH(contexto, material, qd, reloj.Ahora())
	ct109OK(t, "capacidad detalle", e)
	od, e := ct.NuevaOrdenConsultaDetalleRRHH(contexto, capacidad, qd, reloj.Ahora())
	ct109OK(t, "orden detalle", e)
	exportacion, e = od.ExportacionParaSQL()
	ct109OK(t, "exportación detalle", e)
	ct109Consultar(t, ctx, lector, org, proof, exportacion, "detalle", ref("expediente_ref"))

	// Las concesiones nuevas son registros PDP durables. Solo las lecturas
	// confirmadas pueden conservar consumos V3, accesos CT y pruebas de resultado.
	emitirDetalle := func() vp.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
		mat, e := emisor.EmitirMaterialDetalleRRHH(ctx, contexto, qd)
		ct109OK(t, "material adverso real", e)
		cap, e := ct.NuevaCapacidadConsultaDetalleRRHH(contexto, mat, qd, reloj.Ahora())
		ct109OK(t, "capacidad adversa", e)
		orden, e := ct.NuevaOrdenConsultaDetalleRRHH(contexto, cap, qd, reloj.Ahora())
		ct109OK(t, "orden adversa", e)
		x, e := orden.ExportacionParaSQL()
		ct109OK(t, "exportación adversa", e)
		return x
	}
	contar := func() string {
		var n string
		ct109OK(t, "contadores durables", admin.QueryRow(ctx, `SELECT
   (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)::text||':'||
   (SELECT count(*) FROM vec_autorizacion_atestada_v3.atestacion_decision_v3)::text||':'||
   (SELECT count(*) FROM vec_contratacion_temporal.registro_acceso_rrhh)::text||':'||
   (SELECT count(*) FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2)::text||':'||
   (SELECT count(*) FROM vec_contratacion_temporal.vinculo_identidad_acceso_rrhh_v2)::text||':'||
   (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)::text`).Scan(&n))
		return n
	}

	antes := contar()
	mRollback := emitirDetalle()
	ct109OK(t, "lectura seguida de rollback real", ct109Ejecutar(ctx, lector, org, proof, mRollback, "detalle", ref("expediente_ref"), true, false))
	if contar() != antes {
		t.Fatal("rollback dejó efectos CT/V3")
	}
	mFirma := emitirDetalle()
	denegada := func(etapa string, err error) {
		var pgerr *pgconn.PgError
		if !errors.As(err, &pgerr) || pgerr.Code != "42501" {
			ct109OK(t, etapa, err)
			t.Fatal(etapa + ": no denegó con 42501")
		}
		if contar() != antes {
			t.Fatal(etapa + ": dejó efectos CT/V3")
		}
	}
	denegada("firma COSE alterada", ct109Ejecutar(ctx, lector, org, proof, mFirma, "detalle", ref("expediente_ref"), false, true))
	mRevocada := emitirDetalle()
	tx, e := admin.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	ct109OK(t, "revocación CA6", e)
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, "SET LOCAL ROLE vec_contexto_actor_v1_propietario")
	ct109OK(t, "propietario CA6", e)
	_, e = tx.Exec(ctx, `DO $revocar$ DECLARE v vec_contexto_actor_v1.vinculo_corporativo_versiones%ROWTYPE;
 BEGIN SELECT h.* INTO STRICT v FROM vec_contexto_actor_v1.vinculo_corporativo_actual a
 JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones h
 ON h.vinculo_corporativo_ref=a.vinculo_corporativo_ref AND h.version=a.version
 WHERE a.cuenta_ref='cta_corporativa_rrhh_000000000001';
 v.version:=v.version+1;v.estado:='revocado';
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones SELECT v.*;
 UPDATE vec_contexto_actor_v1.vinculo_corporativo_actual SET version=v.version
 WHERE cuenta_ref=v.cuenta_ref; END $revocar$`)
	ct109OK(t, "nueva revisión revocada", e)
	ct109OK(t, "commit revocación CA6", tx.Commit(ctx))
	denegada("CA6 revocada tras emisión", ct109Ejecutar(ctx, lector, org, proof, mRevocada, "detalle", ref("expediente_ref"), false, false))
	t.Log("CT109: cuadro y detalle positivos; firma alterada y CA6 revocada denegadas; rollback sin efectos CT/V3")

}

func ct109OK(t *testing.T, etapa string, err error) {
	t.Helper()
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) {
			t.Fatalf("%s: SQLSTATE %s", etapa, pe.Code)
		}
		t.Fatalf("%s: dependencia fallida (%T)", etapa, err)
	}
}

// Adaptador de transporte: PostgreSQL resuelve ambos motivos nominales.
type ct109Motivos struct{ pool *pgxpool.Pool }

func (m ct109Motivos) resolver(ctx context.Context, ahora time.Time, funcion string) (core.ReferenciaEntradaCatalogo, error) {
	var r core.ReferenciaEntradaCatalogo
	err := m.pool.QueryRow(ctx, "SELECT catalogo_id,catalogo_version,catalogo_huella_sha256,entrada_clave FROM vec_autorizacion."+funcion+"($1::timestamptz)", ahora).Scan(&r.CatalogoID, &r.CatalogoVersion, &r.CatalogoHuellaSHA256, &r.EntradaClave)
	return r, err
}
func (m ct109Motivos) ResolverMotivoCuadroRRHH(ctx context.Context, t time.Time) (core.ReferenciaEntradaCatalogo, error) {
	return m.resolver(ctx, t, "resolver_motivo_cuadro_rrhh_v1")
}
func (m ct109Motivos) ResolverMotivoDetalleRRHH(ctx context.Context, t time.Time) (core.ReferenciaEntradaCatalogo, error) {
	return m.resolver(ctx, t, "resolver_motivo_detalle_rrhh_v1")
}

func ct109Emisor(t *testing.T, ctx context.Context, admin *pgxpool.Pool, pdp vp.AutorizadorSolicitudLigadaV3, audiencia, sufijo string) *confianza.EmisorMaterialAutorizacionAtestadaV3 {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	desde, hasta := ahora.Add(-time.Minute), ahora.Add(time.Hour)
	publica, privada, e := ed25519.GenerateKey(rand.Reader)
	ct109OK(t, "Ed25519 efímera", e)
	t.Cleanup(func() { clear(privada) })
	claveID := "clave:ct109:" + sufijo
	hmacID := "capacidad:ct109:" + sufijo
	emisorID := "emisor:ct109:" + sufijo
	revision := "confianza:ct109:" + sufijo
	hmac := make([]byte, 32)
	_, e = rand.Read(hmac)
	ct109OK(t, "HMAC efímero", e)
	t.Cleanup(func() { clear(hmac) })
	hash := func(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
	raiz, e := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(claveID, 1, publica, audienciaAtestacionContratacionTemporalDesarrollo, confianza.EstadoClaveAtestacionAutorizacionV3Activa, desde, hasta, time.Time{})
	ct109OK(t, "raíz", e)
	var secuencia, orden uint64
	ct109OK(t, "orden configuración", admin.QueryRow(ctx, "SELECT coalesce(max(orden),0)+1 FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual").Scan(&secuencia))
	ct109OK(t, "orden clave", admin.QueryRow(ctx, "SELECT coalesce(max(orden),0)+1 FROM vec_autorizacion_atestada_v3.puntero_clave_emision").Scan(&orden))
	cfg, e := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(revision, secuencia, desde, hasta, raiz)
	ct109OK(t, "confianza", e)
	huella, e := cfg.HuellaSHA256ParaGobierno()
	ct109OK(t, "huella confianza", e)
	spki, e := x509.MarshalPKIXPublicKey(publica)
	ct109OK(t, "SPKI", e)
	clave, e := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(hmacID, orden, hmac, emisorID, audiencia, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, desde, hasta, time.Time{}, orden, hash(hmac))
	ct109OK(t, "clave capacidad", e)
	tx, e := admin.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	ct109OK(t, "gobierno", e)
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, "SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario")
	ct109OK(t, "propietario gobierno", e)
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version(clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref) VALUES($1,$8,$8,$2,$3,$2,$4,$5,$6,$7,$9)`, []any{hmacID, hash(hmac), hmac, emisorID, audiencia, desde, hasta, orden, "acto:ct109:clave:" + sufijo}},
		{`INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision(orden,clave_id,version,establecida_en,acto_ref)VALUES($1,$2,$1,$3,$4)`, []any{orden, hmacID, desde, "acto:ct109:puntero:" + sufijo}},
		{`INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version(clave_id,version,clave_publica_spki,huella_spki_sha256,valida_desde,valida_hasta,suite,audiencia_despliegue,acto_ref)VALUES($1,1,$2,$3,$4,$5,$6,$7,$8)`, []any{claveID, spki, hash(spki), desde, hasta, confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, audienciaAtestacionContratacionTemporalDesarrollo, "acto:ct109:raiz:" + sufijo}},
		{`INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version(revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)VALUES($1,$2,$3,$4,$5,$6)`, []any{revision, secuencia, huella, desde, hasta, "acto:ct109:confianza:" + sufijo}},
		{`INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz(configuracion_revision,raiz_clave_id,raiz_version)VALUES($1,$2,1)`, []any{revision, claveID}},
		{`INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual(orden,configuracion_revision,establecida_en,acto_ref)VALUES($1,$2,$3,$4)`, []any{secuencia, revision, desde, "acto:ct109:confianza-actual:" + sufijo}},
	} {
		_, e = tx.Exec(ctx, q.sql, q.args...)
		ct109OK(t, "publicación gobierno "+sufijo, e)
	}
	ct109OK(t, "commit gobierno", tx.Commit(ctx))
	reloj := relojContratacionTemporalDesarrollo{}
	at, e := app.NuevoServicioAtestacionesAutorizacionV3(core.CabeceraAtestacionAutorizacionV3{FormatoVersion: core.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: claveID, Audiencia: audienciaAtestacionContratacionTemporalDesarrollo}, &firmanteAtestacionAltaContratacionTemporalDesarrollo{claveID: claveID, privada: privada, reloj: reloj})
	ct109OK(t, "atestador real", e)
	ver, e := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(cfg, reloj)
	ct109OK(t, "verificador real", e)
	cap, e := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
	ct109OK(t, "capacidades reales", e)
	em, e := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(pdp, at, ver, cap)
	ct109OK(t, "material real", e)
	return em
}

func ct109Ejecutar(ctx context.Context, pool *pgxpool.Pool, org string, proof []byte, m vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, tipo, expediente string, rollback, alterarFirma bool) error {
	args := []any{org, string(proof), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), strconv.FormatUint(m.PersonaVersion(), 10), strconv.FormatUint(m.PerfilVersion(), 10), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()}
	consulta := `SELECT contenido_canonico,acceso_ref,auditoria_vec_ref,consumo_vec_huella_sha256 FROM vec_contratacion_temporal.consultar_cuadro_rrhh_ambito_v1(ROW($1::text,'organizacion',$1::text)::vec_contratacion_temporal.alcance_consulta_rrhh_v1,ROW('','','',100::smallint,'')::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,$2::jsonb,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,$9::bytea,$10::bytea,$11::bytea,$12::bytea)`
	if tipo == "detalle" {
		args = append(args, expediente)
		consulta = `SELECT contenido_canonico,acceso_ref,auditoria_vec_ref,consumo_vec_huella_sha256 FROM vec_contratacion_temporal.consultar_detalle_rrhh_ambito_v1(ROW($1::text,'organizacion',$1::text)::vec_contratacion_temporal.alcance_consulta_rrhh_v1,ROW($13::text,1::numeric)::vec_contratacion_temporal.consulta_detalle_rrhh_v1,$2::jsonb,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,$9::bytea,$10::bytea,$11::bytea,$12::bytea)`
	}
	if alterarFirma {
		firma := append([]byte(nil), m.SobreCOSESign1()...)
		firma[len(firma)-1] ^= 1
		args[9] = firma
		defer clear(firma)
	}
	tx, e := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var contenido []byte
	var acceso, auditoria, consumo string
	if e = tx.QueryRow(ctx, consulta, args...).Scan(&contenido, &acceso, &auditoria, &consumo); e != nil {
		return e
	}
	if len(contenido) == 0 || acceso == "" || auditoria == "" || len(consumo) != 64 {
		return errors.New("recibo CT109 incompleto")
	}
	if rollback {
		return tx.Rollback(ctx)
	}
	return tx.Commit(ctx)
}

// Publica antes de la primera emisión de cada audiencia, con las lecturas
// anteriores ya confirmadas. La autorización, atestación y verificación se
// delegan íntegramente en los servicios V3 existentes.
type ct109EmisorNominal struct {
	t                 *testing.T
	admin             *pgxpool.Pool
	pdp               vp.AutorizadorSolicitudLigadaV3
	audiencia, sufijo string
	real              *confianza.EmisorMaterialAutorizacionAtestadaV3
}

func (e *ct109EmisorNominal) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	if e.real == nil {
		e.real = ct109Emisor(e.t, ctx, e.admin, e.pdp, e.audiencia, e.sufijo)
	}
	return e.real.EmitirMaterialAutorizacionAtestadaV3(ctx, s, r)
}

func ct109Consultar(t *testing.T, ctx context.Context, pool *pgxpool.Pool, org string, proof []byte, m vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, tipo, expediente string) {
	t.Helper()
	ct109OK(t, "CT109 "+tipo, ct109Ejecutar(ctx, pool, org, proof, m, tipo, expediente, false, false))
}

// Solo se ejecuta tras el rechazo del constructor nominal. El preparador lee
// metadatos de la efímera; nunca reintenta la autoridad usando su privilegio.
// La salida está restringida a etiquetas constantes, booleanos y conteos.
func ct109DiagnosticarRuntime(t *testing.T, ctx context.Context, admin, runtime *pgxpool.Pool) {
	t.Helper()
	var identidadExacta, roleNone bool
	if err := runtime.QueryRow(ctx, `SELECT session_user=current_user, current_setting('role')='none'`).Scan(&identidadExacta, &roleNone); err != nil {
		t.Log("CT109 diagnóstico: contexto de conexión no disponible")
	} else {
		t.Logf("CT109 diagnóstico: identidad_exacta=%t current_role_none=%t", identidadExacta, roleNone)
	}
	var encoded []byte
	if err := admin.QueryRow(ctx, ct109DiagnosticoRuntimeSQL).Scan(&encoded); err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			t.Logf("CT109 diagnóstico: consulta metadatos SQLSTATE=%s", pgerr.Code)
		} else {
			t.Log("CT109 diagnóstico: metadatos no disponibles")
		}
		return
	}
	var valores map[string]json.RawMessage
	if json.Unmarshal(encoded, &valores) != nil {
		t.Log("CT109 diagnóstico: resultado no estructurado")
		return
	}
	claves := make([]string, 0, len(valores))
	for k := range valores {
		claves = append(claves, k)
	}
	sort.Strings(claves)
	for _, k := range claves {
		// Incluso un resultado SQL inesperado queda cerrado: no imprimimos texto,
		// JSON arbitrario, nombres de objetos, errores originales ni parámetros.
		var n int64
		if json.Unmarshal(valores[k], &n) == nil && string(valores[k]) != "null" {
			t.Logf("CT109 diagnóstico: %s=%d", k, n)
			continue
		}
		var b bool
		if json.Unmarshal(valores[k], &b) == nil && string(valores[k]) != "null" {
			t.Logf("CT109 diagnóstico: %s=%t", k, b)
			continue
		}
		t.Log("CT109 diagnóstico: valor fuera de contrato")
	}
}

const ct109DiagnosticoRuntimeSQL = `WITH
 l AS (SELECT * FROM pg_roles WHERE rolname='ct109_runtime_contexto'),
 g AS (SELECT * FROM pg_roles WHERE rolname='vec_contexto_actor_v1_runtime'),
 b AS (SELECT oid FROM pg_database WHERE datname=current_database()),
 n AS (SELECT oid FROM pg_namespace WHERE nspname='vec_contexto_actor_v1'),
 f AS (SELECT ARRAY[
  to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()')::oid,
  to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)')::oid,
  to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)')::oid] ids),
 x AS (SELECT l.oid login,g.oid grupo,b.oid base,n.oid esquema,f.ids FROM l,g,b,n,f)
SELECT jsonb_build_object(
 'objetos_resueltos',(SELECT count(*)=1 FROM x),
 'funciones_resueltas',(SELECT cardinality(ids)=3 AND array_position(ids,NULL) IS NULL FROM f),
 'login_seguro',(SELECT rolcanlogin AND rolinherit AND NOT rolsuper AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolreplication AND NOT rolbypassrls AND rolconfig IS NULL FROM l),
 'grupo_seguro',(SELECT NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolreplication AND NOT rolbypassrls AND rolconfig IS NULL FROM g),
 'membresias_directas',(SELECT count(*) FROM pg_auth_members m,x WHERE m.member=x.login),
 'membresia_exacta',(SELECT count(*)=1 FROM pg_auth_members m,x WHERE m.member=x.login AND m.roleid=x.grupo AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option),
 'membresias_login_ajenas',(SELECT count(*) FROM pg_roles r,x WHERE r.oid<>x.login AND r.oid<>x.grupo AND pg_has_role(x.login,r.oid,'MEMBER')),
 'membresias_grupo',(SELECT count(*) FROM pg_roles r,x WHERE r.oid<>x.grupo AND pg_has_role(x.grupo,r.oid,'MEMBER')),
 'configuraciones_roles',(SELECT count(*) FROM pg_db_role_setting s,x WHERE s.setrole IN(x.login,x.grupo)),
 'acl_predeterminadas',(SELECT count(*) FROM pg_default_acl d LEFT JOIN LATERAL aclexplode(coalesce(d.defaclacl,'{}'::aclitem[])) a ON true CROSS JOIN x WHERE d.defaclrole IN(x.login,x.grupo) OR a.grantee IN(x.login,x.grupo) OR a.grantor IN(x.login,x.grupo)),
 'politicas_roles',(SELECT count(*) FROM pg_policy p,x WHERE x.login=ANY(p.polroles) OR x.grupo=ANY(p.polroles)),
 'dependencias_login',(SELECT count(*) FROM pg_shdepend d,x WHERE d.refclassid='pg_authid'::regclass AND d.refobjid=x.login),
 'dependencias_grupo',(SELECT count(*) FROM pg_shdepend d,x WHERE d.refclassid='pg_authid'::regclass AND d.refobjid=x.grupo),
 'dependencias_grupo_exactas',(SELECT count(*)=5 AND bool_and(d.deptype='a' AND d.objsubid=0 AND ((d.classid='pg_database'::regclass AND d.objid=x.base) OR (d.classid='pg_namespace'::regclass AND d.objid=x.esquema) OR (d.classid='pg_proc'::regclass AND d.objid=ANY(x.ids)))) FROM pg_shdepend d,x WHERE d.refclassid='pg_authid'::regclass AND d.refobjid=x.grupo),
 'acl_base_exacta',(SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable) FROM pg_database d CROSS JOIN LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a CROSS JOIN x WHERE d.oid=x.base AND a.grantee=x.grupo),
 'acl_esquema_exacta',(SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable) FROM pg_namespace d CROSS JOIN LATERAL aclexplode(coalesce(d.nspacl,acldefault('n',d.nspowner))) a CROSS JOIN x WHERE d.oid=x.esquema AND a.grantee=x.grupo),
 'acl_funciones_exacta',(SELECT count(*)=3 AND count(DISTINCT p.oid)=3 AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a CROSS JOIN x WHERE p.oid=ANY(x.ids) AND a.grantee=x.grupo),
 'acl_efectiva_minima',(SELECT vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(login,base,esquema,ids) FROM x)
) || CASE WHEN (SELECT vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(login,base,esquema,ids) FROM x) IS TRUE THEN '{}'::jsonb ELSE jsonb_build_object(
 'efectivo_public_usage',(SELECT has_schema_privilege(x.login,n.oid,'USAGE') FROM pg_namespace n,x WHERE n.nspname='public'),
 'efectivo_public_funciones',(SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace CROSS JOIN x WHERE n.nspname='public' AND has_function_privilege(x.login,p.oid,'EXECUTE')),
 'efectivo_connect',(SELECT has_database_privilege(login,base,'CONNECT') FROM x),
 'efectivo_create_base',(SELECT has_database_privilege(login,base,'CREATE') FROM x),
 'efectivo_temporary',(SELECT has_database_privilege(login,base,'TEMPORARY') FROM x),
 'efectivo_esquemas',(SELECT count(*) FROM pg_namespace n,x WHERE n.nspname<>'information_schema' AND n.nspname!~'^pg_' AND ((n.oid<>x.esquema AND has_schema_privilege(x.login,n.oid,'USAGE')) OR has_schema_privilege(x.login,n.oid,'CREATE'))),
 'efectivo_tablas_columnas',(SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN x WHERE n.nspname<>'information_schema' AND n.nspname!~'^pg_' AND c.relkind IN('r','p','v','m','f') AND (has_table_privilege(x.login,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN') OR has_any_column_privilege(x.login,c.oid,'SELECT,INSERT,UPDATE,REFERENCES'))),
 'efectivo_secuencias',(SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN x WHERE n.nspname<>'information_schema' AND n.nspname!~'^pg_' AND c.relkind='S' AND has_sequence_privilege(x.login,c.oid,'USAGE,SELECT,UPDATE')),
 'efectivo_funciones_ajenas',(SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace CROSS JOIN x WHERE n.nspname<>'information_schema' AND n.nspname!~'^pg_' AND p.oid<>ALL(x.ids) AND has_function_privilege(x.login,p.oid,'EXECUTE')),
 'efectivo_tipos',(SELECT count(*) FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace CROSS JOIN x WHERE n.nspname<>'information_schema' AND n.nspname!~'^pg_' AND t.typtype IN('c','d','e','m','r') AND has_type_privilege(x.login,t.oid,'USAGE') AND NOT (t.typtype='c' AND t.typacl IS NULL AND NOT has_schema_privilege(x.login,n.oid,'USAGE') AND EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=t.typrelid AND c.relkind IN('r','p','v','m','f')))),
 'efectivo_largeobjects',(SELECT count(*) FROM pg_largeobject_metadata l,x WHERE has_largeobject_privilege(x.login,l.oid,'SELECT,UPDATE')),
 'efectivo_fdw',(SELECT count(*) FROM pg_foreign_data_wrapper f,x WHERE has_foreign_data_wrapper_privilege(x.login,f.oid,'USAGE')),
 'efectivo_servidores',(SELECT count(*) FROM pg_foreign_server s,x WHERE has_server_privilege(x.login,s.oid,'USAGE')),
 'efectivo_lenguajes',(SELECT count(*) FROM pg_language l,x WHERE l.oid>=16384 AND has_language_privilege(x.login,l.oid,'USAGE')),
 'efectivo_tablespaces',(SELECT count(*) FROM pg_tablespace t,x WHERE t.oid>=16384 AND has_tablespace_privilege(x.login,t.oid,'CREATE')),
 'efectivo_parametros',(SELECT count(*) FROM pg_parameter_acl a,x WHERE has_parameter_privilege(x.login,a.parname,'SET,ALTER SYSTEM'))
) END`
