package simulacion

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/application"
)

func TestTresModalidadesUsanBaremadorDeBolsa(t *testing.T) {
	ejemplos, err := Ejemplos()
	if err != nil {
		t.Fatal(err)
	}
	modalidades := map[string]bool{}
	for _, ejemplo := range ejemplos {
		modalidades[ejemplo.Configuracion.Modalidad] = true
		entrada, baremador, err := Preparar(ejemplo.Referencia)
		if err != nil {
			t.Fatalf("%s: %v (%v)", ejemplo.Referencia, err, errors.Unwrap(err))
		}
		resultado, err := application.Simular(ejemplo.Configuracion, entrada, baremador)
		if err != nil {
			t.Fatalf("%s: %v (%v)", ejemplo.Referencia, err, errors.Unwrap(err))
		}
		if resultado.Alcance != "ensayo_sintetico" || resultado.ConvocatoriaRef != ejemplo.ConvocatoriaRef || resultado.BasesVersion != ejemplo.BasesVersion || resultado.Plazas != ejemplo.Configuracion.Plazas || len(resultado.Solicitudes) != len(entrada.Solicitudes) {
			t.Fatalf("%s: respuesta sin alcance o referencias de la instantánea: %+v", ejemplo.Referencia, resultado)
		}
		for _, solicitud := range resultado.Solicitudes {
			for _, fase := range solicitud.Fases {
				if fase.Tipo == "meritos" && (fase.Origen != "motor_bolsa" || fase.HuellaMeritosSHA256 == "") {
					t.Fatalf("%s/%s: méritos sin traza de Bolsa: %+v", ejemplo.Referencia, solicitud.Referencia, fase)
				}
			}
		}
	}
	if len(ejemplos) != 3 || !modalidades["oposicion"] || !modalidades["concurso"] || !modalidades["concurso_oposicion"] {
		t.Fatalf("ejemplos retirables inesperados: %v", modalidades)
	}
}

func TestSolicitudSoloAceptaConfiguracionYEjemplo(t *testing.T) {
	casos := []string{
		`{"ejemplo_ref":"seleccion_oposicion_v1","configuracion":{},"solicitudes":[{"nombre":"Persona añadida"}]}`,
		`{"ejemplo_ref":"seleccion_oposicion_v1","configuracion":{},"entrada":{"notas":{"ejercicio_1":10000000}}}`,
		`{"ejemplo_ref":"seleccion_oposicion_v1","configuracion":{},"Ejemplo_ref":"seleccion_concurso_v1"}`,
		`{"ejemplo_ref":"seleccion_oposicion_v1","ejemplo_ref":"seleccion_concurso_v1","configuracion":{}}`,
	}
	for _, datos := range casos {
		if _, err := Decodificar([]byte(datos)); err == nil {
			t.Fatalf("se aceptaron hechos o claves ambiguas del cliente: %s", datos)
		}
	}
	if _, err := Decodificar([]byte(`{"ejemplo_ref":"seleccion_oposicion_v1","configuracion":{"version":1}}`)); err != nil {
		t.Fatalf("se rechazó la forma permitida: %v", err)
	}
}
