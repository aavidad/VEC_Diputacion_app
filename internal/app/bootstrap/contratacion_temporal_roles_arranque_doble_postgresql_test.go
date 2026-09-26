package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Reproduce el fallo del clon de ensayo del 26/09/2026 (T3): el binario
// anterior había publicado el rol del perfil del centro («rol:…:v1») con sus
// dos concesiones; el nuevo, con la incorporación acreditada y la cancelación
// encendidas, le añade las de la bandeja de incorporaciones y la cancelación.
// Una versión de rol publicada es inmutable y republicarla con otro contenido
// se rechazaba sin error SQL: el arranque se detenía con «PostgreSQL de
// contratacion temporal no disponible» y sin etapa. Arranque doble contra
// PostgreSQL 18 desechable con el volcado sintético y las migraciones de los
// huecos de RRHH (probar_huecos_rrhh_conjunto_pg18.sh), como el gobernador
// real: primero la publicación del binario anterior, después la del nuevo, su
// rearranque y la vuelta a los selectores apagados.
func TestArranqueDobleRolPeticionCentroPostgreSQL(t *testing.T) {
	if os.Getenv("VEC_ARRANQUE_T3_PG_DESECHABLE") != "1" {
		t.Skip("requiere PostgreSQL 18 desechable con el volcado sintético y las migraciones de los huecos de RRHH")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
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
	gobierno, administracion := abrir("VEC_ARRANQUE_T3_PG_DSN_GOBIERNO"), abrir("VEC_ARRANQUE_T3_PG_DSN_ADMIN")

	// Un rol propio de la prueba: el volcado puede traer ya los del centro.
	var sufijo [6]byte
	if _, err := rand.Read(sufijo[:]); err != nil {
		t.Fatal(err)
	}
	rolID := "solicitante_centro_t3_" + hex.EncodeToString(sufijo[:])
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	v, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, soporte); err != nil {
		t.Fatalf("contexto: %v", err)
	}

	// Concesiones del binario anterior (9361e961): consultar y presentar.
	anteriores := []vecdomain.ConcesionRol{}
	for _, accion := range []string{ports.AccionConsultarPeticionCentro, ports.AccionPresentarPeticionCentro} {
		anteriores = append(anteriores, vecdomain.ConcesionRol{Accion: accion, ModuloID: "contratacion_temporal",
			TipoRecurso: ports.TipoRecursoPeticionCentro, Finalidades: []string{finalidadPeticionCentro}, GarantiaMinima: vecdomain.AuthAssuranceHigh})
	}
	// Las del binario nuevo con la incorporación acreditada y la cancelación
	// por el centro encendidas, calculadas por la composición real.
	incorporacion := &incorporacionCentroDesarrollo{activo: true, roles: []string{"solicitante_centro", "ratificador_centro"}}
	cancelacion := &cancelacionCentroDesarrollo{piezas: &piezasCancelacionCTDesarrollo{}, roles: []string{"solicitante_centro", "ratificador_centro"}}
	nuevas := slices.Clone(anteriores)
	nuevas = append(nuevas, incorporacion.concesiones("solicitante_centro")...)
	nuevas = append(nuevas, cancelacion.concesiones("solicitante_centro")...)
	if len(nuevas) <= len(anteriores) {
		t.Fatal("el binario nuevo no añade concesiones al perfil del centro")
	}

	versiones := func() []int {
		t.Helper()
		filas, err := administracion.Query(ctx, `SELECT version FROM vec_autorizacion.version_rol WHERE rol_id=$1 ORDER BY version`, rolID)
		if err != nil {
			t.Fatal(err)
		}
		defer filas.Close()
		var resultado []int
		for filas.Next() {
			var version int
			if err := filas.Scan(&version); err != nil {
				t.Fatal(err)
			}
			resultado = append(resultado, version)
		}
		if filas.Err() != nil {
			t.Fatal(filas.Err())
		}
		return resultado
	}
	rolVigente := func() string {
		t.Helper()
		var ref string
		if err := administracion.QueryRow(ctx, `SELECT asignacion.version_rol_ref
		  FROM vec_autorizacion.asignacion_perfil_actual AS vigente
		  JOIN vec_autorizacion.asignacion_perfil AS asignacion
		    ON asignacion.perfil_activo_ref=vigente.perfil_activo_ref AND asignacion.asignacion_ref=vigente.asignacion_ref
		 WHERE vigente.perfil_activo_ref=$1`, v.PerfilActivoRef).Scan(&ref); err != nil {
			t.Fatal(err)
		}
		return ref
	}
	arrancar := func(nombre string, concesiones []vecdomain.ConcesionRol, versionEsperada int, versionesEsperadas []int) {
		t.Helper()
		instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, time.Now(), rolID,
			"Petición de centro de desarrollo", "peticion-centro-desarrollo-t3", concesiones,
			[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
		if err != nil {
			t.Fatal(err)
		}
		soporte.mu.Lock()
		soporte.instantanea = instantanea
		soporte.mu.Unlock()
		if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, soporte); err != nil {
			t.Fatalf("%s: el arranque no publica el rol del centro: %v", nombre, err)
		}
		soporte.mu.Lock()
		publicada := soporte.instantanea
		soporte.mu.Unlock()
		if publicada.VersionRol.Version != versionEsperada || rolVigente() != publicada.VersionRol.Referencia() {
			t.Fatalf("%s: rol v%d vigente %s; se esperaba v%d", nombre, publicada.VersionRol.Version, rolVigente(), versionEsperada)
		}
		if got := versiones(); !slices.Equal(got, versionesEsperadas) {
			t.Fatalf("%s: versiones del rol %v; se esperaban %v", nombre, got, versionesEsperadas)
		}
	}

	arrancar("binario anterior", anteriores, 1, []int{1})
	arrancar("binario anterior, rearranque", anteriores, 1, []int{1})
	arrancar("binario nuevo con los selectores encendidos", nuevas, 2, []int{1, 2})
	arrancar("binario nuevo, rearranque", nuevas, 2, []int{1, 2})
	arrancar("binario nuevo con los selectores apagados", anteriores, 1, []int{1, 2})
	arrancar("binario nuevo, selectores encendidos otra vez", nuevas, 2, []int{1, 2})
}
