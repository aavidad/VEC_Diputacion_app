package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Requieren exclusivamente un PostgreSQL 18 desechable con la estructura de
// la principal (por ejemplo, un clon restaurado en /dev/shm). Nunca apuntan a
// una base conservada.
func poolesPerfilesCentroPostgreSQLPrueba(t *testing.T) (context.Context, *pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	if os.Getenv("VEC_PERFILES_CENTRO_PG_DESECHABLE") != "1" {
		t.Skip("requiere PostgreSQL 18 desechable con la estructura de la principal")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancelar)
	abrir := func(variable string) *pgxpool.Pool {
		pool, err := pgxpool.New(ctx, os.Getenv(variable))
		if err != nil {
			t.Fatalf("pool %s no disponible", variable)
		}
		t.Cleanup(pool.Close)
		return pool
	}
	return ctx, abrir("VEC_PERFILES_CENTRO_PG_DSN_GOBIERNO"), abrir("VEC_PERFILES_CENTRO_PG_DSN_ADMIN")
}

func identidadCentroPostgreSQLPrueba(t *testing.T, pool *pgxpool.Pool) *identidadPeticionCentroDesarrollo {
	t.Helper()
	var aleatorio [8]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		t.Fatal(err)
	}
	id, _ := escenarioIdentidadCentroPrueba(t, "pg_"+hex.EncodeToString(aleatorio[:]))
	for _, s := range []*soporteAltaContratacionTemporalDesarrollo{id.soporte, id.cancelacion.soporte} {
		s.autoridadAsignaciones = &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: s}
	}
	return id
}

type historiaPerfilPrueba struct {
	versiones int
	actual    string
	acto      string
}

func historiaPerfilPostgreSQLPrueba(t *testing.T, ctx context.Context, admin *pgxpool.Pool, perfil string) historiaPerfilPrueba {
	t.Helper()
	var h historiaPerfilPrueba
	if err := admin.QueryRow(ctx, `SELECT count(*) FROM vec_autorizacion.asignacion_perfil WHERE perfil_activo_ref=$1`, perfil).Scan(&h.versiones); err != nil {
		t.Fatal(err)
	}
	_ = admin.QueryRow(ctx, `SELECT asignacion_ref, acto_ref FROM vec_autorizacion.asignacion_perfil_actual WHERE perfil_activo_ref=$1`, perfil).Scan(&h.actual, &h.acto)
	return h
}

// revocarPorOtroActoPrueba publica, como lo haría una revocación gobernada
// con su propio acto, la asignación vigente revocada.
func revocarPorOtroActoPrueba(t *testing.T, ctx context.Context, s *soporteAltaContratacionTemporalDesarrollo, pool *pgxpool.Pool, estrechar bool) {
	t.Helper()
	publicada, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, pool, s.instantanea.AsignacionPerfil.PerfilActivoRef)
	if err != nil || !encontrada {
		t.Fatalf("sin asignación que revocar: %v", err)
	}
	administrativa := (&autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: s}).autoridadComun()
	administrativa.exigirOrigenOperativo = false
	administrativa.actoAsignacion = "acto:seguridad:prueba:revocacion-centro"
	nueva := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(publicada.instantanea)
	if estrechar {
		nueva.AsignacionPerfil.Ambitos = append(nueva.AsignacionPerfil.Ambitos,
			vecdomain.AmbitoPerfil{Clave: "restriccion_ref", Valores: []string{"restriccion:prueba"}})
	} else {
		nueva.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
		nueva.AsignacionPerfil.RevocadaEn = nueva.AsignacionPerfil.EmitidaEn.Add(time.Second)
		nueva.AsignacionPerfil.RevocadaPor = "seguridad:prueba"
		nueva.AsignacionPerfil.RevocacionRef = "revocacion:prueba:centro"
	}
	nueva.AsignacionPerfil.Version = publicada.instantanea.AsignacionPerfil.Version + 1
	if err := administrativa.publicarInstantaneaDesdePreimagen(ctx, nueva, publicada.instantanea); err != nil {
		t.Fatalf("la revocación de prueba no se publicó: %v", err)
	}
}

func estrecharComoBinarioAnteriorPrueba(t *testing.T, ctx context.Context, s *soporteAltaContratacionTemporalDesarrollo, pool *pgxpool.Pool, expediente string) {
	t.Helper()
	comun := (&autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: s}).autoridadComun()
	objetivo := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(s.instantanea)
	objetivo.AsignacionPerfil.Ambitos = append(objetivo.AsignacionPerfil.Ambitos,
		vecdomain.AmbitoPerfil{Clave: "expediente_ref", Valores: []string{expediente}})
	preparada, err := comun.prepararInstantanea(ctx, objetivo, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := comun.publicarInstantanea(ctx, preparada); err != nil {
		t.Fatalf("el estrechamiento del binario anterior no se publicó: %v", err)
	}
}

func TestPerfilGeneralCentroArrancaYConsumeSinRepublicarPostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilesCentroPostgreSQLPrueba(t)
	id := identidadCentroPostgreSQLPrueba(t, gobierno)
	s := id.soporte
	perfil := s.instantanea.AsignacionPerfil.PerfilActivoRef
	if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, s); err != nil {
		t.Fatal(err)
	}
	asegurar := func(aprobacion string, esperado estadoPerfilCentroDesarrollo) {
		t.Helper()
		estado, err := asegurarPerfilCentroConsumibleDesarrollo(ctx, gobierno, s, aprobacion)
		if err != nil || estado != esperado {
			t.Fatalf("arranque: %q %v; se esperaba %q", estado, err, esperado)
		}
	}
	consumir := func() error {
		_, err := s.autoridadAsignaciones.(consumidorInstantaneaPublicadaDesarrollo).consumirInstantaneaPublicada(ctx, s.instantanea)
		return err
	}
	asegurar("", perfilCentroPublicadoInicial)
	inicial := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if inicial.versiones != 1 || inicial.acto != actoAsignacionCTDesarrollo {
		t.Fatalf("publicación inicial: %+v", inicial)
	}
	asegurar("", perfilCentroVigente)
	asegurar("aprobacion:prueba:centro:1", perfilCentroVigente)
	if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); h != inicial {
		t.Fatalf("el rearranque republicó: %+v", h)
	}

	// Treinta consumos simultáneos: ninguno escribe ni falla.
	var wg sync.WaitGroup
	fallos := make(chan error, 30)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fallos <- consumir()
		}()
	}
	wg.Wait()
	close(fallos)
	for err := range fallos {
		if err != nil {
			t.Fatalf("consumo concurrente: %v", err)
		}
	}
	if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); h != inicial {
		t.Fatalf("un consumo escribió: %+v", h)
	}

	// Lo que dejaba el binario anterior tras consultar una cancelación.
	estrecharComoBinarioAnteriorPrueba(t, ctx, s, gobierno, "expediente:centro:uno")
	estrechada := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if consumir() == nil {
		t.Fatal("se consumió una asignación estrechada a un expediente")
	}
	asegurar("", perfilCentroPendienteProvision)
	if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); h != estrechada {
		t.Fatal("sin aprobación el arranque escribió")
	}
	asegurar("aprobacion:prueba:centro:1", perfilCentroProvisionado)
	provisionada := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if provisionada.versiones != estrechada.versiones+1 || consumir() != nil {
		t.Fatalf("la provisión aprobada no dejó la asignación del centro: %+v", provisionada)
	}
	asegurar("aprobacion:prueba:centro:1", perfilCentroVigente)

	// Una restricción de otro acto y una revocación nunca se deshacen.
	for _, caso := range []struct {
		nombre    string
		estrechar bool
	}{{"restriccion", true}, {"revocacion", false}} {
		revocarPorOtroActoPrueba(t, ctx, s, gobierno, caso.estrechar)
		cerrada := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
		if consumir() == nil {
			t.Fatalf("%s: se consumió", caso.nombre)
		}
		asegurar("", perfilCentroPendienteProvision)
		asegurar("aprobacion:prueba:centro:1", perfilCentroPendienteProvision)
		if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); h != cerrada {
			t.Fatalf("%s: el arranque la reactivó: %+v", caso.nombre, h)
		}
	}
}

func TestPerfilCancelacionCentroNuncaReactivaLoRevocadoPostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilesCentroPostgreSQLPrueba(t)
	id := identidadCentroPostgreSQLPrueba(t, gobierno)
	s := id.cancelacion.soporte
	perfil := s.instantanea.AsignacionPerfil.PerfilActivoRef
	if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, id.soporte); err != nil {
		t.Fatal(err)
	}
	if err := publicarContextoCancelacionCentroDesarrollo(ctx, gobierno, s); err != nil {
		t.Fatal(err)
	}
	if err := asegurarPerfilCancelacionCentroDesarrollo(ctx, gobierno, s); err != nil {
		t.Fatal(err)
	}
	inicial := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	if inicial.versiones != 1 || inicial.acto != actoAsignacionCancelacionCentroDesarrollo {
		t.Fatalf("publicación inicial del perfil de cancelación: %+v", inicial)
	}
	if err := asegurarPerfilCancelacionCentroDesarrollo(ctx, gobierno, s); err != nil ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil) != inicial {
		t.Fatal("el rearranque republicó el perfil de cancelación")
	}
	comun := (&autoridadPostgreSQLContratacionTemporalDesarrollo{pool: gobierno, soporte: s}).autoridadComun()
	cancelar := func(expediente string) error {
		objetivo := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(s.instantanea)
		objetivo.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{
			{Clave: "centro_ref", Valores: []string{id.actor.CentroRef}},
			{Clave: "estado_previo", Valores: []string{"en_curso"}},
			{Clave: "expediente_ref", Valores: []string{expediente}},
			{Clave: "fase_previa", Valores: []string{"solicitud"}},
			{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
		}
		preparada, err := comun.prepararInstantanea(ctx, objetivo, false)
		if err != nil {
			return err
		}
		return comun.publicarInstantanea(ctx, preparada)
	}
	for _, expediente := range []string{"expediente:cancelar:uno", "expediente:cancelar:dos"} {
		if err := cancelar(expediente); err != nil {
			t.Fatalf("la cancelación no estrechó su perfil: %v", err)
		}
	}
	// Candidato preparado antes de una revocación concurrente.
	objetivo := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(s.instantanea)
	objetivo.AsignacionPerfil.Ambitos = append(objetivo.AsignacionPerfil.Ambitos,
		vecdomain.AmbitoPerfil{Clave: "expediente_ref", Valores: []string{"expediente:cancelar:tres"}})
	candidato, err := comun.prepararInstantanea(ctx, objetivo, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre    string
		estrechar bool
	}{{"restriccion", true}, {"revocacion", false}} {
		revocarPorOtroActoPrueba(t, ctx, s, gobierno, caso.estrechar)
		cerrada := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
		if cancelar("expediente:cancelar:cuatro") == nil || comun.publicarInstantanea(ctx, candidato) == nil {
			t.Fatalf("%s: una cancelación reactivó el perfil", caso.nombre)
		}
		if err := asegurarPerfilCancelacionCentroDesarrollo(ctx, gobierno, s); err != nil {
			t.Fatal(err)
		}
		if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil); h != cerrada {
			t.Fatalf("%s: la historia cambió tras el cierre: %+v", caso.nombre, h)
		}
	}
	// El perfil general de la misma persona no se ve afectado.
	if h := historiaPerfilPostgreSQLPrueba(t, ctx, admin, id.soporte.instantanea.AsignacionPerfil.PerfilActivoRef); h.versiones != 0 {
		t.Fatalf("la cancelación publicó en el perfil general: %+v", h)
	}
}

// El binario anterior publicaba al arrancar el rol del centro con otras
// concesiones cuando cambiaban los selectores. Ahora eso exige la aprobación.
func TestPerfilGeneralCentroCambioDeRolExigeAprobacionPostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilesCentroPostgreSQLPrueba(t)
	id := identidadCentroPostgreSQLPrueba(t, gobierno)
	s := id.soporte
	perfil := s.instantanea.AsignacionPerfil.PerfilActivoRef
	if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, s); err != nil {
		t.Fatal(err)
	}
	completa := s.instantanea
	reducida := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(completa)
	reducida.VersionRol.Concesiones = reducida.VersionRol.Concesiones[:1]
	s.instantanea = reducida
	if estado, err := asegurarPerfilCentroConsumibleDesarrollo(ctx, gobierno, s, ""); err != nil || estado != perfilCentroPublicadoInicial {
		t.Fatalf("inicial: %q %v", estado, err)
	}
	antes := historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil)
	s.instantanea = completa
	if estado, err := asegurarPerfilCentroConsumibleDesarrollo(ctx, gobierno, s, ""); err != nil || estado != perfilCentroPendienteProvision ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, perfil) != antes {
		t.Fatalf("un rol con concesiones nuevas se publicó sin aprobación: %q %v", estado, err)
	}
	if estado, err := asegurarPerfilCentroConsumibleDesarrollo(ctx, gobierno, s, "aprobacion:prueba:centro:2"); err != nil || estado != perfilCentroProvisionado {
		t.Fatalf("provisión aprobada del rol nuevo: %q %v", estado, err)
	}
	if _, err := s.autoridadAsignaciones.(consumidorInstantaneaPublicadaDesarrollo).consumirInstantaneaPublicada(ctx, s.instantanea); err != nil {
		t.Fatalf("el rol provisionado no se consume: %v", err)
	}
}
