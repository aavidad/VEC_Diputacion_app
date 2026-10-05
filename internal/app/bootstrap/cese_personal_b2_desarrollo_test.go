package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
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
	if e, ok := nulo.envolver(servicio).(*appct.ServicioOperacionesSeguimiento); !ok || e != servicio {
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

// Si no se puede derivar el contexto nominal, el cese de CT no se presenta
// como completo: la respuesta queda pendiente (503) para repetir con la clave.
func TestCesePersonalB2SinContextoNominalNoDaPorHechoElFin(t *testing.T) {
	f := &finCesePersonalB2Desarrollo{cese: &inc.CesePersonalB2{}, soporte: &soporteAltaContratacionTemporalDesarrollo{}}
	err := f.finalizar(context.Background(), appct.SolicitudRegistrarCese{}, ct.ReciboOperacionSeguimiento{})
	if !errors.Is(err, ct.ErrOperacionSeguimientoNoDisponible) || !errors.Is(err, errFinPersonalB2Pendiente) || errors.Is(err, ct.ErrAutorizacionDenegada) {
		t.Fatalf("fin sin contexto nominal no quedó pendiente: %v", err)
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := f.finalizar(cancelado, appct.SolicitudRegistrarCese{}, ct.ReciboOperacionSeguimiento{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación perdida: %v", err)
	}
}
