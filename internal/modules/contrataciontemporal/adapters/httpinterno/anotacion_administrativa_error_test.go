package httpinterno

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestManejadorAnotacionClasificaErroresNominales(t *testing.T) {
	contexto := ContextoCanalAnotacionAdministrativa{
		AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa",
		SesionRef:        "ses_bbbbbbbbbbbbbbbbbbbbbbbb",
		PerfilRef:        "prf_cccccccccccccccccccccccc",
		OrganizacionRef:  refD(),
	}
	casos := []struct {
		nombre, codigo string
		estado         int
		errorAutoridad error
		errorEjecutor  error
	}{
		{"autenticacion ausente", "autenticacion_requerida", http.StatusUnauthorized, ErrContextoCanalAusente, nil},
		{"acceso denegado", "acceso_denegado", http.StatusForbidden, nil, application.ErrAnotacionAdministrativaDenegada},
		{"CAS en conflicto", "conflicto", http.StatusConflict, nil, domain.ErrVersionEnConflicto},
		{"persistencia no disponible", "servicio_no_disponible", http.StatusServiceUnavailable, nil, ports.ErrPersistenciaAnotacionAdministrativaNoDisponible},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			ejecutor := &ejecutorAnotacionPrueba{err: caso.errorEjecutor}
			h, err := NuevoManejadorAnotacionAdministrativa(
				autoridadAnotacionPrueba{c: contexto, err: caso.errorAutoridad}, ejecutor,
			)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodPost, RutaAnotacionesAdministrativas,
				strings.NewReader(`{"expediente_ref":"`+refA()+`","version_esperada":8,"clave_idempotencia":"11111111-2222-4333-8444-555555555555","observaciones":"nota"}`))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado {
				t.Fatalf("estado=%d, quiere %d: %s", w.Code, caso.estado, w.Body.String())
			}
			var salida struct {
				Error struct {
					Codigo    string `json:"codigo"`
					ClaveI18n string `json:"clave_i18n"`
				} `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &salida); err != nil {
				t.Fatal(err)
			}
			if salida.Error.Codigo != caso.codigo || salida.Error.ClaveI18n != "api.contratacion_temporal.anotacion_administrativa.error."+caso.codigo {
				t.Fatalf("error público=%+v", salida.Error)
			}
			if strings.Contains(w.Body.String(), "persistencia") || strings.Contains(w.Body.String(), "contratacion temporal:") {
				t.Fatalf("la respuesta expone detalle interno: %s", w.Body.String())
			}
			if caso.errorAutoridad != nil && ejecutor.registros != 0 {
				t.Fatalf("ejecutor llamado antes de resolver autoridad: %d", ejecutor.registros)
			}
			if caso.errorEjecutor != nil && ejecutor.registros != 1 {
				t.Fatalf("ejecutor no llamado: %d", ejecutor.registros)
			}
		})
	}
}

func TestManejadorAnotacionRechazaRecibosIncompletosOSinIncrementoSeguro(t *testing.T) {
	c := ContextoCanalAnotacionAdministrativa{AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa", SesionRef: "ses_bbbbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_cccccccccccccccccccccccc", OrganizacionRef: refD()}
	casos := []struct {
		nombre  string
		alterar func(*ports.ReciboAnotacionAdministrativa)
	}{
		{"fase_omitida", func(r *ports.ReciboAnotacionAdministrativa) { r.FaseResultante = "" }},
		{"fase_invalida", func(r *ports.ReciboAnotacionAdministrativa) { r.FaseResultante = "fase con espacios" }},
		{"estado_omitido", func(r *ports.ReciboAnotacionAdministrativa) { r.EstadoResultante = "" }},
		{"estado_desconocido", func(r *ports.ReciboAnotacionAdministrativa) { r.EstadoResultante = "inventado" }},
		{"estado_completado", func(r *ports.ReciboAnotacionAdministrativa) { r.EstadoResultante = domain.EstadoCompletado }},
		{"estado_cancelado", func(r *ports.ReciboAnotacionAdministrativa) { r.EstadoResultante = domain.EstadoCancelado }},
		{"cero_a_uno", func(r *ports.ReciboAnotacionAdministrativa) { r.VersionAnterior = 0; r.VersionResultante = 1 }},
		{"overflow_a_cero", func(r *ports.ReciboAnotacionAdministrativa) { r.VersionAnterior = ^uint64(0); r.VersionResultante = 0 }},
		{"sobre_maximo_seguro", func(r *ports.ReciboAnotacionAdministrativa) {
			r.VersionAnterior = ports.MaximoEnteroSeguroOperacionAnalisis
			r.VersionResultante = r.VersionAnterior + 1
		}},
		{"sin_incremento", func(r *ports.ReciboAnotacionAdministrativa) { r.VersionResultante = r.VersionAnterior }},
		{"version_original_distinta", func(r *ports.ReciboAnotacionAdministrativa) { r.SeguimientoOriginal.VersionSeguimiento = 2 }},
		{"raiz_original_omitida", func(r *ports.ReciboAnotacionAdministrativa) { r.SeguimientoOriginal.HuellaRaizSeguimientoSHA256 = "" }},
		{"actor_omitido", func(r *ports.ReciboAnotacionAdministrativa) { r.ActorRef = "" }},
		{"recibo_omitido", func(r *ports.ReciboAnotacionAdministrativa) { r.ReciboRef = "" }},
		{"auditoria_omitida", func(r *ports.ReciboAnotacionAdministrativa) { r.AuditoriaRef = "" }},
		{"evento_omitido", func(r *ports.ReciboAnotacionAdministrativa) { r.EventoRef = "" }},
		{"organizacion_ajena", func(r *ports.ReciboAnotacionAdministrativa) { r.OrganizacionRef = refA() }},
		{"expediente_ajeno", func(r *ports.ReciboAnotacionAdministrativa) { r.ExpedienteRef = refC() }},
		{"operacion_ajena", func(r *ports.ReciboAnotacionAdministrativa) { r.Operacion = "registrar_incorporacion" }},
	}
	for _, metodo := range []string{http.MethodPost, http.MethodGet} {
		for _, caso := range casos {
			t.Run(metodo+"/"+caso.nombre, func(t *testing.T) {
				recibo := reciboAnotacionPrueba(c, refA(), 8)
				caso.alterar(&recibo)
				e := &ejecutorAnotacionPrueba{r: recibo}
				h, _ := NuevoManejadorAnotacionAdministrativa(autoridadAnotacionPrueba{c: c}, e)
				ruta := RutaAnotacionesAdministrativas
				cuerpo := `{"expediente_ref":"` + refA() + `","version_esperada":8,"clave_idempotencia":"11111111-2222-4333-8444-555555555555","observaciones":"nota"}`
				if metodo == http.MethodGet {
					ruta = RutaRecuperacionAnotacionesAdministrativas + "?expediente_ref=" + refA() + "&clave_idempotencia=11111111-2222-4333-8444-555555555555"
					cuerpo = ""
				}
				req := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
				if metodo == http.MethodPost {
					req.Header.Set("Content-Type", "application/json")
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, req)
				if w.Code != http.StatusServiceUnavailable || e.registros+e.recuperaciones != 1 {
					t.Fatalf("recibo inconsistente publicado: %d", w.Code)
				}
				var sobre map[string]json.RawMessage
				if err := json.Unmarshal(w.Body.Bytes(), &sobre); err != nil || len(sobre) != 1 || sobre["error"] == nil || sobre["data"] != nil {
					t.Fatalf("respuesta expone recibo inválido: %v", err)
				}
				var detalle struct {
					Codigo         string `json:"codigo"`
					ClaveI18n      string `json:"clave_i18n"`
					CorrelacionRef string `json:"correlacion_ref"`
				}
				if err := json.Unmarshal(sobre["error"], &detalle); err != nil || detalle.Codigo != "servicio_no_disponible" || detalle.ClaveI18n != "api.contratacion_temporal.anotacion_administrativa.error.servicio_no_disponible" || detalle.CorrelacionRef == "" {
					t.Fatalf("error no nominal: %v", err)
				}
			})
		}
	}
}
func TestManejadorAnotacionGETAdmiteUltimoIncrementoSeguro(t *testing.T) {
	c := ContextoCanalAnotacionAdministrativa{AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa", SesionRef: "ses_bbbbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_cccccccccccccccccccccccc", OrganizacionRef: refD()}
	e := &ejecutorAnotacionPrueba{r: reciboAnotacionPrueba(c, refA(), ports.MaximoEnteroSeguroOperacionAnalisis-1)}
	h, _ := NuevoManejadorAnotacionAdministrativa(autoridadAnotacionPrueba{c: c}, e)
	req := httptest.NewRequest(http.MethodGet, RutaRecuperacionAnotacionesAdministrativas+"?expediente_ref="+refA()+"&clave_idempotencia=11111111-2222-4333-8444-555555555555", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK || e.recuperaciones != 1 {
		t.Fatalf("último incremento rechazado: %d", w.Code)
	}
	comprobarReciboAnotacionJSON(t, w, e.r)
}
