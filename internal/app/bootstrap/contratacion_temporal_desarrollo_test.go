package bootstrap

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	appbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorSeleccionConstituidaDegradadaCT struct {
	puente *puenteBolsaLlamamientoDesarrollo
}

func (e ejecutorSeleccionConstituidaDegradadaCT) SeleccionarYLlamarParaAdaptador(ctx context.Context,
	_ application.SolicitudSeleccionLlamamiento,
) (application.DatosReciboSeleccionLlamamientoParaAdaptador, error) {
	d := preparacionLlamamientoDesarrollo{expediente: ports.ExpedienteParaSeleccion{Fiscalizado: domain.Expediente{
		Analisis: &domain.AnalisisRRHH{}, Fiscalizacion: &domain.FiscalizacionRegistrada{},
	}}}
	_, err := e.puente.prepararOrigenConstituidoSinCacheCT(ctx, d)
	return application.DatosReciboSeleccionLlamamientoParaAdaptador{}, err
}

func TestSeleccionConstituidaConfigRotaDevuelve503SinAbortarMontajeOpcional(t *testing.T) {
	rutaRPTRota := filepath.Join(t.TempDir(), "rpt-invalida.json")
	if err := os.WriteFile(rutaRPTRota, []byte(`{"esquema":"invalido"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre, rpt, etapa, causa string
	}{
		{nombre: "rpt_ausente", etapa: "rpt_ruta", causa: "fuente_ausente"},
		{nombre: "rpt_rota", rpt: rutaRPTRota, etapa: "rpt_publicacion", causa: "publicacion_no_verificada"},
		{nombre: "personal_ausente", rpt: filepath.Join("..", "..", "..", "data", "catalogos", "rpt", "v1.rpt-2026.json"),
			etapa: "personal_consulta", causa: "material_privado_no_admitido"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if caso.rpt != "" {
				if _, err := os.Stat(caso.rpt); err != nil {
					t.Fatalf("la fuente RPT publicada de la prueba no existe: %v", err)
				}
			}
			var salidaLog bytes.Buffer
			anterior := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&salidaLog, nil)))
			defer slog.SetDefault(anterior)
			puente := &puenteBolsaLlamamientoDesarrollo{
				emisorNativo: &appbolsa.ServicioEmisionLlamamiento{},
				auditarOrdenNativo: func(context.Context, preparacionLlamamientoDesarrollo, string) (puertosbolsa.CapturaOrigenCT, error) {
					return puertosbolsa.CapturaOrigenCT{}, nil
				},
			}
			// El proceso común puede ser válido para otras lecturas y ajeno a B110.
			configurarOrigenConstituidoOpcionalCT(&pgxpool.Pool{}, configuracionAuditoriaIntentosDesarrollo{
				Proceso: "otro-proceso", Canal: "interna_corporativa",
			}, puente)
			if puente.origenNativo != nil {
				t.Fatal("proceso ajeno compuso el lector B110")
			}
			configurarOrigenConstituidoOpcionalCT(&pgxpool.Pool{}, configuracionAuditoriaIntentosDesarrollo{
				Proceso: "vec-rrhh", Canal: "otro-canal",
			}, puente)
			if puente.origenNativo != nil {
				t.Fatal("canal ajeno compuso el lector B110")
			}
			// Aisla la degradación M10 en el consumidor: aun con fuente B110
			// presente, los datos no publicados impiden abrir un llamamiento.
			puente.origenNativo = &postgresbolsa.LectorOrigenLlamamientoCTAuditadoPostgreSQL{}
			configurarDatosEmisionConstituidaOpcionalCT(config.Config{RPTCatalogoPath: caso.rpt}, puente, nil,
				func() time.Time { return time.Now().UTC() }, nil)
			if puente.configuracionNativa != nil {
				t.Fatal("se configuraron datos de una fuente privada inválida")
			}
			traza := salidaLog.String()
			if !strings.Contains(traza, "etapa="+caso.etapa) || !strings.Contains(traza, "causa="+caso.causa) ||
				strings.Contains(traza, rutaRPTRota) ||
				strings.Contains(traza, `{"version":`) || strings.Contains(traza, `{"esquema":`) {
				t.Fatalf("registro de degradación incompleto o con material privado: %q", traza)
			}
			handler, err := httpinterno.NuevoManejadorSeleccionLlamamiento(ejecutorSeleccionConstituidaDegradadaCT{puente})
			if err != nil {
				t.Fatal(err)
			}
			peticion := httptest.NewRequest(http.MethodPost, httpinterno.RutaSeleccionLlamamiento,
				strings.NewReader(`{"expediente_ref":"expediente:ct:001","version_esperada":6,"clave_idempotencia":"11111111-1111-4111-8111-111111111111"}`))
			peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
			peticion.Header.Set("Accept", "application/json")
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusServiceUnavailable || !bytes.Contains(respuesta.Body.Bytes(), []byte(`"codigo":"servicio_no_disponible"`)) {
				t.Fatalf("selección degradada: estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
		})
	}
}

func TestConsultasContratacionTemporalDesarrolloDenieganComoNoCompuestasEnListenerTLSReal(
	t *testing.T,
) {
	cfg, rutas := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	servidor, err := NewHTTPServerWithConfig(cfg)
	if err != nil {
		t.Fatalf("componer desarrollo: %v", err)
	}
	escucha, err := net.Listen("tcp", servidor.Addr)
	if err != nil {
		t.Fatalf("abrir listener: %v", err)
	}
	t.Cleanup(func() {
		_ = servidor.Close()
		_ = escucha.Close()
	})
	go func() {
		_ = servidor.ServeTLS(escucha, cfg.TLSCertFile, cfg.TLSKeyFile)
	}()
	baseURL := fmt.Sprintf("https://localhost:%d", escucha.Addr().(*net.TCPAddr).Port)

	cliente := nuevoClienteMTLSContratacionTemporalDesarrollo(t, rutas)
	anadirCadenaCompletaClienteMTLSContratacionTemporalDesarrollo(t, cliente, rutas)
	casos := []struct {
		nombre string
		ruta   string
		cuerpo string
	}{
		{
			nombre: "cuadro",
			ruta:   httpinterno.RutaConsultaCuadroRRHH,
			cuerpo: "{\"filtros\":{\"texto\":\"2026/CT\",\"estado_clave\":\"en_curso\"," +
				"\"fase_clave\":\"analisis\"},\"paginacion\":{\"limite\":50,\"cursor\":\"\"}}",
		},
		{
			nombre: "detalle",
			ruta:   httpinterno.RutaConsultaDetalleRRHH,
			cuerpo: "{\"expediente_ref\":\"" + expedienteContratacionTemporalDesarrolloRef +
				"\",\"version_observada\":3}",
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			peticion, err := http.NewRequest(
				http.MethodPost,
				baseURL+caso.ruta,
				strings.NewReader(caso.cuerpo),
			)
			if err != nil {
				t.Fatal(err)
			}
			peticion.Header.Set("Content-Type", "application/json")
			peticion.Header.Set("Accept", "application/json")
			respuesta, err := cliente.Do(peticion)
			if err != nil {
				t.Fatalf("consulta mTLS: %v", err)
			}
			contenido, err := io.ReadAll(respuesta.Body)
			respuesta.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if respuesta.StatusCode != http.StatusServiceUnavailable ||
				!bytes.Contains(contenido, []byte(`"codigo":"servicio_no_disponible"`)) ||
				bytes.Contains(contenido, []byte(expedienteContratacionTemporalDesarrolloRef)) {
				t.Fatalf("respuesta=%d %s", respuesta.StatusCode, contenido)
			}
		})
	}
}

func TestCarcasaDesarrolloAceptaLaMismaCadenaMTLSVerificada(
	t *testing.T,
) {
	cfg, rutas := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	servidor, err := NewHTTPServerWithConfig(cfg)
	if err != nil {
		t.Fatalf("componer desarrollo: %v", err)
	}
	escucha, err := net.Listen("tcp", servidor.Addr)
	if err != nil {
		t.Fatalf("abrir listener: %v", err)
	}
	t.Cleanup(func() {
		_ = servidor.Close()
		_ = escucha.Close()
	})
	go func() {
		_ = servidor.ServeTLS(escucha, cfg.TLSCertFile, cfg.TLSKeyFile)
	}()
	baseURL := fmt.Sprintf("https://localhost:%d", escucha.Addr().(*net.TCPAddr).Port)

	cliente := nuevoClienteMTLSContratacionTemporalDesarrollo(t, rutas)
	anadirCadenaCompletaClienteMTLSContratacionTemporalDesarrollo(t, cliente, rutas)
	t.Cleanup(func() { cliente.CloseIdleConnections() })

	t.Run("catalogo_de_modulos", func(t *testing.T) {
		peticion, err := http.NewRequest(http.MethodGet, baseURL+"/api/vec/modules", nil)
		if err != nil {
			t.Fatal(err)
		}
		peticion.Header.Set("Accept", "application/json")
		respuesta, err := cliente.Do(peticion)
		if err != nil {
			t.Fatalf("consultar catalogo por mTLS: %v", err)
		}
		contenido, err := io.ReadAll(respuesta.Body)
		respuesta.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if respuesta.StatusCode != http.StatusOK ||
			!bytes.Contains(contenido, []byte(`"id":"vec.module.contratacion_temporal"`)) {
			t.Fatalf("catalogo mTLS=%d %s", respuesta.StatusCode, contenido)
		}
	})

	t.Run("personal_publico_compuesto", func(t *testing.T) {
		// Las consultas públicas de Personal se sirven en la raíz real y no
		// dependen de un permiso de la carcasa; sin fuente, 503 y nunca 404.
		for ruta, esperado := range map[string]int{
			"/api/vec/personal/categories?q=&area=&limit=1&offset=0": http.StatusOK,
			"/api/vec/personal/rpt-publica?q=&limit=1&offset=0":      http.StatusServiceUnavailable,
			"/api/vec/personal/estructura-organizativa-publica":      http.StatusServiceUnavailable,
		} {
			respuesta, err := cliente.Get(baseURL + ruta)
			if err != nil {
				t.Fatalf("consultar %s por mTLS: %v", ruta, err)
			}
			contenido, _ := io.ReadAll(respuesta.Body)
			respuesta.Body.Close()
			if respuesta.StatusCode != esperado {
				t.Fatalf("%s=%d %s", ruta, respuesta.StatusCode, contenido)
			}
		}
	})

	t.Run("autoridad_ambiental_denegada", func(t *testing.T) {
		peticion, err := http.NewRequest(http.MethodGet, baseURL+"/api/vec/modules", nil)
		if err != nil {
			t.Fatal(err)
		}
		peticion.Header.Set("X-Vec-Principal", "administrador")
		respuesta, err := cliente.Do(peticion)
		if err != nil {
			t.Fatal(err)
		}
		contenido, _ := io.ReadAll(respuesta.Body)
		respuesta.Body.Close()
		if respuesta.StatusCode != http.StatusBadRequest ||
			bytes.Contains(contenido, []byte("vec.module.contratacion_temporal")) {
			t.Fatalf("autoridad ambiental=%d %s", respuesta.StatusCode, contenido)
		}
	})

	t.Run("sin_certificado_no_alcanza_la_carcasa", func(t *testing.T) {
		transporte := cliente.Transport.(*http.Transport).Clone()
		transporte.TLSClientConfig = transporte.TLSClientConfig.Clone()
		transporte.TLSClientConfig.Certificates = nil
		sinCertificado := &http.Client{Transport: transporte}
		t.Cleanup(func() { sinCertificado.CloseIdleConnections() })
		if respuesta, err := sinCertificado.Get(baseURL + "/api/vec/modules"); err == nil {
			respuesta.Body.Close()
			t.Fatalf("la carcasa fue accesible sin certificado: %d", respuesta.StatusCode)
		}
	})
}

func anadirCadenaCompletaClienteMTLSContratacionTemporalDesarrollo(
	t *testing.T,
	cliente *http.Client,
	rutas config.DevelopmentMaterialPaths,
) {
	t.Helper()
	caPEM, err := os.ReadFile(rutas.CACertificate)
	if err != nil {
		t.Fatal(err)
	}
	ca, resto := pem.Decode(caPEM)
	if ca == nil || ca.Type != "CERTIFICATE" || len(bytes.TrimSpace(resto)) != 0 {
		t.Fatal("CA de prueba invalida")
	}
	transporte, ok := cliente.Transport.(*http.Transport)
	if !ok || transporte.TLSClientConfig == nil ||
		len(transporte.TLSClientConfig.Certificates) != 1 {
		t.Fatal("cliente mTLS de prueba invalido")
	}
	transporte.TLSClientConfig.Certificates[0].Certificate = append(
		transporte.TLSClientConfig.Certificates[0].Certificate,
		ca.Bytes,
	)
}

func TestPeticionIdentidadConsultasContratacionTemporalDesarrolloSoloNormalizaCadenaVerificada(
	t *testing.T,
) {
	hoja := &x509.Certificate{Raw: []byte("hoja")}
	raiz := &x509.Certificate{Raw: []byte("raiz")}
	peticion := &http.Request{TLS: &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{hoja, raiz},
		VerifiedChains:   [][]*x509.Certificate{{hoja, raiz}},
	}}
	normalizada := peticionIdentidadConsultasContratacionTemporalDesarrollo(peticion)
	if normalizada == peticion || normalizada.TLS == peticion.TLS ||
		len(normalizada.TLS.PeerCertificates) != 1 ||
		normalizada.TLS.PeerCertificates[0] != hoja ||
		len(peticion.TLS.PeerCertificates) != 2 {
		t.Fatal("la cadena verificada no se normalizo mediante una copia aislada")
	}

	adulterada := &http.Request{TLS: &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{hoja, {Raw: []byte("otra")}},
		VerifiedChains:   [][]*x509.Certificate{{hoja, raiz}},
	}}
	if obtenida := peticionIdentidadConsultasContratacionTemporalDesarrollo(adulterada); obtenida != adulterada {
		t.Fatal("una cadena presentada ajena a la verificada fue normalizada")
	}
}

func TestPerimetroMTLSContratacionTemporalDesarrolloIncluyeSoloRutasAnalisisCompuestas(
	t *testing.T,
) {
	t.Parallel()

	for _, ruta := range []string{
		httpinterno.RutaRegistroAnalisisRRHH,
		rutaConfiguracionAnalisisContratacionTemporalDesarrollo,
	} {
		peticion := httptest.NewRequest(http.MethodGet, ruta, nil)
		if !esRutaContratacionTemporalDesarrollo(peticion) {
			t.Fatalf("ruta de analisis fuera del perimetro mTLS: %s", ruta)
		}
		if _, denegada := rutasCapacidadNoCompuestaContratacionTemporal[ruta]; denegada {
			t.Fatalf("ruta de analisis conserva la denegacion: %s", ruta)
		}
	}

	peticionRectificacion := httptest.NewRequest(
		http.MethodPost,
		httpinterno.RutaRectificacionAnalisisRRHH,
		nil,
	)
	if !esRutaContratacionTemporalDesarrollo(peticionRectificacion) {
		t.Fatal("rectificacion no queda protegida por el perimetro mTLS")
	}
	if _, denegada := rutasCapacidadNoCompuestaContratacionTemporal[httpinterno.RutaRectificacionAnalisisRRHH]; denegada {
		t.Fatal("rectificacion de analisis conserva la barrera no compuesta")
	}
}

func TestConsultasContratacionTemporalDesarrolloNoAceptanAutoridadCliente(
	t *testing.T,
) {
	cfg, rutas := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	servidor, err := NewHTTPServerWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	prueba := httptest.NewUnstartedServer(servidor.Handler)
	prueba.TLS = servidor.TLSConfig.Clone()
	prueba.StartTLS()
	t.Cleanup(prueba.Close)
	cliente := nuevoClienteMTLSContratacionTemporalDesarrollo(t, rutas)
	anadirCadenaCompletaClienteMTLSContratacionTemporalDesarrollo(t, cliente, rutas)
	peticion, err := http.NewRequest(
		http.MethodPost,
		prueba.URL+httpinterno.RutaConsultaCuadroRRHH,
		strings.NewReader(
			"{\"filtros\":{\"texto\":\"\",\"estado_clave\":\"\",\"fase_clave\":\"\"},"+
				"\"paginacion\":{\"limite\":50,\"cursor\":\"\"}}",
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	peticion.Header.Set("X-Vec-Principal", "administrador")
	respuesta, err := cliente.Do(peticion)
	if err != nil {
		t.Fatal(err)
	}
	contenido, _ := io.ReadAll(respuesta.Body)
	respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusBadRequest ||
		bytes.Contains(contenido, []byte(expedienteContratacionTemporalDesarrolloRef)) {
		t.Fatalf("cabecera cliente autorizo la consulta: %d %s", respuesta.StatusCode, contenido)
	}
}

func nuevoClienteMTLSContratacionTemporalDesarrollo(
	t *testing.T,
	rutas config.DevelopmentMaterialPaths,
) *http.Client {
	t.Helper()
	certificado, err := tls.LoadX509KeyPair(rutas.ClientCertificate, rutas.ClientPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	caPEM, err := os.ReadFile(rutas.CACertificate)
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(caPEM) {
		t.Fatal("CA local no cargada")
	}
	return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		Certificates: []tls.Certificate{certificado},
		RootCAs:      raices,
		ServerName:   "localhost",
		MinVersion:   tls.VersionTLS13,
	}}}
}
