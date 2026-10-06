package bootstrap

import (
	"context"
	"strings"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// La aceptación en Bolsa se autorizaba solo si el recurso llevaba la
// categoría C2 y la unidad de ejemplo. En un expediente de otra categoría la
// resolución CT quedaba confirmada y Bolsa nunca registraba la aceptación:
// 503 en cada reintento. El permiso se liga ahora a la categoría del análisis
// y a la unidad de la asignación del expediente, que son las de la necesidad.
// (La unidad del escenario coincide con la de ejemplo; se comprueba aparte
// que otra unidad no se acepta.)
func TestAceptacionBolsaSeLigaALaCategoriaYUnidadDelExpediente(t *testing.T) {
	ctx, _, l := escenarioRevisionManualPrueba(t)
	preparacion, ok := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	if !ok || preparacion.expediente.Fiscalizado.Analisis == nil || preparacion.expediente.Fiscalizado.Asignacion == nil {
		t.Fatal("escenario sin expediente preparado")
	}
	preparacion.expediente.Fiscalizado = preparacion.expediente.Fiscalizado.Clonar()
	preparacion.expediente.Fiscalizado.Analisis.CategoriaRef = "categoria:desarrollo:c1"
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, preparacion)
	ctx = context.WithValue(ctx, claveAceptacionRevisadaDesarrollo{}, l)
	datos := func(categoria, unidad string) dominiovec.DatosSolicitudAutorizacionLigadaV3 {
		return dominiovec.DatosSolicitudAutorizacionLigadaV3{Finalidad: "gestionar_contratacion_temporal",
			Accion: puertosbolsa.AccionAceptarLlamamientoRRHHDesarrollo, ReferenciaMotivo: motivoResolucionManualDesarrollo(true),
			Recurso: dominiovec.RecursoAutorizable{Referencia: operacionAceptacionManualDesarrollo(l),
				ModuloID: "bolsa", Tipo: "integracion_llamamientos_bolsa",
				Ambitos:   map[string]string{"categoria_ref": categoria, "unidad_ref": unidad},
				Atributos: map[string]string{"necesidad_ref": l.justificante.Seleccion.Necesidad.Referencia, "contenido_sha256": strings.Repeat("a", 64)}}}
	}
	unidad := preparacion.expediente.Fiscalizado.Asignacion.UnidadRef
	ruta := httpinterno.RutaResolucionComunicacionLlamamiento
	if !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, ruta, datos("categoria:desarrollo:c1", unidad)) {
		t.Fatal("aceptación del expediente C1 rechazada")
	}
	for nombre, d := range map[string]dominiovec.DatosSolicitudAutorizacionLigadaV3{
		"categoría fija de ejemplo": datos("categoria:desarrollo:c2", unidad),
		"otra unidad":               datos("categoria:desarrollo:c1", "unidad:desarrollo:otra"),
		"ámbitos vacíos":            datos("", ""),
	} {
		if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, ruta, d) {
			t.Fatalf("%s autorizada para otro expediente", nombre)
		}
	}
	d := datos("categoria:desarrollo:c1", unidad)
	d.Recurso.Ambitos["otro"] = "x"
	if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, ruta, d) {
		t.Fatal("ámbito adicional aceptado")
	}
	sinAnalisis := preparacion
	sinAnalisis.expediente.Fiscalizado = preparacion.expediente.Fiscalizado.Clonar()
	sinAnalisis.expediente.Fiscalizado.Analisis = nil
	if ambitosBolsaDelExpedienteDesarrollo(sinAnalisis.expediente, map[string]string{"categoria_ref": "", "unidad_ref": ""}) {
		t.Fatal("expediente sin análisis aceptado")
	}
}
