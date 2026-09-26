package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	docpg "vec-diputacion-granada/internal/vec/documentos/adapters/postgres"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Sin cadena verificada, o con una cadena vacía, la frontera no indexa el
// certificado ni escribe en la bitácora: 401 en la ruta publicada y 404 fuera.
func TestFronteraDocumentosSinCadenaVerificadaNoIndexaNiAudita(t *testing.T) {
	registrador := &registradorDocumentosPrueba{}
	a := &autoridadDocumentosDesarrollo{base: &autoridadRutasDietasDesarrollo{}, reloj: relojRutasDietas{}, registrador: registrador,
		publicadas: map[string]bool{docpg.RutaFronteraConsulta: true}}
	servidos := 0
	h := a.proteger(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { servidos++ }))
	for _, estado := range []*tls.ConnectionState{
		{HandshakeComplete: true, Version: tls.VersionTLS13},
		{HandshakeComplete: true, Version: tls.VersionTLS13, VerifiedChains: [][]*x509.Certificate{{}}},
		{HandshakeComplete: true, Version: tls.VersionTLS13, VerifiedChains: [][]*x509.Certificate{{nil}}},
	} {
		for ruta, esperado := range map[string]int{docpg.RutaFronteraConsulta: http.StatusUnauthorized, "/api/vec/documentos/otra": http.StatusNotFound} {
			r := peticionDocumentos(ruta, false)
			r.TLS = estado
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != esperado || len(registrador.ordenes) != 0 || servidos != 0 {
				t.Fatalf("%s sin cadena: estado=%d esperado=%d auditadas=%d servidos=%d", ruta, w.Code, esperado, len(registrador.ordenes), servidos)
			}
		}
	}
}

// Recorrido con mTLS real y sesión verificada: sólo la petición limpia llega
// al manejador; cabeceras libres y cuentas cruzadas se deniegan sin actor; una
// ruta no publicada se deniega con el actor verificado; el contexto queda
// ligado a la ruta exacta y el material V3 debe estar ligado a la solicitud.
func TestFronteraDocumentosConSesionVerificada(t *testing.T) {
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
	reloj := &relojSesionConsultaPrueba{ahora: fixture.ahora}
	cuenta := cuentaRutasDietasDesarrollo{CertificadoSHA256: huella, Sujeto: principal.ID, CuentaRef: fixture.resultado.Contexto.Instantanea.CuentaRef, PerfilRef: fixture.resultado.Contexto.PerfilActivoRef}
	registro := &registroSesionConsultaPrueba{reloj: reloj, cuenta: cuenta.CuentaRef}
	base := &autoridadRutasDietasDesarrollo{resolvedor: identidad, registro: registro, revalidador: &revalidadorSesionConsultaPrueba{registro: registro},
		contextos: &resolutorSesionConsultaPrueba{base: fixture.resultado, reloj: reloj}, reloj: reloj, instancia: strings.Repeat("a", 64)}
	registrador := &registradorDocumentosPrueba{}
	a := &autoridadDocumentosDesarrollo{base: base, reloj: reloj, cuentas: map[string]cuentaRutasDietasDesarrollo{huella: cuenta}, registrador: registrador,
		publicadas: map[string]bool{docpg.RutaFronteraConsulta: true}}
	servidos := 0
	var capturado context.Context
	servidor := httptest.NewUnstartedServer(a.proteger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		servidos++
		capturado = context.WithoutCancel(r.Context())
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
	enviar := func(ruta string, cabeceras map[string]string) int {
		t.Helper()
		solicitud, err := http.NewRequest(http.MethodPost, servidor.URL+ruta, strings.NewReader(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range cabeceras {
			solicitud.Header.Set(k, v)
		}
		respuesta, err := cliente.Do(solicitud)
		if err != nil {
			t.Fatal(err)
		}
		respuesta.Body.Close()
		return respuesta.StatusCode
	}

	if estado := enviar(docpg.RutaFronteraConsulta, nil); estado != http.StatusNoContent || servidos != 1 || len(registrador.ordenes) != 0 || capturado == nil {
		t.Fatalf("petición limpia: estado=%d servidos=%d auditadas=%d", estado, servidos, len(registrador.ordenes))
	}
	// curl/OpenSSL completa la cadena del cliente y envía también la CA: la
	// frontera la reduce a la hoja (como Cronos, Personal, Dietas y CT) y la
	// petición llega al manejador sin denegación.
	bloqueCA, _ := pem.Decode(ca)
	if bloqueCA == nil {
		t.Fatal("CA de prueba sin PEM")
	}
	conCadena := clienteCert
	conCadena.Certificate = append([][]byte{clienteCert.Certificate[0]}, bloqueCA.Bytes)
	transporteCadena := &http.Transport{TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{conCadena}, RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13}}
	t.Cleanup(transporteCadena.CloseIdleConnections)
	solicitudCadena, err := http.NewRequest(http.MethodPost, servidor.URL+docpg.RutaFronteraConsulta, strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	respuestaCadena, err := (&http.Client{Transport: transporteCadena}).Do(solicitudCadena)
	if err != nil {
		t.Fatal(err)
	}
	respuestaCadena.Body.Close()
	if respuestaCadena.StatusCode != http.StatusNoContent || servidos != 2 || len(registrador.ordenes) != 0 {
		t.Fatalf("cliente con cadena completa: estado=%d servidos=%d auditadas=%d", respuestaCadena.StatusCode, servidos, len(registrador.ordenes))
	}
	for _, cabecera := range []string{"Cookie", "Authorization"} {
		antes := len(registrador.ordenes)
		estado := enviar(docpg.RutaFronteraConsulta, map[string]string{cabecera: "x"})
		if estado != http.StatusUnauthorized || servidos != 2 || len(registrador.ordenes) != antes+1 {
			t.Fatalf("%s: estado=%d servidos=%d", cabecera, estado, servidos)
		}
		if o := registrador.ordenes[antes]; o.Motivo != docpg.MotivoFronteraAutenticacion || o.ActorRef != "" || o.Ruta != docpg.RutaFronteraConsulta {
			t.Fatalf("%s: orden %+v", cabecera, o)
		}
	}
	if estado := enviar("/api/vec/documentos/otra", nil); estado != http.StatusNotFound || servidos != 2 {
		t.Fatalf("ruta no publicada: estado=%d servidos=%d", estado, servidos)
	}
	if o := registrador.ordenes[len(registrador.ordenes)-1]; o.Motivo != docpg.MotivoFronteraDenegado || o.ActorRef != actorRef || o.Ruta != docpg.RutaFronteraOtra {
		t.Fatalf("ruta no publicada sin actor verificado: %+v", o)
	}

	// El contexto de la petición sólo autoriza su ruta exacta.
	cadena := autoridadExactasConDocumentos{documentos: a}
	if err := cadena.AutorizarRutaExacta(capturado, docpg.RutaFronteraConsulta); err != nil {
		t.Fatalf("ruta propia denegada: %v", err)
	}
	for _, otra := range []string{"/api/vec/documentos/originales/descargas", "/api/vec/documentos/otra"} {
		if err := cadena.AutorizarRutaExacta(capturado, otra); !errors.Is(err, vechttp.ErrAccesoRutaExactaDenegado) {
			t.Fatalf("contexto de %s reutilizado en %s: %v", docpg.RutaFronteraConsulta, otra, err)
		}
	}

	// Una decisión válida con material no ligado a la solicitud se deniega.
	emisor := &emisorDocumentosPrueba{t: t, ahora: fixture.ahora}
	consulta := autoridadConsultaDocumentos{autoridad: a, emisor: emisor, motivo: fixture.motivo}
	expediente := "ref:" + strings.Repeat("2", 64)
	if _, err := consulta.ResolverConsultaExpediente(capturado, docports.ConsultaExpediente{ExpedienteRef: expediente, Limite: 5}); !errors.Is(err, docports.ErrAccesoDenegado) || emisor.llamadas != 1 {
		t.Fatalf("material no ligado aceptado: %v (emisiones=%d)", err, emisor.llamadas)
	}

	// Sujeto TLS y cuenta cruzados: 403 sin actor y sin llegar al manejador.
	cuenta.Sujeto = "desarrollo:otro-sujeto"
	a.cuentas[huella] = cuenta
	antes := len(registrador.ordenes)
	if estado := enviar(docpg.RutaFronteraConsulta, nil); estado != http.StatusForbidden || servidos != 2 || len(registrador.ordenes) != antes+1 {
		t.Fatalf("cuenta cruzada: estado=%d servidos=%d", estado, servidos)
	}
	if o := registrador.ordenes[antes]; o.Motivo != docpg.MotivoFronteraDenegado || o.ActorRef != "" {
		t.Fatalf("cuenta cruzada: orden %+v", o)
	}
}

// emisorDocumentosPrueba concede una decisión V3 real para la solicitud
// recibida, pero exporta un material vacío que no está ligado a ella.
type emisorDocumentosPrueba struct {
	t        *testing.T
	ahora    time.Time
	llamadas int
}

func (e *emisorDocumentosPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, s core.SolicitudAutorizacionLigadaV3, _ core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	datos, err := s.Datos()
	if err != nil {
		e.t.Fatal(err)
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		e.t.Fatal(err)
	}
	version := core.VersionRol{
		RolID: "consulta_documentos", Version: 1, Nombre: "Consulta documentos", Estado: core.EstadoVersionRolPublicada,
		Concesiones: []core.ConcesionRol{{Accion: datos.Accion, ModuloID: datos.Recurso.ModuloID, TipoRecurso: datos.Recurso.Tipo,
			Finalidades: []string{datos.Finalidad}, GarantiaMinima: core.AuthAssuranceSubstantial}},
		PublicadaPor: "responsable-seguridad", PublicadaEn: e.ahora.Add(-24 * time.Hour),
	}
	huellaCatalogo, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		e.t.Fatal(err)
	}
	instantanea := core.InstantaneaAutorizacion{
		AsignacionPerfil: core.AsignacionPerfil{AsignacionID: "asig-documentos", Version: 1, PerfilActivoRef: vinculo.PerfilActivoRef,
			PrincipalID: vinculo.PrincipalID, VersionRolRef: version.Referencia(), Estado: core.EstadoAsignacionPerfilActiva,
			Ambitos:      []core.AmbitoPerfil{{Clave: "ambito_ref", Valores: []string{"granada"}}},
			VigenteDesde: e.ahora.Add(-time.Hour), VigenteHasta: e.ahora.Add(time.Hour), EmitidaPor: "administrador-identidades", EmitidaEn: e.ahora.Add(-2 * time.Hour)},
		VersionRol: version,
		ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{VersionRolRef: version.Referencia(), Revision: 1,
			Estado: core.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	if err := instantanea.Validar(); err != nil {
		e.t.Fatalf("instantánea: %v", err)
	}
	evidencia, err := core.NuevaEvidenciaEvaluacionAutorizacionV3(s, instantanea, "dec_0123456789abcdef0123456789abcdef", e.ahora, e.ahora.Add(90*time.Second))
	if err != nil {
		e.t.Fatal(err)
	}
	decision, err := core.NuevaDecisionAutorizacionLigadaV3(s, evidencia)
	if err != nil || decision.ValidarPara(s) != nil {
		e.t.Fatalf("decisión de prueba: %v", err)
	}
	return decision, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, exportadorNoLigadoPrueba{}, nil
}

type exportadorNoLigadoPrueba struct{}

func (exportadorNoLigadoPrueba) String() string { return "exportador-no-ligado" }
func (exportadorNoLigadoPrueba) LogValue() slog.Value {
	return slog.StringValue("exportador-no-ligado")
}
func (exportadorNoLigadoPrueba) ExportarMaterialParaConsumidor() (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}
