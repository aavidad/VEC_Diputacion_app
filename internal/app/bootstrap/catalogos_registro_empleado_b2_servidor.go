package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	apppersonal "vec-diputacion-granada/internal/modules/personal/application"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Publicar y retirar entradas del catálogo de registro de empleado (régimen,
// modalidad...) desde vec-server reutiliza el manejador y el servicio de
// Personal. Cada acto es una decisión V3 nominal del perfil propio de este
// grupo, que no amplía la asignación ya publicada del perfil de Personal.
// Las dos operaciones son opcionales y van juntas: sin ellas en la
// configuración privada no hay perfil ni ruta montada (404) y el servidor
// arranca como antes.
const (
	claveCatalogoPublicarB2             = "personal_catalogo_publicar"
	claveCatalogoRetirarB2              = "personal_catalogo_retirar"
	grupoCatalogoEmpleadoB2             = "personal_catalogo"
	tipoEntradaCatalogoEmpleadoB2       = "entrada_catalogo_empleado_rrhh"
	finalidadGobiernoCatalogoEmpleadoB2 = "gobernar_catalogo_empleado"
	// La ruta vive bajo el prefijo CT para pasar por la misma frontera mTLS,
	// el mismo inventario y la misma auditoría que el resto del plan B2.
	rutaCatalogosRegistroEmpleadoB2 = "/api/vec/contratacion-temporal/incorporacion-personal-b2/catalogos-registro-empleado/v1"
	// La frontera de la ruta declara la publicación; la retirada usa el mismo
	// perfil y la autoridad B2 decide cada acto con su acción exacta.
	accionFronteraCatalogoEmpleadoB2 = personal.AccionPublicarCatalogoEmpleadoB2
)

func accionGobiernoCatalogoEmpleadoB2(accion string) bool {
	return accion == personal.AccionPublicarCatalogoEmpleadoB2 || accion == personal.AccionRetirarCatalogoEmpleadoB2
}

func accionCatalogoEmpleadoB2(operacion string) (string, bool) {
	switch operacion {
	case "consultar":
		return personal.AccionConsultarCatalogoEmpleadoB2, true
	case "publicar":
		return personal.AccionPublicarCatalogoEmpleadoB2, true
	case "retirar":
		return personal.AccionRetirarCatalogoEmpleadoB2, true
	}
	return "", false
}

// negativaGobiernoCatalogoEmpleadoB2 marca como denegada para Personal la
// negativa del punto de decisión al publicar o retirar, para que el manejador
// responda 403 auditado y no 503. La consulta conserva su traducción previa.
func negativaGobiernoCatalogoEmpleadoB2(accion string, e error) error {
	if e != nil && accionGobiernoCatalogoEmpleadoB2(accion) && (errors.Is(e, ct.ErrAutorizacionDenegada) ||
		errors.Is(e, core.ErrAutorizacionDenegada) || errors.Is(e, vp.ErrDenegacionExplicitaAutorizacionLigadaV3)) {
		return errors.Join(personal.ErrRegistroEmpleadoB2Denegado, e)
	}
	return e
}

// gobiernoCatalogoEmpleadoB2Coherente rechaza una configuración que habilite
// sólo una de las dos operaciones: comparten perfil y ruta.
func gobiernoCatalogoEmpleadoB2Coherente(c *archivoIncorporacionPersonalB2) bool {
	return c != nil && operacionConfiguradaB2(c, claveCatalogoPublicarB2) == operacionConfiguradaB2(c, claveCatalogoRetirarB2)
}

// autoridadCatalogosEmpleadoB2 entrega al manejador de Personal el actor del
// perfil nominal de gobierno y el organismo de la configuración del servidor.
// Publicar y retirar comparten perfil, así que el actor es el mismo para ambos.
type autoridadCatalogosEmpleadoB2 struct {
	autoridad *autoridadIncorporacionPersonalB2
}

func (a autoridadCatalogosEmpleadoB2) ResolverContextoRegistroEmpleadoB2(ctx context.Context) (core.ContextoActor, string, error) {
	if a.autoridad == nil || a.autoridad.organismoRef == "" {
		return core.ContextoActor{}, "", errors.New("vec: catálogo de registro B2 no disponible")
	}
	accion := personal.AccionPublicarCatalogoEmpleadoB2
	if ruta, ok := ctx.Value(claveRutaPeticionIncorporacionB2{}).(rutaPeticionIncorporacionB2); ok &&
		ruta.ruta == httpapi.RutaCatalogosRegistroEmpleadoB2 && ruta.metodo == http.MethodGet {
		accion = personal.AccionConsultarCatalogoEmpleadoB2
	}
	return contextoRegistroPersonalB2(ctx, a.autoridad, accion)
}

// auditorCatalogosEmpleadoB2 registra las negativas en la auditoría de frontera
// CT de vec-server, con la ruta CT por la que entró la petición.
type auditorCatalogosEmpleadoB2 struct {
	registrador vp.RegistradorAuditoriaFronteraRutaExacta
	ruta        string
}

func (a auditorCatalogosEmpleadoB2) RegistrarDenegacionRegistroEmpleadoB2(ctx context.Context, d httpapi.DenegacionRegistroEmpleadoB2) error {
	if dependenciaEsNulaContratacionTemporalDesarrollo(a.registrador) || ctx == nil || ctx.Err() != nil {
		return errors.New("vec: auditoría de frontera B2 no disponible")
	}
	ruta := a.ruta
	if ruta == "" {
		ruta = d.Ruta
		if ruta == "" {
			ruta = rutaCatalogosRegistroEmpleadoB2
		}
	}
	superficie := vp.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal
	if ruta != rutaCatalogosRegistroEmpleadoB2 {
		superficie = vp.SuperficieAuditoriaFronteraRutaExactaPersonal
	}
	orden := vp.OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: d.CorrelacionRef, Motivo: vp.MotivoAuditoriaFronteraRutaExacta(d.Motivo),
		Superficie: superficie, Ruta: ruta, ActorRef: d.ActorRef}
	if e := orden.Validar(); e != nil {
		return e
	}
	return a.registrador.RegistrarAuditoriaFronteraRutaExacta(ctx, orden)
}

// El GET usa el LOGIN de consultas y el POST el de actos. Ambos servicios
// comparten el contrato y la autoridad nominal de Personal.
type operadorCatalogosPersonalB2 struct {
	consulta *apppersonal.ServicioCatalogosRegistroEmpleadoB2
	gobierno *apppersonal.ServicioCatalogosRegistroEmpleadoB2
}

func (o operadorCatalogosPersonalB2) Consultar(ctx context.Context, s personal.SolicitudConsultaCatalogoEmpleadoB2) (pp.ResultadoConsultaCatalogoEmpleadoB2, error) {
	return o.consulta.Consultar(ctx, s)
}

func (o operadorCatalogosPersonalB2) Cambiar(ctx context.Context, s personal.SolicitudCambioCatalogoEmpleadoB2) (pp.ResultadoCambioCatalogoEmpleadoB2, error) {
	if o.gobierno == nil {
		return pp.ResultadoCambioCatalogoEmpleadoB2{}, personal.ErrRegistroEmpleadoB2Denegado
	}
	return o.gobierno.Cambiar(ctx, s)
}

// rutaPersonalCatalogosEmpleadoB2 publica la ruta que consume la ficha RRHH.
// Conserva el manejador común y selecciona el perfil nominal por verbo.
func rutaPersonalCatalogosEmpleadoB2(consulta, gobierno *apppersonal.ServicioCatalogosRegistroEmpleadoB2,
	autoridad *autoridadIncorporacionPersonalB2, registrador vp.RegistradorAuditoriaFronteraRutaExacta,
	soporte *soporteAltaContratacionTemporalDesarrollo, fronteras catalogoFronterasComunDesarrollo) (httpapi.RutaExacta, error) {
	if consulta == nil || autoridad == nil || dependenciaEsNulaContratacionTemporalDesarrollo(registrador) {
		return httpapi.RutaExacta{}, errors.New("vec: catálogo de registro B2 incompleto")
	}
	autorizar := autoridadCatalogosEmpleadoB2{autoridad: autoridad}
	auditar := auditorCatalogosEmpleadoB2{registrador: registrador, ruta: httpapi.RutaCatalogosRegistroEmpleadoB2}
	h, err := httpapi.NewHandlerCatalogosRegistroEmpleadoB2(autorizar, operadorCatalogosPersonalB2{consulta, gobierno}, auditar)
	if err != nil {
		return httpapi.RutaExacta{}, err
	}
	adaptado := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || r.URL.Path != httpapi.RutaCatalogosRegistroEmpleadoB2 || r.URL.RawPath != "" {
			http.NotFound(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
	return httpapi.RutaExacta{Ruta: httpapi.RutaCatalogosRegistroEmpleadoB2,
		Manejador: ligarContextoIncorporacionPersonalB2(adaptado, soporte, fronteras)}, nil
}

// rutaCatalogosEmpleadoB2 monta el manejador de Personal en la ruta CT. El
// manejador sólo admite su ruta propia: la petición se le entrega con esa ruta
// tras comprobar que llegó exactamente por la ruta CT, sin consulta añadida.
func rutaCatalogosEmpleadoB2(servicio *apppersonal.ServicioCatalogosRegistroEmpleadoB2, autoridad *autoridadIncorporacionPersonalB2,
	registrador vp.RegistradorAuditoriaFronteraRutaExacta, soporte *soporteAltaContratacionTemporalDesarrollo, fronteras catalogoFronterasComunDesarrollo) (httpapi.RutaExacta, error) {
	if servicio == nil || autoridad == nil || dependenciaEsNulaContratacionTemporalDesarrollo(registrador) {
		return httpapi.RutaExacta{}, errors.New("vec: catálogo de registro B2 incompleto")
	}
	h, e := httpapi.NewHandlerCatalogosRegistroEmpleadoB2(autoridadCatalogosEmpleadoB2{autoridad}, servicio, auditorCatalogosEmpleadoB2{registrador: registrador, ruta: rutaCatalogosRegistroEmpleadoB2})
	if e != nil {
		return httpapi.RutaExacta{}, e
	}
	adaptado := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || r.URL.Path != rutaCatalogosRegistroEmpleadoB2 || r.URL.RawPath != "" {
			http.NotFound(w, r)
			return
		}
		copia := r.Clone(r.Context())
		copia.URL = &url.URL{Path: httpapi.RutaCatalogosRegistroEmpleadoB2, RawQuery: r.URL.RawQuery}
		h.ServeHTTP(w, copia)
	})
	return httpapi.RutaExacta{Ruta: rutaCatalogosRegistroEmpleadoB2, Manejador: ligarContextoIncorporacionPersonalB2(adaptado, soporte, fronteras)}, nil
}
