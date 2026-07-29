package ports_test

import (
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestPaginaCuadroRRHHValidaContenidoPublicableContraSolicitud(t *testing.T) {
	solicitud, err := ports.NuevaSolicitudCuadroRRHH(
		"2026/CT",
		domain.EstadoEnCurso,
		domain.ClaveFase("solicitud"),
		1,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	pagina := paginaContenidoCuadroRRHHPrueba("")
	if err := pagina.ValidarContenidoPublicablePara(solicitud); err != nil {
		t.Fatalf("contenido nominal rechazado: %v", err)
	}
}

func TestPaginaCuadroRRHHRechazaContenidoPublicableDivergente(t *testing.T) {
	base := paginaContenidoCuadroRRHHPrueba("")
	segundo := base.Expedientes[0]
	segundo.ExpedienteRef = "expediente:rrhh:002"
	segundo.NumeroVisible = "2026/CT-002"
	segundo.ActualizadoEn = segundo.ActualizadoEn.Add(-time.Minute)

	solicitudBase, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 2, "")
	if err != nil {
		t.Fatal(err)
	}
	solicitudLimiteUno, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	solicitudTextoAjeno, err := ports.NuevaSolicitudCuadroRRHH("2025", "", "", 2, "")
	if err != nil {
		t.Fatal(err)
	}
	solicitudEstadoAjeno, err := ports.NuevaSolicitudCuadroRRHH(
		"", domain.EstadoCompletado, "", 2, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudFaseAjena, err := ports.NuevaSolicitudCuadroRRHH(
		"", "", domain.ClaveFase("asignacion"), 2, "",
	)
	if err != nil {
		t.Fatal(err)
	}

	casos := []struct {
		nombre    string
		pagina    ports.PaginaCuadroRRHH
		solicitud ports.SolicitudCuadroRRHH
	}{
		{
			"cursor corrupto",
			func() ports.PaginaCuadroRRHH {
				pagina := clonarPaginaContenidoCuadroRRHH(base)
				pagina.HayMas = true
				pagina.CursorSiguiente = "cursor-no-canonico"
				return pagina
			}(),
			solicitudBase,
		},
		{
			"sobreentrega",
			func() ports.PaginaCuadroRRHH {
				pagina := clonarPaginaContenidoCuadroRRHH(base)
				pagina.Expedientes = append(pagina.Expedientes, segundo)
				return pagina
			}(),
			solicitudLimiteUno,
		},
		{"filtro de texto", base, solicitudTextoAjeno},
		{"filtro de estado", base, solicitudEstadoAjeno},
		{"filtro de fase", base, solicitudFaseAjena},
		{
			"duplicado",
			func() ports.PaginaCuadroRRHH {
				pagina := clonarPaginaContenidoCuadroRRHH(base)
				pagina.Expedientes = append(
					pagina.Expedientes,
					pagina.Expedientes[0],
				)
				return pagina
			}(),
			solicitudBase,
		},
		{
			"orden inverso",
			func() ports.PaginaCuadroRRHH {
				pagina := clonarPaginaContenidoCuadroRRHH(base)
				primero := pagina.Expedientes[0]
				primero.ActualizadoEn = segundo.ActualizadoEn.Add(-time.Minute)
				pagina.Expedientes = []ports.ResumenExpedienteRRHH{
					primero,
					segundo,
				}
				return pagina
			}(),
			solicitudBase,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if err := caso.pagina.ValidarContenidoPublicablePara(
				caso.solicitud,
			); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("contenido divergente aceptado: %v", err)
			}
		})
	}
}
