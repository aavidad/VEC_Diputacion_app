package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestRegistroMOADExigeNumeroAntesDeReservar(t *testing.T) {
	e := nuevoEscenarioRegistro(t)
	s, d := construirServicioRegistro(t, e)
	e.solicitud.NumeroExpedienteMOAD = ""
	if _, err := s.Registrar(context.Background(), e.solicitud); !errors.Is(err, ErrSolicitudRegistroInvalida) {
		t.Fatalf("alta sin MOAD: %v", err)
	}
	if d.referencias.llamadasReferencias != 0 {
		t.Fatal("se generaron referencias para un número ausente")
	}
	p := domain.PoliticaNumeroExpediente{Referencia: "catalogo:ct:moad", Version: 2, Patron: `^[0-9]{4}/[1-9][0-9]{0,9}$`, Ejemplo: "2026/5487"}
	s.politicaNumero = &p
	e.solicitud.NumeroExpedienteMOAD = "2026/CT-0001"
	recuperacion := &recuperadorPoliticaFinPrueba{}
	if err := s.ConfigurarRecuperacionPoliticaFin(recuperacion); err != nil {
		t.Fatal(err)
	}
	for intento := 0; intento < 2; intento++ {
		if _, err := s.Registrar(context.Background(), e.solicitud); !errors.Is(err, ErrSolicitudRegistroInvalida) {
			t.Fatalf("intento %d: se ignoró formato configurado: %v", intento+1, err)
		}
	}
	if recuperacion.consultas != 2 || d.candidaturas.llamadas != 0 || d.autorizador.llamadas != 0 || d.transaccion.llamadas != 0 {
		t.Fatal("un número rechazado por el catálogo alcanzó autorización o confirmación")
	}
}

func TestRegistroMOADRecuperaNumeroDeFormatoRetirado(t *testing.T) {
	e := nuevoEscenarioRegistro(t)
	datos, err := e.candidatura.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.Recuperada = true
	e.candidatura, err = ports.NuevaCandidaturaAlta(datos)
	if err != nil {
		t.Fatal(err)
	}
	s, d := construirServicioRegistro(t, e)
	p := domain.PoliticaNumeroExpediente{Referencia: "catalogo:ct:moad", Version: 2, Patron: `^[0-9]{4}/[1-9][0-9]{0,9}$`, Ejemplo: "2026/5487"}
	if err := s.ConfigurarPoliticaNumeroExpediente(p); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfigurarRecuperacionPoliticaFin(&recuperadorPoliticaFinPrueba{confirmada: true}); err != nil {
		t.Fatal(err)
	}
	recibo, err := s.Registrar(context.Background(), e.solicitud)
	if err != nil || recibo != e.recibo || d.transaccion.llamadas != 1 {
		t.Fatalf("replay con formato retirado: recibo=%+v error=%v transacciones=%d", recibo, err, d.transaccion.llamadas)
	}
}

func TestRegistroMOADRecuperaAltaAnteriorSinAtribuirMOAD(t *testing.T) {
	e := nuevoEscenarioRegistro(t)
	datos, err := e.candidatura.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.Recuperada = true
	e.candidatura, err = ports.NuevaCandidaturaAlta(datos)
	if err != nil {
		t.Fatal(err)
	}
	e.solicitud.NumeroExpedienteMOAD = ""
	s, d := construirServicioRegistro(t, e)
	p := domain.PoliticaNumeroExpediente{Referencia: "catalogo:ct:moad", Version: 2, Patron: `^[0-9]{4}/[1-9][0-9]{0,9}$`, Ejemplo: "2026/5487"}
	if err := s.ConfigurarPoliticaNumeroExpediente(p); err != nil {
		t.Fatal(err)
	}
	confirmacion := &recuperadorPoliticaFinPrueba{confirmada: true}
	if err := s.ConfigurarRecuperacionPoliticaFin(confirmacion); err != nil {
		t.Fatal(err)
	}
	d.referencias.referencias.NumeroVisible = p.Ejemplo
	d.huellas.antes = func(m *ports.MaterialHuellaAlta) {
		if m.NumeroExpedienteMOAD != "" {
			t.Fatal("el número histórico se atribuyó a MOAD en el HMAC")
		}
	}
	recibo, err := s.Registrar(context.Background(), e.solicitud)
	if err != nil || recibo != e.recibo || d.transaccion.llamadas != 1 || confirmacion.consultas != 1 {
		t.Fatalf("alta anterior: recibo=%+v error=%v transacciones=%d consultas=%d", recibo, err, d.transaccion.llamadas, confirmacion.consultas)
	}
}

func TestRegistroMOADNoAdmiteSustituirNumeroEnCandidatura(t *testing.T) {
	e := nuevoEscenarioRegistro(t)
	s, d := construirServicioRegistro(t, e)
	e.solicitud.NumeroExpedienteMOAD = "2026/5487"
	d.referencias.referencias.NumeroVisible = e.solicitud.NumeroExpedienteMOAD
	if _, err := s.Registrar(context.Background(), e.solicitud); !errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
		t.Fatalf("otra numeración reservada: %v", err)
	}
}
