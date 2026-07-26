package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const firmaWrapperO404D = "vec_autorizacion." +
	"registrar_decision_cobertura_contratacion_temporal_v1" +
	"(bytea,bytea,numeric,numeric,jsonb)"

type resultadoWrapperO404D struct {
	Rama           string
	Concedida      bool
	Codigo         string
	DecisionRef    string
	CorrelacionRef string
	Organizacion   string
	Expediente     string
	Version        int64
	Reserva        string
	ContextoHuella string
	DecisionHuella string
	OrdenHuella    string
	LoteHuella     *string
	PruebaVinculo  string
	RegistradaEn   time.Time
	RevalidadaEn   *time.Time
}

func probarWrapperVECO404D(
	t *testing.T,
	ctx context.Context,
	cluster *pgxpool.Pool,
	dsn string,
	raiz string,
	sufijo string,
) {
	t.Helper()
	nombreBase := "vec_o404d_wrapper_" + sufijo
	crearBasePreparacionDecisionCobertura(t, ctx, cluster, nombreBase)
	admin := abrirBasePreparacionDecisionCobertura(
		t,
		ctx,
		dsn,
		nombreBase,
		"postgres",
	)
	defer func() {
		admin.Close()
		eliminarBasePreparacionDecisionCobertura(t, cluster, nombreBase)
		eliminarRolesWrapperO404D(t, cluster)
	}()
	instalarAutorizacionRealO404D(t, ctx, admin, raiz)
	probarContratoWrapperO404D(t, ctx, admin)
	probarACLDependenciaYCicloWrapperO404D(t, ctx, admin, raiz)
}

func instalarAutorizacionRealO404D(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	raiz string,
) {
	t.Helper()
	instalarContextoActorRealO404D(t, ctx, admin, raiz)
	for _, ruta := range []string{
		"deploy/postgresql/autorizacion/roles_up.sql",
		"deploy/postgresql/autorizacion/roles_v2_up.sql",
		"deploy/postgresql/autorizacion/migraciones/" +
			"000001_autorizacion.up.sql",
		"deploy/postgresql/ejecucion_documental_v4/" +
			"migraciones_autorizacion/" +
			"000002_vinculo_autenticacion_actor_actual.up.sql",
		"deploy/postgresql/autorizacion/migraciones/" +
			"000003_proyeccion_motivos_autorizacion_v2.up.sql",
		"deploy/postgresql/autorizacion/migraciones/" +
			"000004_registro_decisiones_solicitud_ligada_v2.up.sql",
	} {
		ejecutarArchivoPreparacionDecisionCobertura(
			t,
			ctx,
			admin,
			filepath.Join(raiz, ruta),
		)
	}
	for _, ruta := range []string{
		"deploy/postgresql/autorizacion/migraciones/" +
			"000005_registro_decisiones_contexto_actor_v3.up.sql",
		"deploy/postgresql/autorizacion/migraciones/" +
			"000006_funcion_registro_decisiones_contexto_actor_v3.up.sql",
		"deploy/postgresql/autorizacion/migraciones/" +
			"000007_revalidacion_viva_decision_contexto_actor_v3.up.sql",
	} {
		ejecutarArchivoPreparacionDecisionCobertura(
			t,
			ctx,
			admin,
			filepath.Join(raiz, ruta),
		)
	}
	instalarFixtureAutorizacionCoberturaO404D(t, ctx, admin, raiz)
	for numero := 1; numero <= 4; numero++ {
		rutas, err := filepath.Glob(filepath.Join(
			raiz,
			"deploy/postgresql/contratacion_temporal/"+
				"migraciones_autorizacion/"+
				fmt.Sprintf("%06d_*.up.sql", numero),
		))
		if err != nil || len(rutas) != 1 {
			t.Fatalf(
				"migración autorización CT %06d no unívoca: %v / %v",
				numero,
				rutas,
				err,
			)
		}
		ejecutarArchivoPreparacionDecisionCobertura(
			t,
			ctx,
			admin,
			rutas[0],
		)
	}
}

func instalarContextoActorRealO404D(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	raiz string,
) {
	t.Helper()
	if _, err := admin.Exec(ctx, `
		DO $b$
		BEGIN
		  EXECUTE format(
		    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
		    current_database()
		  );
		END
		$b$;
		REVOKE ALL ON SCHEMA public FROM PUBLIC`); err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{
		"deploy/postgresql/contexto_actor_v1/roles_up.sql",
		"deploy/postgresql/contexto_actor_v1/migraciones/" +
			"000001_contexto_actor_v1.up.sql",
		"deploy/postgresql/contexto_actor_v1/migraciones/" +
			"000002_acreditacion_uso_registro_contexto_actor_v2.up.sql",
		"deploy/postgresql/contexto_actor_v1/pruebas_sql/" +
			"fixtures_sinteticos.sql",
		"deploy/postgresql/autorizacion/pruebas_sql/" +
			"fixture_contexto_actor_v3.sql",
	} {
		ejecutarArchivoPreparacionDecisionCobertura(
			t,
			ctx,
			admin,
			filepath.Join(raiz, ruta),
		)
	}
	_, err := admin.Exec(ctx, `
		CREATE ROLE vec_contexto_o404d_runtime
		  LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
		  NOREPLICATION NOBYPASSRLS;
		GRANT vec_contexto_actor_v1_runtime
		  TO vec_contexto_o404d_runtime
		  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE`)
	if err != nil {
		t.Fatal(err)
	}
	conexion, err := admin.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conexion.Release()
	if _, err := conexion.Exec(ctx, `
		SET SESSION AUTHORIZATION vec_contexto_o404d_runtime`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = conexion.Exec(
			context.Background(),
			"RESET SESSION AUTHORIZATION",
		)
	}()
	tx, err := conexion.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var filas int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		  FROM vec_contexto_actor_v1
		       .resolver_y_registrar_contexto_actor_v2(
		         'oca_registro_v3_000000000000000000000000',
		         'rca_registro_v3_000000000000000000000000',
		         'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa',
		         'prf_sintetico_cccccccccccccccccccccccc',
		         'certificado',
		         'alto',
		         clock_timestamp()
		       )`).Scan(&filas); err != nil || filas != 1 {
		t.Fatalf("registro ContextoActor real: filas=%d / %v", filas, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func instalarFixtureAutorizacionCoberturaO404D(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	raiz string,
) {
	t.Helper()
	ruta := filepath.Join(
		raiz,
		"deploy/postgresql/autorizacion/pruebas_sql/"+
			"fixture_autorizacion_contexto_actor_v3.sql",
	)
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	fixture := string(contenido)
	reemplazos := map[string]string{
		"'accion','consultar','modulo_id','bolsa'," +
			"'tipo_recurso','expediente'": "" +
			"'accion','contratacion_temporal.cobertura.decidir'," +
			"'modulo_id','contratacion_temporal'," +
			"'tipo_recurso','decision_cobertura_gobernada'",
		"'finalidades',jsonb_build_array('gestion')": "" +
			"'finalidades',jsonb_build_array('gestion_cobertura')",
	}
	for original, nuevo := range reemplazos {
		if !strings.Contains(fixture, original) {
			t.Fatalf("fixture VEC cambió: falta %q", original)
		}
		fixture = strings.ReplaceAll(fixture, original, nuevo)
	}
	if _, err := admin.Exec(ctx, fixture); err != nil {
		t.Fatalf("fixture VEC cobertura: %v", err)
	}
}

func nuevaDecisionWrapperO404D(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	referencia string,
	concedida bool,
	version int,
) ([]byte, []byte, map[string]any) {
	t.Helper()
	var autenticada, emitidaSesion, revalidadaSesion, validaSesion time.Time
	if err := admin.QueryRow(ctx, `
		SELECT s.autenticacion_verificada_en,s.sesion_emitida_en,
		       c.sesion_revalidada_en,c.sesion_valida_hasta
		  FROM vec_autorizacion.sesion_autenticacion_v1 s
		  JOIN vec_autorizacion.control_sesion_v1 c USING (sesion_ref)
		 WHERE s.sesion_ref=
		       'ses_registro_v3_0000000000000000000000'`).Scan(
		&autenticada,
		&emitidaSesion,
		&revalidadaSesion,
		&validaSesion,
	); err != nil {
		t.Fatal(err)
	}
	var huellaContexto, huellaManifiesto string
	if err := admin.QueryRow(ctx, `
		SELECT huella_sha256,manifiesto_procedencia_huella_sha256
		  FROM vec_contexto_actor_v1.registros_contexto
		 WHERE registro_contexto_ref=
		       'rca_registro_v3_000000000000000000000000'`).Scan(
		&huellaContexto,
		&huellaManifiesto,
	); err != nil {
		t.Fatal(err)
	}
	motivo := []byte(
		`{"esquema":"vec.autorizacion.motivo.v2.referencia-opaca-catalogada",` +
			`"referencia":{"catalogo_id":"motivos_v3","catalogo_version":1,` +
			`"catalogo_huella_sha256":"` + strings.Repeat("9", 64) + `",` +
			`"entrada_clave":"motivo_33333333333333333333333333333333"}}`,
	)
	huellaMotivo := sha256.Sum256(motivo)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	vinculo := map[string]any{
		"esquema":                              "vec.autenticacion-actor.vinculo.v2.contexto-registrado",
		"bloque_version":                       2,
		"autenticacion_ref":                    "aut_registro_v3_0000000000000000000000",
		"autenticacion_huella_sha256":          strings.Repeat("5", 64),
		"asercion_ref":                         "ase_registro_v3_0000000000000000000000",
		"sesion_ref":                           "ses_registro_v3_0000000000000000000000",
		"control_sesion_ref":                   "cse_registro_v3_0000000000000000000000",
		"control_sesion_revision":              1,
		"control_sesion_huella_sha256":         strings.Repeat("7", 64),
		"cuenta_ref":                           "cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa",
		"cuenta_ordinaria_ref":                 "cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa",
		"principal_id":                         "per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb",
		"perfil_activo_ref":                    "prf_sintetico_cccccccccccccccccccccccc",
		"cuenta_privilegiada":                  false,
		"superficie":                           "interna_corporativa",
		"metodo_observado":                     "certificado",
		"garantia_observada":                   "alto",
		"politica_garantia_ref":                "pga_registro_v3_0000000000000000000000",
		"politica_garantia_huella_sha256":      strings.Repeat("6", 64),
		"autenticacion_verificada_en":          instanteCanonWrapperO404D(autenticada),
		"sesion_emitida_en":                    instanteCanonWrapperO404D(emitidaSesion),
		"sesion_valida_hasta":                  instanteCanonWrapperO404D(validaSesion),
		"sesion_revalidada_en":                 instanteCanonWrapperO404D(revalidadaSesion),
		"registro_contexto_ref":                "rca_registro_v3_000000000000000000000000",
		"contexto_actor_esquema":               "vec.contexto-actor.vinculado.v2",
		"contexto_actor_ref":                   "vca_sintetico_dddddddddddddddddddddddd",
		"contexto_actor_version":               2,
		"contexto_actor_cuenta_version":        2,
		"contexto_actor_huella_sha256":         huellaContexto,
		"manifiesto_procedencia_huella_sha256": huellaManifiesto,
		"autoridad_efectiva":                   "autoridad_maestra_acreditada",
	}
	codigo := "concedida"
	rama := "concedida"
	if !concedida {
		codigo = "accion_no_concedida"
		rama = "denegada"
	}
	reserva := "reserva:wrapper-o404d:" + referencia
	orden := strings.Repeat("d", 64)
	var lote *string
	if concedida {
		huella := strings.Repeat("e", 64)
		lote = &huella
	}
	contextoRecurso := huellaContextoRecursoO404D(
		rama,
		"organizacion:wrapper-o404d",
		"expediente:wrapper-o404d",
		uint64(version),
		reserva,
		orden,
		lote,
	)
	huellaCorrelacion := sha256.Sum256(
		[]byte("correlacion:" + referencia),
	)
	decision := map[string]any{
		"esquema":                        "vec.autorizacion.decision.v3.solicitud-ligada.actor-v2",
		"bloque_version":                 3,
		"decision_ref":                   "decision:wrapper-o404d:" + referencia,
		"concedida":                      concedida,
		"codigo":                         codigo,
		"principal_id":                   "per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb",
		"perfil_activo_ref":              "prf_sintetico_cccccccccccccccccccccccc",
		"accion":                         "contratacion_temporal.cobertura.decidir",
		"recurso_ref":                    reserva,
		"modulo_id":                      "contratacion_temporal",
		"tipo_recurso":                   "decision_cobertura_gobernada",
		"contexto_recurso_huella_sha256": contextoRecurso,
		"finalidad":                      "gestion_cobertura",
		"correlacion_ref": "correlacion_" +
			hex.EncodeToString(huellaCorrelacion[:])[:32],
		"esquema_huella_solicitud":                   "vec.autorizacion.solicitud.v3.efectiva-minimizada.actor-v2",
		"solicitud_huella_sha256":                    strings.Repeat("c", 64),
		"esquema_huella_motivo":                      "vec.autorizacion.motivo.v2.referencia-opaca-catalogada",
		"motivo_huella_sha256":                       hex.EncodeToString(huellaMotivo[:]),
		"vinculo_autenticacion_actor":                vinculo,
		"asignacion_ref":                             "asignacion:registro_v3:v1",
		"asignacion_huella_sha256":                   strings.Repeat("4", 64),
		"version_rol_ref":                            "rol:registro_v3:v1",
		"version_rol_huella_sha256":                  strings.Repeat("2", 64),
		"control_vigencia_version_rol_ref":           "rol:registro_v3:v1",
		"control_vigencia_version_rol_revision":      1,
		"control_vigencia_version_rol_huella_sha256": strings.Repeat("3", 64),
		"revision_catalogo_politicas":                1,
		"catalogo_politicas_huella_sha256":           "4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945",
		"politicas_evaluadas":                        []any{},
		"politicas_aplicables":                       []any{},
		"garantia_minima":                            "alto",
		"campos_permitidos":                          []any{"estado"},
		"obligaciones":                               []any{"auditar"},
		"emitida_en":                                 instanteCanonWrapperO404D(ahora),
		"valida_hasta":                               instanteCanonWrapperO404D(ahora.Add(2 * time.Minute)),
	}
	contenido, err := json.Marshal(decision)
	if err != nil {
		t.Fatal(err)
	}
	var canon []byte
	if err := admin.QueryRow(ctx, `
		SELECT vec_autorizacion.decision_contexto_actor_v3_canonica(
		  $1::jsonb
		)`, contenido).Scan(&canon); err != nil {
		t.Fatal(err)
	}
	var valida bool
	if err := admin.QueryRow(ctx, `
		SELECT vec_autorizacion.decision_contexto_actor_v3_valida(
		  $1::jsonb
		)`, contenido).Scan(&valida); err != nil || !valida {
		t.Fatalf("decisión VEC de cobertura inválida: %t / %v", valida, err)
	}
	return canon, motivo, decision
}

func instanteCanonWrapperO404D(instante time.Time) string {
	return instante.UTC().Format("2006-01-02T15:04:05.000000Z")
}

func efectoWrapperO404D(
	decision map[string]any,
	rama string,
	version int,
) map[string]any {
	efecto := map[string]any{
		"rama":                           rama,
		"accion":                         decision["accion"],
		"organizacion_ref":               "organizacion:wrapper-o404d",
		"expediente_ref":                 "expediente:wrapper-o404d",
		"version_expediente":             version,
		"reserva_ref":                    decision["recurso_ref"],
		"decision_ref":                   decision["decision_ref"],
		"correlacion_ref":                decision["correlacion_ref"],
		"finalidad":                      decision["finalidad"],
		"contexto_recurso_huella_sha256": decision["contexto_recurso_huella_sha256"],
		"huella_orden_sha256":            strings.Repeat("d", 64),
	}
	if rama == "concedida" {
		efecto["lote_huella_sha256"] = strings.Repeat("e", 64)
	}
	return efecto
}

func invocarWrapperO404D(
	ctx context.Context,
	admin *pgxpool.Pool,
	decision []byte,
	motivo []byte,
	efecto map[string]any,
) (resultadoWrapperO404D, bool, error) {
	contenido, err := json.Marshal(efecto)
	if err != nil {
		return resultadoWrapperO404D{}, false, err
	}
	tx, err := admin.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return resultadoWrapperO404D{}, false, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, `
		SET LOCAL statement_timeout='15s';
		SET LOCAL idle_in_transaction_session_timeout='20s';
		SET LOCAL ROLE vec_contratacion_temporal_propietario`); err != nil {
		return resultadoWrapperO404D{}, false, err
	}
	var resultado resultadoWrapperO404D
	err = tx.QueryRow(ctx, `
		SELECT rama,concedida,codigo,decision_ref,correlacion_ref,
		       organizacion_ref,expediente_ref,version_expediente,
		       reserva_ref,contexto_recurso_huella_sha256,
		       decision_huella_sha256,huella_orden_sha256,
		       lote_huella_sha256,prueba_vinculo_sha256,
		       registrada_en,revalidada_en
		  FROM vec_autorizacion
		       .registrar_decision_cobertura_contratacion_temporal_v1(
		         $1,$2,2,2,$3::jsonb
		       )`, decision, motivo, contenido).Scan(
		&resultado.Rama,
		&resultado.Concedida,
		&resultado.Codigo,
		&resultado.DecisionRef,
		&resultado.CorrelacionRef,
		&resultado.Organizacion,
		&resultado.Expediente,
		&resultado.Version,
		&resultado.Reserva,
		&resultado.ContextoHuella,
		&resultado.DecisionHuella,
		&resultado.OrdenHuella,
		&resultado.LoteHuella,
		&resultado.PruebaVinculo,
		&resultado.RegistradaEn,
		&resultado.RevalidadaEn,
	)
	if err == pgx.ErrNoRows {
		return resultadoWrapperO404D{}, false, tx.Commit(ctx)
	}
	if err != nil {
		return resultadoWrapperO404D{}, false, err
	}
	return resultado, true, tx.Commit(ctx)
}

func probarContratoWrapperO404D(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
) {
	t.Helper()
	decision, motivo, documento := nuevaDecisionWrapperO404D(
		t,
		ctx,
		admin,
		"concedida",
		true,
		10,
	)
	efecto := efectoWrapperO404D(documento, "concedida", 10)
	concedida, encontrada, err := invocarWrapperO404D(
		ctx,
		admin,
		decision,
		motivo,
		efecto,
	)
	if err != nil || !encontrada || !concedida.Concedida ||
		concedida.Rama != "concedida" || concedida.LoteHuella == nil ||
		concedida.RevalidadaEn == nil ||
		concedida.ContextoHuella !=
			efecto["contexto_recurso_huella_sha256"] {
		var registradas int
		_ = admin.QueryRow(ctx, `
			SELECT count(*)
			  FROM vec_autorizacion
			       .decision_concedida_contexto_actor_v3`).Scan(
			&registradas,
		)
		t.Fatalf(
			"concesión wrapper inválida: %#v / %v / base=%d",
			concedida,
			err,
			registradas,
		)
	}
	comprobarPruebaWrapperO404D(t, concedida)
	for _, caso := range []struct {
		nombre string
		mutar  func(map[string]any)
	}{
		{"organizacion", func(v map[string]any) {
			v["organizacion_ref"] = "organizacion:wrapper-o404d:mutada"
		}},
		{"expediente", func(v map[string]any) {
			v["expediente_ref"] = "expediente:wrapper-o404d:mutado"
		}},
		{"version", func(v map[string]any) {
			v["version_expediente"] = 11
		}},
		{"reserva", func(v map[string]any) {
			v["reserva_ref"] = "reserva:wrapper-o404d:mutada"
		}},
		{"orden", func(v map[string]any) {
			v["huella_orden_sha256"] = strings.Repeat("1", 64)
		}},
		{"lote", func(v map[string]any) {
			v["lote_huella_sha256"] = strings.Repeat("2", 64)
		}},
		{"contexto", func(v map[string]any) {
			v["contexto_recurso_huella_sha256"] = strings.Repeat("3", 64)
		}},
		{"sin_lote", func(v map[string]any) {
			delete(v, "lote_huella_sha256")
		}},
		{"version_uno", func(v map[string]any) {
			v["version_expediente"] = 1
		}},
		{"correlacion", func(v map[string]any) {
			v["correlacion_ref"] = "correlacion:wrapper-o404d:mutada"
		}},
		{"clave_extra", func(v map[string]any) {
			v["inesperada"] = true
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			copia := clonarLoteFixtureConsumoC1O404D(t, efecto)
			caso.mutar(copia)
			_, encontrada, err := invocarWrapperO404D(
				ctx,
				admin,
				decision,
				motivo,
				copia,
			)
			if err != nil || encontrada {
				t.Fatalf("mutación wrapper aceptada: %t / %v", encontrada, err)
			}
		})
	}
	decisionDenegada, motivoDenegado, documentoDenegado :=
		nuevaDecisionWrapperO404D(t, ctx, admin, "denegada", false, 100)
	efectoDenegado := efectoWrapperO404D(documentoDenegado, "denegada", 100)
	denegada, encontrada, err := invocarWrapperO404D(
		ctx,
		admin,
		decisionDenegada,
		motivoDenegado,
		efectoDenegado,
	)
	if err != nil || !encontrada || denegada.Concedida ||
		denegada.Rama != "denegada" || denegada.LoteHuella != nil ||
		denegada.RevalidadaEn != nil {
		t.Fatalf("denegación wrapper inválida: %#v / %v", denegada, err)
	}
	comprobarPruebaWrapperO404D(t, denegada)
	conLote := clonarLoteFixtureConsumoC1O404D(t, efectoDenegado)
	conLote["lote_huella_sha256"] = strings.Repeat("e", 64)
	if _, encontrada, err := invocarWrapperO404D(
		ctx,
		admin,
		decisionDenegada,
		motivoDenegado,
		conLote,
	); err != nil || encontrada {
		t.Fatalf("denegación con lote aceptada: %t / %v", encontrada, err)
	}
}

func comprobarPruebaWrapperO404D(
	t *testing.T,
	resultado resultadoWrapperO404D,
) {
	t.Helper()
	canon := &canonConsumoC1O404D{}
	for _, prueba := range []string{
		resultado.DecisionHuella,
		resultado.ContextoHuella,
		resultado.OrdenHuella,
	} {
		contenido, err := hex.DecodeString(prueba)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = canon.Write(contenido)
	}
	if resultado.LoteHuella != nil {
		contenido, err := hex.DecodeString(*resultado.LoteHuella)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = canon.Write(contenido)
	}
	canon.texto(resultado.DecisionRef)
	canon.texto(resultado.CorrelacionRef)
	canon.instante(resultado.RegistradaEn)
	if resultado.RevalidadaEn == nil {
		canon.entero64(0)
	} else {
		canon.instante(*resultado.RevalidadaEn)
	}
	huella := sha256.Sum256(canon.Bytes())
	if resultado.PruebaVinculo != hex.EncodeToString(huella[:]) {
		t.Fatal("prueba criptográfica del wrapper divergente")
	}
}

func probarACLDependenciaYCicloWrapperO404D(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	raiz string,
) {
	t.Helper()
	comprobarACLWrapperO404D(t, ctx, admin)
	downBase := filepath.Join(
		raiz,
		"deploy/postgresql/autorizacion/migraciones/"+
			"000007_revalidacion_viva_decision_contexto_actor_v3.down.sql",
	)
	contenido, err := os.ReadFile(downBase)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, string(contenido)); err == nil ||
		codigoPostgreSQLO404D(err) != "2BP01" {
		t.Fatalf("wrapper no retuvo dependencia base VEC: %v", err)
	}
	down := rutaMigracionAutorizacionWrapperO404D(t, raiz, "down")
	up := rutaMigracionAutorizacionWrapperO404D(t, raiz, "up")
	ejecutarArchivoPreparacionDecisionCobertura(t, ctx, admin, down)
	var existe bool
	if err := admin.QueryRow(ctx, `
		SELECT to_regprocedure($1) IS NOT NULL`, firmaWrapperO404D).Scan(
		&existe,
	); err != nil || existe {
		t.Fatalf("down wrapper incompleto: %t / %v", existe, err)
	}
	ejecutarArchivoPreparacionDecisionCobertura(t, ctx, admin, up)
	comprobarACLWrapperO404D(t, ctx, admin)
}

func comprobarACLWrapperO404D(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
) {
	t.Helper()
	var propietario, runtime, sinPublico, configurada, ownerCerrado bool
	err := admin.QueryRow(ctx, `
		SELECT has_function_privilege(
		         'vec_contratacion_temporal_propietario',$1,'EXECUTE'
		       ),
		       has_function_privilege(
		         'vec_contratacion_temporal_ejecutor',$1,'EXECUTE'
		       ),
		       NOT EXISTS (
		         SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
		          WHERE a.grantee=0 AND a.privilege_type='EXECUTE'
		       ),
		       p.proconfig @> ARRAY[
		         'search_path=pg_catalog',
		         'TimeZone=UTC',
		         'lock_timeout=2s'
		       ],
		       p.proowner='vec_autorizacion_propietario'::regrole
		       AND NOT r.rolcanlogin AND NOT r.rolsuper
		  FROM pg_catalog.pg_proc p
		  JOIN pg_catalog.pg_roles r ON r.oid=p.proowner
		 WHERE p.oid=$1::regprocedure`,
		firmaWrapperO404D,
	).Scan(
		&propietario,
		&runtime,
		&sinPublico,
		&configurada,
		&ownerCerrado,
	)
	if err != nil || !propietario || runtime || !sinPublico ||
		!configurada || !ownerCerrado {
		t.Fatalf(
			"ACL wrapper inválida: %t/%t/%t/%t/%t / %v",
			propietario,
			runtime,
			sinPublico,
			configurada,
			ownerCerrado,
			err,
		)
	}
}

func rutaMigracionAutorizacionWrapperO404D(
	t *testing.T,
	raiz string,
	direccion string,
) string {
	t.Helper()
	rutas, err := filepath.Glob(filepath.Join(
		raiz,
		"deploy/postgresql/contratacion_temporal/"+
			"migraciones_autorizacion/000004_*o4_04d."+
			direccion+".sql",
	))
	if err != nil || len(rutas) != 1 {
		t.Fatalf("wrapper O4-04D %s no unívoco: %v / %v", direccion, rutas, err)
	}
	return rutas[0]
}

func eliminarRolesWrapperO404D(
	t *testing.T,
	cluster *pgxpool.Pool,
) {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	_, err := cluster.Exec(ctx, `
		DROP ROLE IF EXISTS vec_contexto_o404d_runtime;
		DROP ROLE IF EXISTS vec_autorizacion_motivos_evaluador;
		DROP ROLE IF EXISTS vec_autorizacion_motivos_proyector;
		DROP ROLE IF EXISTS vec_autorizacion_registro;
		DROP ROLE IF EXISTS vec_autorizacion_fuente;
		DROP ROLE IF EXISTS vec_autorizacion_migrador;
		DROP ROLE IF EXISTS vec_autorizacion_propietario;
		DROP ROLE IF EXISTS vec_contexto_actor_v1_runtime;
		DROP ROLE IF EXISTS vec_contexto_actor_v1_migrador;
		DROP ROLE IF EXISTS vec_contexto_actor_v1_propietario`)
	if err != nil {
		t.Errorf("retirar roles wrapper O4-04D: %v", err)
	}
}
