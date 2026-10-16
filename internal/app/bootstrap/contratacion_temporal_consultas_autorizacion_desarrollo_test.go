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
