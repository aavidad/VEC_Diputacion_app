package bootstrap

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/auditoria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type emisorAuditoriaConsultaPrueba struct{ llamadas int }

func (e *emisorAuditoriaConsultaPrueba) EmitirMaterialAutorizacionAtestadaV3(
	context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2,
) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, auditoria.ErrNoDisponible
}

type identidadAuditoriaConsultaPrueba struct {
	resuelta auditoria.IdentidadResuelta
	err      error
}

func (i *identidadAuditoriaConsultaPrueba) ResolverIdentidadConsulta(context.Context, *http.Request, auditoria.FuenteConsulta) (auditoria.IdentidadResuelta, error) {
	return i.resuelta, i.err
}

type opcionesAuditoriaConsultaPrueba struct {
	opciones auditoria.Opciones
	err      error
}

type registradorIntentoConsultaPrueba struct {
	ordenes []vecports.OrdenIntentoAuditoria
}

func (r *registradorIntentoConsultaPrueba) AppendIntentoAuditoria(_ context.Context, orden vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	r.ordenes = append(r.ordenes, orden)
	datos, err := orden.Datos()
	if err != nil {
		return vecports.AcuseIntentoAuditoria{}, err
	}
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_sintetica", Secuencia: int64(len(r.ordenes)),
		HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: datos.Datos.CorrelacionRef,
		RegistradaEn: time.Now().UTC().Truncate(time.Microsecond)}, nil
}

func configuracionIntentosConsultaPrueba(motivo vecdomain.ReferenciaEntradaCatalogo) auditoria.ConfiguracionIntentos {
	return auditoria.ConfiguracionIntentos{Registrador: &registradorIntentoConsultaPrueba{},
		Proceso: "vec_server_ensayo", Canal: string(vecdomain.SuperficieAutenticacionInternaCorporativaV1),
		Finalidad: "revision_administrativa_auditoria_rrhh", Motivo: motivo}
}

func (o *opcionesAuditoriaConsultaPrueba) Actuales(context.Context) (auditoria.Opciones, error) {
	return o.opciones, o.err
}

func TestRutasAuditoriaConsultaRRHHExigenDependenciasYRegistranAmbasRutas(t *testing.T) {
	ct, bolsa := &emisorAuditoriaConsultaPrueba{}, &emisorAuditoriaConsultaPrueba{}
	escenario := nuevoEscenarioMaterialRutasDietasPrueba(t, "dietas.ruta.catalogo.consultar")
	deps := dependenciasAuditoriaConsultaRRHH{
		PoolCT: &pgxpool.Pool{}, PoolBolsa: &pgxpool.Pool{},
		EmisorCT: ct, EmisorBolsa: bolsa,
		Identidad: &identidadAuditoriaConsultaPrueba{}, Opciones: &opcionesAuditoriaConsultaPrueba{},
		Intentos: configuracionIntentosConsultaPrueba(escenario.motivo),
	}
	rutas, err := nuevasRutasAuditoriaConsultaRRHH(deps)
	if err != nil || len(rutas) != 2 || rutas[0].Ruta != auditoria.RutaOpciones ||
		rutas[1].Ruta != auditoria.RutaConsulta || rutas[0].Manejador == nil ||
		rutas[0].Manejador != rutas[1].Manejador {
		t.Fatalf("rutas de auditoría incompletas: rutas=%v error=%v", rutas, err)
	}
	casos := []struct {
		nombre  string
		alterar func(*dependenciasAuditoriaConsultaRRHH)
	}{
		{"pool CT", func(d *dependenciasAuditoriaConsultaRRHH) { d.PoolCT = nil }},
		{"pool Bolsa", func(d *dependenciasAuditoriaConsultaRRHH) { d.PoolBolsa = nil }},
		{"emisor CT", func(d *dependenciasAuditoriaConsultaRRHH) { d.EmisorCT = (*emisorAuditoriaConsultaPrueba)(nil) }},
		{"emisor Bolsa", func(d *dependenciasAuditoriaConsultaRRHH) { d.EmisorBolsa = (*emisorAuditoriaConsultaPrueba)(nil) }},
		{"identidad", func(d *dependenciasAuditoriaConsultaRRHH) { d.Identidad = (*identidadAuditoriaConsultaPrueba)(nil) }},
		{"intentos", func(d *dependenciasAuditoriaConsultaRRHH) {
			d.Intentos.Registrador = (*registradorIntentoConsultaPrueba)(nil)
		}},
		{"opciones", func(d *dependenciasAuditoriaConsultaRRHH) { d.Opciones = (*opcionesAuditoriaConsultaPrueba)(nil) }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			incompletas := deps
			caso.alterar(&incompletas)
			if rutas, err := nuevasRutasAuditoriaConsultaRRHH(incompletas); rutas != nil || !errors.Is(err, auditoria.ErrNoDisponible) {
				t.Fatalf("dependencia ausente activó rutas: %v, %v", rutas, err)
			}
		})
	}
}

func TestEmisorAuditoriaConsultaRRHHDespachaSoloFuenteExacta(t *testing.T) {
	escenario := nuevoEscenarioMaterialRutasDietasPrueba(t, "dietas.ruta.catalogo.consultar")
	base, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ct, bolsa := &emisorAuditoriaConsultaPrueba{}, &emisorAuditoriaConsultaPrueba{}
	emisor := emisorAuditoriaConsultaRRHH{ct: ct, bolsa: bolsa}
	preparar := func(fuente, accion string) vecdomain.SolicitudAutorizacionLigadaV3 {
		t.Helper()
		datos := base
		datos.Accion = accion
		ambitos := map[string]string{"expediente_ref": "expediente:opaco:123"}
		if fuente != "" {
			ambitos["fuente"] = fuente
		}
		datos.Recurso = vecdomain.RecursoAutorizable{
			Referencia: "expediente:opaco:123", ModuloID: auditoria.ModuloAutorizacion,
			Tipo:      auditoria.TipoRecurso,
			Ambitos:   ambitos,
			Atributos: map[string]string{"filtro_sha256": strings.Repeat("a", 64)},
		}
		solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(datos)
		if err != nil {
			t.Fatalf("solicitud de prueba: %v", err)
		}
		return solicitud
	}
	for _, fuente := range []string{"ct", "bolsa"} {
		_, _, _, err := emisor.EmitirMaterialAutorizacionAtestadaV3(context.Background(), preparar(fuente, auditoria.AccionConsultar), escenario.resultado)
		if !errors.Is(err, auditoria.ErrNoDisponible) {
			t.Fatalf("fuente %s no alcanzó su emisor: %v", fuente, err)
		}
	}
	if ct.llamadas != 1 || bolsa.llamadas != 1 {
		t.Fatalf("despacho cruzado: CT=%d Bolsa=%d", ct.llamadas, bolsa.llamadas)
	}
	for _, solicitud := range []vecdomain.SolicitudAutorizacionLigadaV3{
		{}, preparar("", auditoria.AccionConsultar), preparar("otro", auditoria.AccionConsultar),
		preparar("ct", "vec.auditoria.modificar"),
	} {
		_, _, _, err := emisor.EmitirMaterialAutorizacionAtestadaV3(context.Background(), solicitud, escenario.resultado)
		if !errors.Is(err, auditoria.ErrDenegada) {
			t.Fatalf("solicitud no exacta no denegada: %v", err)
		}
	}
	if ct.llamadas != 1 || bolsa.llamadas != 1 {
		t.Fatalf("solicitud inválida alcanzó un emisor: CT=%d Bolsa=%d", ct.llamadas, bolsa.llamadas)
	}
}

func TestRutasAuditoriaConsultaRRHHContratoHTTPFallaCerrado(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	escenario := nuevoEscenarioMaterialRutasDietasPrueba(t, "dietas.ruta.catalogo.consultar", ahora)
	datos, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if err := escenario.resultado.Validar(); err != nil {
		t.Fatalf("contexto de prueba inválido: %v", err)
	}
	if err := datos.VinculoAutenticacionActor.ValidarPara(escenario.resultado); err != nil ||
		!datos.VinculoAutenticacionActor.VigenteEn(time.Now().UTC().Truncate(time.Microsecond), escenario.resultado) {
		t.Fatalf("vínculo de prueba inválido: %v", err)
	}
	ct, bolsa := &emisorAuditoriaConsultaPrueba{}, &emisorAuditoriaConsultaPrueba{}
	opciones := auditoria.Opciones{
		FinalidadRef: "revision_administrativa_auditoria_rrhh",
		MotivoRef:    escenario.motivo.Referencia(), PermisoRequerido: auditoria.AccionConsultar,
		Fuentes: []string{"ct", "bolsa"}, Motivo: escenario.motivo,
	}
	rutas, err := nuevasRutasAuditoriaConsultaRRHH(dependenciasAuditoriaConsultaRRHH{
		PoolCT: &pgxpool.Pool{}, PoolBolsa: &pgxpool.Pool{},
		EmisorCT: ct, EmisorBolsa: bolsa,
		Identidad: &identidadAuditoriaConsultaPrueba{resuelta: auditoria.IdentidadResuelta{
			Vinculo: datos.VinculoAutenticacionActor, Resultado: escenario.resultado,
			Correlacion: datos.Correlacion,
		}},
		Opciones: &opcionesAuditoriaConsultaPrueba{opciones: opciones},
		Intentos: configuracionIntentosConsultaPrueba(escenario.motivo),
	})
	if err != nil {
		t.Fatal(err)
	}
	manejador := rutas[0].Manejador
	probar := func(metodo, ruta, cuerpo string, esperado int) {
		t.Helper()
		peticion := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
		if cuerpo != "" {
			peticion.Header.Set("Content-Type", "application/json")
		}
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		if respuesta.Code != esperado || respuesta.Header().Get("Set-Cookie") != "" {
			t.Fatalf("%s %s: HTTP %d, esperado %d", metodo, ruta, respuesta.Code, esperado)
		}
	}
	probar(http.MethodGet, auditoria.RutaOpciones, "", http.StatusOK)
	if ct.llamadas != 0 || bolsa.llamadas != 0 {
		t.Fatal("GET opciones emitió autorización V3")
	}
	desde := ahora.Add(-time.Hour).Format(time.RFC3339Nano)
	hasta := ahora.Add(time.Hour).Format(time.RFC3339Nano)
	cuerpo := func(fuente, motivo, expediente string) string {
		b, err := json.Marshal(map[string]any{
			"fuente": fuente, "expediente_ref": expediente, "desde": desde, "hasta": hasta,
			"limite": 1, "finalidad_ref": opciones.FinalidadRef, "motivo_ref": motivo,
		})
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	probar(http.MethodPost, auditoria.RutaConsulta, cuerpo("", opciones.MotivoRef, "expediente:opaco:123"), http.StatusBadRequest)
	probar(http.MethodPost, auditoria.RutaConsulta, cuerpo("ct,bolsa", opciones.MotivoRef, "expediente:opaco:123"), http.StatusBadRequest)
	probar(http.MethodPost, auditoria.RutaConsulta, cuerpo("ct", "motivo:ajeno", "expediente:opaco:123"), http.StatusForbidden)
	probar(http.MethodPost, auditoria.RutaConsulta, cuerpo("ct", opciones.MotivoRef, ""), http.StatusForbidden)
	probar(http.MethodGet, auditoria.RutaConsulta, "", http.StatusMethodNotAllowed)
	probar(http.MethodPost, auditoria.RutaOpciones, "", http.StatusMethodNotAllowed)
	probar(http.MethodGet, "/api/vec/auditoria/desconocida", "", http.StatusNotFound)
	if ct.llamadas != 0 || bolsa.llamadas != 0 {
		t.Fatal("petición inválida alcanzó un emisor V3")
	}
}

// Requiere un servidor HTTPS sintético con AD3-91/CT132/B48, dos expedientes
// con al menos dos actuaciones del mismo actor y una referencia ajena por
// fuente. No se conecta por defecto a la base conservada ni simula el éxito.
func TestAuditoriaConsultaRRHHHTTPPostgreSQL18(t *testing.T) {
	base := os.Getenv("VEC_AUDITORIA_E2E_URL")
	if base == "" {
		t.Skip("sin VEC_AUDITORIA_E2E_URL: no hay servidor/PG18 sintético de auditoría configurado")
	}
	config := auditoriaE2EConfiguracion{
		url:   base,
		desde: os.Getenv("VEC_AUDITORIA_E2E_DESDE"), hasta: os.Getenv("VEC_AUDITORIA_E2E_HASTA"),
		ct:    auditoriaE2EFuente{"ct", os.Getenv("VEC_AUDITORIA_E2E_CT_EXPEDIENTE"), os.Getenv("VEC_AUDITORIA_E2E_CT_ACTOR"), os.Getenv("VEC_AUDITORIA_E2E_CT_AJENO")},
		bolsa: auditoriaE2EFuente{"bolsa", os.Getenv("VEC_AUDITORIA_E2E_BOLSA_EXPEDIENTE"), os.Getenv("VEC_AUDITORIA_E2E_BOLSA_ACTOR"), os.Getenv("VEC_AUDITORIA_E2E_BOLSA_AJENO")},
	}
	cliente := config.cliente(t)
	var opciones auditoria.Opciones
	config.pedir(t, cliente, http.MethodGet, auditoria.RutaOpciones, nil, http.StatusOK, &opciones)
	if opciones.FinalidadRef == "" || opciones.MotivoRef == "" || opciones.PermisoRequerido != auditoria.AccionConsultar ||
		len(opciones.Fuentes) != 2 || opciones.Fuentes[0] != "ct" || opciones.Fuentes[1] != "bolsa" {
		t.Fatalf("opciones de auditoría incompletas: %+v", opciones)
	}
	for _, fuente := range []auditoriaE2EFuente{config.ct, config.bolsa} {
		t.Run(fuente.nombre, func(t *testing.T) { config.recorrerFuente(t, cliente, opciones, fuente) })
	}
}

type auditoriaE2EFuente struct{ nombre, expediente, actor, ajeno string }
type auditoriaE2EConfiguracion struct {
	url, desde, hasta string
	ct, bolsa         auditoriaE2EFuente
}

func (c auditoriaE2EConfiguracion) cliente(t *testing.T) *http.Client {
	t.Helper()
	u, err := url.Parse(c.url)
	if err != nil || u.Scheme != "https" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") ||
		u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		t.Fatal("VEC_AUDITORIA_E2E_URL debe ser HTTPS local sin credenciales ni ruta")
	}
	desde, errorDesde := time.Parse(time.RFC3339Nano, c.desde)
	hasta, errorHasta := time.Parse(time.RFC3339Nano, c.hasta)
	if errorDesde != nil || errorHasta != nil || !hasta.After(desde) || hasta.Sub(desde) > auditoria.MaximoIntervalo ||
		!strings.HasSuffix(c.desde, "Z") || !strings.HasSuffix(c.hasta, "Z") ||
		c.ct.expediente == "" || c.ct.actor == "" || c.ct.ajeno == "" ||
		c.bolsa.expediente == "" || c.bolsa.actor == "" || c.bolsa.ajeno == "" ||
		c.ct.expediente == c.ct.ajeno || c.bolsa.expediente == c.bolsa.ajeno {
		t.Fatal("fixture HTTP sintético de auditoría incompleto")
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13}
	ca := os.Getenv("VEC_AUDITORIA_E2E_CA")
	if ca != "" {
		certificado, err := os.ReadFile(ca)
		if err != nil {
			t.Fatal("CA de ensayo no disponible")
		}
		pool, err := x509.SystemCertPool()
		if err != nil {
			t.Fatal("almacén de CA no disponible")
		}
		if !pool.AppendCertsFromPEM(certificado) {
			t.Fatal("CA de ensayo inválida")
		}
		tlsConfig.RootCAs = pool
	}
	cert, clave := os.Getenv("VEC_AUDITORIA_E2E_CERT"), os.Getenv("VEC_AUDITORIA_E2E_KEY")
	if (cert == "") != (clave == "") {
		t.Fatal("certificado y clave de cliente deben configurarse juntos")
	}
	if cert != "" {
		par, err := tls.LoadX509KeyPair(cert, clave)
		if err != nil {
			t.Fatal("par de certificado de ensayo no disponible")
		}
		tlsConfig.Certificates = []tls.Certificate{par}
	}
	transporte := http.DefaultTransport.(*http.Transport).Clone()
	transporte.Proxy = nil
	transporte.TLSClientConfig = tlsConfig
	return &http.Client{Transport: transporte, Timeout: 15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func (c auditoriaE2EConfiguracion) pedir(t *testing.T, cliente *http.Client, metodo, ruta string, cuerpo any, estado int, salida any) {
	t.Helper()
	var contenido io.Reader
	if cuerpo != nil {
		b, err := json.Marshal(cuerpo)
		if err != nil {
			t.Fatal(err)
		}
		contenido = bytes.NewReader(b)
	}
	peticion, err := http.NewRequest(metodo, strings.TrimRight(c.url, "/")+ruta, contenido)
	if err != nil {
		t.Fatal(err)
	}
	if cuerpo != nil {
		peticion.Header.Set("Content-Type", "application/json")
	}
	respuesta, err := cliente.Do(peticion)
	if err != nil {
		t.Fatalf("%s %s: transporte de ensayo falló: %v", metodo, ruta, err)
	}
	defer respuesta.Body.Close()
	if respuesta.StatusCode != estado || respuesta.Header.Get("Set-Cookie") != "" {
		t.Fatalf("%s %s: HTTP %d, esperado %d", metodo, ruta, respuesta.StatusCode, estado)
	}
	if salida != nil && json.NewDecoder(io.LimitReader(respuesta.Body, 1024*1024)).Decode(salida) != nil {
		t.Fatalf("%s %s: JSON inválido", metodo, ruta)
	}
}

func (c auditoriaE2EConfiguracion) recorrerFuente(t *testing.T, cliente *http.Client, opciones auditoria.Opciones, fuente auditoriaE2EFuente) {
	t.Helper()
	consulta := map[string]any{
		"fuente": fuente.nombre, "expediente_ref": fuente.expediente, "actor_ref": fuente.actor,
		"desde": c.desde, "hasta": c.hasta, "limite": 1,
		"finalidad_ref": opciones.FinalidadRef, "motivo_ref": opciones.MotivoRef,
	}
	var primera, segunda auditoria.Pagina
	c.pedir(t, cliente, http.MethodPost, auditoria.RutaConsulta, consulta, http.StatusOK, &primera)
	if len(primera.Registros) != 1 || primera.SiguienteCursor == "" {
		t.Fatal("la primera página no contiene un registro y cursor")
	}
	consulta["cursor"] = primera.SiguienteCursor
	c.pedir(t, cliente, http.MethodPost, auditoria.RutaConsulta, consulta, http.StatusOK, &segunda)
	if len(segunda.Registros) != 1 || primera.Registros[0].ID == segunda.Registros[0].ID {
		t.Fatal("la segunda página duplica o pierde el registro")
	}
	conAntesDespues := false
	desde, _ := time.Parse(time.RFC3339Nano, c.desde)
	hasta, _ := time.Parse(time.RFC3339Nano, c.hasta)
	for _, registro := range []auditoria.Registro{primera.Registros[0], segunda.Registros[0]} {
		if registro.Fuente != fuente.nombre || registro.ExpedienteRef != fuente.expediente ||
			registro.ActorRef != fuente.actor || registro.OcurridoEn.Before(desde) || !registro.OcurridoEn.Before(hasta) {
			t.Fatal("fila ajena al filtro exacto")
		}
		if len(registro.Antes) > 0 && len(registro.Despues) > 0 {
			conAntesDespues = true
		}
	}
	if !conAntesDespues {
		t.Fatal("fixture sin antes y después consultables")
	}
	delete(consulta, "cursor")
	consulta["motivo_ref"] = "motivo:ajeno"
	c.pedir(t, cliente, http.MethodPost, auditoria.RutaConsulta, consulta, http.StatusForbidden, nil)
	consulta["motivo_ref"] = opciones.MotivoRef
	consulta["expediente_ref"] = fuente.ajeno
	c.pedir(t, cliente, http.MethodPost, auditoria.RutaConsulta, consulta, http.StatusForbidden, nil)
}
