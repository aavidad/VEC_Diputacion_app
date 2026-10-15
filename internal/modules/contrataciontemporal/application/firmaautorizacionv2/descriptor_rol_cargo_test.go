package firmaautorizacionv2

import (
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

func descriptorRolCargoPrueba() (ports.MaterialFirmaVerificadaV2, ports.DescriptorConstructorFirmaV2) {
	h := strings.Repeat("a", 64)
	m := ports.MaterialFirmaVerificadaV2{MaterialFirmaExterna: ports.MaterialFirmaExterna{
		OrganizacionRef: "org_prueba", ExpedienteRef: "exp_prueba", Documento: "informe_definitivo",
		CatalogoRef: "catalogo_prueba", CatalogoHuella: h, PasoRef: "paso_prueba", PasoOrden: 1,
		OriginalRef: "original_prueba", OriginalVersion: 1, OriginalHuella: h,
		FirmadoHuella: strings.Repeat("b", 64), CertificadoHuella: strings.Repeat("c", 64),
		PerfilFirmanteRef: "perfil_prueba", PerfilActivoFirmanteRef: "prf_prueba", UnidadFirmanteRef: "unidad_prueba",
		CargoFirmante: "car_prueba", DocumentoCustodiaRef: "custodia_prueba", DocumentoCustodiaVersion: 1,
	}, RolIDFirmante: "ct_cargo_tecnico_prueba", CatalogoVersion: 1, OrdenFirmaPDF: 1}
	d := ports.DescriptorConstructorFirmaV2{
		Esquema: "vec.competencia-firmante.constructor-ct.v1", CertificadoDERSHA256: m.CertificadoHuella,
		Seleccion: ports.SeleccionConstructorFirmaV2{PerfilEsperadoRef: m.PerfilFirmanteRef,
			PerfilActivoRef: m.PerfilActivoFirmanteRef, RolID: m.RolIDFirmante,
			CargoRef: m.CargoFirmante, EnlaceEjercicioRef: "enlace_prueba"},
		Recurso: vd.RecursoFirmaHistoricaV1{
			OrganizacionRef: m.OrganizacionRef, UnidadRef: m.UnidadFirmanteRef, ExpedienteRef: m.ExpedienteRef,
			DocumentoRef: m.OriginalRef, RecursoAutorizableRef: m.OriginalRef,
			ModuloID: ports.ModuloContratacion, TipoRecurso: "tipo_prueba", RecursoContextoSHA256: h,
			Original:         vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.OriginalRef, Version: m.OriginalVersion, HuellaSHA256: m.OriginalHuella},
			PDFRaizSHA256:    m.OriginalHuella,
			Firmado:          vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.DocumentoCustodiaRef, Version: m.DocumentoCustodiaVersion, HuellaSHA256: m.FirmadoHuella},
			PDFFirmadoSHA256: m.FirmadoHuella, NumeroFirmas: 1,
		},
		Circuito: vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.CatalogoRef, Version: m.CatalogoVersion, HuellaSHA256: m.CatalogoHuella},
		PasoRef:  m.PasoRef, PasoOrden: 1, Accion: "accion_prueba", Finalidad: "finalidad_prueba",
		Motivo: vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos_prueba", CatalogoVersion: 1, CatalogoHuellaSHA256: h, EntradaClave: "firma"},
	}
	return m, d
}

func TestDescriptorFirmaV2VinculaRolYCargoDelMaterial(t *testing.T) {
	m, d := descriptorRolCargoPrueba()
	if m.RolIDFirmante == m.CargoFirmante {
		t.Fatal("la prueba requiere identificadores distintos")
	}
	canon, err := CanonicoDescriptorFirmaVerificadaV2(m, d)
	if err != nil || ValidarDescriptorFirmaVerificadaV2(m, canon) != nil {
		t.Fatalf("descriptor auténtico rechazado: %v", err)
	}
	for nombre, cambiar := range map[string]func(*ports.MaterialFirmaVerificadaV2){
		"cargo_ajeno": func(x *ports.MaterialFirmaVerificadaV2) { x.CargoFirmante = "car_ajeno" },
		"rol_ajeno":   func(x *ports.MaterialFirmaVerificadaV2) { x.RolIDFirmante = "ct_cargo_ajeno" },
		"intercambio": func(x *ports.MaterialFirmaVerificadaV2) {
			x.RolIDFirmante, x.CargoFirmante = x.CargoFirmante, x.RolIDFirmante
		},
		"ambos_ajenos": func(x *ports.MaterialFirmaVerificadaV2) {
			x.RolIDFirmante, x.CargoFirmante = "ct_cargo_ajeno", "car_ajeno"
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alterado := m
			cambiar(&alterado)
			if !errors.Is(ValidarDescriptorFirmaVerificadaV2(alterado, canon), ports.ErrFirmaDocumentoDenegada) {
				t.Fatal("material alterado aceptó el descriptor auténtico")
			}
		})
	}
	alterado := d
	alterado.Seleccion.CargoRef = "car_ajeno"
	if _, err := CanonicoDescriptorFirmaVerificadaV2(m, alterado); !errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) {
		t.Fatal("descriptor de otro cargo aceptado")
	}
}
