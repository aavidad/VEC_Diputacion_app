package bootstrap

import (
	"context"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func escenarioConsultasTecnicoFijasPrueba(t *testing.T) (*soporteAltaContratacionTemporalDesarrollo,
	*autoridadConsultasRRHHDesarrollo, *autoridadAsignacionesContratacionTemporalDesarrolloPrueba,
	*perfilFijoCTDesarrollo, dominiovec.Principal,
) {
	t.Helper()
	alta, autoridad, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	s := alta.soporte
	if err := componerPerfilesFijosAltaCoberturaCTDesarrollo(s, principal, s.reloj.Ahora(), origenEntregaPerfilFijoPrueba(t)); err != nil {
		t.Fatal(err)
	}
	if err := componerPerfilFijoConsultasTecnicoCTDesarrollo(s, s.reloj.Ahora()); err != nil {
		t.Fatal(err)
	}
	p := perfilFijoConsultasTecnicoCTDesarrollo(s)
	if p == nil {
		t.Fatal("perfil de lecturas ausente")
	}
	publicador := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	publicador.asignaciones = map[string]instantaneaPublicadaDesarrollo{p.perfilRef(): {
		instantanea:    clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.plantilla),
		actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo,
	}}
	s.mu.Lock()
	p.contextoEsperadoRegistrado = p.contexto.Resultado
	s.mu.Unlock()
	autoridad.perfilLectura = p
	return s, autoridad, publicador, p, principal
}

func TestConsultasTecnicoPerfilEstableYFronteraNominal(t *testing.T) {
	s, autoridad, _, p, _ := escenarioConsultasTecnicoFijasPrueba(t)
	cuadro, detalle := s.instantaneaCuadroRRHH, s.instantaneaDetalleRRHH
	esperadas := append(append([]dominiovec.ConcesionRol(nil), cuadro.VersionRol.Concesiones...), detalle.VersionRol.Concesiones...)
	if len(esperadas) != 3 || !reflect.DeepEqual(p.plantilla.VersionRol.Concesiones, esperadas) ||
		!reflect.DeepEqual(p.plantilla.AsignacionPerfil.Ambitos, cuadro.AsignacionPerfil.Ambitos) ||
		p.perfilRef() == s.contexto.Resultado.Contexto.PerfilActivoRef ||
		p.perfilRef() == s.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes).perfilRef() {
		t.Fatal("el perfil fijo alteró concesiones, ámbitos o perfil de Alta")
	}
	if !autoridad.contextoConsultaRRHHConservaActor(p.contexto) ||
		autoridad.contextoConsultaRRHHConservaActor(s.contexto) {
		t.Fatal("se aceptó el perfil dinámico o se rechazó el fijo")
	}
	dinamico := s.contexto.Resultado.Contexto.PerfilActivoRef
	original := []descriptorFronteraComunDesarrollo{{Metodo: "POST", Ruta: httpinterno.RutaConsultaCuadroRRHH,
		PerfilesActivosRef: []string{dinamico, "prf_lector_nominal"}},
		{Metodo: "POST", Ruta: httpinterno.RutaConsultaDetalleRRHH,
			PerfilesActivosRef: []string{dinamico, "prf_lector_nominal"}}}
	fronteras, err := asignarPerfilesFijosEnFronterasCTDesarrollo(s, dinamico, original)
	if err != nil || fronteras[0].PerfilesActivosRef[0] != p.perfilRef() ||
		fronteras[1].PerfilesActivosRef[0] != p.perfilRef() ||
		fronteras[0].PerfilesActivosRef[1] != "prf_lector_nominal" ||
		original[0].PerfilesActivosRef[0] != dinamico || original[1].PerfilesActivosRef[0] != dinamico {
		t.Fatalf("fronteras de lectura incoherentes: %v", err)
	}
}

func TestConsultasTecnicoConsumenAsignacionYDenieganRevocacion(t *testing.T) {
	s, _, publicador, p, principal := escenarioConsultasTecnicoFijasPrueba(t)
	for _, ruta := range []string{httpinterno.RutaConsultaCuadroRRHH, httpinterno.RutaConsultaDetalleRRHH} {
		ctx := contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, ruta)
		datos := datosSolicitudConsultasRRHHDesarrolloPrueba(t, s, ruta)
		datos.VinculoAutenticacionActor = p.contexto.Vinculo
		ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
		instantanea, ok := s.instantaneaParaContexto(ctx, ruta)
		if !ok || instantanea.AsignacionPerfil.PerfilActivoRef != p.perfilRef() ||
			instantanea.VersionRol.RolID != rolConsultasTecnicoCTDesarrollo {
			t.Fatalf("lectura %s sin perfil estable", ruta)
		}
		if publicador.preparadas != 0 || publicador.publicadas != 0 {
			t.Fatal("la consulta preparó o publicó una asignación")
		}
		alterada := datos
		alterada.Recurso.Ambitos = map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
			"clase_ambito": string(ports.AmbitoOrganizacionRRHH), "ambito_ref": "organizacion:ajena"}
		if _, ok := s.instantaneaParaContexto(context.WithValue(ctx,
			claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, alterada), ruta); ok {
			t.Fatal("se amplió el ámbito desde la petición")
		}
		publicada := publicador.asignaciones[p.perfilRef()]
		publicada.actoAsignacion = actoAsignacionCTDesarrollo
		publicador.asignaciones[p.perfilRef()] = publicada
		if _, ok := s.instantaneaParaContexto(ctx, ruta); ok {
			t.Fatal("se reactivó una asignación ajena al perfil fijo")
		}
		publicada.actoAsignacion = actoAsignacionPerfilFijoCTDesarrollo
		publicador.asignaciones[p.perfilRef()] = publicada
	}
}

func TestConsultasTecnicoDenieganCertificadoAjeno(t *testing.T) {
	s, _, _, _, principal := escenarioConsultasTecnicoFijasPrueba(t)
	s.certificadoSHA256 = "certificado-ajeno"
	if err := componerPerfilFijoConsultasTecnicoCTDesarrollo(s, time.Now().UTC().Truncate(time.Microsecond)); err == nil {
		t.Fatal("aceptó otro certificado para componer el perfil")
	}
	principal.ID = "persona:ajena"
	if _, ok := s.capacidadValida(contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, httpinterno.RutaConsultaCuadroRRHH)); ok {
		t.Fatal("aceptó otro actor en el canal")
	}
}
