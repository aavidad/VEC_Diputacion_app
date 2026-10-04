package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

type actorFichaPropiaHTTP struct {
	actor    core.ContextoActor
	err      error
	llamadas *int
}

func (a actorFichaPropiaHTTP) ResolverActorFichaPropia(context.Context) (core.ContextoActor, error) {
	if a.llamadas != nil {
		*a.llamadas++
	}
	return a.actor, a.err
}

type consultaFichaPropiaHTTP struct {
	err       error
	solicitud personaldomain.SolicitudFichaPropia
	llamadas  int
}

func (c *consultaFichaPropiaHTTP) Consultar(_ context.Context, s personaldomain.SolicitudFichaPropia) (personalports.ResultadoFichaPropia, error) {
	c.llamadas++
	c.solicitud = s
	if c.err != nil {
		return personalports.ResultadoFichaPropia{}, c.err
	}
	return personalports.ResultadoFichaPropia{
		Ficha:     personaldomain.FichaPropia{Corte: s.Corte, Relaciones: []personaldomain.RelacionFichaPropia{{Inicio: "2026-01-01", Estado: "vigente", Regimen: "Laboral fijo"}}, Servicios: []personaldomain.ServicioFichaPropia{}},
		Evidencia: personalports.EvidenciaRegistroEmpleadoB2{ReciboRef: "fichapropia:abc", DecisionRef: "dec_x", EfectoRef: "emp_" + strings.Repeat("a", 24), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "auditoria:x", ConsultadaEn: time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)},
	}, nil
}

type registroFichaPropiaHTTP struct {
	intentos       []personalports.IntentoFichaPropia
	contextos      []context.Context
	verificaciones int
	errVerificar   error
	err            error
}

func (r *registroFichaPropiaHTTP) VerificarRegistroFichaPropia(context.Context) error {
	r.verificaciones++
	return r.errVerificar
}

func (r *registroFichaPropiaHTTP) RegistrarIntentoFichaPropia(ctx context.Context, d personalports.IntentoFichaPropia) error {
	r.intentos = append(r.intentos, d)
	r.contextos = append(r.contextos, ctx)
	return r.err
}

func actorFichaPropiaPruebaHTTP(t *testing.T) core.ContextoActor {
	t.Helper()
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}
	instantanea := core.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := core.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return actor
}

func manejadorFichaPropiaPrueba(t *testing.T, consulta *consultaFichaPropiaHTTP, registro *registroFichaPropiaHTTP) *ManejadorFichaPropia {
	t.Helper()
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	// 23:30 UTC del 24 es ya día 25 en Madrid.
	ahora := func() time.Time { return time.Date(2026, 9, 24, 23, 30, 0, 500, time.UTC) }
	m, err := NuevoManejadorFichaPropia(actorFichaPropiaHTTP{actor: actorFichaPropiaPruebaHTTP(t)}, consulta, registro, ahora, madrid)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestFichaPropiaHTTPSirveSoloDatosVisibles(t *testing.T) {
	consulta := &consultaFichaPropiaHTTP{}
	registro := &registroFichaPropiaHTTP{}
	m := manejadorFichaPropiaPrueba(t, consulta, registro)
	resoluciones := 0
	m.actor = actorFichaPropiaHTTP{actor: actorFichaPropiaPruebaHTTP(t), llamadas: &resoluciones}
	w := httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil))
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/json; charset=utf-8" || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("respuesta %d %v", w.Code, w.Header())
	}
	if resoluciones != 1 || len(registro.intentos) != 0 {
		t.Fatalf("consulta duplicó resolución o registro HTTP: resoluciones=%d intentos=%+v", resoluciones, registro.intentos)
	}
	cuerpo := w.Body.String()
	for _, prohibido := range []string{"emp_", "per_", "dec_x", "auditoria", "consumo_huella"} {
		if strings.Contains(cuerpo, prohibido) {
			t.Fatalf("la respuesta expone %q: %s", prohibido, cuerpo)
		}
	}
	var sobre struct {
		Data struct {
			Ficha        personaldomain.FichaPropia `json:"ficha"`
			ReciboRef    string                     `json:"recibo_ref"`
			ConsultadaEn string                     `json:"consultada_en"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &sobre); err != nil || sobre.Data.ReciboRef != "fichapropia:abc" || sobre.Data.ConsultadaEn != "2026-09-25T08:00:00.000000Z" || sobre.Data.Ficha.Relaciones[0].Regimen != "Laboral fijo" {
		t.Fatalf("sobre inesperado: %v %s", err, cuerpo)
	}
	if consulta.solicitud.Corte.VigenteEn != "2026-09-25" || !consulta.solicitud.Corte.ConocidoEn.Equal(time.Date(2026, 9, 24, 23, 29, 59, 0, time.UTC)) {
		t.Fatalf("corte inesperado: %+v", consulta.solicitud.Corte)
	}
}

func TestFichaPropiaHTTPRegistraSoloRechazosPrevios(t *testing.T) {
	casos := map[string]struct {
		peticion func() *http.Request
		err      error
		estado   int
		codigo   string
		consulta bool
	}{
		"metodo": {func() *http.Request { return httptest.NewRequest(http.MethodPost, RutaFichaPropia, nil) }, nil, 405, "metodo_no_permitido", false},
		"parametros": {func() *http.Request {
			return httptest.NewRequest(http.MethodGet, RutaFichaPropia+"?empleado=emp_x", nil)
		}, nil, 404, "no_encontrada", false},
		"cookie": {func() *http.Request {
			r := httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil)
			r.Header.Set("Cookie", "a=b")
			return r
		}, nil, 400, "peticion_invalida", false},
		"sin_empleado": {func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil) }, personaldomain.ErrFichaPropiaSinEmpleado, 403, "sin_empleado", true},
		"ambiguo":      {func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil) }, personaldomain.ErrFichaPropiaAmbigua, 403, "empleado_ambiguo", true},
		"denegado":     {func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil) }, personaldomain.ErrFichaPropiaDenegada, 403, "acceso_denegado", true},
		"caido":        {func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil) }, errors.New("detalle"), 503, "no_disponible", true},
		"excede":       {func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil) }, personaldomain.ErrFichaPropiaExcedeLimite, 422, "excede_limite", true},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			consulta := &consultaFichaPropiaHTTP{err: caso.err}
			registro := &registroFichaPropiaHTTP{}
			w := httptest.NewRecorder()
			manejadorFichaPropiaPrueba(t, consulta, registro).ServeHTTP(w, caso.peticion())
			intentosEsperados := 1
			if caso.consulta {
				intentosEsperados = 0 // La aplicación es responsable de sus errores.
			}
			if w.Code != caso.estado || !strings.Contains(w.Body.String(), `"error":"`+caso.codigo+`"`) || len(registro.intentos) != intentosEsperados || (consulta.llamadas == 1) != caso.consulta {
				t.Fatalf("estado %d cuerpo %s intentos %+v", w.Code, w.Body.String(), registro.intentos)
			}
			if !caso.consulta && registro.intentos[0].Motivo != "entrada_invalida" {
				t.Fatalf("motivo de transporte no minimizado: %+v", registro.intentos)
			}
		})
	}
}

func TestFichaPropiaHTTPSinAuditoriaNoRevelaMotivo(t *testing.T) {
	consulta := &consultaFichaPropiaHTTP{}
	w := httptest.NewRecorder()
	manejadorFichaPropiaPrueba(t, consulta, &registroFichaPropiaHTTP{err: errors.New("caída")}).ServeHTTP(w, httptest.NewRequest(http.MethodPost, RutaFichaPropia, nil))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "metodo_no_permitido") {
		t.Fatalf("sin auditoría se reveló el motivo: %d %s", w.Code, w.Body.String())
	}
}

func TestFichaPropiaHTTPFechaReferenciaNoCambiaIdentidadNiConocimiento(t *testing.T) {
	for _, fecha := range []string{"2020-02-29", "2026-09-25", "2027-01-01"} {
		t.Run(fecha, func(t *testing.T) {
			consulta := &consultaFichaPropiaHTTP{}
			w := httptest.NewRecorder()
			manejadorFichaPropiaPrueba(t, consulta, &registroFichaPropiaHTTP{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaFichaPropia+"?fecha_referencia="+fecha, nil))
			if w.Code != http.StatusOK || consulta.solicitud.Corte.VigenteEn.Texto() != fecha || !consulta.solicitud.Corte.ConocidoEn.Equal(time.Date(2026, 9, 24, 23, 29, 59, 0, time.UTC)) {
				t.Fatalf("estado %d corte %+v", w.Code, consulta.solicitud.Corte)
			}
			if consulta.solicitud.Actor.Principal.ID != actorFichaPropiaPruebaHTTP(t).Principal.ID {
				t.Fatal("la fecha alteró el actor efectivo")
			}
		})
	}
}

func TestFichaPropiaHTTPRechazaVariantesDeCorteAntesDeConsultar(t *testing.T) {
	for _, caso := range []struct {
		query  string
		estado int
	}{
		{"?", 404}, {"?fecha_referencia=", 400}, {"?fecha_referencia=2025-02-29", 400},
		{"?fecha_referencia=0000-01-01", 400}, {"?fecha_referencia=2026-9-25", 400},
		{"?fecha_referencia=2026-09-25T00:00:00Z", 400}, {"?fecha_referencia=%32%30%32%36-09-25", 400},
		{"?fecha_referencia=2026-09-25&fecha_referencia=2020-01-01", 404},
		{"?fecha_referencia=2026-09-25&empleado=emp_x", 404},
		{"?conocido_en=2020-01-01", 404}, {"?fecha_referencia=2026-09-25&conocido_en=2020-01-01", 404},
	} {
		t.Run(caso.query, func(t *testing.T) {
			consulta := &consultaFichaPropiaHTTP{}
			registro := &registroFichaPropiaHTTP{}
			w := httptest.NewRecorder()
			manejadorFichaPropiaPrueba(t, consulta, registro).ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaFichaPropia+caso.query, nil))
			if w.Code != caso.estado || consulta.llamadas != 0 || len(registro.intentos) != 1 {
				t.Fatalf("estado %d consultas %d auditoría %+v", w.Code, consulta.llamadas, registro.intentos)
			}
		})
	}
}

func TestFichaPropiaHTTPContextoAusenteNoAtribuyeIntento(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		actor  actorFichaPropiaHTTP
	}{
		{"sin_contexto", actorFichaPropiaHTTP{err: errors.New("contexto ausente")}},
		{"actor_invalido", actorFichaPropiaHTTP{}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			consulta := &consultaFichaPropiaHTTP{}
			registro := &registroFichaPropiaHTTP{}
			m := manejadorFichaPropiaPrueba(t, consulta, registro)
			m.actor = caso.actor
			w := httptest.NewRecorder()
			m.ServeHTTP(w, httptest.NewRequest(http.MethodPost, RutaFichaPropia+"?persona=per_secreto", nil))
			if w.Code != http.StatusServiceUnavailable || w.Body.String() != `{"error":"no_disponible"}` || len(registro.intentos) != 0 || consulta.llamadas != 0 {
				t.Fatalf("rechazo sin identidad: %d %s intentos=%+v consultas=%d", w.Code, w.Body.String(), registro.intentos, consulta.llamadas)
			}
		})
	}
}

func TestFichaPropiaHTTPPreflightFallidoRegistraErrorSinConsultar(t *testing.T) {
	consulta := &consultaFichaPropiaHTTP{}
	registro := &registroFichaPropiaHTTP{errVerificar: errors.New("destino caído")}
	w := httptest.NewRecorder()
	manejadorFichaPropiaPrueba(t, consulta, registro).ServeHTTP(w, httptest.NewRequest(http.MethodPost, RutaFichaPropia, nil))
	if w.Code != http.StatusServiceUnavailable || w.Body.String() != `{"error":"no_disponible"}` || len(registro.intentos) != 1 || registro.intentos[0].Motivo != "no_disponible" || consulta.llamadas != 0 || registro.verificaciones != 1 {
		t.Fatalf("preflight: %d %s intentos=%+v consultas=%d", w.Code, w.Body.String(), registro.intentos, consulta.llamadas)
	}
}

type contextoFichaPropiaHTTPPrueba struct{}

func TestFichaPropiaHTTPConservaContextoOriginalEnRechazoCancelado(t *testing.T) {
	consulta := &consultaFichaPropiaHTTP{}
	registro := &registroFichaPropiaHTTP{}
	ctx, cancelar := context.WithCancel(context.WithValue(context.Background(), contextoFichaPropiaHTTPPrueba{}, "correlacion_servidor"))
	cancelar()
	w := httptest.NewRecorder()
	manejadorFichaPropiaPrueba(t, consulta, registro).ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaFichaPropia+"?empleado=emp_secreto", nil).WithContext(ctx))
	if w.Code != http.StatusNotFound || consulta.llamadas != 0 || len(registro.intentos) != 1 || registro.intentos[0].Motivo != "entrada_invalida" {
		t.Fatalf("rechazo: %d consultas=%d intentos=%+v", w.Code, consulta.llamadas, registro.intentos)
	}
	if registro.contextos[0].Value(contextoFichaPropiaHTTPPrueba{}) != "correlacion_servidor" {
		t.Fatal("el append perdió el contexto servidor original")
	}
}

func TestFichaPropiaHTTPCancelacionDeConsultaNoDuplicaRegistro(t *testing.T) {
	consulta := &consultaFichaPropiaHTTP{err: context.Canceled}
	registro := &registroFichaPropiaHTTP{}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	w := httptest.NewRecorder()
	manejadorFichaPropiaPrueba(t, consulta, registro).ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil).WithContext(ctx))
	if consulta.llamadas != 1 || len(registro.intentos) != 0 || w.Body.Len() != 0 {
		t.Fatalf("cancelación: consultas=%d intentos=%+v cuerpo=%s", consulta.llamadas, registro.intentos, w.Body.String())
	}
}

func TestFichaPropiaHTTPPreflightCanceladoConIdentidadConservaIntento(t *testing.T) {
	consulta := &consultaFichaPropiaHTTP{}
	registro := &registroFichaPropiaHTTP{errVerificar: context.Canceled}
	ctx, cancelar := context.WithCancel(context.WithValue(context.Background(), contextoFichaPropiaHTTPPrueba{}, "correlacion_servidor"))
	cancelar()
	w := httptest.NewRecorder()
	manejadorFichaPropiaPrueba(t, consulta, registro).ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil).WithContext(ctx))
	if w.Code != http.StatusServiceUnavailable || consulta.llamadas != 0 || len(registro.intentos) != 1 || registro.intentos[0].Motivo != "no_disponible" {
		t.Fatalf("cancelación previa: estado=%d consultas=%d intentos=%d", w.Code, consulta.llamadas, len(registro.intentos))
	}
	if registro.contextos[0].Value(contextoFichaPropiaHTTPPrueba{}) != "correlacion_servidor" {
		t.Fatal("el intento perdió la correlación original")
	}
}

func TestFichaPropiaHTTPNegociaDisponibilidadSinCambiarContratoAnterior(t *testing.T) {
	for _, caso := range []struct {
		nombre     string
		acepta     []string
		disponible bool
		negociada  bool
	}{
		{"anterior", nil, true, false},
		{"json_anterior", []string{"application/json"}, true, false},
		{"nuevo_sin_montaje", []string{PerfilAceptacionFichaPropiaExportacion}, false, true},
		{"nuevo_con_montaje", []string{PerfilAceptacionFichaPropiaExportacion}, true, true},
		{"duplicada", []string{PerfilAceptacionFichaPropiaExportacion, PerfilAceptacionFichaPropiaExportacion}, true, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			m := manejadorFichaPropiaPrueba(t, &consultaFichaPropiaHTTP{}, &registroFichaPropiaHTTP{})
			m.exportacionDisponible = caso.disponible
			r := httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil)
			for _, acepta := range caso.acepta {
				r.Header.Add("Accept", acepta)
			}
			w := httptest.NewRecorder()
			w.Header().Set("Vary", "Origin")
			m.ServeHTTP(w, r)
			var sobre struct {
				Data map[string]json.RawMessage `json:"data"`
			}
			if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &sobre) != nil {
				t.Fatal("consulta no disponible", w.Code)
			}
			valor, existe := sobre.Data["exportacion_servicios_disponible"]
			if existe != caso.negociada || len(sobre.Data) != 3+map[bool]int{true: 1, false: 0}[caso.negociada] {
				t.Fatal("representación incompatible")
			}
			if existe {
				var disponible bool
				if json.Unmarshal(valor, &disponible) != nil || disponible != caso.disponible {
					t.Fatal("disponibilidad distinta al montaje")
				}
			}
			if w.Header().Get("Cache-Control") != "no-store" || len(w.Header().Values("Vary")) != 3 {
				t.Fatal("perdió las condiciones de caché anteriores")
			}
		})
	}
}

func TestFichaPropiaHistoriaEsRepresentacionOptativaSinReutilizarExportacion(t *testing.T) {
	for _, prefer := range []string{"", PreferenciaHistoriaServiciosFichaPropia, "otro"} {
		m := manejadorFichaPropiaPrueba(t, &consultaFichaPropiaHTTP{}, &registroFichaPropiaHTTP{})
		m.historiaDisponible = true
		r := httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil)
		r.Header.Set("Accept", PerfilAceptacionFichaPropiaExportacion)
		if prefer != "" {
			r.Header.Set("Prefer", prefer)
		}
		w := httptest.NewRecorder()
		m.ServeHTTP(w, r)
		var sobre struct {
			Data map[string]json.RawMessage `json:"data"`
		}
		if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &sobre) != nil {
			t.Fatal("consulta no disponible")
		}
		valor, existe := sobre.Data["historia_servicios_disponible"]
		if existe != (prefer == PreferenciaHistoriaServiciosFichaPropia) {
			t.Fatal("historia sin negociación")
		}
		if existe && string(valor) != "true" {
			t.Fatal("historia no coincide con montaje")
		}
		if string(sobre.Data["exportacion_servicios_disponible"]) != "false" {
			t.Fatal("historia habilitó exportación")
		}
	}
}
