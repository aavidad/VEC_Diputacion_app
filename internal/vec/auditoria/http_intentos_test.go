package auditoria

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type opcionesIntentosPrueba struct {
	valor  Opciones
	err    error
	alLeer func()
}

func (o *opcionesIntentosPrueba) Actuales(context.Context) (Opciones, error) {
	if o.alLeer != nil {
		o.alLeer()
	}
	return o.valor, o.err
}

type registradorIntentosConsultaPrueba struct {
	ordenes             []ports.OrdenIntentoAuditoria
	contextosCancelados []bool
	err                 error
	primeroNoDisponible bool
}

func (r *registradorIntentosConsultaPrueba) AppendIntentoAuditoria(ctx context.Context, orden ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	r.ordenes = append(r.ordenes, orden)
	r.contextosCancelados = append(r.contextosCancelados, ctx.Err() != nil)
	if r.primeroNoDisponible && len(r.ordenes) == 1 {
		return ports.AcuseIntentoAuditoria{}, ports.ErrIntentoAuditoriaNoDisponible
	}
	if r.err != nil {
		return ports.AcuseIntentoAuditoria{}, r.err
	}
	datos, err := orden.Datos()
	if err != nil {
		return ports.AcuseIntentoAuditoria{}, err
	}
	return ports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_sintetica", Secuencia: int64(len(r.ordenes)),
		HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: datos.Datos.CorrelacionRef,
		RegistradaEn: time.Now().UTC().Truncate(time.Microsecond)}, nil
}

func TestIntentosConsultaAuditoriaConservanIdentidadYResultado(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	p := peticionAuditoriaIdentidadPrueba(t, ahora)
	identidad := IdentidadResuelta{Vinculo: p.Contexto.Vinculo, Resultado: p.Contexto.Resultado,
		Correlacion: p.Contexto.Correlacion}
	registrador := &registradorIntentosConsultaPrueba{}
	opciones := &opcionesIntentosPrueba{valor: Opciones{FinalidadRef: p.Filtro.FinalidadRef,
		MotivoRef: p.Filtro.MotivoRef, Motivo: p.Contexto.Motivo, PermisoRequerido: AccionConsultar}}
	s, err := NuevoServicio(&emisorAuditoriaIdentidadPrueba{}, fuenteAuditoriaIdentidadPrueba{}, fuenteAuditoriaIdentidadPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConfigurarIntentos(ConfiguracionIntentos{Registrador: registrador,
		Proceso: "vec_server_ensayo", Canal: string(domain.SuperficieAutenticacionInternaCorporativaV1),
		Finalidad: p.Filtro.FinalidadRef, Motivo: p.Contexto.Motivo}); err != nil {
		t.Fatal(err)
	}
	h, err := NuevoManejador(s, opciones, identidadAuditoriaHTTPPrueba{identidad})
	if err != nil {
		t.Fatal(err)
	}
	consultar := func(desde string) *httptest.ResponseRecorder {
		t.Helper()
		cuerpo, _ := json.Marshal(map[string]any{"fuente": "ct", "expediente_ref": p.Filtro.ExpedienteRef,
			"desde": desde, "hasta": p.Filtro.Hasta.Format(time.RFC3339Nano), "limite": 5,
			"finalidad_ref": p.Filtro.FinalidadRef, "motivo_ref": p.Filtro.MotivoRef})
		r := httptest.NewRequest(http.MethodPost, RutaConsulta, strings.NewReader(string(cuerpo)))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	w := consultar("fecha no válida")
	if w.Code != http.StatusBadRequest || w.Header().Get("X-Audit-Ref") != "auditoria_sintetica" {
		t.Fatalf("400 identificado sin acuse: %d %q", w.Code, w.Header().Get("X-Audit-Ref"))
	}
	w = consultar(p.Filtro.Desde.Format(time.RFC3339Nano))
	if w.Code != http.StatusForbidden || w.Header().Get("X-Audit-Ref") != "auditoria_sintetica" {
		t.Fatalf("403 V3 sin acuse: %d %q", w.Code, w.Header().Get("X-Audit-Ref"))
	}
	s.emisor = &emisorAuditoriaIdentidadPrueba{fallo: errors.New("emisor no disponible")}
	w = consultar(p.Filtro.Desde.Format(time.RFC3339Nano))
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("X-Audit-Ref") != "auditoria_sintetica" {
		t.Fatalf("503 V3 sin acuse: %d %q", w.Code, w.Header().Get("X-Audit-Ref"))
	}
	opciones.err = ErrNoDisponible
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaOpciones, nil))
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("X-Audit-Ref") != "auditoria_sintetica" {
		t.Fatalf("GET 503 sin acuse: %d %q", w.Code, w.Header().Get("X-Audit-Ref"))
	}
	if len(registrador.ordenes) != 4 {
		t.Fatalf("intentos nominales = %d", len(registrador.ordenes))
	}
	for i, esperado := range []struct {
		resultado domain.ResultadoIntentoAuditoria
		recurso   string
	}{{domain.ResultadoIntentoAuditoriaError, "auditoria:consulta_invalida"},
		{domain.ResultadoIntentoAuditoriaDenegado, "auditoria:consulta_ct"},
		{domain.ResultadoIntentoAuditoriaError, "auditoria:consulta_ct"},
		{domain.ResultadoIntentoAuditoriaError, "auditoria:opciones_rrhh"}} {
		datos, err := registrador.ordenes[i].Datos()
		if err != nil || datos.Datos.Resultado != esperado.resultado || datos.Datos.RecursoRef != esperado.recurso ||
			datos.Datos.Accion != AccionConsultar || datos.Datos.FinalidadRef != p.Filtro.FinalidadRef ||
			datos.ResultadoContexto.Contexto.PerfilActivoRef != identidad.Resultado.Contexto.PerfilActivoRef ||
			datos.Vinculo.ValidarPara(identidad.Resultado) != nil {
			t.Fatalf("intento %d perdió contexto nominal: %+v %v", i, datos.Datos, err)
		}
	}
	registrador.err = errors.New("registrador caído")
	opciones.err = nil
	s.emisor = &emisorAuditoriaIdentidadPrueba{}
	w = consultar(p.Filtro.Desde.Format(time.RFC3339Nano))
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("X-Audit-Ref") != "" {
		t.Fatalf("fallo AD169 no cerró respuesta: %d %q", w.Code, w.Header().Get("X-Audit-Ref"))
	}
	registrador.err, registrador.ordenes, registrador.primeroNoDisponible = nil, nil, true
	w = consultar(p.Filtro.Desde.Format(time.RFC3339Nano))
	if w.Code != http.StatusForbidden || len(registrador.ordenes) != 2 {
		t.Fatalf("recuperación AD169: HTTP %d intentos=%d", w.Code, len(registrador.ordenes))
	}
	primero, err1 := registrador.ordenes[0].Datos()
	segundo, err2 := registrador.ordenes[1].Datos()
	if err1 != nil || err2 != nil || primero.IntentoRef != segundo.IntentoRef {
		t.Fatal("un COMMIT ambiguo recibió otra clave de intento")
	}
	const identificadorPersonal = "dni:12345678z"
	registrador.ordenes, registrador.primeroNoDisponible = nil, false
	p.Filtro.ExpedienteRef = identificadorPersonal
	w = consultar(p.Filtro.Desde.Format(time.RFC3339Nano))
	if w.Code != http.StatusForbidden || strings.Contains(w.Body.String(), identificadorPersonal) ||
		len(registrador.ordenes) != 1 {
		t.Fatalf("identificador personal en respuesta o sin intento: %d %q", w.Code, w.Body.String())
	}
	datosPersonales, err := registrador.ordenes[0].Datos()
	if err != nil || datosPersonales.Datos.RecursoRef != "auditoria:consulta_ct" ||
		strings.Contains(datosPersonales.Datos.RecursoRef, identificadorPersonal) {
		t.Fatalf("identificador personal en AD169: %+v %v", datosPersonales.Datos, err)
	}
	if _, err := s.registrarIntento(t.Context(), identidad, identificadorPersonal,
		domain.ResultadoIntentoAuditoriaDenegado, &opciones.valor); !errors.Is(err, ErrNoDisponible) ||
		len(registrador.ordenes) != 1 {
		t.Fatal("el servicio aceptó un recurso HTTP bruto para AD169")
	}
	registrador.err = errors.New("registrador caído")
	var registro bytes.Buffer
	previo := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&registro, nil)))
	defer slog.SetDefault(previo)
	w = consultar(p.Filtro.Desde.Format(time.RFC3339Nano))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), identificadorPersonal) ||
		strings.Contains(registro.String(), identificadorPersonal) {
		t.Fatalf("identificador personal en fallo AD169: respuesta=%q registro=%q", w.Body.String(), registro.String())
	}
	registrador.err = nil
	anteriores := len(registrador.ordenes)
	h.identidad = identidadAuditoriaHTTPPrueba{}
	w = consultar(p.Filtro.Desde.Format(time.RFC3339Nano))
	if w.Code != http.StatusForbidden || len(registrador.ordenes) != anteriores {
		t.Fatal("identidad ausente produjo intento nominal")
	}
}

func TestRecursoIntentoConsultaNoPersisteTextoPersonal(t *testing.T) {
	p := peticionAuditoriaIdentidadPrueba(t, time.Now().UTC().Truncate(time.Microsecond))
	for _, ref := range []string{"persona@example.invalid", "http:privado", "expediente:ñ", "dni:12345678z", strings.Repeat("a", 201)} {
		p.Filtro.ExpedienteRef = ref
		if recursoIntentoFiltro(p.Filtro) != "auditoria:consulta_ct" {
			t.Fatalf("referencia no minimizada: %q", ref)
		}
	}
	p.Filtro.Fuente = "bolsa"
	if recursoIntentoFiltro(p.Filtro) != "auditoria:consulta_bolsa" {
		t.Fatal("fuente Bolsa sin recurso fijo")
	}
	p.Filtro.ExpedienteRef = "*"
	if recursoIntentoFiltro(p.Filtro) != "auditoria:consulta_invalida" {
		t.Fatal("filtro inválido recibió recurso de consulta")
	}
}

func TestCancelacionTrasOpcionesRegistraErrorSinFalsaDenegacion(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	p := peticionAuditoriaIdentidadPrueba(t, ahora)
	identidad := IdentidadResuelta{Vinculo: p.Contexto.Vinculo, Resultado: p.Contexto.Resultado,
		Correlacion: p.Contexto.Correlacion}
	registrador := &registradorIntentosConsultaPrueba{primeroNoDisponible: true}
	emisor := &emisorAuditoriaIdentidadPrueba{}
	s, err := NuevoServicio(emisor, fuenteAuditoriaIdentidadPrueba{}, fuenteAuditoriaIdentidadPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConfigurarIntentos(ConfiguracionIntentos{Registrador: registrador,
		Proceso: "vec_server_ensayo", Canal: string(domain.SuperficieAutenticacionInternaCorporativaV1),
		Finalidad: p.Filtro.FinalidadRef, Motivo: p.Contexto.Motivo}); err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(t.Context())
	defer cancelar()
	opciones := &opcionesIntentosPrueba{valor: Opciones{FinalidadRef: p.Filtro.FinalidadRef,
		MotivoRef: p.Filtro.MotivoRef, Motivo: p.Contexto.Motivo, PermisoRequerido: AccionConsultar},
		alLeer: cancelar}
	h, err := NuevoManejador(s, opciones, identidadAuditoriaHTTPPrueba{identidad})
	if err != nil {
		t.Fatal(err)
	}
	cuerpo, _ := json.Marshal(map[string]any{"fuente": "ct", "expediente_ref": p.Filtro.ExpedienteRef,
		"desde": p.Filtro.Desde.Format(time.RFC3339Nano), "hasta": p.Filtro.Hasta.Format(time.RFC3339Nano),
		"limite": 5, "finalidad_ref": p.Filtro.FinalidadRef, "motivo_ref": p.Filtro.MotivoRef})
	r := httptest.NewRequest(http.MethodPost, RutaConsulta, strings.NewReader(string(cuerpo))).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("X-Audit-Ref") != "auditoria_sintetica" ||
		emisor.llamadas != 0 || len(registrador.ordenes) != 2 ||
		len(registrador.contextosCancelados) != 2 || !registrador.contextosCancelados[0] || !registrador.contextosCancelados[1] {
		t.Fatalf("cancelación mal clasificada: HTTP %d acuse=%q emisor=%d intentos=%d",
			w.Code, w.Header().Get("X-Audit-Ref"), emisor.llamadas, len(registrador.ordenes))
	}
	primero, e1 := registrador.ordenes[0].Datos()
	segundo, e2 := registrador.ordenes[1].Datos()
	if e1 != nil || e2 != nil || primero.IntentoRef != segundo.IntentoRef ||
		primero.Datos.Resultado != domain.ResultadoIntentoAuditoriaError ||
		segundo.Datos.Resultado != domain.ResultadoIntentoAuditoriaError ||
		primero.Datos.RecursoRef != "auditoria:consulta_ct" {
		t.Fatalf("cancelación sin orden idempotente de error: %+v %+v", primero.Datos, segundo.Datos)
	}
}
