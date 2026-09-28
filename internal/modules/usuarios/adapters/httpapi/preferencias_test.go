package httpapi

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/application"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ordenHTTPPrueba struct {
	orden ports.OrdenPreferencias
	err   error
}

func (o ordenHTTPPrueba) ResolverOrdenPreferencias(context.Context) (ports.OrdenPreferencias, error) {
	return o.orden, o.err
}

type proveedorHTTPPrueba struct{}

func (proveedorHTTPPrueba) ProveerMaterialPreferencias(_ context.Context, m ports.MaterialPreferencias) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	r, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion, m.PersonaRef, strings.Repeat("d", 64), "usuarios_preferencias", ahora, ahora.Add(3*time.Second))
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), r, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
}

type registroHTTPPrueba struct {
	recibo ports.ReciboPreferencias
	replay bool
	err    error
}

func (r *registroHTTPPrueba) CatalogoVigente(context.Context) (domain.CatalogoPreferencias, error) {
	return domain.CatalogoBasePreferencias(), nil
}
func (r *registroHTTPPrueba) ConsultarPropias(context.Context, ports.OrdenPreferencias, ports.MaterialPreferencias, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.EstadoPreferencias, bool, error) {
	return ports.EstadoPreferencias{}, false, r.err
}
func (r *registroHTTPPrueba) RecuperarOperacion(context.Context, ports.OrdenPreferencias, ports.MaterialPreferencias, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboPreferencias, bool, error) {
	return r.recibo, r.replay, r.err
}
func (r *registroHTTPPrueba) Guardar(context.Context, ports.OrdenPreferencias, ports.PeticionGuardarPreferencias, ports.MaterialPreferencias, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboPreferencias, error) {
	return r.recibo, r.err
}

func manejadorPrueba(t *testing.T) (*ManejadorPreferencias, *registroHTTPPrueba) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}
	snap := core.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 1, Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := core.NuevoContextoActor(cuenta, snap, ahora)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenPreferencias(actor, proveedorHTTPPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	c := domain.CatalogoBasePreferencias()
	r := &registroHTTPPrueba{recibo: ports.ReciboPreferencias{ReciboRef: "recibo:prueba", PersonaRef: actor.PersonaRef, Version: 1, CatalogoVersionRef: c.VersionRef, Valores: c.Predeterminados, FechaUTC: ahora}}
	s, err := application.NuevoServicioPreferencias(r, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NuevoManejadorPreferencias(s, ordenHTTPPrueba{orden: orden})
	if err != nil {
		t.Fatal(err)
	}
	return m, r
}

func hacerPeticion(t *testing.T, m http.Handler, metodo, ruta, cuerpo string) (int, map[string]json.RawMessage, http.Header) {
	t.Helper()
	r := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	if metodo == http.MethodPut {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	var sobre map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &sobre); err != nil {
		t.Fatalf("respuesta no JSON: %v", err)
	}
	return w.Code, sobre, w.Header()
}

func TestContratoHTTPPreferenciasGETPUTYReplay(t *testing.T) {
	m, r := manejadorPrueba(t)
	estado, sobre, cabeceras := hacerPeticion(t, m, http.MethodGet, RutaMisPreferencias, "")
	if estado != 200 || len(sobre) != 1 || len(sobre["data"]) == 0 || !strings.Contains(string(sobre["data"]), `"version":0`) || !strings.Contains(string(sobre["data"]), `"tamanos_texto":`) || !strings.Contains(cabeceras.Get("Cache-Control"), "no-store") {
		t.Fatalf("GET fuera de contrato: %d %s", estado, sobre)
	}
	c := domain.CatalogoBasePreferencias()
	b, _ := json.Marshal(ports.PeticionGuardarPreferencias{VersionEsperada: 0, CatalogoVersionRef: c.VersionRef, ClaveOperacion: "operacion-1234567890", Valores: c.Predeterminados})
	estado, sobre, _ = hacerPeticion(t, m, http.MethodPut, RutaMisPreferencias, string(b))
	if estado != 201 || len(sobre) != 1 || !strings.Contains(string(sobre["data"]), `"replay":false`) {
		t.Fatalf("PUT fuera de contrato: %d %s", estado, sobre)
	}
	r.replay = true
	estado, sobre, _ = hacerPeticion(t, m, http.MethodPut, RutaMisPreferencias, string(b))
	if estado != 200 || !strings.Contains(string(sobre["data"]), `"replay":true`) || !strings.Contains(string(sobre["data"]), `"recibo_ref":"recibo:prueba"`) {
		t.Fatalf("replay fuera de contrato: %d %s", estado, sobre)
	}
}

func TestErroresHTTPPreferenciasYJSONCerrado(t *testing.T) {
	m, r := manejadorPrueba(t)
	for _, caso := range []struct {
		err    error
		estado int
		codigo string
	}{
		{ports.ErrProhibido, 403, "prohibido"}, {ports.ErrConflicto, 409, "conflicto"}, {ports.ErrNoDisponible, 503, "no_disponible"},
	} {
		r.err = caso.err
		estado, sobre, _ := hacerPeticion(t, m, http.MethodGet, RutaMisPreferencias, "")
		if estado != caso.estado || !strings.Contains(string(sobre["error"]), `"codigo":"`+caso.codigo+`"`) || len(sobre) != 1 {
			t.Fatalf("error fuera de contrato: %d %s", estado, sobre)
		}
	}
	r.err = nil
	for _, cuerpo := range []string{
		`{"persona_ref":"per_ajena","version_esperada":0,"catalogo_version_ref":"usuarios-preferencias-v1","clave_operacion":"operacion-1234567890","valores":{}}`,
		`{"version_esperada":0,"version_esperada":1,"catalogo_version_ref":"usuarios-preferencias-v1","clave_operacion":"operacion-1234567890","valores":{}}`,
		`{"version_esperada":0,"catalogo_version_ref":"usuarios-preferencias-v1","clave_operacion":"operacion-1234567890","valores":{"idioma":"es"}}`,
	} {
		estado, sobre, _ := hacerPeticion(t, m, http.MethodPut, RutaMisPreferencias, cuerpo)
		if estado != 422 || !strings.Contains(string(sobre["error"]), `"codigo":"peticion_invalida"`) {
			t.Fatalf("JSON no cerrado: %d %s", estado, sobre)
		}
	}
	s, _ := application.NuevoServicioPreferencias(r, time.Now)
	anonimo, _ := NuevoManejadorPreferencias(s, ordenHTTPPrueba{err: ports.ErrNoAutenticado})
	estado, sobre, _ := hacerPeticion(t, anonimo, http.MethodGet, RutaMisPreferencias, "")
	if estado != 401 || !strings.Contains(string(sobre["error"]), `"codigo":"no_autenticado"`) {
		t.Fatalf("anónimo aceptado: %d %s", estado, sobre)
	}
}
