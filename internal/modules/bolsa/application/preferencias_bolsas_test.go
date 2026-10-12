package application

import (
	"errors"
	"reflect"
	"testing"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestPreferenciaEntreBolsasNoDependeDelRangoDeCadaBolsa(t *testing.T) {
	actuales := []string{"bolsa:a", "bolsa:b", "bolsa:c"}
	if err := ValidarOrdenPreferencias(actuales, []string{"bolsa:c", "bolsa:a", "bolsa:b"}); err != nil {
		t.Fatal(err)
	}
	estado, err := ProyectarEstadoPreferencias(4, []string{"bolsa:c", "bolsa:a", "bolsa:b"}, actuales)
	if err != nil || estado.RevisionRequerida {
		t.Fatalf("estado inesperado: %+v %v", estado, err)
	}
	// El turno propio de a y b puede ser primero y el de c segundo: esta
	// proyeccion no recibe ni modifica tales posiciones.
	recomendacion, err := RecomendarSeleccionSimultanea(estado, []string{"bolsa:a", "bolsa:c"})
	if err != nil || recomendacion.BolsaPreferida != "bolsa:c" || recomendacion.MotivoCodigo != "primera_preferencia_elegible" || recomendacion.Version != 4 {
		t.Fatalf("recomendacion inesperada: %+v %v", recomendacion, err)
	}
}

func TestCambioDeParticipacionesExigeRevisionSinPerderOrdenAnterior(t *testing.T) {
	estado, err := ProyectarEstadoPreferencias(7, []string{"bolsa:b", "bolsa:a"}, []string{"bolsa:b", "bolsa:c"})
	if err != nil || !estado.RevisionRequerida || !reflect.DeepEqual(estado.BolsasNuevas, []string{"bolsa:c"}) ||
		!reflect.DeepEqual(estado.BolsasRetiradas, []string{"bolsa:a"}) ||
		!reflect.DeepEqual(estado.OrdenGuardado, []string{"bolsa:b", "bolsa:a"}) {
		t.Fatalf("cambio mal proyectado: %+v %v", estado, err)
	}
	recomendacion, err := RecomendarSeleccionSimultanea(estado, []string{"bolsa:b", "bolsa:c"})
	if err != nil || recomendacion.BolsaPreferida != "" || recomendacion.MotivoCodigo != "revision_requerida" {
		t.Fatalf("la seleccion no debe decidir con orden incompleto: %+v %v", recomendacion, err)
	}
	if err := ValidarOrdenPreferencias(estado.BolsasActuales, []string{"bolsa:b", "bolsa:a"}); !errors.Is(err, bolsa.ErrOrdenPreferenciasInvalido) {
		t.Fatalf("se acepto bolsa retirada: %v", err)
	}
}

func TestOrdenPreferenciasRechazaDuplicadosYBolsaAjena(t *testing.T) {
	actuales := []string{"bolsa:a", "bolsa:b"}
	for _, orden := range [][]string{{"bolsa:a", "bolsa:a"}, {"bolsa:a", "bolsa:ajena"}, {"bolsa:b"}, nil} {
		if err := ValidarOrdenPreferencias(actuales, orden); !errors.Is(err, bolsa.ErrOrdenPreferenciasInvalido) {
			t.Fatalf("orden %q aceptado: %v", orden, err)
		}
	}
	estado, err := ProyectarEstadoPreferencias(1, []string{"bolsa:a", "bolsa:b"}, actuales)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RecomendarSeleccionSimultanea(estado, []string{"bolsa:ajena"}); !errors.Is(err, bolsa.ErrOrdenPreferenciasInvalido) {
		t.Fatalf("se acepto elegibilidad ajena: %v", err)
	}
}
