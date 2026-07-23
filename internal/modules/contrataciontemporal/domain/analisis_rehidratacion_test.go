package domain

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestExpedienteRehidratadoAdmiteAnalisisMaterializadosValidos(t *testing.T) {
	casos := []struct {
		nombre    string
		construir func(*testing.T) Expediente
	}{
		{"sin análisis", func(t *testing.T) Expediente {
			return expedienteValido(t)
		}},
		{"RC validada", func(t *testing.T) Expediente {
			return expedienteConAnalisisRehidratado(t, RCValidada)
		}},
		{"RC no requerida", func(t *testing.T) Expediente {
			return expedienteConAnalisisRehidratado(t, RCNoRequerida)
		}},
		{"RC rechazada sin vía", func(t *testing.T) Expediente {
			return expedienteConAnalisisRehidratado(t, RCRechazada)
		}},
		{"vía con RC validada", func(t *testing.T) Expediente {
			return expedienteConViaRehidratado(t, RCValidada)
		}},
		{"vía con RC no requerida", func(t *testing.T) Expediente {
			return expedienteConViaRehidratado(t, RCNoRequerida)
		}},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			expediente := caso.construir(t)
			if err := expediente.Validar(); err != nil {
				t.Fatalf("expediente válido rechazado: %v", err)
			}
		})
	}
}

func TestExpedienteRehidratadoRechazaViaConRCRechazada(t *testing.T) {
	expediente := expedienteConViaRehidratado(t, RCValidada)
	prepararRCNegativa(&expediente.Analisis.ValidacionRC, RCRechazada)
	if !errors.Is(expediente.Validar(), ErrExpedienteInvalido) {
		t.Fatal("se aceptó una vía de cobertura sostenida por una RC rechazada")
	}
}

func TestExpedienteRehidratadoRechazaAnalisisSinActuacionMaterial(t *testing.T) {
	expediente := expedienteValido(t)
	analisis := analisisValido()
	expediente.Analisis = &analisis
	if !errors.Is(expediente.Validar(), ErrExpedienteInvalido) {
		t.Fatal("se aceptó un análisis inyectado en la versión inicial")
	}

	vinculo := nuevoVinculoActuacionAnalisis(
		2,
		2,
		actuacion("analisis.validado", "gestion_bolsa", instanteBase.Add(time.Minute)),
	)
	analisis.ActuacionRegistro = &vinculo
	expediente.Analisis = &analisis
	if !errors.Is(expediente.Validar(), ErrExpedienteInvalido) {
		t.Fatal("se aceptó un vínculo a una actuación inexistente")
	}
}

func TestExpedienteRehidratadoRechazaValidacionPosteriorAActuacion(t *testing.T) {
	expediente := expedienteConAnalisisRehidratado(t, RCValidada)
	vinculo := expediente.Analisis.ActuacionRegistro
	actuacionMaterial := expediente.Actuaciones[vinculo.Secuencia-1]
	expediente.Analisis.ValidacionRC.ValidadaEn =
		actuacionMaterial.RealizadaEn.Add(time.Microsecond)
	if !errors.Is(expediente.Validar(), ErrExpedienteInvalido) {
		t.Fatal("se aceptó una validación RC posterior a su actuación material")
	}
}

func TestExpedienteRehidratadoCotejaVersionSecuenciaAccionYFase(t *testing.T) {
	pruebas := []struct {
		nombre    string
		adulterar func(*Expediente)
	}{
		{"versión del vínculo", func(e *Expediente) {
			e.Analisis.ActuacionRegistro.VersionExpediente++
		}},
		{"secuencia del vínculo", func(e *Expediente) {
			e.Analisis.ActuacionRegistro.Secuencia++
		}},
		{"acción del vínculo", func(e *Expediente) {
			e.Analisis.ActuacionRegistro.AccionClave = "analisis.otra_accion"
		}},
		{"fase del vínculo", func(e *Expediente) {
			e.Analisis.ActuacionRegistro.FaseDestino = "otra_fase"
		}},
		{"recibo del vínculo", func(e *Expediente) {
			e.Analisis.ActuacionRegistro.ReciboRef = "recibo_distinto_01"
		}},
		{"versión de la actuación", func(e *Expediente) {
			e.Actuaciones[1].VersionExpediente++
		}},
		{"secuencia de la actuación", func(e *Expediente) {
			e.Actuaciones[1].Secuencia++
		}},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			expediente := expedienteConAnalisisRehidratado(t, RCValidada)
			prueba.adulterar(&expediente)
			if !errors.Is(expediente.Validar(), ErrExpedienteInvalido) {
				t.Fatal("el expediente aceptó un vínculo o actuación adulterados")
			}
		})
	}
}

func TestExpedienteClonaDefensivamenteVinculoDeAnalisis(t *testing.T) {
	original := expedienteConAnalisisRehidratado(t, RCValidada)
	clon := original.Clonar()
	clon.Analisis.ActuacionRegistro.VersionExpediente++
	if clon.Analisis.ActuacionRegistro == original.Analisis.ActuacionRegistro ||
		clon.Analisis.ActuacionRegistro.VersionExpediente ==
			original.Analisis.ActuacionRegistro.VersionExpediente {
		t.Fatal("el clon comparte el vínculo mutable de la actuación de análisis")
	}
}

func TestExpedienteRehidratadoDesdeJSONConservaVinculoMaterial(t *testing.T) {
	original := expedienteConViaRehidratado(t, RCValidada)
	contenido, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("serializar expediente: %v", err)
	}
	var rehidratado Expediente
	if err := json.Unmarshal(contenido, &rehidratado); err != nil {
		t.Fatalf("rehidratar expediente: %v", err)
	}
	if err := rehidratado.Validar(); err != nil {
		t.Fatalf("expediente rehidratado válido: %v", err)
	}

	comando, err := json.Marshal(analisisValido())
	if err != nil {
		t.Fatalf("serializar comando de análisis: %v", err)
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(comando, &campos); err != nil {
		t.Fatalf("decodificar comando de análisis: %v", err)
	}
	if _, existe := campos["actuacion_registro"]; existe {
		t.Fatalf("el comando no materializado emitió un vínculo de actuación: %s", comando)
	}
}

func expedienteConAnalisisRehidratado(
	t *testing.T,
	resultado ResultadoValidacionRC,
) Expediente {
	t.Helper()
	analisis := analisisValido()
	if resultado != RCValidada {
		prepararRCNegativa(&analisis.ValidacionRC, resultado)
	}
	expediente, err := expedienteValido(t).RegistrarAnalisis(
		1,
		analisis,
		actuacion("analisis.validado", "gestion_bolsa", instanteBase.Add(time.Minute)),
	)
	if err != nil {
		t.Fatalf("registrar análisis %s: %v", resultado, err)
	}
	return expediente
}

func expedienteConViaRehidratado(
	t *testing.T,
	resultado ResultadoValidacionRC,
) Expediente {
	t.Helper()
	expediente := expedienteConAnalisisRehidratado(t, resultado)
	conVia, err := expediente.RegistrarViaCobertura(
		2,
		decisionValida(),
		actuacion("cobertura.decidida", "asignacion_unidad", instanteBase.Add(2*time.Minute)),
	)
	if err != nil {
		t.Fatalf("registrar vía con %s: %v", resultado, err)
	}
	return conVia
}
