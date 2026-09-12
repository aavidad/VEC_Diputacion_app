package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	adminmodule "vec-diputacion-granada/internal/modules/administracion"
	adminhttp "vec-diputacion-granada/internal/modules/administracion/adapters/http"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type proveedorAdministracionDesarrolloPrueba struct {
	mu        sync.Mutex
	principal vecdomain.Principal
	err       error
	llamadas  int
	capacidad bool
	despues   func(*vecdomain.Principal)
}

func (p *proveedorAdministracionDesarrolloPrueba) PrincipalAdministracionDesarrollo(ctx context.Context) (vecdomain.Principal, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.llamadas++
	_, p.capacidad = capacidadAdministracionDesdeContexto(ctx)
	if !p.capacidad {
		return vecdomain.Principal{}, errors.New("capacidad administrativa ausente")
	}
	principal := clonarPrincipalDesarrollo(p.principal)
	if p.despues != nil {
		p.despues(&p.principal)
	}
	return principal, p.err
}

func (p *proveedorAdministracionDesarrolloPrueba) observacion() (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.llamadas, p.capacidad
}

type materialTLSAdministracionPrueba struct {
	servidor tls.Certificate
	admin    tls.Certificate
	otro     tls.Certificate
	raices   *x509.CertPool
	huella   [sha256.Size]byte
}

func nuevoMaterialTLSAdministracionPrueba(t *testing.T) materialTLSAdministracionPrueba {
	t.Helper()
	ahora := time.Now().UTC()
	caPub, caKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ca := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkixName("vec-ca-prueba"), NotBefore: ahora.Add(-time.Minute), NotAfter: ahora.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	caDER, err := x509.CreateCertificate(rand.Reader, ca, ca, caPub, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	crear := func(serial int64, servidor bool) tls.Certificate {
		t.Helper()
		pub, key, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkixName("vec-prueba"), NotBefore: ahora.Add(-time.Minute), NotAfter: ahora.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature}
		if servidor {
			plantilla.DNSNames = []string{"localhost"}
			plantilla.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		} else {
			plantilla.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, caCert, pub, caKey)
		if err != nil {
			t.Fatal(err)
		}
		return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	}
	admin := crear(2, false)
	raices := x509.NewCertPool()
	raices.AddCert(caCert)
	return materialTLSAdministracionPrueba{servidor: crear(3, true), admin: admin, otro: crear(4, false), raices: raices, huella: sha256.Sum256(admin.Certificate[0])}
}

// pkix.Name is deliberately hidden behind this small helper so the TLS fixture
// remains entirely ephemeral and does not use the development material loader.
func pkixName(nombre string) pkix.Name { return pkix.Name{CommonName: nombre} }

func nuevaAutoridadAdministracionPrueba(t *testing.T, m materialTLSAdministracionPrueba, permiso bool) (*autoridadAdministracionDesarrollo, *proveedorAdministracionDesarrolloPrueba, config.Config) {
	t.Helper()
	permisos := []string(nil)
	if permiso {
		permisos = []string{adminmodule.PermissionIntegrationsManage}
	}
	identidad := vecdomain.Principal{ID: "per_aaaaaaaaaaaaaaaaaaaaaa", DisplayName: "Administración sintética", Roles: []string{"administrador"}, AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh, Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": fmt.Sprintf("%x", m.huella)}}
	res, err := nuevoResolvedorAdministracionDesarrollo(&identidadAdministracionDesarrollo{identidad: identidadCertificadoDesarrollo{huella: m.huella, principal: identidad}, cuentaRef: "cta_bbbbbbbbbbbbbbbbbbbbbb", cuentaOrdinariaRef: "cta_cccccccccccccccccccccc", perfilRef: "prf_dddddddddddddddddddddd", personaRef: identidad.ID})
	if err != nil {
		t.Fatal(err)
	}
	principalProveedor := clonarPrincipalDesarrollo(identidad)
	principalProveedor.Permissions = permisos
	proveedor := &proveedorAdministracionDesarrolloPrueba{principal: principalProveedor}
	cfg := config.Config{Address: "127.0.0.1:0", HTTPAllowedCIDRs: []string{"127.0.0.1/32"}, ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement, DevelopmentMaterialDir: t.TempDir()}
	a, err := nuevaAutoridadAdministracionDesarrollo(cfg, res, proveedor)
	if err != nil {
		t.Fatal(err)
	}
	return a, proveedor, cfg
}

func iniciarServidorAdministracionPrueba(t *testing.T, a *autoridadAdministracionDesarrollo, cfg config.Config, m materialTLSAdministracionPrueba, handler http.Handler) string {
	t.Helper()
	s := &http.Server{Addr: cfg.Address, Handler: handler, TLSConfig: &tls.Config{Certificates: []tls.Certificate{m.servidor}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: m.raices, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13}}
	if err := a.vincularServidor(s); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = s.Serve(tls.NewListener(ln, s.TLSConfig)) }()
	t.Cleanup(func() { _ = s.Close() })
	return "https://" + ln.Addr().String()
}

func clienteAdministracionPrueba(m materialTLSAdministracionPrueba, certificado *tls.Certificate) *http.Client {
	tlsCfg := &tls.Config{RootCAs: m.raices, ServerName: "localhost", MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13}
	if certificado != nil {
		tlsCfg.Certificates = []tls.Certificate{*certificado}
	}
	return &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}}
}

func TestAutoridadAdministracionDesarrolloTLSLoopback(t *testing.T) {
	m := nuevoMaterialTLSAdministracionPrueba(t)
	a, proveedor, cfg := nuevaAutoridadAdministracionPrueba(t, m, true)
	var mu sync.Mutex
	var contextoCapturado context.Context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ajena" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if _, ok := capacidadAdministracionDesdeContexto(r.Context()); !ok {
			t.Error("el manejador no recibió capacidad activa")
		}
		if _, err := a.PrincipalConfiguracionCorreo(r.Context()); err != nil {
			t.Errorf("principal acreditado: %v", err)
		}
		mu.Lock()
		contextoCapturado = r.Context()
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	url := iniciarServidorAdministracionPrueba(t, a, cfg, m, handler)
	cliente := clienteAdministracionPrueba(m, &m.admin)
	respuesta, err := cliente.Get(url + adminhttp.RutaConfiguracionCorreo)
	if err != nil {
		t.Fatal(err)
	}
	respuesta.Body.Close()
	llamadas, capacidad := proveedor.observacion()
	if respuesta.StatusCode != http.StatusNoContent || llamadas != 1 || !capacidad {
		t.Fatalf("aceptación TLS=%d llamadas=%d capacidad=%v", respuesta.StatusCode, llamadas, capacidad)
	}
	mu.Lock()
	capturado := contextoCapturado
	mu.Unlock()
	if _, ok := capacidadAdministracionDesdeContexto(capturado); ok {
		t.Fatal("la capacidad sobrevivió fuera de ServeHTTP")
	}
	respuesta, err = cliente.Get(url + "/ajena")
	if err != nil {
		t.Fatal(err)
	}
	respuesta.Body.Close()
	llamadas, _ = proveedor.observacion()
	if respuesta.StatusCode != http.StatusNoContent || llamadas != 1 {
		t.Fatalf("ruta ajena alterada: %d llamadas=%d", respuesta.StatusCode, llamadas)
	}
}

func TestAdministracionCatalogosRechazanCertificadosCruzadosEnTLS(t *testing.T) {
	m := nuevoMaterialTLSAdministracionPrueba(t)
	a, proveedor, cfg := nuevaAutoridadAdministracionPrueba(t, m, true)
	admin, err := nuevoResolvedorAdministracionDesarrollo(a.identidad)
	if err != nil {
		t.Fatal(err)
	}
	principalRRHH := clonarPrincipalDesarrollo(a.identidad.identidad.principal)
	principalRRHH.ID = "rrhh-sintetico-separado"
	principalRRHH.Roles = []string{"tecnico_rrhh"}
	huellaRRHH := sha256.Sum256(m.otro.Certificate[0])
	principalRRHH.Attributes["certificate_sha256"] = fmt.Sprintf("%x", huellaRRHH)
	rrhh, err := nuevoResolvedorIdentidadDesarrollo(identidadCertificadoDesarrollo{huella: huellaRRHH, principal: principalRRHH})
	if err != nil {
		t.Fatal(err)
	}
	for nombre, catalogo := range map[string]*resolvedorIdentidadDesarrollo{"ADMIN": admin, "RRHH": rrhh} {
		t.Run(nombre, func(t *testing.T) {
			// TLS real verifica ambos certificados de la CA; el catálogo debe
			// aceptar únicamente la identidad de su superficie.
			url := iniciarServidorAdministracionPrueba(t, a, cfg, m, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := catalogo.ResolveDemoIdentity(r.Context(), r); err != nil {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			for identidad, cert := range map[string]*tls.Certificate{"ADMIN": &m.admin, "RRHH": &m.otro} {
				cliente := clienteAdministracionPrueba(m, cert)
				defer cliente.CloseIdleConnections()
				respuesta, err := cliente.Get(url + "/comprobacion-catalogo")
				if err != nil {
					t.Fatal(err)
				}
				respuesta.Body.Close()
				quiere := http.StatusForbidden
				if identidad == nombre {
					quiere = http.StatusNoContent
				}
				if respuesta.StatusCode != quiere {
					t.Fatalf("catálogo %s/certificado %s: %d, esperado %d", nombre, identidad, respuesta.StatusCode, quiere)
				}
			}
		})
		// Cada servidor tiene su propia autoridad: una autoridad vinculada no
		// puede reutilizarse como frontera de un segundo listener.
		a, _, cfg = nuevaAutoridadAdministracionPrueba(t, m, true)
	}
	admin.porHuella[huellaRRHH] = principalRRHH
	if _, err := nuevaAutoridadAdministracionDesarrollo(cfg, admin, proveedor); err == nil {
		t.Fatal("autoridad ADMIN aceptó catálogo mezclado con RRHH")
	}
}

func TestAdministracionActivosExigenCertificadoYPermisoSinCapacidadSMTP(t *testing.T) {
	m := nuevoMaterialTLSAdministracionPrueba(t)
	a, proveedor, cfg := nuevaAutoridadAdministracionPrueba(t, m, true)
	ajena, _, _ := nuevaAutoridadAdministracionPrueba(t, m, true)
	_ = iniciarServidorAdministracionPrueba(t, ajena, cfg, m, http.NotFoundHandler())
	principalSMTP := clonarPrincipalDesarrollo(proveedor.principal)
	url := iniciarServidorAdministracionPrueba(t, a, cfg, m, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capacidad, ok := capacidadAdministracionDesdeContexto(r.Context())
		if !ok || capacidad.ruta != r.URL.Path || capacidad.metodo != r.Method || !rutaLecturaAdministracionDesarrollo(capacidad.ruta, capacidad.metodo) {
			t.Error("lectura UI sin ruta y método originales acreditados")
		}
		if _, ok := ajena.capacidadValida(r.Context()); ok {
			t.Error("la capacidad cruzó de autoridad")
		}
		if _, err := a.PrincipalConfiguracionCorreo(r.Context()); err == nil {
			t.Error("lectura UI reutilizada como principal API SMTP")
		}
		if err := a.VerificarAccesoConfiguracionCorreo(r.Context(), principalSMTP); err == nil {
			t.Error("lectura UI reutilizada como autorización API SMTP")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, ruta := range []string{"/administracion", "/administracion/", "/administracion/administracion.js", "/administracion/tema.css"} {
		for _, metodo := range []string{http.MethodGet, http.MethodHead, http.MethodPut} {
			for nombre, certificado := range map[string]*tls.Certificate{"ADMIN": &m.admin, "RRHH": &m.otro} {
				cliente := clienteAdministracionPrueba(m, certificado)
				peticion, err := http.NewRequest(metodo, url+ruta, nil)
				if err != nil {
					t.Fatal(err)
				}
				respuesta, err := cliente.Do(peticion)
				if err != nil {
					t.Fatal(err)
				}
				respuesta.Body.Close()
				cliente.CloseIdleConnections()
				quiere := http.StatusForbidden
				if metodo == http.MethodPut {
					quiere = http.StatusNotFound
				}
				if nombre == "ADMIN" && metodo != http.MethodPut {
					quiere = http.StatusNoContent
				}
				if respuesta.StatusCode != quiere {
					t.Fatalf("%s %s con %s: %d, esperado %d", metodo, ruta, nombre, respuesta.StatusCode, quiere)
				}
			}
		}
	}
	if llamadas, _ := proveedor.observacion(); llamadas != 8 {
		t.Fatalf("lecturas no revalidaron el permiso: %d", llamadas)
	}
	proveedor.mu.Lock()
	proveedor.principal.Permissions = nil
	proveedor.mu.Unlock()
	cliente := clienteAdministracionPrueba(m, &m.admin)
	defer cliente.CloseIdleConnections()
	respuesta, err := cliente.Get(url + "/administracion/")
	if err != nil {
		t.Fatal(err)
	}
	respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusForbidden {
		t.Fatal("interfaz accesible tras revocar el permiso")
	}
}

func TestAutoridadAdministracionDesarrolloDeniegaCanalesNoAcreditados(t *testing.T) {
	for _, caso := range []struct {
		nombre      string
		certificado func(materialTLSAdministracionPrueba) *tls.Certificate
		cabecera    bool
		directo     bool
		permiso     bool
	}{
		{"otro certificado", func(m materialTLSAdministracionPrueba) *tls.Certificate { return &m.otro }, false, false, true},
		{"sin certificado", func(materialTLSAdministracionPrueba) *tls.Certificate { return nil }, false, false, true},
		{"cabecera proxy", func(m materialTLSAdministracionPrueba) *tls.Certificate { return &m.admin }, true, false, true},
		{"sin permiso fresco", func(m materialTLSAdministracionPrueba) *tls.Certificate { return &m.admin }, false, false, false},
		{"handler directo TLS fabricado", func(materialTLSAdministracionPrueba) *tls.Certificate { return nil }, false, true, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			m := nuevoMaterialTLSAdministracionPrueba(t)
			a, proveedor, cfg := nuevaAutoridadAdministracionPrueba(t, m, caso.permiso)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := a.PrincipalConfiguracionCorreo(r.Context()); err != nil {
					http.Error(w, "acceso denegado", http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			if caso.directo {
				s := &http.Server{Addr: cfg.Address, Handler: handler, TLSConfig: &tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: m.raices, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13}}
				if err := a.vincularServidor(s); err != nil {
					t.Fatal(err)
				}
				r := httptest.NewRequest(http.MethodGet, "https://localhost"+adminhttp.RutaConfiguracionCorreo, nil)
				r.RemoteAddr = "127.0.0.1:40000"
				r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13}
				w := httptest.NewRecorder()
				s.Handler.ServeHTTP(w, r)
				llamadas, _ := proveedor.observacion()
				if w.Code != http.StatusForbidden || llamadas != 0 {
					t.Fatalf("directo=%d llamadas=%d", w.Code, llamadas)
				}
				return
			}
			url := iniciarServidorAdministracionPrueba(t, a, cfg, m, handler)
			r, err := http.NewRequest(http.MethodGet, url+adminhttp.RutaConfiguracionCorreo, nil)
			if err != nil {
				t.Fatal(err)
			}
			if caso.cabecera {
				r.Header.Set("X-Forwarded-For", "127.0.0.1")
			}
			respuesta, err := clienteAdministracionPrueba(m, caso.certificado(m)).Do(r)
			if caso.nombre == "sin certificado" {
				if err == nil {
					respuesta.Body.Close()
					t.Fatal("mTLS aceptó cliente sin certificado")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			respuesta.Body.Close()
			if respuesta.StatusCode != http.StatusForbidden {
				t.Fatalf("estado=%d", respuesta.StatusCode)
			}
			quiereLlamada := caso.nombre == "sin permiso fresco"
			llamadas, _ := proveedor.observacion()
			if (llamadas == 1) != quiereLlamada {
				t.Fatalf("llamadas=%d quiere=%v", llamadas, quiereLlamada)
			}
		})
	}
}

func TestAutoridadAdministracionDesarrolloExigeServidorTLSExacto(t *testing.T) {
	m := nuevoMaterialTLSAdministracionPrueba(t)
	a, _, cfg := nuevaAutoridadAdministracionPrueba(t, m, true)
	for _, s := range []*http.Server{
		{Addr: "127.0.0.1:1", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), TLSConfig: &tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: m.raices, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13}},
		{Addr: cfg.Address, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), TLSConfig: &tls.Config{ClientAuth: tls.RequireAnyClientCert, ClientCAs: m.raices, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13}},
	} {
		if err := a.vincularServidor(s); !errors.Is(err, ErrConfiguracionCorreoAdministracionNoDisponible) {
			t.Fatalf("servidor inseguro admitido: %v", err)
		}
	}
}

func TestAutoridadAdministracionDesarrolloRevalidaProveedorEnLaPeticionTLS(t *testing.T) {
	t.Run("identidad ajena", func(t *testing.T) {
		m := nuevoMaterialTLSAdministracionPrueba(t)
		a, proveedor, cfg := nuevaAutoridadAdministracionPrueba(t, m, true)
		proveedor.mu.Lock()
		proveedor.principal.ID = "per_bbbbbbbbbbbbbbbbbbbbbb"
		proveedor.mu.Unlock()
		url := iniciarServidorAdministracionPrueba(t, a, cfg, m, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := a.PrincipalConfiguracionCorreo(r.Context()); err == nil {
				t.Error("proveedor con identidad ajena fue admitido")
			}
			http.Error(w, "acceso denegado", http.StatusForbidden)
		}))
		respuesta, err := clienteAdministracionPrueba(m, &m.admin).Get(url + adminhttp.RutaConfiguracionCorreo)
		if err != nil {
			t.Fatal(err)
		}
		respuesta.Body.Close()
		llamadas, _ := proveedor.observacion()
		if respuesta.StatusCode != http.StatusForbidden || llamadas != 1 {
			t.Fatalf("estado=%d llamadas=%d", respuesta.StatusCode, llamadas)
		}
	})
	t.Run("revocación dentro de petición", func(t *testing.T) {
		m := nuevoMaterialTLSAdministracionPrueba(t)
		a, proveedor, cfg := nuevaAutoridadAdministracionPrueba(t, m, true)
		proveedor.mu.Lock()
		proveedor.despues = func(principal *vecdomain.Principal) { principal.Permissions = nil }
		proveedor.mu.Unlock()
		url := iniciarServidorAdministracionPrueba(t, a, cfg, m, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := a.PrincipalConfiguracionCorreo(r.Context()); err != nil {
				t.Fatalf("primera comprobación denegada: %v", err)
			}
			if _, err := a.PrincipalConfiguracionCorreo(r.Context()); err == nil {
				t.Error("la revocación de permiso no se revalidó")
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		respuesta, err := clienteAdministracionPrueba(m, &m.admin).Get(url + adminhttp.RutaConfiguracionCorreo)
		if err != nil {
			t.Fatal(err)
		}
		respuesta.Body.Close()
		llamadas, _ := proveedor.observacion()
		if respuesta.StatusCode != http.StatusNoContent || llamadas != 2 {
			t.Fatalf("estado=%d llamadas=%d", respuesta.StatusCode, llamadas)
		}
	})
}

func TestAutoridadAdministracionDesarrolloRechazaCapacidadesYExportadoresFabricados(t *testing.T) {
	if _, ok := capacidadAdministracionDesdeContexto(context.Background()); ok {
		t.Fatal("un contexto aislado obtuvo capacidad administrativa")
	}
	m := nuevoMaterialTLSAdministracionPrueba(t)
	a, _, cfg := nuevaAutoridadAdministracionPrueba(t, m, true)
	s := &http.Server{Addr: cfg.Address, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), TLSConfig: &tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: m.raices, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13}}
	if err := a.vincularServidor(s); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "https://localhost"+adminhttp.RutaConfiguracionCorreo, nil)
	r.RemoteAddr = "127.0.0.1:40001"
	r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13}
	r = r.WithContext(context.WithValue(r.Context(), claveConexionAdministracionDesarrollo{}, conexionAdministracionDesarrollo{autoridad: a, conexion: &tls.Conn{}}))
	if capacidad, ok := a.acreditarPeticion(r); ok || capacidad != nil {
		t.Fatal("un exportador TLS fabricado emitió capacidad")
	}
}
