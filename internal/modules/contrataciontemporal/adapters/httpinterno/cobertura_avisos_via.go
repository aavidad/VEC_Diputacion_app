package httpinterno

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

// Proyección JSON de las comprobaciones automáticas de la vía de cobertura.
// Solo transporta recuentos, fechas y la procedencia de cada regla; ninguna
// persona de la bolsa.

type avisosViaCoberturaJSON struct {
	Esquema    string                  `json:"esquema"`
	Estado     string                  `json:"estado"`
	EvaluadaEn string                  `json:"evaluada_en"`
	Avisos     []avisoViaCoberturaJSON `json:"avisos"`
}

type avisoViaCoberturaJSON struct {
	Clave               string                      `json:"clave"`
	Disponibles         *int                        `json:"disponibles,omitempty"`
	Integrantes         *int                        `json:"integrantes,omitempty"`
	Umbral              *int                        `json:"umbral,omitempty"`
	DuracionMaximaMeses int                         `json:"duracion_maxima_meses,omitempty"`
	FinMaximo           string                      `json:"fin_maximo,omitempty"`
	FinPrevisto         string                      `json:"fin_previsto,omitempty"`
	ExcedeDuracion      *bool                       `json:"excede_duracion,omitempty"`
	Motivos             []string                    `json:"motivos,omitempty"`
	ConstituidaEn       string                      `json:"constituida_en,omitempty"`
	VigenciaHasta       string                      `json:"vigencia_hasta,omitempty"`
	Reglas              []procedenciaReglaAvisoJSON `json:"reglas"`
}

type procedenciaReglaAvisoJSON struct {
	Clave        string `json:"clave"`
	Etiqueta     string `json:"etiqueta"`
	Descripcion  string `json:"descripcion"`
	Origen       string `json:"origen"`
	Articulo     string `json:"articulo,omitempty"`
	Norma        string `json:"norma"`
	ParteEjemplo string `json:"parte_ejemplo,omitempty"`
	Referencia   string `json:"referencia"`
	Ejemplo      bool   `json:"ejemplo"`
}

func proyectarAvisosViaCobertura(entrada *application.ResultadoAvisosViaCobertura) *avisosViaCoberturaJSON {
	if entrada == nil {
		return nil
	}
	salida := &avisosViaCoberturaJSON{
		Esquema: "vec.contratacion-temporal.avisos-via-cobertura.v1", Estado: string(entrada.Estado),
		EvaluadaEn: entrada.EvaluadaEn.UTC().Format(time.RFC3339Nano),
		Avisos:     make([]avisoViaCoberturaJSON, 0, len(entrada.Avisos)),
	}
	for _, aviso := range entrada.Avisos {
		proyectado := avisoViaCoberturaJSON{
			Clave: string(aviso.Clave), Motivos: append([]string(nil), aviso.Motivos...),
			ConstituidaEn: aviso.ConstituidaEn, VigenciaHasta: aviso.VigenciaHasta,
			Reglas: make([]procedenciaReglaAvisoJSON, 0, len(aviso.Reglas)),
		}
		switch aviso.Clave {
		case application.AvisoBolsaAgotadaProvisionalmente:
			disponibles, integrantes, umbral := aviso.Disponibles, aviso.Integrantes, aviso.Umbral
			proyectado.Disponibles, proyectado.Integrantes, proyectado.Umbral = &disponibles, &integrantes, &umbral
		case application.AvisoPropuestaOfertaSAE:
			excede := aviso.ExcedeDuracion
			proyectado.DuracionMaximaMeses, proyectado.FinMaximo = aviso.DuracionMaximaMeses, aviso.FinMaximo
			proyectado.FinPrevisto, proyectado.ExcedeDuracion = aviso.FinPrevisto, &excede
		}
		for _, regla := range aviso.Reglas {
			proyectado.Reglas = append(proyectado.Reglas, procedenciaReglaAvisoJSON{
				Clave: regla.Clave, Etiqueta: regla.Etiqueta, Descripcion: regla.Descripcion,
				Origen: regla.Origen, Articulo: regla.Articulo, Norma: regla.Norma,
				ParteEjemplo: regla.ParteEjemplo, Referencia: regla.Referencia, Ejemplo: regla.Ejemplo,
			})
		}
		salida.Avisos = append(salida.Avisos, proyectado)
	}
	return salida
}
