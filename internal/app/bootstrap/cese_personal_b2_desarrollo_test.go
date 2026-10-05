package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	domct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
)

// Sin cese_fecha_efecto el paso no se compone y el cese sigue siendo solo de
// CT; un valor que no está en la lista cerrada impide cargar la configuración.
func TestCesePersonalB2ConfiguracionDeFechaObligatoriaParaComponer(t *testing.T) {
	c := configuracionB2PuraPrueba()
	if cese, e := componerCesePersonalB2(c.PersonalB2, nil, nil, nil, nil, nil, nil); cese != nil || e != nil {
		t.Fatalf("sin regla de fecha se compuso el cese: %v", e)
	}
	if (&montajeIncorporacionPersonalB2{}).finCese(&soporteAltaContratacionTemporalDesarrollo{}, catalogoFronterasComunDesarrollo{}) != nil {
		t.Fatal("montaje sin cese devolvió un enganche")
	}
	var nulo *finCesePersonalB2Desarrollo
	servicio := &appct.ServicioOperacionesSeguimiento{}
	if e, ok := nulo.envolver(servicio, &estadoCeseB2Prueba{}).(*appct.ServicioOperacionesSeguimiento); !ok || e != servicio {
		t.Fatal("sin enganche el ejecutor de seguimiento debe ser el original")
	}
	c.PersonalB2.CeseFechaEfecto = string(inc.CeseUltimoDiaTrabajado)
	if _, e := componerCesePersonalB2(c.PersonalB2, nil, nil, nil, nil, nil, nil); e == nil {
		t.Fatal("con regla de fecha se compuso sin dependencias")
	}

	dir, e := os.MkdirTemp("/var/tmp", "vec-cese-b2-")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	if e = os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	ruta := filepath.Join(dir, "incorporacion.json")
	for _, caso := range []struct {
		valor  string
		valido bool
	}{{"", true}, {"ultimo_dia_trabajado", true}, {"primer_dia_sin_relacion", true}, {"fecha_de_efectos", false}} {
		c := configuracionB2PuraPrueba()
		c.PersonalB2.CeseFechaEfecto = caso.valor
		b, _ := json.Marshal(c)
		if e = os.WriteFile(ruta, b, 0600); e != nil {
			t.Fatal(e)
		}
		r, raiz, e := leerConfiguracionIncorporacionV2(ruta)
		if raiz != nil {
			raiz.Close()
		}
		if caso.valido && (e != nil || r.PersonalB2.CeseFechaEfecto != caso.valor) {
			t.Fatalf("regla %q rechazada: %v", caso.valor, e)
		}
		if !caso.valido && e == nil {
			t.Fatalf("regla %q admitida", caso.valor)
		}
	}
}

// En la ruta del cese los perfiles B2 solo pueden leer el origen y la ficha y
// registrar el hecho de la relación; nada de planes, RPT ni Bolsa.
func TestCesePersonalB2RutaDelCeseSoloAdmiteLecturaYHecho(t *testing.T) {
	ctx := context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: "POST", ruta: httpct.RutaCesesNombramiento})
	admitidas := map[string]bool{ct.AccionConsultarDetalleRRHH: true, ct.AccionLeerPlanNominalB2: true,
		personal.AccionFichaEmpleadoB2: true, personal.AccionHechoEmpleadoB2: true}
	for _, d := range operacionesIncorporacionB2() {
		if operacionPermitidaEnRutaIncorporacionB2(ctx, d.accion) != admitidas[d.accion] {
			t.Fatalf("acción %s mal acotada en la ruta del cese", d.accion)
		}
	}
	get := context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: "GET", ruta: httpct.RutaCesesNombramiento})
	if operacionPermitidaEnRutaIncorporacionB2(get, personal.AccionHechoEmpleadoB2) {
		t.Fatal("GET del cese no puede registrar hechos")
	}
	plan := context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: "POST", ruta: httpct.RutaPlanB2})
	if operacionPermitidaEnRutaIncorporacionB2(plan, personal.AccionHechoEmpleadoB2) {
		t.Fatal("la preparación del plan no registra hechos sueltos")
	}
}

type estadoCeseB2Prueba struct {
	incorporacion string
	fallo         error
	lecturas      int
}

func (e *estadoCeseB2Prueba) ConsultarEstadoSeguimiento(_ context.Context, _, exp string) (ct.EstadoSeguimientoExpediente, error) {
	e.lecturas++
	return ct.EstadoSeguimientoExpediente{ExpedienteRef: exp, IncorporacionRef: e.incorporacion}, e.fallo
}

type canalCeseB2Prueba struct{}

func (canalCeseB2Prueba) ResolverContextoCanalSeguimiento(context.Context) (appct.ContextoCanalSeguimiento, error) {
	return appct.ContextoCanalSeguimiento{AutenticacionRef: "aut_" + strings.Repeat("a", 32), SesionRef: "ses_" + strings.Repeat("b", 32),
		PerfilRef: "prf_0123456789abcdefghijkl", OrganizacionRef: "organizacion:prueba"}, nil
}
func (canalCeseB2Prueba) AutorizarLecturaSeguimiento(context.Context, string, string) error {
	return nil
}

// cesePorHTTPB2 pasa un cese por el manejador real con el enganche B2 y un
// soporte sin perfiles nominales: cualquier lectura con permiso B2 fallaría.
func cesePorHTTPB2(t *testing.T, estado *estadoCeseB2Prueba, falloCT error) (*httptest.ResponseRecorder, *int) {
	t.Helper()
	ceses := 0
	fin := &finCesePersonalB2Desarrollo{cese: &inc.CesePersonalB2{}, soporte: &soporteAltaContratacionTemporalDesarrollo{}}
	e, ok := fin.envolver(&appct.ServicioOperacionesSeguimiento{}, estado).(*ejecutorCeseConPersonalB2)
	if !ok {
		t.Fatal("el enganche no envolvió el servicio")
	}
	e.registrar = func(context.Context, appct.SolicitudRegistrarCese) (ct.ReciboOperacionSeguimiento, error) {
		ceses++
		return ct.ReciboOperacionSeguimiento{Operacion: ct.OperacionRegistrarCese, OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:prueba",
			VersionAnterior: 7, VersionResultante: 8, FaseResultante: domct.FaseNombramiento, EstadoResultante: domct.EstadoEnCurso,
			ReciboRef: "recibo:prueba", AuditoriaRef: "auditoria:prueba", EventoRef: "evento:prueba", ActorRef: "per_prueba",
			RegistradaEn: time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC), CausaClave: "fin_sustitucion", FechaEfecto: "2027-02-15"}, falloCT
	}
	m, err := httpct.NuevosManejadoresSeguimiento(canalCeseB2Prueba{}, canalCeseB2Prueba{}, e)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"expediente_ref":"expediente:prueba","version_esperada":7,"clave_idempotencia":"11111111-1111-4111-8111-111111111111","causa_clave":"fin_sustitucion","fecha_efecto":"2027-02-15","justificante_ref":"documento:prueba","justificante_sha256":"` + strings.Repeat("a", 64) + `","observaciones":""}`
	r := httptest.NewRequest(http.MethodPost, httpct.RutaCesesNombramiento, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m[httpct.RutaCesesNombramiento].ServeHTTP(w, r)
	return w, &ceses
}

// Un cese cuyo expediente no tiene origen B2 en CT responde 201 aunque no
// existan perfiles nominales B2; solo con origen B2 se intenta Personal.
func TestCesePersonalB2CeseSinOrigenB2NoDependeDePermisosB2(t *testing.T) {
	for _, incorporacion := range []string{"", "recibo:ct-incorporacion-v2:0123abcd"} {
		estado := &estadoCeseB2Prueba{incorporacion: incorporacion}
		w, ceses := cesePorHTTPB2(t, estado, nil)
		if w.Code != http.StatusCreated || *ceses != 1 || estado.lecturas != 1 {
			t.Fatalf("cese sin origen B2 (%q): %d %s", incorporacion, w.Code, w.Body.String())
		}
	}
	estado := &estadoCeseB2Prueba{incorporacion: ct.PrefijoReciboOrigenIncorporacionPersonalB2 + "5f0c2a4e-8d1b-4c7a-9e3f-1a2b3c4d5e6f"}
	if w, _ := cesePorHTTPB2(t, estado, nil); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("con origen B2 y sin contexto nominal el fin debe quedar pendiente: %d %s", w.Code, w.Body.String())
	}
	if w, _ := cesePorHTTPB2(t, &estadoCeseB2Prueba{fallo: ct.ErrResultadoSeguimientoNoConfiable}, nil); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("sin poder leer el estado de CT no se da el cese por completo: %d", w.Code)
	}
	estado = &estadoCeseB2Prueba{}
	if w, _ := cesePorHTTPB2(t, estado, ct.ErrCeseYaRegistrado); w.Code != http.StatusConflict || estado.lecturas != 0 {
		t.Fatalf("un rechazo de CT no debe pasar por el enganche: %d lecturas=%d", w.Code, estado.lecturas)
	}
}

// Si no se puede derivar el contexto nominal, el cese de CT no se presenta
// como completo: la respuesta queda pendiente (503) para repetir con la clave.
func TestCesePersonalB2SinContextoNominalNoDaPorHechoElFin(t *testing.T) {
	f := &finCesePersonalB2Desarrollo{cese: &inc.CesePersonalB2{}, soporte: &soporteAltaContratacionTemporalDesarrollo{}}
	err := f.finalizar(context.Background(), appct.SolicitudRegistrarCese{}, ct.ReciboOperacionSeguimiento{}, "")
	if !errors.Is(err, ct.ErrOperacionSeguimientoNoDisponible) || !errors.Is(err, errFinPersonalB2Pendiente) || errors.Is(err, ct.ErrAutorizacionDenegada) {
		t.Fatalf("fin sin contexto nominal no quedó pendiente: %v", err)
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := f.finalizar(cancelado, appct.SolicitudRegistrarCese{}, ct.ReciboOperacionSeguimiento{}, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación perdida: %v", err)
	}
}
