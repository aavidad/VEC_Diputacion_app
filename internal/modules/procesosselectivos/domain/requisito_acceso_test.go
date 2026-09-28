package domain

import (
	"errors"
	"strings"
	"testing"
)

func requisitoOPEPrueba() RequisitoAcceso {
	return RequisitoAcceso{
		Referencia: "req:titulacion:1", BasesRef: "bases:ope:2026", BasesVersion: 2,
		BasesHuellaSHA256: strings.Repeat("a", 64), TipoProceso: ProcesoOPE,
		Clase: RequisitoTitulacion, CodigoEstructurado: "titulo:grado:ejemplo",
		Hito: HitoPrueba, FechaLimite: "2027-06-15", PermiteTituloPendiente: true,
		EvidenciaPrevisionRef: "tipo-evidencia:matricula",
	}
}

func TestPrevisionSoloEsCondicionalEnBasesOPEExpresas(t *testing.T) {
	r := requisitoOPEPrueba()
	hecho := &HechoAcceso{
		CodigoEstructurado: r.CodigoEstructurado, Estado: HechoDeclarado,
		FuenteRef: "rum:declaracion:1", PrevisionEn: "2027-05-01",
		EvidenciaPrevisionRef: r.EvidenciaPrevisionRef,
	}
	resultado, err := EvaluarAcceso(r, hecho)
	if err != nil || resultado.Estado != EvaluacionPendiente || !resultado.Condicional || resultado.Motivo != MotivoPrevision {
		t.Fatalf("prevision OPE = %+v, %v", resultado, err)
	}

	r.TipoProceso = ProcesoBolsaInmediata
	if !errors.Is(r.Validar(), ErrRequisitoAccesoInvalido) {
		t.Fatal("una bolsa inmediata no puede publicar la excepcion")
	}
	r.PermiteTituloPendiente = false
	r.EvidenciaPrevisionRef = ""
	resultado, err = EvaluarAcceso(r, hecho)
	if err != nil || resultado.Estado != EvaluacionPendiente || resultado.Condicional {
		t.Fatalf("prevision en bolsa = %+v, %v", resultado, err)
	}
}

func TestHechoAcreditadoUsaFechaDeBasesYNoPuntuacion(t *testing.T) {
	r := requisitoOPEPrueba()
	hecho := &HechoAcceso{
		CodigoEstructurado: r.CodigoEstructurado, Estado: HechoAcreditado,
		FuenteRef: "rum:hecho:1", EvidenciaRef: "documento:1", CumplidoEn: r.FechaLimite,
	}
	resultado, err := EvaluarAcceso(r, hecho)
	if err != nil || resultado.Estado != EvaluacionCumple || resultado.Motivo != MotivoAcreditado {
		t.Fatalf("acreditado en hito = %+v, %v", resultado, err)
	}
	hecho.CumplidoEn = "2027-06-16"
	resultado, err = EvaluarAcceso(r, hecho)
	if err != nil || resultado.Estado != EvaluacionNoCumple || resultado.Motivo != MotivoAcreditadoTarde {
		t.Fatalf("acreditado tras hito = %+v, %v", resultado, err)
	}
}

func TestTextoLibreYDatoAusentePermanecenPendientes(t *testing.T) {
	r := requisitoOPEPrueba()
	r.CodigoEstructurado = ""
	resultado, err := EvaluarAcceso(r, nil)
	if err != nil || resultado.Estado != EvaluacionPendiente || resultado.Motivo != MotivoTextoSinCriterio {
		t.Fatalf("texto libre = %+v, %v", resultado, err)
	}
	r.CodigoEstructurado = "titulo:grado:ejemplo"
	resultado, err = EvaluarAcceso(r, nil)
	if err != nil || resultado.Estado != EvaluacionPendiente || resultado.Motivo != MotivoFaltaDato {
		t.Fatalf("dato ausente = %+v, %v", resultado, err)
	}
}

func TestExcepcionExigeHitoFechaYEvidenciaVersionados(t *testing.T) {
	r := requisitoOPEPrueba()
	for nombre, cambiar := range map[string]func(*RequisitoAcceso){
		"fecha no valida":  func(r *RequisitoAcceso) { r.FechaLimite = "2027-02-30" },
		"hito no conocido": func(r *RequisitoAcceso) { r.Hito = "automatico" },
		"sin evidencia":    func(r *RequisitoAcceso) { r.EvidenciaPrevisionRef = "" },
		"sin version":      func(r *RequisitoAcceso) { r.BasesVersion = 0 },
		"sin huella":       func(r *RequisitoAcceso) { r.BasesHuellaSHA256 = "" },
	} {
		t.Run(nombre, func(t *testing.T) {
			copia := r
			cambiar(&copia)
			if !errors.Is(copia.Validar(), ErrRequisitoAccesoInvalido) {
				t.Fatal("aceptado sin condiciones de bases completas")
			}
		})
	}
}
