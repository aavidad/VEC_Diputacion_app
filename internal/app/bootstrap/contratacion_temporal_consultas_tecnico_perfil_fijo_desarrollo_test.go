package bootstrap

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestDetalleIncorporacionConservaPerfilTecnicoFijoLigado(t *testing.T) {
	s, _, _, fijo, principal := escenarioConsultasTecnicoFijasPrueba(t)
	dinamico := s.contexto.Resultado.Contexto.PerfilActivoRef
	declaraciones := descriptoresFronterasContratacionTemporalDesarrollo(dinamico, []string{dinamico})
	asignadas, err := asignarPerfilesFijosEnFronterasCTDesarrollo(s, dinamico, declaraciones)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(asignadas)
	if err != nil {
		t.Fatal(err)
	}
	d, ok := catalogo.resolver(http.MethodPost, httpinterno.RutaConsultaDetalleRRHH)
	if !ok || !esDescriptorDetalleContratacionTemporalDesarrollo(d, s) ||
		len(d.PerfilesActivosRef) == 0 || d.PerfilesActivosRef[0] != fijo.perfilRef() ||
		d.admitePerfil(dinamico) {
		t.Fatal("el descriptor no conserva el perfil técnico exacto")
	}
	for nombre, mutar := range map[string]func(*descriptorFronteraComunDesarrollo){
		"fijo_ajeno": func(x *descriptorFronteraComunDesarrollo) {
			x.PerfilesActivosRef[0] = s.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes).perfilRef()
		},
		"dinamico_retirado": func(x *descriptorFronteraComunDesarrollo) { x.PerfilesActivosRef[0] = dinamico },
		"sin_perfil":        func(x *descriptorFronteraComunDesarrollo) { x.PerfilesActivosRef = nil },
		"dinamico_anadido": func(x *descriptorFronteraComunDesarrollo) {
			x.PerfilesActivosRef = append(x.PerfilesActivosRef, dinamico)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alterado := d
			alterado.PerfilesActivosRef = append([]string(nil), d.PerfilesActivosRef...)
			mutar(&alterado)
			if esDescriptorDetalleContratacionTemporalDesarrollo(alterado, s) {
				t.Fatal("descriptor no ligado admitido")
			}
		})
	}
	if _, err := asignarPerfilesNominalesIncorporacionEnFronteras(s, asignadas); err != nil {
		t.Fatal("incorporación V2 rechazó el detalle ligado:", err)
	}
	if _, err := asignarPerfilesNominalesB2EnFronteras(s, asignadas); err != nil {
		t.Fatal("incorporación B2 rechazó el detalle ligado:", err)
	}
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaIncorporacionEjercicioV2)
	c := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	c.certificadoVerificadoEn = s.reloj.Ahora().Add(-time.Second)
	c.certificadoValidoHasta = s.reloj.Ahora().Add(time.Minute)
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
	hijo, err := subconsultaDetalleContratacionTemporalDesarrollo(ctx, s, catalogo)
	if err != nil {
		t.Fatal("subconsulta rechazó el detalle ligado:", err)
	}
	frontera, ok := hijo.Value(claveFronteraSeguridadComunDesarrollo{}).(fronteraSeguridadComunDesarrollo)
	if !ok || !frontera.descriptor.admitePerfil(fijo.perfilRef()) || frontera.descriptor.admitePerfil(dinamico) {
		t.Fatal("subconsulta heredó otro perfil")
	}
}

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
