package bootstrap

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type escenarioFirmasR5V2Prueba struct {
	soporte   *soporteAltaContratacionTemporalDesarrollo
	perfil    *perfilFijoCTDesarrollo
	principal dominiovec.Principal
	fuente    *fuenteNominalFirmasR5V2CTDesarrollo
}

func nuevoEscenarioFirmasR5V2Prueba(t *testing.T) escenarioFirmasR5V2Prueba {
	t.Helper()
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := s.reloj.Ahora()
	p, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoFirmasR5V2CTDesarrollo,
		[]string{httpinterno.RutaConsultaFirmasR5V2, httpinterno.RutaRecuperacionFirmasR5V2},
		func(actor, ref string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaFirmasR5V2CTDesarrollo(actor, ref, ahora)
		})
	if err != nil {
		t.Fatal(err)
	}
	p.metodo = http.MethodPost
	if err := s.registrarPerfilFijoCTDesarrollo(p); err != nil {
		t.Fatal(err)
	}
	p.contextoEsperadoRegistrado = p.contexto.Resultado
	p.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: p.contexto}
	asignaciones := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	asignaciones.asignaciones = map[string]instantaneaPublicadaDesarrollo{p.perfilRef(): {
		instantanea:    clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.plantilla),
		actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}
	return escenarioFirmasR5V2Prueba{soporte: s, perfil: p, principal: principal,
		fuente: &fuenteNominalFirmasR5V2CTDesarrollo{soporte: s, perfil: p, reloj: s.reloj, proceso: "vec-rrhh"}}
}

func (e escenarioFirmasR5V2Prueba) ctx(ruta, metodo string) context.Context {
	ahora := e.soporte.reloj.Ahora()
	return context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{},
		capacidadConsultaContratacionTemporalDesarrollo{
			sello: e.soporte.sello, ruta: ruta, metodo: metodo, principal: e.principal,
			certificadoVerificadoEn: ahora.Add(-time.Second), certificadoValidoHasta: ahora.Add(time.Hour),
			contextoOperacion: &contextoOperacionCTDesarrollo{},
		})
}

// El identificador del certificado mTLS no es la persona registrada. La
// fuente debe aceptar la sesión del perfil fijo y entregar como candidato la
// persona registrada, nunca el ID del certificado ni un dato de la petición.
func TestFirmasR5V2FuenteNominalUsaSesionRegistradaDelPerfil(t *testing.T) {
	e := nuevoEscenarioFirmasR5V2Prueba(t)
	registrada := e.perfil.contexto.Resultado.Contexto
	if registrada.PersonaRef == e.principal.ID || !strings.HasPrefix(registrada.PersonaRef, "per_") {
		t.Fatal("el escenario debe distinguir certificado y persona registrada")
	}
	for _, ruta := range []string{httpinterno.RutaConsultaFirmasR5V2, httpinterno.RutaRecuperacionFirmasR5V2} {
		ctx := e.ctx(ruta, http.MethodPost)
		actor, err := e.fuente.RevalidarContextoActorFirmaV2(ctx)
		if err != nil {
			t.Fatalf("%s: la sesión del perfil fijo se ha denegado: %v", ruta, err)
		}
		if actor.Resultado.Contexto.PerfilActivoRef != e.perfil.perfilRef() ||
			actor.CertificadoCanalSHA256 != e.principal.Attributes["certificate_sha256"] {
			t.Fatalf("%s: contexto o certificado ajenos", ruta)
		}
		c, err := e.fuente.ResolverContextoConsultaFirmasR5V2(ctx)
		if err != nil || c.FirmantePrincipalCandidatoRef != registrada.PersonaRef ||
			c.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
			t.Fatalf("%s: candidato u organización no proceden de la sesión registrada: %v", ruta, err)
		}
	}
}

func TestFirmasR5V2FuenteDeniegaOtroCanalYPerfilRevocado(t *testing.T) {
	e := nuevoEscenarioFirmasR5V2Prueba(t)
	for nombre, ctx := range map[string]context.Context{
		"metodo GET":     e.ctx(httpinterno.RutaConsultaFirmasR5V2, http.MethodGet),
		"otra ruta":      e.ctx(httpinterno.RutaConsultaFirmaDocumento, http.MethodPost),
		"sin capacidad":  context.Background(),
		"otro principal": context.WithValue(e.ctx(httpinterno.RutaConsultaFirmasR5V2, http.MethodPost), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{sello: e.soporte.sello, ruta: httpinterno.RutaConsultaFirmasR5V2, metodo: http.MethodPost}),
	} {
		if _, err := e.fuente.RevalidarContextoActorFirmaV2(ctx); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
			t.Fatalf("%s: canal no admitido aceptado: %v", nombre, err)
		}
	}
	// Una asignación publicada distinta de la plantilla cierra la ruta.
	asignaciones := e.soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	asignaciones.asignaciones = map[string]instantaneaPublicadaDesarrollo{}
	if _, err := e.fuente.RevalidarContextoActorFirmaV2(e.ctx(httpinterno.RutaConsultaFirmasR5V2, http.MethodPost)); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("perfil sin asignación publicada aceptado: %v", err)
	}
	// Sin perfil consumible la orden de auditoría sigue disponible.
	ctx := e.ctx(httpinterno.RutaRecuperacionFirmasR5V2, http.MethodPost)
	if _, err := e.fuente.CrearOrdenIntentoFirma(ctx, "aud_v3_i_"+strings.Repeat("a", 32), ports.AccionConsultarFirmasR5V2, "", dominiovec.ResultadoIntentoAuditoriaDenegado); err == nil {
		t.Fatal("orden de auditoría con la acción de otra ruta")
	}
}

func TestFirmasR5V2ContextoPropioExigePerfilPersonaYCuenta(t *testing.T) {
	e := nuevoEscenarioFirmasR5V2Prueba(t)
	propio := e.perfil.contexto.Resultado.Contexto
	if !contextoPropioFirmasR5V2CTDesarrollo(propio, e.perfil) {
		t.Fatal("el contexto del perfil fijo se ha rechazado")
	}
	for nombre, alterar := range map[string]func(*dominiovec.ContextoActor){
		"perfil":    func(c *dominiovec.ContextoActor) { c.PerfilActivoRef = "prf_otro" },
		"persona":   func(c *dominiovec.ContextoActor) { c.PersonaRef = e.principal.ID },
		"principal": func(c *dominiovec.ContextoActor) { c.Principal.ID = e.principal.ID },
		"cuenta":    func(c *dominiovec.ContextoActor) { c.Instantanea.CuentaRef = "cta_otra" },
	} {
		c := propio
		alterar(&c)
		if contextoPropioFirmasR5V2CTDesarrollo(c, e.perfil) {
			t.Fatalf("%s distinto aceptado", nombre)
		}
	}
	if contextoPropioFirmasR5V2CTDesarrollo(propio, nil) {
		t.Fatal("sin perfil aceptado")
	}
}

func TestFirmasR5V2PredicadoLigaRutaAccionYAmbito(t *testing.T) {
	recurso := dominiovec.RecursoAutorizable{Referencia: "expediente:ct:r5v2:001", ModuloID: ports.ModuloContratacion,
		Tipo: ports.TipoRecursoConsultaFirmasR5, Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo},
		Atributos: map[string]string{"material_sha256": strings.Repeat("a", 64)}}
	datos := func(accion string) dominiovec.DatosSolicitudAutorizacionLigadaV3 {
		return dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: accion, Recurso: recurso,
			Finalidad: ports.FinalidadFirmaDocumento, ReferenciaMotivo: motivoFirmasR5V2CTDesarrollo()}
	}
	if !solicitudAutorizacionFirmasR5V2CTDesarrolloValida(httpinterno.RutaConsultaFirmasR5V2, datos(ports.AccionConsultarFirmasR5V2)) ||
		!solicitudAutorizacionFirmasR5V2CTDesarrolloValida(httpinterno.RutaRecuperacionFirmasR5V2, datos(ports.AccionRecuperarFirmasR5V2)) {
		t.Fatal("solicitud exacta denegada")
	}
	if solicitudAutorizacionFirmasR5V2CTDesarrolloValida(httpinterno.RutaConsultaFirmasR5V2, datos(ports.AccionRecuperarFirmasR5V2)) ||
		solicitudAutorizacionFirmasR5V2CTDesarrolloValida(httpinterno.RutaRecuperacionFirmasR5V2, datos(ports.AccionConsultarFirmasR5V2)) {
		t.Fatal("la ruta acepta la acción de la otra")
	}
	for nombre, alterar := range map[string]func(*dominiovec.DatosSolicitudAutorizacionLigadaV3){
		"organizacion": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Ambitos = map[string]string{"organizacion_ref": "organizacion:otra"}
		},
		"ambito extra": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Ambitos = maps.Clone(recurso.Ambitos)
			d.Recurso.Ambitos["unidad_ref"] = "unidad:x"
		},
		"atributo extra": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos = maps.Clone(recurso.Atributos)
			d.Recurso.Atributos["estado"] = "x"
		},
		"tipo": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Tipo = ports.TipoRecursoFirmaDocumento
		},
		"finalidad": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Finalidad = "otra" },
		"motivo": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.ReferenciaMotivo = motivoConsultaFirmasDocumentoCTDesarrollo()
		},
	} {
		d := datos(ports.AccionConsultarFirmasR5V2)
		alterar(&d)
		if solicitudAutorizacionFirmasR5V2CTDesarrolloValida(httpinterno.RutaConsultaFirmasR5V2, d) {
			t.Fatalf("%s distinto aceptado", nombre)
		}
	}
}

func TestFirmasR5V2InstantaneaConcedeSoloLecturasExactas(t *testing.T) {
	i, err := nuevaInstantaneaFirmasR5V2CTDesarrollo("per_r5v2_prueba", "prf_r5v2_prueba", time.Now().UTC().Truncate(time.Microsecond))
	if err != nil || i.Validar() != nil {
		t.Fatal(err)
	}
	esperados := map[string][]string{
		ports.AccionConsultarFirmasR5V2: ports.CamposConsultaFirmasR5V2(),
		ports.AccionRecuperarFirmasR5V2: ports.CamposRecuperacionFirmasV2(),
	}
	if len(i.VersionRol.Concesiones) != len(esperados) {
		t.Fatal("concesiones de más o de menos")
	}
	for _, c := range i.VersionRol.Concesiones {
		campos, ok := esperados[c.Accion]
		if !ok || !slices.Equal(c.CamposPermitidos, campos) || len(c.Obligaciones) != 0 ||
			c.TipoRecurso != ports.TipoRecursoConsultaFirmasR5 || !slices.Equal(c.Finalidades, []string{ports.FinalidadFirmaDocumento}) {
			t.Fatalf("concesión %q distinta de la que exige el núcleo", c.Accion)
		}
	}
}

func TestFirmasR5V2RutasInventariadasYAudienciasGobernadas(t *testing.T) {
	inventario := inventarioRutasCTDesarrollo()
	for _, ruta := range []string{httpinterno.RutaConsultaFirmasR5V2, httpinterno.RutaRecuperacionFirmasR5V2} {
		m := inventario[ruta]
		if len(m) != 1 || m[0].metodo != http.MethodPost || m[0].guardia == "" {
			t.Fatalf("%s sin contrato nominal POST", ruta)
		}
		if !rutaContextoAutorizacionContratacionTemporalDesarrollo(ruta) || !rutaSesionConIndisponibilidadCTDesarrollo(ruta) {
			t.Fatalf("%s sin sesión operativa", ruta)
		}
		peticion, _ := http.NewRequest(http.MethodPost, "https://localhost"+ruta, nil)
		if !esRutaContratacionTemporalDesarrollo(peticion) {
			t.Fatalf("%s fuera de la frontera mTLS de CT", ruta)
		}
	}
	audiencias := audienciasConsumoGobiernoCTDesarrollo()
	if !slices.Contains(audiencias, ports.AudienciaConsultaFirmasR5V2) || !slices.Contains(audiencias, ports.AudienciaRecuperacionFirmasR5V2) {
		t.Fatal("audiencias R5 V2 sin clave de capacidad publicable")
	}
	e := nuevoEscenarioFirmasR5V2Prueba(t)
	for _, ruta := range []string{httpinterno.RutaConsultaFirmasR5V2, httpinterno.RutaRecuperacionFirmasR5V2} {
		if m, ok := e.soporte.motivoAutorizacionParaRuta(ruta); !ok || m != motivoFirmasR5V2CTDesarrollo() {
			t.Fatalf("%s sin motivo nominal", ruta)
		}
	}
	if _, ok := (&soporteAltaContratacionTemporalDesarrollo{}).motivoAutorizacionParaRuta(httpinterno.RutaConsultaFirmasR5V2); ok {
		t.Fatal("motivo resuelto sin perfil compuesto")
	}
}

func TestFirmasR5V2EmisorSoloAtiendeAccionesR5(t *testing.T) {
	e := &emisorFirmasR5V2CTDesarrollo{}
	if _, _, _, err := e.EmitirMaterialAutorizacionAtestadaV3(context.Background(), dominiovec.SolicitudAutorizacionLigadaV3{}, dominiovec.ResultadoContextoActorRegistradoV2{}); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatal("emisión sin solicitud válida")
	}
}

func TestFirmasR5V2SelectorApagadoNoCompone(t *testing.T) {
	t.Setenv(envCTFirmasR5V2Enabled, "")
	if r, err := nuevasRutasFirmasR5V2CTDesarrollo(config.Config{}, nil, nil, relojContratacionTemporalDesarrollo{}); r != nil || err != nil {
		t.Fatal("selector apagado compone rutas")
	}
	t.Setenv(envCTFirmasR5V2Enabled, "si")
	if _, err := nuevasRutasFirmasR5V2CTDesarrollo(config.Config{}, nil, nil, relojContratacionTemporalDesarrollo{}); err == nil {
		t.Fatal("valor de selector no admitido aceptado")
	}
	t.Setenv(envCTFirmasR5V2Enabled, "true")
	if _, err := nuevasRutasFirmasR5V2CTDesarrollo(config.Config{}, nil, nil, relojContratacionTemporalDesarrollo{}); err == nil {
		t.Fatal("selector sin doble llave de desarrollo aceptado")
	}
}
