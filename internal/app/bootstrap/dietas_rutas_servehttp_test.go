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

	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	dp "vec-diputacion-granada/internal/modules/dietas/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/memory"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type servicioRutasServeHTTPPrueba struct {
	llamadas  int
	solicitud dp.SolicitudAccesoRutasDietas
	err       error
}

func (s *servicioRutasServeHTTPPrueba) AutorizarYConsumirAccesoRutas(_ context.Context, q dp.SolicitudAccesoRutasDietas) (dp.ReciboAccesoRutasDietas, error) {
	s.llamadas++
	s.solicitud = q
	return dp.ReciboAccesoRutasDietas{}, s.err
}

// TLS y resolvedor son reales; los puertos centrales son dobles declarados.
// Esta prueba acredita composición de frontera, no PostgreSQL ni E2E publicado.
func TestDietasRutasServeHTTPConMTLSYSesionNominal(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	composicion, e := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if e != nil {
		t.Fatal(e)
	}
	identidad := composicion.identidad.(*resolvedorIdentidadDesarrollo)
	clienteCert, e := tls.LoadX509KeyPair(rutas.ClientCertificate, rutas.ClientPrivateKey)
	if e != nil {
		t.Fatal(e)
	}
	digest := sha256.Sum256(clienteCert.Certificate[0])
	huella := hex.EncodeToString(digest[:])
	principal, ok := identidad.porHuella[digest]
	if !ok {
		t.Fatal("certificado no registrado")
	}
	fixture := nuevoEscenarioMaterialRutasDietasPrueba(t, dp.AccionConsultarCatalogoRutasDietas, time.Now().UTC().Truncate(time.Microsecond))
	reloj := &relojSesionConsultaPrueba{ahora: fixture.ahora}
	cuenta := cuentaRutasDietasDesarrollo{CertificadoSHA256: huella, Sujeto: principal.ID, CuentaRef: fixture.resultado.Contexto.Instantanea.CuentaRef, PerfilRef: fixture.resultado.Contexto.PerfilActivoRef}
	registro := &registroSesionConsultaPrueba{reloj: reloj, cuenta: cuenta.CuentaRef}
	revalidador := &revalidadorSesionConsultaPrueba{registro: registro}
	resolutor := &resolutorSesionConsultaPrueba{base: fixture.resultado, reloj: reloj}
	servicio := &servicioRutasServeHTTPPrueba{}
	autoridad := &autoridadRutasDietasDesarrollo{resolvedor: identidad, cuentas: map[string]cuentaRutasDietasDesarrollo{huella: cuenta}, registro: registro, revalidador: revalidador, contextos: resolutor, reloj: reloj, servicio: servicio, instancia: strings.Repeat("a", 64), ambito: "granada", grafo: "grafo-v1", huellaGrafo: strings.Repeat("b", 64), huellaCatalogo: strings.Repeat("c", 64), motivoCatalogo: fixture.motivo, motivoCalculo: fixture.motivo}
	store := memory.NewStore()
	shell, e := vecapp.NewService(store, store, store)
	if e != nil {
		t.Fatal(e)
	}
	datos := 0
	destino := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { datos++; w.WriteHeader(200) })
	handler, e := httpapi.NewHandlerWithOptions(shell, httpapi.HandlerOptions{AutoridadRutasDietas: autoridad, ManejadorCatalogoRutaDietas: destino, ManejadorRutaDietas: destino})
	if e != nil {
		t.Fatal(e)
	}
	servidor := httptest.NewUnstartedServer(handler)
	servidor.TLS = composicion.tls.Clone()
	servidor.StartTLS()
	t.Cleanup(servidor.Close)
	ca, e := os.ReadFile(rutas.CACertificate)
	if e != nil {
		t.Fatal(e)
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(ca) {
		t.Fatal("CA")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{clienteCert}, RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13}}
	t.Cleanup(transport.CloseIdleConnections)
	cliente := &http.Client{Transport: transport}
	pedir := func(metodo, ruta string, estado int) {
		t.Helper()
		req, _ := http.NewRequest(metodo, servidor.URL+ruta, nil)
		res, err := cliente.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != estado {
			t.Fatalf("%s %s estado%d esperado%d", metodo, ruta, res.StatusCode, estado)
		}
	}
	pedir("GET", "/api/vec/dietas/route-catalog", 200)
	if datos != 1 || servicio.llamadas != 1 || len(registro.altas) != 1 || revalidador.llamadas != 1 {
		t.Fatal("frontera o sesión omitidas")
	}
	if servicio.solicitud.Recurso.Atributos["ruta"] != "/api/vec/dietas/route-catalog" || servicio.solicitud.Vinculo.ValidarPara(servicio.solicitud.ResultadoContexto) != nil {
		t.Fatal("vinculo de petición incoherente")
	}
	pedir("POST", "/api/vec/dietas/road-route", 200)
	if servicio.solicitud.Accion != dp.AccionSolicitarCalculoRutasDietas || servicio.solicitud.Recurso.Tipo != dp.TipoCalculoRutasDietas || len(registro.altas) != 2 || registro.altas[0].SesionID == registro.altas[1].SesionID {
		t.Fatal("sesión/cálculo no nominal")
	}
	pedir("POST", "/api/vec/dietas/route-catalog", 405)
	servicio.err = dp.ErrAccesoRutasDietasDenegado
	pedir("GET", "/api/vec/dietas/route-catalog", 403)
	servicio.err = dp.ErrAccesoRutasDietasNoDisponible
	pedir("GET", "/api/vec/dietas/route-catalog", 503)
	if datos != 2 {
		t.Fatal("datos tras denegación")
	}
	delete(autoridad.cuentas, huella)
	pedir("GET", "/api/vec/dietas/route-catalog", 403)
	autoridad.cuentas[huella] = cuenta
	revalidador.alterar = func(a *core.AutenticacionRevalidadaV1) { a.CuentaPrivilegiada = true }
	pedir("GET", "/api/vec/dietas/route-catalog", 403)
	if datos != 2 {
		t.Fatal("cruce cuenta permitió datos")
	}
	sinTLS := httptest.NewRecorder()
	handler.ServeHTTP(sinTLS, httptest.NewRequest("GET", "/api/vec/dietas/route-catalog", nil))
	if sinTLS.Code != 401 {
		t.Fatal("sin mTLS no denegado")
	}
}

func TestDietasRutasAusenteFallaCerradoSinAfectarComposicion(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	a, cerrar, e := nuevasRutasDietasDesarrollo(cfg, nil, nil)
	if e != nil || a != nil || cerrar == nil {
		t.Fatal("ausencia no fue explícita")
	}
	cerrar()
	var autoridad *autoridadRutasDietasDesarrollo
	if !errors.Is(autoridad.AutorizarPeticionRutaDietas(context.Background(), nil), httpapi.ErrRutaDietasNoDisponible) {
		t.Fatal("nil abierto")
	}
}

type autorizadorRutasAplicacionPrueba struct {
	e escenarioMaterialRutasDietasPrueba
}

func (a autorizadorRutasAplicacionPrueba) ExigirSolicitudLigadaV3(_ context.Context, q core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	if a.e.decision.ValidarPara(q) != nil || r.HuellaSHA256 != a.e.resultado.HuellaSHA256 {
		return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, dp.ErrAccesoRutasDietasDenegado
	}
	return a.e.decision, a.e.confirmacion, nil
}

type consumidorRutasAplicacionPrueba struct {
	llamadas int
	cruzar   bool
}

func (c *consumidorRutasAplicacionPrueba) ConsumirAccesoRutasDietas(_ context.Context, o dp.OrdenConsumoAccesoRutasDietas) (dp.ReciboAccesoRutasDietas, error) {
	c.llamadas++
	z := o.Material.ResumenCapacidad()
	r := dp.ReciboAccesoRutasDietas{DecisionRef: z.DecisionRef(), EfectoRef: z.EfectoRef(), HuellaEfectoSHA256: z.EfectoHuellaSHA256(), ConsumoHuellaSHA256: strings.Repeat("c", 64), AuditoriaRef: "auditoria:sintetica", ConsumidaEn: z.EmitidaEn(), ConsumoNuevo: true}
	if c.cruzar {
		r.DecisionRef = "decision:ajena"
	}
	return r, nil
}
func TestDietasRutasAplicacionExigeMaterialYReciboExactos(t *testing.T) {
	for _, accion := range []string{dp.AccionConsultarCatalogoRutasDietas, dp.AccionSolicitarCalculoRutasDietas} {
		e := nuevoEscenarioMaterialRutasDietasPrueba(t, accion)
		consumidor := &consumidorRutasAplicacionPrueba{}
		servicio, err := dietasapp.NuevoServicioAutorizacionRutasDietas(autorizadorRutasAplicacionPrueba{e}, e.proveedor, consumidor)
		if err != nil {
			t.Fatal(err)
		}
		datos, err := e.solicitud.Datos()
		if err != nil {
			t.Fatal(err)
		}
		q := dp.SolicitudAccesoRutasDietas{ResultadoContexto: e.resultado, Vinculo: datos.VinculoAutenticacionActor, ReferenciaMotivo: e.motivo, Correlacion: datos.Correlacion, Accion: accion, Recurso: datos.Recurso, Audiencia: dp.AudienciaAccesoRutasDietas, Finalidad: dp.FinalidadConsultarItinerarioDietas}
		if _, err = servicio.AutorizarYConsumirAccesoRutas(context.Background(), q); err != nil || consumidor.llamadas != 1 {
			t.Fatalf("material nominal/recibo: %v", err)
		}
		consumidor.cruzar = true
		if _, err = servicio.AutorizarYConsumirAccesoRutas(context.Background(), q); err == nil {
			t.Fatal("recibo ajeno")
		}
		q.Recurso.Atributos["metodo"] = "DELETE"
		if _, err = servicio.AutorizarYConsumirAccesoRutas(context.Background(), q); err == nil || consumidor.llamadas != 2 {
			t.Fatal("cruce método llegó al consumo")
		}
	}
}
