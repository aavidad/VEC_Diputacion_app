package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// TestIntegracionPostgreSQLAlcanceProyeccionesContextoActorV2 se ejecuta desde
// deploy/postgresql/contexto_actor_v1/pruebas_sql/alcance_proyecciones_000007_pg18.sh,
// que deja la persona sintetica P con un empleado activo en Personal y la
// persona L sin proyeccion.
func TestIntegracionPostgreSQLAlcanceProyeccionesContextoActorV2(t *testing.T) {
	dsn := os.Getenv("VEC_CONTEXTO_ACTOR_ALCANCE_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requiere el ensayo PostgreSQL 18 de ContextoActor 000007")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	adaptador, err := NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pool)
	if err != nil {
		t.Fatalf("runtime con cinco funciones rechazado: %v", err)
	}
	generador := NuevoGeneradorOperacionContextoActorV2Criptografico()
	empleado, err := domain.NuevoAlcanceProyeccionesContextoActor(domain.ProyeccionContextoActorEmpleado)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := func(letra string, alcance domain.AlcanceProyeccionesContextoActor) ports.SolicitudResolucionRegistroContextoActorV2 {
		t.Helper()
		operacion, err := generador.NuevaReferenciaOperacionContextoActorV2(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return ports.SolicitudResolucionRegistroContextoActorV2{
			OperacionRef: operacion,
			Contexto: domain.SolicitudContextoActor{
				Cuenta: domain.CuentaAutenticadaContextoActor{
					CuentaRef: "cta_sintetica_alcance_" + letra + "_0000000000001",
					Metodo:    domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh,
				},
				PerfilActivoRef: "prf_sintetico_alcance_" + letra + "_0000000000001",
			},
			SolicitadoEn: time.Now().UTC().Truncate(time.Microsecond),
			Proyecciones: alcance,
		}
	}

	conEmpleado := solicitud("p", empleado)
	confirmacion, err := adaptador.ResolverYRegistrarContextoActorV2(ctx, conEmpleado)
	if err != nil || confirmacion.ValidarParaProductiva(conEmpleado) != nil ||
		!confirmacion.Contexto.AlcanceProyecciones().IncluyeEmpleado() {
		t.Fatalf("contexto con empleado de Personal rechazado: %v", err)
	}
	empleados, err := confirmacion.Contexto.Referencias(domain.TipoReferenciaContextoActorEmpleado)
	candidatos, errCandidatos := confirmacion.Contexto.Referencias(domain.TipoReferenciaContextoActorCandidato)
	if err != nil || errCandidatos != nil || len(empleados) != 1 || !strings.HasPrefix(empleados[0], "emp_") ||
		len(candidatos) != 1 {
		t.Fatal("el contexto no une candidato y empleado canonico de Personal")
	}
	repetida, err := adaptador.ResolverYRegistrarContextoActorV2(ctx, conEmpleado)
	if err != nil || repetida.RegistroContextoRef != confirmacion.RegistroContextoRef {
		t.Fatal("la operacion idempotente con alcance no recupero el recibo")
	}
	otroAlcance := conEmpleado
	otroAlcance.Proyecciones = domain.AlcanceProyeccionesContextoActor{}
	if _, err := adaptador.ResolverYRegistrarContextoActorV2(ctx, otroAlcance); !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) {
		t.Fatal("la misma operacion con otro alcance no colisiono")
	}

	vacio := solicitud("p", domain.AlcanceProyeccionesContextoActor{})
	heredado, err := adaptador.ResolverYRegistrarContextoActorV2(ctx, vacio)
	if err != nil || !heredado.Contexto.AlcanceProyecciones().Vacio() {
		t.Fatalf("alcance vacio rechazado: %v", err)
	}
	if sinEmpleado, _ := heredado.Contexto.Referencias(domain.TipoReferenciaContextoActorEmpleado); len(sinEmpleado) != 0 {
		t.Fatal("el alcance vacio incorporo el empleado de Personal")
	}

	if _, err := adaptador.ResolverYRegistrarContextoActorV2(ctx, solicitud("l", empleado)); !errors.Is(err, ports.ErrProyeccionEmpleadoContextoActorAusente) {
		t.Fatalf("ausencia pedida sin motivo: %v", err)
	}
}
