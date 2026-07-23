package domain

import (
	"errors"
	"testing"
	"time"
)

func TestRectificarAnalisisConservaCronologiaYActualizaProyeccion(t *testing.T) {
	original := expedienteConAnalisisRehidratado(t, RCValidada)
	vinculoAnterior := *original.Analisis.ActuacionRegistro
	analisis := analisisValido()
	analisis.Observaciones = "Se corrige la jornada tras cotejar el informe del centro."
	analisis.PorcentajeJornada = 7_500
	actuacionRectificacion := actuacion(
		"analisis.rectificado",
		string(original.FaseActual),
		instanteBase.Add(2*time.Minute),
	)
	actuacionRectificacion.Observaciones = "Rectificación motivada por el informe horario recibido."

	rectificado, err := original.RectificarAnalisis(
		original.Version,
		analisis,
		actuacionRectificacion,
	)
	if err != nil {
		t.Fatalf("rectificar análisis: %v", err)
	}
	if rectificado.Version != 3 || len(rectificado.Actuaciones) != 3 ||
		rectificado.Analisis.PorcentajeJornada != 7_500 ||
		rectificado.Analisis.ActuacionRegistro.Secuencia != 3 {
		t.Fatalf("rectificación inesperada: %#v", rectificado)
	}
	if original.Version != 2 || len(original.Actuaciones) != 2 ||
		original.Analisis.PorcentajeJornada != JornadaCompletaDiezmilesimas ||
		*original.Analisis.ActuacionRegistro != vinculoAnterior {
		t.Fatal("la rectificación mutó el expediente o el análisis anterior")
	}
	if err := rectificado.Validar(); err != nil {
		t.Fatalf("expediente rectificado inválido: %v", err)
	}
}

func TestRectificarAnalisisExigeVersionYMotivacion(t *testing.T) {
	expediente := expedienteConAnalisisRehidratado(t, RCValidada)
	analisis := analisisValido()
	analisis.PorcentajeJornada = 5_000
	base := actuacion(
		"analisis.rectificado",
		string(expediente.FaseActual),
		instanteBase.Add(2*time.Minute),
	)
	base.Observaciones = "Corrección sustentada en documentación sobrevenida."

	if _, err := expediente.RectificarAnalisis(
		expediente.Version+1,
		analisis,
		base,
	); !errors.Is(err, ErrVersionEnConflicto) {
		t.Fatalf("no detectó el conflicto optimista: %v", err)
	}

	sinMotivo := base
	sinMotivo.Observaciones = ""
	if _, err := expediente.RectificarAnalisis(
		expediente.Version,
		analisis,
		sinMotivo,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("aceptó una rectificación sin motivación: %v", err)
	}
}

func TestRectificarAnalisisNoAceptaAutoridadMaterialDelComando(t *testing.T) {
	expediente := expedienteConAnalisisRehidratado(t, RCValidada)
	analisis := analisisValido()
	actuacionRectificacion := actuacion(
		"analisis.rectificado",
		string(expediente.FaseActual),
		instanteBase.Add(2*time.Minute),
	)
	actuacionRectificacion.Observaciones = "Rectificación motivada y documentada."
	vinculoFabricado := nuevoVinculoActuacionAnalisis(
		expediente.Version+1,
		expediente.Version+1,
		actuacionRectificacion,
	)
	analisis.ActuacionRegistro = &vinculoFabricado

	if _, err := expediente.RectificarAnalisis(
		expediente.Version,
		analisis,
		actuacionRectificacion,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("aceptó un vínculo material aportado por el comando: %v", err)
	}
}

func TestRectificarAnalisisNoSaltaFaseEstadoNiDecisionesPosteriores(t *testing.T) {
	expediente := expedienteConAnalisisRehidratado(t, RCValidada)
	analisis := analisisValido()
	base := actuacion(
		"analisis.rectificado",
		string(expediente.FaseActual),
		instanteBase.Add(2*time.Minute),
	)
	base.Observaciones = "Rectificación motivada por nueva evidencia."

	saltaFase := base
	saltaFase.FaseDestino = "otra_fase"
	if _, err := expediente.RectificarAnalisis(
		expediente.Version,
		analisis,
		saltaFase,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("la rectificación saltó de fase: %v", err)
	}

	cambiaEstado := base
	cambiaEstado.EstadoDestino = EstadoIncidencia
	if _, err := expediente.RectificarAnalisis(
		expediente.Version,
		analisis,
		cambiaEstado,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("la rectificación cambió el estado: %v", err)
	}

	conVia := expedienteConViaRehidratado(t, RCValidada)
	base.RealizadaEn = instanteBase.Add(3 * time.Minute)
	base.FaseDestino = conVia.FaseActual
	if _, err := conVia.RectificarAnalisis(
		conVia.Version,
		analisis,
		base,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("rectificó bajo una decisión de cobertura vigente: %v", err)
	}
}

func TestRectificarAnalisisOrdenaEvidenciaAntesDeActuacion(t *testing.T) {
	expediente := expedienteConAnalisisRehidratado(t, RCValidada)
	analisis := analisisValido()
	instanteActuacion := instanteBase.Add(2 * time.Minute)
	analisis.ValidacionRC.ValidadaEn = instanteActuacion.Add(time.Microsecond)
	actuacionRectificacion := actuacion(
		"analisis.rectificado",
		string(expediente.FaseActual),
		instanteActuacion,
	)
	actuacionRectificacion.Observaciones = "Rectificación motivada."

	if _, err := expediente.RectificarAnalisis(
		expediente.Version,
		analisis,
		actuacionRectificacion,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("aceptó evidencia posterior a la actuación: %v", err)
	}
}
