package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

func TestCapacidadExternaAcreditaSoloEsquemaYGrupoNominales(t *testing.T) {
	fila := filaDoble{valores: []any{
		"login_externo", "login_externo", true, true, true, true, true, true, true,
	}}
	consultor := &consultorCapacidadPrueba{fila: fila}
	usuario, err := acreditarCapacidadExternaConsultor(
		context.Background(), consultor, capacidadRegistrarExterna,
	)
	firmas, _ := firmasCapacidadExterna(capacidadRegistrarExterna)
	if err != nil || usuario != "login_externo" || len(consultor.argumentos) != 2 ||
		consultor.argumentos[0] != capacidadRegistrarExterna ||
		!strings.Contains(consultor.consulta, esquemaAcreditacionExterno) ||
		strings.Contains(consultor.consulta, esquemaAcreditacionInterno) ||
		len(firmas) != 2 || !strings.Contains(firmas[0], "vec_identidad_externa_v1.") ||
		!strings.Contains(firmas[1], "vec_identidad_externa_v1.") {
		t.Fatal("la acreditacion externa cambio esquema o funciones")
	}
	if _, err = acreditarCapacidadExternaConsultor(
		context.Background(), consultor, capacidadRegistrar,
	); !errors.Is(err, httpseguridad.ErrRegistroSesionesAusente) {
		t.Fatal("se acepto grupo registrador interno")
	}
}

func TestCapacidadExternaDeniegaMembresiaOACLAmpliada(t *testing.T) {
	for indice := 2; indice < 9; indice++ {
		valores := []any{
			"login_externo", "login_externo", true, true, true, true, true, true, true,
		}
		valores[indice] = false
		consultor := &consultorCapacidadPrueba{fila: filaDoble{valores: valores}}
		if _, err := acreditarCapacidadExternaConsultor(
			context.Background(), consultor, capacidadRevalidarExterna,
		); !errors.Is(err, httpseguridad.ErrRegistroSesionesAusente) {
			t.Fatalf("comprobacion %d ausente", indice)
		}
	}
}
