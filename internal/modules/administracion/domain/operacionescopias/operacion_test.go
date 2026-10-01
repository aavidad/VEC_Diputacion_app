package operacionescopias

import (
	"errors"
	"strings"
	"testing"
)

func solicitud() Solicitud {
	return Solicitud{Operacion: "operacion:1", Clave: "solicitud:1", SHA256: strings.Repeat("1", 64), Conjunto: "conjunto:1", Destino: "destino:1", Politica: "politica:1"}
}

func comando(o Operacion, action string) Comando {
	return Comando{Clave: action, SolicitudSHA256: o.solicitud.SHA256, VersionEsperada: o.Version(), Accion: action}
}

func aplicar(t *testing.T, o Operacion, c Comando) Operacion {
	t.Helper()
	next, _, replay, err := o.Aplicar(c)
	if err != nil || replay {
		t.Fatalf("apply: %v replay=%v", err, replay)
	}
	return next
}

func verificando(t *testing.T) Operacion {
	t.Helper()
	o, err := Nueva(solicitud())
	if err != nil {
		t.Fatal(err)
	}
	o = aplicar(t, o, comando(o, "iniciar_captura"))
	c := comando(o, "confirmar_captura")
	c.ManifiestoSHA256 = strings.Repeat("2", 64)
	o = aplicar(t, o, c)
	c = comando(o, "iniciar_verificacion")
	c.ManifiestoSHA256 = o.manifiesto
	c.Ejecucion = "ejecucion:1"
	return aplicar(t, o, c)
}

func ensayo(o Operacion, mode string) Comando {
	c := comando(o, "declarar_ensayo")
	c.Clave = "ensayo:" + mode
	c.Evidencia = &Evidencia{Modo: mode, Conjunto: o.solicitud.Conjunto, ManifiestoSHA256: o.manifiesto, Ejecucion: o.ejecucion, Referencia: "evidencia:" + mode, SHA256: strings.Repeat("3", 64), Resultado: "satisfactorio"}
	return c
}

func TestAmbosEnsayosMismoConjunto(t *testing.T) {
	for _, primero := range []string{"fisico", "logico"} {
		t.Run(primero, func(t *testing.T) {
			o := verificando(t)
			o = aplicar(t, o, ensayo(o, primero))
			if o.Estado() != Verificando || o.Reconciliar() != "conciliar_ensayos_pendientes" {
				t.Fatal("single test accepted")
			}
			segundo := "logico"
			if primero == "logico" {
				segundo = "fisico"
			}
			o = aplicar(t, o, ensayo(o, segundo))
			if o.Estado() != VerificadaDeclarada {
				t.Fatal(o.Estado())
			}
			recovered, err := Reconstruir(solicitud(), o.Historia())
			if err != nil || recovered.Estado() != o.Estado() || recovered.Version() != o.Version() {
				t.Fatal("reconstruction failed", err)
			}
		})
	}
}

func TestReplayReciboOriginalYConflicto(t *testing.T) {
	o := verificando(t)
	c := ensayo(o, "fisico")
	next, receipt, _, err := o.Aplicar(c)
	if err != nil {
		t.Fatal(err)
	}
	next = aplicar(t, next, ensayo(next, "logico"))
	r, original, replay, err := next.Aplicar(c)
	if err != nil || !replay || r.Version() != next.Version() || original.Version != receipt.Version || original.SelloSHA256 != receipt.SelloSHA256 {
		t.Fatal("replay changed receipt/history", err)
	}
	c.Evidencia.Resultado = "fallido"
	if _, _, _, err := next.Aplicar(c); !errors.Is(err, ErrConflicto) {
		t.Fatal(err)
	}
}

func TestFallosNoValidanNiCambianHistoria(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*Comando)
		want   error
	}{
		{"cas", func(c *Comando) { c.VersionEsperada++ }, ErrVersion},
		{"request", func(c *Comando) { c.SolicitudSHA256 = strings.Repeat("4", 64) }, ErrVinculo},
		{"bundle", func(c *Comando) { c.Evidencia.Conjunto = "conjunto:otro" }, ErrVinculo},
		{"manifest", func(c *Comando) { c.Evidencia.ManifiestoSHA256 = strings.Repeat("4", 64) }, ErrVinculo},
		{"run", func(c *Comando) { c.Evidencia.Ejecucion = "ejecucion:otra" }, ErrVinculo},
		{"evidence", func(c *Comando) { c.Evidencia.Referencia = "" }, ErrEntrada},
		{"result", func(c *Comando) { c.Evidencia.Resultado = "desconocido" }, ErrEntrada},
	} {
		t.Run(test.name, func(t *testing.T) {
			o := verificando(t)
			c := ensayo(o, "fisico")
			test.change(&c)
			r, _, _, err := o.Aplicar(c)
			if !errors.Is(err, test.want) || r.Version() != o.Version() || r.Estado() != Verificando {
				t.Fatalf("%v", err)
			}
		})
	}
	o := verificando(t)
	c := ensayo(o, "fisico")
	c.Evidencia.Resultado = "fallido"
	o = aplicar(t, o, c)
	if o.Estado() != NoValidaDeclarada {
		t.Fatal("failed test accepted")
	}
	if _, _, _, err := o.Aplicar(ensayo(o, "logico")); !errors.Is(err, ErrTransicion) {
		t.Fatal(err)
	}
}

func TestHistoriaInalterableYReconstruccionEstricta(t *testing.T) {
	o := verificando(t)
	c := ensayo(o, "fisico")
	next := aplicar(t, o, c)
	c.Evidencia.Referencia = "alterada"
	h := next.Historia()
	h[len(h)-1].Comando.Evidencia.Conjunto = "alterado"
	if o.Version() != 3 || next.Historia()[3].Comando.Evidencia.Referencia != "evidencia:fisico" {
		t.Fatal("caller mutated state")
	}
	for _, alterar := range []func([]Evento) []Evento{
		func(h []Evento) []Evento { return h[1:] },
		func(h []Evento) []Evento { h[1], h[2] = h[2], h[1]; return h },
		func(h []Evento) []Evento { return append(h, h[3]) },
		func(h []Evento) []Evento { h[3].Destino = "destino:otro"; return h },
		func(h []Evento) []Evento { h[3].Estado = VerificadaDeclarada; return h },
		func(h []Evento) []Evento { h[3].SelloSHA256 = strings.Repeat("0", 64); return h },
	} {
		if _, err := Reconstruir(solicitud(), alterar(next.Historia())); !errors.Is(err, ErrHistoria) {
			t.Fatal("corrupt history accepted", err)
		}
	}
	for n := 0; n <= len(next.Historia()); n++ {
		r, err := Reconstruir(solicitud(), next.Historia()[:n])
		if err != nil || r.Estado() == VerificadaDeclarada {
			t.Fatal("interrupted state accepted", err)
		}
	}
}

func TestReservaRepetidaNoCreaOtraOperacion(t *testing.T) {
	s := solicitud()
	o, replay, err := Reservar(nil, s)
	if err != nil || replay {
		t.Fatal(err)
	}
	r, replay, err := Reservar(&o, s)
	if err != nil || !replay || r.Version() != 0 {
		t.Fatal(err)
	}
	s.Destino = "destino:otro"
	if _, _, err := Reservar(&o, s); !errors.Is(err, ErrConflicto) {
		t.Fatal(err)
	}
}
