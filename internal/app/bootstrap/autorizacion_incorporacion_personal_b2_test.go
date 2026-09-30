package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestIncorporacionB2PerfilPersonalConservaActorYRevocaLectura(t *testing.T) {
	base, ctx, _, publicador := escenarioNominalIncorporacion(t)
	refs := base.referencias
	refs.PerfilV3Ref = base.soporte.contexto.Resultado.Contexto.PerfilActivoRef
	c := &archivoIncorporacionPersonalB2{Protocolo: "personal_b2_v1", OrganismoRef: "organismo:prueba", CatalogoRPTID: "categorias_rpt", ModuloRPTID: "personal"}
	if e := extenderPerfilesNominalesB2(base.nominales, refs, c, base.reloj.Ahora()); e != nil {
		t.Fatal(e)
	}
	for _, p := range base.nominales.todos() {
		p.contextoEsperadoRegistrado = p.contexto.Resultado
		p.sesionOperativa = &sesionNominalIncorporacionPrueba{contexto: p.contexto}
		publicador.publicadas[p.perfilRef()] = instantaneaPublicadaDesarrollo{instantanea: p.plantilla, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}
	}
	autoridad := &autoridadIncorporacionPersonalB2{perfiles: base.nominales, reloj: base.reloj}
	ctx = context.WithValue(ctx, claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: "POST", ruta: "/api/interno/contratacion-temporal/incorporacion-personal-b2/confirmar/v1"})
	primero, e := autoridad.ActorPreparacionPlanB2(ctx)
	if e != nil {
		t.Fatal(e)
	}
	for _, accion := range []string{"personal.plan_incorporacion_ct.consultar", "personal.plan_incorporacion_ct.ejecutar", "personal.plan_incorporacion_ct.confirmar", personal.AccionAltaEmpleadoB2, personal.AccionHechoEmpleadoB2, personal.AccionFichaEmpleadoB2} {
		actual, e := autoridad.actor(ctx, accion)
		if e != nil || !reflect.DeepEqual(actual, primero) {
			t.Fatalf("%s sustituyó actor/cuenta: %v", accion, e)
		}
	}
	perfil := base.nominales.b2[personal.AccionFichaEmpleadoB2]
	revocada := publicador.publicadas[perfil.perfilRef()]
	revocada.instantanea.AsignacionPerfil.Estado = core.EstadoAsignacionPerfilRevocada
	publicador.publicadas[perfil.perfilRef()] = revocada
	if _, e = autoridad.ActorLecturaHechosB2(ctx); !errors.Is(e, ct.ErrAutorizacionDenegada) {
		t.Fatalf("contexto capturado conservó concesión revocada: %v", e)
	}
}
func TestIncorporacionB2GETYPrepararNoAutorizanEfectoPersonal(t *testing.T) {
	for _, metodo := range []string{"GET", "POST"} {
		ctx := context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: metodo, ruta: "/api/interno/contratacion-temporal/incorporacion-personal-b2/plan/v1"})
		for _, accion := range []string{"personal.plan_incorporacion_ct.preparar", "personal.plan_incorporacion_ct.ejecutar", personal.AccionAltaEmpleadoB2, personal.AccionHechoEmpleadoB2, "vec.catalogos.categorias.reservar_uso", "vec.catalogos.categorias.confirmar_uso", ct.AccionConfirmarOrigenB2} {
			if operacionPermitidaEnRutaIncorporacionB2(ctx, accion) {
				t.Fatalf("%s permitió efecto %s", metodo, accion)
			}
		}
		if operacionPermitidaEnRutaIncorporacionB2(ctx, ct.AccionRegistrarPlanNominalB2) != (metodo == "POST") {
			t.Fatal("preparación CT no está ligada al POST")
		}
	}
	if operacionPermitidaEnRutaIncorporacionB2(context.Background(), ct.AccionLeerPlanNominalB2) {
		t.Fatal("falta de frontera concedió lectura")
	}
	if operacionPermitidaEnRutaIncorporacionB2(nil, ct.AccionLeerPlanNominalB2) {
		t.Fatal("contexto ausente concedió lectura")
	}
	ctx, cancelar := context.WithCancel(context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: "POST", ruta: "/api/interno/contratacion-temporal/incorporacion-personal-b2/confirmar/v1"}))
	cancelar()
	if operacionPermitidaEnRutaIncorporacionB2(ctx, ct.AccionConfirmarOrigenB2) {
		t.Fatal("petición cancelada conservó autorización de efecto")
	}
}
