package domain

import (
	"errors"
	"testing"
	"time"
)

func TestRegistrarFiscalizacionResultadosYRetornoDerivado(t *testing.T) {
	casos := []struct {
		nombre        string
		resultado     ResultadoFiscalizacion
		observaciones string
		fase          ClaveFase
		estado        EstadoOperativo
		retorno       bool
	}{
		{"favorable", FiscalizacionFavorable, "", FaseFiscalizacion, EstadoEnCurso, false},
		{"favorable con observaciones", FiscalizacionFavorableConObservaciones,
			"Fiscalización favorable con condición sintética.", FaseFiscalizacion, EstadoEnCurso, false},
		{"desfavorable", FiscalizacionDesfavorable,
			"Reparo sintético que requiere subsanación.", FaseSubsanacionUnidad, EstadoIncidencia, true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			anterior := expedienteFiscalizablePrueba(t)
			instante := anterior.ActualizadoEn.Add(time.Minute)
			retornoRef := ""
			if caso.retorno {
				retornoRef = "retorno:fiscalizacion:sintetico:01"
			}
			actuacion := DatosActuacion{
				AccionClave: AccionRegistrarFiscalizacion,
				ActorRef:    "actor:intervencion:sintetico:01",
				UnidadRef:   "unidad:intervencion:sintetica:01",
				ReciboRef:   "recibo:fiscalizacion:sintetico:01",
				RealizadaEn: instante, FaseDestino: caso.fase,
				EstadoDestino: caso.estado, Observaciones: caso.observaciones,
				DocumentosRef: []string{anterior.InformeJuridico.DocumentoRef},
			}
			siguiente, err := anterior.RegistrarFiscalizacion(
				5,
				DatosRegistrarFiscalizacion{
					FiscalizacionRef:       "fiscalizacion:sintetica:01",
					Resultado:              caso.resultado,
					UnidadFiscalizadoraRef: actuacion.UnidadRef,
					Observaciones:          caso.observaciones,
					FiscalizadaEn:          instante,
					RetornoRef:             retornoRef,
				},
				actuacion,
			)
			if err != nil {
				t.Fatalf("registrar fiscalización: %v", err)
			}
			if siguiente.Validar() != nil || siguiente.Version != 6 ||
				siguiente.FaseActual != caso.fase || siguiente.EstadoActual != caso.estado ||
				len(siguiente.Actuaciones) != 6 || siguiente.Fiscalizacion == nil {
				t.Fatalf("expediente fiscalizado inesperado: %#v", siguiente)
			}
			if anterior.Fiscalizacion != nil || anterior.Version != 5 ||
				anterior.Asignacion.UnidadRef != siguiente.Asignacion.UnidadRef {
				t.Fatal("la transición mutó la instantánea anterior o la asignación")
			}
			if caso.retorno {
				retorno := siguiente.Fiscalizacion.Retorno
				if retorno == nil || retorno.UnidadRef != anterior.Asignacion.UnidadRef ||
					retorno.ResponsableRef != anterior.Asignacion.ResponsableRef ||
					retorno.Estado != EstadoRetornoFiscalizacionPendiente {
					t.Fatalf("retorno no derivado de asignación: %#v", retorno)
				}
			} else if siguiente.Fiscalizacion.Retorno != nil {
				t.Fatal("un resultado favorable creó retorno")
			}
		})
	}
}

func TestRegistrarFiscalizacionRechazaCombinacionesYAdulteracion(t *testing.T) {
	base := expedienteFiscalizablePrueba(t)
	instante := base.ActualizadoEn.Add(time.Minute)
	actuacion := DatosActuacion{
		AccionClave: AccionRegistrarFiscalizacion,
		ActorRef:    "actor:intervencion:sintetico:01",
		UnidadRef:   "unidad:intervencion:sintetica:01",
		ReciboRef:   "recibo:fiscalizacion:sintetico:01",
		RealizadaEn: instante, FaseDestino: FaseSubsanacionUnidad,
		EstadoDestino: EstadoIncidencia,
		Observaciones: "Reparo sintético.",
		DocumentosRef: []string{base.InformeJuridico.DocumentoRef},
	}
	datos := DatosRegistrarFiscalizacion{
		FiscalizacionRef:       "fiscalizacion:sintetica:01",
		Resultado:              FiscalizacionDesfavorable,
		UnidadFiscalizadoraRef: actuacion.UnidadRef,
		Observaciones:          actuacion.Observaciones,
		FiscalizadaEn:          instante,
		RetornoRef:             "retorno:fiscalizacion:sintetico:01",
	}
	siguiente, err := base.RegistrarFiscalizacion(5, datos, actuacion)
	if err != nil {
		t.Fatalf("registrar base: %v", err)
	}

	adulterado := siguiente.Clonar()
	adulterado.Fiscalizacion.Retorno.UnidadRef = "unidad:inyectada:sintetica:01"
	if !errors.Is(adulterado.Validar(), ErrExpedienteInvalido) {
		t.Fatal("se aceptó un retorno desligado de su actuación")
	}

	datos.Resultado = FiscalizacionFavorable
	datos.Observaciones = "no permitida"
	datos.RetornoRef = ""
	actuacion.FaseDestino = FaseFiscalizacion
	actuacion.EstadoDestino = EstadoEnCurso
	actuacion.Observaciones = datos.Observaciones
	if _, err := base.RegistrarFiscalizacion(5, datos, actuacion); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("favorable con observaciones aceptado: %v", err)
	}
}

func expedienteFiscalizablePrueba(t *testing.T) Expediente {
	t.Helper()
	expediente := expedienteValido(t)
	var err error
	expediente, err = expediente.RegistrarAnalisis(
		expediente.Version, analisisValido(),
		actuacion("analisis.validado", "gestion_bolsa", expediente.ActualizadoEn.Add(time.Minute)),
	)
	if err != nil {
		t.Fatal(err)
	}
	expediente, err = expediente.RegistrarViaCobertura(
		expediente.Version, decisionValida(),
		actuacion("cobertura.decidida", "asignacion_unidad", expediente.ActualizadoEn.Add(time.Minute)),
	)
	if err != nil {
		t.Fatal(err)
	}
	asignadaEn := expediente.ActualizadoEn.Add(time.Minute)
	expediente, err = expediente.RegistrarAsignacion(
		expediente.Version,
		AsignacionUnidad{
			UnidadRef:       "unidad:gestora:sintetica:01",
			ResponsableRef:  "persona:responsable:sintetica:01",
			NotificacionRef: "notificacion:asignacion:sintetica:01",
			AsignadaEn:      asignadaEn,
		},
		actuacion("unidad.asignada", "asignacion_unidad", asignadaEn),
	)
	if err != nil {
		t.Fatal(err)
	}
	datosBorrador := datosInformeJuridicoPrueba()
	datosBorrador.ExpedienteRef = expediente.Referencia
	datosBorrador.VersionEsperadaExpediente = expediente.Version
	borrador, err := NuevoBorradorInformeJuridico(datosBorrador)
	if err != nil {
		t.Fatal(err)
	}
	emitidoEn := expediente.ActualizadoEn.Add(time.Minute)
	informe := InformeJuridicoEmitido{
		Borrador:         borrador.Estado(),
		InformeRef:       "informe:juridico:sintetico:01",
		DocumentoRef:     "documento:informe:juridico:sintetico:01",
		VersionDocumento: 1, HuellaDocumentoSHA256: cadena64("f"),
		EmitidoEn: emitidoEn,
	}
	actuacionInforme := DatosActuacion{
		AccionClave: AccionEmitirInformeJuridico,
		ActorRef:    "actor:juridico:sintetico:01",
		UnidadRef:   expediente.Asignacion.UnidadRef,
		ReciboRef:   "recibo:informe:juridico:sintetico:01",
		RealizadaEn: emitidoEn, FaseDestino: FaseInformeJuridico,
		EstadoDestino: EstadoEnCurso,
		DocumentosRef: []string{informe.DocumentoRef},
	}
	expediente, err = expediente.RegistrarInformeJuridico(
		expediente.Version, informe, actuacionInforme,
	)
	if err != nil {
		t.Fatalf("registrar informe: %v", err)
	}
	return expediente
}
