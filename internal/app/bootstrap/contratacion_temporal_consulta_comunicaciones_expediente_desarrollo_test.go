package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type autoridadConsultaComunicacionesPrueba struct{ err error }

func (a autoridadConsultaComunicacionesPrueba) ResolverContextoConsultaComunicacionesExpediente(context.Context) error {
	return a.err
}

type consultorComunicacionesPrueba struct{ err error }

func (c consultorComunicacionesPrueba) ConsultarComunicacionesExpediente(context.Context, ports.ConsultaComunicacionesExpediente) (ports.PaginaComunicacionesExpediente, error) {
	return ports.PaginaComunicacionesExpediente{}, c.err
}

func TestConsultaComunicacionesMontajePermisoV3YMaterialExactos(t *testing.T) {
	ruta := httpinterno.RutaConsultaComunicacionesExpediente
	c := ports.ConsultaComunicacionesExpediente{ExpedienteRef: "expediente:ct140:sintetico", Limite: 10}
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, ruta)
	ctx = context.WithValue(ctx, claveConsultaComunicacionesExpedienteDesarrollo{}, solicitudLigadaComunicacionesExpedienteDesarrollo{c, organizacionAltaContratacionTemporalDesarrollo})
	recurso, err := postgresct.RecursoConsultaComunicacionesExpediente(c, organizacionAltaContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatal(err)
	}
	d := dominiovec.DatosSolicitudAutorizacionLigadaV3{Finalidad: "gestionar_contratacion_temporal",
		ReferenciaMotivo: motivoConsultaComunicacionesExpedienteDesarrollo(),
		Accion:           postgresct.AccionConsultaComunicacionesExpediente, Recurso: recurso}
	if !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, ruta, d) ||
		!esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, ruta, nil)) {
		t.Fatal("lista CT140 sin ligadura o mTLS")
	}
	for _, otro := range []dominiovec.DatosSolicitudAutorizacionLigadaV3{
		{Finalidad: d.Finalidad, ReferenciaMotivo: d.ReferenciaMotivo, Accion: postgresct.AccionConsultaReciboRespuesta, Recurso: recurso},
		{Finalidad: d.Finalidad, ReferenciaMotivo: motivoConsultaReciboRespuestaDesarrollo(), Accion: d.Accion, Recurso: recurso},
		{Finalidad: d.Finalidad, ReferenciaMotivo: d.ReferenciaMotivo, Accion: d.Accion, Recurso: dominiovec.RecursoAutorizable{Referencia: "expediente:ajeno", ModuloID: recurso.ModuloID, Tipo: recurso.Tipo, Ambitos: recurso.Ambitos, Atributos: recurso.Atributos}},
	} {
		if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, ruta, otro) {
			t.Fatal("permiso ajeno admitido")
		}
	}
	if solicitudAutorizacionLlamamientoDesarrolloValida(ctx, httpinterno.RutaConsultaReciboRespuesta, d) {
		t.Fatal("la lista CT140 reutilizó el GET CT139")
	}
	concesion := concesionConsultaComunicacionesExpedienteDesarrollo()
	if concesion.Accion != postgresct.AccionConsultaComunicacionesExpediente ||
		concesion.TipoRecurso != postgresct.TipoRecursoConsultaComunicacionesExpediente ||
		!reflect.DeepEqual(concesion.CamposPermitidos, []string{"antecedente_tipo", "comunicacion_ref", "estado", "expediente_ref", "llamamiento_ref", "organizacion_ref", "recibo_antecedente_ref", "recibo_comunicacion_ref", "registrada_en", "version"}) ||
		len(concesion.Obligaciones) != 0 {
		t.Fatal("concesión CT140 divergente")
	}
	descriptor := descriptorMaterialConsultaComunicacionesExpedienteDesarrollo()
	if descriptor.Audiencia != postgresct.AudienciaConsultaComunicacionesExpediente ||
		descriptor.ProveedorNominal != proveedorMaterialContratacionTemporal {
		t.Fatal("audiencia de consumo CT140 divergente")
	}
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(
		append(descriptoresMaterialAutorizacionContratacionTemporalDesarrollo(), descriptor)); err != nil {
		t.Fatal("audiencia o derivación de clave CT140 colisiona con otras capacidades")
	}
	otroMaterial, err := postgresct.RecursoConsultaComunicacionesExpediente(ports.ConsultaComunicacionesExpediente{ExpedienteRef: c.ExpedienteRef, Limite: 20}, organizacionAltaContratacionTemporalDesarrollo)
	if err != nil || otroMaterial.Atributos["material_sha256"] == recurso.Atributos["material_sha256"] {
		t.Fatal("límite o cursor no ligado a la autorización")
	}
}

func TestConsultaComunicacionesMontaje401Y403FronteraSinQuery(t *testing.T) {
	ruta := httpinterno.RutaConsultaComunicacionesExpediente
	registro := &auditorConsultaReciboPrueba{}
	autoridad := &autoridadConsultasContratacionTemporalDesarrollo{sello: &selloConsultasContratacionTemporalDesarrollo{}, resolvedor: &resolvedorIdentidadDesarrollo{}}
	h, err := vechttp.NewHandlerSoloRutasExactas([]vechttp.RutaExacta{{Ruta: ruta, Manejador: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("denegación llegó al GET")
	})}}, autoridad, registro)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, ruta+"?expediente_ref=expediente:ct140:sintetico", nil)
	w := httptest.NewRecorder()
	autoridad.proteger(h).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || len(registro.ordenes) != 1 || registro.ordenes[0].Ruta != ruta ||
		registro.ordenes[0].Motivo != puertosvec.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida {
		t.Fatalf("401 sin auditoría minimizada: %d %#v", w.Code, registro.ordenes)
	}
	r = r.WithContext(context.WithValue(r.Context(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{ruta: ruta}))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || len(registro.ordenes) != 2 || registro.ordenes[1].Ruta != ruta ||
		registro.ordenes[1].Motivo != puertosvec.MotivoAuditoriaFronteraRutaExactaAccesoDenegado {
		t.Fatalf("403 sin auditoría minimizada: %d %#v", w.Code, registro.ordenes)
	}
}

func TestConsultaComunicacionesMontajeContextoSelladoYAutorizacionFresca(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ruta := httpinterno.RutaConsultaComunicacionesExpediente
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, ruta)
	a := &autorizacionComunicacionDesarrolloPrueba{}
	p := &proveedorConsultaComunicacionesExpedienteDesarrollo{soporte: soporte, autorizador: a, reloj: soporte.reloj}
	if err := p.ResolverContextoConsultaComunicacionesExpediente(context.Background()); !errors.Is(err, ports.ErrConsultaComunicacionesExpedienteDenegada) {
		t.Fatal("consulta sin mTLS admitida")
	}
	if err := p.ResolverContextoConsultaComunicacionesExpediente(ctx); err != nil {
		t.Fatalf("capacidad sellada rechazada: %v", err)
	}
	c := ports.ConsultaComunicacionesExpediente{ExpedienteRef: "expediente:ct140:sintetico", Limite: 10}
	for n := 1; n <= 2; n++ {
		if _, err := p.AutorizarConsultaComunicacionesExpediente(ctx, c); !errors.Is(err, ports.ErrConsultaComunicacionesExpedienteNoDisponible) ||
			a.llamadas != n || a.accion != postgresct.AccionConsultaComunicacionesExpediente ||
			a.recurso.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo {
			t.Fatal("consulta omitió permiso fresco o tomó ámbito de query")
		}
	}
	if _, err := p.AutorizarConsultaComunicacionesExpediente(context.Background(), c); !errors.Is(err, ports.ErrConsultaComunicacionesExpedienteDenegada) || a.llamadas != 2 {
		t.Fatal("referencia de expediente sustituyó identidad")
	}
}

func TestConsultaComunicacionesMontaje403CorrelacionYActor404503(t *testing.T) {
	ruta := httpinterno.RutaConsultaComunicacionesExpediente
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, ruta)
	for _, caso := range []struct {
		nombre string
		err    error
		estado int
	}{
		{"denegada", ports.ErrConsultaComunicacionesExpedienteDenegada, http.StatusForbidden},
		{"ausente", ports.ErrConsultaComunicacionesExpedienteNoEncontrado, http.StatusNotFound},
		{"cursor_derivado", ports.ErrConsultaComunicacionesExpedienteNoDisponible, http.StatusServiceUnavailable},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			get, err := httpinterno.NuevoManejadorConsultaComunicacionesExpediente(autoridadConsultaComunicacionesPrueba{}, consultorComunicacionesPrueba{caso.err})
			if err != nil {
				t.Fatal(err)
			}
			registro := &auditorConsultaReciboPrueba{}
			h := auditorConsultaCTDenegada{ruta: ruta, registrador: registro, soporte: soporte, siguiente: get}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta+"?expediente_ref=expediente:ct140:sintetico", nil).WithContext(ctx))
			if w.Code != caso.estado || strings.Contains(w.Body.String(), "comunicacion:sintetica") {
				t.Fatalf("estado/datos: %d %q", w.Code, w.Body.String())
			}
			if caso.estado == http.StatusForbidden {
				var cuerpo struct {
					Error struct {
						CorrelacionRef string `json:"correlacion_ref"`
					} `json:"error"`
				}
				if json.Unmarshal(w.Body.Bytes(), &cuerpo) != nil || len(registro.ordenes) != 1 ||
					registro.ordenes[0].Validar() != nil || registro.ordenes[0].CorrelacionRef != cuerpo.Error.CorrelacionRef ||
					registro.ordenes[0].ActorRef != principal.ID || registro.ordenes[0].Ruta != ruta {
					t.Fatalf("403/correlación/actor: %#v", registro.ordenes)
				}
			} else if len(registro.ordenes) != 0 {
				t.Fatalf("%d creó denegación de frontera", caso.estado)
			}
		})
	}
	get, err := httpinterno.NuevoManejadorConsultaComunicacionesExpediente(autoridadConsultaComunicacionesPrueba{}, consultorComunicacionesPrueba{ports.ErrConsultaComunicacionesExpedienteDenegada})
	if err != nil {
		t.Fatal(err)
	}
	registro := &auditorConsultaReciboPrueba{err: errors.New("auditoria caída")}
	h := auditorConsultaCTDenegada{ruta: ruta, registrador: registro, soporte: soporte, siguiente: get}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta+"?expediente_ref=expediente:ct140:sintetico", nil).WithContext(ctx))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "acceso_denegado") {
		t.Fatal("403 sin auditoría durable no cerró como 503")
	}
}

func TestConsultaComunicacionesMontaje401InteriorAuditaSinActor(t *testing.T) {
	ruta := httpinterno.RutaConsultaComunicacionesExpediente
	get, err := httpinterno.NuevoManejadorConsultaComunicacionesExpediente(
		autoridadConsultaComunicacionesPrueba{httpinterno.ErrContextoCanalCaducado},
		consultorComunicacionesPrueba{errors.New("no debe consultarse")})
	if err != nil {
		t.Fatal(err)
	}
	registro := &auditorConsultaReciboPrueba{}
	h := auditorConsultaCTDenegada{ruta: ruta, registrador: registro, siguiente: get}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta+"?expediente_ref=expediente:ct140:sintetico", nil))
	var cuerpo struct {
		Error struct {
			CorrelacionRef string `json:"correlacion_ref"`
		} `json:"error"`
	}
	if w.Code != http.StatusUnauthorized || json.Unmarshal(w.Body.Bytes(), &cuerpo) != nil ||
		len(registro.ordenes) != 1 || registro.ordenes[0].Validar() != nil ||
		registro.ordenes[0].Motivo != puertosvec.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida ||
		registro.ordenes[0].ActorRef != "" || registro.ordenes[0].CorrelacionRef != cuerpo.Error.CorrelacionRef {
		t.Fatalf("401 interno sin auditoría ligada: %d %#v", w.Code, registro.ordenes)
	}
}

func TestConsultaComunicacionesMontajeSeparaDenegacionDeDependenciaCaida(t *testing.T) {
	if !errors.Is(errorAutorizacionConsultaComunicacionesExpediente(dominiovec.ErrAutorizacionDenegada), ports.ErrConsultaComunicacionesExpedienteDenegada) ||
		!errors.Is(errorAutorizacionConsultaComunicacionesExpediente(errors.Join(dominiovec.ErrAutorizacionDenegada, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible)), ports.ErrConsultaComunicacionesExpedienteNoDisponible) {
		t.Fatal("dependencia caída confundida con falta de concesión")
	}
}
