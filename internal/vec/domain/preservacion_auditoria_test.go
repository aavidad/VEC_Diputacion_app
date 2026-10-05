package domain

import (
	"strings"
	"testing"
	"time"
)

func TestPreservacionAcuseSegunEstado(t *testing.T) {
	s := SolicitudPreservacionAuditoria{PublicacionRef: "preservacion_" + strings.Repeat("1", 32), Version: 1, PreimagenSHA256: strings.Repeat("0", 64), DecisionTecnicaRef: "decision_tecnica_" + strings.Repeat("2", 32), Estado: "provisional", Medida: "conservar_todo_sin_expurgo"}
	a := AcusePreservacionAuditoria{AuditoriaRef: "aud_v3_pre_" + strings.Repeat("3", 32), Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64), RegistradaEn: time.Date(2026, 10, 4, 6, 0, 0, 0, time.UTC), CorrelacionRef: "correlacion_" + strings.Repeat("4", 32)}
	r := ResultadoPreservacionAuditoria{Estado: "publicada", Configuracion: s, ConfiguracionSHA256: strings.Repeat("b", 64), RegistradaEn: a.RegistradaEn, AcuseOriginal: a, AcuseAcceso: a}
	if r.Validar() != nil {
		t.Fatal("publicación válida rechazada")
	}
	for _, estado := range []string{"replay", "consultada"} {
		r.Estado = estado
		if r.Validar() == nil {
			t.Fatal("lectura sin nuevo acuse aceptada")
		}
		r.AcuseAcceso.Secuencia = 2
		if r.Validar() != nil {
			t.Fatal("lectura auditada rechazada")
		}
		r.AcuseAcceso = a
	}
	r.Estado = "publicada"
	r.AcuseAcceso.Secuencia = 2
	if r.Validar() == nil {
		t.Fatal("publicación con acuse distinto aceptada")
	}
}
