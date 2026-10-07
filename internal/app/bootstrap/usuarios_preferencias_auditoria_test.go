package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	"vec-diputacion-granada/internal/shared/telemetria"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type entropiaDenegacionFallida struct{}

func (entropiaDenegacionFallida) Read([]byte) (int, error) { return 0, errors.New("sin entropía") }

func TestCorrelacionDenegacionFallaSinEntropia(t *testing.T) {
	if ref, err := correlacionDenegacionPreferenciasDesde(entropiaDenegacionFallida{}); err == nil || ref != "" {
		t.Fatal("se fabricó correlación repetible sin entropía")
	}
	ref, err := correlacionDenegacionPreferenciasDesde(bytes.NewReader(bytes.Repeat([]byte{0x5a}, 16)))
	if err != nil || ref != "corr_5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a" {
		t.Fatalf("correlación CSPRNG: %q %v", ref, err)
	}
}

type registradorDenegacionPreferenciasPrueba struct {
	ordenes []vecports.OrdenAuditoriaFronteraRutaExacta
	err     error
}

func (r *registradorDenegacionPreferenciasPrueba) RegistrarAuditoriaFronteraRutaExacta(_ context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta) error {
	r.ordenes = append(r.ordenes, orden)
	return r.err
}

type registradorFronteraDelegadoPrueba struct {
	ordenes []vecports.OrdenAuditoriaFronteraRutaExacta
}

func (r *registradorFronteraDelegadoPrueba) RegistrarAuditoriaFronteraRutaExacta(_ context.Context, o vecports.OrdenAuditoriaFronteraRutaExacta) error {
	r.ordenes = append(r.ordenes, o)
	return nil
}

func TestAuditoriaUsuariosWrapperTempranoUnaFilaSinCT(t *testing.T) {
	r := &registradorDenegacionPreferenciasPrueba{}
	a := &autoridadPreferenciasUsuariosDesarrollo{base: &autoridadRutasDietasDesarrollo{}, manejador: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), registrador: r, ruta: usuarioshttp.RutaMisPreferencias}
	pasos := 0
	h := a.proteger(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { pasos++ }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, usuarioshttp.RutaMisPreferencias, nil))
	if rec.Code != 401 || pasos != 0 || len(r.ordenes) != 1 || r.ordenes[0].ActorRef != "" || r.ordenes[0].Motivo != vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida || r.ordenes[0].Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias || r.ordenes[0].Ruta != usuarioshttp.RutaMisPreferencias {
		t.Fatalf("401 sin fila propia: %d %+v", rec.Code, r.ordenes)
	}
	r.err = errors.New("auditoria no disponible")
	var acceso bytes.Buffer
	h = telemetria.Middleware(telemetria.Opciones{Destino: &acceso, Servicio: "vec-server", Superficie: "interno", Entorno: "desarrollo"}, h)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, usuarioshttp.RutaMisPreferencias, nil))
	if rec.Code != 503 || pasos != 0 || len(r.ordenes) != 2 {
		t.Fatalf("caída registrador no cierra: %d %+v", rec.Code, r.ordenes)
	}
	var linea map[string]any
	if json.Unmarshal(acceso.Bytes(), &linea) != nil || linea["http.response.status_code"] != float64(503) {
		t.Fatalf("registro de acceso fallido: %s", acceso.String())
	}
	fases, ok := linea["vec.fases"].([]any)
	if !ok || len(fases) != 1 || fases[0].(map[string]any)["nombre"] != "auditoria_denegacion" || fases[0].(map[string]any)["errores"] != float64(1) || strings.Contains(acceso.String(), "auditoria no disponible") {
		t.Fatalf("fase de fallo o saneamiento: %s", acceso.String())
	}
}

func TestAuditoriaDispatcherUsuariosNoPasaPorCT(t *testing.T) {
	usuarios := &registradorDenegacionPreferenciasPrueba{}
	exterior := &registradorDenegacionPreferenciasPrueba{}
	ct := &registradorFronteraDelegadoPrueba{}
	m := registradorFronterasConUsuariosPreferencias{delegado: ct, interna: usuarios, externa: exterior}
	o := vecports.OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: "corr_0123456789abcdef0123456789abcdef", Motivo: vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado, Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias, Ruta: usuarioshttp.RutaMisPreferencias, ActorRef: "per_0123456789abcdefghijkl"}
	if err := m.RegistrarAuditoriaFronteraRutaExacta(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if len(usuarios.ordenes) != 1 || len(ct.ordenes) != 0 || usuarios.ordenes[0].ActorRef != o.ActorRef {
		t.Fatal("403 del dispatcher se atribuyó a CT o se duplicó")
	}
	o.Ruta = usuarioshttp.RutaMisPreferenciasAreaPersonal
	if err := m.RegistrarAuditoriaFronteraRutaExacta(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if len(exterior.ordenes) != 1 || len(usuarios.ordenes) != 1 || len(ct.ordenes) != 0 {
		t.Fatal("ruta exterior mezcló registradores")
	}
}

func TestAuditoriaAnonimaFijaRutaIntentada(t *testing.T) {
	for _, ruta := range []string{usuarioshttp.RutaMisPreferencias, usuarioshttp.RutaMisPreferenciasAreaPersonal} {
		r := &registradorDenegacionPreferenciasPrueba{}
		a := &autoridadPreferenciasUsuariosDesarrollo{base: &autoridadRutasDietasDesarrollo{}, manejador: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), registrador: r, ruta: ruta}
		rec := httptest.NewRecorder()
		a.proteger(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("anónimo llegó al despacho") })).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ruta, nil))
		if rec.Code != 401 || len(r.ordenes) != 1 || r.ordenes[0].Ruta != ruta || r.ordenes[0].ActorRef != "" {
			t.Fatalf("401 no fija ruta intentada: %s %d %+v", ruta, rec.Code, r.ordenes)
		}
	}
}
