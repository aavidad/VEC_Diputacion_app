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

// El arranque consume la concesión central vigente sin ampliar por un selector
// nuevo. La prueba usa PostgreSQL 18 desechable: una provisión explícita de
// ensayo, con preimagen/CAS, acredita la transición v1→v2; después restringe
// y revoca la asignación para comprobar que ningún reinicio la revive. Esa
// provisión técnica no representa una aprobación funcional de RRHH.
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
	asignacionVigente := func() (string, string) {
		t.Helper()
		var asignacionRef, rolRef string
		if err := administracion.QueryRow(ctx, `SELECT asignacion.asignacion_ref, asignacion.version_rol_ref
		  FROM vec_autorizacion.asignacion_perfil_actual AS vigente
		  JOIN vec_autorizacion.asignacion_perfil AS asignacion
		    ON asignacion.perfil_activo_ref=vigente.perfil_activo_ref AND asignacion.asignacion_ref=vigente.asignacion_ref
		 WHERE vigente.perfil_activo_ref=$1`, v.PerfilActivoRef).Scan(&asignacionRef, &rolRef); err != nil {
			t.Fatal(err)
		}
		return asignacionRef, rolRef
	}
	semilla := func(concesiones []vecdomain.ConcesionRol) vecdomain.InstantaneaAutorizacion {
		t.Helper()
		instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, time.Now(), rolID,
			"Petición de centro de desarrollo", "peticion-centro-desarrollo-t3", concesiones,
			[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
		if err != nil {
			t.Fatal(err)
		}
		return instantanea
	}
	arrancar := func(nombre string, solicitada vecdomain.InstantaneaAutorizacion,
		esperada vecdomain.InstantaneaAutorizacion, versionesEsperadas []int, denegada bool) {
		t.Helper()
		soporte.mu.Lock()
		soporte.instantanea = solicitada
		soporte.mu.Unlock()
		antesRef, antesRol := "", ""
		if len(versiones()) > 0 {
			antesRef, antesRol = asignacionVigente()
		}
		err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, soporte)
		if denegada {
			if err == nil {
				t.Fatalf("%s: arranque aceptó autoridad restringida o revocada", nombre)
			}
		} else if err != nil {
			t.Fatalf("%s: arranque no consumió autoridad vigente: %v", nombre, err)
		}
		actualRef, actualRol := asignacionVigente()
		if actualRef != esperada.AsignacionPerfil.Referencia() || actualRol != esperada.VersionRol.Referencia() {
			t.Fatalf("%s: cambió la autoridad publicada: %s/%s", nombre, actualRef, actualRol)
		}
		if antesRef != "" && (antesRef != actualRef || antesRol != actualRol) {
			t.Fatalf("%s: el reinicio reescribió la asignación", nombre)
		}
		if !denegada {
			soporte.mu.Lock()
			consumida := soporte.instantanea
			soporte.mu.Unlock()
			if consumida.AsignacionPerfil.Referencia() != esperada.AsignacionPerfil.Referencia() ||
				consumida.VersionRol.Referencia() != esperada.VersionRol.Referencia() {
				t.Fatalf("%s: no conservó la instantánea publicada", nombre)
			}
		}
		if got := versiones(); !slices.Equal(got, versionesEsperadas) {
			t.Fatalf("%s: versiones del rol %v; se esperaban %v", nombre, got, versionesEsperadas)
		}
	}

	v1 := semilla(anteriores)
	arrancar("alta inicial ausente", v1, v1, []int{1}, false)
	soporte.mu.Lock()
	v1 = soporte.instantanea
	soporte.mu.Unlock()
	arrancar("reinicio con v1", semilla(anteriores), v1, []int{1}, false)
	arrancar("selectores nuevos sin provisión", semilla(nuevas), v1, []int{1}, false)

	// Acto gobernado de ensayo, separado del arranque: prepara desde la v1
	// vigente y confirma con esa preimagen exacta bajo CAS.
	autoridad := &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: gobierno, soporte: soporte}
	v2, err := autoridad.prepararInstantanea(ctx, semilla(nuevas), false)
	if err != nil || v2.VersionRol.Version != 2 || v2.AsignacionPerfil.Version != 2 {
		t.Fatalf("provisión explícita no preparó v2: %v", err)
	}
	if err := autoridad.autoridadComun().publicarInstantaneaDesdePreimagen(ctx, v2, v1); err != nil {
		t.Fatalf("provisión explícita v1→v2 rechazada: %v", err)
	}
	arrancar("reinicio con v2 gobernada", semilla(nuevas), v2, []int{1, 2}, false)
	arrancar("selector apagado no rebaja", semilla(anteriores), v2, []int{1, 2}, false)

	// La restricción y la revocación son actos explícitos sobre el clon. El
	// arranque debe cerrarse y dejar intacto el puntero central en ambos casos.
	restringida := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(v2)
	restringida.AsignacionPerfil.Ambitos[0].Valores = []string{"organizacion:restringida"}
	v3, err := autoridad.prepararInstantanea(ctx, restringida, false)
	if err != nil || v3.AsignacionPerfil.Version != 3 {
		t.Fatalf("restricción no preparada: %v", err)
	}
	if err := autoridad.autoridadComun().publicarInstantaneaDesdePreimagen(ctx, v3, v2); err != nil {
		t.Fatalf("restricción no publicada: %v", err)
	}
	arrancar("reinicio restringido", semilla(nuevas), v3, []int{1, 2}, true)

	revocada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(v3)
	revocada.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
	revocada.AsignacionPerfil.RevocadaPor = "autoridad:ensayo:revocacion"
	revocada.AsignacionPerfil.RevocadaEn = time.Now().UTC().Truncate(time.Microsecond)
	revocada.AsignacionPerfil.RevocacionRef = "revocacion:ensayo:t3"
	v4, err := autoridad.prepararInstantanea(ctx, revocada, false)
	if err != nil || v4.AsignacionPerfil.Version != 4 {
		t.Fatalf("revocación no preparada: %v", err)
	}
	if err := autoridad.autoridadComun().publicarInstantaneaDesdePreimagen(ctx, v4, v3); err != nil {
		t.Fatalf("revocación no publicada: %v", err)
	}
	arrancar("reinicio revocado", semilla(nuevas), v4, []int{1, 2}, true)
}
