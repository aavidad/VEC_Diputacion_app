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

func TestFronteraBolsaBaseSoloSellaConsultaExactaDePerfilPropio(t *testing.T) {
	alta, _, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	certificado := &x509.Certificate{
		Raw:       []byte("certificado_rrhh_bolsa_base_sintetico"),
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
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
	for _, caso := range []struct {
		nombre, metodo, ruta string
		admitida             bool
		frontera             bool
	}{
		{"GET perfil propio", http.MethodGet, bolsahttp.RutaBolsasGestion + "?perfil=propio", true, true},
		{"HEAD perfil propio", http.MethodHead, bolsahttp.RutaBolsasGestion + "?perfil=propio", true, true},
		{"GET cuadro sin consulta", http.MethodGet, bolsahttp.RutaBolsasGestion, false, false},
		{"HEAD cuadro sin consulta", http.MethodHead, bolsahttp.RutaBolsasGestion, false, false},
		{"GET filtro global", http.MethodGet, bolsahttp.RutaBolsasGestion + "?filtro=todos&corte=corte_prueba", false, false},
		{"GET duplicada", http.MethodGet, bolsahttp.RutaBolsasGestion + "?perfil=propio&perfil=propio", false, false},
		{"GET extra", http.MethodGet, bolsahttp.RutaBolsasGestion + "?perfil=propio&limite=1", false, false},
		{"GET clave codificada", http.MethodGet, bolsahttp.RutaBolsasGestion + "?%70erfil=propio", false, false},
		{"GET valor codificado", http.MethodGet, bolsahttp.RutaBolsasGestion + "?perfil=propio%20", false, false},
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
			lecturas := 0
			llamadas := 0
			revalidador := &revalidadorConsultasContratacionTemporalDesarrollo{
				autoridad: autoridad, fronteras: catalogo,
				siguiente: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					llamadas++
					capacidad, sellada := r.Context().Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
					actual, frontera := fronteraSeguridadComunDesdeContexto(r.Context())
					if sellada != caso.admitida || frontera != caso.frontera ||
						(frontera && (!actual.descriptor.admitePerfil("prf_bolsa_bback") ||
							actual.descriptor.admitePerfil("prf_ct_ajeno"))) ||
						(sellada && (capacidad.sello != autoridad.sello || capacidad.ruta != r.URL.Path)) {
						t.Errorf("capacidad o frontera inesperada: sellada=%t frontera=%t", sellada, frontera)
					}
					if err := autoridad.AutorizarRutaExacta(r.Context(), r.URL.Path); err != nil {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					lecturas++
					w.WriteHeader(http.StatusOK)
				}),
			}
			respuesta := httptest.NewRecorder()
			revalidador.ServeHTTP(respuesta, peticion)
			esperadas := 0
			if caso.admitida {
				esperadas = 1
			}
			if llamadas != 1 || lecturas != esperadas {
				t.Fatalf("se alcanzó la lectura fuera de la frontera: manejador=%d lectura=%d", llamadas, lecturas)
			}
			if caso.admitida && respuesta.Code != http.StatusOK ||
				!caso.admitida && respuesta.Code != http.StatusUnauthorized {
				t.Fatalf("estado=%d, admitida=%t", respuesta.Code, caso.admitida)
			}
		})
	}
}
