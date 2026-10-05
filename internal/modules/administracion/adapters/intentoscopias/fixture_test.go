package intentoscopias

import (
	"context"
	"strings"
	"testing"
	"time"

	http "vec-diputacion-granada/internal/modules/administracion/adapters/httpcopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
	d "vec-diputacion-granada/internal/vec/domain"
	v "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

const correlacionPrueba = "correlacion_11111111111111111111111111111111"

type fuenteFunc func(context.Context, string) (Acreditacion, error)

func (f fuenteFunc) RecuperarIntento(ctx context.Context, ref string) (Acreditacion, error) {
	return f(ctx, ref)
}

type registradorFunc func(context.Context, v.OrdenIntentoAuditoria) (v.AcuseIntentoAuditoria, error)

func (r registradorFunc) AppendIntentoAuditoria(ctx context.Context, o v.OrdenIntentoAuditoria) (v.AcuseIntentoAuditoria, error) {
	return r(ctx, o)
}

type autenticacionPrueba struct{ a d.AutenticacionRevalidadaV1 }

func (a autenticacionPrueba) RevalidarAutenticacionActorV1(context.Context, d.SolicitudRevalidacionAutenticacionActorV1) (d.AutenticacionRevalidadaV1, error) {
	return a.a, nil
}

type contextoPrueba struct {
	r d.ResultadoContextoActorRegistradoV2
}

func (r contextoPrueba) ResolverContextoActorRegistradoV2(context.Context, d.SolicitudContextoActor) (d.ResultadoContextoActorRegistradoV2, error) {
	return r.r.Clonar()
}

type relojPrueba struct{ t time.Time }

func (r relojPrueba) Ahora() time.Time { return r.t }

func acreditacionPrueba(t *testing.T, persona, perfil string, admin bool) Acreditacion {
	t.Helper()
	// Test factories only. The privileged pair must cross the sealed V2 factory;
	// it is never fabricated by parsing a packet or changing an opaque binding.
	instante := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(instante, persona, perfil, d.AuthMethodCertificate, d.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	if admin {
		vd, err := vinculo.Datos()
		if err != nil {
			t.Fatal(err)
		}
		auth := vd.Autenticacion()
		auth.Superficie = d.SuperficieAutenticacionAdministracionPrivilegiadaV1
		auth.CuentaPrivilegiada = true
		auth.CuentaOrdinariaRef = "cta_1123456789abcdefghijkl"
		vinculo, resultado, err = d.CrearVinculoAutenticacionActorV2ConResultado(context.Background(), autenticacionPrueba{auth}, d.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auth.AutenticacionRef, SesionRef: auth.SesionRef}, contextoPrueba{resultado}, d.SolicitudContextoActor{Cuenta: d.CuentaAutenticadaContextoActor{CuentaRef: auth.CuentaRef, Metodo: auth.MetodoObservado, Garantia: auth.GarantiaObservada}, PerfilActivoRef: perfil}, relojPrueba{instante})
		if err != nil {
			t.Fatal(err)
		}
	}
	return Acreditacion{CorrelacionRef: correlacionPrueba, Resultado: resultado, Vinculo: vinculo}
}

func configuracionPrueba() Configuracion {
	c := Configuracion{Proceso: "vec-admin", Canal: "administracion_privilegiada", Plazo: time.Second, FronteraNominal: Operacion{Accion: "administracion.copias.frontera", FinalidadRef: "administracion_copias", RecursoFallback: "copias:frontera"}, Motivos: map[string]d.ReferenciaEntradaCatalogo{}, Operaciones: map[p.Operacion]Operacion{}}
	for _, code := range codigos {
		c.Motivos[code] = d.ReferenciaEntradaCatalogo{CatalogoID: "motivos_auditoria", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: code}
	}
	for _, action := range acciones {
		c.Operaciones[action] = Operacion{Accion: "administracion." + string(action), FinalidadRef: "administracion_copias", RecursoFallback: "copias:administracion"}
	}
	return c
}

func intentoPrueba() http.Denegacion {
	return http.Denegacion{Codigo: "acceso_denegado", Accion: string(p.Lanzar), RecursoRef: "copia:uno", ActorPersonaRef: "per_0123456789abcdefghijkl", PerfilActivoRef: "prf_0123456789abcdefghijkl", CorrelacionRef: correlacionPrueba}
}

func acusePrueba(t *testing.T, orden v.OrdenIntentoAuditoria) v.AcuseIntentoAuditoria {
	t.Helper()
	datos, err := orden.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return v.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_11111111111111111111111111111111", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: datos.Datos.CorrelacionRef, RegistradaEn: time.Now().UTC()}
}
