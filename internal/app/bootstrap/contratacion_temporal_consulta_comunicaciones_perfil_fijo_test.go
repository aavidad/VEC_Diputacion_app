package bootstrap

import (
	"context"
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// F1-150: la consulta de comunicaciones tiene perfil fijo propio con el mismo
// rol y ámbito que tenía en el perfil dinámico, su frontera común admite solo
// ese perfil y la autorización V3 se concede sin publicar nada.
func TestConsultaComunicacionesPerfilFijoFronteraYConsumoSinPublicar(t *testing.T) {
	s, base, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ruta := httpinterno.RutaConsultaComunicacionesExpediente
	dinamico := s.contexto.Resultado.Contexto.PerfilActivoRef
	fijo, err := nuevoPerfilFijoConsultaComunicacionesExpedienteDesarrollo(principal, s.contexto, s.reloj.Ahora())
	if err != nil || s.registrarPerfilFijoCTDesarrollo(fijo) != nil {
		t.Fatal("perfil fijo de comunicaciones no compuesto", err)
	}
	historica, err := nuevaInstantaneaConsultaComunicacionesExpedienteDesarrollo(principal.ID, dinamico, s.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	if fijo.perfilRef() == dinamico || s.perfilFijoParaRuta(ruta) != fijo ||
		fijo.contexto.Resultado.Contexto.PersonaRef != s.contexto.Resultado.Contexto.PersonaRef ||
		fijo.plantilla.VersionRol.Referencia() != historica.VersionRol.Referencia() ||
		!reflect.DeepEqual(fijo.plantilla.AsignacionPerfil.Ambitos, historica.AsignacionPerfil.Ambitos) {
		t.Fatal("el perfil fijo cambia la persona, el rol o el ámbito de la consulta")
	}
	asignadas, err := asignarPerfilesFijosEnFronterasCTDesarrollo(s, dinamico, descriptoresFronterasLlamamientoDesarrollo(dinamico))
	if err != nil {
		t.Fatal(err)
	}
	vistas := 0
	for _, d := range asignadas {
		esperado := []string{dinamico}
		if d.Ruta == ruta {
			vistas++
			esperado = []string{fijo.perfilRef()}
		}
		if !reflect.DeepEqual(d.PerfilesActivosRef, esperado) {
			t.Fatalf("frontera %s con perfiles %v", d.Ruta, d.PerfilesActivosRef)
		}
	}
	if vistas != 1 {
		t.Fatal("frontera de comunicaciones ausente o repetida")
	}

	fijo.contextoEsperadoRegistrado = fijo.contexto.Resultado
	fijo.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: fijo.contexto}
	s.motivoConsultaComunicaciones = motivoConsultaComunicacionesExpedienteDesarrollo()
	a := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	a.asignaciones = map[string]instantaneaPublicadaDesarrollo{fijo.perfilRef(): {
		instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(fijo.plantilla), actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}
	solicitar := func(expediente, organizacion string) (context.Context, dominiovec.SolicitudAutorizacionLigadaV3) {
		t.Helper()
		c := ports.ConsultaComunicacionesExpediente{ExpedienteRef: expediente, Limite: 10}
		recurso, err := postgresct.RecursoConsultaComunicacionesExpediente(c, organizacion)
		if err != nil {
			t.Fatal(err)
		}
		ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, ruta)
		correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
		if err != nil {
			t.Fatal(err)
		}
		d := dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: fijo.contexto.Vinculo,
			Accion: postgresct.AccionConsultaComunicacionesExpediente, ReferenciaMotivo: motivoConsultaComunicacionesExpedienteDesarrollo(),
			Recurso: recurso, Finalidad: "gestionar_contratacion_temporal", Correlacion: correlacion}
		ctx = context.WithValue(ctx, claveConsultaComunicacionesExpedienteDesarrollo{}, solicitudLigadaComunicacionesExpedienteDesarrollo{c, organizacion})
		ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, d)
		solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(d)
		if err != nil {
			t.Fatal(err)
		}
		return ctx, solicitud
	}
	autorizador := base.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo)
	for _, expediente := range []string{"expediente:ct150:uno", "expediente:ct150:dos"} {
		ctx, solicitud := solicitar(expediente, organizacionAltaContratacionTemporalDesarrollo)
		if _, _, err := autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, fijo.contexto.Resultado); err != nil {
			t.Fatalf("consulta nominal de %s denegada: %v", expediente, err)
		}
		// La sesión del perfil dinámico no sirve para esta ruta.
		if _, _, err := autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, s.contexto.Resultado); err == nil {
			t.Fatal("el perfil dinámico obtuvo la consulta de comunicaciones")
		}
	}
	if a.preparadas != 0 || a.publicadas != 0 {
		t.Fatal("la consulta publicó permisos")
	}
	ajena, _ := solicitar("expediente:ct150:uno", "organizacion:desarrollo:ajena")
	if _, ok := s.instantaneaParaContexto(ajena, ruta); ok {
		t.Fatal("se concedió la consulta de otra organización")
	}
	ctx, _ := solicitar("expediente:ct150:uno", organizacionAltaContratacionTemporalDesarrollo)
	publicada := a.asignaciones[fijo.perfilRef()]
	publicada.instantanea.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
	a.asignaciones[fijo.perfilRef()] = publicada
	if _, ok := s.instantaneaParaContexto(ctx, ruta); ok {
		t.Fatal("asignación revocada consumida")
	}
}
