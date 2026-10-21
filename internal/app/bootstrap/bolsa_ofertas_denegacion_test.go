package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	confianzavec "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// autorizadorOfertasDenegantePrueba responde como el punto de decisión real
// ante un perfil sin la concesión pedida: una decisión V3 negativa ligada a la
// solicitud, sin confirmación de concesión, y la denegación explícita
// registrada. La instantánea es la del actor de la sesión de prueba.
type autorizadorOfertasDenegantePrueba struct {
	t           *testing.T
	instantanea vecdomain.InstantaneaAutorizacion
	llamadas    int
	// caida, si no es nil, sustituye la denegación por una avería.
	caida error
}

func (a *autorizadorOfertasDenegantePrueba) ExigirSolicitudLigadaV3(_ context.Context, s vecdomain.SolicitudAutorizacionLigadaV3, r vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	a.llamadas++
	if a.caida != nil {
		return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, a.caida
	}
	datos, err := s.Datos()
	if err != nil {
		a.t.Fatal(err)
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		a.t.Fatal(err)
	}
	// La decisión se emite dentro de la sesión y tras resolver el contexto.
	emitida := vinculo.SesionRevalidadaEn
	for _, t := range []time.Time{r.ResueltoEnAutoritativo, r.Contexto.ResueltoEn} {
		if t.After(emitida) {
			emitida = t
		}
	}
	evidencia, err := vecdomain.NuevaEvidenciaEvaluacionAutorizacionV3(s, a.instantanea, "dec_0123456789abcdef0123456789abcdef", emitida, emitida.Add(30*time.Second))
	if err != nil {
		a.t.Fatalf("evidencia de denegación: %v", err)
	}
	decision, err := vecdomain.NuevaDecisionAutorizacionLigadaV3(s, evidencia)
	if err != nil {
		a.t.Fatalf("decisión de denegación: %v", err)
	}
	if concedida, _, err := decision.Resultado(); err != nil || concedida {
		a.t.Fatalf("la instantánea de prueba concede %s: %v", datos.Accion, err)
	}
	return decision, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
		errors.Join(vecdomain.ErrAutorizacionDenegada, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3)
}

// operadorOfertasNoLlamadoPrueba cuenta llamadas: una petición denegada nunca
// llega a consultar ni a publicar la política.
type operadorOfertasNoLlamadoPrueba struct{ llamadas int }

func (o *operadorOfertasNoLlamadoPrueba) ConsultarAutorizada(context.Context, puertosbolsa.ConsultaPoliticaOfertasAutorizada) (puertosbolsa.VersionPoliticaOfertas, error) {
	o.llamadas++
	return puertosbolsa.VersionPoliticaOfertas{}, nil
}

func (o *operadorOfertasNoLlamadoPrueba) Publicar(context.Context, puertosbolsa.ComandoPublicarPoliticaOfertas) (puertosbolsa.VersionPoliticaOfertas, error) {
	o.llamadas++
	return puertosbolsa.VersionPoliticaOfertas{}, nil
}

// Ruta final de la política de ofertas: el preparador de la composición, con
// su emisor renovable real, recibe una denegación explícita registrada del
// punto de decisión. GET y POST responden 403 «acceso_denegado», no 503, y
// el operador no se llama; una avería de la fuente sigue siendo 503.
func TestPoliticaOfertasDenegacionExplicitaEsAccesoDenegado(t *testing.T) {
	base, fuente, ctx := preparadorLecturasAmbitoF160(t)
	// La política se publica sobre una bolsa constituida y vigente.
	base.vigentes.(*bolsasVigentesFalsas).constituirEn("bolsa:prueba", puertosbolsa.AmbitoBolsa{UnidadRef: "unidad:prueba", AmbitoRef: "ambito:prueba"})
	autorizador := &autorizadorOfertasDenegantePrueba{t: t, instantanea: fuente.i}
	emisorRenovable := &emisorMaterialRenovableCTDesarrollo{autoridad: autorizador, proveedor: &proveedorMaterialAltaContratacionTemporalDesarrollo{
		atestador: &aplicacionvec.ServicioAtestacionesAutorizacionV3{},
		confianza: &confianzavec.ServicioConfianzaAtestacionAutorizacionV3{},
		emisor:    &confianzavec.EmisorCapacidadesAtestacionAutorizacionV3{},
	}}
	preparador := &preparadorPoliticaOfertasBolsaDesarrollo{base: base, emisor: &emisorBorradorLlamamientoDesarrollo{
		politicaOfertas: emisorRenovable, consultaPoliticaOfertas: emisorRenovable}}
	operador := &operadorOfertasNoLlamadoPrueba{}
	h, err := bolsahttp.NuevoHandlerPoliticaOfertas(preparador, operador)
	if err != nil {
		t.Fatal(err)
	}
	politica := `{"bolsa_ref":"bolsa:prueba","version_esperada":0,"clave_idempotencia":"clave-0001","politica":` +
		`{"plazo":{"inicio":"notificacion","unidad":"dias_habiles","cantidad":2,"computo":"administrativo","municipio_sede":"18087"},` +
		`"adjudicacion":{"criterio":"orden_vigente","elegibilidad":"disposicion_en_plazo"},` +
		`"no_cubierta":{"accion":"llamamiento_directo","condicion":"sin_disposiciones_elegibles"}}}`
	for _, caso := range []struct {
		metodo, ruta, cuerpo string
		caida                error
		estado               int
		codigo               string
	}{
		{http.MethodGet, bolsahttp.RutaPoliticaOfertas + "?bolsa_ref=bolsa:prueba", "", nil, http.StatusForbidden, "acceso_denegado"},
		{http.MethodPost, bolsahttp.RutaPoliticaOfertas, politica, nil, http.StatusForbidden, "acceso_denegado"},
		// Contraste: la fuente de autorización caída sigue siendo 503.
		{http.MethodGet, bolsahttp.RutaPoliticaOfertas + "?bolsa_ref=bolsa:prueba", "", vecports.ErrFuenteAutorizacionNoDisponible, http.StatusServiceUnavailable, "servicio_no_disponible"},
		{http.MethodPost, bolsahttp.RutaPoliticaOfertas, politica, vecports.ErrFuenteAutorizacionNoDisponible, http.StatusServiceUnavailable, "servicio_no_disponible"},
	} {
		autorizador.caida = caso.caida
		antes := autorizador.llamadas
		r := httptest.NewRequestWithContext(ctx, caso.metodo, caso.ruta, strings.NewReader(caso.cuerpo))
		if caso.metodo == http.MethodPost {
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Accept", "application/json")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != caso.estado || !strings.Contains(w.Body.String(), `"codigo":"`+caso.codigo+`"`) ||
			autorizador.llamadas != antes+1 || operador.llamadas != 0 {
			t.Fatalf("%s (caída=%v): estado=%d cuerpo=%s decisiones=%d operador=%d",
				caso.metodo, caso.caida, w.Code, w.Body.String(), autorizador.llamadas-antes, operador.llamadas)
		}
	}
}
