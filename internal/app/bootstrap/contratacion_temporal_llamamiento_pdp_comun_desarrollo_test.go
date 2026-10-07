package bootstrap

import (
	"net/http"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctpostgres "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Con B-BACK el autorizador de Contratación delega en el PDP común. Cada
// petición del llamamiento debe encontrar su frontera (ruta, método y perfil
// CT) y una política solo para las acciones declaradas en ella; cualquier
// otra combinación sigue denegada.
func TestLlamamientoDeclaradoEnElPDPComun(t *testing.T) {
	const perfil = "prf_ct_prueba"
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo(descriptoresFronterasContratacionTemporalDesarrollo(perfil, []string{perfil}))
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoAutorizacionComunDesarrollo(fronteras, descriptoresAutorizacionContratacionTemporalDesarrollo(politicaDescriptoresCTPrueba(t)))
	if err != nil {
		t.Fatal(err)
	}
	declaradas := fronterasLlamamientoComunDesarrollo()
	if len(declaradas) != 10 {
		t.Fatalf("fronteras del llamamiento=%d, want 10", len(declaradas))
	}
	for _, f := range declaradas {
		d, ok := fronteras.resolver(f.metodo, f.ruta)
		if !ok || d.Clave != f.clave || !d.admitePerfil(perfil) || d.admitePerfil("prf_ajeno") || d.ClaveCapacidad != f.accion {
			t.Fatalf("%s sin frontera propia: %#v", f.ruta, d)
		}
		for _, accion := range append([]string{f.accion}, f.adicionales...) {
			if _, ok := catalogo.politicaPara(accion, f.clave, clavePoliticaContratacionTemporalDesarrollo, f.accion); !ok {
				t.Fatalf("%s sin política en %s", accion, f.clave)
			}
		}
	}
	// Un método no declarado no hereda la frontera de su ruta.
	for _, caso := range []struct{ metodo, ruta string }{
		{http.MethodGet, cthttp.RutaSeleccionLlamamiento},
		{http.MethodGet, cthttp.RutaResolucionFormalizacion},
		{http.MethodPost, cthttp.RutaConsultaComunicacionesExpediente},
		{http.MethodHead, cthttp.RutaConsultaReciboRespuesta},
	} {
		if d, ok := fronteras.resolver(caso.metodo, caso.ruta); ok {
			t.Fatalf("%s %s resolvió %s", caso.metodo, caso.ruta, d.Clave)
		}
	}
	// Las acciones solo valen en su frontera: ni efectos de Bolsa fuera de su
	// paso ni efectos del llamamiento en otras fronteras CT.
	for _, caso := range []struct{ accion, frontera, capacidad string }{
		{puertosbolsa.AccionAceptarLlamamientoRRHHDesarrollo, "ct-llamamiento-seleccionar", accionConsultarLlamamientoDesarrollo},
		{puertosbolsa.AccionPrepararOrdenDesarrollo, "ct-llamamiento-resolver", ctpostgres.AccionResolucionManualLlamamiento},
		{puertosbolsa.AccionAbrirSiguienteLlamamientoDesarrollo, "ct-llamamiento-resolver", ctpostgres.AccionResolucionManualLlamamiento},
		{puertosbolsa.AccionAceptarLlamamientoRRHHDesarrollo, "ct-llamamiento-continuar", ctpostgres.AccionContinuacionLlamamiento},
		{ctpostgres.AccionRegistroRespuestaRecibida, "ct-llamamiento-comunicacion-registrar", ctpostgres.AccionRegistroComunicacionLlamamiento},
		{ctpostgres.AccionResolucionFormalizacion, "ct-formalizacion-proponer", ctpostgres.AccionPropuestaFormalizacion},
		{ctpostgres.AccionRegistroComunicacionLlamamiento, "ct-analisis-registrar", ctports.AccionRegistrarAnalisis},
	} {
		if _, ok := catalogo.politicaPara(caso.accion, caso.frontera, clavePoliticaContratacionTemporalDesarrollo, caso.capacidad); ok {
			t.Fatalf("%s admitida en %s", caso.accion, caso.frontera)
		}
	}
}
