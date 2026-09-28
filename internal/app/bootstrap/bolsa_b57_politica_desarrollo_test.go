package bootstrap

import (
	"net/http"
	"testing"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestPoliticaB57SoloVersionesNuevasConConcesionesNominales(t *testing.T) {
	for _, base := range []int{5, 6, 7, 8} {
		anterior, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
			"actor_b57", "perfil_b57", "unidad_b57", "ambito_b57", time.Date(2026, 9, 28, 7, 0, 0, 0, time.UTC), base)
		if err != nil {
			t.Fatal(err)
		}
		nueva, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
			"actor_b57", "perfil_b57", "unidad_b57", "ambito_b57", time.Date(2026, 9, 28, 7, 0, 0, 0, time.UTC), base+4)
		if err != nil {
			t.Fatal(err)
		}
		for _, accion := range []string{puertosbolsa.AccionProponerCausasParticipacion, puertosbolsa.AccionConsultarPropuestaCausasParticipacion, puertosbolsa.AccionConsultarCausasParticipacion, puertosbolsa.AccionPublicarCausasParticipacion} {
			if concesionB57Existe(anterior.VersionRol.Concesiones, accion) || !concesionB57Existe(nueva.VersionRol.Concesiones, accion) {
				t.Fatalf("v%d→v%d: concesión %s", base, base+4, accion)
			}
		}
		if len(nueva.VersionRol.Concesiones) != len(anterior.VersionRol.Concesiones)+4 {
			t.Fatalf("v%d→v%d cambió concesiones ajenas", base, base+4)
		}
	}
}

func TestFronterasB57SeparanLecturaYPublicacion(t *testing.T) {
	fronteras, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo("prf_bolsa_bback", false, false, true)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	get, ok := catalogo.resolver(http.MethodGet, bolsahttp.RutaCatalogoCausasParticipacion)
	if !ok || get.Clave != claveFronteraConsultarCausasParticipacionBolsa || get.ClaveCapacidad != claveCapacidadConsultarCausasParticipacionBolsa {
		t.Fatal("GET B57 sin frontera nominal de lectura")
	}
	if _, ok := catalogo.resolver(http.MethodPost, bolsahttp.RutaCatalogoCausasParticipacion); ok {
		t.Fatal("POST directo de catálogo sigue expuesto")
	}
	proponer, ok := catalogo.resolver(http.MethodPost, bolsahttp.RutaPropuestasCausasParticipacion)
	if !ok || proponer.Clave != claveFronteraProponerCausasParticipacionBolsa || proponer.ClaveCapacidad != claveCapacidadProponerCausasParticipacionBolsa {
		t.Fatal("POST propuesta sin frontera nominal")
	}
	ref := bolsahttp.RutaPropuestasCausasParticipacion + "/propuesta:causa:abc"
	consultarPropuesta, ok := catalogo.resolver(http.MethodGet, ref)
	if !ok || consultarPropuesta.Clave != claveFronteraConsultarPropuestaCausasBolsa || consultarPropuesta.ClaveCapacidad != claveCapacidadConsultarPropuestaCausasBolsa {
		t.Fatal("GET propuesta sin frontera nominal")
	}
	publicar, ok := catalogo.resolver(http.MethodPost, ref+"/publicar")
	if !ok || publicar.Clave != claveFronteraPublicarCausasParticipacionBolsa || publicar.ClaveCapacidad != claveCapacidadPublicarCausasParticipacionBolsa || publicar.ClaveCapacidad == proponer.ClaveCapacidad {
		t.Fatal("POST publicación sin frontera nominal separada")
	}
	previas, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo("prf_bolsa_bback")
	if err != nil {
		t.Fatal(err)
	}
	catalogoPrevio, err := nuevoCatalogoFronterasComunDesarrollo(previas)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := catalogoPrevio.resolver(http.MethodGet, bolsahttp.RutaCatalogoCausasParticipacion); ok {
		t.Fatal("ruta B57 expuesta con selector apagado")
	}
}

func concesionB57Existe(concesiones []dominiovec.ConcesionRol, accion string) bool {
	for _, c := range concesiones {
		if c.Accion == accion && c.ModuloID == "bolsa" &&
			(c.TipoRecurso == puertosbolsa.TipoRecursoCatalogoCausasParticipacion || c.TipoRecurso == puertosbolsa.TipoRecursoPropuestaCatalogoCausasParticipacion) {
			return true
		}
	}
	return false
}
