package bootstrap

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Arranque doble de vec-server con VEC_SELECCION_SOLICITUDES_ENABLED contra
// PostgreSQL 18 desechable con el volcado sintético, los roles de Selección,
// AD3-89, AD3-90 y Selección 000001 (deploy/postgresql/seleccion/probar_pg18.sh
// con VEC_SELECCION_ARRANQUE_GO=1). Repite dos veces lo que publica el
// arranque: convocatorias (LOGIN de Bolsa), contexto, rol, asignación y motivo
// del perfil de RRHH de Selección, y el rol de la persona con las acciones
// propias de Selección y su motivo. El segundo arranque no puede fallar ni
// crear versiones; apagar el selector y volver a encenderlo reutiliza las
// versiones publicadas.
func TestArranqueDobleSeleccionPostgreSQL(t *testing.T) {
	if os.Getenv("VEC_SELECCION_ARRANQUE_PG") != "1" {
		t.Skip("requiere PostgreSQL 18 desechable con el volcado sintético y las migraciones de Selección")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancelar()
	abrir := func(variable string) *pgxpool.Pool {
		t.Helper()
		pool, err := pgxpool.New(ctx, os.Getenv(variable))
		if err != nil {
			t.Fatalf("pool %s no disponible", variable)
		}
		t.Cleanup(pool.Close)
		return pool
	}
	gobierno, bolsa, administracion := abrir("VEC_SELECCION_ARRANQUE_PG_DSN_GOBIERNO"), abrir("VEC_SELECCION_ARRANQUE_PG_DSN_BOLSA"), abrir("VEC_SELECCION_ARRANQUE_PG_DSN_ADMIN")
	cfg := config.Config{SeleccionConvocatoriasSourcePath: "../../../data/demo/reglas/seleccion_convocatorias.ejemplo.demo.json"}
	reloj := relojContratacionTemporalDesarrollo{}

	soporteCT, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	soporteRRHH, err := nuevoSoporteSeleccionRRHHDesarrollo(soporteCT, reloj.Ahora())
	if err != nil {
		t.Fatalf("soporte de RRHH de Selección: %v", err)
	}
	// Persona sintética con su propio perfil (en la principal es el de «Mi
	// bolsa», leído de la identidad del certificado).
	persona := dominiovec.Principal{ID: "certificado_persona_seleccion_prueba", AuthMethod: dominiovec.AuthMethodCertificate,
		AuthAssurance: dominiovec.AuthAssuranceHigh, Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("e", 64)}}
	contextoPersona, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(persona, reloj.Ahora(), discriminadorContextoSinteticoDesarrollo{
		perfil: "perfil-persona-seleccion-prueba", vinculo: "vinculo-persona-seleccion-prueba", procedencia: "procedencia",
		registro: "registro-persona-seleccion-prueba", autenticacion: "aut-persona-seleccion-prueba", asercion: "ase-persona-seleccion-prueba",
		sesion: "ses-persona-seleccion-prueba", controlSesion: "cse-persona-seleccion-prueba", politicaGarantia: "pga-persona-seleccion-prueba"})
	if err != nil {
		t.Fatal(err)
	}
	if err := publicarResultadoContextoPostgreSQLDesarrollo(ctx, gobierno, contextoPersona.Resultado,
		referenciaAltaContratacionTemporalDesarrollo("oca_", "seleccion-persona-prueba:registro-contexto:v1")); err != nil {
		t.Fatalf("contexto de la persona: %v", err)
	}
	datosPersona, err := contextoPersona.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	identidad := &identidadCandidatoBolsaDesarrollo{candidatoRef: referenciaAltaContratacionTemporalDesarrollo("can_", "persona-seleccion-prueba"),
		personaRef: datosPersona.PrincipalID, perfilRef: datosPersona.PerfilActivoRef}
	autoridadPersona := autoridadPostgreSQLDesarrollo{pool: gobierno, vinculo: contextoPersona.Vinculo,
		prefijoBloqueo: "vec:bolsa:mi-bolsa:autorizacion:", actoControlRol: "acto:bolsa:mi-bolsa:control-rol:v1",
		actoAsignacion: "acto:bolsa:mi-bolsa:asignacion:v1", actoSesion: "acto:bolsa:mi-bolsa:sesion:v1"}
	desde, _, _ := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	publicarPersona := func(etapa string, seleccion bool) string {
		t.Helper()
		semilla, err := nuevaInstantaneaMiBolsaDesarrollo(identidad, reloj.Ahora())
		if err == nil && seleccion {
			semilla, err = conConcesionesSeleccionPropiaDesarrollo(semilla)
		}
		if err != nil {
			t.Fatalf("%s: instantánea de la persona: %v", etapa, err)
		}
		preparada, err := autoridadPersona.prepararInstantanea(ctx, semilla, true)
		if err != nil || autoridadPersona.publicarInstantanea(ctx, preparada) != nil {
			t.Fatalf("%s: publicación del rol de la persona: %v", etapa, err)
		}
		if seleccion && publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, []dominiovec.ReferenciaEntradaCatalogo{motivoSeleccionPropiaDesarrollo()}, desde) != nil {
			t.Fatalf("%s: motivo de la persona", etapa)
		}
		return preparada.VersionRol.Referencia()
	}
	arrancar := func(etapa string, seleccion bool) (string, string) {
		t.Helper()
		if seleccion {
			if _, err := prepararSeleccionDesarrollo(ctx, cfg, bolsa); err != nil {
				t.Fatalf("%s: migraciones o convocatorias: %v", etapa, err)
			}
		}
		instantanea, err := publicarAutoridadSeleccionRRHHDesarrollo(ctx, gobierno, soporteRRHH, reloj)
		if err != nil {
			t.Fatalf("%s: autoridad de RRHH: %v", etapa, err)
		}
		return instantanea.VersionRol.Referencia(), publicarPersona(etapa, seleccion)
	}
	// contar devuelve convocatorias/roles/motivos y, aparte, las versiones de
	// asignación: cambiar de rol (apagar o encender el selector) añade una
	// versión de asignación a la historia, pero nunca roles ni convocatorias.
	contar := func() (string, int) {
		t.Helper()
		var convocatorias, roles, asignaciones, motivos int
		if err := administracion.QueryRow(ctx, `SELECT
		  (SELECT count(*) FROM vec_seleccion.convocatoria_publicada),
		  (SELECT count(*) FROM vec_autorizacion.version_rol WHERE rol_id IN ('tecnico_rrhh_seleccion_desarrollo','candidato_bolsa_consulta_propia_desarrollo')),
		  (SELECT count(*) FROM vec_autorizacion.asignacion_perfil WHERE perfil_activo_ref = ANY($1)),
		  (SELECT count(*) FROM vec_autorizacion.motivo_v2_entrada WHERE catalogo_id IN ('motivos_seleccion_solicitudes_propias','motivos_seleccion_consulta_rrhh'))`,
			[]string{soporteRRHH.perfilRef(), identidad.perfilRef}).Scan(&convocatorias, &roles, &asignaciones, &motivos); err != nil {
			t.Fatal(err)
		}
		return strings.Join([]string{strconv.Itoa(convocatorias), strconv.Itoa(roles), strconv.Itoa(motivos)}, "/"), asignaciones
	}

	rrhh1, persona1 := arrancar("primer arranque con Selección", true)
	tras1, asignaciones1 := contar()
	rrhh2, persona2 := arrancar("segundo arranque con Selección", true)
	if tras2, asignaciones2 := contar(); tras2 != tras1 || asignaciones2 != asignaciones1 || rrhh1 != rrhh2 || persona1 != persona2 {
		t.Fatalf("el segundo arranque cambió la base: %s → %s, asignaciones %d → %d, rol RRHH %s → %s, rol persona %s → %s",
			tras1, tras2, asignaciones1, asignaciones2, rrhh1, rrhh2, persona1, persona2)
	}
	_, personaSin := arrancar("arranque con el selector apagado", false)
	if personaSin == persona1 {
		t.Fatal("sin Selección el rol de la persona no puede llevar sus acciones")
	}
	tras3, _ := contar()
	_, personaOtraVez := arrancar("selector encendido otra vez", true)
	if tras4, _ := contar(); personaOtraVez != persona1 || tras4 != tras3 {
		t.Fatalf("volver a encender no reutilizó lo publicado: rol %s → %s, base %s → %s", persona1, personaOtraVez, tras3, tras4)
	}
	_, personaRepetida := arrancar("selector encendido otra vez, rearranque", true)
	if personaRepetida != persona1 {
		t.Fatal("el rearranque cambió el rol de la persona")
	}
}

// Con el selector encendido y sin las migraciones instaladas, el arranque se
// detiene nombrando la primera que falta (en el orden de instalación).
func TestArranqueSeleccionSinMigracionesSeDetienePostgreSQL(t *testing.T) {
	dsn := os.Getenv("VEC_SELECCION_SIN_MIGRACIONES_PG_DSN_BOLSA")
	if dsn == "" {
		t.Skip("requiere PostgreSQL 18 desechable sin las migraciones de Selección")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	cfg := config.Config{SeleccionConvocatoriasSourcePath: "../../../data/demo/reglas/seleccion_convocatorias.ejemplo.demo.json"}
	esperado := map[string]error{"ad3_89": ErrSeleccionFaltaAD389, "ad3_90": ErrSeleccionFaltaAD390, "sel_1": ErrSeleccionFaltaSeleccion1}[os.Getenv("VEC_SELECCION_SIN_MIGRACIONES_FALTA")]
	if esperado == nil {
		t.Fatal("VEC_SELECCION_SIN_MIGRACIONES_FALTA: ad3_89, ad3_90 o sel_1")
	}
	if _, err := prepararSeleccionDesarrollo(ctx, cfg, pool); !errors.Is(err, esperado) {
		t.Fatalf("se esperaba %v; se obtuvo %v", esperado, err)
	}
}
