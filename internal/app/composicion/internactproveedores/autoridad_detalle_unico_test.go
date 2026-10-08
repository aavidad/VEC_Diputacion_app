package internactproveedores

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type relojDetalleUnicoPrueba struct{ ahora time.Time }

func (r relojDetalleUnicoPrueba) Ahora() time.Time { return r.ahora }

type fuenteDetalleUnicoPrueba struct {
	peticion inc.PeticionAutoridad
	contexto ct.ContextoAutorizacionAltaV3
	llamadas int
}

func (f *fuenteDetalleUnicoPrueba) ResolverPeticionYContexto(context.Context) (inc.PeticionAutoridad, ct.ContextoAutorizacionAltaV3, error) {
	f.llamadas++
	return f.peticion, f.contexto, nil
}

func contextoDetalleUnicoPrueba(t *testing.T, ahora time.Time) *fuenteDetalleUnicoPrueba {
	t.Helper()
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		ahora, "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "prf_0123456789abcdefghijkl",
		core.AuthMethodCertificate, core.AuthAssuranceSubstantial)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	p := inc.PeticionAutoridad{
		Autenticacion: core.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: datos.AutenticacionRef, SesionRef: datos.SesionRef},
		Contexto: core.SolicitudContextoActor{
			Cuenta: core.CuentaAutenticadaContextoActor{
				CuentaRef: datos.CuentaRef, Metodo: datos.MetodoObservado, Garantia: datos.GarantiaObservada},
			PerfilActivoRef: datos.PerfilActivoRef},
		PreparacionCT: ct.PreparacionSeguimientoConfirmacionIncorporacion{
			ActorRef: datos.PrincipalID, OrganizacionRef: "ref:" + strings.Repeat("a", 64),
			UnidadRef: "ref:" + strings.Repeat("b", 64), CorrelacionRef: "ref:" + strings.Repeat("c", 64)},
	}
	return &fuenteDetalleUnicoPrueba{peticion: p, contexto: ct.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}}
}

func TestAutoridadDetalleUsaUnaCapturaCoherente(t *testing.T) {
	ahora := time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)
	f := contextoDetalleUnicoPrueba(t, ahora)
	a := AutoridadDetalle{Reloj: relojDetalleUnicoPrueba{ahora}, fuentePrueba: f}
	c, err := a.ResolverContextoConsultaRRHH(context.Background())
	if err != nil || f.llamadas != 1 || c.AutenticacionRef() != f.peticion.Autenticacion.AutenticacionRef ||
		c.SesionRef() != f.peticion.Autenticacion.SesionRef || c.ActorRef() != f.peticion.PreparacionCT.ActorRef ||
		c.PerfilRef() != f.peticion.Contexto.PerfilActivoRef ||
		c.OrganizacionRef() != f.peticion.PreparacionCT.OrganizacionRef {
		t.Fatalf("detalle sin captura nominal unica: %v, llamadas=%d", err, f.llamadas)
	}
}

func TestAutoridadDetalleDeniegaMezclaDeCapturas(t *testing.T) {
	ahora := time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)
	for _, caso := range []struct {
		nombre  string
		cambiar func(*testing.T, *fuenteDetalleUnicoPrueba)
	}{
		{"autenticacion", func(_ *testing.T, f *fuenteDetalleUnicoPrueba) {
			f.peticion.Autenticacion.AutenticacionRef = "aut_" + strings.Repeat("b", 32)
		}},
		{"sesion", func(_ *testing.T, f *fuenteDetalleUnicoPrueba) {
			f.peticion.Autenticacion.SesionRef = "ses_" + strings.Repeat("b", 32)
		}},
		{"actor", func(_ *testing.T, f *fuenteDetalleUnicoPrueba) {
			f.peticion.PreparacionCT.ActorRef = "per_" + strings.Repeat("b", 32)
		}},
		{"cuenta", func(_ *testing.T, f *fuenteDetalleUnicoPrueba) {
			f.peticion.Contexto.Cuenta.CuentaRef = "cta_" + strings.Repeat("b", 32)
		}},
		{"perfil", func(_ *testing.T, f *fuenteDetalleUnicoPrueba) {
			f.peticion.Contexto.PerfilActivoRef = "prf_" + strings.Repeat("b", 32)
		}},
		{"persona_f1", func(t *testing.T, f *fuenteDetalleUnicoPrueba) {
			otro, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_"+strings.Repeat("b", 32),
				"prf_0123456789abcdefghijkl", core.AuthMethodCertificate, core.AuthAssuranceSubstantial)
			if err != nil {
				t.Fatal(err)
			}
			f.contexto.Resultado = otro
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			f := contextoDetalleUnicoPrueba(t, ahora)
			caso.cambiar(t, f)
			a := AutoridadDetalle{Reloj: relojDetalleUnicoPrueba{ahora}, fuentePrueba: f}
			if _, err := a.ResolverContextoConsultaRRHH(context.Background()); !errors.Is(err, ErrAutoridadCTNoDisponible) || f.llamadas != 1 {
				t.Fatalf("capturas mezcladas admitidas o consulta repetida: %v, llamadas=%d", err, f.llamadas)
			}
		})
	}
}
