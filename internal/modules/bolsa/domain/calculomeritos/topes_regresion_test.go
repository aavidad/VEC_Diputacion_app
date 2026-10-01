package calculomeritos

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestTopesConservanHuellasHistoricas(t *testing.T) {
	// Huellas del motor anterior a la extraccion del operador comun.
	for _, caso := range []struct{ reglas, entrada, huella string }{
		{"meritos_reglas_a", "meritos_entrada", "0f9f896229d4de8fb3ce9eae0798d98fa9d73df11af36712a124e8ceb7d379fc"},
		{"meritos_reglas_b", "meritos_entrada", "835aff04fed3004b7ce731db3b2cad349e75e279910037a16cecf7b57c8b519d"},
		{"meritos_reglas_a", "meritos_bloqueada", "8ad39d4d577aed9523b735482c06ab613bf1f0622f5e8e18452507c7a48bdba9"},
	} {
		t.Run(caso.reglas+"/"+caso.entrada, func(t *testing.T) {
			c, _ := ejemplo(t, caso.reglas)
			b := fixture(t, caso.entrada)
			e, err := RestaurarEntrada(b, HuellaSHA256(b))
			if err != nil {
				t.Fatal(err)
			}
			r, err := Calcular(c, e)
			if err != nil {
				t.Fatal(err)
			}
			h, err := r.HuellaSHA256()
			if err != nil || h != caso.huella {
				t.Fatalf("huella historica modificada: %s, %v", h, err)
			}
		})
	}
}

func TestTopeBajoNoOcultaDesbordamientoPrevio(t *testing.T) {
	for _, fase := range []string{"producto", "suma_reglas", "suma_secciones"} {
		t.Run(fase, func(t *testing.T) {
			c, e := ejemplo(t, "meritos_reglas_a")
			c.MaximoTotal = puntos(t, 1)
			if fase == "producto" {
				c.Reglas[0].PuntosPorUnidad = puntos(t, baremacion.MaximoMicropuntos)
				c.Reglas[0].MaximoPuntos = puntos(t, 1)
			} else {
				// Cada regla produce un valor valido; la suma supera el limite
				// tecnico antes del tope de seccion o del total.
				e.Meritos = []Merito{e.Meritos[0], e.Meritos[6]}
				for i := range e.Meritos {
					e.Meritos[i].Unidades = racional(t, 1, 1)
				}
				for i := 0; i < 2; i++ {
					c.Reglas[i].MinimoUnidades = racional(t, 0, 1)
					c.Reglas[i].PuntosPorUnidad = puntos(t, baremacion.MaximoMicropuntos)
					c.Reglas[i].MaximoPuntos = puntos(t, baremacion.MaximoMicropuntos)
				}
				if fase == "suma_reglas" {
					c.Reglas[1].SeccionClave = c.Reglas[0].SeccionClave
					c.Secciones = []Seccion{c.Secciones[0], c.Secciones[2]}
					c.Secciones[0].MaximoPuntos = puntos(t, 1)
				} else {
					for i := range c.Secciones {
						c.Secciones[i].MaximoPuntos = puntos(t, baremacion.MaximoMicropuntos)
					}
				}
			}
			r, err := Calcular(c, e)
			if !errors.Is(err, baremacion.ErrDesbordamiento) || r.Total != nil || r.SumaSecciones != nil || len(r.Reglas) != 0 {
				t.Fatalf("tope oculto desbordamiento: %+v, %v", r, err)
			}
		})
	}
}
