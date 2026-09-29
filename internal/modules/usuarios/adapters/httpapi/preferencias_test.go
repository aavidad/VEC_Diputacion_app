package httpapi

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

type auditorHTTPPrueba struct {
	estados []int
	err     error
}

func (a *auditorHTTPPrueba) AuditarDenegacionPreferencias(_ context.Context, estado int) error {
	a.estados = append(a.estados, estado)
	return a.err
}

func (o ordenHTTPPrueba) ResolverOrdenPreferencias(context.Context) (ports.OrdenPreferencias, error) {
	return o.orden, o.err
}

type proveedorHTTPPrueba struct{ emisiones int }

func (p *proveedorHTTPPrueba) ProveerMaterialPreferencias(_ context.Context, _ core.VinculoAutenticacionActorV2, m ports.MaterialPreferencias) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.emisiones++
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	audiencia, err := ports.AudienciaPreferencias(m.Accion, m.Superficie)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	r, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(fmt.Sprintf("dec_prueba_%d", p.emisiones), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion, m.PersonaRef, strings.Repeat("d", 64), audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{byte('x' + p.emisiones)}, 512), r, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
}

type registroHTTPPrueba struct {
	recibo ports.ReciboPreferencias
	replay bool
	err    error
}

func (r *registroHTTPPrueba) CatalogoVigente(context.Context, ports.OrdenPreferencias) (domain.CatalogoPreferencias, error) {
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

type revalidadorHTTPPrueba struct {
	a core.AutenticacionRevalidadaV1
}

func (r revalidadorHTTPPrueba) RevalidarAutenticacionActorV1(context.Context, core.SolicitudRevalidacionAutenticacionActorV1) (core.AutenticacionRevalidadaV1, error) {
	return r.a, nil
}

type resolutorHTTPPrueba struct {
	r core.ResultadoContextoActorRegistradoV2
}

func (r resolutorHTTPPrueba) ResolverContextoActorRegistradoV2(context.Context, core.SolicitudContextoActor) (core.ResultadoContextoActorRegistradoV2, error) {
	return r.r, nil
}

type relojHTTPPrueba struct{ ahora time.Time }

func (r relojHTTPPrueba) Ahora() time.Time { return r.ahora }

func identidadHTTPPrueba(t *testing.T, superficie core.SuperficieAutenticacionActorV1) (core.ContextoActor, core.VinculoAutenticacionActorV2) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	z := strings.Repeat("a", 24)
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}
	snap := core.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + strings.Repeat("r", 24), PersonaVersion: 1, PerfilActivoRef: "prf_" + strings.Repeat("p", 24), PerfilVersion: 1, Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := core.NuevoContextoActor(cuenta, snap, ahora)
	if err != nil {
		t.Fatal(err)
	}
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	huella, _ := actor.HuellaSHA256VinculadaV2()
	ac := core.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + z, ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	man := core.ManifiestoProcedenciaContextoActorV1{Esquema: core.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: core.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Persona: core.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac},
		Perfil: core.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Contexto: core.ProcedenciaVinculoContextoActorV1{VinculoRef: snap.VinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Vinculos: []core.ProcedenciaVinculoReferenciaContextoActorV1{}}
	bm, _ := man.RepresentacionCanonicaV1()
	hm, _ := core.HuellaSHA256ManifiestoProcedenciaContextoActorV1(bm)
	res := core.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_" + z, Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: bm, ManifiestoProcedenciaHuellaSHA256: hm, AutoridadEfectiva: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: ahora}
	if err = res.Validar(); err != nil {
		t.Fatal(err)
	}
	auth := core.AutenticacionRevalidadaV1{AutenticacionRef: "aut_" + z, AutenticacionHuellaSHA256: strings.Repeat("a", 64), AsercionRef: "ase_" + z, SesionRef: "ses_" + z, ControlSesionRef: "cse_" + z, ControlSesionRevision: 1, ControlSesionHuellaSHA256: strings.Repeat("b", 64), CuentaRef: cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: superficie, MetodoObservado: core.AuthMethodCertificate, GarantiaObservada: core.AuthAssuranceHigh, PoliticaGarantiaRef: "pga_" + z, PoliticaGarantiaHuellaSHA256: strings.Repeat("c", 64), AutenticacionVerificadaEn: ahora.Add(-time.Minute), SesionEmitidaEn: ahora.Add(-time.Minute), SesionRevalidadaEn: ahora.Add(-time.Second), SesionValidaHasta: ahora.Add(time.Minute)}
	if err = auth.Validar(); err != nil {
		t.Fatal(err)
	}
	v, err := core.CrearVinculoAutenticacionActorV2(context.Background(), revalidadorHTTPPrueba{auth}, core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auth.AutenticacionRef, SesionRef: auth.SesionRef}, resolutorHTTPPrueba{res}, core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef}, relojHTTPPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	return actor, v
}

func manejadorPrueba(t *testing.T) (*ManejadorPreferencias, *registroHTTPPrueba) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	actor, vinculo := identidadHTTPPrueba(t, core.SuperficieAutenticacionInternaCorporativaV1)
	orden, err := ports.NuevaOrdenPreferencias(actor, vinculo, core.SuperficieAutenticacionInternaCorporativaV1, &proveedorHTTPPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	c := domain.CatalogoBasePreferencias()
	r := &registroHTTPPrueba{recibo: ports.ReciboPreferencias{ReciboRef: "recibo:prueba", PersonaRef: actor.PersonaRef, Version: 1, CatalogoVersionRef: c.VersionRef, Valores: c.Predeterminados, FechaUTC: ahora}}
	s, err := application.NuevoServicioPreferencias(r, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NuevoManejadorPreferencias(s, ordenHTTPPrueba{orden: orden}, &auditorHTTPPrueba{})
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
	anonimo, _ := NuevoManejadorPreferencias(s, ordenHTTPPrueba{err: ports.ErrNoAutenticado}, &auditorHTTPPrueba{})
	estado, sobre, _ := hacerPeticion(t, anonimo, http.MethodGet, RutaMisPreferencias, "")
	if estado != 401 || !strings.Contains(string(sobre["error"]), `"codigo":"no_autenticado"`) {
		t.Fatalf("anónimo aceptado: %d %s", estado, sobre)
	}
}

func TestHandlerAuditaSoloDenegacionesDelCasoDeUso(t *testing.T) {
	m, r := manejadorPrueba(t)
	a := &auditorHTTPPrueba{}
	m.auditor = a
	r.err = ports.ErrProhibido
	estado, _, _ := hacerPeticion(t, m, http.MethodGet, RutaMisPreferencias, "")
	if estado != 403 || len(a.estados) != 1 || a.estados[0] != 403 {
		t.Fatal("403 no auditado una sola vez")
	}
	r.err = ports.ErrConflicto
	estado, _, _ = hacerPeticion(t, m, http.MethodGet, RutaMisPreferencias, "")
	if estado != 409 || len(a.estados) != 1 {
		t.Fatal("conflicto auditado como denegación")
	}
	r.err = ports.ErrProhibido
	a.err = ports.ErrNoDisponible
	estado, sobre, _ := hacerPeticion(t, m, http.MethodGet, RutaMisPreferencias, "")
	if estado != 503 || !strings.Contains(string(sobre["error"]), `"codigo":"no_disponible"`) || len(a.estados) != 2 {
		t.Fatal("caída de auditoría no cierra la respuesta")
	}
}

func TestRutasPreferenciasSeparadasPorSuperficie(t *testing.T) {
	interna, _ := manejadorPrueba(t)
	actor, vinculo := identidadHTTPPrueba(t, core.SuperficieAutenticacionExternaPersonalV1)
	orden, err := ports.NuevaOrdenPreferencias(actor, vinculo, core.SuperficieAutenticacionExternaPersonalV1, &proveedorHTTPPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	exterior, err := NuevoManejadorPreferenciasEnRuta(interna.servicio, ordenHTTPPrueba{orden: orden}, &auditorHTTPPrueba{}, RutaMisPreferenciasAreaPersonal)
	if err != nil {
		t.Fatal(err)
	}
	if estado, _, _ := hacerPeticion(t, exterior, http.MethodGet, RutaMisPreferenciasAreaPersonal, ""); estado != 200 {
		t.Fatalf("exterior: %d", estado)
	}
	if estado, _, _ := hacerPeticion(t, interna, http.MethodGet, RutaMisPreferenciasAreaPersonal, ""); estado != 422 {
		t.Fatalf("ruta exterior aceptada por handler interno: %d", estado)
	}
	if estado, _, _ := hacerPeticion(t, exterior, http.MethodGet, RutaMisPreferencias, ""); estado != 422 {
		t.Fatalf("ruta interna aceptada por handler exterior: %d", estado)
	}
}
