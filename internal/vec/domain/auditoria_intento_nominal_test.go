package domain

import (
	"strings"
	"testing"
)

func TestExitoIntentoNominalSoloLecturasDocumentosBolsaRRHH(t *testing.T) {
	base := DatosIntentoAuditoria{
		Accion: "lectura_bolsas_rrhh_consultar", ModuloID: "bolsa",
		RecursoRef: "bolsa:constituida:administrativo", FinalidadRef: "gestion_bolsa_rrhh",
		Resultado: ResultadoIntentoAuditoriaExito,
		Motivo: ReferenciaEntradaCatalogo{CatalogoID: "motivos_auditoria", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "lectura_acreditada"},
		Proceso: "vec-rrhh", Canal: "interna_corporativa",
		CorrelacionRef: "correlacion_11111111111111111111111111111111",
	}
	for _, accion := range []string{"lectura_bolsas_rrhh_consultar", "lectura_bolsas_rrhh_descargar_recibo"} {
		d := base
		d.Accion = accion
		if err := d.Validar(); err != nil {
			t.Fatalf("lectura %s rechazada: %v", accion, err)
		}
	}
	for nombre, cambio := range map[string]func(*DatosIntentoAuditoria){
		"otra acción":      func(d *DatosIntentoAuditoria) { d.Accion = "lectura_bolsas_rrhh_exportar" },
		"otro módulo":      func(d *DatosIntentoAuditoria) { d.ModuloID = "personal" },
		"otra finalidad":   func(d *DatosIntentoAuditoria) { d.FinalidadRef = "gestion_personal" },
		"otro proceso":     func(d *DatosIntentoAuditoria) { d.Proceso = "vec-portal-personal" },
		"otro canal":       func(d *DatosIntentoAuditoria) { d.Canal = "externa_personal" },
		"recurso ajeno":    func(d *DatosIntentoAuditoria) { d.RecursoRef = "persona:01" },
		"recurso no opaco": func(d *DatosIntentoAuditoria) { d.RecursoRef = "bolsa:alberto@example.org" },
	} {
		d := base
		cambio(&d)
		if err := d.Validar(); err == nil {
			t.Errorf("exito de %s admitido", nombre)
		}
	}
	for _, resultado := range []ResultadoIntentoAuditoria{ResultadoIntentoAuditoriaDenegado, ResultadoIntentoAuditoriaError} {
		d := base
		d.Resultado, d.Accion, d.ModuloID = resultado, "otra_lectura", "vec"
		if err := d.Validar(); err != nil {
			t.Errorf("resultado anterior %s rechazado: %v", resultado, err)
		}
	}
}
