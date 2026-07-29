package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDetalleRRHHRechazaBloquesPrematuros(t *testing.T) {
	t.Parallel()
	inicial := nuevoEntornoConsultaRRHH(t)
	completo := nuevoEntornoConsultaRRHH(t)
	configurarDetalleCompletoRRHHPrueba(t, completo)
	inicial.sesion.detalle.Analisis = completo.sesion.detalle.Analisis
	inicial.sesion.detalle.Cobertura = completo.sesion.detalle.Cobertura
	inicial.sesion.detalle.Asignacion = completo.sesion.detalle.Asignacion
	inicial.sesion.detalle.Resumen.ModalidadClave =
		completo.sesion.detalle.Resumen.ModalidadClave
	inicial.sesion.detalle.Resumen.UnidadRef =
		completo.sesion.detalle.Resumen.UnidadRef
	if _, err := servicioDetalleRRHHPrueba(t, inicial).Consultar(
		context.Background(), inicial.detalle,
	); !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatalf("versión 1 aceptó bloques de fases futuras: %v", err)
	}
}

func TestDetalleRRHHRechazaBloqueAusenteTrasHito(t *testing.T) {
	t.Parallel()
	for _, bloque := range []string{"analisis", "cobertura", "asignacion"} {
		bloque := bloque
		t.Run(bloque, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			configurarDetalleCompletoRRHHPrueba(t, entorno)
			switch bloque {
			case "analisis":
				entorno.sesion.detalle.Analisis = nil
			case "cobertura":
				entorno.sesion.detalle.Cobertura = nil
			case "asignacion":
				entorno.sesion.detalle.Asignacion = nil
			}
			if _, err := servicioDetalleRRHHPrueba(t, entorno).Consultar(
				context.Background(), entorno.detalle,
			); !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("hitos sin bloque %s aceptados: %v", bloque, err)
			}
		})
	}
}

func TestDetalleRRHHRechazaBloqueNoCoincidenteConSuHito(t *testing.T) {
	t.Parallel()
	for _, caso := range []struct {
		nombre  string
		indice  int
		alterar func(*ports.HitoExpedienteRRHH)
	}{
		{"analisis", 1, func(h *ports.HitoExpedienteRRHH) {
			h.AccionClave = "analisis.distinto"
		}},
		{"cobertura", 2, func(h *ports.HitoExpedienteRRHH) {
			h.AccionClave = "cobertura.distinta"
		}},
		{"asignacion", 3, func(h *ports.HitoExpedienteRRHH) {
			h.AccionClave = "asignacion.distinta"
		}},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			configurarDetalleCompletoRRHHPrueba(t, entorno)
			caso.alterar(&entorno.sesion.detalle.Hitos[caso.indice])
			if _, err := servicioDetalleRRHHPrueba(t, entorno).Consultar(
				context.Background(), entorno.detalle,
			); !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("bloque no ligado a hito aceptado: %v", err)
			}
		})
	}
}

func TestDetalleRRHHHuellaRechazaMutacionesPublicas(t *testing.T) {
	t.Parallel()
	for _, caso := range []struct {
		nombre  string
		alterar func(*ports.DetalleExpedienteRRHH)
	}{
		{"fecha_asignacion", func(d *ports.DetalleExpedienteRRHH) {
			d.Asignacion.AsignadaEn = d.Asignacion.AsignadaEn.Add(time.Microsecond)
		}},
		{"unidad_y_resumen", func(d *ports.DetalleExpedienteRRHH) {
			d.Asignacion.UnidadRef = "unidad:rrhh:mutada"
			d.Resumen.UnidadRef = "unidad:rrhh:mutada"
		}},
		{"resultado_rc", func(d *ports.DetalleExpedienteRRHH) {
			d.Analisis.ResultadoRC = domain.RCRechazada
		}},
		{"via", func(d *ports.DetalleExpedienteRRHH) {
			d.Cobertura.ViaClave = "bolsa.alternativa"
		}},
		{"causa", func(d *ports.DetalleExpedienteRRHH) {
			d.Analisis.CausaClave = "causa.alternativa"
		}},
		{"jornada", func(d *ports.DetalleExpedienteRRHH) {
			d.Analisis.PorcentajeJornada = 5_000
		}},
		{"coste", func(d *ports.DetalleExpedienteRRHH) {
			d.Analisis.CostePrevisto.Centimos++
		}},
		{"fuente_coste", func(d *ports.DetalleExpedienteRRHH) {
			d.Analisis.FuenteCosteRef = "fuente:coste:mutada"
		}},
		{"comprobacion", func(d *ports.DetalleExpedienteRRHH) {
			d.Cobertura.Comprobaciones[0].Resultado =
				domain.ComprobacionNegativa
		}},
		{"hito", func(d *ports.DetalleExpedienteRRHH) {
			d.Hitos[1].AccionClave = "analisis.mutado"
		}},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			configurarDetalleCompletoRRHHPrueba(t, entorno)
			caso.alterar(&entorno.sesion.detalle)
			if _, err := servicioDetalleRRHHPrueba(t, entorno).Consultar(
				context.Background(), entorno.detalle,
			); !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("mutación pública %s aceptada: %v", caso.nombre, err)
			}
		})
	}
}

func TestDetalleRRHHHuellaSobreviveAlClonDefensivo(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	configurarDetalleCompletoRRHHPrueba(t, entorno)
	entorno.sesion.detalle = entorno.sesion.detalle.Clonar()
	if _, err := servicioDetalleRRHHPrueba(t, entorno).Consultar(
		context.Background(), entorno.detalle,
	); err != nil {
		t.Fatalf("el clon defensivo perdió validez: %v", err)
	}
}

func TestJSONCuadroYDetalleNoPublicaEvidenciaLectura(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	for nombre, valor := range map[string]any{
		"cuadro":  entorno.sesion.pagina,
		"detalle": entorno.sesion.detalle,
	} {
		contenido, err := json.Marshal(valor)
		if err != nil {
			t.Fatalf("%s no serializable: %v", nombre, err)
		}
		for _, prohibido := range []string{
			`"lectura"`, `"registrada_en"`, "lectura:rrhh:001",
			"auditoria:rrhh:001",
		} {
			if strings.Contains(string(contenido), prohibido) {
				t.Fatalf("%s publica %q: %s", nombre, prohibido, contenido)
			}
		}
	}
	if _, err := json.Marshal(entorno.sesion.detalle.Lectura); !errors.Is(
		err, ports.ErrMaterialConsultaRRHHSensible,
	) {
		t.Fatalf("el recibo interno es serializable directamente: %v", err)
	}
}

func TestDetalleRRHHAceptaReasignacionLigadaAlHitoVigente(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	configurarDetalleCompletoRRHHPrueba(t, entorno)
	instante := entorno.ahora.Add(30 * time.Second)
	observaciones := "Cambio motivado de unidad."
	reasignacion := domain.AsignacionUnidad{
		UnidadRef:       "unidad:rrhh:002",
		ResponsableRef:  "responsable:rrhh:002",
		NotificacionRef: "notificacion:rrhh:002",
		AsignadaEn:      instante, MotivoClave: "necesidad.servicio",
		Observaciones: observaciones,
	}
	actualizado, err := entorno.expediente.ReasignarUnidad(
		entorno.expediente.Version, reasignacion, domain.DatosActuacion{
			AccionClave: "unidad.reasignada", ActorRef: "actor:rrhh:002",
			UnidadRef: "unidad:rrhh:002", ReciboRef: "recibo:actuacion:002",
			RealizadaEn: instante, FaseDestino: entorno.expediente.FaseActual,
			EstadoDestino: entorno.expediente.EstadoActual,
			Observaciones: observaciones,
		},
	)
	if err != nil {
		t.Fatalf("reasignar: %v", err)
	}
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(
		actualizado.Referencia, actualizado.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	contexto := contextoConsultaRRHHV3Prueba(t, instante)
	capacidad := capacidadConsultaDetalleRRHHV3Prueba(
		t, contexto, solicitud, instante,
	)
	recibo := reciboConsultaRRHHPrueba(
		t, contexto, capacidad,
		instante, actualizado.Referencia, actualizado.Version, 1,
	)
	detalle, err := ports.NuevoDetalleExpedienteRRHH(actualizado, recibo)
	if err != nil {
		t.Fatalf("proyectar reasignación: %v", err)
	}
	entorno.expediente = actualizado
	entorno.contexto = contexto
	entorno.autoridad.contexto = contexto
	entorno.sesion.detalle = detalle
	entorno.detalle = solicitud
	entorno.reloj.instante = instante
	if _, err = servicioDetalleRRHHPrueba(t, entorno).Consultar(
		context.Background(), solicitud,
	); err != nil {
		t.Fatalf("reasignación vigente rechazada: %v", err)
	}
}

func configurarDetalleCompletoRRHHPrueba(
	t *testing.T,
	entorno *entornoConsultaRRHH,
) {
	t.Helper()
	expediente := entorno.expediente
	huella := strings.Repeat("b", 64)
	coste := domain.Importe{Centimos: 125_000, Moneda: "EUR"}
	analisis := domain.AnalisisRRHH{
		ModalidadClave:    "interinidad",
		CategoriaRef:      expediente.Solicitud.CategoriaRef,
		GrupoSubgrupo:     expediente.Solicitud.GrupoSubgrupo,
		CausaClave:        "sustitucion",
		Periodo:           expediente.Solicitud.Periodo,
		PorcentajeJornada: 10_000,
		EntradaRCEsperada: domain.VinculoEntradaRC{
			Referencia: "entrada:rc:rrhh:001", HuellaSHA256: huella,
		},
		ValidacionRC: domain.ValidacionRC{
			Resultado:           domain.RCNoRequerida,
			EntradaRef:          "entrada:rc:rrhh:001",
			HuellaEntradaSHA256: huella,
			FuenteRef:           "fuente:rc:rrhh:001",
			ReciboRef:           "recibo:rc:rrhh:001",
			ValidadaEn:          entorno.ahora,
			Motivo:              "No requerida para la prueba.",
		},
		CostePrevisto: &coste, FuenteCosteRef: "fuente:coste:001",
	}
	var err error
	expediente, err = expediente.RegistrarAnalisis(
		1, analisis, actuacionDetalleRRHHPrueba(
			"analisis.validado", "gestion_bolsa", entorno,
		),
	)
	if err != nil {
		t.Fatalf("registrar análisis: %v", err)
	}
	decision := domain.DecisionViaCobertura{
		ViaClave: "bolsa", ProcedimientoRef: "procedimiento:rrhh:001",
		BolsaRef: "bolsa:rrhh:001",
		Comprobaciones: []domain.ComprobacionCobertura{{
			Clave: "disponibilidad", Resultado: domain.ComprobacionAfirmativa,
			FuenteRef: "fuente:bolsa:001", ReciboRef: "recibo:bolsa:001",
			EvaluadaEn: entorno.ahora,
		}},
		Motivacion: "Existe una bolsa vigente.",
	}
	expediente, err = expediente.RegistrarViaCobertura(
		2, decision, actuacionDetalleRRHHPrueba(
			"cobertura.decidida", "asignacion_unidad", entorno,
		),
	)
	if err != nil {
		t.Fatalf("registrar cobertura: %v", err)
	}
	asignacion := domain.AsignacionUnidad{
		UnidadRef:       "unidad:rrhh:001",
		ResponsableRef:  "responsable:rrhh:001",
		NotificacionRef: "notificacion:rrhh:001",
		AsignadaEn:      entorno.ahora,
	}
	expediente, err = expediente.RegistrarAsignacion(
		3, asignacion, actuacionDetalleRRHHPrueba(
			"unidad.asignada", "unidad_gestora", entorno,
		),
	)
	if err != nil {
		t.Fatalf("registrar asignación: %v", err)
	}
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(
		expediente.Referencia, expediente.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadConsultaDetalleRRHHV3Prueba(
		t, entorno.contexto, solicitud, entorno.ahora,
	)
	recibo := reciboConsultaRRHHPrueba(
		t, entorno.contexto, capacidad,
		entorno.ahora, expediente.Referencia, expediente.Version, 1,
	)
	detalle, err := ports.NuevoDetalleExpedienteRRHH(expediente, recibo)
	if err != nil {
		t.Fatalf("proyectar detalle completo: %v", err)
	}
	entorno.expediente = expediente
	entorno.sesion.detalle = detalle
	entorno.detalle = solicitud
}

func actuacionDetalleRRHHPrueba(
	accion string,
	fase string,
	entorno *entornoConsultaRRHH,
) domain.DatosActuacion {
	return domain.DatosActuacion{
		AccionClave: domain.ClaveCatalogo(accion),
		ActorRef:    "actor:rrhh:001", UnidadRef: "unidad:rrhh:001",
		ReciboRef: "recibo:actuacion:001", RealizadaEn: entorno.ahora,
		FaseDestino:   domain.ClaveFase(fase),
		EstadoDestino: domain.EstadoEnCurso,
	}
}
