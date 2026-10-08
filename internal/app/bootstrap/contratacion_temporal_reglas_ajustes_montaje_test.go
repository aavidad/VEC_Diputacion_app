package bootstrap

import (
	"net/http"
	"testing"
	"time"

	ajusteshttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/ajustesreglas"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

func TestAjustesReglasCTPerfilFijoSoloConSelectorYProvisionCAS(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	origen := origenEntregaPerfilFijoPrueba(t)
	sin, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	sin.origen = origen
	if err := componerPerfilesFijosAltaCoberturaCTDesarrollo(sin, principal, ahora, origen, false); err != nil {
		t.Fatal(err)
	}
	base := sin.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes)
	if base == nil || base.plantilla.Validar() != nil || len(base.plantilla.VersionRol.Concesiones) != 1 ||
		sin.perfilFijoParaRutaYMetodo(ajusteshttp.Ruta, http.MethodGet) != nil ||
		sin.perfilFijoParaRutaYMetodo(ajusteshttp.Ruta, http.MethodPost) != nil {
		t.Fatal("selector apagado alteró el permiso o las rutas de alta")
	}
	objetivo, err := ampliarInstantaneaAltaConAjustesCT(base.plantilla)
	if err != nil || objetivo.Validar() != nil || len(objetivo.VersionRol.Concesiones) != 3 {
		t.Fatalf("plantilla ampliada inválida: %v", err)
	}
	if _, concedida := instantaneaConsumible(instantaneaPublicadaDesarrollo{instantanea: base.plantilla}, objetivo, ahora); concedida ||
		(aprobacionProvisionPerfilesRRHHDesarrollo{}).valida() {
		t.Fatal("una asignación anterior se consideró provisionada sin CAS aprobado")
	}
	con, _, principalCon := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	con.origen = origen
	if err := componerPerfilesFijosAltaCoberturaCTDesarrollo(con, principalCon, ahora, origen, true); err != nil {
		t.Fatal(err)
	}
	fijo := con.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes)
	if fijo == nil || fijo.plantilla.Validar() != nil || len(fijo.plantilla.VersionRol.Concesiones) != 3 ||
		con.perfilFijoParaRutaYMetodo(ajusteshttp.Ruta, http.MethodGet) != fijo ||
		con.perfilFijoParaRutaYMetodo(ajusteshttp.Ruta, http.MethodPost) != fijo {
		t.Fatal("GET y POST no comparten el perfil fijo de alta con concesiones exactas")
	}
}
