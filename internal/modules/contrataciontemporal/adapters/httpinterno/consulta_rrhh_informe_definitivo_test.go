package httpinterno

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Doble exclusivo del transporte: el PDF real y sus precondiciones se prueban
// en el adaptador documental; aquí no se fabrica autoridad ni persistencia.
type renderizadorBorradorRRHHPrueba struct {
	contenido    []byte
	err          error
	llamadas     int
	tipo         ports.TipoBorradorRRHH
	alRenderizar func(context.Context, ports.DetalleExpedienteRRHH)
}

func (r *renderizadorBorradorRRHHPrueba) RenderizarBorrador(ctx context.Context, tipo ports.TipoBorradorRRHH, d ports.DetalleExpedienteRRHH) ([]byte, error) {
	r.llamadas++
	r.tipo = tipo
	if r.alRenderizar != nil {
		r.alRenderizar(ctx, d)
	}
	return r.contenido, r.err
}

func detalleInformeRRHHPrueba() ports.DetalleExpedienteRRHH {
	d := detalleRRHHPrueba()
	for _, fase := range []domain.ClaveFase{"asignacion", "informe_juridico", "fiscalizacion", "nombramiento"} {
		anterior := d.Hitos[len(d.Hitos)-1]
		d.Hitos = append(d.Hitos, ports.HitoExpedienteRRHH{
			Secuencia: anterior.Secuencia + 1, VersionExpediente: anterior.VersionExpediente + 1,
			AccionClave: "actualizar", RealizadaEn: anterior.RealizadaEn.Add(time.Hour),
			FaseOrigen: anterior.FaseDestino, FaseDestino: fase,
			EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso,
		})
	}
	d.Hitos[6].AccionClave = "registrar_propuesta_formalizacion"
	d.Resumen.Version = 7
	d.Resumen.FaseClave = "nombramiento"
	d.Resumen.ActualizadoEn = d.Hitos[6].RealizadaEn
	return d
}

func peticionInformeRRHHPrueba() *http.Request {
	r := nuevaPeticionConsultaRRHHPrueba(RutaConsultaDetalleRRHH,
		`{"expediente_ref":"expediente:ct:0001","version_observada":7}`)
	r.Header.Set("Accept", AcceptInformeDefinitivoRRHH)
	return r
}

func TestConsultaRRHHBorradorDescargaTrasLectura(t *testing.T) {
	for _, caso := range []struct {
		tipo          ports.TipoBorradorRRHH
		accept        string
		nombreArchivo string
	}{
		{ports.BorradorInformeDefinitivo, AcceptInformeDefinitivoRRHH, "informe-definitivo-borrador.pdf"},
		{ports.BorradorResolucion, AcceptResolucionRRHH, "resolucion-borrador.pdf"},
		{ports.BorradorDiligencia, AcceptDiligenciaRRHH, "diligencia-borrador.pdf"},
	} {
		t.Run(string(caso.tipo), func(t *testing.T) {
			c := &consultorDetalleRRHHPrueba{detalle: detalleInformeRRHHPrueba()}
			r := &renderizadorBorradorRRHHPrueba{contenido: []byte("%PDF-1.4\nfixture transporte\n%%EOF\n")}
			r.alRenderizar = func(_ context.Context, d ports.DetalleExpedienteRRHH) {
				if c.llamadas != 1 || r.tipo != caso.tipo || d.Resumen.Version != 7 || d.Resumen.ExpedienteRef != c.solicitud.ExpedienteRef() {
					t.Fatal("render sin lectura previa, tipo o detalle distinto")
				}
				d.Hitos[0].AccionClave = "mutacion_del_doble"
			}
			h, err := NuevoManejadorConsultaDetalleRRHH(c, r)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			w.Header().Set("Set-Cookie", "prueba=1")
			peticion := peticionInformeRRHHPrueba()
			peticion.Header.Set("Accept", caso.accept)
			h.ServeHTTP(w, peticion)
			if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), r.contenido) || r.llamadas != 1 {
				t.Fatalf("descarga: estado=%d render=%d", w.Code, r.llamadas)
			}
			if c.detalle.Hitos[0].AccionClave != "alta" {
				t.Fatal("se compartió detalle mutable")
			}
			for k, v := range map[string]string{
				"Content-Type":        "application/pdf",
				"Content-Disposition": `attachment; filename="` + caso.nombreArchivo + `"`,
				"Content-Length":      strconv.Itoa(len(r.contenido)),
				"Cache-Control":       "no-store, no-transform", "Pragma": "no-cache",
				"Set-Cookie": "", "X-Content-Type-Options": "nosniff", "Location": "",
			} {
				if w.Header().Get(k) != v {
					t.Fatalf("cabecera %s inesperada", k)
				}
			}
		})
	}
}

func TestConsultaRRHHBorradorJSONYConstructor(t *testing.T) {
	c := &consultorDetalleRRHHPrueba{detalle: detalleInformeRRHHPrueba()}
	r := &renderizadorBorradorRRHHPrueba{}
	var nulo *renderizadorBorradorRRHHPrueba
	for _, lista := range [][]ports.RenderizadorBorradorRRHH{{nil}, {nulo}, {r, r}} {
		if _, err := NuevoManejadorConsultaDetalleRRHH(c, lista...); !errors.Is(err, ErrManejadorConsultaRRHHInvalido) {
			t.Fatal("constructor admitió renderizador inválido")
		}
	}
	var cuerpoJSON []byte
	for _, lista := range [][]ports.RenderizadorBorradorRRHH{nil, {r}} {
		h, err := NuevoManejadorConsultaDetalleRRHH(c, lista...)
		if err != nil {
			t.Fatal(err)
		}
		peticion := peticionInformeRRHHPrueba()
		peticion.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticion)
		if w.Code != http.StatusOK || r.llamadas != 0 || !strings.Contains(w.Body.String(), esquemaConsultaDetalleRRHH) {
			t.Fatalf("JSON alterado: %d", w.Code)
		}
		if cuerpoJSON != nil && !bytes.Equal(cuerpoJSON, w.Body.Bytes()) {
			t.Fatal("el renderer cambia la representación JSON")
		}
		cuerpoJSON = bytes.Clone(w.Body.Bytes())
		comprobarCabecerasConsultaRRHH(t, w)
	}
}

func TestConsultaRRHHBorradorRechazosAntesDeRender(t *testing.T) {
	for _, caso := range []struct {
		nombre            string
		mutar             func(*http.Request, *consultorDetalleRRHHPrueba)
		estado, consultas int
	}{
		{"sin permiso", func(_ *http.Request, c *consultorDetalleRRHHPrueba) { c.err = application.ErrConsultaRRHHNoObservable }, 404, 1},
		{"resolución sin permiso", func(r *http.Request, c *consultorDetalleRRHHPrueba) {
			r.Header.Set("Accept", AcceptResolucionRRHH)
			c.err = application.ErrConsultaRRHHNoObservable
		}, 404, 1},
		{"diligencia sin permiso", func(r *http.Request, c *consultorDetalleRRHHPrueba) {
			r.Header.Set("Accept", AcceptDiligenciaRRHH)
			c.err = application.ErrConsultaRRHHNoObservable
		}, 404, 1},
		{"detalle ajeno", func(_ *http.Request, c *consultorDetalleRRHHPrueba) {
			c.detalle.Resumen.ExpedienteRef = "expediente:ct:otro"
		}, 502, 1},
		{"PDF sin documento", func(r *http.Request, _ *consultorDetalleRRHHPrueba) { r.Header.Set("Accept", "application/pdf") }, 406, 0},
		{"otro documento", func(r *http.Request, _ *consultorDetalleRRHHPrueba) {
			r.Header.Set("Accept", "application/pdf; documento=resolucion")
		}, 406, 0},
		{"Accept duplicado", func(r *http.Request, _ *consultorDetalleRRHHPrueba) {
			r.Header.Add("Accept", AcceptInformeDefinitivoRRHH)
		}, 406, 0},
		{"dos documentos", func(r *http.Request, _ *consultorDetalleRRHHPrueba) {
			r.Header.Add("Accept", AcceptResolucionRRHH)
		}, 406, 0},
		{"cookie", func(r *http.Request, _ *consultorDetalleRRHHPrueba) { r.Header.Set("Cookie", "prueba=1") }, 400, 0},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			c := &consultorDetalleRRHHPrueba{detalle: detalleInformeRRHHPrueba()}
			r := &renderizadorBorradorRRHHPrueba{}
			h, err := NuevoManejadorConsultaDetalleRRHH(c, r)
			if err != nil {
				t.Fatal(err)
			}
			peticion := peticionInformeRRHHPrueba()
			caso.mutar(peticion, c)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticion)
			if w.Code != caso.estado || c.llamadas != caso.consultas || r.llamadas != 0 {
				t.Fatalf("estado/consulta/render=%d/%d/%d", w.Code, c.llamadas, r.llamadas)
			}
			comprobarCabecerasConsultaRRHH(t, w)
		})
	}
	cuadro := &consultorCuadroRRHHPrueba{pagina: paginaRRHHPrueba()}
	h, err := NuevoManejadorConsultaCuadroRRHH(cuadro)
	if err != nil {
		t.Fatal(err)
	}
	for _, accept := range []string{AcceptInformeDefinitivoRRHH, AcceptResolucionRRHH, AcceptDiligenciaRRHH} {
		peticion := nuevaPeticionConsultaRRHHPrueba(RutaConsultaCuadroRRHH, cuerpoCuadroRRHHPrueba())
		peticion.Header.Set("Accept", accept)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticion)
		if w.Code != http.StatusNotAcceptable || cuadro.llamadas != 0 {
			t.Fatal("PDF habilitado en cuadro")
		}
	}
}

func TestConsultaRRHHBorradorFallosSinPDFParcial(t *testing.T) {
	for _, caso := range []struct {
		nombre    string
		contenido []byte
		err       error
		estado    int
	}{
		{"precondición", nil, ports.ErrBorradorRRHHNoDisponible, 409},
		{"error render", []byte("%PDF-parcial"), errors.New("detalle interno no publicable"), 500},
		{"no PDF", []byte("no es PDF"), nil, 502},
		{"exceso tamaño", append([]byte("%PDF-"), make([]byte, MaximoPDFBorradorRRHHBytes)...), nil, 502},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			c := &consultorDetalleRRHHPrueba{detalle: detalleInformeRRHHPrueba()}
			r := &renderizadorBorradorRRHHPrueba{contenido: caso.contenido, err: caso.err}
			h, err := NuevoManejadorConsultaDetalleRRHH(c, r)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			peticion := peticionInformeRRHHPrueba()
			peticion.Header.Set("Accept", AcceptResolucionRRHH)
			h.ServeHTTP(w, peticion)
			if w.Code != caso.estado || c.llamadas != 1 || r.llamadas != 1 || strings.Contains(w.Body.String(), "%PDF-") || strings.Contains(w.Body.String(), "detalle interno") {
				t.Fatalf("error con éxito parcial: estado=%d", w.Code)
			}
			if caso.estado == 409 && !strings.Contains(w.Body.String(), `"codigo":"documento_no_disponible"`) {
				t.Fatal("código de precondición incorrecto")
			}
			if w.Header().Get("Content-Disposition") != "" {
				t.Fatal("attachment en error")
			}
			comprobarCabecerasConsultaRRHH(t, w)
		})
	}
}

func TestConsultaRRHHBorradorCancelacion(t *testing.T) {
	for _, duranteRender := range []bool{false, true} {
		ctx, cancelar := context.WithCancel(context.Background())
		c := &consultorDetalleRRHHPrueba{detalle: detalleInformeRRHHPrueba()}
		r := &renderizadorBorradorRRHHPrueba{contenido: []byte("%PDF-1.4\n%%EOF\n")}
		if duranteRender {
			r.alRenderizar = func(context.Context, ports.DetalleExpedienteRRHH) { cancelar() }
		} else {
			c.alConsultar = func(context.Context) { cancelar() }
		}
		h, err := NuevoManejadorConsultaDetalleRRHH(c, r)
		if err != nil {
			cancelar()
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		peticion := peticionInformeRRHHPrueba().WithContext(ctx)
		peticion.Header.Set("Accept", AcceptResolucionRRHH)
		h.ServeHTTP(w, peticion)
		cancelar()
		if w.Code != http.StatusRequestTimeout || c.llamadas != 1 || (!duranteRender && r.llamadas != 0) || strings.Contains(w.Body.String(), "%PDF-") {
			t.Fatalf("cancelación: estado/consulta/render=%d/%d/%d", w.Code, c.llamadas, r.llamadas)
		}
		comprobarCabecerasConsultaRRHH(t, w)
	}
}
