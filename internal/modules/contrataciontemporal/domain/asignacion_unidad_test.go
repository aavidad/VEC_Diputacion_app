package domain

import (
	"errors"
	"testing"
	"time"
)

func TestExpedienteReasignaUnidadSinReescribirHistoria(t *testing.T) {
	anterior := expedienteConAsignacion(t)
	instantaneaAnterior := anterior.Clonar()
	instante := anterior.ActualizadoEn.Add(time.Minute)
	nueva := AsignacionUnidad{
		UnidadRef:       "unidad:contratacion-temporal-002",
		ResponsableRef:  "persona:responsable-sintetica-002",
		NotificacionRef: "notificacion:asignacion-sintetica-002",
		AsignadaEn:      instante,
		Observaciones:   "Reasignación motivada por el catálogo vigente.",
	}
	actuacion := actuacion(
		"unidad.reasignada",
		string(anterior.FaseActual),
		instante,
	)
	actuacion.EstadoDestino = anterior.EstadoActual
	actuacion.Observaciones = nueva.Observaciones

	siguiente, err := anterior.ReasignarUnidad(
		anterior.Version,
		nueva,
		actuacion,
	)
	if err != nil {
		t.Fatalf("reasignar unidad: %v", err)
	}
	if siguiente.Version != anterior.Version+1 ||
		len(siguiente.Actuaciones) != len(anterior.Actuaciones)+1 ||
		siguiente.Asignacion == nil ||
		siguiente.Asignacion.UnidadRef != nueva.UnidadRef ||
		siguiente.Asignacion.ResponsableRef != nueva.ResponsableRef ||
		siguiente.Asignacion.NotificacionRef != nueva.NotificacionRef ||
		siguiente.Asignacion.ActuacionRegistro == nil {
		t.Fatalf("resultado de reasignación inesperado: %#v", siguiente)
	}
	if anterior.Validar() != nil ||
		anterior.Version != instantaneaAnterior.Version ||
		len(anterior.Actuaciones) != len(instantaneaAnterior.Actuaciones) ||
		anterior.Asignacion == nil ||
		anterior.Asignacion.UnidadRef != instantaneaAnterior.Asignacion.UnidadRef ||
		anterior.Asignacion.ResponsableRef !=
			instantaneaAnterior.Asignacion.ResponsableRef {
		t.Fatal("la transición modificó la instantánea anterior")
	}
	ultima := siguiente.Actuaciones[len(siguiente.Actuaciones)-1]
	if ultima.FaseOrigen != anterior.FaseActual ||
		ultima.FaseDestino != anterior.FaseActual ||
		ultima.EstadoOrigen != anterior.EstadoActual ||
		ultima.EstadoDestino != anterior.EstadoActual ||
		ultima.Observaciones != nueva.Observaciones {
		t.Fatalf("actuación de reasignación incompleta: %#v", ultima)
	}
}

func TestExpedienteRehidratadoRechazaAsignacionLigadaAOtraActuacion(t *testing.T) {
	expediente := expedienteConAsignacion(t)
	adulterado := expediente.Clonar()
	adulterado.Asignacion.ActuacionRegistro.ReciboRef =
		"recibo:asignacion-adulterado"
	if !errors.Is(adulterado.Validar(), ErrExpedienteInvalido) {
		t.Fatal("se aceptó una asignación ligada a otro recibo")
	}

	adulterado = expediente.Clonar()
	adulterado.Asignacion.AsignadaEn =
		adulterado.Asignacion.AsignadaEn.Add(time.Microsecond)
	if !errors.Is(adulterado.Validar(), ErrExpedienteInvalido) {
		t.Fatal("se aceptó una asignación ligada a otro instante")
	}
}

func TestExpedienteReasignacionExigeCambioMotivoEInstanteNuevo(t *testing.T) {
	base := expedienteConAsignacion(t)
	casos := []struct {
		nombre     string
		asignacion AsignacionUnidad
		actuacion  DatosActuacion
	}{
		{
			nombre: "sin cambio de destino",
			asignacion: AsignacionUnidad{
				UnidadRef:       base.Asignacion.UnidadRef,
				ResponsableRef:  base.Asignacion.ResponsableRef,
				NotificacionRef: "notificacion:asignacion-sintetica-003",
				AsignadaEn:      base.ActualizadoEn.Add(time.Minute),
				Observaciones:   "Motivo suficiente.",
			},
		},
		{
			nombre: "sin motivo",
			asignacion: AsignacionUnidad{
				UnidadRef:       "unidad:contratacion-temporal-003",
				ResponsableRef:  "persona:responsable-sintetica-003",
				NotificacionRef: "notificacion:asignacion-sintetica-003",
				AsignadaEn:      base.ActualizadoEn.Add(time.Minute),
			},
		},
		{
			nombre: "instante no posterior",
			asignacion: AsignacionUnidad{
				UnidadRef:       "unidad:contratacion-temporal-003",
				ResponsableRef:  "persona:responsable-sintetica-003",
				NotificacionRef: "notificacion:asignacion-sintetica-003",
				AsignadaEn:      base.Asignacion.AsignadaEn,
				Observaciones:   "Motivo suficiente.",
			},
		},
		{
			nombre: "notificacion reutilizada",
			asignacion: AsignacionUnidad{
				UnidadRef:       "unidad:contratacion-temporal-003",
				ResponsableRef:  "persona:responsable-sintetica-003",
				NotificacionRef: base.Asignacion.NotificacionRef,
				AsignadaEn:      base.ActualizadoEn.Add(time.Minute),
				Observaciones:   "Motivo suficiente.",
			},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			caso.actuacion = actuacion(
				"unidad.reasignada",
				string(base.FaseActual),
				caso.asignacion.AsignadaEn,
			)
			caso.actuacion.EstadoDestino = base.EstadoActual
			caso.actuacion.Observaciones = caso.asignacion.Observaciones
			if _, err := base.ReasignarUnidad(
				base.Version,
				caso.asignacion,
				caso.actuacion,
			); !errors.Is(err, ErrTransicionInvalida) {
				t.Fatalf("error = %v; se esperaba transición inválida", err)
			}
		})
	}
}

func TestExpedienteReasignacionRechazaConflictoYCambioImplicitoDeFase(t *testing.T) {
	base := expedienteConAsignacion(t)
	instante := base.ActualizadoEn.Add(time.Minute)
	asignacion := AsignacionUnidad{
		UnidadRef:       "unidad:contratacion-temporal-004",
		ResponsableRef:  "persona:responsable-sintetica-004",
		NotificacionRef: "notificacion:asignacion-sintetica-004",
		AsignadaEn:      instante,
		Observaciones:   "Reasignación motivada.",
	}
	actuacionValida := actuacion(
		"unidad.reasignada",
		string(base.FaseActual),
		instante,
	)
	actuacionValida.EstadoDestino = base.EstadoActual
	actuacionValida.Observaciones = asignacion.Observaciones

	if _, err := base.ReasignarUnidad(
		base.Version+1,
		asignacion,
		actuacionValida,
	); !errors.Is(err, ErrVersionEnConflicto) {
		t.Fatalf("conflicto de versión: %v", err)
	}
	conSalto := actuacionValida
	conSalto.FaseDestino = "informe_juridico"
	if _, err := base.ReasignarUnidad(
		base.Version,
		asignacion,
		conSalto,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("salto implícito de fase: %v", err)
	}
}

func expedienteConAsignacion(t *testing.T) Expediente {
	t.Helper()
	expediente := expedienteValido(t)
	var err error
	expediente, err = expediente.RegistrarAnalisis(
		expediente.Version,
		analisisValido(),
		actuacion(
			"analisis.validado",
			"gestion_bolsa",
			expediente.ActualizadoEn.Add(time.Minute),
		),
	)
	if err != nil {
		t.Fatalf("registrar análisis: %v", err)
	}
	expediente, err = expediente.RegistrarViaCobertura(
		expediente.Version,
		decisionValida(),
		actuacion(
			"cobertura.decidida",
			"asignacion_unidad",
			expediente.ActualizadoEn.Add(time.Minute),
		),
	)
	if err != nil {
		t.Fatalf("registrar vía: %v", err)
	}
	asignacion := AsignacionUnidad{
		UnidadRef:       "unidad:contratacion-temporal-001",
		ResponsableRef:  "persona:responsable-sintetica-001",
		NotificacionRef: "notificacion:asignacion-sintetica-001",
		AsignadaEn:      expediente.ActualizadoEn.Add(time.Minute),
	}
	expediente, err = expediente.RegistrarAsignacion(
		expediente.Version,
		asignacion,
		actuacion(
			"unidad.asignada",
			"unidad_gestora",
			asignacion.AsignadaEn,
		),
	)
	if err != nil {
		t.Fatalf("registrar asignación: %v", err)
	}
	return expediente
}
