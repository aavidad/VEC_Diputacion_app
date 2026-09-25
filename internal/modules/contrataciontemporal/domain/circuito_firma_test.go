package domain

import (
	"errors"
	"strings"
	"testing"
)

func circuitoPrueba() CircuitoFirmaDocumento {
	return CircuitoFirmaDocumento{Documento: "informe_definitivo", Etiqueta: "Informe definitivo", Pasos: []PasoCircuitoFirma{
		{Orden: 1, Cargo: "Técnico", PerfilRef: "perfil:ct:tecnico", Accion: "firma", Devolucion: DevolucionVuelveARedaccion, Referencia: "c:1:p1"},
		{Orden: 2, Cargo: "Jefatura", PerfilRef: "perfil:ct:jefatura", Accion: "firma", Devolucion: DevolucionVuelvePasoAnterior, Referencia: "c:1:p2"},
		{Orden: 3, Cargo: "Diputado", PerfilRef: "perfil:ct:diputado", Accion: "firma", Devolucion: DevolucionVuelveARedaccion, Referencia: "c:1:p3"},
	}}
}

var (
	catalogo = strings.Repeat("c", 64)
	h1       = strings.Repeat("1", 64)
	h2       = strings.Repeat("2", 64)
	h3       = strings.Repeat("3", 64)
	hOrig    = strings.Repeat("a", 64)
)

func estados(e EstadoCircuitoDocumento) string {
	var b []string
	for _, p := range e.Pasos {
		b = append(b, string(p.Estado))
	}
	return strings.Join(b, ",")
}

func TestEstadoCircuitoSinFirmas(t *testing.T) {
	e, err := CalcularEstadoCircuitoFirma(circuitoPrueba(), catalogo, nil)
	if err != nil || e.PasoPendiente != 1 || e.Completo || e.UltimaSecuencia != 0 || e.OriginalEsperadoHuella != "" ||
		estados(e) != "pendiente_firma,en_espera,en_espera" {
		t.Fatalf("estado inicial: %+v %v", e, err)
	}
}

func TestEstadoCircuitoFirmasEncadenadasYCompleto(t *testing.T) {
	ev := []EventoFirmaDocumento{
		{Secuencia: 1, CatalogoHuella: catalogo, PasoOrden: 1, Resultado: ResultadoFirmaFirmado, OriginalHuella: hOrig, FirmadoHuella: h1},
		{Secuencia: 2, CatalogoHuella: catalogo, PasoOrden: 2, Resultado: ResultadoFirmaFirmado, OriginalHuella: hOrig, FirmadoHuella: h2},
	}
	e, err := CalcularEstadoCircuitoFirma(circuitoPrueba(), catalogo, ev)
	if err != nil || e.PasoPendiente != 3 || e.OriginalEsperadoHuella != hOrig || estados(e) != "firmado,firmado,pendiente_firma" {
		t.Fatalf("dos firmas: %+v %v", e, err)
	}
	ev = append(ev, EventoFirmaDocumento{Secuencia: 3, CatalogoHuella: catalogo, PasoOrden: 3, Resultado: ResultadoFirmaFirmado, OriginalHuella: hOrig, FirmadoHuella: h3})
	e, err = CalcularEstadoCircuitoFirma(circuitoPrueba(), catalogo, ev)
	if err != nil || !e.Completo || e.PasoPendiente != 0 || estados(e) != "firmado,firmado,firmado" {
		t.Fatalf("completo: %+v %v", e, err)
	}
}

func TestEstadoCircuitoDevoluciones(t *testing.T) {
	base := []EventoFirmaDocumento{
		{Secuencia: 1, CatalogoHuella: catalogo, PasoOrden: 1, Resultado: ResultadoFirmaFirmado, OriginalHuella: hOrig, FirmadoHuella: h1},
		{Secuencia: 2, CatalogoHuella: catalogo, PasoOrden: 2, Resultado: ResultadoFirmaDevuelto, ConMotivoDevolucion: true},
	}
	// El paso 2 vuelve al paso anterior: el 1 debe firmar otra vez.
	e, err := CalcularEstadoCircuitoFirma(circuitoPrueba(), catalogo, base)
	if err != nil || e.PasoPendiente != 1 || estados(e) != "pendiente_firma,devuelto,en_espera" || e.Pasos[1].Estado != EstadoPasoDevuelto {
		t.Fatalf("vuelve al paso anterior: %+v %v", e, err)
	}
	base = append(base,
		EventoFirmaDocumento{Secuencia: 3, CatalogoHuella: catalogo, PasoOrden: 1, Resultado: ResultadoFirmaFirmado, OriginalHuella: hOrig, FirmadoHuella: h1},
		EventoFirmaDocumento{Secuencia: 4, CatalogoHuella: catalogo, PasoOrden: 2, Resultado: ResultadoFirmaFirmado, OriginalHuella: hOrig, FirmadoHuella: h2},
		EventoFirmaDocumento{Secuencia: 5, CatalogoHuella: catalogo, PasoOrden: 3, Resultado: ResultadoFirmaDevuelto, ConMotivoDevolucion: true},
	)
	// El paso 3 vuelve a redacción: todo el circuito empieza de nuevo.
	e, err = CalcularEstadoCircuitoFirma(circuitoPrueba(), catalogo, base)
	if err != nil || e.PasoPendiente != 1 || e.UltimaSecuencia != 5 || estados(e) != "pendiente_firma,en_espera,devuelto" {
		t.Fatalf("vuelve a redacción: %+v %v", e, err)
	}
}

func TestEstadoCircuitoRechazaHistoriaIncoherente(t *testing.T) {
	casos := map[string][]EventoFirmaDocumento{
		"salto de secuencia": {{Secuencia: 2, CatalogoHuella: catalogo, PasoOrden: 1, Resultado: ResultadoFirmaFirmado, OriginalHuella: hOrig, FirmadoHuella: h1}},
		"paso no pendiente":  {{Secuencia: 1, CatalogoHuella: catalogo, PasoOrden: 2, Resultado: ResultadoFirmaFirmado, OriginalHuella: hOrig, FirmadoHuella: h1}},
		"cadena rota": {
			{Secuencia: 1, CatalogoHuella: catalogo, PasoOrden: 1, Resultado: ResultadoFirmaFirmado, OriginalHuella: hOrig, FirmadoHuella: h1},
			{Secuencia: 2, CatalogoHuella: catalogo, PasoOrden: 2, Resultado: ResultadoFirmaFirmado, OriginalHuella: h1, FirmadoHuella: h2},
		},
		"devolución sin motivo": {{Secuencia: 1, CatalogoHuella: catalogo, PasoOrden: 1, Resultado: ResultadoFirmaDevuelto}},
		"resultado ajeno":       {{Secuencia: 1, CatalogoHuella: catalogo, PasoOrden: 1, Resultado: "anulado"}},
	}
	for nombre, ev := range casos {
		if _, err := CalcularEstadoCircuitoFirma(circuitoPrueba(), catalogo, ev); !errors.Is(err, ErrHistoriaFirmaIncoherente) {
			t.Fatalf("%s: %v", nombre, err)
		}
	}
}

func TestEstadoCircuitoCatalogoNuevoReinicia(t *testing.T) {
	otro := strings.Repeat("d", 64)
	ev := []EventoFirmaDocumento{{Secuencia: 1, CatalogoHuella: otro, PasoOrden: 2, Resultado: ResultadoFirmaFirmado, OriginalHuella: hOrig, FirmadoHuella: h1}}
	e, err := CalcularEstadoCircuitoFirma(circuitoPrueba(), catalogo, ev)
	if err != nil || e.PasoPendiente != 1 || e.UltimaSecuencia != 1 || e.EventosCatalogoAnterior != 1 {
		t.Fatalf("catálogo nuevo: %+v %v", e, err)
	}
}

func TestCircuitoFirmaDocumentoIncoherente(t *testing.T) {
	c := circuitoPrueba()
	c.Pasos[0].Devolucion = DevolucionVuelvePasoAnterior
	if _, err := CalcularEstadoCircuitoFirma(c, catalogo, nil); !errors.Is(err, ErrCircuitoFirmaIncoherente) {
		t.Fatal("el primer paso no puede volver al anterior")
	}
	c = circuitoPrueba()
	c.Pasos[2].Orden = 4
	if c.Validar() == nil {
		t.Fatal("hueco en los pasos admitido")
	}
}
