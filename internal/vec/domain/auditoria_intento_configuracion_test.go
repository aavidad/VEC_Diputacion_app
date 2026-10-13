package domain

import (
	"strings"
	"testing"
)

func TestExitoBolsaSoloAdmiteParejasDeAccionYFinalidad(t *testing.T) {
	base := DatosIntentoAuditoria{ModuloID: "bolsa", Proceso: "vec-rrhh", Canal: "interna_corporativa", RecursoRef: "bolsa:perfil:" + strings.Repeat("a", 64), Resultado: ResultadoIntentoAuditoriaExito}
	for _, par := range [][2]string{{"lectura_bolsas_rrhh_consultar", "gestion_bolsa_rrhh"}, {"lectura_bolsas_rrhh_descargar_recibo", "gestion_bolsa_rrhh"}, {"bolsa.configuracion.consultar", "consulta_configuracion_bolsa_rrhh"}, {"bolsa.origen_ct.consultar", "gestionar_contratacion_temporal"}} {
		d := base
		d.Accion, d.FinalidadRef = par[0], par[1]
		if !resultadoIntentoAuditoriaValido(d) {
			t.Fatalf("pareja real rechazada: %s/%s", par[0], par[1])
		}
		d.FinalidadRef = "finalidad:ajena"
		if resultadoIntentoAuditoriaValido(d) {
			t.Fatalf("finalidad ajena admitida: %s", par[0])
		}
	}
	d := base
	d.Accion, d.FinalidadRef = "bolsa.configuracion.consultar", "consulta_configuracion_bolsa_rrhh"
	d.RecursoRef = "bolsa:otra"
	if resultadoIntentoAuditoriaValido(d) {
		t.Fatal("configuración admitió recurso distinto del perfil")
	}
	d.RecursoRef = base.RecursoRef
	d.Canal = "externa_personal"
	if resultadoIntentoAuditoriaValido(d) {
		t.Fatal("canal exterior admitido")
	}
}
