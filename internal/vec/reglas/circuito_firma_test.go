package reglas

import (
	"errors"
	"testing"
)

const rutaCircuitoFirmaCTPrueba = "../../../data/demo/reglas/ct_circuito_firma.ejemplo.demo.json"

func TestCircuitoFirmaDeEjemploCubreLosSeisBorradores(t *testing.T) {
	resolutor := resolutorReal(t, rutaCircuitoFirmaCTPrueba, CatalogoCircuitoFirmaCT, ModuloContratacionTemporal, nil)
	circuito, err := resolutor.CircuitoFirma(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	esperados := []string{"informe_definitivo", "resolucion", "diligencia", "toma_posesion", "notificacion", "comunicacion_centro"}
	if len(circuito.Documentos) != len(esperados) || !circuito.PaqueteEjemplo ||
		circuito.CatalogoID != CatalogoCircuitoFirmaCT || circuito.Version != 1 || len(circuito.HuellaCatalogo) != 64 {
		t.Fatalf("circuito inesperado: %+v", circuito)
	}
	for indice, documento := range circuito.Documentos {
		if documento.Documento != esperados[indice] || len(documento.Pasos) == 0 || documento.Etiqueta == "" {
			t.Fatalf("documento %d inesperado: %+v", indice, documento)
		}
		for orden, paso := range documento.Pasos {
			if paso.Orden != orden+1 || paso.Cargo == "" || paso.Referencia == "" {
				t.Fatalf("paso inesperado en %s: %+v", documento.Documento, paso)
			}
		}
	}
	informe := circuito.Documentos[0].Pasos
	if informe[len(informe)-1].Habilita != HabilitaRemisionIntervencion {
		t.Fatalf("en el ejemplo, la última firma del informe habilita la remisión a Intervención: %+v", informe)
	}
	if circuito.Documentos[1].Pasos[0].Accion != AccionFirmaVistoBueno {
		t.Fatalf("la propuesta de resolución es un visto bueno: %+v", circuito.Documentos[1].Pasos[0])
	}
}

func TestCircuitoFirmaRechazaCircuitosIncoherentes(t *testing.T) {
	resolutor := resolutorReal(t, rutaCircuitoFirmaCTPrueba, CatalogoCircuitoFirmaCT, ModuloContratacionTemporal, nil)
	vigentes, err := resolutor.Reglas(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	alterar := func(indice int, atributo, valor string) []Regla {
		copia := append([]Regla(nil), vigentes...)
		atributos := make(map[string]string, len(copia[indice].Atributos))
		for clave, actual := range copia[indice].Atributos {
			atributos[clave] = actual
		}
		atributos[atributo] = valor
		copia[indice].Atributos = atributos
		return copia
	}
	casos := map[string][]Regla{
		"sin reglas":                    nil,
		"paso con hueco":                alterar(1, "paso", "3"),
		"paso no canónico":              alterar(0, "paso", "01"),
		"accion desconocida":            alterar(0, "accion", "rubrica"),
		"condición del primer paso":     alterar(0, "condicion", "firma_paso_anterior"),
		"habilitación desconocida":      alterar(0, "habilita", "publicar"),
		"último paso sin cierre":        alterar(1, "habilita", "siguiente_paso"),
		"paso intermedio que cierra":    alterar(0, "habilita", "cierre_circuito"),
		"devolución al paso anterior 1": alterar(0, "devolucion", "vuelve_paso_anterior"),
		"sustitución desconocida":       alterar(0, "sustitucion", "cualquiera"),
		"perfil no opaco":               alterar(0, "perfil_ref", "Jefatura RRHH"),
		"sin cargo":                     alterar(0, "cargo", ""),
		"etiqueta cambiada":             alterar(1, "documento_etiqueta", "Otro"),
	}
	for nombre, reglas := range casos {
		if _, err := CircuitoFirmaDesdeReglas(reglas); !errors.Is(err, ErrCircuitoFirmaInvalido) {
			t.Errorf("%s: debe rechazarse, err=%v", nombre, err)
		}
	}
}

func TestEstadoPasoSinFirmas(t *testing.T) {
	if EstadoPasoSinFirmas(1) != EstadoPasoPendienteFirma || EstadoPasoSinFirmas(2) != EstadoPasoEnEspera {
		t.Fatal("sin firmas, solo el primer paso está pendiente")
	}
}
