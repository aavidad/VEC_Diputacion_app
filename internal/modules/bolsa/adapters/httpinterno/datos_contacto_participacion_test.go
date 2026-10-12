package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type preparadorDatosContactoPrueba struct {
	solicitud      puertosbolsa.SolicitudConsultarDatosContactoParticipacion
	err            error
	ultimaEntrada  EntradaRegistrarDatosContactoParticipacion
	ultimaConsulta [2]string
	ultimoCompleto bool
}

func (p *preparadorDatosContactoPrueba) PrepararSolicitudRegistrarDatosContacto(_ context.Context, e EntradaRegistrarDatosContactoParticipacion) (puertosbolsa.SolicitudRegistrarDatosContactoParticipacion, error) {
	p.ultimaEntrada = e
	if p.err != nil {
		return puertosbolsa.SolicitudRegistrarDatosContactoParticipacion{}, p.err
	}
	return puertosbolsa.SolicitudRegistrarDatosContactoParticipacion{BolsaRef: e.BolsaRef, ParticipacionRef: e.ParticipacionRef, Datos: e.Datos, Motivo: e.Motivo, ClaveIdempotencia: e.ClaveIdempotencia}, nil
}

func (p *preparadorDatosContactoPrueba) PrepararSolicitudConsultarDatosContacto(_ context.Context, bolsa, participacion string, completo bool) (puertosbolsa.SolicitudConsultarDatosContactoParticipacion, error) {
	p.ultimaConsulta, p.ultimoCompleto = [2]string{bolsa, participacion}, completo
	if p.err != nil {
		return p.solicitud, p.err
	}
	return puertosbolsa.SolicitudConsultarDatosContactoParticipacion{BolsaRef: bolsa, ParticipacionRef: participacion, Completo: completo}, nil
}

type operadorDatosContactoPrueba struct {
	registro  puertosbolsa.RegistroDatosContactoParticipacion
	leidos    puertosbolsa.DatosContactoParticipacionLeidos
	errRegist error
	errCons   error
	registros int
}

func (o *operadorDatosContactoPrueba) Registrar(context.Context, puertosbolsa.SolicitudRegistrarDatosContactoParticipacion) (puertosbolsa.RegistroDatosContactoParticipacion, error) {
	o.registros++
	return o.registro, o.errRegist
}

func (o *operadorDatosContactoPrueba) Consultar(context.Context, puertosbolsa.SolicitudConsultarDatosContactoParticipacion) (puertosbolsa.DatosContactoParticipacionLeidos, error) {
	return o.leidos, o.errCons
}

func peticionDatosContacto(metodo, ruta, cuerpo string) *http.Request {
	var r *http.Request
	if cuerpo == "" {
		r = httptest.NewRequest(metodo, ruta, nil)
	} else {
		r = httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "b4-alta-0001")
	}
	r.Header.Set("Accept", "application/json")
	return r
}

const rutaDatosContactoPrueba = RutaBolsasGestion + "/bolsa:b4/candidatos/participacion:b4/datos-contacto"

func TestHandlerDatosContactoRegistraYDevuelveRecibo(t *testing.T) {
	ahora := time.Date(2026, 9, 22, 15, 0, 0, 0, time.UTC)
	prep := &preparadorDatosContactoPrueba{}
	op := &operadorDatosContactoPrueba{registro: puertosbolsa.RegistroDatosContactoParticipacion{ReciboRef: "recibo:datos-contacto:0001", ParticipacionRef: "participacion:b4", Version: 1, RegistradaEn: ahora}}
	h, err := NuevoHandlerDatosContactoParticipacion(prep, op)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, peticionDatosContacto(http.MethodPost, rutaDatosContactoPrueba, `{"correo":"c@dipgra.es","telefono_1":"600123456","telefono_2":"","motivo":"Alta comunicada"}`))
	if rec.Code != http.StatusCreated || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("estado=%d cabeceras=%v", rec.Code, rec.Header())
	}
	var salida struct {
		Data struct {
			ReciboRef string `json:"recibo_ref"`
			Version   uint64 `json:"version"`
		} `json:"data"`
	}
	if json.Unmarshal(rec.Body.Bytes(), &salida) != nil || salida.Data.ReciboRef != "recibo:datos-contacto:0001" || salida.Data.Version != 1 {
		t.Fatalf("cuerpo=%s", rec.Body.String())
	}
	if prep.ultimaEntrada.Datos.Correo != "c@dipgra.es" || prep.ultimaEntrada.ClaveIdempotencia != "b4-alta-0001" || prep.ultimaEntrada.Datos.ParticipacionRef != "participacion:b4" {
		t.Fatalf("entrada=%+v", prep.ultimaEntrada)
	}
	// Repetición: el caso de uso decide; el adaptador responde 200.
	op.registro.Reutilizada = true
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, peticionDatosContacto(http.MethodPost, rutaDatosContactoPrueba, `{"correo":"c@dipgra.es","telefono_1":"600123456","telefono_2":"","motivo":"Alta comunicada"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("repetición: %d", rec.Code)
	}
}

func TestHandlerDatosContactoConsultaEnmascaraSalvoVerCompleto(t *testing.T) {
	ahora := time.Date(2026, 9, 22, 15, 0, 0, 0, time.UTC)
	datos := dominiobolsa.DatosContactoParticipacion{ParticipacionRef: "participacion:b4", Correo: "candidata@dipgra.es", Telefono1: "600123456"}
	prep := &preparadorDatosContactoPrueba{}
	op := &operadorDatosContactoPrueba{leidos: puertosbolsa.DatosContactoParticipacionLeidos{ParticipacionRef: "participacion:b4", Version: 2, RegistradaEn: ahora, Datos: datos, Enmascarados: datos.Enmascarados(), AuditoriaRef: "aud_v3_0123456789abcdef0123456789abcdef", DecisionRef: "decision:prueba"}}
	h, _ := NuevoHandlerDatosContactoParticipacion(prep, op)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, peticionDatosContacto(http.MethodGet, rutaDatosContactoPrueba, ""))
	if rec.Code != http.StatusOK || prep.ultimoCompleto || strings.Contains(rec.Body.String(), "candidata@dipgra.es") || strings.Contains(rec.Body.String(), "600123456") || rec.Header().Get("X-Audit-Ref") == "" {
		t.Fatalf("la lectura ordinaria no debe exponer el claro: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "c***@dipgra.es") || !strings.Contains(rec.Body.String(), "***3456") {
		t.Fatalf("falta el enmascarado: %s", rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, peticionDatosContacto(http.MethodGet, rutaDatosContactoPrueba+"?ver=completo", ""))
	if rec.Code != http.StatusOK || !prep.ultimoCompleto || !strings.Contains(rec.Body.String(), "candidata@dipgra.es") || rec.Header().Get("X-Audit-Ref") != "aud_v3_0123456789abcdef0123456789abcdef" {
		t.Fatalf("ver=completo: %d %s %v", rec.Code, rec.Body.String(), rec.Header())
	}
	// Un claro sin acuse de consumo no se entrega.
	op.leidos.AuditoriaRef = ""
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, peticionDatosContacto(http.MethodGet, rutaDatosContactoPrueba+"?ver=completo", ""))
	if rec.Code != http.StatusServiceUnavailable || strings.Contains(rec.Body.String(), "candidata@dipgra.es") {
		t.Fatalf("completo sin acuse: %d %s", rec.Code, rec.Body.String())
	}
	if prep.ultimaConsulta != [2]string{"bolsa:b4", "participacion:b4"} {
		t.Fatalf("consulta=%v", prep.ultimaConsulta)
	}
}

func TestHandlerDatosContactoRechazaRutasMetodosYErrores(t *testing.T) {
	prep := &preparadorDatosContactoPrueba{}
	op := &operadorDatosContactoPrueba{}
	h, _ := NuevoHandlerDatosContactoParticipacion(prep, op)
	casos := []struct {
		nombre, metodo, ruta, cuerpo string
		esperado                     int
	}{
		{"ruta ajena", http.MethodGet, RutaBolsasGestion + "/bolsa:b4/candidatos/participacion:b4/otra", "", http.StatusNotFound},
		{"consulta desconocida", http.MethodGet, rutaDatosContactoPrueba + "?ver=todo", "", http.StatusNotFound},
		{"método", http.MethodDelete, rutaDatosContactoPrueba, "", http.StatusMethodNotAllowed},
		{"cuerpo sin motivo", http.MethodPost, rutaDatosContactoPrueba, `{"correo":"c@dipgra.es","motivo":""}`, http.StatusBadRequest},
		{"campo desconocido", http.MethodPost, rutaDatosContactoPrueba, `{"correo":"c@dipgra.es","motivo":"x","otro":1}`, http.StatusBadRequest},
	}
	for _, caso := range casos {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, peticionDatosContacto(caso.metodo, caso.ruta, caso.cuerpo))
		if rec.Code != caso.esperado {
			t.Errorf("%s: %d, se esperaba %d", caso.nombre, rec.Code, caso.esperado)
		}
	}
	if op.registros != 0 {
		t.Fatalf("ninguna solicitud inválida debe llegar al caso de uso: %d", op.registros)
	}
	for _, caso := range []struct {
		err      error
		esperado int
	}{
		{dominiovec.ErrAutorizacionDenegada, http.StatusForbidden},
		{dominiobolsa.ErrDatosContactoParticipacionInvalidos, http.StatusConflict},
		{puertosbolsa.ErrDatosContactoParticipacionNoEncontrados, http.StatusNotFound},
		{puertosbolsa.ErrDatosContactoParticipacionNoDisponibles, http.StatusServiceUnavailable},
	} {
		op.errRegist = caso.err
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, peticionDatosContacto(http.MethodPost, rutaDatosContactoPrueba, `{"correo":"c@dipgra.es","motivo":"Alta"}`))
		if rec.Code != caso.esperado {
			t.Errorf("error %v: %d, se esperaba %d", caso.err, rec.Code, caso.esperado)
		}
		if strings.Contains(rec.Body.String(), "dipgra") {
			t.Errorf("la respuesta de error no debe traer datos: %s", rec.Body.String())
		}
	}
}

func TestHandlerContactoVacioExigeAsientoYNoOculta404(t *testing.T) {
	op := &operadorDatosContactoPrueba{leidos: puertosbolsa.DatosContactoParticipacionLeidos{ParticipacionRef: "participacion:b4", SinContacto: true, AuditoriaRef: "aud_v3_lc_0123456789abcdef0123456789abcdef"}}
	h, _ := NuevoHandlerDatosContactoParticipacion(&preparadorDatosContactoPrueba{}, op)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, peticionDatosContacto(http.MethodGet, rutaDatosContactoPrueba, ""))
	if rec.Code != http.StatusOK || rec.Header().Get("X-Audit-Ref") == "" || !strings.Contains(rec.Body.String(), `"sin_contacto":true`) || !strings.Contains(rec.Body.String(), `"registrada_en":null`) {
		t.Fatalf("vacío no explícito/auditado: %d %s", rec.Code, rec.Body.String())
	}
	op.leidos.AuditoriaRef = ""
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, peticionDatosContacto(http.MethodGet, rutaDatosContactoPrueba, ""))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatal("vacío sin auditoría aceptado")
	}
	op.errCons = puertosbolsa.ErrDatosContactoParticipacionNoEncontrados
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, peticionDatosContacto(http.MethodGet, rutaDatosContactoPrueba, ""))
	if rec.Code != http.StatusNotFound {
		t.Fatal("recurso inexistente ocultado como vacío")
	}
}

func TestHandlerDatosContactoAuditaDenegacionEnPreparacion(t *testing.T) {
	ctx, err := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(time.Now().UTC(), "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	q := puertosbolsa.SolicitudConsultarDatosContactoParticipacion{ContextoActor: actor.Contexto, ResultadoContexto: actor, Vinculo: vinculo, BolsaRef: "bolsa:b4", ParticipacionRef: "participacion:b4", Correlacion: correlacion,
		MotivoAutorizacion: dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("b", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"}}
	r := &registradorDocumentalesHTTPPrueba{t: t}
	o := &operadorDatosContactoPrueba{}
	auditado, err := application.NuevosDatosContactoRRHHAuditados(o, r, "vec-bolsa-prueba")
	if err != nil {
		t.Fatal(err)
	}
	h, err := NuevoHandlerDatosContactoParticipacion(&preparadorDatosContactoPrueba{solicitud: q, err: dominiovec.ErrAutorizacionDenegada}, auditado)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionDatosContacto(http.MethodGet, rutaDatosContactoPrueba, "").WithContext(ctx))
	if w.Code != http.StatusForbidden || len(r.ordenes) != 1 || w.Header().Get("X-Audit-Ref") != "aud_documental_http_sintetica" || r.ordenes[0].Datos.Accion != puertosbolsa.AccionConsultarOrigenDatosContactoParticipacion {
		t.Fatalf("fallo no auditado nominal: %d %s ordenes=%d", w.Code, w.Body.String(), len(r.ordenes))
	}
}
