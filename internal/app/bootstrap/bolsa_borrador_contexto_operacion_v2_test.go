package bootstrap

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"time"
	core "vec-diputacion-granada/internal/vec/domain"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func fixtureContextoOperacionBolsa(t *testing.T, actor string) ports.ContextoAutorizacionAltaV3 {
	t.Helper()
	_, _, principal, ahora := fixtureSoporteSesionBorradorBolsa(t)
	principal.ID = "desarrollo:operacion-bolsa:" + actor
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(principal, ahora, discriminadorContextoSinteticoBorradorBolsaDesarrollo())
	if err != nil {
		t.Fatal(err)
	}
	return contexto
}

func registroOperacionBolsa(t *testing.T, contexto ports.ContextoAutorizacionAltaV3, operacion string) registroOperacionContextoBorradorBolsa {
	t.Helper()
	datos, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return registroOperacionContextoBorradorBolsa{operacion: operacion, registro: datos.RegistroContextoRef, cuenta: datos.CuentaRef, perfil: datos.PerfilActivoRef}
}

func TestOperacionContextoBolsaV2LegacyYActorNuevoCoexisten(t *testing.T) {
	a, b := fixtureContextoOperacionBolsa(t, "a"), fixtureContextoOperacionBolsa(t, "b")
	legacy := operacionContextoBorradorBolsaDesarrollo()
	original := registroOperacionBolsa(t, a, legacy)
	datos := []registroOperacionContextoBorradorBolsa{original}
	seleccion, err := elegirOperacionContextoBorradorBolsaDesarrollo(a, datos)
	if err != nil || seleccion != legacy {
		t.Fatal("el actor histórico perdió su operación")
	}
	nueva, err := elegirOperacionContextoBorradorBolsaDesarrollo(b, datos)
	if err != nil || nueva == legacy || nueva == "" {
		t.Fatal("el actor nuevo reutilizó la operación histórica")
	}
	scoped, _ := operacionContextoBorradorBolsaV2Desarrollo(b)
	if nueva != scoped {
		t.Fatal("la operación nueva no corresponde al contexto validado")
	}
	datos = append(datos, registroOperacionBolsa(t, b, nueva))
	for _, caso := range []struct {
		contexto ports.ContextoAutorizacionAltaV3
		esperada string
	}{{a, legacy}, {b, scoped}} {
		actual, err := elegirOperacionContextoBorradorBolsaDesarrollo(caso.contexto, datos)
		if err != nil || actual != caso.esperada {
			t.Fatal("la recuperación cambió de operación")
		}
	}
	if datos[0] != original {
		t.Fatal("la selección alteró la fila histórica")
	}
}

func TestOperacionContextoBolsaV2RechazaPreimagenesIncompatibles(t *testing.T) {
	a, b := fixtureContextoOperacionBolsa(t, "a"), fixtureContextoOperacionBolsa(t, "b")
	scoped, _ := operacionContextoBorradorBolsaV2Desarrollo(a)
	exacta := registroOperacionBolsa(t, a, scoped)
	legacy := registroOperacionBolsa(t, a, operacionContextoBorradorBolsaDesarrollo())
	otroRegistro := legacy
	otroRegistro.registro = referenciaAltaContratacionTemporalDesarrollo("rca_", "otro-registro")
	desconocida := exacta
	desconocida.operacion = referenciaAltaContratacionTemporalDesarrollo("oca_", "otra-operacion")
	colision := registroOperacionBolsa(t, b, scoped)
	registroAjeno := colision
	registroAjeno.registro = exacta.registro
	for nombre, filas := range map[string][]registroOperacionContextoBorradorBolsa{
		"legacy_mismo_actor_otro_registro":    {otroRegistro},
		"registro_bajo_operacion_desconocida": {desconocida},
		"scoped_ocupada_por_otro_actor":       {colision},
		"registro_propio_bajo_actor_ajeno":    {registroAjeno},
		"dos_operaciones_para_registro":       {legacy, exacta},
		"scoped_duplicada":                    {exacta, exacta},
	} {
		t.Run(nombre, func(t *testing.T) {
			actual, err := elegirOperacionContextoBorradorBolsaDesarrollo(a, filas)
			if actual != "" || !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
				t.Fatal("preimagen incompatible aceptada")
			}
		})
	}
}

func TestOperacionContextoBolsaV2ExigeVinculoYResultadoValidos(t *testing.T) {
	a, b := fixtureContextoOperacionBolsa(t, "a"), fixtureContextoOperacionBolsa(t, "b")
	cruzado := a
	cruzado.Resultado = b.Resultado
	for _, invalido := range []ports.ContextoAutorizacionAltaV3{{}, cruzado} {
		if actual, err := operacionContextoBorradorBolsaV2Desarrollo(invalido); actual != "" || err == nil {
			t.Fatal("contexto no acreditado aceptado")
		}
		if actual, err := elegirOperacionContextoBorradorBolsaDesarrollo(invalido, nil); actual != "" || err == nil {
			t.Fatal("selector aceptó contexto no acreditado")
		}
	}
	if actual, err := seleccionarOperacionContextoBorradorBolsaDesarrollo(context.Background(), nil, a); actual != "" || err == nil {
		t.Fatal("selector sin fuente aceptado")
	}
}

func TestOperacionContextoBolsaV2NoDependeDeHoraDeSesion(t *testing.T) {
	_, _, principal, ahora := fixtureSoporteSesionBorradorBolsa(t)
	a, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(principal, ahora, discriminadorContextoSinteticoBorradorBolsaDesarrollo())
	if err != nil {
		t.Fatal(err)
	}
	b, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(principal, ahora.Add(time.Hour), discriminadorContextoSinteticoBorradorBolsaDesarrollo())
	if err != nil {
		t.Fatal(err)
	}
	claveA, err := operacionContextoBorradorBolsaV2Desarrollo(a)
	if err != nil {
		t.Fatal(err)
	}
	claveB, err := operacionContextoBorradorBolsaV2Desarrollo(b)
	if err != nil || claveA != claveB {
		t.Fatal("la hora alteró la operación nominal")
	}
}

// La fixture es una base desechable: no instala migraciones ni usa el clon de
// recorrido. Prueba el publicador común y sus postimágenes con PostgreSQL real.
func TestOperacionContextoBolsaV2PG18CoexistenciaYPostimagen(t *testing.T) {
	dsn := os.Getenv("VEC_BBACK_OPERACION_TEST_DSN")
	if dsn == "" {
		t.Skip("PostgreSQL desechable no seleccionado")
	}
	configuration, err := pgxpool.ParseConfig(dsn)
	if err != nil || configuration.ConnConfig.Host != "127.0.0.1" || configuration.ConnConfig.Database != "vec_bback_operation_fixture" || os.Getenv("VEC_BBACK_OPERACION_TEST_SYSTEM_ID") == "" {
		t.Fatal("destino de fixture no acreditado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, configuration)
	if err != nil {
		t.Fatal("fixture PostgreSQL no disponible")
	}
	defer pool.Close()
	var version int
	if pool.QueryRow(ctx, `SELECT current_setting('server_version_num')::int`).Scan(&version) != nil || version/10000 != 18 {
		t.Fatal("la fixture exige PostgreSQL 18")
	}
	var cluster string
	if pool.QueryRow(ctx, `SELECT system_identifier::text FROM pg_catalog.pg_control_system()`).Scan(&cluster) != nil || cluster != os.Getenv("VEC_BBACK_OPERACION_TEST_SYSTEM_ID") {
		t.Fatal("otra instancia PostgreSQL")
	}
	if _, err = pool.Exec(ctx, fixtureSQLContextoBolsaV2); err != nil {
		t.Fatal("no se pudo preparar la fixture privada")
	}
	a, b := fixtureContextoOperacionBolsa(t, "a"), fixtureContextoOperacionBolsa(t, "b")
	legacy := operacionContextoBorradorBolsaDesarrollo()
	if err = publicarResultadoContextoPostgreSQLDesarrollo(ctx, pool, a.Resultado, legacy); err != nil {
		t.Fatal("no se pudo conservar el registro antiguo")
	}
	historical := huellaRegistroFixtureBolsa(t, ctx, pool, legacy)
	for _, actor := range []ports.ContextoAutorizacionAltaV3{a, b, a, b} {
		soporte := &soporteSesionBorradorBolsaDesarrollo{soporteCanal: &soporteAltaContratacionTemporalDesarrollo{contexto: actor}}
		if err = publicarContextoPostgreSQLBorradorBolsaDesarrollo(ctx, pool, soporte); err != nil {
			t.Fatal("publicación o recuperación compatible rechazada")
		}
	}
	var count int
	if pool.QueryRow(ctx, `SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto`).Scan(&count) != nil || count != 2 {
		t.Fatal("el replay duplicó un registro")
	}
	if huellaRegistroFixtureBolsa(t, ctx, pool, legacy) != historical {
		t.Fatal("la historia del actor antiguo cambió")
	}
	// Mismo actor y referencias, contenido distinto: la clave se conserva y el
	// publicador común rechaza la postimagen completa, sin otra operación.
	altered := contextoBolsaConVigenciaDistinta(t, b)
	scoped, _ := operacionContextoBorradorBolsaV2Desarrollo(b)
	alteredKey, _ := operacionContextoBorradorBolsaV2Desarrollo(altered)
	if scoped != alteredKey {
		t.Fatal("el contenido mutable alteró la operación")
	}
	before := huellaRegistroFixtureBolsa(t, ctx, pool, scoped)
	soporte := &soporteSesionBorradorBolsaDesarrollo{soporteCanal: &soporteAltaContratacionTemporalDesarrollo{contexto: altered}}
	if err = publicarContextoPostgreSQLBorradorBolsaDesarrollo(ctx, pool, soporte); !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
		t.Fatal("postimagen divergente aceptada")
	}
	if huellaRegistroFixtureBolsa(t, ctx, pool, scoped) != before {
		t.Fatal("el rechazo alteró el registro existente")
	}
	if pool.QueryRow(ctx, `SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto`).Scan(&count) != nil || count != 2 {
		t.Fatal("el rechazo añadió otro registro")
	}
}

func huellaRegistroFixtureBolsa(t *testing.T, ctx context.Context, pool *pgxpool.Pool, operacion string) string {
	t.Helper()
	var hash string
	if pool.QueryRow(ctx, `SELECT md5(row_to_json(r)::text) FROM vec_contexto_actor_v1.registros_contexto r WHERE operacion_ref=$1`, operacion).Scan(&hash) != nil {
		t.Fatal("registro de fixture ausente")
	}
	return hash
}

func contextoBolsaConVigenciaDistinta(t *testing.T, old ports.ContextoAutorizacionAltaV3) ports.ContextoAutorizacionAltaV3 {
	t.Helper()
	data, err := old.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	snapshot := old.Resultado.Contexto.Instantanea
	snapshot.VigenteHasta = snapshot.VigenteHasta.Add(time.Hour)
	actor, err := core.NuevoContextoActor(core.CuentaAutenticadaContextoActor{CuentaRef: data.CuentaRef, Metodo: data.MetodoObservado, Garantia: data.GarantiaObservada}, snapshot, old.Resultado.ResueltoEnAutoritativo)
	if err != nil {
		t.Fatal(err)
	}
	result := old.Resultado
	result.Contexto = actor
	result.RepresentacionCanonica, err = actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	result.HuellaSHA256, err = actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	auth := core.AutenticacionRevalidadaV1{AutenticacionRef: data.AutenticacionRef, AutenticacionHuellaSHA256: data.AutenticacionHuellaSHA256, AsercionRef: data.AsercionRef, SesionRef: data.SesionRef, ControlSesionRef: data.ControlSesionRef, ControlSesionRevision: data.ControlSesionRevision, ControlSesionHuellaSHA256: data.ControlSesionHuellaSHA256, CuentaRef: data.CuentaRef, CuentaOrdinariaRef: data.CuentaOrdinariaRef, CuentaPrivilegiada: data.CuentaPrivilegiada, Superficie: data.Superficie, MetodoObservado: data.MetodoObservado, GarantiaObservada: data.GarantiaObservada, PoliticaGarantiaRef: data.PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256: data.PoliticaGarantiaHuellaSHA256, AutenticacionVerificadaEn: data.AutenticacionVerificadaEn, SesionEmitidaEn: data.SesionEmitidaEn, SesionRevalidadaEn: data.SesionRevalidadaEn, SesionValidaHasta: data.SesionValidaHasta}
	link, result, err := core.CrearVinculoAutenticacionActorV2ConResultado(context.Background(), revalidadorAutenticacionAltaContratacionTemporalDesarrollo{valor: auth}, core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auth.AutenticacionRef, SesionRef: auth.SesionRef}, resolutorContextoAltaContratacionTemporalDesarrollo{valor: result}, core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: data.CuentaRef, Metodo: data.MetodoObservado, Garantia: data.GarantiaObservada}, PerfilActivoRef: data.PerfilActivoRef}, relojFijoAltaContratacionTemporalDesarrollo{ahora: time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoAutorizacionAltaV3{Vinculo: link, Resultado: result}
}

const fixtureSQLContextoBolsaV2 = `
CREATE ROLE vec_contexto_actor_v1_propietario NOLOGIN;
CREATE SCHEMA vec_contexto_actor_v1 AUTHORIZATION vec_contexto_actor_v1_propietario;
SET ROLE vec_contexto_actor_v1_propietario;
CREATE TABLE vec_contexto_actor_v1.procedencias(procedencia_ref text,procedencia_version bigint,procedencia_huella_sha256 text,procedencia_autoridad text,PRIMARY KEY(procedencia_ref,procedencia_version));
CREATE TABLE vec_contexto_actor_v1.proyeccion_cuenta_versiones(cuenta_ref text,version bigint,procedencia_ref text,procedencia_version bigint,procedencia_huella_sha256 text,procedencia_autoridad text,estado text,vigente_desde timestamptz,vigente_hasta timestamptz,PRIMARY KEY(cuenta_ref,version));
CREATE TABLE vec_contexto_actor_v1.proyeccion_cuenta_actual(cuenta_ref text PRIMARY KEY,version bigint);
CREATE TABLE vec_contexto_actor_v1.persona_versiones(persona_ref text,version bigint,procedencia_ref text,procedencia_version bigint,procedencia_huella_sha256 text,procedencia_autoridad text,estado text,vigente_desde timestamptz,vigente_hasta timestamptz,PRIMARY KEY(persona_ref,version));
CREATE TABLE vec_contexto_actor_v1.persona_actual(persona_ref text PRIMARY KEY,version bigint);
CREATE TABLE vec_contexto_actor_v1.perfil_versiones(perfil_ref text,version bigint,persona_ref text,procedencia_ref text,procedencia_version bigint,procedencia_huella_sha256 text,procedencia_autoridad text,estado text,vigente_desde timestamptz,vigente_hasta timestamptz,PRIMARY KEY(perfil_ref,version));
CREATE TABLE vec_contexto_actor_v1.perfil_actual(perfil_ref text PRIMARY KEY,version bigint);
CREATE TABLE vec_contexto_actor_v1.vinculo_contexto_versiones(vinculo_ref text,version bigint,cuenta_ref text,perfil_ref text,persona_ref text,procedencia_ref text,procedencia_version bigint,procedencia_huella_sha256 text,procedencia_autoridad text,estado text,vigente_desde timestamptz,vigente_hasta timestamptz,PRIMARY KEY(vinculo_ref,version));
CREATE TABLE vec_contexto_actor_v1.vinculo_contexto_actual(vinculo_ref text PRIMARY KEY,version bigint);
CREATE TABLE vec_contexto_actor_v1.registros_contexto(operacion_ref text PRIMARY KEY,registro_contexto_ref text UNIQUE,cuenta_ref text,perfil_ref text,metodo text,garantia text,solicitado_en timestamptz,resuelto_en timestamptz,representacion_canonica bytea,huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text);
RESET ROLE;
`
