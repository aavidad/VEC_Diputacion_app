package application

import (
	"testing"

	core "vec-diputacion-granada/internal/vec/domain"
)

func TestAdmisionFirmaSeleccionaConcesionUnicaSinReescribirRol(t *testing.T) {
	d := DescriptorAdmisionGarantiaActo{Accion: "contratacion_temporal.documento.firma_vec.registrar", ModuloID: "contratacion_temporal",
		TipoRecurso: "firma_vec_documento_contratacion_temporal", Finalidad: "gestionar_contratacion_temporal"}
	c := core.ConcesionRol{Accion: d.Accion, ModuloID: d.ModuloID, TipoRecurso: d.TipoRecurso,
		Finalidades: []string{d.Finalidad}, GarantiaMinima: core.AuthAssuranceSubstantial}
	otra := c
	otra.Accion = "contratacion_temporal.documento.firmas_r5_v2.consultar"
	rol := []core.ConcesionRol{otra, c}
	if !concesionUnicaAdmisionFirmaVec(rol, d) || len(rol) != 2 || rol[0].Accion != otra.Accion { t.Fatal("rol íntegro con dos permisos") }
	if concesionUnicaAdmisionFirmaVec([]core.ConcesionRol{otra}, d) { t.Fatal("acto ausente admitido") }
	if concesionUnicaAdmisionFirmaVec(append(rol, c), d) { t.Fatal("acto duplicado admitido") }
	alta := c
	alta.GarantiaMinima = core.AuthAssuranceHigh
	if concesionUnicaAdmisionFirmaVec([]core.ConcesionRol{alta}, d) { t.Fatal("garantía alta rebajada") }
	if concesionUnicaAdmisionFirmaVec(append(rol, alta), d) { t.Fatal("garantías ambiguas admitidas") }
	c.Finalidades = append(c.Finalidades, "otra_finalidad")
	if concesionUnicaAdmisionFirmaVec([]core.ConcesionRol{c}, d) { t.Fatal("finalidad ampliada admitida") }
}
