package bootstrap

import (
	"net/http"
	"testing"
	"time"

	ajusteshttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/ajustesreglas"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestAjustesReglasCTPerfilFijoSoloConSelectorYProvisionCAS(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	origen := origenEntregaPerfilFijoPrueba(t)
	sin, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	sin.origen = origen
	if err := componerPerfilesFijosAltaCoberturaCTDesarrollo(sin, principal, ahora, origen, false); err != nil {
		t.Fatal(err)
	}
	base := sin.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, http.MethodGet)
	if base == nil || base.plantilla.Validar() != nil || len(base.plantilla.VersionRol.Concesiones) != 1 ||
		sin.perfilFijoParaRutaYMetodo(ajusteshttp.Ruta, http.MethodGet) != nil ||
		sin.perfilFijoParaRutaYMetodo(ajusteshttp.Ruta, http.MethodPost) != nil {
		t.Fatal("selector apagado alteró el permiso o las rutas del lector de entrega")
	}
	objetivo, err := ampliarInstantaneaLectorEntregaConAjustesCT(base.plantilla)
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
	fijo := con.perfilFijoParaRutaYMetodo(rutaEntregaPeticionCentro, http.MethodGet)
	if fijo == nil || fijo.plantilla.Validar() != nil || len(fijo.plantilla.VersionRol.Concesiones) != 3 ||
		con.perfilFijoParaRutaYMetodo(ajusteshttp.Ruta, http.MethodGet) != fijo ||
		con.perfilFijoParaRutaYMetodo(ajusteshttp.Ruta, http.MethodPost) != fijo {
		t.Fatal("GET y POST no comparten el perfil fijo lector con concesiones exactas")
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: "vec.contratacion_temporal.reglas", ModuloID: "contratacion_temporal",
		Tipo: "catalogo_reglas", Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo},
		Atributos: map[string]string{"material_sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}
	if !fijo.plantilla.AsignacionPerfil.Cubre(recurso) ||
		con.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes).plantilla.AsignacionPerfil.Cubre(recurso) {
		t.Fatal("AD114 exige exactamente el ámbito de organización del perfil lector")
	}
}
