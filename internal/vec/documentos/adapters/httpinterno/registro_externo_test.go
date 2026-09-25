package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type servicioRegistroPrueba struct {
	recibido ports.AltaExterna
	llamadas int
	err      error
	alterar  func(*domain.Documento)
}

func (s *servicioRegistroPrueba) RegistrarExternoAutorizado(_ context.Context, a ports.AltaExterna, autorizador ports.AutorizadorRegistroExterno) (domain.Documento, error) {
	s.llamadas++
	s.recibido = a
	if autorizador == nil || s.err != nil {
		return domain.Documento{}, s.err
	}
	d := documentoPrueba()
	d.ID, d.ExpedienteRef, d.TipoRef, d.ModuloID = a.ID, a.ExpedienteRef, a.TipoRef, a.ModuloID
	d.ObjetoRef, d.ObjetoVersion, d.MIME, d.Tamano = "", "", "", 0
	d.HuellaSHA256, d.Custodia, d.CustodiaExternaRef = a.Custodia.HuellaSHA256, domain.CustodiaExterna, a.Custodia
	d.CreadoEn = time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	if s.alterar != nil {
		s.alterar(&d)
	}
	return d, nil
}

type autorizadorRegistroPrueba struct{}

func (autorizadorRegistroPrueba) AutorizarRegistroExterno(context.Context, []byte, string, string) (ports.AutorizacionV3, error) {
	return ports.AutorizacionV3{}, nil
}

// politicasRegistroPrueba solo conoce un tipo; cualquier otro no está catalogado.
type politicasRegistroPrueba struct{ t *testing.T }

func (p politicasRegistroPrueba) SolicitudPara(tipo, expediente string) (vecports.SolicitudPoliticaConservacionDocumental, error) {
	if tipo != "contratacion_temporal.formalizacion.titulacion.v1" {
		return vecports.SolicitudPoliticaConservacionDocumental{}, errors.New("sin politica")
	}
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	desde := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s, err := vecports.NuevaSolicitudPoliticaConservacionDocumental(ref("a"), ref("b"), ref("c"), expediente,
		ref("d"), 1, []byte(strings.Repeat("e", 32)), ref("f"), desde, desde.AddDate(5, 0, 0))
	if err != nil {
		p.t.Fatal(err)
	}
	return s, nil
}

func rutaRegistroPrueba(t *testing.T, s *servicioRegistroPrueba) http.Handler {
	t.Helper()
	ruta, err := NuevaRutaRegistroExterno(ConfiguracionRegistroExterno{
		Servicio: s, Autoridad: autorizadorRegistroPrueba{}, Politicas: politicasRegistroPrueba{t},
		Admitidos: []RegistroExternoAdmitido{{PrefijoTipo: "contratacion_temporal.formalizacion.", ModuloID: "contratacion_temporal", CustodioID: "registro_entrada"}},
	})
	if err != nil || ruta.Ruta != RutaRegistroExterno {
		t.Fatalf("ruta: %v", err)
	}
	return ruta.Manejador
}

const cuerpoRegistroPrueba = `{"clave_idempotencia":"clave:0b0c6f8e-6a51-4a4b-9d51-3a1f2e5c7d90","expediente_ref":"ref:2222222222222222222222222222222222222222222222222222222222222222","tipo":"contratacion_temporal.formalizacion.titulacion.v1","referencia":"REGE-2026-000123","huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`

func TestRegistroExternoDerivaModuloCustodioEIdentificadorEnServidor(t *testing.T) {
	s := &servicioRegistroPrueba{}
	h := rutaRegistroPrueba(t, s)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, solicitud(RutaRegistroExterno, cuerpoRegistroPrueba))
	cuerpo := w.Body.String()
	if w.Code != http.StatusCreated || s.llamadas != 1 || s.recibido.ModuloID != "contratacion_temporal" ||
		s.recibido.Custodia.CustodioID != "registro_entrada" || s.recibido.Custodia.Referencia != "REGE-2026-000123" ||
		s.recibido.Version != 1 || !domain.ReferenciaOpacaValida(s.recibido.ID) || s.recibido.TipoRef != "ref:"+strings.Repeat("c", 64) ||
		!strings.Contains(cuerpo, `"numero_vec":"VEC-2026-1"`) || !strings.Contains(cuerpo, `"tipo":"contratacion_temporal.formalizacion.titulacion.v1"`) ||
		strings.Contains(cuerpo, "REGE-2026") || strings.Contains(cuerpo, "registro_entrada") || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("registro=%d %s %+v", w.Code, cuerpo, s.recibido)
	}
	primero := s.recibido.ID
	h.ServeHTTP(httptest.NewRecorder(), solicitud(RutaRegistroExterno, cuerpoRegistroPrueba))
	if s.recibido.ID != primero {
		t.Fatal("el reintento con la misma clave debe nombrar el mismo documento")
	}
}

func TestRegistroExternoRechazaEntradasNoAdmitidasSinLlamarAlServicio(t *testing.T) {
	casos := map[string]string{
		"tipo de otro modulo":  strings.Replace(cuerpoRegistroPrueba, "contratacion_temporal.formalizacion.titulacion.v1", "dietas.justificante.v1", 1),
		"tipo sin politica":    strings.Replace(cuerpoRegistroPrueba, "titulacion.v1", "otro.v1", 1),
		"solo el prefijo":      strings.Replace(cuerpoRegistroPrueba, "contratacion_temporal.formalizacion.titulacion.v1", "contratacion_temporal.formalizacion.", 1),
		"referencia con ruta":  strings.Replace(cuerpoRegistroPrueba, "REGE-2026-000123", "../etc", 1),
		"referencia comodin":   strings.Replace(cuerpoRegistroPrueba, "REGE-2026-000123", "REGE*", 1),
		"huella corta":         strings.Replace(cuerpoRegistroPrueba, `"aaaa`, `"aa`, 1),
		"huella nula":          strings.Replace(cuerpoRegistroPrueba, strings.Repeat("a", 64), strings.Repeat("0", 64), 1),
		"clave no opaca":       strings.Replace(cuerpoRegistroPrueba, "clave:0b0c6f8e", "DNI:0b0c6f8e", 1),
		"modulo desde cliente": strings.Replace(cuerpoRegistroPrueba, `"tipo"`, `"modulo_id":"dietas","tipo"`, 1),
		"custodio del cliente": strings.Replace(cuerpoRegistroPrueba, `"tipo"`, `"custodio_id":"x","tipo"`, 1),
	}
	for nombre, cuerpo := range casos {
		s := &servicioRegistroPrueba{}
		w := httptest.NewRecorder()
		rutaRegistroPrueba(t, s).ServeHTTP(w, solicitud(RutaRegistroExterno, cuerpo))
		if w.Code != http.StatusUnprocessableEntity || s.llamadas != 0 {
			t.Errorf("%s: %d llamadas=%d", nombre, w.Code, s.llamadas)
		}
	}
}

func TestRegistroExternoTraduceErroresSinFiltrarDetalles(t *testing.T) {
	casos := map[error]int{
		ports.ErrAccesoDenegado:        http.StatusForbidden,
		ports.ErrCapacidadNoDisponible: http.StatusServiceUnavailable,
		ports.ErrConflicto:             http.StatusConflict,
	}
	for causa, estado := range casos {
		s := &servicioRegistroPrueba{err: causa}
		w := httptest.NewRecorder()
		rutaRegistroPrueba(t, s).ServeHTTP(w, solicitud(RutaRegistroExterno, cuerpoRegistroPrueba))
		if w.Code != estado || strings.Contains(w.Body.String(), causa.Error()) {
			t.Errorf("%v: %d %s", causa, w.Code, w.Body.String())
		}
	}
	s := &servicioRegistroPrueba{alterar: func(d *domain.Documento) { d.CustodiaExternaRef.Referencia = "otra" }}
	w := httptest.NewRecorder()
	rutaRegistroPrueba(t, s).ServeHTTP(w, solicitud(RutaRegistroExterno, cuerpoRegistroPrueba))
	if w.Code != http.StatusBadGateway {
		t.Fatalf("confirmación discordante aceptada: %d", w.Code)
	}
	w = httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, RutaRegistroExterno, nil)
	rutaRegistroPrueba(t, &servicioRegistroPrueba{}).ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET=%d", w.Code)
	}
}

func TestRegistroExternoExigeConfiguracionCompletaYPrefijosDisjuntos(t *testing.T) {
	base := ConfiguracionRegistroExterno{Servicio: &servicioRegistroPrueba{}, Autoridad: autorizadorRegistroPrueba{}, Politicas: politicasRegistroPrueba{t}}
	for nombre, admitidos := range map[string][]RegistroExternoAdmitido{
		"sin admitidos":     nil,
		"prefijo sin punto": {{PrefijoTipo: "contratacion_temporal", ModuloID: "contratacion_temporal", CustodioID: "registro"}},
		"custodio vacio":    {{PrefijoTipo: "contratacion_temporal.", ModuloID: "contratacion_temporal"}},
		"solapados": {{PrefijoTipo: "contratacion_temporal.", ModuloID: "contratacion_temporal", CustodioID: "registro"},
			{PrefijoTipo: "contratacion_temporal.formalizacion.", ModuloID: "contratacion_temporal", CustodioID: "registro"}},
	} {
		c := base
		c.Admitidos = admitidos
		if _, err := NuevaRutaRegistroExterno(c); !errors.Is(err, ErrManejadorInvalido) {
			t.Errorf("%s aceptada", nombre)
		}
	}
}
