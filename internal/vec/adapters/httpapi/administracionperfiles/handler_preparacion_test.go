package administracionperfiles

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// lotesADMINPrueba aplica con la autoridad de prueba del lote y prepara
// devolviendo una opción de alta y otra de baja para la persona pedida.
type lotesADMINPrueba struct {
	*autoridadLoteHTTP
	preparaciones []domain.SolicitudPreparacionLoteAdministracionPerfiles
	err           error
	alterar       func(*domain.PreparacionLoteAdministracionPerfiles)
}

func (l *lotesADMINPrueba) PrepararLoteOrdinario(_ context.Context, s domain.SolicitudPreparacionLoteAdministracionPerfiles) (domain.PreparacionLoteAdministracionPerfiles, error) {
	l.preparaciones = append(l.preparaciones, s)
	if l.err != nil {
		return domain.PreparacionLoteAdministracionPerfiles{}, l.err
	}
	ahora := s.Actor.ResueltoEn
	p := domain.PreparacionLoteAdministracionPerfiles{OperacionRef: s.OperacionRef, AuditoriaRef: "aud_v3_" + strings.Repeat("a", 32),
		PreparadaEn: ahora, PersonaRef: s.PersonaRef, PersonaVersion: 1, CuentaRef: "cta_" + strings.Repeat("c", 32), CuentaVersion: 2,
		ProcedenciaRef: "prc_" + strings.Repeat("d", 32), ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("e", 64),
		OrganizacionRef: s.OrganizacionRef, UnidadRef: s.UnidadRef,
		Altas: []domain.AltaPosibleLoteAdministracion{{RolVersionRef: "rol:gestor_cronos:v1", Nombre: "Gestor",
			VigenteHastaMaxima: ahora.Add(24 * time.Hour), DuracionPropuesta: time.Hour,
			PerfilRef: "prf_" + strings.Repeat("1", 32), VinculoRef: "vca_" + strings.Repeat("2", 32), HuellaSHA256: strings.Repeat("3", 64)}},
		Bajas: []domain.BajaPosibleLoteAdministracion{{RolVersionRef: "rol:dietas_liquidacion_rrhh:v1", Nombre: "Liquidación de dietas",
			PerfilRef: "prf_" + strings.Repeat("4", 32), VinculoRef: "vca_" + strings.Repeat("5", 32), PerfilVersion: 1, VinculoVersion: 3,
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), HuellaSHA256: strings.Repeat("6", 64)}}}
	if l.alterar != nil {
		l.alterar(&p)
	}
	return p, nil
}

func handlerConLotePrueba(t *testing.T) (*Handler, *lotesADMINPrueba, *sesionPrueba, *auditorPrueba, SolicitudLote) {
	t.Helper()
	base, dto, autoridad, sesion, auditor, catalogo := loteHTTPPrueba(t)
	lotes := &lotesADMINPrueba{autoridadLoteHTTP: autoridad}
	h, err := NuevoHandlerUsuariosMetadatosConLote("https://admin.example.test", "org_prueba", sesion,
		base.lecturas, catalogo, lotes, auditor)
	if err != nil {
		t.Fatal(err)
	}
	return h, lotes, sesion, auditor, dto
}

const personaPreparacionPrueba = "per_" + "ffffffffffffffffffffff"

func getPreparacionPrueba(h *Handler, persona, consulta string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	ruta := PrefijoV1 + "/personas/" + persona + SufijoPreparacionLote
	if consulta != "" {
		ruta += "?" + consulta
	}
	h.ServeHTTP(w, peticionADMIN(http.MethodGet, ruta, ""))
	return w
}

func TestHTTPPreparacionLoteDevuelveOpcionesConOrganizacionPrivada(t *testing.T) {
	h, lotes, sesion, auditor, _ := handlerConLotePrueba(t)
	w := getPreparacionPrueba(h, personaPreparacionPrueba, "unidad_ref=unidad:prueba")
	if w.Code != http.StatusOK || len(lotes.preparaciones) != 1 || auditor.llamadas != 0 {
		t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
	}
	s := lotes.preparaciones[0]
	if s.OrganizacionRef != "org_prueba" || s.UnidadRef != "unidad:prueba" || s.PersonaRef != personaPreparacionPrueba ||
		s.Actor.PersonaRef != sesion.resultado.Actor.PersonaRef || !domain.ReferenciaPreparacionLoteValida(s.OperacionRef) {
		t.Fatal("preparación con datos no confiables")
	}
	var r struct {
		Preparacion PreparacionLote `json:"preparacion"`
	}
	if json.Unmarshal(w.Body.Bytes(), &r) != nil || r.Preparacion.CuentaVersion != 2 || len(r.Preparacion.Altas) != 1 ||
		r.Preparacion.Altas[0].DuracionPropuestaSegundos != 3600 || r.Preparacion.Bajas[0].VinculoVersion != 3 || r.Preparacion.Bajas[0].Nombre != "Liquidación de dietas" ||
		strings.Contains(w.Body.String(), "org_prueba") {
		t.Fatal("respuesta de preparación incompleta o con la organización privada")
	}
	if w.Header().Get("Set-Cookie") != "" {
		t.Fatal("preparación introduce persistencia web")
	}
}

func TestHTTPPreparacionLoteRechazaConsultaYPersonaAntesDelServicio(t *testing.T) {
	for nombre, caso := range map[string]struct{ persona, consulta string }{
		"sin_unidad":      {personaPreparacionPrueba, ""},
		"clave_extra":     {personaPreparacionPrueba, "unidad_ref=unidad:prueba&organizacion_ref=otra"},
		"unidad_doble":    {personaPreparacionPrueba, "unidad_ref=unidad:prueba&unidad_ref=unidad:otra"},
		"unidad_invalida": {personaPreparacionPrueba, "unidad_ref=Unidad*"},
		"persona_opaca":   {"per_corta", "unidad_ref=unidad:prueba"},
	} {
		t.Run(nombre, func(t *testing.T) {
			h, lotes, _, auditor, _ := handlerConLotePrueba(t)
			w := getPreparacionPrueba(h, caso.persona, caso.consulta)
			if w.Code != http.StatusBadRequest || len(lotes.preparaciones) != 0 || auditor.llamadas != 1 {
				t.Fatalf("estado=%d servicio=%d auditoria=%d", w.Code, len(lotes.preparaciones), auditor.llamadas)
			}
		})
	}
}

func TestHTTPPreparacionLoteNoSePrepararParaUnoMismo(t *testing.T) {
	h, lotes, sesion, auditor, _ := handlerConLotePrueba(t)
	w := getPreparacionPrueba(h, sesion.resultado.Actor.PersonaRef, "unidad_ref=unidad:prueba")
	if w.Code != http.StatusBadRequest || len(lotes.preparaciones) != 0 || auditor.llamadas != 1 {
		t.Fatal("preparación sobre el propio actor")
	}
}

func TestHTTPPreparacionLoteErroresDelServicio(t *testing.T) {
	for nombre, caso := range map[string]struct {
		err    error
		estado int
	}{
		"denegada":   {domain.ErrAutorizacionDenegada, http.StatusForbidden},
		"sin_lote":   {domain.ErrControlAdministracionPerfilesInvalido, http.StatusForbidden},
		"conflicto":  {ErrConflictoEstado, http.StatusConflict},
		"invalida":   {domain.ErrActoAdministracionPerfilesInvalido, http.StatusServiceUnavailable},
		"no_dispone": {context.DeadlineExceeded, http.StatusServiceUnavailable},
	} {
		t.Run(nombre, func(t *testing.T) {
			h, lotes, _, auditor, _ := handlerConLotePrueba(t)
			lotes.err = caso.err
			w := getPreparacionPrueba(h, personaPreparacionPrueba, "unidad_ref=unidad:prueba")
			if w.Code != caso.estado || auditor.llamadas != 1 {
				t.Fatalf("estado=%d", w.Code)
			}
		})
	}
}

func TestHTTPPreparacionLoteRespuestaAjenaNoSeEntrega(t *testing.T) {
	h, lotes, _, auditor, _ := handlerConLotePrueba(t)
	lotes.alterar = func(p *domain.PreparacionLoteAdministracionPerfiles) { p.PersonaRef = "per_" + strings.Repeat("9", 22) }
	w := getPreparacionPrueba(h, personaPreparacionPrueba, "unidad_ref=unidad:prueba")
	if w.Code != http.StatusServiceUnavailable || auditor.llamadas != 1 || strings.Contains(w.Body.String(), "per_9") {
		t.Fatal("respuesta ajena entregada")
	}
}

// Con sólo el lote montado, el POST del lote llega al servicio y cualquier
// otra escritura no existe; las lecturas nominales siguen igual.
func TestHTTPConLoteSoloAbreLoteYPreparacion(t *testing.T) {
	h, lotes, _, auditor, dto := handlerConLotePrueba(t)
	if w := postLotePrueba(t, h, dto); w.Code != http.StatusOK || len(lotes.solicitudes) != 1 {
		t.Fatalf("lote no aplicado: %d %s", w.Code, w.Body.String())
	}
	for _, ruta := range []string{PrefijoV1 + "/actos-ordinarios", PrefijoV1 + "/propuestas"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionADMIN(http.MethodPost, ruta, "{}"))
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s abierto: %d", ruta, w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/capacidades", ""))
	if w.Code != http.StatusNotFound || auditor.llamadas != 3 {
		t.Fatalf("lectura fuera de metadatos abierta: %d", w.Code)
	}
}

// Sin la autoridad del lote, la ruta de preparación no existe.
func TestHTTPMetadatosSinLoteNoPrepara(t *testing.T) {
	base, _, _, sesion, auditor, _ := loteHTTPPrueba(t)
	h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", sesion, base.lecturas, auditor)
	if err != nil {
		t.Fatal(err)
	}
	w := getPreparacionPrueba(h, personaPreparacionPrueba, "unidad_ref=unidad:prueba")
	if w.Code == http.StatusOK {
		t.Fatal("preparación servida sin autoridad del lote")
	}
}

// La preparación se audita con su propia clave de destino.
func TestHTTPPreparacionLoteAuditaConSuAccion(t *testing.T) {
	h, _, _, auditor, _ := handlerConLotePrueba(t)
	getPreparacionPrueba(h, personaPreparacionPrueba, "")
	if auditor.llamadas != 1 || auditor.ultima.Accion != "preparar_lote_ordinario" {
		t.Fatalf("acción auditada: %q", auditor.ultima.Accion)
	}
}

// Sin la concesión del lote, GET y POST responden igual: 403 auditado.
func TestHTTPSinConcesionDelLoteGETyPOSTDan403(t *testing.T) {
	h, lotes, _, auditor, dto := handlerConLotePrueba(t)
	lotes.err = domain.ErrControlAdministracionPerfilesInvalido
	lotes.autoridadLoteHTTP.err = domain.ErrControlAdministracionPerfilesInvalido
	if w := getPreparacionPrueba(h, personaPreparacionPrueba, "unidad_ref=unidad:prueba"); w.Code != http.StatusForbidden {
		t.Fatalf("GET %d", w.Code)
	}
	if w := postLotePrueba(t, h, dto); w.Code != http.StatusForbidden {
		t.Fatalf("POST %d", w.Code)
	}
	if auditor.llamadas != 2 || auditor.ultima.Accion != "aplicar_lote_ordinario" {
		t.Fatal("denegaciones sin auditar")
	}
}
