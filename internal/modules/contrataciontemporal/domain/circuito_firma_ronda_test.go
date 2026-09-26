package domain

import (
	"strings"
	"testing"
	"time"
)

func circuitoRondaPrueba() CircuitoFirmaDocumento {
	return CircuitoFirmaDocumento{Documento: "informe_definitivo", Etiqueta: "Informe definitivo", Pasos: []PasoCircuitoFirma{
		{Orden: 1, Cargo: "Técnico/a de RRHH", Referencia: "circuito:1:p1", Devolucion: DevolucionVuelveARedaccion},
		{Orden: 2, Cargo: "Jefatura del Servicio de RRHH", Referencia: "circuito:1:p2", Devolucion: DevolucionVuelveARedaccion, Habilita: HabilitaRemisionIntervencion},
	}}
}

func firmaRondaPrueba(secuencia, paso int, version uint64, original string) EventoFirmaDocumento {
	return EventoFirmaDocumento{
		Secuencia: secuencia, CatalogoHuella: strings.Repeat("a", 64), PasoOrden: paso, Resultado: ResultadoFirmaFirmado,
		OriginalHuella: original, FirmadoHuella: strings.Repeat(string(rune('0'+secuencia)), 64),
		ReciboRef: "recibo-firma:ronda", RegistradaEn: time.Date(2026, 9, 26, 10, secuencia, 0, 0, time.UTC),
		ExpedienteVersion: version,
	}
}

// Con el informe nuevo (versión 9) el circuito completo de la primera ronda
// deja de contar: hay que firmar el documento nuevo desde el paso 1.
func TestCircuitoFirmaSegundaRondaTrasInformeNuevo(t *testing.T) {
	c := circuitoRondaPrueba()
	huella := strings.Repeat("a", 64)
	viejo, nuevo := strings.Repeat("b", 64), strings.Repeat("c", 64)
	eventos := []EventoFirmaDocumento{firmaRondaPrueba(1, 1, 5, viejo), firmaRondaPrueba(2, 2, 5, viejo)}
	unica, err := CalcularEstadoCircuitoFirma(c, huella, eventos)
	if err != nil || !unica.Completo {
		t.Fatalf("la primera ronda debía estar completa: %+v %v", unica, err)
	}
	ronda, err := CalcularEstadoCircuitoFirmaEnRonda(c, huella, eventos, 9)
	if err != nil || ronda.Completo || ronda.PasoPendiente != 1 || ronda.EventosRondaAnterior != 2 ||
		ronda.UltimaSecuencia != 2 || ronda.EventosCatalogoAnterior != 0 {
		t.Fatalf("la segunda ronda debe empezar en el paso 1: %+v %v", ronda, err)
	}
	if p := PasosRemisionSinFirmar(CircuitoFirma{Documentos: []CircuitoFirmaDocumento{c}}, []EstadoCircuitoDocumento{ronda}); len(p) != 1 {
		t.Fatalf("sin firmar el informe nuevo no cabe remitir: %+v", p)
	}
	eventos = append(eventos, firmaRondaPrueba(3, 1, 9, nuevo), firmaRondaPrueba(4, 2, 10, nuevo))
	ronda, err = CalcularEstadoCircuitoFirmaEnRonda(c, huella, eventos, 9)
	if err != nil || !ronda.Completo || ronda.EventosRondaAnterior != 2 {
		t.Fatalf("la segunda ronda firmada debía completarse: %+v %v", ronda, err)
	}
	if p := PasosRemisionSinFirmar(CircuitoFirma{Documentos: []CircuitoFirmaDocumento{c}}, []EstadoCircuitoDocumento{ronda}); len(p) != 0 {
		t.Fatalf("con el informe nuevo firmado cabe remitir: %+v", p)
	}
	// El paso 2 de la ronda nueva no puede firmar el borrador de la anterior.
	cruzado := append(eventos[:3:3], firmaRondaPrueba(4, 2, 10, viejo))
	if _, err := CalcularEstadoCircuitoFirmaEnRonda(c, huella, cruzado, 9); err == nil {
		t.Fatal("se aceptó una firma de la ronda nueva sobre el borrador anterior")
	}
}
