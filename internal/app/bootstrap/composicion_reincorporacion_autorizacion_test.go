package bootstrap

import (
	"maps"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func recursoReincorporacionPerfilFijoPrueba(expediente string) vecdomain.RecursoAutorizable {
	return vecdomain.RecursoAutorizable{Referencia: expediente, ModuloID: ports.ModuloContratacion,
		Tipo:    ports.TipoRecursoReincorporacionTitular,
		Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo},
		Atributos: map[string]string{"version_expediente": "4", "fase_previa": "nombramiento", "estado_previo": "en_curso",
			"relacion_ref": "relacion:1", "fecha_efectiva": "2026-09-20", "documento_ref": "documento:1", "documento_sha256": strings.Repeat("a", 64),
			"cese_evento_ref": "evento:cese", "cese_recibo_ref": "recibo:cese",
			"ambito_idempotencia_hmac": "hmac-sha256:" + ports.DominioAmbitoReincorporacionTitular + "/v1:" + strings.Repeat("a", 64),
			"huella_peticion_hmac":     "hmac-sha256:" + ports.DominioHuellaReincorporacionTitular + "/v1:" + strings.Repeat("b", 64),
			"politica_ref":             "politica:1", "politica_version": "1", "politica_huella_sha256": strings.Repeat("c", 64)}}
}

func TestReincorporacionPerfilFijoValidaEvidenciaYFormaCerrada(t *testing.T) {
	recurso := recursoReincorporacionPerfilFijoPrueba("expediente:arbitrario")
	datos := vecdomain.DatosSolicitudAutorizacionLigadaV3{Accion: string(domain.AccionRegistrarReincorporacionTitular),
		Finalidad: ports.FinalidadRegistrarReincorporacionTitular, ReferenciaMotivo: motivoSeguimientoCeseDesarrollo(httpinterno.RutaReincorporacionesTitular), Recurso: recurso}
	if !solicitudAutorizacionReincorporacionTitularValida(httpinterno.RutaReincorporacionesTitular, datos) {
		t.Fatal("recurso válido rechazado")
	}
	for _, caso := range []struct{ clave, valor string }{
		{"version_expediente", "04"}, {"version_expediente", "0"}, {"relacion_ref", ""}, {"fecha_efectiva", "2026-09-31"},
		{"documento_ref", ""}, {"documento_sha256", "mal"}, {"fase_previa", "solicitud"}, {"estado_previo", "completado"},
		{"cese_evento_ref", ""}, {"cese_recibo_ref", ""}, {"ambito_idempotencia_hmac", "mal"}, {"huella_peticion_hmac", "mal"},
		{"politica_ref", ""}, {"politica_version", "01"}, {"politica_huella_sha256", "mal"}, {"extra", "valor"},
	} {
		t.Run(caso.clave+"_"+caso.valor, func(t *testing.T) {
			alterada := datos
			alterada.Recurso.Atributos = maps.Clone(recurso.Atributos)
			alterada.Recurso.Atributos[caso.clave] = caso.valor
			if solicitudAutorizacionReincorporacionTitularValida(httpinterno.RutaReincorporacionesTitular, alterada) {
				t.Fatal("evidencia o forma alterada admitida")
			}
		})
	}
	for _, ruta := range []string{httpinterno.RutaCesesNombramiento, httpinterno.RutaCapacidadReincorporacionTitular} {
		if solicitudAutorizacionReincorporacionTitularValida(ruta, datos) {
			t.Fatalf("escritura CT130 admitida por ruta %s", ruta)
		}
	}
	datos.Recurso.Ambitos["expediente_ref"] = recurso.Referencia
	if solicitudAutorizacionReincorporacionTitularValida(httpinterno.RutaReincorporacionesTitular, datos) {
		t.Fatal("expediente en ámbitos admitido")
	}
}

func TestReincorporacionPerfilFijoAutorizaExpedientesSinPublicar(t *testing.T) {
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ctxNominal, err := nuevoContextoReincorporacionTitularDesarrollo(s, s.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	v, _ := ctxNominal.Vinculo.Datos()
	pl, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, s.reloj.Ahora(),
		"reincorporacion_titular_ct_desarrollo", "Reincorporación titular de desarrollo", "reincorporacion-titular-ct-desarrollo",
		[]vecdomain.ConcesionRol{{Accion: string(domain.AccionRegistrarReincorporacionTitular), ModuloID: ports.ModuloContratacion,
			TipoRecurso: ports.TipoRecursoReincorporacionTitular, Finalidades: []string{ports.FinalidadRegistrarReincorporacionTitular}, GarantiaMinima: vecdomain.AuthAssuranceHigh}},
		[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		t.Fatal(err)
	}
	s.reincorporacionTitular = &soporteSeguimientoCeseDesarrollo{contexto: ctxNominal, instantanea: pl,
		contextoEsperadoRegistrado: ctxNominal.Resultado, sesionOperativa: proveedorSesionOperativaCTPrueba{contexto: ctxNominal}}
	fijo, err := perfilFijoReincorporacionTitular(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.registrarPerfilFijoCTDesarrollo(fijo); err != nil {
		t.Fatal(err)
	}
	central := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	central.asignaciones = map[string]instantaneaPublicadaDesarrollo{fijo.perfilRef(): {
		instantanea: pl, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo, actoControl: actoControlRolReincorporacionTitularDesarrollo}}
	servicio, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(s, s, s, s, s.reloj,
		seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{})
	if err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadSeguimientoCeseDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: s,
		autorizador: &autorizadorAnalisisContratacionTemporalDesarrollo{delegado: servicio, soporte: s, instalado: true}}}
	for _, expediente := range []string{"expediente:uno", "expediente:dos", "expediente:uno"} {
		ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaReincorporacionesTitular)
		solicitud, _, _, _, err := autoridad.exigir(ctx, string(domain.AccionRegistrarReincorporacionTitular),
			ports.FinalidadRegistrarReincorporacionTitular, recursoReincorporacionPerfilFijoPrueba(expediente))
		if err != nil {
			t.Fatalf("%s: %v", expediente, err)
		}
		d, _ := solicitud.Datos()
		if d.Recurso.Referencia != expediente {
			t.Fatal("el recurso cambió de expediente")
		}
	}
	publicada := central.asignaciones[fijo.perfilRef()]
	publicada.instantanea.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
	publicada.instantanea.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
	central.asignaciones[fijo.perfilRef()] = publicada
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaReincorporacionesTitular)
	if _, _, _, _, err := autoridad.exigir(ctx, string(domain.AccionRegistrarReincorporacionTitular), ports.FinalidadRegistrarReincorporacionTitular,
		recursoReincorporacionPerfilFijoPrueba("expediente:uno")); err == nil {
		t.Fatal("revocación reactivada")
	}
	if central.preparadas != 0 || central.publicadas != 0 {
		t.Fatalf("peticiones publicaron autoridad: %+v", central)
	}
}
