package adminperfiles

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type filaRuntimeContextoADMIN struct {
	login      string
	acreditada bool
	err        error
}

func vinculoContextoADMINPrueba(ahora time.Time) VinculoSesionADMIN {
	a, b := strings.Repeat("a", 22), strings.Repeat("b", 22)
	h := strings.Repeat("a", 64)
	return VinculoSesionADMIN{Referencia: "vis_" + strings.Repeat("a", 32), Version: 1,
		HuellaSHA256: h, AutenticacionRef: "aut_" + a, SesionRef: "ses_" + a,
		PersonaRef: "per_" + a, CuentaRef: "cta_" + a, CuentaOrdinariaRef: "cta_" + b,
		PerfilActivoRef: "prf_" + a, CertificadoSHA256: h, CASHA256: h,
		VinculoCertificadoRef: "vca_" + a, VinculoCertificadoVersion: 1,
		PoliticaRef: "pga_" + a, PoliticaSHA256: h, SeleccionRevision: 1,
		ControlSesionRef: "cse_" + a, ControlSesionRevision: 1, ControlSesionSHA256: h,
		VinculadaEn: ahora.Add(-time.Minute), VigenteHasta: ahora.Add(time.Minute),
		FuenteRef: "fuente:prueba", FuenteSHA256: h}
}

func TestContextoADMINNoRegistraSinVinculoDePeticionActual(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	s := ports.SolicitudResolucionRegistroContextoActorV2{
		OperacionRef: "oca_" + strings.Repeat("a", 32), SolicitadoEn: ahora,
		Contexto: domain.SolicitudContextoActor{
			Cuenta: domain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + strings.Repeat("a", 22),
				Metodo: domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh},
			PerfilActivoRef: "prf_" + strings.Repeat("a", 22)},
	}
	if err := s.Validar(); err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre string
		v      *VinculoSesionADMIN
	}{
		{nombre: "ausente"},
		{nombre: "otro_perfil", v: func() *VinculoSesionADMIN {
			v := vinculoContextoADMINPrueba(ahora)
			v.PerfilActivoRef = "prf_" + strings.Repeat("b", 22)
			return &v
		}()},
		{nombre: "caducado", v: func() *VinculoSesionADMIN { v := vinculoContextoADMINPrueba(ahora); v.VigenteHasta = ahora; return &v }()},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			ctx := context.Background()
			if caso.v != nil {
				var err error
				ctx, err = ContextoConVinculoSesionADMIN(ctx, *caso.v)
				if err != nil {
					t.Fatal(err)
				}
			}
			pool := &poolContextoFalso{}
			r := &resolutorContexto{base: &PostgreSQL{pool: pool, reloj: relojPrueba{ahora: ahora}}, proceso: "vec_admin", login: "login_contexto"}
			c, err := r.ResolverYRegistrarContextoActorV2(ctx, s)
			if !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) || c.OperacionRef != "" || len(pool.opciones) != 0 {
				t.Fatal("contexto ADMIN aceptó vínculo ausente, ajeno o vencido")
			}
		})
	}
}

func (f filaRuntimeContextoADMIN) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*string) = f.login
	*destinos[1].(*bool) = f.acreditada
	return nil
}

type poolRuntimeContextoADMIN struct {
	poolContextoFalso
	fila     filaRuntimeContextoADMIN
	consulta string
}

func (p *poolRuntimeContextoADMIN) QueryRow(_ context.Context, q string, _ ...any) pgx.Row {
	p.consulta = q
	return p.fila
}

func TestContextoADMINExigeGrupoPropioAlConstruir(t *testing.T) {
	const acreditar = `SELECT identidad_login,acreditada FROM vec_contexto_actor_v1.acreditar_runtime_contexto_admin_v1()`
	for _, caso := range []struct {
		fila   filaRuntimeContextoADMIN
		valida bool
	}{
		{filaRuntimeContextoADMIN{login: "", acreditada: true}, false},
		{filaRuntimeContextoADMIN{login: "login_admin", acreditada: false}, false},
		{filaRuntimeContextoADMIN{err: errors.New("42501")}, false},
		{filaRuntimeContextoADMIN{login: "login_admin", acreditada: true}, true},
	} {
		pool := &poolRuntimeContextoADMIN{fila: caso.fila}
		a, err := nuevoContextoRegistradoPostgreSQL(context.Background(), pool, relojPrueba{ahora: time.Now().UTC()},
			ConfiguracionContextoADMIN{Proceso: "vec_admin"})
		if (a != nil && err == nil) != caso.valida || pool.consulta != acreditar || len(pool.opciones) != 0 {
			t.Fatal("el contexto aceptó un LOGIN no acreditado o inició trabajo antes de construir")
		}
	}
	if a, err := nuevoContextoRegistradoPostgreSQL(context.Background(), &poolRuntimeContextoADMIN{},
		relojPrueba{ahora: time.Now().UTC()}, ConfiguracionContextoADMIN{}); a != nil || err == nil {
		t.Fatal("constructor aceptó proceso AD192 ausente")
	}
}
