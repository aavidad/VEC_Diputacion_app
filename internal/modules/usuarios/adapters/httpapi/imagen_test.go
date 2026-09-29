package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	imagenadapter "vec-diputacion-granada/internal/modules/usuarios/adapters/imagen"
	"vec-diputacion-granada/internal/modules/usuarios/application"
	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorImagenHTTPPrueba struct{ emisiones int }

func (p *proveedorImagenHTTPPrueba) ProveerMaterialImagen(_ context.Context, _ core.VinculoAutenticacionActorV2, m ports.MaterialImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.emisiones++
	vacia := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	audiencia, err := canonico.AudienciaImagen(m.Accion, m.Superficie)
	if err != nil {
		return vacia, err
	}
	recurso, err := canonico.RecursoImagen(m)
	if err != nil {
		return vacia, err
	}
	huella, _ := recurso.HuellaContextoAutorizacionSHA256()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	r, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(fmt.Sprintf("dec_img_%d", p.emisiones), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion, m.PersonaRef, huella, audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		return vacia, err
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{byte('a' + p.emisiones)}, 512), r, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
}

type ordenImagenHTTPPrueba struct {
	orden ports.OrdenImagen
	err   error
}

func (o ordenImagenHTTPPrueba) ResolverOrdenImagen(context.Context) (ports.OrdenImagen, error) {
	return o.orden, o.err
}

type registroImagenHTTPPrueba struct {
	persona string
	foto    *ports.FotoImagen
	estado  *ports.EstadoImagen
	err     error
	bytes   []byte
}

func (r *registroImagenHTTPPrueba) CatalogoVigente(context.Context, ports.OrdenImagen) (domain.CatalogoImagen, error) {
	return domain.CatalogoBaseImagen(), nil
}
func (r *registroImagenHTTPPrueba) Consultar(context.Context, ports.OrdenImagen, ports.MaterialImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.EstadoImagen, bool, *ports.FotoImagen, error) {
	if r.estado == nil {
		return ports.EstadoImagen{PersonaRef: r.persona}, false, nil, r.err
	}
	return *r.estado, true, r.foto, r.err
}
func (r *registroImagenHTTPPrueba) RecuperarOperacion(context.Context, ports.OrdenImagen, ports.MaterialImagen, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboImagen, bool, error) {
	return ports.ReciboImagen{}, false, nil
}
func (r *registroImagenHTTPPrueba) Guardar(_ context.Context, _ ports.OrdenImagen, m ports.MaterialImagen, foto []byte, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboImagen, error) {
	if r.err != nil {
		return ports.ReciboImagen{}, r.err
	}
	r.bytes = append([]byte(nil), foto...)
	return ports.ReciboImagen{ReciboRef: "img_" + strings.Repeat("1", 32), PersonaRef: r.persona, Version: m.VersionEsperada + 1, CatalogoVersionRef: m.CatalogoVersionRef, Eleccion: m.Eleccion, FotoNueva: len(foto) > 0, FechaUTC: time.Now().UTC()}, nil
}

func manejadorImagenPrueba(t *testing.T) (*ManejadorImagen, *registroImagenHTTPPrueba, *auditorHTTPPrueba) {
	t.Helper()
	actor, vinculo := identidadHTTPPrueba(t, core.SuperficieAutenticacionInternaCorporativaV1)
	orden, err := application.NuevaOrdenImagen(actor, vinculo, core.SuperficieAutenticacionInternaCorporativaV1, &proveedorImagenHTTPPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	r := &registroImagenHTTPPrueba{persona: actor.PersonaRef}
	s, err := application.NuevoServicioImagen(r, imagenadapter.Nuevo(1), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	auditor := &auditorHTTPPrueba{}
	m, err := NuevoManejadorImagenEnRuta(s, ordenImagenHTTPPrueba{orden: orden}, auditor, RutaMiImagen)
	if err != nil {
		t.Fatal(err)
	}
	return m, r, auditor
}

func llamarImagenHTTP(t *testing.T, m http.Handler, metodo string, cuerpo []byte) (int, map[string]json.RawMessage) {
	t.Helper()
	r := httptest.NewRequest(metodo, RutaMiImagen, bytes.NewReader(cuerpo))
	if metodo == http.MethodPost {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Header().Get("Cache-Control") != "private, no-store, max-age=0" || w.Header().Get("Set-Cookie") != "" || w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatal("cabeceras de privacidad o tipo ausentes")
	}
	var sobre map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &sobre); err != nil {
		t.Fatalf("respuesta no JSON: %s", w.Body.String())
	}
	return w.Code, sobre
}

func pngPrueba(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	img.Set(1, 1, color.RGBA{255, 0, 0, 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func cuerpoImagen(operacion, eleccion, foto string) []byte {
	c := `{"operacion":"` + operacion + `","version_esperada":0,"catalogo_version_ref":"usuarios-imagen-v1","clave_operacion":"web-imagen-1234567890","eleccion":` + eleccion
	if foto != "" {
		c += `,"foto_base64":"` + foto + `"`
	}
	return []byte(c + "}")
}

func TestImagenGETDevuelveCatalogoEstadoYFotoSinPersona(t *testing.T) {
	m, r, _ := manejadorImagenPrueba(t)
	estado, sobre := llamarImagenHTTP(t, m, http.MethodGet, nil)
	if estado != http.StatusOK || !bytes.Contains(sobre["data"], []byte(`"foto":null`)) || !bytes.Contains(sobre["data"], []byte(`"paletas":[`)) || bytes.Contains(sobre["data"], []byte("per_")) {
		t.Fatalf("GET: %d %s", estado, sobre["data"])
	}
	r.estado = &ports.EstadoImagen{PersonaRef: r.persona, Version: 2, CatalogoVersionRef: "usuarios-imagen-v1", Eleccion: domain.EleccionImagen{Modo: domain.ModoImagenFoto, Paleta: "azul"}}
	r.foto = &ports.FotoImagen{Tipo: ports.TipoFotoImagen, Datos: []byte{0xff, 0xd8, 0xff, 7, 0xff, 0xd9}}
	estado, sobre = llamarImagenHTTP(t, m, http.MethodGet, nil)
	if estado != http.StatusOK || !bytes.Contains(sobre["data"], []byte(`"foto":{"tipo":"image/jpeg","datos":"/9j/B//Z"}`)) {
		t.Fatalf("GET con foto: %d %s", estado, sobre["data"])
	}
}

func TestImagenPOSTEligeYSubeFotoRecodificada(t *testing.T) {
	m, r, _ := manejadorImagenPrueba(t)
	estado, sobre := llamarImagenHTTP(t, m, http.MethodPost, cuerpoImagen("elegir", `{"modo":"icono","paleta":"verde","icono":"sol"}`, ""))
	if estado != http.StatusCreated || !bytes.Contains(sobre["data"], []byte(`"version":1`)) || r.bytes != nil {
		t.Fatalf("elegir: %d %s", estado, sobre["data"])
	}
	original := pngPrueba(t)
	estado, sobre = llamarImagenHTTP(t, m, http.MethodPost, cuerpoImagen("subir_foto", `{"modo":"foto","paleta":"azul","icono":""}`, base64.StdEncoding.EncodeToString(original)))
	if estado != http.StatusCreated || !bytes.Contains(sobre["data"], []byte(`"foto_nueva":true`)) {
		t.Fatalf("subir: %d %s", estado, sobre["data"])
	}
	if !bytes.HasPrefix(r.bytes, []byte{0xff, 0xd8, 0xff}) || bytes.Equal(r.bytes, original) {
		t.Fatal("se custodió el original en lugar del JPEG recodificado")
	}
}

func TestImagenRechazaFicherosNoAdmitidosYGrandes(t *testing.T) {
	m, r, _ := manejadorImagenPrueba(t)
	foto := `{"modo":"foto","paleta":"azul","icono":""}`
	svg := base64.StdEncoding.EncodeToString([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))
	if estado, sobre := llamarImagenHTTP(t, m, http.MethodPost, cuerpoImagen("subir_foto", foto, svg)); estado != http.StatusUnprocessableEntity || !bytes.Contains(sobre["error"], []byte(`"foto_no_admitida"`)) {
		t.Fatalf("SVG admitido: %d %s", estado, sobre["error"])
	}
	if estado, sobre := llamarImagenHTTP(t, m, http.MethodPost, cuerpoImagen("subir_foto", foto, "no es base64!")); estado != http.StatusUnprocessableEntity || !bytes.Contains(sobre["error"], []byte(`"foto_no_admitida"`)) {
		t.Fatalf("base64 inválido: %d %s", estado, sobre["error"])
	}
	grande := strings.Repeat("A", int(limiteFotoBase64)+4)
	if estado, sobre := llamarImagenHTTP(t, m, http.MethodPost, cuerpoImagen("subir_foto", foto, grande)); estado != http.StatusRequestEntityTooLarge || !bytes.Contains(sobre["error"], []byte(`"foto_grande"`)) {
		t.Fatalf("foto enorme: %d %s", estado, sobre["error"])
	}
	if r.bytes != nil {
		t.Fatal("una foto rechazada llegó al registro")
	}
}

func TestImagenRechazaCuerposFueraDeContrato(t *testing.T) {
	m, _, _ := manejadorImagenPrueba(t)
	foto := base64.StdEncoding.EncodeToString(pngPrueba(t))
	for _, cuerpo := range [][]byte{
		cuerpoImagen("elegir", `{"modo":"iniciales","paleta":"azul","icono":""}`, foto),
		cuerpoImagen("subir_foto", `{"modo":"foto","paleta":"azul","icono":""}`, ""),
		cuerpoImagen("elegir", `{"modo":"iniciales","paleta":"azul"}`, ""),
		cuerpoImagen("elegir", `{"modo":"iniciales","paleta":"#ff0000","icono":""}`, ""),
		cuerpoImagen("elegir", `{"modo":"iniciales","paleta":"azul","icono":"","color":"rojo"}`, ""),
		cuerpoImagen("borrar", `{"modo":"iniciales","paleta":"azul","icono":""}`, ""),
		[]byte(`{"operacion":"elegir","operacion":"elegir","version_esperada":0,"catalogo_version_ref":"usuarios-imagen-v1","clave_operacion":"web-imagen-1234567890","eleccion":{"modo":"iniciales","paleta":"azul","icono":""}}`),
		[]byte(`{"operacion":"elegir","version_esperada":0,"catalogo_version_ref":"usuarios-imagen-v1","clave_operacion":"web-imagen-1234567890","eleccion":{"modo":"iniciales","paleta":"azul","icono":""},"persona_ref":"per_otra"}`),
		[]byte(`[]`),
	} {
		if estado, _ := llamarImagenHTTP(t, m, http.MethodPost, cuerpo); estado != http.StatusUnprocessableEntity {
			t.Fatalf("cuerpo aceptado (%d): %.200s", estado, cuerpo)
		}
	}
	if estado, _ := llamarImagenHTTP(t, m, http.MethodPut, []byte(`{}`)); estado != http.StatusMethodNotAllowed {
		t.Fatalf("PUT: %d", estado)
	}
	r := httptest.NewRequest(http.MethodGet, RutaMiImagen+"?persona=per_otra", nil)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("consulta con parámetros: %d", w.Code)
	}
}

func TestImagenDenegacionSeAuditaYErroresNoFiltranDetalles(t *testing.T) {
	m, r, auditor := manejadorImagenPrueba(t)
	r.err = errors.New("detalle interno docimg_123")
	if estado, sobre := llamarImagenHTTP(t, m, http.MethodPost, cuerpoImagen("elegir", `{"modo":"iniciales","paleta":"azul","icono":""}`, "")); estado != http.StatusServiceUnavailable || bytes.Contains(sobre["error"], []byte("docimg")) {
		t.Fatalf("error interno filtrado: %d %s", estado, sobre["error"])
	}
	m.orden = ordenImagenHTTPPrueba{err: ports.ErrImagenNoAutenticado}
	if estado, _ := llamarImagenHTTP(t, m, http.MethodGet, nil); estado != http.StatusUnauthorized || len(auditor.estados) != 1 || auditor.estados[0] != 401 {
		t.Fatalf("401 sin auditoría: %d %v", estado, auditor.estados)
	}
	auditor.err = errors.New("auditoría caída")
	if estado, _ := llamarImagenHTTP(t, m, http.MethodGet, nil); estado != http.StatusServiceUnavailable {
		t.Fatalf("denegación sin auditoría devolvió %d", estado)
	}
	if _, err := NuevoManejadorImagenEnRuta(&application.ServicioImagen{}, m.orden, auditor, RutaMisCorreos); err == nil {
		t.Fatal("manejador en ruta ajena")
	}
}
