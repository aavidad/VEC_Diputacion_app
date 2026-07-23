package domain

import (
	"errors"
	"testing"
	"time"
)

var (
	instanteBase = time.Date(2026, 7, 23, 8, 30, 0, 0, time.UTC)
	fechaInicio  = time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	fechaFin     = time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)
)

func TestExpedienteRecorrePrimerasFasesSinClavesFuncionalesCompiladas(t *testing.T) {
	expediente := expedienteValido(t)
	analisis := analisisValido()
	siguiente, err := expediente.RegistrarAnalisis(1, analisis, actuacion(
		"analisis.validado", "gestion_bolsa", instanteBase.Add(time.Minute),
	))
	if err != nil {
		t.Fatalf("registrar análisis: %v", err)
	}
	decision := decisionValida()
	siguiente, err = siguiente.RegistrarViaCobertura(2, decision, actuacion(
		"cobertura.decidida", "asignacion_unidad", instanteBase.Add(2*time.Minute),
	))
	if err != nil {
		t.Fatalf("registrar vía: %v", err)
	}
	asignacion := AsignacionUnidad{
		UnidadRef: "uni_gestion_social", ResponsableRef: "cuenta_responsable_01",
		NotificacionRef: "notificacion_asignacion_01",
		AsignadaEn:      instanteBase.Add(3 * time.Minute),
	}
	siguiente, err = siguiente.RegistrarAsignacion(3, asignacion, actuacion(
		"unidad.asignada", "unidad_gestora", asignacion.AsignadaEn,
	))
	if err != nil {
		t.Fatalf("registrar asignación: %v", err)
	}
	if siguiente.Version != 4 || len(siguiente.Actuaciones) != 4 ||
		siguiente.FaseActual != "unidad_gestora" || siguiente.Asignacion == nil {
		t.Fatalf("expediente final inesperado: %#v", siguiente)
	}
}

func TestExpedienteRechazaConflictoYOrdenInvalido(t *testing.T) {
	expediente := expedienteValido(t)
	if _, err := expediente.RegistrarViaCobertura(1, decisionValida(), actuacion(
		"cobertura.decidida", "asignacion_unidad", instanteBase.Add(time.Minute),
	)); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("vía sin análisis: %v", err)
	}
	if _, err := expediente.RegistrarAnalisis(99, analisisValido(), actuacion(
		"analisis.validado", "gestion_bolsa", instanteBase.Add(time.Minute),
	)); !errors.Is(err, ErrVersionEnConflicto) {
		t.Fatalf("conflicto: %v", err)
	}
}

func TestExpedienteClonaColeccionesYValoresOpcionales(t *testing.T) {
	expediente := expedienteValido(t)
	clon := expediente.Clonar()
	clon.Solicitud.DocumentosAdjuntos[0] = "documento_alterado_01"
	clon.Actuaciones[0].DocumentosRef[0] = "documento_alterado_02"
	if expediente.Solicitud.DocumentosAdjuntos[0] == clon.Solicitud.DocumentosAdjuntos[0] ||
		expediente.Actuaciones[0].DocumentosRef[0] == clon.Actuaciones[0].DocumentosRef[0] {
		t.Fatal("el clon comparte memoria mutable")
	}
}

func TestDeclaracionRCSinCreditoNoAdmiteDatosResiduales(t *testing.T) {
	declaracion := DeclaracionRC{Existe: false, Numero: "rc_2026_0001"}
	if declaracion.Validar() == nil {
		t.Fatal("se aceptó una RC inexistente con número")
	}
}

func expedienteValido(t *testing.T) Expediente {
	t.Helper()
	alta := AltaExpediente{
		Referencia: "exp_temporal_2026_5487", NumeroVisible: "2026/5487",
		Flujo: ReferenciaFlujo{
			DefinicionRef: "flujo_contratacion_temporal",
			Version:       3, HuellaSHA256: cadena64("a"),
		},
		FaseInicial: "solicitud",
		Solicitud: SolicitudCentro{
			CentroRef: "centro_social_01", ContactoRef: "persona_contacto_01",
			CategoriaRef: "categoria_trabajador_social", GrupoSubgrupo: "A2",
			MotivoClave: "sustitucion_it", Detalle: "Sustitución temporal de la persona titular.",
			Periodo: PeriodoPrevisto{Inicio: fechaInicio, Fin: fechaFin},
			RC: DeclaracionRC{
				Existe: true, Numero: "rc_2026_0001", Fecha: fechaInicio,
				Importe:      Importe{Centimos: 3_245_000, Moneda: "EUR"},
				DocumentoRef: "documento_rc_01",
			},
			DocumentosAdjuntos: []string{"documento_peticion_01"},
		},
		Actuacion: actuacion("solicitud.registrada", "solicitud", instanteBase),
	}
	alta.Actuacion.DocumentosRef = []string{"documento_peticion_01"}
	expediente, err := NuevoExpediente(alta)
	if err != nil {
		t.Fatalf("nuevo expediente: %v", err)
	}
	return expediente
}

func analisisValido() AnalisisRRHH {
	coste := Importe{Centimos: 3_148_025, Moneda: "EUR"}
	return AnalisisRRHH{
		ModalidadClave: "sustitucion", CategoriaRef: "categoria_trabajador_social",
		GrupoSubgrupo: "A2", CausaClave: "incapacidad_temporal",
		Periodo:           PeriodoPrevisto{Inicio: fechaInicio, Fin: fechaFin},
		PorcentajeJornada: 10000,
		ValidacionRC: ValidacionRC{
			Resultado: RCValidada, FuenteRef: "fuente_presupuestaria_01",
			ReciboRef: "recibo_validacion_rc_01", ValidadaEn: instanteBase,
			Numero: "rc_2026_0001", Importe: Importe{Centimos: 3_245_000, Moneda: "EUR"},
			DocumentoRef: "documento_rc_01",
		},
		CostePrevisto: &coste, FuenteCosteRef: "tabla_retributiva_2026_v3",
	}
}

func decisionValida() DecisionViaCobertura {
	return DecisionViaCobertura{
		ViaClave: "bolsa_vigente", ProcedimientoRef: "procedimiento_bolsa_01",
		BolsaRef: "bolsa_trabajo_social_2026",
		Comprobaciones: []ComprobacionCobertura{
			{
				Clave: "existe_bolsa_vigente", Resultado: ComprobacionAfirmativa,
				FuenteRef: "fuente_bolsa_01", ReciboRef: "recibo_comprobacion_01",
				EvaluadaEn: instanteBase,
			},
			{
				Clave: "requiere_oferta_sae", Resultado: ComprobacionNegativa,
				FuenteRef: "fuente_reglas_cobertura_01", ReciboRef: "recibo_comprobacion_02",
				EvaluadaEn: instanteBase,
			},
		},
		Motivacion: "La bolsa vigente dispone de candidaturas elegibles.",
	}
}

func actuacion(accion string, fase string, instante time.Time) DatosActuacion {
	return DatosActuacion{
		AccionClave: ClaveCatalogo(accion), ActorRef: "actor_rrhh_01",
		UnidadRef: "unidad_rrhh_01", ReciboRef: "recibo_" + accion,
		RealizadaEn: instante, FaseDestino: ClaveFase(fase),
		EstadoDestino: EstadoEnCurso,
	}
}

func cadena64(caracter string) string {
	var resultado string
	for range 64 {
		resultado += caracter
	}
	return resultado
}
