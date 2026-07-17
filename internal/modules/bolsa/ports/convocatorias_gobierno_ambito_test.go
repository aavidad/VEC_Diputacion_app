package ports

import (
	"errors"
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

func TestRecursoMutacionDerivaAmbitosExactosDeVersionConfirmada(t *testing.T) {
	version := versionGobernadaPuertosPrueba(t)
	sellado := atestacionMotivoConvocatoriaPrueba(
		t, AccionCrearBorradorConvocatoria, version.Referencia(), 'a',
	)
	material, err := MaterialAltaBorradorConvocatoria(version, nil, nil, sellado)
	if err != nil {
		t.Fatal(err)
	}
	recurso, err := RecursoAutorizableMutacionConvocatoria(material, version)
	if err != nil || len(recurso.Ambitos) != 2 ||
		recurso.Ambitos["organizacion_ref"] != version.AmbitoOrganizativo.OrganizacionRef() ||
		recurso.Ambitos["unidad_gestion_ref"] != version.AmbitoOrganizativo.UnidadGestionRef() {
		t.Fatalf("ambitos no derivados de la version: %#v / %v", recurso.Ambitos, err)
	}
	for clave := range recurso.Ambitos {
		if clave != "organizacion_ref" && clave != "unidad_gestion_ref" {
			t.Fatalf("ambito libre admitido: %q", clave)
		}
	}

	otroAmbito, err := dominiobolsa.NuevoAmbitoOrganizativoConvocatoria(
		"org_diputaciongranada", "uni_gestionrecursoshumanos",
	)
	if err != nil {
		t.Fatal(err)
	}
	versionAjena := version
	versionAjena.AmbitoOrganizativo = otroAmbito
	versionAjena, err = versionAjena.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RecursoAutorizableMutacionConvocatoria(
		material, versionAjena,
	); !errors.Is(err, ErrAutorizacionGobiernoConvocatoriaInvalida) {
		t.Fatalf("material se aplico a una version de otro ambito: %v", err)
	}
}
