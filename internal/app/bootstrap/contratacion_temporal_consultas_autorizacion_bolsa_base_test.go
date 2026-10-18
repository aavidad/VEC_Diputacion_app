package bootstrap

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
)

func TestFronteraBolsaBaseConservaCuadroYSeparaPerfilPropio(t *testing.T) {
	alta, _, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	certificado := &x509.Certificate{
		Raw:       []byte("certificado_rrhh_bolsa_base_sintetico"),
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(90 * 24 * time.Hour),
	}
	huella := sha256.Sum256(certificado.Raw)
	principal.Attributes["certificate_sha256"] = hex.EncodeToString(huella[:])
	resolvedor, err := nuevoResolvedorIdentidadDesarrollo(identidadCertificadoDesarrollo{
		huella: huella, principal: principal,
	})
	if err != nil {
		t.Fatal(err)
	}
	fronteras, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo("prf_bolsa_bback")
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadConsultasContratacionTemporalDesarrollo{
		sello: alta.soporte.sello, resolvedor: resolvedor,
	}
	// Esta capa acredita la sesión; el manejador conserva la validación de filtros.
	for _, caso := range []struct {
		nombre, metodo, ruta string
		autenticada          bool
		frontera             bool
	}{
		{"GET perfil propio", http.MethodGet, bolsahttp.RutaBolsasGestion + "?perfil=propio", true, true},
		{"HEAD perfil propio", http.MethodHead, bolsahttp.RutaBolsasGestion + "?perfil=propio", true, true},
		{"GET cuadro sin consulta", http.MethodGet, bolsahttp.RutaBolsasGestion, true, false},
		{"HEAD cuadro sin consulta", http.MethodHead, bolsahttp.RutaBolsasGestion, true, false},
		{"GET filtro global", http.MethodGet, bolsahttp.RutaBolsasGestion + "?filtro=todos&corte=corte_prueba", true, false},
		{"GET duplicada", http.MethodGet, bolsahttp.RutaBolsasGestion + "?perfil=propio&perfil=propio", true, false},
		{"GET extra", http.MethodGet, bolsahttp.RutaBolsasGestion + "?perfil=propio&limite=1", true, false},
		{"GET clave codificada", http.MethodGet, bolsahttp.RutaBolsasGestion + "?%70erfil=propio", true, false},
		{"GET valor codificado", http.MethodGet, bolsahttp.RutaBolsasGestion + "?perfil=propio%20", true, false},
		{"GET detalle existente", http.MethodGet, bolsahttp.RutaBolsasGestion + "/bolsa:01/candidatos?limite=1", true, true},
		{"GET estadísticas existentes", http.MethodGet, rutaEstadisticasBolsaRRHHDesarrollo, true, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			peticion := httptest.NewRequest(caso.metodo, "https://localhost"+caso.ruta, nil)
			peticion.RemoteAddr = "127.0.0.1:41000"
			peticion.TLS = &tls.ConnectionState{
				HandshakeComplete: true, Version: tls.VersionTLS13,
				PeerCertificates: []*x509.Certificate{certificado},
				VerifiedChains:   [][]*x509.Certificate{{certificado, {}}},
			}
			despachos := 0
			llamadas := 0
			revalidador := &revalidadorConsultasContratacionTemporalDesarrollo{
				autoridad: autoridad, fronteras: catalogo,
				siguiente: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					llamadas++
					capacidad, sellada := r.Context().Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
					actual, frontera := fronteraSeguridadComunDesdeContexto(r.Context())
					if sellada != caso.autenticada || frontera != caso.frontera ||
						(frontera && (!actual.descriptor.admitePerfil("prf_bolsa_bback") ||
							actual.descriptor.admitePerfil("prf_ct_ajeno"))) ||
						(sellada && (capacidad.sello != autoridad.sello || capacidad.ruta != r.URL.Path)) {
						t.Errorf("capacidad o frontera inesperada: sellada=%t frontera=%t", sellada, frontera)
					}
					if err := autoridad.AutorizarRutaExacta(r.Context(), r.URL.Path); err != nil {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					despachos++
					w.WriteHeader(http.StatusOK)
				}),
			}
			respuesta := httptest.NewRecorder()
			revalidador.ServeHTTP(respuesta, peticion)
			esperadas := 0
			if caso.autenticada {
				esperadas = 1
			}
			if llamadas != 1 || despachos != esperadas {
				t.Fatalf("capacidad del cuadro o frontera adicional incorrecta: manejador=%d despacho=%d", llamadas, despachos)
			}
			if caso.autenticada && respuesta.Code != http.StatusOK ||
				!caso.autenticada && respuesta.Code != http.StatusUnauthorized {
				t.Fatalf("estado=%d, autenticada=%t", respuesta.Code, caso.autenticada)
			}
		})
	}
	for _, consulta := range []string{"", "?perfil=propio"} {
		t.Run("sin certificado"+consulta, func(t *testing.T) {
			peticion := httptest.NewRequest(http.MethodGet, "https://localhost"+bolsahttp.RutaBolsasGestion+consulta, nil)
			revalidador := &revalidadorConsultasContratacionTemporalDesarrollo{
				autoridad: autoridad, fronteras: catalogo,
				siguiente: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if autoridad.AutorizarRutaExacta(r.Context(), r.URL.Path) == nil {
						t.Fatal("se autenticó la ruta sin certificado")
					}
					w.WriteHeader(http.StatusUnauthorized)
				}),
			}
			respuesta := httptest.NewRecorder()
			revalidador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusUnauthorized {
				t.Fatalf("estado sin certificado=%d", respuesta.Code)
			}
		})
	}
}
