package adminselector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
)

const perfilA = "prf_ABCDEFGHIJKLMNOPQRSTUV"
const perfilB = "prf_ZYXWVUTSRQPONMLKJIHGFE"

var ahora = time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)

type relojPrueba struct{}

func (relojPrueba) Ahora() time.Time { return ahora }

type observadorPrueba struct {
	observado adminperfiles.ObservacionADMIN
	err       error
	llamadas  int
}

func (o *observadorPrueba) ObservarADMIN(context.Context, *http.Request) (adminperfiles.ObservacionADMIN, error) {
	o.llamadas++
	return o.observado, o.err
}

type selectorPrueba struct {
	lista                adminperfiles.PerfilesPropios
	recibo               adminperfiles.SeleccionPerfil
	err                  error
	lecturas, escrituras int
	perfil               string
	revision             uint64
	observacion          adminperfiles.ObservacionADMIN
}

func (s *selectorPrueba) ListarPropiosADMIN(_ context.Context, o adminperfiles.ObservacionADMIN) (adminperfiles.PerfilesPropios, error) {
	s.lecturas++
	s.observacion = o
	return s.lista, s.err
}
func (s *selectorPrueba) SeleccionarPerfilADMIN(_ context.Context, o adminperfiles.ObservacionADMIN, p string, v uint64) (adminperfiles.SeleccionPerfil, error) {
	s.escrituras++
	s.observacion, s.perfil, s.revision = o, p, v
	return s.recibo, s.err
}

type auditorPrueba struct {
	err       error
	registros []api.DenegacionADMIN
}

func (a *auditorPrueba) RegistrarDenegacionADMIN(_ context.Context, d api.DenegacionADMIN) error {
	a.registros = append(a.registros, d)
	return a.err
}

func escenario(t *testing.T) (*Handler, *observadorPrueba, *selectorPrueba, *auditorPrueba) {
	t.Helper()
	o := &observadorPrueba{observado: adminperfiles.ObservacionADMIN{
		Entorno: "desarrollo", Host: "admin.example.invalid", Audiencia: "vec.admin.perfiles.v1",
		CertificadoSHA256: strings.Repeat("a", 64), CASHA256: strings.Repeat("b", 64),
		AutenticacionVerificadaEn: ahora.Add(-time.Minute), RevocacionVerificadaEn: ahora,
		CRLVigenteHasta: ahora.Add(time.Minute), CertificadoVigenteHasta: ahora.Add(time.Hour),
	}}
	s := &selectorPrueba{
		lista: adminperfiles.PerfilesPropios{Revision: 3, Perfiles: []adminperfiles.PerfilPropio{
			{PerfilRef: perfilA, RolVersionRef: "rol:administracion:v1", ClaveI18N: "admin.rol.administracion", CategoriaADMIN: "aplicacion"},
			{PerfilRef: perfilB, RolVersionRef: "rol:plataforma:v1", ClaveI18N: "admin.rol.plataforma", CategoriaADMIN: "sistemas"},
		}},
		recibo: adminperfiles.SeleccionPerfil{PerfilActivoRef: perfilB, Revision: 4, SeleccionadaEn: ahora, AuditoriaRef: "auditoria_seleccion_admin:" + strings.Repeat("c", 64)},
	}
	a := &auditorPrueba{}
	h, err := NuevoHandler("https://admin.example.invalid", "vec.admin.perfiles.v1", o, s, a, relojPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	return h, o, s, a
}

func peticion(metodo, ruta, cuerpo string) *http.Request {
	r := httptest.NewRequest(metodo, "https://admin.example.invalid"+ruta, strings.NewReader(cuerpo))
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	r.Header.Set("Sec-Fetch-Mode", "cors")
	r.Header.Set("Sec-Fetch-Dest", "empty")
	if metodo == http.MethodPost {
		r.Header.Set("Origin", "https://admin.example.invalid")
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func cuerpoValido() string { return fmt.Sprintf(`{"perfil_ref":%q,"revision_esperada":3}`, perfilB) }

func TestSelectorDelegaPerfilSolicitadoYRevisionSinConceder(t *testing.T) {
	h, o, s, _ := escenario(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodPost, RutaSeleccion, cuerpoValido()))
	if w.Code != 200 || s.escrituras != 1 || s.perfil != perfilB || s.revision != 3 || s.observacion != o.observado {
		t.Fatalf("contrato central perdido: estado=%d selector=%+v", w.Code, s)
	}
	for _, clave := range []string{`"perfil_activo_ref"`, `"revision":4`, `"seleccionada_en"`, `"auditoria_ref"`} {
		if !strings.Contains(w.Body.String(), clave) {
			t.Fatalf("falta %s: %s", clave, w.Body)
		}
	}
	if w.Header().Get("Set-Cookie") != "" || w.Header().Get("Cache-Control") != "no-store, no-transform" {
		t.Fatal("respuesta persistente")
	}
	if strings.Contains(w.Body.String(), o.observado.CertificadoSHA256) {
		t.Fatal("huella expuesta")
	}
}

func TestDosPerfilesNoSeSeleccionanDesdeHTTP(t *testing.T) {
	h, _, s, _ := escenario(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodGet, RutaPropios, ""))
	if w.Code != 200 || s.lecturas != 1 || s.escrituras != 0 || strings.Contains(w.Body.String(), "perfil_activo_ref") {
		t.Fatalf("selección implícita: %d %s", w.Code, w.Body)
	}
}

func TestPerfilSistemasActivoPuedeConsultarYSeleccionarAplicacion(t *testing.T) {
	h, o, s, _ := escenario(t)
	s.lista.PerfilActivoRef = perfilB
	s.recibo.PerfilActivoRef = perfilA
	// Solo están inyectados el observador base y la autoridad central. No se
	// construye ni conecta el proveedor del perfil de Aplicación.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodGet, RutaPropios, ""))
	if w.Code != http.StatusOK || s.lecturas != 1 || s.escrituras != 0 {
		t.Fatalf("Sistemas no pudo consultar el selector: %d", w.Code)
	}
	w = httptest.NewRecorder()
	body := fmt.Sprintf(`{"perfil_ref":%q,"revision_esperada":3}`, perfilA)
	h.ServeHTTP(w, peticion(http.MethodPost, RutaSeleccion, body))
	if w.Code != http.StatusOK || s.escrituras != 1 || s.perfil != perfilA || o.llamadas != 2 {
		t.Fatalf("selección desde Sistemas no delegada en la autoridad central: %d", w.Code)
	}
}

func TestCuentaSoloSistemasPuedeRecuperarSuSeleccion(t *testing.T) {
	h, o, s, _ := escenario(t)
	s.lista.Perfiles = s.lista.Perfiles[1:]
	s.lista.PerfilActivoRef = perfilB
	s.recibo.Revision = s.lista.Revision
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodGet, RutaPropios, ""))
	if w.Code != http.StatusOK || s.lecturas != 1 {
		t.Fatalf("cuenta solo Sistemas denegada: %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodPost, RutaSeleccion, cuerpoValido()))
	if w.Code != http.StatusOK || s.revision != 3 || o.llamadas != 2 {
		t.Fatalf("selección vigente no recuperable: %d", w.Code)
	}
}

func TestCategoriaYClaveI18NProcedenDelCatalogoSinInferirRol(t *testing.T) {
	h, _, s, _ := escenario(t)
	// Mantener el identificador de rol y cambiar el dato de catálogo prueba
	// que la proyección HTTP no clasifica mediante nombres de rol.
	s.lista.Perfiles[0].CategoriaADMIN = "sistemas"
	s.lista.Perfiles[0].ClaveI18N = "catalogo.sistemas.publicado"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodGet, RutaPropios, ""))
	var dto PerfilesPropios
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || len(dto.Perfiles) != 2 || dto.Perfiles[0].CategoriaADMIN != "sistemas" ||
		dto.Perfiles[0].ClaveI18N != s.lista.Perfiles[0].ClaveI18N || dto.Perfiles[0].RolVersionRef != s.lista.Perfiles[0].RolVersionRef {
		t.Fatalf("catálogo alterado: %d %+v", w.Code, dto)
	}
	for _, categoria := range []string{"", "administrador", "Sistemas"} {
		s.lista.Perfiles[0].CategoriaADMIN = categoria
		w = httptest.NewRecorder()
		h.ServeHTTP(w, peticion(http.MethodGet, RutaPropios, ""))
		if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "categoria_admin") {
			t.Fatalf("categoría no aprobada publicada: %q %d", categoria, w.Code)
		}
	}
}

func TestPayloadAmbiguoNuncaLlegaALaAutoridad(t *testing.T) {
	casos := map[string]string{
		"categoria cliente":      strings.TrimSuffix(cuerpoValido(), "}") + `,"categoria_admin":"sistemas"}`,
		"extra":                  strings.TrimSuffix(cuerpoValido(), "}") + `,"rol":"administrador"}`,
		"duplicado":              strings.TrimSuffix(cuerpoValido(), "}") + `,"perfil_ref":"` + perfilA + `"}`,
		"duplicado escapado":     strings.TrimSuffix(cuerpoValido(), "}") + `,"perfil_\u0072ef":"` + perfilA + `"}`,
		"revision duplicada":     strings.TrimSuffix(cuerpoValido(), "}") + `,"revision_esperada":4}`,
		"revision texto":         `{"perfil_ref":"` + perfilB + `","revision_esperada":"3"}`,
		"revision null":          `{"perfil_ref":"` + perfilB + `","revision_esperada":null}`,
		"revision ausente":       `{"perfil_ref":"` + perfilB + `"}`,
		"revision decimal":       `{"perfil_ref":"` + perfilB + `","revision_esperada":3.0}`,
		"revision negativa":      `{"perfil_ref":"` + perfilB + `","revision_esperada":-1}`,
		"revision overflow":      `{"perfil_ref":"` + perfilB + `","revision_esperada":9223372036854775807}`,
		"revision JSON inexacta": `{"perfil_ref":"` + perfilB + `","revision_esperada":9007199254740992}`,
		"array":                  `[` + cuerpoValido() + `]`, "null": `null`, "segunda entidad": cuerpoValido() + cuerpoValido(),
		"perfil con URL": `{"perfil_ref":"https://example.invalid","revision_esperada":3}`,
	}
	for nombre, cuerpo := range casos {
		t.Run(nombre, func(t *testing.T) {
			h, _, s, a := escenario(t)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticion(http.MethodPost, RutaSeleccion, cuerpo))
			if w.Code != 400 || s.escrituras != 0 || len(a.registros) != 1 {
				t.Fatalf("payload aceptado: %d %s", w.Code, w.Body)
			}
		})
	}
}

func TestCuerpoAcotadoIncluyeChunked(t *testing.T) {
	for _, longitud := range []int64{int64(maxCuerpo + 1), -1} {
		t.Run(fmt.Sprint(longitud), func(t *testing.T) {
			h, _, s, _ := escenario(t)
			r := peticion(http.MethodPost, RutaSeleccion, cuerpoValido()+strings.Repeat(" ", maxCuerpo))
			r.ContentLength = longitud
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 413 || s.escrituras != 0 {
				t.Fatalf("exceso aceptado: %d", w.Code)
			}
		})
	}
}

func TestFronteraDeniegaAntesDelObservador(t *testing.T) {
	casos := map[string]struct {
		cambiar func(*http.Request)
		estado  int
	}{
		"cookie":                       {func(r *http.Request) { r.Header.Set("Cookie", "sesion=sintetica") }, 401},
		"authorization vacío":          {func(r *http.Request) { r.Header["Authorization"] = []string{""} }, 401},
		"cabecera identidad minúscula": {func(r *http.Request) { r.Header["x-remote-user"] = []string{"sintetico"} }, 401},
		"cert reenviado":               {func(r *http.Request) { r.Header.Set("X-Forwarded-Client-Cert", "sintetico") }, 401},
		"host ajeno":                   {func(r *http.Request) { r.Host = "otro.example.invalid" }, 401},
		"origin ajeno":                 {func(r *http.Request) { r.Header.Set("Origin", "https://otro.example.invalid") }, 403},
		"origin ausente":               {func(r *http.Request) { r.Header.Del("Origin") }, 403},
		"origin duplicado":             {func(r *http.Request) { r.Header.Add("Origin", "https://admin.example.invalid") }, 403},
		"fetch cross site":             {func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, 403},
		"navegación":                   {func(r *http.Request) { r.Header.Set("Sec-Fetch-Mode", "navigate") }, 403},
		"fetch ausente":                {func(r *http.Request) { r.Header.Del("Sec-Fetch-Site") }, 403},
		"query":                        {func(r *http.Request) { r.URL.RawQuery = "perfil_ref=" + perfilB }, 400},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			h, o, s, a := escenario(t)
			r := peticion(http.MethodPost, RutaSeleccion, cuerpoValido())
			caso.cambiar(r)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado || o.llamadas != 0 || s.escrituras != 0 || len(a.registros) != 1 {
				t.Fatalf("frontera abierta: %d observador=%d selector=%d", w.Code, o.llamadas, s.escrituras)
			}
			if a.registros[0].ActorPersonaRef != "" || a.registros[0].PerfilActivoRef != "" {
				t.Fatal("identidad cliente auditada")
			}
		})
	}
}

func TestObservacionDebeEstarVigenteYLigadaAlListener(t *testing.T) {
	for nombre, cambiar := range map[string]func(*adminperfiles.ObservacionADMIN){
		"crl caducada":    func(o *adminperfiles.ObservacionADMIN) { o.CRLVigenteHasta = ahora },
		"cert caducado":   func(o *adminperfiles.ObservacionADMIN) { o.CertificadoVigenteHasta = ahora },
		"audiencia ajena": func(o *adminperfiles.ObservacionADMIN) { o.Audiencia = "vec.publico" },
		"host ajeno":      func(o *adminperfiles.ObservacionADMIN) { o.Host = "otro.example.invalid" },
		"vacía":           func(o *adminperfiles.ObservacionADMIN) { *o = adminperfiles.ObservacionADMIN{} },
	} {
		t.Run(nombre, func(t *testing.T) {
			h, o, s, _ := escenario(t)
			cambiar(&o.observado)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticion(http.MethodGet, RutaPropios, ""))
			if w.Code != 401 || s.lecturas != 0 {
				t.Fatalf("observación inválida aceptada: %d", w.Code)
			}
		})
	}
}

func TestFallosCentralesNoConfirmanSeleccionNiExponenDetalles(t *testing.T) {
	for _, caso := range []struct {
		err    error
		estado int
	}{
		{api.ErrAutenticacionRequerida, 401}, {api.ErrAccesoDenegado, 403}, {fmt.Errorf("CAS: %w", api.ErrConflictoEstado), 409},
		{errors.New("DSN_SECRETO_SINTETICO"), 503},
	} {
		t.Run(fmt.Sprint(caso.estado), func(t *testing.T) {
			h, _, s, a := escenario(t)
			s.err = caso.err
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticion(http.MethodPost, RutaSeleccion, cuerpoValido()))
			if w.Code != caso.estado || strings.Contains(w.Body.String(), "DSN") || strings.Contains(w.Body.String(), "perfil_activo_ref") ||
				len(a.registros) != 1 || a.registros[0].RecursoRef != RutaSeleccion ||
				a.registros[0].Accion != "seleccionar_perfil_propio" {
				t.Fatalf("error no normalizado: %d %s", w.Code, w.Body)
			}
		})
	}
}

func TestDenegacionCentralDependeDelReciboDeAuditoria(t *testing.T) {
	h, _, s, a := escenario(t)
	s.err = api.ErrConflictoEstado
	a.err = errors.New("auditoria indisponible")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodPost, RutaSeleccion, cuerpoValido()))
	if w.Code != http.StatusServiceUnavailable || len(a.registros) != 1 ||
		a.registros[0].RecursoRef != RutaSeleccion || s.escrituras != 1 {
		t.Fatalf("denegación sin auditoría: estado=%d registros=%d", w.Code, len(a.registros))
	}
}

func TestDependenciasYAuditoriaFallanCerrado(t *testing.T) {
	h, o, s, a := escenario(t)
	var ausente *observadorPrueba
	if _, err := NuevoHandler(h.origen, h.audiencia, ausente, s, a, relojPrueba{}); !errors.Is(err, api.ErrConfiguracionIncompleta) {
		t.Fatal("observador typed-nil admitido")
	}
	a.err = errors.New("auditor caído")
	r := peticion(http.MethodPost, RutaSeleccion, cuerpoValido())
	r.Header.Set("Cookie", "sintetica")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 503 || o.llamadas != 0 || s.escrituras != 0 {
		t.Fatal("efecto sin auditoría")
	}
	var vacio Handler
	w = httptest.NewRecorder()
	vacio.ServeHTTP(w, peticion(http.MethodGet, RutaPropios, ""))
	if w.Code != 503 {
		t.Fatal("handler sin dependencias disponible")
	}
}

func TestGETNoAceptaCuerpoNiLecturaIncoherente(t *testing.T) {
	h, _, s, _ := escenario(t)
	r := peticion(http.MethodGet, RutaPropios, "x")
	r.ContentLength = 0
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 || s.lecturas != 0 {
		t.Fatal("GET con cuerpo consultó autoridad")
	}
	s.lista.PerfilActivoRef = "prf_NOASIGNADOABCDEFGHIJKL"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodGet, RutaPropios, ""))
	if w.Code != 503 {
		t.Fatal("lectura con activo ajeno publicada")
	}
}

// La selección auditada real (IS14/AD171) devuelve la referencia de la
// auditoría común; el manejador debe publicarla tras el COMMIT.
func TestPOSTPublicaReferenciaDeAuditoriaComun(t *testing.T) {
	h, _, s, _ := escenario(t)
	s.recibo.AuditoriaRef = "aud_v3_p_" + strings.Repeat("0a", 16)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion(http.MethodPost, RutaSeleccion, cuerpoValido()))
	if w.Code != 200 || !strings.Contains(w.Body.String(), s.recibo.AuditoriaRef) {
		t.Fatalf("referencia común rechazada: %d %s", w.Code, w.Body)
	}
	for _, mala := range []string{"aud_v3_p_" + strings.Repeat("A", 32), "aud_v3_p_" + strings.Repeat("a", 31), "aud_v3_p_" + strings.Repeat("a", 33), "aud_v3_x_" + strings.Repeat("a", 32)} {
		if auditoriaValida(mala) {
			t.Fatalf("referencia malformada admitida: %s", mala)
		}
	}
}

func TestPOSTNoConfirmaReciboIncoherente(t *testing.T) {
	for nombre, cambiar := range map[string]func(*adminperfiles.SeleccionPerfil){
		"perfil ajeno":  func(s *adminperfiles.SeleccionPerfil) { s.PerfilActivoRef = perfilA },
		"sin auditoría": func(s *adminperfiles.SeleccionPerfil) { s.AuditoriaRef = "" },
		"CAS anterior":  func(s *adminperfiles.SeleccionPerfil) { s.Revision = 2 },
		"CAS salto":     func(s *adminperfiles.SeleccionPerfil) { s.Revision = 5 },
		"fecha ausente": func(s *adminperfiles.SeleccionPerfil) { s.SeleccionadaEn = time.Time{} },
	} {
		t.Run(nombre, func(t *testing.T) {
			h, _, s, _ := escenario(t)
			cambiar(&s.recibo)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticion(http.MethodPost, RutaSeleccion, cuerpoValido()))
			if w.Code != 503 {
				t.Fatalf("recibo incoherente publicado: %d", w.Code)
			}
		})
	}
}
