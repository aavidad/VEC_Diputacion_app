package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	variableDSNRegistroV2Fuente    = "VEC_POSTGRES_TEST_REGISTRO_V2_FUENTE_DSN"
	variableDSNRegistroV2Registro  = "VEC_POSTGRES_TEST_REGISTRO_V2_REGISTRO_DSN"
	variableDSNRegistroV2Evaluador = "VEC_POSTGRES_TEST_REGISTRO_V2_EVALUADOR_DSN"
	variableDSNRegistroV2Admin     = "VEC_POSTGRES_TEST_REGISTRO_V2_ADMIN_DSN"
)

type generadorCorrelacionRegistroV2Prueba string

func (g generadorCorrelacionRegistroV2Prueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return string(g), nil
}

type registroDenegacionesSolicitudV2PostgreSQLPrueba struct{}

func (registroDenegacionesSolicitudV2PostgreSQLPrueba) RegistrarDenegacionAutorizacionSolicitudLigadaV2(
	context.Context,
	ports.OrdenRegistroDecisionAutorizacionSolicitudLigadaV2,
) error {
	return nil
}

// Solo prepara una decision real mediante el servicio. La prueba temporal la
// presenta despues directamente a la funcion SQL para controlar el bloqueo sin
// que los timeouts defensivos del adaptador oculten la carrera que se audita.
type registroConcesionesSolicitudV2SinPersistenciaPostgreSQLPrueba struct{}

func (registroConcesionesSolicitudV2SinPersistenciaPostgreSQLPrueba) RegistrarDecisionSolicitudLigadaV2SiInstantaneaVigente(
	context.Context,
	ports.OrdenRegistroDecisionAutorizacionSolicitudLigadaV2,
) error {
	return nil
}

type registroSolicitudV2PostgreSQLConMutacion struct {
	destino ports.RegistroDecisionesAutorizacionSolicitudLigadaV2
	mutar   func(context.Context) error
	unaVez  sync.Once
	err     error
}

func (r *registroSolicitudV2PostgreSQLConMutacion) RegistrarDecisionSolicitudLigadaV2SiInstantaneaVigente(
	ctx context.Context,
	orden ports.OrdenRegistroDecisionAutorizacionSolicitudLigadaV2,
) error {
	r.unaVez.Do(func() { r.err = r.mutar(ctx) })
	if r.err != nil {
		return r.err
	}
	return r.destino.RegistrarDecisionSolicitudLigadaV2SiInstantaneaVigente(ctx, orden)
}

func TestIntegracionRegistroDecisionSolicitudLigadaV2PostgreSQL(t *testing.T) {
	dsnFuente := os.Getenv(variableDSNRegistroV2Fuente)
	dsnRegistro := os.Getenv(variableDSNRegistroV2Registro)
	dsnEvaluador := os.Getenv(variableDSNRegistroV2Evaluador)
	dsnAdmin := os.Getenv(variableDSNRegistroV2Admin)
	if dsnFuente == "" || dsnRegistro == "" || dsnEvaluador == "" || dsnAdmin == "" {
		t.Skip("prueba de registro V2 omitida; ejecute deploy/postgresql/autorizacion/probar_integracion.sh")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	poolFuente := abrirPoolRegistroV2Prueba(t, ctx, dsnFuente)
	defer poolFuente.Close()
	poolRegistro := abrirPoolRegistroV2Prueba(t, ctx, dsnRegistro)
	defer poolRegistro.Close()
	poolEvaluador := abrirPoolRegistroV2Prueba(t, ctx, dsnEvaluador)
	defer poolEvaluador.Close()
	poolAdmin := abrirPoolRegistroV2Prueba(t, ctx, dsnAdmin)
	defer poolAdmin.Close()

	almacenFuente, err := NuevoAlmacenAutorizacion(poolFuente)
	if err != nil {
		t.Fatalf("crear fuente V2: %v", err)
	}
	almacenRegistro, err := NuevoAlmacenAutorizacion(poolRegistro)
	if err != nil {
		t.Fatalf("crear registro V2: %v", err)
	}
	validadorMotivo, err := NuevoValidadorReferenciaMotivoPostgreSQLV2(
		poolEvaluador,
		"motivos_autorizacion",
	)
	if err != nil {
		t.Fatalf("crear evaluador de motivo: %v", err)
	}

	verificarACLRegistroSolicitudV2PostgreSQL(
		t, ctx, poolFuente, poolRegistro, poolEvaluador,
	)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	motivo := publicarMotivoRegistroV2PostgreSQLPrueba(t, ctx, poolAdmin, ahora)
	fixture := nuevaFixtureAutorizacionPostgreSQL("registro_v2", ahora)
	sembrarFixtureAutorizacionPostgreSQL(t, ctx, poolAdmin, fixture)

	servicio := nuevoServicioRegistroSolicitudV2PostgreSQLPrueba(
		t, almacenFuente, almacenRegistro, validadorMotivo, ahora,
		"decision:postgres:solicitud-v2:registrada",
	)
	solicitud := solicitudRegistroV2PostgreSQLPrueba(t, fixture, motivo,
		"correlacion_0123456789abcdef0123456789abcdef")
	decision, err := servicio.ExigirSolicitudLigadaV2(ctx, solicitud)
	if err != nil {
		t.Fatalf("registrar decision V2 real: %v", err)
	}
	verificarDecisionSolicitudV2DurablePostgreSQL(
		t, ctx, poolAdmin, decision, motivo,
	)

	orden, err := ports.NuevaOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(
		decision,
		motivo,
	)
	if err != nil {
		t.Fatalf("reconstruir orden nominal: %v", err)
	}
	if err = almacenRegistro.RegistrarDecisionSolicitudLigadaV2SiInstantaneaVigente(
		ctx,
		orden,
	); !errors.Is(err, ports.ErrVersionAutorizacionYaExiste) {
		t.Fatalf("replay de decision V2 no rechazado: %v", err)
	}
	verificarEntradasHostilesRegistroV2PostgreSQL(t, ctx, poolRegistro, decision, motivo)

	t.Run("expiracion del motivo durante bloqueo impide registrar", func(t *testing.T) {
		casoAhora := time.Now().UTC().Truncate(time.Microsecond)
		caso := nuevaFixtureAutorizacionPostgreSQL("motivo_expira_v2", casoAhora)
		sembrarFixtureAutorizacionPostgreSQL(t, ctx, poolAdmin, caso)
		instanteMotivo := time.Now().UTC().Truncate(time.Microsecond)
		expira := instanteMotivo.Add(3 * time.Second)
		motivoExpirable := publicarMotivoExpirableRegistroV2PostgreSQLPrueba(
			t, ctx, poolAdmin, instanteMotivo, expira,
		)
		validadorExpirable, err := NuevoValidadorReferenciaMotivoPostgreSQLV2(
			poolEvaluador,
			motivoExpirable.CatalogoID,
		)
		if err != nil {
			t.Fatalf("crear evaluador de motivo expirable: %v", err)
		}
		preparador := nuevoServicioRegistroSolicitudV2PostgreSQLPrueba(
			t,
			almacenFuente,
			registroConcesionesSolicitudV2SinPersistenciaPostgreSQLPrueba{},
			validadorExpirable,
			casoAhora,
			"decision:postgres:solicitud-v2:motivo-expirado-en-bloqueo",
		)
		solicitudExpirable := solicitudRegistroV2PostgreSQLPrueba(
			t,
			caso,
			motivoExpirable,
			"correlacion_11111111111111111111111111111111",
		)
		decisionExpirable, err := preparador.ExigirSolicitudLigadaV2(
			ctx,
			solicitudExpirable,
		)
		if err != nil {
			t.Fatalf("preparar decision con motivo expirable: %v", err)
		}
		decisionCanonica, motivoCanonico, err :=
			serializarDecisionSolicitudLigadaV2PostgreSQL(
				decisionExpirable,
				motivoExpirable,
			)
		if err != nil {
			t.Fatalf("serializar decision expirable: %v", err)
		}
		defer borrarBytesAutorizacionPostgreSQL(decisionCanonica, motivoCanonico)

		_, vinculo, err := contextoYVinculoAutenticacionPostgreSQLPrueba(caso)
		if err != nil {
			t.Fatalf("obtener vinculo para bloqueo: %v", err)
		}
		datosVinculo, err := vinculo.Datos()
		if err != nil {
			t.Fatalf("obtener sesion para bloqueo: %v", err)
		}
		bloqueo, err := poolAdmin.Begin(ctx)
		if err != nil {
			t.Fatalf("iniciar bloqueo de sesion: %v", err)
		}
		defer revertirTransaccionPostgreSQL(bloqueo)
		var uno int
		err = bloqueo.QueryRow(ctx, `
			SELECT 1
			FROM vec_autorizacion.control_sesion_actual_v1
			WHERE sesion_ref=$1
			FOR UPDATE`,
			datosVinculo.SesionRef,
		).Scan(&uno)
		if err != nil || uno != 1 {
			t.Fatalf("bloquear sesion actual: uno=%d err=%v", uno, err)
		}

		type resultadoRegistro struct {
			registrada bool
			err        error
		}
		resultado := make(chan resultadoRegistro, 1)
		go func() {
			var registrada bool
			errConsulta := poolRegistro.QueryRow(ctx, `
				SELECT vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
					$1::bytea, $2::bytea
				)`,
				decisionCanonica,
				motivoCanonico,
			).Scan(&registrada)
			resultado <- resultadoRegistro{registrada: registrada, err: errConsulta}
		}()
		if !esperarBloqueoRegistroV2PostgreSQLPrueba(t, ctx, poolAdmin) {
			t.Fatal("el registro V2 no alcanzo el bloqueo de sesion")
		}
		if espera := time.Until(expira.Add(100 * time.Millisecond)); espera > 0 {
			time.Sleep(espera)
		}
		if err = bloqueo.Commit(ctx); err != nil {
			t.Fatalf("liberar bloqueo tras expirar motivo: %v", err)
		}
		obtenido := <-resultado
		if obtenido.err != nil || obtenido.registrada {
			t.Fatalf(
				"motivo expirado durante espera registrado: registrada=%t err=%v",
				obtenido.registrada,
				obtenido.err,
			)
		}
		var filas int
		err = poolAdmin.QueryRow(ctx, `
			SELECT count(*)
			FROM vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
			WHERE decision_ref=$1`,
			decisionExpirable.DecisionRef,
		).Scan(&filas)
		if err != nil || filas != 0 {
			t.Fatalf("quedo evidencia con motivo expirado: filas=%d err=%v", filas, err)
		}
	})

	t.Run("retirada concurrente del motivo impide registrar", func(t *testing.T) {
		caso := nuevaFixtureAutorizacionPostgreSQL("motivo_retirado_v2", ahora)
		sembrarFixtureAutorizacionPostgreSQL(t, ctx, poolAdmin, caso)
		registro := &registroSolicitudV2PostgreSQLConMutacion{
			destino: almacenRegistro,
			mutar: func(ctx context.Context) error {
				return retirarMotivoRegistroV2PostgreSQLPrueba(ctx, poolAdmin, motivo)
			},
		}
		servicioRetirada := nuevoServicioRegistroSolicitudV2PostgreSQLPrueba(
			t, almacenFuente, registro, validadorMotivo, ahora,
			"decision:postgres:solicitud-v2:motivo-retirado",
		)
		solicitudRetirada := solicitudRegistroV2PostgreSQLPrueba(t, caso, motivo,
			"correlacion_fedcba9876543210fedcba9876543210")
		_, err := servicioRetirada.ExigirSolicitudLigadaV2(ctx, solicitudRetirada)
		if !errors.Is(err, ports.ErrInstantaneaAutorizacionObsoleta) {
			t.Fatalf("retirada concurrente no cerro el registro: %v", err)
		}
	})
}

func abrirPoolRegistroV2Prueba(
	t *testing.T,
	ctx context.Context,
	dsn string,
) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil || pool.Ping(ctx) != nil {
		t.Fatalf("abrir pool de registro V2: %v", err)
	}
	return pool
}

func verificarACLRegistroSolicitudV2PostgreSQL(
	t *testing.T,
	ctx context.Context,
	poolFuente, poolRegistro, poolEvaluador *pgxpool.Pool,
) {
	t.Helper()
	const firma = "vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(bytea,bytea)"
	for nombre, caso := range map[string]struct {
		pool    *pgxpool.Pool
		permiso bool
	}{
		"fuente": {poolFuente, false}, "registro": {poolRegistro, true},
		"evaluador": {poolEvaluador, false},
	} {
		var ejecutar bool
		var tablas int
		err := caso.pool.QueryRow(ctx, `
			SELECT has_function_privilege(current_user, $1, 'EXECUTE'),
			       (SELECT count(*) FROM pg_catalog.pg_class AS c
			        JOIN pg_catalog.pg_namespace AS n ON n.oid=c.relnamespace
			        WHERE n.nspname='vec_autorizacion' AND c.relkind IN ('r','p')
			          AND has_table_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))`,
			firma,
		).Scan(&ejecutar, &tablas)
		if err != nil || ejecutar != caso.permiso || tablas != 0 {
			t.Fatalf("ACL %s insegura: ejecutar=%t tablas=%d err=%v", nombre, ejecutar, tablas, err)
		}
	}
	_, err := poolFuente.Exec(ctx, `
		SELECT vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
			'00'::bytea, '00'::bytea
		)`)
	exigirPrivilegioDenegadoRegistroV2PostgreSQL(t, err, "fuente ejecuto registro V2")
	_, err = poolEvaluador.Exec(ctx, `
		SELECT vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(
			'00'::bytea, '00'::bytea
		)`)
	exigirPrivilegioDenegadoRegistroV2PostgreSQL(t, err, "evaluador ejecuto registro V2")
}

func publicarMotivoRegistroV2PostgreSQLPrueba(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	ahora time.Time,
) domain.ReferenciaEntradaCatalogo {
	t.Helper()
	referencia := domain.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: strings.Repeat("7", 64),
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
	entradas, err := json.Marshal([]map[string]any{{
		"clave": referencia.EntradaClave, "vigente_desde": ahora.Add(-time.Hour),
		"vigente_hasta": ahora.Add(time.Hour),
	}})
	if err != nil {
		t.Fatal(err)
	}
	var publicada bool
	err = pool.QueryRow(ctx, `
		SELECT vec_autorizacion.publicar_motivos_autorizacion_v2(
			$1,1,$2,$3,1,$4,$5,$6::jsonb
		)`,
		"evento_0123456789abcdef0123456789abcdef", strings.Repeat("6", 64),
		referencia.CatalogoID, referencia.CatalogoHuellaSHA256,
		ahora.Add(-2*time.Hour), entradas,
	).Scan(&publicada)
	if err != nil || !publicada {
		t.Fatalf("publicar motivo V2: publicada=%t err=%v", publicada, err)
	}
	return referencia
}

func publicarMotivoExpirableRegistroV2PostgreSQLPrueba(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	ahora time.Time,
	expira time.Time,
) domain.ReferenciaEntradaCatalogo {
	t.Helper()
	referencia := domain.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion_expirable",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: strings.Repeat("3", 64),
		EntradaClave:         "motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	entradas, err := json.Marshal([]map[string]any{{
		"clave": referencia.EntradaClave, "vigente_desde": ahora.Add(-time.Hour),
		"vigente_hasta": expira,
	}})
	if err != nil {
		t.Fatal(err)
	}
	var publicada bool
	err = pool.QueryRow(ctx, `
		SELECT vec_autorizacion.publicar_motivos_autorizacion_v2(
			$1,2,$2,$3,1,$4,$5,$6::jsonb
		)`,
		"evento_11111111111111111111111111111111", strings.Repeat("2", 64),
		referencia.CatalogoID, referencia.CatalogoHuellaSHA256,
		ahora.Add(-time.Minute), entradas,
	).Scan(&publicada)
	if err != nil || !publicada {
		t.Fatalf("publicar motivo V2 expirable: publicada=%t err=%v", publicada, err)
	}
	return referencia
}

func esperarBloqueoRegistroV2PostgreSQLPrueba(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
) bool {
	t.Helper()
	limite := time.Now().Add(time.Second)
	for time.Now().Before(limite) {
		var esperando bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_catalog.pg_stat_activity
				WHERE datname=current_database()
				  AND usename='vec_autorizacion_registro_prueba'
				  AND wait_event_type='Lock'
				  AND query LIKE '%registrar_decision_solicitud_ligada_v2_si_vigente%'
			)`,
		).Scan(&esperando)
		if err != nil {
			t.Fatalf("observar bloqueo del registro V2: %v", err)
		}
		if esperando {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func solicitudRegistroV2PostgreSQLPrueba(
	t *testing.T,
	fixture fixtureAutorizacionPostgreSQL,
	motivo domain.ReferenciaEntradaCatalogo,
	correlacionValor string,
) domain.SolicitudAutorizacionLigadaV2 {
	t.Helper()
	actor, vinculo, err := contextoYVinculoAutenticacionPostgreSQLPrueba(fixture)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionRegistroV2Prueba(correlacionValor),
	)
	if err != nil {
		t.Fatalf("generar correlacion V2: %v", err)
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV2(
		domain.DatosSolicitudAutorizacionLigadaV2{
			ContextoActor: actor, VinculoAutenticacionActor: vinculo,
			ReferenciaMotivo: motivo, Accion: "bolsa.merito.revisar",
			Recurso: domain.RecursoAutorizable{
				Referencia: "merito:" + fixture.prefijo, ModuloID: "bolsa", Tipo: "merito",
				Ambitos:   map[string]string{"provincia": "granada"},
				Atributos: map[string]string{"estado": "presentado"},
			},
			Finalidad: "gestion_bolsa", Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatalf("crear solicitud V2: %v", err)
	}
	return solicitud
}

func nuevoServicioRegistroSolicitudV2PostgreSQLPrueba(
	t *testing.T,
	almacenFuente ports.FuenteAutorizacion,
	registro ports.RegistroDecisionesAutorizacionSolicitudLigadaV2,
	validador ports.ValidadorReferenciaMotivoAutorizacionV2,
	ahora time.Time,
	decisionRef string,
) *application.ServicioAutorizacionSolicitudLigadaV2 {
	t.Helper()
	servicio, err := application.NuevoServicioAutorizacionSolicitudLigadaV2(
		almacenFuente, registro, registroDenegacionesSolicitudV2PostgreSQLPrueba{},
		validador, relojPostgreSQLPrueba{ahora: ahora},
		generadorPostgreSQLPrueba(decisionRef),
		application.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatalf("crear servicio V2 PostgreSQL: %v", err)
	}
	return servicio
}

func verificarDecisionSolicitudV2DurablePostgreSQL(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	decision domain.DecisionAutorizacion,
	motivo domain.ReferenciaEntradaCatalogo,
) {
	t.Helper()
	var decisionCanonica, motivoCanonico, documentoV2 []byte
	var huella string
	err := pool.QueryRow(ctx, `
		SELECT decision_canonica, motivo_canonico, documento_v2, huella_decision_sha256
		FROM vec_autorizacion.decision_autorizacion_solicitud_ligada_v2
		WHERE decision_ref=$1`, decision.DecisionRef,
	).Scan(&decisionCanonica, &motivoCanonico, &documentoV2, &huella)
	if err != nil {
		t.Fatalf("leer decision V2 durable: %v", err)
	}
	esperada, motivoEsperado, err := serializarDecisionSolicitudLigadaV2PostgreSQL(decision, motivo)
	if err != nil || string(decisionCanonica) != string(esperada) ||
		string(motivoCanonico) != string(motivoEsperado) ||
		!documentosJSONPostgreSQLIguales(documentoV2, esperada) {
		t.Fatalf("evidencia V2 durable divergente: %v", err)
	}
	if huella != huellaSHA256BytesPostgreSQLPrueba(esperada) {
		t.Fatal("huella durable de decision V2 incorrecta")
	}
	var filasV1 int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM vec_autorizacion.decision_autorizacion WHERE decision_ref=$1`, decision.DecisionRef).Scan(&filasV1); err != nil || filasV1 != 0 {
		t.Fatalf("decision V2 reinterpretada como V1: filas=%d err=%v", filasV1, err)
	}
}

func verificarEntradasHostilesRegistroV2PostgreSQL(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	decision domain.DecisionAutorizacion,
	motivo domain.ReferenciaEntradaCatalogo,
) {
	t.Helper()
	_, motivoCanonico, err := serializarDecisionSolicitudLigadaV2PostgreSQL(decision, motivo)
	if err != nil {
		t.Fatal(err)
	}
	casos := [][]byte{nil, []byte(`{}`), []byte(`[]`), []byte(`{"esquema":"v1"}`)}
	for indice, contenido := range casos {
		var registrada bool
		err = pool.QueryRow(ctx, `
			SELECT vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente($1,$2)`,
			contenido, motivoCanonico,
		).Scan(&registrada)
		if err != nil || registrada {
			t.Fatalf("entrada hostil %d no fue rechazada limpiamente: %t %v", indice, registrada, err)
		}
	}
	_, err = pool.Exec(ctx, `SELECT count(*) FROM vec_autorizacion.decision_autorizacion_solicitud_ligada_v2`)
	exigirPrivilegioDenegadoRegistroV2PostgreSQL(t, err, "registro leyo tabla V2")
}

func retirarMotivoRegistroV2PostgreSQLPrueba(
	ctx context.Context,
	pool *pgxpool.Pool,
	motivo domain.ReferenciaEntradaCatalogo,
) error {
	var retirada bool
	err := pool.QueryRow(ctx, `
		SELECT vec_autorizacion.retirar_motivos_autorizacion_v2(
			$1,3,$2,$3,1,$4,$5,clock_timestamp()
		)`,
		"evento_fedcba9876543210fedcba9876543210", strings.Repeat("5", 64),
		motivo.CatalogoID, motivo.CatalogoHuellaSHA256, strings.Repeat("4", 64),
	).Scan(&retirada)
	if err != nil {
		return err
	}
	if !retirada {
		return errors.New("retirada de motivo V2 rechazada")
	}
	return nil
}

func exigirPrivilegioDenegadoRegistroV2PostgreSQL(t *testing.T, err error, operacion string) {
	t.Helper()
	var errorPG *pgconn.PgError
	if !errors.As(err, &errorPG) || errorPG.Code != "42501" {
		t.Fatalf("%s: %v", operacion, err)
	}
}

func huellaSHA256BytesPostgreSQLPrueba(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}
