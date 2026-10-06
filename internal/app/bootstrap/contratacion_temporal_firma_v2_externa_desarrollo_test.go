package bootstrap

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Escenario de la vía externa: el perfil fijo de la PR 1 registrado para la
// ruta de registro externo, con su asignación publicada.
func nuevoEscenarioFirmaExternaV2Prueba(t *testing.T) escenarioFirmasR5V2Prueba {
	t.Helper()
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := s.reloj.Ahora()
	p, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoFirmaExternaV2CTDesarrollo,
		[]string{httpinterno.RutaRegistroFirmaExterna},
		func(actor, ref string) (dominiovec.InstantaneaAutorizacion, error) {
			return nuevaInstantaneaFirmaExternaV2CTDesarrollo(actor, ref, ahora)
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
	return escenarioFirmasR5V2Prueba{soporte: s, perfil: p, principal: principal}
}

func recursoFirmaExternaV2Prueba(atributos map[string]string) dominiovec.RecursoAutorizable {
	return dominiovec.RecursoAutorizable{Referencia: ports.PrefijoRecursoFirmaExterna + "clave-firma-externa-0001",
		ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoFirmaExterna,
		Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo}, Atributos: atributos}
}

// El PDP de CT sólo admite en la ruta las decisiones interior y exterior de
// la firma externa y la consulta R5 V2 previa, con el motivo de la firma V2,
// la organización como único ámbito y la forma exacta del recurso.
func TestFirmaExternaV2SolicitudesAdmitidasPorElPDP(t *testing.T) {
	h := strings.Repeat("a", 64)
	datos := func(accion string, r dominiovec.RecursoAutorizable) dominiovec.DatosSolicitudAutorizacionLigadaV3 {
		return dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: accion, Recurso: r, Finalidad: ports.FinalidadFirmaDocumento,
			ReferenciaMotivo: motivoFirmaV2CTDesarrollo()}
	}
	interior := recursoFirmaExternaV2Prueba(map[string]string{"material_sha256": h, "descriptor_firma_sha256": h})
	exterior := recursoFirmaExternaV2Prueba(map[string]string{"material_sha256": h, "plan_firma_sha256": h})
	consulta := dominiovec.RecursoAutorizable{Referencia: "exp:prueba", ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoConsultaFirmasR5,
		Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo}, Atributos: map[string]string{"material_sha256": h}}
	for caso, d := range map[string]dominiovec.DatosSolicitudAutorizacionLigadaV3{
		"interior": datos(ports.AccionRegistrarFirmaExterna, interior),
		"exterior": datos(ports.AccionRegistrarFirmaExterna, exterior),
		"consulta": datos(ports.AccionConsultarFirmasR5V2, consulta),
	} {
		if !solicitudAutorizacionFirmaExternaV2CTDesarrolloValida(d) {
			t.Fatalf("%s denegada", caso)
		}
	}
	conUnidad := recursoFirmaExternaV2Prueba(maps.Clone(interior.Atributos))
	conUnidad.Ambitos["unidad_ref"] = "unidad:rrhh"
	otraOrg := recursoFirmaExternaV2Prueba(maps.Clone(interior.Atributos))
	otraOrg.Ambitos["organizacion_ref"] = "org_otra"
	vec := interior
	vec.Referencia = ports.PrefijoRecursoFirmaVec + "clave-firma-externa-0001"
	tipoVec := interior
	tipoVec.Tipo = ports.TipoRecursoFirmaVec
	otroMotivo := datos(ports.AccionRegistrarFirmaExterna, interior)
	otroMotivo.ReferenciaMotivo = motivoFirmasR5V2CTDesarrollo()
	otraFinalidad := datos(ports.AccionRegistrarFirmaExterna, interior)
	otraFinalidad.Finalidad = "otra"
	for caso, d := range map[string]dominiovec.DatosSolicitudAutorizacionLigadaV3{
		"firma_vec":         datos(ports.AccionRegistrarFirmaVec, interior),
		"recuperacion":      datos(ports.AccionRecuperarFirmasR5V2, consulta),
		"con_unidad":        datos(ports.AccionRegistrarFirmaExterna, conUnidad),
		"otra_organizacion": datos(ports.AccionRegistrarFirmaExterna, otraOrg),
		"prefijo_vec":       datos(ports.AccionRegistrarFirmaExterna, vec),
		"tipo_vec":          datos(ports.AccionRegistrarFirmaExterna, tipoVec),
		"descriptor_y_plan": datos(ports.AccionRegistrarFirmaExterna, recursoFirmaExternaV2Prueba(map[string]string{"material_sha256": h, "descriptor_firma_sha256": h, "plan_firma_sha256": h})),
		"sin_descriptor":    datos(ports.AccionRegistrarFirmaExterna, recursoFirmaExternaV2Prueba(map[string]string{"material_sha256": h, "otra": h})),
		"sin_material":      datos(ports.AccionRegistrarFirmaExterna, recursoFirmaExternaV2Prueba(map[string]string{"descriptor_firma_sha256": h, "otra": h})),
		"otro_motivo":       otroMotivo,
		"otra_finalidad":    otraFinalidad,
		"consulta_con_mas":  datos(ports.AccionConsultarFirmasR5V2, dominiovec.RecursoAutorizable{Referencia: "exp:prueba", ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoConsultaFirmasR5, Ambitos: consulta.Ambitos, Atributos: map[string]string{"material_sha256": h, "otra": h}}),
	} {
		if solicitudAutorizacionFirmaExternaV2CTDesarrolloValida(d) {
			t.Fatalf("%s admitida", caso)
		}
	}
}

// La fuente de ámbitos responde sólo por el perfil fijo y su persona, con la
// asignación publicada; sin asignación consumible, no hay ámbitos.
func TestFirmaExternaV2AmbitosDelPerfilFijo(t *testing.T) {
	e := nuevoEscenarioFirmaExternaV2Prueba(t)
	a := ambitosPerfilFijoCTDesarrollo{soporte: e.soporte, perfil: e.perfil}
	persona := e.perfil.plantilla.AsignacionPerfil.PrincipalID
	i, err := a.ObtenerInstantaneaAutorizacion(context.Background(), persona, e.perfil.perfilRef())
	if err != nil || len(i.AsignacionPerfil.Ambitos) != 1 || i.AsignacionPerfil.Ambitos[0].Clave != "organizacion_ref" {
		t.Fatalf("ámbitos del perfil fijo no leídos: %v", err)
	}
	for caso, x := range map[string][2]string{"otro_perfil": {persona, "prf_otro_perfil_000000000"}, "otra_persona": {"per_otra_persona_00000000", e.perfil.perfilRef()}} {
		if _, err := a.ObtenerInstantaneaAutorizacion(context.Background(), x[0], x[1]); !errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) {
			t.Fatalf("%s: %v", caso, err)
		}
	}
	e.soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba).asignaciones = nil
	if _, err := a.ObtenerInstantaneaAutorizacion(context.Background(), persona, e.perfil.perfilRef()); !errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) {
		t.Fatalf("sin asignación publicada: %v", err)
	}
}

// La fuente nominal de la vía externa sólo atiende su ruta, con el motivo de
// la firma V2, y el emisor se arma con los dos emisores de material.
func TestFirmaExternaV2FuenteYEmisor(t *testing.T) {
	e := nuevoEscenarioFirmaExternaV2Prueba(t)
	m := &emisorMaterialRenovableCTDesarrollo{}
	if _, _, err := nuevoEmisorFirmaExternaV2CTDesarrollo(e.soporte, e.perfil, m, nil, e.soporte.reloj, "vec-rrhh"); err == nil {
		t.Fatal("emisor sin el material de la firma externa")
	}
	emisor, fuente, err := nuevoEmisorFirmaExternaV2CTDesarrollo(e.soporte, e.perfil, m, m, e.soporte.reloj, "vec-rrhh")
	if err != nil || emisor == nil || fuente.motivo != motivoFirmaV2CTDesarrollo() {
		t.Fatalf("emisor de la vía externa: %v", err)
	}
	actor, err := fuente.RevalidarContextoActorFirmaV2(e.ctx(httpinterno.RutaRegistroFirmaExterna, http.MethodPost))
	if err != nil || actor.Resultado.Contexto.PerfilActivoRef != e.perfil.perfilRef() {
		t.Fatalf("la sesión del perfil de la vía externa se ha denegado: %v", err)
	}
	for _, ruta := range []string{httpinterno.RutaRegistroFirmaVec, httpinterno.RutaConsultaFirmasR5V2} {
		if _, err := fuente.RevalidarContextoActorFirmaV2(e.ctx(ruta, http.MethodPost)); err == nil {
			t.Fatalf("%s atendida por la fuente de la vía externa", ruta)
		}
	}
	if _, err := fuente.RevalidarContextoActorFirmaV2(e.ctx(httpinterno.RutaRegistroFirmaExterna, http.MethodGet)); err == nil {
		t.Fatal("GET atendido")
	}
}

// El PDP de CT consume el perfil fijo en la ruta externa sólo con una
// solicitud que el predicado admite, y el soporte da a la ruta el motivo de
// la firma V2.
func TestFirmaExternaV2PDPConsumePerfilFijoConSolicitudExacta(t *testing.T) {
	e := nuevoEscenarioFirmaExternaV2Prueba(t)
	h := strings.Repeat("a", 64)
	ctx := e.ctx(httpinterno.RutaRegistroFirmaExterna, http.MethodPost)
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: ports.AccionRegistrarFirmaExterna,
		Recurso:   recursoFirmaExternaV2Prueba(map[string]string{"material_sha256": h, "plan_firma_sha256": h}),
		Finalidad: ports.FinalidadFirmaDocumento, ReferenciaMotivo: motivoFirmaV2CTDesarrollo()}
	con := func(d dominiovec.DatosSolicitudAutorizacionLigadaV3) context.Context {
		return context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, d)
	}
	i, ok := e.soporte.instantaneaPerfilFijoParaContexto(con(datos), httpinterno.RutaRegistroFirmaExterna, e.perfil)
	if !ok || i.AsignacionPerfil.PerfilActivoRef != e.perfil.perfilRef() {
		t.Fatal("la solicitud exacta no consume el perfil fijo")
	}
	datos.Accion = ports.AccionRegistrarFirmaVec
	if _, ok := e.soporte.instantaneaPerfilFijoParaContexto(con(datos), httpinterno.RutaRegistroFirmaExterna, e.perfil); ok {
		t.Fatal("la ruta externa consume con la acción de la firma VEC")
	}
	if _, ok := e.soporte.instantaneaPerfilFijoParaContexto(ctx, httpinterno.RutaRegistroFirmaExterna, e.perfil); ok {
		t.Fatal("consumo sin solicitud en el contexto")
	}
	if m, ok := e.soporte.motivoAutorizacionParaContexto(ctx, httpinterno.RutaRegistroFirmaExterna); !ok || m != motivoFirmaV2CTDesarrollo() {
		t.Fatal("la ruta externa no tiene el motivo de la firma V2")
	}
}
