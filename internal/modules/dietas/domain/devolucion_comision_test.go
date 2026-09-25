package domain

import "testing"

func comisionDevuelta() ComisionBorrador {
	return ComisionBorrador{Referencia: "dco_1234567890123456789012", NumeroDocumento: "VEC-D-2026-000001",
		FechaApertura: "2026-09-23T07:00:00.000000Z", Version: 4, Estado: EstadoDevuelta,
		FechaInicio: "2026-09-23", FechaFin: "2026-09-23", Motivo: "Visita sintética",
		CodigosRuta: []string{"GR1", "GR2"}, RelacionRef: "rel_1234567890123456789012",
		Devolucion: &DevolucionComision{Etapa: EtapaAutorizacion, Motivo: "Falta el justificante del taxi", Version: 4, DevueltaEn: "2026-09-23T09:00:00.123456Z"}}
}

func TestDevolucionAcompanaAlDocumentoDevueltoOEnCorreccion(t *testing.T) {
	c := comisionDevuelta()
	if err := c.Validar(); err != nil {
		t.Fatalf("devuelta: %v", err)
	}
	c.Estado, c.Version = "borrador", 5
	if err := c.Validar(); err != nil {
		t.Fatalf("en corrección: %v", err)
	}
	// Una devuelta anterior a la proyección de la devolución sigue siendo legible.
	c = comisionDevuelta()
	c.Devolucion = nil
	if err := c.Validar(); err != nil {
		t.Fatalf("devuelta sin devolución proyectada: %v", err)
	}
}

func TestDevolucionRechazaFormasIncoherentes(t *testing.T) {
	casos := map[string]func(*ComisionBorrador){
		"estado enviado":      func(c *ComisionBorrador) { c.Estado = EstadoPendienteAutorizacion },
		"estado eliminado":    func(c *ComisionBorrador) { c.Estado = "eliminado" },
		"etapa desconocida":   func(c *ComisionBorrador) { c.Devolucion.Etapa = "otra" },
		"motivo corto":        func(c *ComisionBorrador) { c.Devolucion.Motivo = "no" },
		"motivo con control":  func(c *ComisionBorrador) { c.Devolucion.Motivo = "Falta\njustificante" },
		"motivo con espacios": func(c *ComisionBorrador) { c.Devolucion.Motivo = " Falta justificante" },
		"versión futura":      func(c *ComisionBorrador) { c.Devolucion.Version = 5 },
		"versión imposible":   func(c *ComisionBorrador) { c.Devolucion.Version = 2 },
		"fecha no canónica":   func(c *ComisionBorrador) { c.Devolucion.DevueltaEn = "2026-09-23T09:00:00Z" },
	}
	for nombre, mutar := range casos {
		c := comisionDevuelta()
		d := *c.Devolucion
		c.Devolucion = &d
		mutar(&c)
		if c.Validar() == nil {
			t.Errorf("%s: aceptada", nombre)
		}
	}
}

// Go y el cliente web recortan conjuntos distintos: Go quita U+0085 y el
// navegador U+FEFF. Un motivo con cualquiera de ellos en un extremo se
// rechaza, para que la titular vea exactamente lo que su cliente acepta.
func TestTextoSinBordesCubreLoQueRecortanGoYElNavegador(t *testing.T) {
	for _, borde := range []string{" ", "\u00a0", "\u0085", "\ufeff", "\u2028", "\u3000", "\u200a"} {
		for _, motivo := range []string{borde + "Falta justificante", "Falta justificante" + borde} {
			if TextoSinBordes(motivo) {
				t.Errorf("%q aceptado", motivo)
			}
			if _, err := ResolverDecisionCircuito(EstadoEnviadoPendienteRevision, EtapaRevision, DecisionDevolver, motivo, "act_a", "act_b", nil); err == nil {
				t.Errorf("devolución con %q aceptada", motivo)
			}
			c := comisionDevuelta()
			d := *c.Devolucion
			d.Motivo = motivo
			c.Devolucion = &d
			if c.Validar() == nil {
				t.Errorf("devolución proyectada con %q aceptada", motivo)
			}
		}
	}
	if !TextoSinBordes("Falta\ufeffel justificante") || !TextoSinBordes("Falta el justificante") {
		t.Fatal("blanco interior rechazado")
	}
}

func TestDevolucionAnteriorAUnReenvio(t *testing.T) {
	d := *comisionDevuelta().Devolucion
	if err := d.ValidarAnteriorA(5); err != nil {
		t.Fatalf("devolución anterior al reenvío: %v", err)
	}
	for nombre, version := range map[string]uint64{"misma versión": 4, "anterior": 3} {
		if d.ValidarAnteriorA(version) == nil {
			t.Errorf("%s: aceptada", nombre)
		}
	}
	d.Etapa = "otra"
	if d.ValidarAnteriorA(5) == nil {
		t.Error("etapa desconocida aceptada")
	}
}
