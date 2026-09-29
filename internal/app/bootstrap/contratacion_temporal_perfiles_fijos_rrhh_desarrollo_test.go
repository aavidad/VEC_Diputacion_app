package bootstrap

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

func escenarioPerfilesFijosPrueba(t *testing.T) (*soporteAltaContratacionTemporalDesarrollo, *autoridadAsignacionesContratacionTemporalDesarrolloPrueba) {
	t.Helper()
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	if err := componerPerfilesFijosAltaCoberturaCTDesarrollo(s, principal, time.Now().UTC().Truncate(time.Microsecond), nil); err != nil {
		t.Fatal(err)
	}
	autoridad, ok := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	if !ok {
		t.Fatal("escenario sin autoridad de prueba")
	}
	return s, autoridad
}

// Las rutas con perfil fijo nunca preparan ni publican; sin la autoridad
// PostgreSQL que consume la asignación publicada, se deniegan.
func TestRutasPerfilFijoNuncaPreparanNiPublican(t *testing.T) {
	s, autoridad := escenarioPerfilesFijosPrueba(t)
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principal.ID, principal.Attributes["certificate_sha256"] = s.principalID, s.certificadoSHA256
	for _, ruta := range []string{httpinterno.RutaAltaSolicitudes, httpinterno.RutaPropuestaCobertura,
		httpinterno.RutaRegistroAnalisisRRHH, httpinterno.RutaRectificacionAnalisisRRHH,
		httpinterno.RutaDecisionCobertura, httpinterno.RutaRectificacionCobertura, httpinterno.RutaResultadoCobertura} {
		ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, ruta)
		if _, ok := s.instantaneaParaContexto(ctx, ruta); ok {
			t.Fatalf("%s: concedida sin asignación publicada consumible", ruta)
		}
		if ruta == httpinterno.RutaDecisionCobertura || ruta == httpinterno.RutaRectificacionCobertura {
			_ = s.publicarInstantaneaDecisionCobertura(ctx, ruta)
		}
	}
	if autoridad.preparadas != 0 || autoridad.publicadas != 0 {
		t.Fatalf("un perfil fijo preparó %d y publicó %d", autoridad.preparadas, autoridad.publicadas)
	}
}

func TestPerfilesFijosSeparanPerfilYRutas(t *testing.T) {
	s, _ := escenarioPerfilesFijosPrueba(t)
	dinamico := s.contexto.Resultado.Contexto.PerfilActivoRef
	alta, cobertura := s.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes), s.perfilFijoParaRuta(httpinterno.RutaPropuestaCobertura)
	if alta == nil || cobertura == nil || alta == cobertura ||
		alta.perfilRef() == dinamico || cobertura.perfilRef() == dinamico ||
		alta.contexto.Resultado.Contexto.PersonaRef != s.contexto.Resultado.Contexto.PersonaRef ||
		alta.contexto.Resultado.Contexto.Instantanea.CuentaRef != s.contexto.Resultado.Contexto.Instantanea.CuentaRef {
		t.Fatal("perfiles fijos sin separar o de otra persona")
	}
	analisis := s.perfilFijoParaRuta(httpinterno.RutaRegistroAnalisisRRHH)
	if analisis == nil || analisis != s.perfilFijoParaRuta(httpinterno.RutaRectificacionAnalisisRRHH) ||
		analisis == alta || analisis == cobertura || analisis.perfilRef() == dinamico {
		t.Fatal("el análisis no tiene su propio perfil fijo")
	}
	for _, ruta := range []string{rutaEntregaPeticionCentro, httpinterno.RutaAsignaciones,
		httpinterno.RutaConsultaCuadroRRHH, httpinterno.RutaConsultaDetalleRRHH} {
		if s.perfilFijoParaRuta(ruta) != nil {
			t.Fatalf("%s no debe usar un perfil fijo", ruta)
		}
	}
	// Una ruta ya asignada no puede pasar a otro perfil.
	if err := s.registrarPerfilFijoCTDesarrollo(&perfilFijoCTDesarrollo{clave: "otro", plantilla: alta.plantilla,
		rutas: map[string]struct{}{httpinterno.RutaAltaSolicitudes: {}}}); err == nil {
		t.Fatal("se registró otro perfil para una ruta ya asignada")
	}
	fronteras := descriptoresFronterasContratacionTemporalDesarrollo(dinamico, []string{dinamico})
	asignadas, err := asignarPerfilesFijosEnFronterasCTDesarrollo(s, dinamico, fronteras)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range asignadas {
		fijo := s.perfilFijoParaRuta(d.Ruta)
		switch {
		case fijo != nil && (len(d.PerfilesActivosRef) != 1 || d.PerfilesActivosRef[0] != fijo.perfilRef()):
			t.Fatalf("%s: la frontera no admite solo su perfil fijo", d.Ruta)
		case fijo == nil && len(d.PerfilesActivosRef) == 1 && d.PerfilesActivosRef[0] != dinamico:
			t.Fatalf("%s: una ruta dinámica cambió de perfil", d.Ruta)
		}
	}
	// Una frontera que no esté en el perfil dinámico no se reasigna.
	fronteras[0].PerfilesActivosRef = []string{"prf_ajeno"}
	for i := range fronteras {
		if fronteras[i].Ruta == httpinterno.RutaAltaSolicitudes {
			fronteras[i].PerfilesActivosRef = []string{"prf_ajeno"}
		}
	}
	if _, err := asignarPerfilesFijosEnFronterasCTDesarrollo(s, dinamico, fronteras); err == nil {
		t.Fatal("se reasignó una frontera que no era del perfil dinámico")
	}
}

// El contexto operativo de una ruta fija usa la sesión de su perfil; sin
// ella, deniega (nunca cae a la sesión del perfil dinámico).
func TestContextoOperativoPerfilFijoNoCaeAlDinamico(t *testing.T) {
	s, _ := escenarioPerfilesFijosPrueba(t)
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	principal.ID, principal.Attributes["certificate_sha256"] = s.principalID, s.certificadoSHA256
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaAltaSolicitudes)
	if _, err := s.contextoOperativoDesarrollo(ctx); err == nil {
		t.Fatal("el alta resolvió contexto sin la sesión de su perfil fijo")
	}
	fijo := s.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes)
	s.mu.Lock()
	fijo.contextoEsperadoRegistrado = fijo.contexto.Resultado
	fijo.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: fijo.contexto}
	s.mu.Unlock()
	contexto, err := s.contextoOperativoDesarrollo(contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaAltaSolicitudes))
	if err != nil || contexto.Resultado.Contexto.PerfilActivoRef != fijo.perfilRef() {
		t.Fatalf("el alta no usa el contexto de su perfil fijo: %v", err)
	}
	_ = context.Background()
}
