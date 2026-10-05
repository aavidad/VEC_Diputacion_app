package httpapi

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/application"
	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const refCorreoHTTP = "correo:0123456789abcdef0123456789abcdef"

type proveedorCorreosHTTPPrueba struct{ emisiones int }

func (p *proveedorCorreosHTTPPrueba) ProveerMaterialCorreos(_ context.Context, _ core.VinculoAutenticacionActorV2, m ports.MaterialCorreos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.emisiones++
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	audiencia, err := canonico.AudienciaCorreos(m.Accion, m.Superficie)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	recurso, err := canonico.RecursoCorreos(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	huella, _ := recurso.HuellaContextoAutorizacionSHA256()
	r, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(fmt.Sprintf("dec_prueba_%d", p.emisiones), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion, m.PersonaRef, huella, audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{byte('x' + p.emisiones)}, 512), r, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
}

type ordenCorreosHTTPPrueba struct {
	orden ports.OrdenCorreos
	err   error
}

func (o ordenCorreosHTTPPrueba) ResolverOrdenCorreos(context.Context) (ports.OrdenCorreos, error) {
	return o.orden, o.err
}

type registroCorreosHTTPPrueba struct {
	persona           string
	err               error
	vista             ports.VistaCorreos
	consultas         int
	consultaTerminada bool
}

func (r *registroCorreosHTTPPrueba) ConsultarPropios(context.Context, ports.OrdenCorreos, ports.MaterialCorreos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.VistaCorreos, error) {
	r.consultas++
	r.consultaTerminada = true
	return r.vista, r.err
}

type auditorIntentosConsultaHTTPPrueba struct {
	estados                   []int
	preparaciones             int
	errPreparar, errRegistrar error
	alRegistrar               func()
}

func (a *auditorIntentosConsultaHTTPPrueba) PrepararIntentoConsultaCorreos(context.Context) error {
	a.preparaciones++
	return a.errPreparar
}
func (a *auditorIntentosConsultaHTTPPrueba) AuditarIntentoConsultaCorreos(_ context.Context, estado int) error {
	if a.alRegistrar != nil {
		a.alRegistrar()
	}
	a.estados = append(a.estados, estado)
	return a.errRegistrar
}

func TestConsultaCorreosInternaErroresAuditadosAntesDeResponder(t *testing.T) {
	for _, errConsulta := range []error{ports.ErrCorreosNoAutenticado, ports.ErrCorreosProhibido, ports.ErrCorreosConflicto, ports.ErrCorreosInvalidos, ports.ErrCorreosLimite, ports.ErrCorreosNoDisponible, context.Canceled} {
		t.Run(errConsulta.Error(), func(t *testing.T) {
			legacy, registro, frontera := manejadorCorreosPrueba(t)
			w := httptest.NewRecorder()
			a := &auditorIntentosConsultaHTTPPrueba{alRegistrar: func() {
				if !registro.consultaTerminada || w.Body.Len() != 0 {
					t.Fatal("registró antes de retorno SQL o después de escribir respuesta")
				}
			}}
			m, err := NuevoManejadorConsultaCorreosInternaConIntentos(legacy.servicio, legacy.orden, frontera, a)
			if err != nil {
				t.Fatal(err)
			}
			registro.err = errConsulta
			m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMisCorreos, nil))
			estado, _ := clasificarErrorCorreos(errConsulta)
			if w.Code != estado || len(a.estados) != 1 || a.estados[0] != estado || len(frontera.estados) != 0 || registro.consultas != 1 {
				t.Fatal("error sin auditoría común o con registro legado duplicado")
			}
		})
	}
}

func TestConsultaCorreosInternaCuerpoInvalidoSeAuditaSinConsultar(t *testing.T) {
	legacy, registro, frontera := manejadorCorreosPrueba(t)
	a := &auditorIntentosConsultaHTTPPrueba{}
	m, err := NuevoManejadorConsultaCorreosInternaConIntentos(legacy.servicio, legacy.orden, frontera, a)
	if err != nil {
		t.Fatal(err)
	}
	if estado, _ := peticionCorreos(t, m, http.MethodGet, `{}`); estado != 422 || registro.consultas != 0 || len(a.estados) != 1 || a.estados[0] != 422 {
		t.Fatal("GET inválido consultó datos o perdió auditoría")
	}
}

func TestConsultaCorreosInternaRegistroCaidoNoDevuelveDatos(t *testing.T) {
	legacy, registro, frontera := manejadorCorreosPrueba(t)
	a := &auditorIntentosConsultaHTTPPrueba{errRegistrar: errors.New("registro caído")}
	m, err := NuevoManejadorConsultaCorreosInternaConIntentos(legacy.servicio, legacy.orden, frontera, a)
	if err != nil {
		t.Fatal(err)
	}
	for _, errorOrden := range []error{ports.ErrCorreosNoAutenticado, ports.ErrCorreosProhibido, ports.ErrCorreosNoDisponible} {
		m.orden = ordenCorreosHTTPPrueba{err: errorOrden}
		estado, sobre := peticionCorreos(t, m, http.MethodGet, "")
		if estado != 503 || len(sobre["data"]) != 0 || registro.consultas != 0 || len(frontera.estados) != 0 {
			t.Fatal("registro fallido expuso datos o volvió al legado")
		}
	}
	a.errPreparar = errors.New("sin captura")
	previas := len(a.estados)
	if estado, _ := peticionCorreos(t, m, http.MethodGet, ""); estado != 503 || len(a.estados) != previas || registro.consultas != 0 {
		t.Fatal("sin captura fabricó auditoría o consultó")
	}
}

func TestConsultaCorreosInternaNoDuplicaPositivoNiCambiaPOST(t *testing.T) {
	legacy, registro, frontera := manejadorCorreosPrueba(t)
	a := &auditorIntentosConsultaHTTPPrueba{}
	m, err := NuevoManejadorConsultaCorreosInternaConIntentos(legacy.servicio, legacy.orden, frontera, a)
	if err != nil {
		t.Fatal(err)
	}
	if estado, _ := peticionCorreos(t, m, http.MethodGet, ""); estado != 200 || len(a.estados) != 0 || a.preparaciones != 1 || registro.consultas != 1 {
		t.Fatal("positivo no consultó o añadió intento")
	}
	m.orden = ordenCorreosHTTPPrueba{err: ports.ErrCorreosProhibido}
	if estado, _ := peticionCorreos(t, m, http.MethodPost, `{}`); estado != 403 || len(a.estados) != 0 || a.preparaciones != 1 || len(frontera.estados) != 1 {
		t.Fatal("POST usó captura o registrador de consulta")
	}
	if _, err := NuevoManejadorConsultaCorreosInternaConIntentos(legacy.servicio, legacy.orden, frontera, nil); err == nil {
		t.Fatal("constructor interno sin registrador")
	}
	var nulo *auditorIntentosConsultaHTTPPrueba
	if _, err := NuevoManejadorConsultaCorreosInternaConIntentos(legacy.servicio, legacy.orden, frontera, nulo); err == nil {
		t.Fatal("constructor interno con registrador nil tipado")
	}
}
func (r *registroCorreosHTTPPrueba) RecuperarOperacion(context.Context, ports.OrdenCorreos, ports.MaterialCorreos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboCorreos, bool, error) {
	return ports.ReciboCorreos{}, false, nil
}
func (r *registroCorreosHTTPPrueba) Aplicar(_ context.Context, _ ports.OrdenCorreos, p ports.PeticionCorreo, m ports.MaterialCorreos, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, _ ports.SobreDireccionCorreo, reserva ports.ReservaDesafio, _ ports.ComprobadorCodigoCorreo) (ports.ResultadoCorreos, error) {
	if r.err != nil {
		return ports.ResultadoCorreos{}, r.err
	}
	resultado := ports.ResultadoCorreos{Recibo: ports.ReciboCorreos{ReciboRef: "correo_recibo:" + strings.Repeat("1", 32), PersonaRef: r.persona, Accion: m.Accion, CorreoRef: p.CorreoRef, Version: p.VersionEsperada + 1, FechaUTC: time.Now().UTC()}}
	if m.Accion == ports.AccionAnadirCorreo {
		resultado.Envios = []ports.EnvioPendiente{{EnvioRef: "correo_envio:" + strings.Repeat("2", 32), ReservaRef: "reserva:" + strings.Repeat("3", 32), Tipo: ports.TipoEnvioCodigo, CorreoRef: p.CorreoRef, DesafioRef: reserva.DesafioRef}}
	}
	return resultado, nil
}
func (r *registroCorreosHTTPPrueba) ConfirmarEnvio(context.Context, ports.OrdenCorreos, ports.EnvioPendiente, bool) error {
	return nil
}

type criptoCorreosHTTPPrueba struct{}

func (criptoCorreosHTTPPrueba) CifrarDireccionCorreo(_ context.Context, _, ref string, version uint64, claro []byte) (ports.SobreDireccionCorreo, error) {
	return ports.SobreDireccionCorreo{CorreoRef: ref, Version: version, ClaveRef: "c", ClaveIgualdadRef: "i", Nonce: make([]byte, 12), Cifrado: append([]byte(nil), claro...), HuellaIgualdad: make([]byte, 32)}, nil
}
func (criptoCorreosHTTPPrueba) ConDireccionCorreoDescifrada(_ context.Context, _ string, _ ports.SobreDireccionCorreo, usar func([]byte) error) error {
	return usar([]byte("a@example.org"))
}
func (criptoCorreosHTTPPrueba) SellarHuellaCorreo(context.Context, []byte) (ports.HuellasSemanticasCorreo, error) {
	return ports.HuellasSemanticasCorreo{Activa: ports.HuellaSemanticaCorreo{ClaveRef: "h", Valor: strings.Repeat("a", 64)}}, nil
}
func (criptoCorreosHTTPPrueba) PrepararDesafioCorreo(_ context.Context, _, _ string, vence time.Time) (ports.ReservaDesafio, error) {
	return ports.ReservaDesafio{DesafioRef: "desafio:" + strings.Repeat("d", 32), Codigo: "12345678", HuellaCodigo: make([]byte, 32), ClaveRef: "k", VenceUTC: vence}, nil
}
func (criptoCorreosHTTPPrueba) ComprobarCodigoCorreo(context.Context, ports.MetadatosDesafioCorreo, string) (bool, error) {
	return true, nil
}
func (criptoCorreosHTTPPrueba) EnviarCorreoPropio(context.Context, ports.MensajeCorreoPropio) bool {
	return true
}

func manejadorCorreosPrueba(t *testing.T) (*ManejadorCorreos, *registroCorreosHTTPPrueba, *auditorHTTPPrueba) {
	t.Helper()
	actor, vinculo := identidadHTTPPrueba(t, core.SuperficieAutenticacionInternaCorporativaV1)
	orden, err := application.NuevaOrdenCorreos(actor, vinculo, core.SuperficieAutenticacionInternaCorporativaV1, &proveedorCorreosHTTPPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	r := &registroCorreosHTTPPrueba{persona: actor.PersonaRef, vista: ports.VistaCorreos{PersonaRef: actor.PersonaRef, Version: 1, Correos: []domain.CorreoPropio{{CorreoRef: refCorreoHTTP, Direccion: "a@example.org", Estado: domain.CorreoPendiente, CreadoUTC: ahora}}}}
	c := criptoCorreosHTTPPrueba{}
	s, err := application.NuevoServicioCorreos(application.DependenciasCorreos{Registro: r, Protector: c, Sellador: c, Desafios: c, Validador: c, Transporte: c, AhoraUTC: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	auditor := &auditorHTTPPrueba{}
	m, err := NuevoManejadorCorreosEnRuta(s, ordenCorreosHTTPPrueba{orden: orden}, auditor, RutaMisCorreos)
	if err != nil {
		t.Fatal(err)
	}
	return m, r, auditor
}

func peticionCorreos(t *testing.T, m http.Handler, metodo, cuerpo string) (int, map[string]json.RawMessage) {
	t.Helper()
	r := httptest.NewRequest(metodo, RutaMisCorreos, strings.NewReader(cuerpo))
	if metodo == http.MethodPost {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Header().Get("Cache-Control") != "private, no-store, max-age=0" || w.Header().Get("Set-Cookie") != "" {
		t.Fatal("cabeceras de privacidad ausentes")
	}
	var sobre map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &sobre); err != nil {
		t.Fatalf("respuesta no JSON: %s", w.Body.String())
	}
	return w.Code, sobre
}

func TestCorreosGETDevuelveConjuntoSinPersona(t *testing.T) {
	m, _, _ := manejadorCorreosPrueba(t)
	estado, sobre := peticionCorreos(t, m, http.MethodGet, "")
	if estado != http.StatusOK || !bytes.Contains(sobre["data"], []byte(`"correos":[{"correo_ref":"`+refCorreoHTTP)) || bytes.Contains(sobre["data"], []byte("per_")) {
		t.Fatalf("GET: %d %s", estado, sobre["data"])
	}
}

func TestCorreosPOSTAltaYCodigos(t *testing.T) {
	m, r, _ := manejadorCorreosPrueba(t)
	estado, sobre := peticionCorreos(t, m, http.MethodPost, `{"operacion":"anadir","version_esperada":1,"clave_operacion":"clave-http-1234567890","direccion":"nueva@example.org"}`)
	if estado != http.StatusCreated || !bytes.Contains(sobre["data"], []byte(`"envio":"aceptado"`)) || bytes.Contains(sobre["data"], []byte("12345678")) || bytes.Contains(sobre["data"], []byte("per_")) {
		t.Fatalf("alta: %d %s", estado, sobre["data"])
	}
	r.err = ports.CodigoIncorrecto{IntentosRestantes: 2}
	estado, sobre = peticionCorreos(t, m, http.MethodPost, `{"operacion":"verificar","version_esperada":1,"clave_operacion":"clave-http-1234567891","correo_ref":"`+refCorreoHTTP+`","codigo":"1234 5678"}`)
	if estado != http.StatusUnprocessableEntity || !bytes.Contains(sobre["error"], []byte(`"codigo":"codigo_incorrecto"`)) || !bytes.Contains(sobre["error"], []byte(`"intentos_restantes":2`)) {
		t.Fatalf("código incorrecto: %d %s", estado, sobre["error"])
	}
	for err, esperado := range map[error]int{ports.ErrCorreosCodigoCaducado: 409, ports.ErrCorreosLimite: 429, ports.ErrCorreosYaRegistrado: 409, ports.ErrCorreosEnUso: 409, ports.ErrCorreosMaximo: 409, ports.ErrCorreosConflicto: 409, errors.New("interno"): 503} {
		r.err = err
		estado, sobre = peticionCorreos(t, m, http.MethodPost, `{"operacion":"retirar","version_esperada":1,"clave_operacion":"clave-http-1234567892","correo_ref":"`+refCorreoHTTP+`"}`)
		if estado != esperado || bytes.Contains(sobre["error"], []byte("interno")) {
			t.Fatalf("%v -> %d %s", err, estado, sobre["error"])
		}
	}
}

func TestCorreosRechazaCuerposFueraDeContrato(t *testing.T) {
	m, _, _ := manejadorCorreosPrueba(t)
	for _, cuerpo := range []string{
		`{"operacion":"anadir","version_esperada":0,"clave_operacion":"clave-http-1234567890"}`,
		`{"operacion":"anadir","version_esperada":0,"clave_operacion":"clave-http-1234567890","direccion":"a@example.org","correo_ref":"` + refCorreoHTTP + `"}`,
		`{"operacion":"verificar","version_esperada":0,"clave_operacion":"clave-http-1234567890","correo_ref":"` + refCorreoHTTP + `"}`,
		`{"operacion":"borrar","version_esperada":0,"clave_operacion":"clave-http-1234567890","correo_ref":"` + refCorreoHTTP + `"}`,
		`{"operacion":"activar","operacion":"activar","version_esperada":0,"clave_operacion":"clave-http-1234567890","correo_ref":"` + refCorreoHTTP + `"}`,
		`{"operacion":"activar","version_esperada":0,"clave_operacion":"clave-http-1234567890","correo_ref":null}`,
		`{"operacion":"activar","version_esperada":0,"clave_operacion":"clave-http-1234567890","persona_ref":"per_otra"}`,
		`[]`,
	} {
		if estado, _ := peticionCorreos(t, m, http.MethodPost, cuerpo); estado != http.StatusUnprocessableEntity {
			t.Fatalf("cuerpo aceptado (%d): %s", estado, cuerpo)
		}
	}
	if estado, _ := peticionCorreos(t, m, http.MethodPut, `{}`); estado != http.StatusMethodNotAllowed {
		t.Fatalf("PUT: %d", estado)
	}
}

func TestCorreosDenegacionSeAudita(t *testing.T) {
	m, _, auditor := manejadorCorreosPrueba(t)
	m.orden = ordenCorreosHTTPPrueba{err: ports.ErrCorreosNoAutenticado}
	if estado, _ := peticionCorreos(t, m, http.MethodGet, ""); estado != http.StatusUnauthorized || len(auditor.estados) != 1 || auditor.estados[0] != 401 {
		t.Fatalf("401 sin auditoría: %d %v", estado, auditor.estados)
	}
	auditor.err = errors.New("auditoría caída")
	if estado, _ := peticionCorreos(t, m, http.MethodGet, ""); estado != http.StatusServiceUnavailable {
		t.Fatalf("denegación sin auditoría devolvió %d", estado)
	}
	if _, err := NuevoManejadorCorreosEnRuta(nil, m.orden, auditor, RutaMisCorreos); err == nil {
		t.Fatal("manejador sin servicio")
	}
	if _, err := NuevoManejadorCorreosEnRuta(&application.ServicioCorreos{}, m.orden, auditor, RutaMisPreferencias); err == nil {
		t.Fatal("manejador en ruta ajena")
	}
}
