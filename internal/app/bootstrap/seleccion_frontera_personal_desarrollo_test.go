package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	seleccioninterno "vec-diputacion-granada/internal/modules/seleccion/adapters/httpinterno"
	seleccionpersonal "vec-diputacion-granada/internal/modules/seleccion/adapters/httppersonal"

	"vec-diputacion-granada/config"
	bolsapersonal "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestSeleccionFronteraSeparaPersonaYRRHH(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	certCandidato := &x509.Certificate{Raw: []byte("certificado-candidato-sintetico"), NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour)}
	certRRHH := &x509.Certificate{Raw: []byte("certificado-rrhh-sintetico"), NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour)}
	huellaCandidato := sha256.Sum256(certCandidato.Raw)
	huellaRRHH := sha256.Sum256(certRRHH.Raw)
	principal := func(rol, sujeto string, huella [32]byte) dominiovec.Principal {
		return dominiovec.Principal{ID: sujeto, Roles: []string{rol}, AuthMethod: dominiovec.AuthMethodCertificate,
			AuthAssurance: dominiovec.AuthAssuranceHigh, Attributes: map[string]string{
				"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
				"certificate_sha256": strings.ToLower(hexHuellaMiBolsaPrueba(huella)),
			}}
	}
	candidato := principal("candidato_bolsa", "per_candidato_sintetico_1234567890123456", huellaCandidato)
	rrhh := principal("tecnico_rrhh", "per_rrhh_sintetico_1234567890123456789", huellaRRHH)
	resolvedor, err := nuevoResolvedorIdentidadDesarrollo(
		identidadCertificadoDesarrollo{huella: huellaCandidato, principal: candidato},
		identidadCertificadoDesarrollo{huella: huellaRRHH, principal: rrhh},
	)
	if err != nil {
		t.Fatal(err)
	}
	identidad := identidadCandidatoBolsaDesarrollo{identidad: identidadCertificadoDesarrollo{huella: huellaCandidato, principal: candidato},
		cuentaRef: "cta_candidato_sintetico_1234567890123456", personaRef: candidato.ID,
		perfilRef: "prf_candidato_sintetico_1234567890123456", candidatoRef: "can_candidato_sintetico_1234567890123456",
		validoHasta: certCandidato.NotAfter}
	if err := resolvedor.registrarCandidatoBolsa(identidad); err != nil {
		t.Fatal(err)
	}
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo(append(descriptoresFronterasSeleccionPersonalDesarrollo(identidad.perfilRef),
		descriptoresFronterasSeleccionRRHHDesarrollo(referenciaAltaContratacionTemporalDesarrollo("prf_", "seleccion-rrhh"))...))
	if err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadConsultasContratacionTemporalDesarrollo{sello: sello, resolvedor: resolvedor}
	revalidador := &revalidadorConsultasContratacionTemporalDesarrollo{
		autoridad: autoridad, fronteras: fronteras,
		siguiente: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capacidad, existe := r.Context().Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
			if !existe || capacidad.sello != sello {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	}
	peticion := func(cert *x509.Certificate, ruta string) *http.Request {
		metodo := http.MethodGet
		if ruta == seleccioninterno.RutaConsultas || ruta == seleccionpersonal.RutaPresentacion {
			metodo = http.MethodPost
		}
		r := httptest.NewRequest(metodo, ruta, nil)
		r.RemoteAddr = "127.0.0.1:12345"
		r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
			PeerCertificates: []*x509.Certificate{cert},
			VerifiedChains:   [][]*x509.Certificate{{cert, {Raw: []byte("ca-sintetica")}}}}
		return r
	}
	for _, caso := range []struct {
		nombre string
		cert   *x509.Certificate
		ruta   string
		estado int
	}{
		{"persona en sus solicitudes", certCandidato, seleccionpersonal.RutaMisSolicitudes, http.StatusOK},
		{"persona presenta", certCandidato, seleccionpersonal.RutaPresentacion, http.StatusOK},
		{"persona en la consulta de RRHH", certCandidato, seleccioninterno.RutaConsultas, http.StatusUnauthorized},
		{"rrhh en la superficie personal", certRRHH, seleccionpersonal.RutaMisSolicitudes, http.StatusUnauthorized},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if seleccionpersonal.EsRuta(caso.ruta) && !esRutaContratacionTemporalDesarrollo(peticion(caso.cert, caso.ruta)) {
				t.Fatal("ruta personal ausente de la lista mTLS")
			}
			w := httptest.NewRecorder()
			revalidador.ServeHTTP(w, peticion(caso.cert, caso.ruta).WithContext(context.Background()))
			if w.Code != caso.estado {
				t.Fatalf("estado = %d; esperado %d", w.Code, caso.estado)
			}
		})
	}
}

// La sesión de la persona (soporte de «Mi bolsa») admite todas sus rutas
// personales, también las de Selección; nunca las internas.
func TestSesionPersonaAdmiteRutasPersonalesDeSeleccion(t *testing.T) {
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	principal := dominiovec.Principal{ID: "per_candidato_sintetico_1234567890123456", Roles: []string{"candidato_bolsa"},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": strings.Repeat("a", 64)}}
	soporte := &soporteAltaContratacionTemporalDesarrollo{sello: sello, principalID: principal.ID, certificadoSHA256: strings.Repeat("a", 64), candidatoBolsa: true}
	for ruta, admitida := range map[string]bool{
		bolsapersonal.RutaMiBolsa: true, bolsapersonal.RutaMiBolsaSolicitudes: true,
		seleccionpersonal.RutaBorrador: true, seleccionpersonal.RutaPresentacion: true,
		seleccioninterno.RutaConsultas: false, cthttp.RutaEstadisticasRRHH: false,
	} {
		ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{},
			capacidadConsultaContratacionTemporalDesarrollo{sello: sello, ruta: ruta, principal: principal})
		if _, valida := soporte.capacidadValida(ctx); valida != admitida {
			t.Fatalf("%s: capacidad válida=%v; se esperaba %v", ruta, valida, admitida)
		}
	}
}
