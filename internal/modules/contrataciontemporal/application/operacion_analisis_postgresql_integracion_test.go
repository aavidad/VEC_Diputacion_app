package application

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	adaptadorpostgres "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type entradaIntegracionAnalisisO304 struct {
	Caso                 string                           `json:"caso"`
	Ahora                string                           `json:"ahora"`
	DecisionPlantillaB64 string                           `json:"decision_plantilla_b64"`
	MotivoB64            string                           `json:"motivo_b64"`
	ContextoB64          string                           `json:"contexto_b64"`
	ManifiestoB64        string                           `json:"manifiesto_b64"`
	ManifiestoHuella     string                           `json:"manifiesto_huella_sha256"`
	AutoridadEfectiva    string                           `json:"autoridad_efectiva"`
	ResueltoEn           string                           `json:"resuelto_en"`
	AltaB64              string                           `json:"alta_b64"`
	SellosB64            string                           `json:"sellos_b64"`
	EfectoHuella         string                           `json:"efecto_huella_sha256"`
	ClaveID              string                           `json:"clave_id"`
	ClaveVersion         uint64                           `json:"clave_version"`
	RevisionGobierno     uint64                           `json:"revision_gobierno"`
	HuellaGobierno       string                           `json:"huella_gobierno_sha256"`
	EmisorID             string                           `json:"emisor_id"`
	AudienciaConsumo     string                           `json:"audiencia_consumo"`
	ClaveHMACB64         string                           `json:"clave_hmac_b64"`
	ClaveValidaDesde     string                           `json:"clave_valida_desde"`
	ClaveValidaHasta     string                           `json:"clave_valida_hasta"`
	RevisionConfianza    string                           `json:"revision_confianza"`
	SecuenciaConfianza   uint64                           `json:"secuencia_confianza"`
	RaizClaveID          string                           `json:"raiz_clave_id"`
	RaizVersion          uint64                           `json:"raiz_version"`
	AudienciaDespliegue  string                           `json:"audiencia_despliegue"`
	Politicas            []dominiovec.PoliticaRestrictiva `json:"politicas"`
	RevisionCatalogo     uint64                           `json:"revision_catalogo"`
	HuellaCatalogo       string                           `json:"huella_catalogo_sha256"`
	AsignacionID         string                           `json:"asignacion_id"`
	AsignacionVersion    int                              `json:"asignacion_version"`
	PersonaVersion       uint64                           `json:"persona_version"`
	PerfilVersion        uint64                           `json:"perfil_version"`
}

type decisionPlantillaIntegracionO304 struct {
	Vinculo dominiovec.DatosVinculoAutenticacionActorV2 `json:"vinculo_autenticacion_actor"`
}

type motivoIntegracionO304 struct {
	Referencia dominiovec.ReferenciaEntradaCatalogo `json:"referencia"`
}

func TestOperacionAnalisisPostgreSQLRecorreOrdenGoReal(t *testing.T) {
	ruta := os.Getenv("VEC_O304_VECTOR_ENTRADA")
	dsnRuntime := os.Getenv("VEC_O304_DSN_RUNTIME")
	dsnAdministrador := os.Getenv("VEC_O304_DSN_ADMIN")
	if ruta == "" || dsnRuntime == "" || dsnAdministrador == "" {
		t.Skip("solo se ejecuta desde la integración PostgreSQL O3-04")
	}
	entrada := leerEntradaIntegracionAnalisisO304(t, ruta)
	ahora := parsearInstanteIntegracionO304(t, entrada.Ahora)
	contexto, motivo := reconstruirContextoIntegracionO304(
		t, entrada, ahora,
	)
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	runtime := abrirPoolIntegracionO304(t, ctx, dsnRuntime)
	defer runtime.Close()
	administrador := abrirPoolIntegracionO304(t, ctx, dsnAdministrador)
	defer administrador.Close()
	var agregadoJSON string
	if err := administrador.QueryRow(ctx, `
		SELECT agregado_json::text
		  FROM vec_contratacion_temporal.expediente_version_integral
		 WHERE expediente_ref='expediente:ct:o205:go_analisis'
		   AND version=1`,
	).Scan(&agregadoJSON); err != nil {
		t.Fatal(err)
	}
	var agregadoInicial domain.Expediente
	decodificadorAgregado := json.NewDecoder(strings.NewReader(agregadoJSON))
	decodificadorAgregado.DisallowUnknownFields()
	if err := decodificadorAgregado.Decode(&agregadoInicial); err != nil {
		t.Fatalf("agregado inicial SQL no se puede restaurar en Go: %v", err)
	}
	if err := agregadoInicial.Validar(); err != nil {
		t.Fatalf("agregado inicial SQL no supera el dominio Go: %v", err)
	}
	preparador, err := adaptadorpostgres.
		NuevoPreparadorOperacionAnalisisPostgreSQL(runtime)
	if err != nil {
		t.Fatal(err)
	}
	preparadorObservado := &preparadorObservadoIntegracionO304{
		delegado: preparador,
	}
	transaccion, err := adaptadorpostgres.
		NuevaTransaccionOperacionesAnalisisPostgreSQL(runtime)
	if err != nil {
		t.Fatal(err)
	}
	transaccionObservada := &transaccionObservadaIntegracionO304{
		delegada: transaccion,
	}
	funcionales := datosFuncionalesAnalisisSinteticos()
	funcionales.CategoriaRef = "categoria:auxiliar"
	funcionales.CausaClave = "acumulacion_tareas"
	funcionales.Periodo = domain.PeriodoPrevisto{
		Inicio: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		Fin:    time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC),
	}
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	solicitud := SolicitudRegistrarAnalisis{
		AutenticacionRef:  vinculo.AutenticacionRef,
		SesionRef:         vinculo.SesionRef,
		PerfilRef:         vinculo.PerfilActivoRef,
		OrganizacionRef:   "organizacion:dipgra",
		ExpedienteRef:     "expediente:ct:o205:go_analisis",
		VersionEsperada:   1,
		ClaveIdempotencia: "77777777-8888-4999-8aaa-bbbbbbbbbbbb",
		ArtefactoRef:      "artefacto:analisis-go-o304",
		DatosFuncionales:  funcionales,
	}
	autorizador := &autorizadorIntegracionAnalisisO304{
		ahora: ahora, entrada: entrada, motivo: motivo,
		administrador: administrador,
	}
	contextos := &resolutorContextoDoble{contexto: contexto}
	artefactos := &preparadorArtefactoAnalisisDoble{
		delegado: nuevoPreparadorArtefactoAnalisisO3AplicacionPruebaParaOrganizacion(
			t,
			ahora,
			solicitud.OrganizacionRef,
		),
	}
	sellador := &selladorOperacionAnalisisDobleSaneado{}
	politicas := &resolutorPoliticaOperacionAnalisisDobleSaneado{
		motivoAutorizacion: motivo,
	}
	servicio, err := NuevoServicioOperacionAnalisis(
		contextos,
		artefactos,
		sellador,
		preparadorObservado,
		politicas,
		&generadorReferenciasDoble{
			correlacion: "correlacion_1234567890abcdef1234567890abcdef",
		},
		autorizador,
		&relojMutable{instante: ahora},
		transaccionRespuestaPerdidaIntegracionO304{
			delegada: transaccionObservada,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer restaurarAsignacionIntegracionO304(t, ctx, administrador)
	_, err = servicio.Registrar(ctx, solicitud)
	if !errors.Is(err, ErrDependenciaOperacionAnalisisNoDisponible) {
		var reservas, decisiones, versiones int
		_ = administrador.QueryRow(ctx, `
			SELECT
			  (SELECT count(*) FROM
			     vec_contratacion_temporal.reserva_operacion_analisis
			    WHERE expediente_ref=$1),
			  (SELECT count(*) FROM
			     vec_autorizacion.decision_concedida_contexto_actor_v3
			    WHERE decision_ref='decision:o304:go-integracion'),
			  (SELECT count(*) FROM
			     vec_contratacion_temporal.expediente_version_integral
			    WHERE expediente_ref=$1)`,
			solicitud.ExpedienteRef,
		).Scan(&reservas, &decisiones, &versiones)
		t.Fatalf(
			"respuesta perdida no quedó indeterminada: %v; reservas=%d decisiones=%d versiones=%d; artefactos=%d sellos=%d consultas=%d error_consulta=%v preparaciones=%d error_preparacion=%v politicas=%d autorizaciones=%d error_autorizacion=%v confirmaciones=%d error_confirmacion=%v",
			err, reservas, decisiones, versiones, artefactos.llamadas,
			sellador.llamadas, preparadorObservado.consultas,
			preparadorObservado.errUltimaConsulta,
			preparadorObservado.llamadas, preparadorObservado.err,
			politicas.llamadas, autorizador.llamadas,
			autorizador.errUltimo,
			transaccionObservada.llamadas,
			transaccionObservada.err,
		)
	}
	runtime.Close()
	runtimeReiniciado := abrirPoolIntegracionO304(t, ctx, dsnRuntime)
	defer runtimeReiniciado.Close()
	preparadorReiniciado, err := adaptadorpostgres.
		NuevoPreparadorOperacionAnalisisPostgreSQL(runtimeReiniciado)
	if err != nil {
		t.Fatal(err)
	}
	transaccionReiniciada, err := adaptadorpostgres.
		NuevaTransaccionOperacionesAnalisisPostgreSQL(runtimeReiniciado)
	if err != nil {
		t.Fatal(err)
	}
	servicioReiniciado, err := NuevoServicioOperacionAnalisis(
		contextos,
		artefactos,
		sellador,
		preparadorReiniciado,
		politicas,
		&generadorReferenciasDoble{
			correlacion: "correlacion_1234567890abcdef1234567890abcdef",
		},
		autorizador,
		&relojMutable{instante: ahora},
		transaccionReiniciada,
	)
	if err != nil {
		t.Fatal(err)
	}
	primero, err := servicioReiniciado.Registrar(ctx, solicitud)
	if err != nil {
		t.Fatalf("reconciliar tras respuesta perdida y reinicio: %v", err)
	}
	segundo, err := servicioReiniciado.Registrar(ctx, solicitud)
	if err != nil {
		t.Fatalf("replay Go → PostgreSQL: %v", err)
	}
	if primero != segundo || primero.VersionAnterior != 1 ||
		primero.VersionResultante != 2 {
		t.Fatalf("recibo o replay incoherente: %#v / %#v", primero, segundo)
	}
	var versiones, recibos int
	err = administrador.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM
		     vec_contratacion_temporal.expediente_version_integral
		    WHERE expediente_ref=$1),
		  (SELECT count(*) FROM
		     vec_contratacion_temporal.confirmacion_operacion_analisis c
		     JOIN vec_contratacion_temporal.reserva_operacion_analisis r
		       USING (ambito_raiz_hmac)
		    WHERE r.expediente_ref=$1)`,
		solicitud.ExpedienteRef,
	).Scan(&versiones, &recibos)
	if err != nil || versiones != 2 || recibos != 1 {
		t.Fatalf(
			"efecto durable inesperado: versiones=%d recibos=%d err=%v",
			versiones, recibos, err,
		)
	}
}

func leerEntradaIntegracionAnalisisO304(
	t *testing.T,
	ruta string,
) entradaIntegracionAnalisisO304 {
	t.Helper()
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var entrada entradaIntegracionAnalisisO304
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&entrada); err != nil {
		t.Fatalf("entrada SQL O3-04 inválida: %v", err)
	}
	return entrada
}

func reconstruirContextoIntegracionO304(
	t *testing.T,
	entrada entradaIntegracionAnalisisO304,
	ahora time.Time,
) (ports.ContextoAutorizacionAltaV3, dominiovec.ReferenciaEntradaCatalogo) {
	t.Helper()
	decodificar := func(valor string) []byte {
		contenido, err := base64.StdEncoding.DecodeString(valor)
		if err != nil || len(contenido) == 0 {
			t.Fatalf("base64 O3-04 inválido: %v", err)
		}
		return contenido
	}
	contextoCanonico := decodificar(entrada.ContextoB64)
	manifiesto := decodificar(entrada.ManifiestoB64)
	plantillaBytes := decodificar(entrada.DecisionPlantillaB64)
	motivoBytes := decodificar(entrada.MotivoB64)
	var plantilla decisionPlantillaIntegracionO304
	var motivo motivoIntegracionO304
	if json.Unmarshal(plantillaBytes, &plantilla) != nil ||
		json.Unmarshal(motivoBytes, &motivo) != nil {
		t.Fatal("plantilla o motivo O3-04 inválidos")
	}
	actor, err := dominiovec.RehidratarContextoActorVinculadoV2(
		contextoCanonico,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: plantilla.Vinculo.RegistroContextoRef,
		Contexto:            actor, RepresentacionCanonica: contextoCanonico,
		HuellaSHA256:                      plantilla.Vinculo.ContextoActorHuellaSHA256,
		ManifiestoProcedenciaCanonico:     manifiesto,
		ManifiestoProcedenciaHuellaSHA256: entrada.ManifiestoHuella,
		AutoridadEfectiva: dominiovec.AutoridadProcedenciaContextoActorV1(
			entrada.AutoridadEfectiva,
		),
		ResueltoEnAutoritativo: parsearInstanteIntegracionO304(
			t, entrada.ResueltoEn,
		),
	}
	if err := resultado.Validar(); err != nil {
		t.Fatal(err)
	}
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: actor.Instantanea.CuentaRef,
		Metodo:    actor.Principal.AuthMethod,
		Garantia:  actor.Principal.AuthAssurance,
	}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV2(
		context.Background(),
		revalidadorVinculoPrueba{resultado: plantilla.Vinculo.Autenticacion()},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: plantilla.Vinculo.AutenticacionRef,
			SesionRef:        plantilla.Vinculo.SesionRef,
		},
		resolutorResultadoVinculoPrueba{resultado: resultado},
		dominiovec.SolicitudContextoActor{
			Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef,
		},
		relojVinculoPrueba{instante: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoAutorizacionAltaV3{
		Vinculo: vinculo, Resultado: resultado,
	}, motivo.Referencia
}

func parsearInstanteIntegracionO304(t *testing.T, valor string) time.Time {
	t.Helper()
	instante, err := time.Parse("2006-01-02T15:04:05.000000Z", valor)
	if err != nil {
		t.Fatalf("instante O3-04 inválido: %v", err)
	}
	return instante
}

func abrirPoolIntegracionO304(
	t *testing.T,
	ctx context.Context,
	dsn string,
) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	return pool
}

func restaurarAsignacionIntegracionO304(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		WITH origen AS (
			SELECT *
			  FROM vec_autorizacion.asignacion_perfil
			 WHERE asignacion_ref='asignacion:registro_v3:v2'
			   AND EXISTS (
			     SELECT 1
			       FROM vec_autorizacion.asignacion_perfil_actual
			      WHERE perfil_activo_ref=
			            'prf_sintetico_cccccccccccccccccccccccc'
			        AND asignacion_ref='asignacion:registro_v3:v3'
			   )
		), preparada AS (
			SELECT origen.*,
			       jsonb_set(
			         jsonb_set(documento,'{version}','4'::jsonb),
			         '{vigente_hasta}',
			         to_jsonb(to_char(
			           (clock_timestamp()+interval '2 hours')
			             AT TIME ZONE 'UTC',
			           'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
			         ))
			       ) AS documento_nuevo
			  FROM origen
		)
		INSERT INTO vec_autorizacion.asignacion_perfil(
			asignacion_ref, asignacion_id, version, perfil_activo_ref,
			principal_id, version_rol_ref, huella_sha256,
			emitida_en, documento
		)
		SELECT 'asignacion:registro_v3:v4', asignacion_id, 4,
		       perfil_activo_ref, principal_id, version_rol_ref,
		       encode(sha256(convert_to(documento_nuevo::text,'UTF8')),'hex'),
		       (documento_nuevo->>'emitida_en')::timestamptz,
		       documento_nuevo
		  FROM preparada
		ON CONFLICT (asignacion_ref) DO NOTHING;
		INSERT INTO vec_autorizacion.asignacion_perfil_actual(
			perfil_activo_ref, asignacion_ref, actualizada_en,
			actualizada_por, acto_ref
		)
		SELECT
			'prf_sintetico_cccccccccccccccccccccccc',
			'asignacion:registro_v3:v4', clock_timestamp(),
			'autoridad-o304-go', 'acto:asignacion:o304:restaurada'
		 WHERE EXISTS (
		   SELECT 1
		     FROM vec_autorizacion.asignacion_perfil_actual
		    WHERE perfil_activo_ref=
		          'prf_sintetico_cccccccccccccccccccccccc'
		      AND asignacion_ref='asignacion:registro_v3:v3'
		 )
		ON CONFLICT (perfil_activo_ref) DO UPDATE SET
			asignacion_ref=EXCLUDED.asignacion_ref,
			actualizada_en=EXCLUDED.actualizada_en,
			actualizada_por=EXCLUDED.actualizada_por,
			acto_ref=EXCLUDED.acto_ref`)
	if err != nil && !strings.Contains(err.Error(), "closed") {
		t.Errorf("restaurar asignación O3-04: %v", err)
	}
}
