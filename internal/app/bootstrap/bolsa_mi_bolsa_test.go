package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	bolsapersonal "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestMiBolsaCargaSoloCertificadoCandidatoNominal(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	certificado := filepath.Join(cfg.DevelopmentMaterialDir, "mtls", "candidato.crt")
	clave := filepath.Join(cfg.DevelopmentMaterialDir, "mtls", "candidato.key")
	csr := filepath.Join(cfg.DevelopmentMaterialDir, "mtls", "candidato.csr")
	extension := filepath.Join(cfg.DevelopmentMaterialDir, "mtls", "candidato.ext")
	contenidoExtension := []byte("basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature\nextendedKeyUsage=clientAuth\nsubjectAltName=URI:urn:vec:desarrollo:candidato-bolsa\n")
	if err := os.WriteFile(extension, contenidoExtension, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, argumentos := range [][]string{
		{"genpkey", "-algorithm", "EC", "-pkeyopt", "ec_paramgen_curve:P-256", "-out", clave},
		{"req", "-new", "-sha256", "-key", clave, "-subj", "/CN=candidato-bolsa-desarrollo/O=VEC Desarrollo/OU=NO AUTORITATIVO", "-out", csr},
		{"x509", "-req", "-sha256", "-days", "30", "-in", csr, "-CA", rutas.CACertificate, "-CAkey", rutas.CAPrivateKey, "-CAserial", filepath.Join(cfg.DevelopmentMaterialDir, "ca", "serie"), "-extfile", extension, "-out", certificado},
	} {
		if salida, err := exec.Command("openssl", argumentos...).CombinedOutput(); err != nil {
			t.Fatalf("certificado sintético: %v: %s", err, salida)
		}
	}
	if err := os.Chmod(certificado, 0o600); err != nil {
		t.Fatal(err)
	}
	pem, err := os.ReadFile(certificado)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := decodificarCertificadoUnico(pem)
	if err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(cert.Raw)
	sujeto := "per_candidato_sintetico_1234567890123456"
	identidad := map[string]any{"version": 1, "autoridad": AutoridadNoAutoritativa,
		"certificate_sha256": hexHuellaMiBolsaPrueba(huella), "subject": sujeto,
		"display_name": "Persona candidata", "roles": []string{"candidato_bolsa"}}
	rutaIdentidad := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "candidato.json")
	contenido, _ := json.Marshal(identidad)
	if err := os.WriteFile(rutaIdentidad, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	base := sujeto + "\x00" + hexHuellaMiBolsaPrueba(huella)
	manifiesto := map[string]any{"version": 1, "autoridad": AutoridadNoAutoritativa,
		"certificado": "mtls/candidato.crt", "identidad": "identidad/candidato.json",
		"sujeto": sujeto, "cuenta_ref": referenciaAltaContratacionTemporalDesarrollo("cta_", base+"\x00cuenta"),
		"persona_ref":   referenciaAltaContratacionTemporalDesarrollo("per_", base+"\x00persona"),
		"perfil_ref":    "prf_candidato_sintetico_1234567890123456",
		"candidato_ref": "can_candidato_sintetico_1234567890123456"}
	rutaManifiesto := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "bolsa-candidato.json")
	contenido, _ = json.Marshal(manifiesto)
	if err := os.WriteFile(rutaManifiesto, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	compuesta, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatalf("identidad personal no compuesta: %v", err)
	}
	resolver, ok := compuesta.identidad.(*resolvedorIdentidadDesarrollo)
	if !ok || resolver.candidatoBolsa == nil {
		t.Fatal("identidad personal no compuesta")
	}
	if resolver.candidatoBolsa.candidatoRef != manifiesto["candidato_ref"] {
		t.Fatal("el candidato nominal no procede del manifiesto privado")
	}
	identidad["roles"] = []string{"tecnico_rrhh"}
	contenido, _ = json.Marshal(identidad)
	if err := os.WriteFile(rutaIdentidad, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard); err == nil {
		t.Fatal("un certificado candidato con rol interno abrió el perfil")
	}
}

func TestMiBolsaFronteraSeparaCertificadosExternosEInternos(t *testing.T) {
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
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{
		Clave: "bolsa-mi-bolsa-consultar", Superficie: superficieExternaPersonalSeguridadComunDesarrollo,
		Metodo: http.MethodGet, Ruta: bolsapersonal.RutaMiBolsa,
		PerfilesActivosRef: []string{identidad.perfilRef}, ClavePolitica: "politica-bolsa-mi-bolsa", ClaveCapacidad: "capacidad-bolsa-mi-bolsa-consultar",
	}})
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
		r := httptest.NewRequest(http.MethodGet, ruta, nil)
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
		{"candidato propia", certCandidato, bolsapersonal.RutaMiBolsa, http.StatusOK},
		{"candidato interna", certCandidato, cthttp.RutaEstadisticasRRHH, http.StatusUnauthorized},
		{"rrhh personal", certRRHH, bolsapersonal.RutaMiBolsa, http.StatusUnauthorized},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if !esRutaContratacionTemporalDesarrollo(peticion(caso.cert, caso.ruta)) {
				t.Fatal("ruta ausente de la lista mTLS")
			}
			w := httptest.NewRecorder()
			revalidador.ServeHTTP(w, peticion(caso.cert, caso.ruta).WithContext(context.Background()))
			if w.Code != caso.estado {
				t.Fatalf("estado = %d; esperado %d", w.Code, caso.estado)
			}
		})
	}
}

func TestMiBolsaPoliticaLimitaCampoYCandidato(t *testing.T) {
	identidad := &identidadCandidatoBolsaDesarrollo{
		personaRef:   "per_candidato_sintetico_1234567890123456",
		perfilRef:    "prf_candidato_sintetico_1234567890123456",
		candidatoRef: "can_candidato_sintetico_1234567890123456",
	}
	instantanea, err := nuevaInstantaneaMiBolsaDesarrollo(identidad, time.Now().UTC())
	if err != nil || instantanea.Validar() != nil {
		t.Fatalf("política candidata inválida: %v", err)
	}
	concesiones := instantanea.VersionRol.Concesiones
	if len(concesiones) != 1 || len(concesiones[0].CamposPermitidos) != 1 ||
		concesiones[0].CamposPermitidos[0] != puertosbolsa.CampoMiBolsa ||
		len(instantanea.AsignacionPerfil.Ambitos) != 1 ||
		instantanea.AsignacionPerfil.Ambitos[0].Clave != "candidato_ref" ||
		len(instantanea.AsignacionPerfil.Ambitos[0].Valores) != 1 ||
		instantanea.AsignacionPerfil.Ambitos[0].Valores[0] != identidad.candidatoRef {
		t.Fatal("la concesión no limita la lectura al campo y candidato propios")
	}
}

func hexHuellaMiBolsaPrueba(v [32]byte) string {
	const digitos = "0123456789abcdef"
	var r [64]byte
	for i, b := range v {
		r[i*2], r[i*2+1] = digitos[b>>4], digitos[b&15]
	}
	return string(r[:])
}
