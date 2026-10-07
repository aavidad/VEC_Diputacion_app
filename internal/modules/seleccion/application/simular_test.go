package application

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

type contadorBaremador struct{ llamadas int }

func (b *contadorBaremador) Valorar(string, int, string) (domain.MeritosValorados, error) {
	b.llamadas++
	return domain.MeritosValorados{}, nil
}

func TestEntradaInvalidaNoInvocaBaremador(t *testing.T) {
	minimo := int64(0)
	config := domain.Configuracion{Version: 1, Modalidad: "concurso", TurnoAcceso: "libre", Destino: "plaza", Plazas: 1,
		Fases: []domain.Fase{{Referencia: "meritos", Tipo: "meritos", MinimoMicropuntos: &minimo, MaximoMicropuntos: 10_000_000, Peso: 100}}}
	for _, entrada := range []domain.Entrada{
		{},
		{ConvocatoriaRef: "convocatoria_sintetica", BasesVersion: 1, Solicitudes: []domain.Solicitud{{Referencia: "a"}, {Referencia: "a"}}},
		{ConvocatoriaRef: "convocatoria_sintetica", BasesVersion: 1, Solicitudes: make([]domain.Solicitud, 129)},
	} {
		b := &contadorBaremador{}
		if _, err := Simular(config, entrada, b); !errors.Is(err, domain.ErrConfiguracion) || b.llamadas != 0 {
			t.Fatalf("entrada inválida invocó %d veces el baremador: %v", b.llamadas, err)
		}
	}
}
