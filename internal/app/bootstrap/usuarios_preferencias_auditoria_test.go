package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

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
	a := &autoridadPreferenciasUsuariosDesarrollo{base: &autoridadRutasDietasDesarrollo{}, manejador: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), registrador: r}
	pasos := 0
	h := a.proteger(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { pasos++ }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, usuarioshttp.RutaMisPreferencias, nil))
	if rec.Code != 401 || pasos != 0 || len(r.ordenes) != 1 || r.ordenes[0].ActorRef != "" || r.ordenes[0].Motivo != vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida || r.ordenes[0].Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias || r.ordenes[0].Ruta != usuarioshttp.RutaMisPreferencias {
		t.Fatalf("401 sin fila propia: %d %+v", rec.Code, r.ordenes)
	}
	r.err = errors.New("auditoria no disponible")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, usuarioshttp.RutaMisPreferencias, nil))
	if rec.Code != 503 || pasos != 0 || len(r.ordenes) != 2 {
		t.Fatalf("caída registrador no cierra: %d %+v", rec.Code, r.ordenes)
	}
}

func TestAuditoriaDispatcherUsuariosNoPasaPorCT(t *testing.T) {
	usuarios := &registradorDenegacionPreferenciasPrueba{}
	ct := &registradorFronteraDelegadoPrueba{}
	m := registradorFronterasConUsuariosPreferencias{delegado: ct, usuarios: usuarios}
	o := vecports.OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: "corr_0123456789abcdef0123456789abcdef", Motivo: vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado, Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias, Ruta: usuarioshttp.RutaMisPreferencias, ActorRef: "per_0123456789abcdefghijkl"}
	if err := m.RegistrarAuditoriaFronteraRutaExacta(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if len(usuarios.ordenes) != 1 || len(ct.ordenes) != 0 || usuarios.ordenes[0].ActorRef != o.ActorRef {
		t.Fatal("403 del dispatcher se atribuyó a CT o se duplicó")
	}
}
