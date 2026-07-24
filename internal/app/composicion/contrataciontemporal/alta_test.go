package contrataciontemporal

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	adaptadorhttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type dependenciasInertes struct{}

func (*dependenciasInertes) ResolverContextoCanalAlta(
	context.Context,
) (application.SolicitudRegistrarExpediente, error) {
	return application.SolicitudRegistrarExpediente{}, errors.New("inactivo")
}

func (*dependenciasInertes) ResolverContextoAutorizacionAltaV3(
	context.Context,
	ports.SolicitudResolverContextoAutorizacionAltaV3,
) (ports.ContextoAutorizacionAltaV3, error) {
	return ports.ContextoAutorizacionAltaV3{}, errors.New("inactivo")
}

func (*dependenciasInertes) ResolverFlujoAlta(
	context.Context,
	ports.SolicitudResolverFlujo,
) (ports.ConfiguracionAltaFlujo, error) {
	return ports.ConfiguracionAltaFlujo{}, errors.New("inactivo")
}

func (*dependenciasInertes) DerivarHuellaAlta(
	context.Context,
	ports.MaterialHuellaAlta,
) (ports.ColeccionSellosHMAC, error) {
	return ports.ColeccionSellosHMAC{}, errors.New("inactivo")
}

func (*dependenciasInertes) SellarAmbitoIdempotencia(
	context.Context,
	ports.SolicitudSellarAmbitoIdempotencia,
) (ports.ColeccionSellosHMAC, error) {
	return ports.ColeccionSellosHMAC{}, errors.New("inactivo")
}

func (*dependenciasInertes) ResolverMotivoAutorizacionAltaV3(
	context.Context,
	ports.SolicitudResolverMotivoAutorizacionAltaV3,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	return dominiovec.ReferenciaEntradaCatalogo{}, errors.New("inactivo")
}

func (*dependenciasInertes) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return "", errors.New("inactivo")
}

func (*dependenciasInertes) NuevaClaveMotivoAutorizacionV2(
	context.Context,
) (string, error) {
	return "", errors.New("inactivo")
}

func (*dependenciasInertes) GenerarReferenciasAlta(
	context.Context,
) (ports.ReferenciasAlta, error) {
	return ports.ReferenciasAlta{}, errors.New("inactivo")
}

func (*dependenciasInertes) NuevaReferenciaReservaAlta(
	context.Context,
) (string, error) {
	return "", errors.New("inactivo")
}

func (*dependenciasInertes) ExigirSolicitudLigadaV3(
	context.Context,
	dominiovec.SolicitudAutorizacionLigadaV3,
	dominiovec.ResultadoContextoActorRegistradoV2,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	error,
) {
	return dominiovec.DecisionAutorizacionLigadaV3{},
		puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
		errors.New("inactivo")
}

func (*dependenciasInertes) ObtenerMaterialConfirmacionAlta(
	context.Context,
	ports.OrdenConfirmarAlta,
) (ports.MaterialConfirmacionAlta, error) {
	return ports.MaterialConfirmacionAlta{}, errors.New("inactivo")
}

func (*dependenciasInertes) Ahora() time.Time {
	return time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
}

func poolInertePrueba(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(
		"postgres://vec@127.0.0.1:1/vec?sslmode=disable&connect_timeout=1",
	)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func dependenciasCompletasPrueba(t *testing.T) DependenciasAlta {
	t.Helper()
	doble := &dependenciasInertes{}
	return DependenciasAlta{
		AutoridadCanal: doble,
		Contextos:      doble,
		Flujos:         doble,
		Huellas:        doble,
		Ambitos:        doble,
		Motivos:        doble,
		Correlaciones:  doble,
		Referencias:    doble,
		Autorizador:    doble,
		Material:       doble,
		Reloj:          doble,
		PoolAltas:      poolInertePrueba(t),
	}
}

func TestNuevaAPIAltaEnumeraTodasLasDependenciasAusentes(t *testing.T) {
	api, err := NuevaAPIAlta(DependenciasAlta{})
	if api != nil || !errors.Is(err, ErrComposicionAltaNoDisponible) {
		t.Fatalf("composición vacía = (%T, %v)", api, err)
	}
	var faltantes *ErrorDependenciasAltaFaltantes
	if !errors.As(err, &faltantes) ||
		!reflect.DeepEqual(faltantes.Faltantes(), ordenDependenciasAlta[:]) {
		t.Fatalf("inventario incompleto: %#v", faltantes)
	}
	for _, dependencia := range ordenDependenciasAlta {
		if !faltantes.Falta(dependencia) {
			t.Errorf("no se declaró %s", dependencia)
		}
	}
	if err.Error() != ErrComposicionAltaNoDisponible.Error() {
		t.Fatalf("error no redactado: %q", err)
	}
}

func TestNuevaAPIAltaRechazaCadaDependenciaAusente(t *testing.T) {
	casos := []struct {
		nombre     string
		esperada   DependenciaAlta
		desactivar func(*DependenciasAlta)
	}{
		{"autoridad de canal", DependenciaAutoridadCanal, func(d *DependenciasAlta) { d.AutoridadCanal = nil }},
		{"contexto", DependenciaContextoActor, func(d *DependenciasAlta) { d.Contextos = nil }},
		{"flujo", DependenciaFlujo, func(d *DependenciasAlta) { d.Flujos = nil }},
		{"huella", DependenciaHuellaHMAC, func(d *DependenciasAlta) { d.Huellas = nil }},
		{"ámbito", DependenciaAmbitoHMAC, func(d *DependenciasAlta) { d.Ambitos = nil }},
		{"motivos", DependenciaMotivos, func(d *DependenciasAlta) { d.Motivos = nil }},
		{"correlaciones", DependenciaCorrelaciones, func(d *DependenciasAlta) { d.Correlaciones = nil }},
		{"referencias", DependenciaReferencias, func(d *DependenciasAlta) { d.Referencias = nil }},
		{"PDP", DependenciaPDPV3, func(d *DependenciasAlta) { d.Autorizador = nil }},
		{"material", DependenciaMaterialVEC, func(d *DependenciasAlta) { d.Material = nil }},
		{"reloj", DependenciaReloj, func(d *DependenciasAlta) { d.Reloj = nil }},
		{"PostgreSQL", DependenciaPostgreSQL, func(d *DependenciasAlta) { d.PoolAltas = nil }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			dependencias := dependenciasCompletasPrueba(t)
			caso.desactivar(&dependencias)
			api, err := NuevaAPIAlta(dependencias)
			var faltantes *ErrorDependenciasAltaFaltantes
			if api != nil || !errors.As(err, &faltantes) ||
				!reflect.DeepEqual(
					faltantes.Faltantes(),
					[]DependenciaAlta{caso.esperada},
				) {
				t.Fatalf("dependencia aceptada = (%T, %v)", api, err)
			}
		})
	}
}

func TestNuevaAPIAltaRechazaInterfazConPunteroNulo(t *testing.T) {
	dependencias := dependenciasCompletasPrueba(t)
	var autoridadNula *dependenciasInertes
	dependencias.AutoridadCanal = autoridadNula
	api, err := NuevaAPIAlta(dependencias)
	var faltantes *ErrorDependenciasAltaFaltantes
	if api != nil || !errors.As(err, &faltantes) ||
		!reflect.DeepEqual(
			faltantes.Faltantes(),
			[]DependenciaAlta{DependenciaAutoridadCanal},
		) {
		t.Fatalf("puntero nulo tipado aceptado = (%T, %v)", api, err)
	}
}

func TestNuevaAPIAltaRegistraSoloLaRutaExacta(t *testing.T) {
	api, err := NuevaAPIAlta(dependenciasCompletasPrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	comprobar := func(metodo, ruta string, cuerpo []byte, esperado int) {
		t.Helper()
		peticion := httptest.NewRequest(metodo, ruta, bytes.NewReader(cuerpo))
		peticion.Header.Set("Content-Type", "application/json")
		respuesta := httptest.NewRecorder()
		api.ServeHTTP(respuesta, peticion)
		if respuesta.Code != esperado {
			t.Fatalf("%s %s = %d; se esperaba %d",
				metodo, ruta, respuesta.Code, esperado)
		}
	}
	comprobar(
		http.MethodGet,
		adaptadorhttp.RutaAltaSolicitudes,
		nil,
		http.StatusMethodNotAllowed,
	)
	comprobar(
		http.MethodPost,
		adaptadorhttp.RutaAltaSolicitudes,
		[]byte(`{}`),
		http.StatusUnprocessableEntity,
	)
	comprobar(
		http.MethodPost,
		adaptadorhttp.RutaAltaSolicitudes+"/otra",
		[]byte(`{}`),
		http.StatusNotFound,
	)
}

var (
	_ adaptadorhttp.AutoridadContextoCanal          = (*dependenciasInertes)(nil)
	_ ports.ResolutorContextoAutorizacionAltaV3     = (*dependenciasInertes)(nil)
	_ ports.ResolutorFlujoAlta                      = (*dependenciasInertes)(nil)
	_ ports.DerivadorHuellaAlta                     = (*dependenciasInertes)(nil)
	_ ports.SelladorAmbitoIdempotencia              = (*dependenciasInertes)(nil)
	_ ports.ResolutorMotivoAutorizacionAltaV3       = (*dependenciasInertes)(nil)
	_ puertosvec.GeneradorReferenciasAutorizacionV2 = (*dependenciasInertes)(nil)
	_ ports.GeneradorReferenciasAlta                = (*dependenciasInertes)(nil)
	_ puertosvec.AutorizadorSolicitudLigadaV3       = (*dependenciasInertes)(nil)
	_ ports.ProveedorMaterialConfirmacionAlta       = (*dependenciasInertes)(nil)
	_ ports.Reloj                                   = (*dependenciasInertes)(nil)
)
