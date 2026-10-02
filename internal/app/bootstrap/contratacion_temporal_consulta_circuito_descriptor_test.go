package bootstrap

import (
	"net/http"
	"testing"

	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctpostgres "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
)

func TestPDPComunConsultaCircuitoExigeFronteraYPerfilFijo(t *testing.T) {
	const base = "prf_ct_prueba"
	const fijo = "prf_ct_circuito"
	fronteras := append(descriptoresFronterasContratacionTemporalDesarrollo(base, []string{base}),
		fronteraContratacionTemporalDesarrollo("ct-circuito-rrhh-consultar",
			ctpostgres.AccionConsultaCircuitoRRHH, cthttp.RutaConsultaCircuitoRRHH, []string{fijo}))
	catalogoFronteras, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	frontera, ok := catalogoFronteras.resolver(http.MethodPost, cthttp.RutaConsultaCircuitoRRHH)
	if !ok || !frontera.admitePerfil(fijo) || frontera.admitePerfil(base) {
		t.Fatalf("consulta sin perfil fijo exclusivo: %+v", frontera)
	}
	politica := politicaDescriptoresCTPrueba(t)
	apagados := descriptoresAutorizacionContratacionTemporalDesarrollo(politica, false, false, false)
	baseAntigua := descriptoresAutorizacionContratacionTemporalDesarrollo(politica, false, false)
	if len(apagados) != len(baseAntigua) {
		t.Fatal("modo apagado cambió las autorizaciones heredadas")
	}
	catalogoApagado, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogoFronteras, apagados)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := catalogoApagado.politicaPara(ctpostgres.AccionConsultaCircuitoRRHH,
		frontera.Clave, frontera.ClavePolitica, frontera.ClaveCapacidad); ok {
		t.Fatal("consulta de circuito autorizada sin activar su perfil fijo")
	}
	catalogoActivo, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogoFronteras,
		descriptoresAutorizacionContratacionTemporalDesarrollo(politica, false, false, true))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := catalogoActivo.politicaPara(ctpostgres.AccionConsultaCircuitoRRHH,
		frontera.Clave, frontera.ClavePolitica, frontera.ClaveCapacidad); !ok {
		t.Fatal("la consulta nominal no entra en el PDP común")
	}
	if _, ok := catalogoActivo.politicaPara(ctpostgres.AccionConsultaCircuitoRRHH,
		"ct-expediente-consultar", frontera.ClavePolitica, frontera.ClaveCapacidad); ok {
		t.Fatal("la acción de circuito quedó accesible por el detalle antiguo")
	}
	if _, err := nuevoCatalogoAutorizacionComunDesarrollo(
		catalogoFronterasSinCircuito(t, base),
		descriptoresAutorizacionContratacionTemporalDesarrollo(politica, false, false, true)); err == nil {
		t.Fatal("aceptó el permiso de circuito sin su frontera")
	}
}

func catalogoFronterasSinCircuito(t *testing.T, perfil string) catalogoFronterasComunDesarrollo {
	t.Helper()
	c, err := nuevoCatalogoFronterasComunDesarrollo(descriptoresFronterasContratacionTemporalDesarrollo(perfil, []string{perfil}))
	if err != nil {
		t.Fatal(err)
	}
	return c
}
