package ports_test

import (
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaCuadroV2RecursoCapacidadYOrdenCompartenDominio(t *testing.T) {
	ahora := instantePuertosRRHH()
	_, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	solicitud, err := ports.NuevaSolicitudCuadroRRHHFiltrada("", "centro:rrhh:001", "",
		[]domain.EstadoOperativo{domain.EstadoEnCurso}, []domain.ClaveFase{"solicitud"}, 20, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = ports.NuevosRecursosConsultaCuadroRRHH(contexto, solicitud, ahora)
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadCuadroPuertosRRHH(t, contexto, solicitud, ahora)
	if capacidad.ConsultaDominio() != ports.DominioHuellaConsultaCuadroRRHHV2 {
		t.Fatal("la capacidad no liga el dominio v2")
	}
	orden, err := ports.NuevaOrdenConsultaCuadroRRHH(contexto, capacidad, solicitud, ahora)
	if err != nil {
		t.Fatal(err)
	}
	canon, err := orden.ExportarConsultaCanonicaParaSQL()
	if err != nil || canon.Dominio() != ports.DominioHuellaConsultaCuadroRRHHV2 ||
		canon.HuellaSHA256() != orden.ConsultaHuellaSHA256() {
		t.Fatalf("orden/canon v2 incoherentes: %v", err)
	}
}
