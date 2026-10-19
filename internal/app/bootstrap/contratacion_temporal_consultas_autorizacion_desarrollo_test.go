package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestJustificanteRRHHNoLlamaResolvedorDemoYConservaCT(t *testing.T) {
	ahora := time.Now().UTC()
	hoja := &x509.Certificate{Raw: []byte("certificado-justificante-prueba"),
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour)}
	suma := sha256.Sum256(hoja.Raw)
	huella := hex.EncodeToString(suma[:])
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{
		fronteraJustificanteBolsaRRHHDesarrollo([]string{"prf_cuadro_prueba"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	base := &autoridadConsultasContratacionTemporalDesarrollo{
		sello:      &selloConsultasContratacionTemporalDesarrollo{},
		resolvedor: &resolvedorIdentidadDesarrollo{},
		cuentasInternasJustificanteRRHH: map[string]cuentaNominalJustificanteRRHH{
			huella: {certificadoSHA256: huella, sujeto: "per_" + strings.Repeat("a", 32), cuentaRef: "cta_" + strings.Repeat("a", 32)},
		},
		fronterasSeguridadComun: fronteras,
	}
	llamadasDemo := 0
	legacy := core.Principal{ID: "per_" + strings.Repeat("b", 32),
		Roles:      []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: core.AuthMethodCertificate, AuthAssurance: core.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": huella}}
	m := &revalidadorConsultasContratacionTemporalDesarrollo{
		autoridad: base, fronteras: fronteras,
		resolverDemo: func(context.Context, *http.Request) (core.Principal, error) { llamadasDemo++; return legacy, nil },
		siguiente: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, ok := r.Context().Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
			if !ok || c.sello != base.sello {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if r.URL.Path == bolsahttp.RutaJustificanteRenunciaRRHH &&
				(c.principal.ID != base.cuentasInternasJustificanteRRHH[huella].sujeto || len(c.principal.Roles) != 0) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	}
	peticion := func(metodo, ruta string) *http.Request {
		r := httptest.NewRequest(metodo, "https://localhost"+ruta, nil)
		r.RemoteAddr = "127.0.0.1:41100"
		r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
			PeerCertificates: []*x509.Certificate{hoja}, VerifiedChains: [][]*x509.Certificate{{hoja, {}}}}
		return r
	}
	get := func(r *http.Request) int { w := httptest.NewRecorder(); m.ServeHTTP(w, r); return w.Code }
	if estado := get(peticion(http.MethodGet, bolsahttp.RutaJustificanteRenunciaRRHH)); estado != http.StatusOK || llamadasDemo != 0 {
		t.Fatalf("GET nominal estado=%d llamadas demo=%d", estado, llamadasDemo)
	}
	base.cuentasInternasJustificanteRRHH = nil
	if estado := get(peticion(http.MethodGet, bolsahttp.RutaJustificanteRenunciaRRHH)); estado != http.StatusForbidden || llamadasDemo != 0 {
		t.Fatalf("certificado sin inventario estado=%d llamadas demo=%d", estado, llamadasDemo)
	}
	base.cuentasInternasJustificanteRRHH = map[string]cuentaNominalJustificanteRRHH{
		huella: {certificadoSHA256: huella, sujeto: "per_" + strings.Repeat("a", 32), cuentaRef: "cta_" + strings.Repeat("a", 32)},
	}
	base.resolvedor.candidatoBolsa = &identidadCandidatoBolsaDesarrollo{
		identidad: identidadCertificadoDesarrollo{huella: suma,
			principal: core.Principal{Attributes: map[string]string{"certificate_sha256": huella}}}}
	if estado := get(peticion(http.MethodGet, bolsahttp.RutaJustificanteRenunciaRRHH)); estado != http.StatusForbidden || llamadasDemo != 0 {
		t.Fatalf("aspirante estado=%d llamadas demo=%d", estado, llamadasDemo)
	}
	if estado := get(peticion(http.MethodPost, cthttp.RutaAltaSolicitudes)); estado != http.StatusOK || llamadasDemo != 1 {
		t.Fatalf("ruta CT estado=%d llamadas demo=%d", estado, llamadasDemo)
	}
}

// La hoja del segundo técnico solo se sella para el POST de no incorporación;
// las lecturas de RRHH, también las comunes de Bolsa y reglas, y el cese
// quedan sin capacidad sellada aunque su rol sea tecnico_rrhh.
func TestSegundoTecnicoRRHHSoloSellaNoIncorporacion(t *testing.T) {
	ahora := time.Now().UTC()
	hoja := &x509.Certificate{Raw: []byte("certificado-segundo-tecnico-prueba"),
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour)}
	suma := sha256.Sum256(hoja.Raw)
	huella := hex.EncodeToString(suma[:])
	segundo := core.Principal{ID: "per_" + strings.Repeat("c", 32),
		Roles:      []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: core.AuthMethodCertificate, AuthAssurance: core.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": huella}}
	resolvedor := &resolvedorIdentidadDesarrollo{segundoTecnicoRRHH: &identidadCertificadoDesarrollo{huella: suma, principal: segundo}}
	base := &autoridadConsultasContratacionTemporalDesarrollo{sello: &selloConsultasContratacionTemporalDesarrollo{}, resolvedor: resolvedor}
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo(nil)
	if err != nil {
		t.Fatal(err)
	}
	actual := segundo
	m := &revalidadorConsultasContratacionTemporalDesarrollo{
		autoridad: base, fronteras: fronteras,
		resolverDemo: func(context.Context, *http.Request) (core.Principal, error) { return actual, nil },
		siguiente: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, ok := r.Context().Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
			if !ok || c.sello != base.sello {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	}
	estado := func(metodo, ruta string) int {
		r := httptest.NewRequest(metodo, "https://localhost"+ruta, nil)
		r.RemoteAddr = "127.0.0.1:41100"
		r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
			PeerCertificates: []*x509.Certificate{hoja}, VerifiedChains: [][]*x509.Certificate{{hoja, {}}}}
		w := httptest.NewRecorder()
		m.ServeHTTP(w, r)
		return w.Code
	}
	for _, caso := range []struct{ metodo, ruta string }{
		{http.MethodGet, cthttp.RutaEstadisticasRRHH},
		{http.MethodGet, rutaBolsasRRHHDesarrollo},
		{http.MethodGet, rutaEstadisticasBolsaRRHHDesarrollo},
		{http.MethodGet, rutaAvisosBolsaRRHHDesarrollo},
		{http.MethodGet, rutaReglasSituacionBolsaDesarrollo},
		{http.MethodGet, rutaOrganizacionContratacionTemporalDesarrollo},
		{http.MethodGet, rutaCatalogosAltaContratacionTemporalDesarrollo},
		{http.MethodGet, rutaCircuitoFirmaContratacionTemporalDesarrollo},
		{http.MethodGet, rutaConfiguracionAnalisisContratacionTemporalDesarrollo},
		{http.MethodGet, "/api/vec/reglas/vigentes"},
		{http.MethodPost, cthttp.RutaCesesNombramiento},
		{http.MethodGet, cthttp.RutaNoIncorporaciones},
	} {
		if got := estado(caso.metodo, caso.ruta); got != http.StatusForbidden {
			t.Errorf("%s %s con la hoja del segundo técnico = %d", caso.metodo, caso.ruta, got)
		}
	}
	if got := estado(http.MethodPost, cthttp.RutaNoIncorporaciones); got != http.StatusOK {
		t.Fatalf("POST de no incorporación sin capacidad sellada: %d", got)
	}
	// Mismo sujeto con otra hoja: no es el segundo técnico y no hereda su límite.
	actual.Attributes = map[string]string{"autoridad": AutoridadNoAutoritativa,
		"perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("d", 64)}
	if resolvedor.esSegundoTecnicoRRHH(actual) {
		t.Fatal("hoja distinta reconocida como segundo técnico")
	}
}
