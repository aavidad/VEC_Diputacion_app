package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	dietashttp "vec-diputacion-granada/internal/modules/dietas/adapters/httpinterno"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

type registradorFronteraComisionPrueba struct {
	orden    dietasports.OrdenAuditoriaFronteraComision
	llamadas int
	err      error
	ctxErr   error
}

func (r *registradorFronteraComisionPrueba) RegistrarAuditoriaFronteraComision(ctx context.Context, orden dietasports.OrdenAuditoriaFronteraComision) error {
	r.orden, r.llamadas = orden, r.llamadas+1
	r.ctxErr = ctx.Err()
	return r.err
}

type autoridadExactaDelegadaPrueba struct{ llamadas int }

func (a *autoridadExactaDelegadaPrueba) AutorizarRutaExacta(context.Context, string) error {
	a.llamadas++
	return nil
}

func TestComisionesDietasSoloSeMontanConSelectorYMaterialNominal(t *testing.T) {
	if a, err := nuevasComisionesDietasDesarrollo(config.Config{DietasBorradoresEnabled: "false"}, nil, nil, materialDietasDesdeCTDesarrollo{}); err != nil || a != nil {
		t.Fatalf("Dietas inerte no debe abrir rutas: %v %v", a, err)
	}
	if a, err := nuevasComisionesDietasDesarrollo(config.Config{DietasBorradoresEnabled: "true"}, nil, nil, materialDietasDesdeCTDesarrollo{}); err == nil || a != nil {
		t.Fatalf("selector sin gobierno abrió ruta: %v %v", a, err)
	}
}

func TestFronteraComisionesDietasNoDelegaNiSirveSinSesion(t *testing.T) {
	for _, caso := range []struct {
		ruta, metodo string
		admitido     bool
	}{
		{dietashttp.RutaBorradores, http.MethodGet, true},
		{dietashttp.RutaBorradores, http.MethodPost, true},
		{dietashttp.RutaBorradores, http.MethodPut, false},
		{dietashttp.RutaBorradores + "/dco_aaaaaaaaaaaaaaaaaaaaaa", http.MethodGet, true},
		{dietashttp.RutaBorradores + "/dco_aaaaaaaaaaaaaaaaaaaaaa", http.MethodPost, false},
		{dietashttp.RutaBorradores + "/a/b", http.MethodGet, false},
		{dietashttp.RutaBorradores, http.MethodDelete, false},
	} {
		if obtenido := metodoComisionesDietasValido(caso.ruta, caso.metodo); obtenido != caso.admitido {
			t.Fatalf("método %s %s admitido=%t", caso.metodo, caso.ruta, obtenido)
		}
	}
	delegada := &autoridadExactaDelegadaPrueba{}
	a := autoridadExactasConDietas{delegada: delegada, dietas: &autoridadComisionesDietasDesarrollo{}}
	for _, ruta := range []string{dietashttp.RutaBorradores, dietashttp.RutaBorradores + "/dco_aaaaaaaaaaaaaaaaaaaaaa"} {
		if err := a.AutorizarRutaExacta(context.Background(), ruta); err != vechttp.ErrAutenticacionRutaExactaRequerida {
			t.Fatalf("ruta %q: se esperaba autenticación, recibida %v", ruta, err)
		}
	}
	if delegada.llamadas != 0 {
		t.Fatal("Dietas pasó por la autoridad ajena")
	}
	if err := a.AutorizarRutaExacta(context.Background(), "/api/vec/contratacion/alta"); err != nil || delegada.llamadas != 1 {
		t.Fatalf("ruta ajena no delegada: %v %d", err, delegada.llamadas)
	}
	llamadas := 0
	registrador := &registradorFronteraComisionPrueba{}
	protegida := (&autoridadComisionesDietasDesarrollo{registrador: registrador}).proteger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { llamadas++; w.WriteHeader(http.StatusNoContent) }))
	respuesta := httptest.NewRecorder()
	protegida.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, dietashttp.RutaBorradores, nil))
	if respuesta.Code != http.StatusUnauthorized || llamadas != 0 || respuesta.Header().Get("Cache-Control") != "no-store" || registrador.llamadas != 1 || registrador.orden.Motivo != dietasports.MotivoFronteraAutenticacion || registrador.orden.Ruta != dietasports.RutaAuditoriaFronteraComision || registrador.orden.Accion != dietasports.AccionFronteraListar {
		t.Fatalf("sin certificado: estado=%d llamadas=%d", respuesta.Code, llamadas)
	}
	registrador.err = errors.New("almacén caído")
	respuesta = httptest.NewRecorder()
	protegida.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, dietashttp.RutaBorradores+"/dco_sintetico", nil))
	if respuesta.Code != http.StatusServiceUnavailable || registrador.llamadas != 2 || registrador.orden.Ruta != dietasports.RutaAuditoriaFronteraDetalle || registrador.orden.Accion != dietasports.AccionFronteraMetodoNoAdmitido || registrador.orden.ActorRef != "" {
		t.Fatalf("auditoría caída o detalle filtrado: estado=%d llamadas=%d orden=%+v", respuesta.Code, registrador.llamadas, registrador.orden)
	}
	registrador.err = nil
	respuesta = httptest.NewRecorder()
	(&autoridadComisionesDietasDesarrollo{registrador: registrador}).denegar(respuesta, httptest.NewRequest(http.MethodPost, dietashttp.RutaBorradores, nil), http.StatusForbidden, "")
	if respuesta.Code != http.StatusForbidden || registrador.llamadas != 3 || registrador.orden.Motivo != dietasports.MotivoFronteraAccesoDenegado || registrador.orden.Accion != dietasports.AccionFronteraCrear || registrador.orden.ActorRef != "" {
		t.Fatalf("403 filtró sujeto o perdió acción: estado=%d orden=%+v", respuesta.Code, registrador.orden)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	respuesta = httptest.NewRecorder()
	(&autoridadComisionesDietasDesarrollo{registrador: registrador}).denegar(respuesta, httptest.NewRequest(http.MethodDelete, dietashttp.RutaBorradores, nil).WithContext(ctx), http.StatusForbidden, "")
	if respuesta.Code != http.StatusForbidden || registrador.ctxErr != nil || registrador.orden.Accion != dietasports.AccionFronteraMetodoNoAdmitido {
		t.Fatalf("cancelación o método denegado: estado=%d orden=%+v ctx=%v", respuesta.Code, registrador.orden, registrador.ctxErr)
	}
	for _, solicitud := range []*http.Request{
		httptest.NewRequest(http.MethodGet, dietashttp.RutaBorradores+"/a/b", nil),
		httptest.NewRequest(http.MethodDelete, dietashttp.RutaBorradores, nil),
	} {
		respuesta = httptest.NewRecorder()
		protegida.ServeHTTP(respuesta, solicitud)
		if respuesta.Code != http.StatusUnauthorized || llamadas != 0 || registrador.orden.Accion != dietasports.AccionFronteraMetodoNoAdmitido {
			t.Fatalf("familia o método eludió protección: estado=%d orden=%+v llamadas=%d", respuesta.Code, registrador.orden, llamadas)
		}
	}
}

func TestAutoridadComisionesDietasAuditaSesionInvalidaSinDelegar(t *testing.T) {
	registrador := &registradorFronteraComisionPrueba{}
	a := &autoridadComisionesDietasDesarrollo{registrador: registrador}
	autoridad := autoridadExactasConDietas{delegada: &autoridadExactaDelegadaPrueba{}, dietas: a}
	ctx := context.WithValue(context.Background(), claveContextoComisionesDietas{}, contextoComisionesDietas{autoridad: a, ruta: dietashttp.RutaBorradores, metodo: http.MethodGet})
	if err := autoridad.AutorizarRutaExacta(ctx, dietashttp.RutaBorradores); err != vechttp.ErrAccesoRutaExactaDenegado || registrador.llamadas != 1 || registrador.orden.Accion != dietasports.AccionFronteraListar {
		t.Fatalf("sesión inválida sin registro Dietas: err=%v orden=%+v", err, registrador.orden)
	}
	registrador.err = errors.New("auditoría caída")
	if err := autoridad.AutorizarRutaExacta(ctx, dietashttp.RutaBorradores); !errors.Is(err, ErrComposicionBorradoresDietasNoDisponible) || registrador.llamadas != 2 {
		t.Fatalf("fallo auditoría no cerró ruta: %v", err)
	}
}

func TestComisionesDietasAuditaActorSoloTrasSesionYContextoVerificados(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	identidad := composicion.identidad.(*resolvedorIdentidadDesarrollo)
	clienteCert, err := tls.LoadX509KeyPair(rutas.ClientCertificate, rutas.ClientPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(clienteCert.Certificate[0])
	huella := hex.EncodeToString(digest[:])
	principal := identidad.porHuella[digest]
	fixture := nuevoEscenarioMaterialRutasDietasPrueba(t, dietasports.AccionConsultarCatalogoRutasDietas, time.Now().UTC().Truncate(time.Microsecond))
	actorRef := fixture.resultado.Contexto.Principal.ID
	if !strings.HasPrefix(principal.ID, "desarrollo:") || !strings.HasPrefix(actorRef, "per_") || principal.ID == actorRef {
		t.Fatalf("el sujeto TLS y la persona V2 deben conservar sus espacios distintos: sujeto=%q actor=%q", principal.ID, actorRef)
	}
	reloj := &relojSesionConsultaPrueba{ahora: fixture.ahora}
	cuenta := cuentaRutasDietasDesarrollo{CertificadoSHA256: huella, Sujeto: principal.ID, CuentaRef: fixture.resultado.Contexto.Instantanea.CuentaRef, PerfilRef: fixture.resultado.Contexto.PerfilActivoRef}
	registro := &registroSesionConsultaPrueba{reloj: reloj, cuenta: cuenta.CuentaRef}
	revalidador := &revalidadorSesionConsultaPrueba{registro: registro}
	resolutor := &resolutorSesionConsultaPrueba{base: fixture.resultado, reloj: reloj}
	base := &autoridadRutasDietasDesarrollo{resolvedor: identidad, registro: registro, revalidador: revalidador, contextos: resolutor, reloj: reloj, instancia: strings.Repeat("a", 64)}
	auditoria := &registradorFronteraComisionPrueba{}
	autoridad := &autoridadComisionesDietasDesarrollo{base: base, reloj: reloj, cuentas: map[string]cuentaRutasDietasDesarrollo{huella: cuenta}, registrador: auditoria}
	servidos := 0
	servidor := httptest.NewUnstartedServer(autoridad.proteger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		servidos++
		w.WriteHeader(http.StatusNoContent)
	})))
	servidor.TLS = composicion.tls.Clone()
	servidor.StartTLS()
	t.Cleanup(servidor.Close)
	ca, err := os.ReadFile(rutas.CACertificate)
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(ca) {
		t.Fatal("CA de prueba inválida")
	}
	transporte := &http.Transport{TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{clienteCert}, RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13}}
	t.Cleanup(transporte.CloseIdleConnections)
	cliente := &http.Client{Transport: transporte}
	for _, caso := range []struct{ metodo, ruta string }{
		{http.MethodDelete, dietashttp.RutaBorradores},
		{http.MethodGet, dietashttp.RutaBorradores + "/ruta-no-admitida"},
	} {
		solicitud, err := http.NewRequest(caso.metodo, servidor.URL+caso.ruta, nil)
		if err != nil {
			t.Fatal(err)
		}
		respuesta, err := cliente.Do(solicitud)
		if err != nil {
			t.Fatal(err)
		}
		respuesta.Body.Close()
		if respuesta.StatusCode != http.StatusForbidden || auditoria.orden.ActorRef != actorRef || auditoria.orden.Motivo != dietasports.MotivoFronteraAccesoDenegado || servidos != 0 {
			t.Fatalf("rechazo de %s %s: estado=%d actor=%q servidos=%d", caso.metodo, caso.ruta, respuesta.StatusCode, auditoria.orden.ActorRef, servidos)
		}
	}
	if len(registro.altas) != 2 || revalidador.llamadas != 2 {
		t.Fatalf("identidad central no verificada antes del rechazo: sesiones=%d revalidaciones=%d", len(registro.altas), revalidador.llamadas)
	}
	anónimo := httptest.NewRecorder()
	autoridad.proteger(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { servidos++ })).ServeHTTP(anónimo, httptest.NewRequest(http.MethodGet, dietashttp.RutaBorradores, nil))
	if anónimo.Code != http.StatusUnauthorized || auditoria.orden.ActorRef != "" || servidos != 0 {
		t.Fatalf("anónimo atribuido o servido: estado=%d actor=%q servidos=%d", anónimo.Code, auditoria.orden.ActorRef, servidos)
	}
	cuenta.Sujeto = "desarrollo:otro-sujeto"
	autoridad.cuentas[huella] = cuenta
	solicitud, err := http.NewRequest(http.MethodDelete, servidor.URL+dietashttp.RutaBorradores, nil)
	if err != nil {
		t.Fatal(err)
	}
	respuesta, err := cliente.Do(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusForbidden || auditoria.orden.ActorRef != "" || servidos != 0 || len(registro.altas) != 2 {
		t.Fatalf("sujeto TLS y cuenta cruzados: estado=%d actor=%q servidos=%d sesiones=%d", respuesta.StatusCode, auditoria.orden.ActorRef, servidos, len(registro.altas))
	}
}

func TestAutoridadComisionesDietasNoAtribuyeContextoAjeno(t *testing.T) {
	fixture := nuevoEscenarioMaterialRutasDietasPrueba(t, dietasports.AccionConsultarCatalogoRutasDietas)
	datos, err := fixture.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	reloj := &relojSesionConsultaPrueba{ahora: fixture.ahora}
	auditoria := &registradorFronteraComisionPrueba{}
	autoridad := &autoridadComisionesDietasDesarrollo{reloj: reloj, registrador: auditoria}
	ajena := &autoridadComisionesDietasDesarrollo{reloj: reloj}
	seguridad := contextoSeguridadComunDesarrollo{Vinculo: datos.VinculoAutenticacionActor, Resultado: fixture.resultado}
	for _, caso := range []struct {
		nombre    string
		origen    *autoridadComisionesDietasDesarrollo
		seguridad contextoSeguridadComunDesarrollo
		esperado  string
	}{
		{"autoridad propia", autoridad, seguridad, fixture.resultado.Contexto.Principal.ID},
		{"autoridad ajena", ajena, seguridad, ""},
		{"vínculo inválido", autoridad, contextoSeguridadComunDesarrollo{Resultado: fixture.resultado}, ""},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), claveContextoComisionesDietas{}, contextoComisionesDietas{autoridad: caso.origen, ruta: dietashttp.RutaBorradores, metodo: http.MethodDelete, seguridad: caso.seguridad})
			acceso := autoridadExactasConDietas{delegada: &autoridadExactaDelegadaPrueba{}, dietas: autoridad}
			if err := acceso.AutorizarRutaExacta(ctx, dietashttp.RutaBorradores); err != vechttp.ErrAccesoRutaExactaDenegado || auditoria.orden.ActorRef != caso.esperado {
				t.Fatalf("denegación=%v actor=%q esperado=%q", err, auditoria.orden.ActorRef, caso.esperado)
			}
		})
	}
	if auditoria.llamadas != 3 {
		t.Fatalf("denegaciones auditadas=%d", auditoria.llamadas)
	}
}
