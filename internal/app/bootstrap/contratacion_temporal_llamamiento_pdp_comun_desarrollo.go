package bootstrap

import (
	"net/http"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctpostgres "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Llamamiento en la autoridad común (PDP compartido con Bolsa).
//
// Con Bolsa B-BACK compuesta, el autorizador de Contratación delega en el PDP
// común, que solo decide acciones declaradas para la frontera exacta de la
// petición (ruta, método y perfil). El llamamiento no estaba declarado y toda
// su cadena respondía 403. Cada ruta del llamamiento tiene aquí su frontera,
// solo para el perfil CT, y la lista cerrada de acciones que se deciden en
// ella. La política es la misma que decide sin B-BACK; el PDP común añade la
// comprobación de frontera y perfil, no concede nada nuevo.
type fronteraLlamamientoComunDesarrollo struct {
	clave, ruta, metodo string
	// accion es la capacidad de la frontera; adicionales son las demás
	// acciones que la misma petición decide (lecturas ligadas o efectos).
	accion      string
	adicionales []string
}

func fronterasLlamamientoComunDesarrollo() []fronteraLlamamientoComunDesarrollo {
	return []fronteraLlamamientoComunDesarrollo{
		{clave: "ct-llamamiento-seleccionar", ruta: cthttp.RutaSeleccionLlamamiento, metodo: http.MethodPost,
			accion:      accionConsultarLlamamientoDesarrollo,
			adicionales: append([]string{ctports.AccionReanudacionSeleccionLlamamiento, ctports.AccionReanudacionSolicitudLlamamiento}, accionesBolsaLlamamientoSeleccionDesarrollo()...)},
		{clave: "ct-llamamiento-comunicacion-registrar", ruta: cthttp.RutaRegistroComunicacionLlamamiento, metodo: http.MethodPost,
			accion:      ctpostgres.AccionRegistroComunicacionLlamamiento,
			adicionales: []string{ctapplication.AccionDespacharCorreoLlamamiento, ctapplication.AccionRegistrarResultadoCorreoLlamamiento}},
		{clave: "ct-llamamiento-comunicaciones-consultar", ruta: cthttp.RutaConsultaComunicacionesExpediente, metodo: http.MethodGet,
			accion: ctpostgres.AccionConsultaComunicacionesExpediente},
		{clave: "ct-llamamiento-respuesta-registrar", ruta: cthttp.RutaRegistroRespuestaRecibida, metodo: http.MethodPost,
			accion: ctpostgres.AccionRegistroRespuestaRecibida},
		{clave: "ct-llamamiento-recibo-respuesta-consultar", ruta: cthttp.RutaConsultaReciboRespuesta, metodo: http.MethodGet,
			accion: ctpostgres.AccionConsultaReciboRespuesta},
		{clave: "ct-llamamiento-resolver", ruta: cthttp.RutaResolucionComunicacionLlamamiento, metodo: http.MethodPost,
			accion:      ctpostgres.AccionResolucionManualLlamamiento,
			adicionales: append([]string{ctpostgres.AccionConsultaJustificanteRespuestaRecibida}, accionesBolsaLlamamientoResolucionDesarrollo()...)},
		{clave: "ct-llamamiento-continuar", ruta: cthttp.RutaContinuacionLlamamiento, metodo: http.MethodPost,
			accion:      ctpostgres.AccionContinuacionLlamamiento,
			adicionales: append([]string{ctpostgres.AccionConsultaJustificanteRespuestaRecibida}, accionesBolsaLlamamientoContinuacionDesarrollo()...)},
		{clave: "ct-llamamiento-plazo-evento", ruta: cthttp.RutaEventoPlazoLlamamiento, metodo: http.MethodPost,
			accion: ctpostgres.AccionResolucionManualLlamamiento},
		{clave: "ct-formalizacion-proponer", ruta: cthttp.RutaPropuestaFormalizacion, metodo: http.MethodPost,
			accion: ctpostgres.AccionPropuestaFormalizacion},
		{clave: "ct-formalizacion-resolver", ruta: cthttp.RutaResolucionFormalizacion, metodo: http.MethodPost,
			accion: ctpostgres.AccionResolucionFormalizacion},
	}
}

// Efectos en Bolsa que hoy pide la raíz de Contratación dentro de la misma
// petición del llamamiento. Se declaran aparte para retirarlos en un solo
// sitio cuando Contratación consuma el llamamiento de Bolsa y sea Bolsa, con
// su propia autoridad, la que decida estas acciones.
func accionesBolsaLlamamientoSeleccionDesarrollo() []string {
	return []string{puertosbolsa.AccionPrepararOrdenDesarrollo, puertosbolsa.AccionAbrirLlamamientoDesarrollo}
}

func accionesBolsaLlamamientoResolucionDesarrollo() []string {
	return []string{puertosbolsa.AccionAceptarLlamamientoRRHHDesarrollo, puertosbolsa.AccionRenunciarLlamamientoRRHHDesarrollo}
}

func accionesBolsaLlamamientoContinuacionDesarrollo() []string {
	return []string{puertosbolsa.AccionRenunciarLlamamientoRRHHDesarrollo, puertosbolsa.AccionAbrirSiguienteLlamamientoDesarrollo}
}

// descriptoresFronterasLlamamientoDesarrollo declara las fronteras del
// llamamiento solo para el perfil CT dinámico, que es el que usan estas rutas.
func descriptoresFronterasLlamamientoDesarrollo(perfilCT string) []descriptorFronteraComunDesarrollo {
	fronteras := fronterasLlamamientoComunDesarrollo()
	descriptores := make([]descriptorFronteraComunDesarrollo, 0, len(fronteras))
	for _, f := range fronteras {
		d := fronteraContratacionTemporalDesarrollo(f.clave, f.accion, f.ruta, []string{perfilCT})
		d.Metodo = f.metodo
		descriptores = append(descriptores, d)
	}
	return descriptores
}

// descriptoresAutorizacionAdicionalesLlamamientoDesarrollo liga cada acción
// adicional a su frontera y a la política CT. La acción principal de cada
// frontera ya la declara el recorrido general de las fronteras CT.
func descriptoresAutorizacionAdicionalesLlamamientoDesarrollo(
	politica politicaAutorizacionSolicitudLigadaV3Desarrollo,
) []descriptorAutorizacionComunDesarrollo {
	var descriptores []descriptorAutorizacionComunDesarrollo
	for _, f := range fronterasLlamamientoComunDesarrollo() {
		for _, accion := range f.adicionales {
			descriptores = append(descriptores, descriptorAutorizacionComunDesarrollo{
				Accion: accion, ClavePolitica: clavePoliticaContratacionTemporalDesarrollo,
				ClaveCapacidad: f.accion, Fronteras: []string{f.clave}, Politica: politica,
			})
		}
	}
	return descriptores
}
