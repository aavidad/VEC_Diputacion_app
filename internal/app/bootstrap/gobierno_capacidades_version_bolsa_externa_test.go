package bootstrap

import (
	"testing"

	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
)

func TestConjuntoSeisConservaB1YAnadeExternoSegregado(t *testing.T) {
	cinco, ok := AudienciasConjuntoCapacidadesAdmin(5)
	if !ok {
		t.Fatal("falta conjunto B1")
	}
	seis, ok := AudienciasConjuntoCapacidadesAdmin(6)
	if !ok || len(seis) != len(cinco)+2 {
		t.Fatalf("conjunto 6 inválido: %+v", seis)
	}
	for i := range cinco {
		if seis[i] != cinco[i] {
			t.Fatalf("conjunto 6 alteró audiencia anterior %d", i)
		}
	}
	if seis[len(cinco)] != (AudienciaCapacidadAdmin{pg.AudienciaVersionarRolBolsaExternaProponer,
		SegmentoVersionBolsaExternaPropuesta, EmisorVersionBolsaExternaAdmin}) ||
		seis[len(cinco)+1] != (AudienciaCapacidadAdmin{pg.AudienciaVersionarRolBolsaExternaAprobar,
			SegmentoVersionBolsaExternaCierre, EmisorVersionBolsaExternaAdmin}) ||
		EmisorVersionBolsaExternaAdmin == EmisorVersionBolsaAdmin {
		t.Fatal("audiencias externas no segregadas")
	}
}
