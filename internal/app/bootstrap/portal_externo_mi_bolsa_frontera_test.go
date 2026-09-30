package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	mibolsa "vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type preparadorNoAutenticadoBolsaExterior struct{ llamadas int }

func (p *preparadorNoAutenticadoBolsaExterior) PrepararMiBolsa(*http.Request) (mibolsa.Orden, error) {
	p.llamadas++
	return mibolsa.Orden{}, bolsahttp.ErrAutenticacionAusente
}

func TestFronteraBolsaExteriorNoConfundeRutaNiAutoridad(t *testing.T) {
	p := &preparadorNoAutenticadoBolsaExterior{}
	a := &fronteraMiBolsaPortalExterno{preparador: p}
	if !errors.Is(a.AutorizarRutaExacta(context.Background(), bolsahttp.RutaMiBolsa), vechttp.ErrAutenticacionRutaExactaRequerida) {
		t.Fatal("aceptó contexto aportado sin frontera")
	}
	siguiente := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if !errors.Is(a.AutorizarRutaExacta(r.Context(), r.URL.Path), vechttp.ErrAutenticacionRutaExactaRequerida) {
			t.Fatal("autenticación fallida permitió continuar")
		}
		if _, err := a.PrepararMiBolsa(r); err == nil {
			t.Fatal("preparador reutilizó orden fallida")
		}
		otra := &fronteraMiBolsaPortalExterno{preparador: p}
		if !errors.Is(otra.AutorizarRutaExacta(r.Context(), r.URL.Path), vechttp.ErrAutenticacionRutaExactaRequerida) {
			t.Fatal("otra autoridad aceptó el contexto")
		}
		if !errors.Is(a.AutorizarRutaExacta(r.Context(), bolsahttp.RutaMiBolsaHistorial), vechttp.ErrAutenticacionRutaExactaRequerida) {
			t.Fatal("la orden pasó a otra ruta")
		}
	})
	a.proteger(siguiente).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, bolsahttp.RutaMiBolsa, nil))
	if p.llamadas != 1 {
		t.Fatal("la frontera preparó más de una vez la misma petición")
	}
}

func TestMiBolsaExteriorApagadaNoAbreInfraestructura(t *testing.T) {
	h, cerrar, err := nuevaMiBolsaPortalExterno(context.Background(), config.Config{}, nil, nil, nil, nil)
	if err != nil || h != nil || cerrar == nil {
		t.Fatalf("apagada: %v", err)
	}
	cerrar()
}

func TestFronteraBolsaExteriorAceptaVinculoVigenteConRelojCanonico(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		"per_candidato_frontera_1234567890123456", "prf_candidato_frontera_1234567890123456", core.AuthMethodCertificate, core.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	a := &fronteraMiBolsaPortalExterno{}
	ctx := context.WithValue(context.Background(), claveOrdenMiBolsaPortalExterno{}, contextoMiBolsaPortalExterno{
		autoridad: a, ruta: bolsahttp.RutaMiBolsa, orden: mibolsa.Orden{ResultadoContexto: resultado, Vinculo: vinculo}})
	if err := a.AutorizarRutaExacta(ctx, bolsahttp.RutaMiBolsa); err != nil {
		t.Fatal("denegó el vínculo vigente con el reloj del proceso")
	}
	if _, err := a.PrepararMiBolsa(httptest.NewRequest(http.MethodGet, bolsahttp.RutaMiBolsa, nil).WithContext(ctx)); err != nil {
		t.Fatal("perdió el par registrado autorizado")
	}
}
