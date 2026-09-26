package bootstrap

import "testing"

func TestCentrosAnterioresCTDelPaqueteDeEjemplo(t *testing.T) {
	centros, err := cargarCentrosAnterioresCT()
	if err != nil || len(centros) != 1 || centros[0].Referencia != centroAltaContratacionTemporalDesarrollo ||
		centros[0].Etiqueta != `RESIDENCIA DE MAYORES "SIERRA NEVADA"` || len(centros[0].Contactos) != 1 {
		t.Fatalf("paquete de centros anteriores inesperado: %+v, %v", centros, err)
	}
}

func TestCentrosAnterioresCTRechazaFicherosNoValidos(t *testing.T) {
	for nombre, contenido := range map[string]string{
		"campo desconocido": `{"version":1,"motivo":"m","centros":[],"extra":1}`,
		"versión":           `{"version":2,"motivo":"m","centros":[]}`,
		"sin motivo":        `{"version":1,"motivo":" ","centros":[]}`,
		"referencia":        `{"version":1,"motivo":"m","centros":[{"referencia":"centro-520","etiqueta":"X"}]}`,
		"duplicado":         `{"version":1,"motivo":"m","centros":[{"referencia":"centro:a:1","etiqueta":"X"},{"referencia":"centro:a:1","etiqueta":"Y"}]}`,
		"etiqueta vacía":    `{"version":1,"motivo":"m","centros":[{"referencia":"centro:a:1","etiqueta":""}]}`,
		"espacios":          `{"version":1,"motivo":"m","centros":[{"referencia":"centro:a:1","etiqueta":" X"}]}`,
		"dos documentos":    `{"version":1,"motivo":"m","centros":[]}{}`,
	} {
		if _, err := centrosAnterioresCTDesdeJSON([]byte(contenido)); err == nil {
			t.Fatalf("%s: se esperaba rechazo", nombre)
		}
	}
	centros, err := centrosAnterioresCTDesdeJSON([]byte(`{"version":1,"motivo":"m","centros":[]}`))
	if err != nil || len(centros) != 0 {
		t.Fatalf("un paquete vacío es válido: %v %v", centros, err)
	}
}
