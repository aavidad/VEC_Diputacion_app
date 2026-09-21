package ports

import "testing"

func TestContratoBorradorDeclaraAccionesFinalidadesYAudienciasSeparadas(t *testing.T) {
	if AccionCrearBorradorLlamamientoInterno == AccionConsultarBorradorLlamamientoInterno || FinalidadCrearBorradorLlamamientoInterno == FinalidadConsultarBorradorLlamamientoInterno || AudienciaCrearBorradorLlamamientoInterno == AudienciaConsultarBorradorLlamamientoInterno {
		t.Fatal("capacidad de crear y consultar mezclada")
	}
}
