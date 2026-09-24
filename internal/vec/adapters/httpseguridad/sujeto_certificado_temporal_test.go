package httpseguridad

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestCertificadoTemporalExigeMismaPersonaResueltaPorF1(t *testing.T) {
	ahora := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	configuracion := configuracionCertificadoDesarrollo()
	verificador := &verificadorFalso{}
	registro := nuevoRegistroMemoria()
	reloj := &relojFijo{ahora: ahora}
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceSubstantial), registro, reloj)
	canal := debeCanalTLS(t, servicio, configuracion)
	personaCertificado := "per_" + strings.Repeat("a", 22)
	personaCuentaAjena := "per_" + strings.Repeat("b", 22)
	asercion := asercionInternaValida(ahora, configuracion, canal)
	asercion.SujetoID = personaCertificado
	asercion.Cuenta.SujetoVinculadoID = personaCertificado
	asercion.ACRVerificado = ACRCertificadoPersonalDesarrolloProtegido
	asercion.Factores = asercion.Factores[1:]
	asercion.Factores[0].SujetoVinculadoID = personaCertificado
	verificador.fijarAsercion(asercion)
	identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("asercion-certificado"), canal))
	if err != nil {
		t.Fatalf("resolver certificado: %v", err)
	}
	capsula, err := servicio.ProyectarCapsulaIdentidadPeticion(context.Background(), identidad, canal)
	if err != nil {
		t.Fatalf("proyectar capsula: %v", err)
	}
	ctx, err := servicio.VincularCapsulaIdentidadPeticion(context.Background(), capsula, canal)
	if err != nil {
		t.Fatalf("vincular capsula: %v", err)
	}
	if err := servicio.ExigirSujetoPersonaCertificadoTemporal(ctx, personaCertificado); err != nil {
		t.Fatalf("F1 resolvio la misma persona: %v", err)
	}
	for _, persona := range []string{personaCuentaAjena, "per_corta", " per_" + strings.Repeat("a", 22), "per_" + strings.Repeat("a", 21) + "*"} {
		if err := servicio.ExigirSujetoPersonaCertificadoTemporal(ctx, persona); !errors.Is(err, ErrSesionNoValida) ||
			strings.Contains(err.Error(), personaCertificado) || strings.Contains(err.Error(), personaCuentaAjena) {
			t.Fatalf("persona F1 distinta o no canonica no debe pasar ni filtrarse: %v", err)
		}
	}
	if err := servicio.ExigirSujetoPersonaCertificadoTemporal(context.Background(), personaCertificado); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("sin capsula vinculada: %v", err)
	}
	if err := servicio.ExigirSujetoPersonaCertificadoTemporal(nil, personaCertificado); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("sin contexto: %v", err)
	}
	ctxCancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if err := servicio.ExigirSujetoPersonaCertificadoTemporal(ctxCancelado, personaCertificado); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("contexto cancelado: %v", err)
	}
	otroServicio := debeServicio(t, configuracion, &verificadorFalso{}, evaluadorValido(dominiovec.AuthAssuranceSubstantial), nuevoRegistroMemoria(), reloj)
	if err := otroServicio.ExigirSujetoPersonaCertificadoTemporal(ctx, personaCertificado); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("instancia ajena: %v", err)
	}
	registro.inactivar("cuenta-tecnica")
	if err := servicio.ExigirSujetoPersonaCertificadoTemporal(ctx, personaCertificado); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("sesion revocada no debe cotejar: %v", err)
	}
}

func TestCertificadoTemporalNoAbrePoliticaCorporativa(t *testing.T) {
	entorno := nuevoEntornoCapsulaIdentidad(t)
	ctx, err := entorno.servicio.VincularCapsulaIdentidadPeticion(context.Background(), entorno.capsula, entorno.canal)
	if err != nil {
		t.Fatalf("vincular capsula corporativa: %v", err)
	}
	if err := entorno.servicio.ExigirSujetoPersonaCertificadoTemporal(ctx, "per_"+strings.Repeat("a", 22)); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("politica corporativa no usa el cotejo temporal: %v", err)
	}
}
