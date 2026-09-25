package config

import (
	"errors"
	"testing"
	"time"
)

func TestEntregaContratosCTBolsaPorDefectoYConfigurable(t *testing.T) {
	r, err := NuevaConfiguracionEntregaContratosCTBolsa("", "", "").Resolver()
	if err != nil || !r.Activa || r.Intervalo != DefaultBolsaContratosCTIntervalo || r.Lote != DefaultBolsaContratosCTLote || r.Relectura != DefaultBolsaContratosCTRelectura {
		t.Fatalf("defecto=%+v err=%v", r, err)
	}
	r, err = NuevaConfiguracionEntregaContratosCTBolsa("5s", "10", "0s").Resolver()
	if err != nil || r.Intervalo != 5*time.Second || r.Lote != 10 || r.Relectura != 0 {
		t.Fatalf("fijado=%+v err=%v", r, err)
	}
	r, err = NuevaConfiguracionEntregaContratosCTBolsa("0", "x", "x").Resolver()
	if err != nil || r.Activa {
		t.Fatalf("desactivado=%+v err=%v", r, err)
	}
}

func TestEntregaContratosCTBolsaRechazaValoresFueraDeRango(t *testing.T) {
	for _, c := range [][3]string{{"500ms", "", ""}, {"2h", "", ""}, {"diez", "", ""}, {"", "0", ""}, {"", "101", ""}, {"", "", "-1s"}, {"", "", "25h"}} {
		if _, err := NuevaConfiguracionEntregaContratosCTBolsa(c[0], c[1], c[2]).Resolver(); !errors.Is(err, ErrEntregaContratosCTBolsaInvalida) {
			t.Errorf("%v aceptado", c)
		}
	}
}

func TestEntregaContratosCTBolsaDesdeEntorno(t *testing.T) {
	t.Setenv(EnvBolsaContratosCTIntervalo, "45s")
	t.Setenv(EnvBolsaContratosCTLote, "20")
	t.Setenv(EnvBolsaContratosCTRelectura, "1m")
	r, err := Load().BolsaContratosCT.Resolver()
	if err != nil || r.Intervalo != 45*time.Second || r.Lote != 20 || r.Relectura != time.Minute {
		t.Fatalf("entorno=%+v err=%v", r, err)
	}
}
