package postgres

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/ports"
)

type filaVinculoCorporativoDoble struct {
	vigente bool
	err     error
}

func (f filaVinculoCorporativoDoble) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destinos[0].(*bool)) = f.vigente
	return nil
}

type consultorVinculoCorporativoDoble struct {
	fila       filaVinculoCorporativoDoble
	argumentos [][]any
}

func (c *consultorVinculoCorporativoDoble) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	c.argumentos = append(c.argumentos, append([]any(nil), args...))
	return c.fila
}

func solicitudVinculoCorporativoPrueba() ports.SolicitudRevalidacionVinculoCorporativoRRHHV1 {
	return ports.SolicitudRevalidacionVinculoCorporativoRRHHV1{
		CuentaRef: "cta_sintetica_corporativa_r_000000001", PerfilRef: "prf_sintetico_corporativo_r_000000001",
		PersonaRef: "per_sintetica_corporativa_r_000000001", VinculoContextoRef: "vca_sintetico_corporativo_r_000000001",
		VinculoContextoVersion: 1,
	}
}

func TestRevalidadorVinculoCorporativoTraduceRespuestaSinDatos(t *testing.T) {
	consultor := &consultorVinculoCorporativoDoble{fila: filaVinculoCorporativoDoble{vigente: true}}
	r := &RevalidadorVinculoCorporativoRRHHPostgreSQLV1{pool: consultor}
	s := solicitudVinculoCorporativoPrueba()
	if err := r.RevalidarVinculoCorporativoRRHHV1(context.Background(), s); err != nil {
		t.Fatalf("vigente: %v", err)
	}
	if want := []any{s.CuentaRef, s.PerfilRef, s.PersonaRef, s.VinculoContextoRef, "1"}; !reflect.DeepEqual(consultor.argumentos[0], want) {
		t.Fatalf("argumentos = %v", consultor.argumentos[0])
	}
	consultor.fila = filaVinculoCorporativoDoble{vigente: false}
	if err := r.RevalidarVinculoCorporativoRRHHV1(context.Background(), s); !errors.Is(err, ports.ErrVinculoCorporativoRRHHNoVigente) {
		t.Fatalf("no vigente: %v", err)
	}
	consultor.fila = filaVinculoCorporativoDoble{err: errors.New("detalle interno de PostgreSQL")}
	err := r.RevalidarVinculoCorporativoRRHHV1(context.Background(), s)
	if !errors.Is(err, ports.ErrVinculoCorporativoRRHHNoDisponible) || strings.Contains(err.Error(), "detalle interno") {
		t.Fatalf("indisponible: %v", err)
	}
}

func TestRevalidadorVinculoCorporativoRechazaEntradaYContextoSinConsultar(t *testing.T) {
	consultor := &consultorVinculoCorporativoDoble{fila: filaVinculoCorporativoDoble{vigente: true}}
	r := &RevalidadorVinculoCorporativoRRHHPostgreSQLV1{pool: consultor}
	for nombre, mutar := range map[string]func(*ports.SolicitudRevalidacionVinculoCorporativoRRHHV1){
		"cuenta": func(s *ports.SolicitudRevalidacionVinculoCorporativoRRHHV1) { s.CuentaRef = "cta_corta" },
		"perfil": func(s *ports.SolicitudRevalidacionVinculoCorporativoRRHHV1) { s.PerfilRef = s.PersonaRef },
		"persona": func(s *ports.SolicitudRevalidacionVinculoCorporativoRRHHV1) {
			s.PersonaRef = "per_con espacio_000000000000"
		},
		"contexto": func(s *ports.SolicitudRevalidacionVinculoCorporativoRRHHV1) { s.VinculoContextoRef = "" },
		"version":  func(s *ports.SolicitudRevalidacionVinculoCorporativoRRHHV1) { s.VinculoContextoVersion = 0 },
	} {
		s := solicitudVinculoCorporativoPrueba()
		mutar(&s)
		if err := r.RevalidarVinculoCorporativoRRHHV1(context.Background(), s); !errors.Is(err, ports.ErrSolicitudVinculoCorporativoRRHHInvalida) {
			t.Fatalf("%s: %v", nombre, err)
		}
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := r.RevalidarVinculoCorporativoRRHHV1(cancelado, solicitudVinculoCorporativoPrueba()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelado: %v", err)
	}
	var nulo *RevalidadorVinculoCorporativoRRHHPostgreSQLV1
	if err := nulo.RevalidarVinculoCorporativoRRHHV1(context.Background(), solicitudVinculoCorporativoPrueba()); !errors.Is(err, ports.ErrVinculoCorporativoRRHHNoDisponible) {
		t.Fatalf("nulo: %v", err)
	}
	if len(consultor.argumentos) != 0 {
		t.Fatalf("consultó con entrada inválida: %v", consultor.argumentos)
	}
	if _, err := NuevoRevalidadorVinculoCorporativoRRHHPostgreSQLV1(context.Background(), nil); !errors.Is(err, ports.ErrVinculoCorporativoRRHHNoDisponible) {
		t.Fatalf("sin pool: %v", err)
	}
}

// TestIntegracionPostgreSQLRevalidacionVinculoCorporativo se ejecuta desde
// deploy/postgresql/contexto_actor_v1/pruebas_sql/revalidacion_vinculo_corporativo_000008_pg18.sh
// con VEC_GO_INTEGRACION=1: la persona sintética R tiene su vínculo
// corporativo en versión 6 activa.
func TestIntegracionPostgreSQLRevalidacionVinculoCorporativo(t *testing.T) {
	dsn, admin := os.Getenv("VEC_CONTEXTO_ACTOR_CORPORATIVO_POSTGRES_DSN"), os.Getenv("VEC_CONTEXTO_ACTOR_CORPORATIVO_ADMIN_DSN")
	if dsn == "" || admin == "" {
		t.Skip("requiere el ensayo PostgreSQL 18 de ContextoActor 000008")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	administrador, err := pgxpool.New(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	defer administrador.Close()
	r, err := NuevoRevalidadorVinculoCorporativoRRHHPostgreSQLV1(ctx, pool)
	if err != nil {
		t.Fatalf("runtime con seis funciones rechazado: %v", err)
	}
	s := solicitudVinculoCorporativoPrueba()
	publicar := func(version int, estado string) {
		t.Helper()
		tx, err := administrador.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(context.Background())
		for _, sentencia := range []string{
			`SET LOCAL ROLE vec_contexto_actor_v1_propietario`,
			`INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones
SELECT vinculo_corporativo_ref,` + strconv.Itoa(version) + `,cuenta_ref,cuenta_version,persona_ref,persona_version,perfil_ref,
       perfil_version,vinculo_contexto_ref,vinculo_contexto_version,organizacion_ref,organizacion_version,
       organizacion_procedencia_ref,organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
       organizacion_procedencia_autoridad,superficie,uso,procedencia_ref,procedencia_version,
       procedencia_huella_sha256,procedencia_autoridad,'` + estado + `',vigente_desde,vigente_hasta
  FROM vec_contexto_actor_v1.vinculo_corporativo_versiones
 WHERE vinculo_corporativo_ref='vcr_sintetico_corporativo_r_000000001' ORDER BY version DESC LIMIT 1`,
			`UPDATE vec_contexto_actor_v1.vinculo_corporativo_actual SET version=` + strconv.Itoa(version) +
				` WHERE cuenta_ref='` + s.CuentaRef + `'`,
		} {
			if _, err := tx.Exec(ctx, sentencia); err != nil {
				t.Fatalf("publicar versión %d: %v", version, err)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("publicar versión %d: %v", version, err)
		}
	}
	if err := r.RevalidarVinculoCorporativoRRHHV1(ctx, s); err != nil {
		t.Fatalf("vínculo vigente: %v", err)
	}
	publicar(7, "revocado")
	if err := r.RevalidarVinculoCorporativoRRHHV1(ctx, s); !errors.Is(err, ports.ErrVinculoCorporativoRRHHNoVigente) {
		t.Fatalf("tras revocar, la siguiente consulta debía denegar: %v", err)
	}
	publicar(8, "activo")
	if err := r.RevalidarVinculoCorporativoRRHHV1(ctx, s); err != nil {
		t.Fatalf("restituido con versión nueva: %v", err)
	}
	otra := s
	otra.PersonaRef = "per_sintetica_corporativa_x_000000001"
	if err := r.RevalidarVinculoCorporativoRRHHV1(ctx, otra); !errors.Is(err, ports.ErrVinculoCorporativoRRHHNoVigente) {
		t.Fatalf("otra persona: %v", err)
	}
}
