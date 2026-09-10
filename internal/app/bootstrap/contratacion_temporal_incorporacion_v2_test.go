package bootstrap

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestIncorporacionV2EnsamblajeContextoNominal(t *testing.T) {
	alta, a, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	v, _ := alta.soporte.contexto.Vinculo.Datos()
	refs := ReferenciasCTIncorporacionDesarrollo{PrincipalV3Ref: v.PrincipalID, PerfilV3Ref: v.PerfilActivoRef, OrganizacionRef: "ref:" + strings.Repeat("a", 64), UnidadRef: "ref:" + strings.Repeat("b", 64), ActorRef: "ref:" + strings.Repeat("c", 64)}
	f := &fuenteAutoridadIncorporacionV2Desarrollo{alta.soporte, a, alta.soporte.motivoDetalleRRHH, alta.soporte.motivoDetalleRRHH, refs}
	ctx := contextoRutaCoberturaDesarrolloPrueba(alta.soporte, principal, httpinterno.RutaIncorporacionEjercicioV2)
	c := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	c.certificadoVerificadoEn = alta.soporte.reloj.Ahora().Add(-time.Second)
	c.certificadoValidoHasta = alta.soporte.reloj.Ahora().Add(time.Minute)
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
	if _, err := f.PeticionVerificada(ctx); !errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) {
		t.Fatal("fuente aceptó contexto sin hijo sellado")
	}
	hijo, err := contextoDetalleIncorporacionV2Desarrollo(ctx, alta.soporte)
	if err != nil {
		t.Fatal(err)
	}
	h := hijo.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if h.ruta != httpinterno.RutaConsultaDetalleRRHH || h.consultaRRHH == nil || h.sello != c.sello || !reflect.DeepEqual(h.principal, c.principal) || h.certificadoValidoHasta != c.certificadoValidoHasta || h.certificadoVerificadoEn != c.certificadoVerificadoEn {
		t.Fatal("cápsula perdió identidad o amplió vigencia")
	}
	if original := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo); !reflect.DeepEqual(original, c) {
		t.Fatal("modificó petición original")
	}
	p, err := f.PeticionVerificada(hijo)
	if err != nil {
		t.Fatal(err)
	}
	if p.Autenticacion.AutenticacionRef != v.AutenticacionRef || p.Autenticacion.SesionRef != v.SesionRef || p.Contexto.PerfilActivoRef != v.PerfilActivoRef || p.PreparacionCT.ActorRef != refs.ActorRef || p.PreparacionCT.OrganizacionRef != refs.OrganizacionRef {
		t.Fatal("identidad no procede del contexto registrado")
	}
	if p.PreparacionCT.CorrelacionRef == "" || p.MotivoAlta != f.motivoAlta || p.MotivoLectura != f.motivoLectura {
		t.Fatal("motivos o correlación perdidos")
	}
	f.referencias.PerfilV3Ref = "perfil:ajeno"
	if _, err := f.PeticionVerificada(hijo); !errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) {
		t.Fatal("mapeo de otro perfil aceptado")
	}
	// Fuente de contexto doble del fixture RRHH: no acredita PG ni concede
	// alta Personal/lectura/CT. Esas autorizaciones siguen en AutoridadAplicacion.
}

func TestIncorporacionV2EnsamblajeBootstrapCerrado(t *testing.T) {
	alta, _, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	for _, ctx := range []context.Context{context.Background(), contextoRutaCoberturaDesarrolloPrueba(alta.soporte, principal, httpinterno.RutaConsultaDetalleRRHH)} {
		if c, err := contextoDetalleIncorporacionV2Desarrollo(ctx, alta.soporte); c != nil || !errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) {
			t.Fatal("ruta ajena admitida")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := contextoDetalleIncorporacionV2Desarrollo(ctx, alta.soporte); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelación perdida")
	}
	if s, err := nuevasDependenciasIncorporacionV2Desarrollo(ConfiguracionIncorporacionDesarrollo{}, alta, dependenciasConsultasRRHHDesarrollo{}, alta.soporte.reloj); s != nil || err == nil {
		t.Fatal("aceptó instalación sin identidad/consulta")
	}
	if s, _, err := NewHTTPServerDesarrolloWithConfig(config.Config{}, io.Discard, ConfiguracionIncorporacionDesarrollo{}, ConfiguracionIncorporacionDesarrollo{}); s != nil || err == nil {
		t.Fatal("aceptó dos configuraciones")
	}
	r := httptest.NewRequest(http.MethodGet, httpinterno.RutaIncorporacionEjercicioV2, nil)
	if !esRutaContratacionTemporalDesarrollo(r) {
		t.Fatal("ruta fuera del guardián mTLS")
	}
	llamadas := 0
	h := ligarContextoIncorporacionV2Desarrollo(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		llamadas++
		if r.Context().Value(claveIncorporacionV2Desarrollo{}) != nil {
			t.Fatal("selló petición no autenticada")
		}
	}), alta.soporte)
	h.ServeHTTP(httptest.NewRecorder(), r)
	if llamadas != 1 {
		t.Fatal("cambió el contrato de denegación del handler")
	}
}
