package postgres

import (
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestArgumentosSQLCuadroFiltradoConservanOrdenYValoresTipados(t *testing.T) {
	s, err := ports.NuevaSolicitudCuadroRRHHFiltrada("2026/CT", "centro:desarrollo:001",
		"categoria:desarrollo:c2", []domain.EstadoOperativo{domain.EstadoEnCurso, domain.EstadoCompletado},
		[]domain.ClaveFase{"solicitud", "fiscalizacion"}, 25, "")
	if err != nil {
		t.Fatal(err)
	}
	s = s.ConResumen()
	args := argumentosSQLCuadroFiltradoRRHH("org:desarrollo:001", "organizacion", "org:desarrollo:001",
		s, argumentosMaterialConsultaRRHH{})
	if len(args) != 21 || args[3] != "2026/CT" || args[4] != "centro:desarrollo:001" ||
		args[5] != "categoria:desarrollo:c2" || args[8] != int16(25) || args[9] != "" || args[10] != true ||
		!reflect.DeepEqual(args[6], []string{"completado", "en_curso"}) ||
		!reflect.DeepEqual(args[7], []string{"fiscalizacion", "solicitud"}) {
		t.Fatalf("argumentos v2 incorrectos: longitud=%d", len(args))
	}
	if !strings.Contains(consultaCuadroFiltradoRRHHPostgreSQL,
		"vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v6(") ||
		!strings.Contains(consultaCuadroFiltradoRRHHPostgreSQL,
			"vec_contratacion_temporal.consulta_cuadro_rrhh_v2") {
		t.Fatal("la sentencia no llama a la fachada v2 nominal")
	}
}
