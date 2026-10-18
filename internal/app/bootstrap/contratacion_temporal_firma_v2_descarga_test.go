package bootstrap

import (
	"errors"
	"strings"
	"testing"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestSeleccionDescargaUsaPasoDelPerfilPropioSinRestringirLaFirmaPedida(t *testing.T) {
	pasos := []ctdomain.CompetenciaPasoFirmaV2{
		{Documento: "resolucion", RolID: "rol:jefatura", OrganizacionRef: "organizacion:central", UnidadRef: "unidad:rrhh",
			PasoOrden: 1, Circuito: ctdomain.VersionPlanFirmaV2{HuellaSHA256: strings.Repeat("a", 64)}},
		{Documento: "resolucion", RolID: "rol:secretaria", OrganizacionRef: "organizacion:central", UnidadRef: "unidad:rrhh",
			PasoOrden: 3, Circuito: ctdomain.VersionPlanFirmaV2{HuellaSHA256: strings.Repeat("a", 64)}},
	}
	orden, huella, err := seleccionarPasoDescargaFirmaV2(pasos, "resolucion", "rol:jefatura", "organizacion:central", "unidad:rrhh")
	if err != nil || orden != 1 || huella != strings.Repeat("a", 64) {
		t.Fatalf("la capacidad no usa el paso propio: orden=%d err=%v", orden, err)
	}
	// La firma_ref elegida se resolverá después en la historia autorizada;
	// aquí no se le atribuye al paso de Jefatura la firma de Secretaría.
}

func TestSeleccionDescargaDeniegaRolAmbitoYPlanAmbiguos(t *testing.T) {
	base := ctdomain.CompetenciaPasoFirmaV2{Documento: "resolucion", RolID: "rol:jefatura",
		OrganizacionRef: "organizacion:central", UnidadRef: "unidad:rrhh", PasoOrden: 1,
		Circuito: ctdomain.VersionPlanFirmaV2{HuellaSHA256: strings.Repeat("a", 64)}}
	casos := []struct {
		nombre      string
		pasos       []ctdomain.CompetenciaPasoFirmaV2
		rol, unidad string
	}{
		{"rol_retirado", []ctdomain.CompetenciaPasoFirmaV2{base}, "rol:retirado", "unidad:rrhh"},
		{"unidad_ajena", []ctdomain.CompetenciaPasoFirmaV2{base}, "rol:jefatura", "unidad:ajena"},
		{"paso_ambiguo", []ctdomain.CompetenciaPasoFirmaV2{base, base}, "rol:jefatura", "unidad:rrhh"},
		{"paso_fuera_de_r5", []ctdomain.CompetenciaPasoFirmaV2{func() ctdomain.CompetenciaPasoFirmaV2 { p := base; p.PasoOrden = 4; return p }()}, "rol:jefatura", "unidad:rrhh"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, _, err := seleccionarPasoDescargaFirmaV2(caso.pasos, "resolucion", caso.rol, "organizacion:central", caso.unidad)
			if !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
				t.Fatalf("paso no autorizado: %v", err)
			}
		})
	}
}
