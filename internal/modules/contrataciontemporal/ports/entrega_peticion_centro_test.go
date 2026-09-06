package ports

import (
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const selloEntregaPeticionCentroPrueba = "hmac-sha256:contratacion-temporal.entrega-alta/v2:" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestContratoEntregaPeticionCentroExigeComandoV2YReferenciaCerrada(t *testing.T) {
	validos := []ComandoEntregarPeticionCentro{{PeticionRef: "peticion:centro:001", VersionEsperada: 2}}
	for _, comando := range validos {
		if err := comando.Validar(); err != nil {
			t.Fatalf("comando válido rechazado: %v", err)
		}
	}
	for _, caso := range []ComandoEntregarPeticionCentro{{PeticionRef: "peticion:centro:001", VersionEsperada: 1}, {PeticionRef: "peticion:centro:001", VersionEsperada: 3}, {PeticionRef: "no", VersionEsperada: 2}} {
		if !errors.Is(caso.Validar(), domain.ErrPeticionCentroInvalida) {
			t.Fatalf("comando inválido aceptado: %#v", caso)
		}
	}
}

func TestContratoEntregaPeticionCentroValidaFuenteRatificadaYReservaUnica(t *testing.T) {
	peticion := datosPeticionCentroEntregaPrueba()
	base := EntregaPeticionCentro{Peticion: peticion, EstadoEntrega: "preparada", ClaveAlta: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", AmbitoAltaHMAC: selloEntregaPeticionCentroPrueba, ActorRef: "actor:rrhh", PerfilRef: "perfil:rrhh"}
	if err := base.ValidarReserva(); err != nil {
		t.Fatalf("reserva válida: %v", err)
	}
	for nombre, mutar := range map[string]func(*EntregaPeticionCentro){
		"fuente": func(e *EntregaPeticionCentro) { e.Peticion.Solicitud.Detalle = "otra" },
		"clave":  func(e *EntregaPeticionCentro) { e.ClaveAlta = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb" },
		"actor":  func(e *EntregaPeticionCentro) { e.ActorRef = "actor:otro" },
		"perfil": func(e *EntregaPeticionCentro) { e.PerfilRef = "perfil:otro" },
	} {
		copia := base
		mutar(&copia)
		if err := copia.ValidarReserva(); err != nil && nombre == "fuente" {
			t.Fatalf("fuente ratificada alterada perdió forma sin detectar el puente: %v", err)
		}
		if nombre != "fuente" && copia.ClaveAlta == base.ClaveAlta && copia.ActorRef == base.ActorRef && copia.PerfilRef == base.PerfilRef {
			t.Fatalf("caso %s no mutó fixture", nombre)
		}
	}
}

func TestContratoEntregaPeticionCentroDeniegaRecibosYMaterialesDivergentes(t *testing.T) {
	peticion := datosPeticionCentroEntregaPrueba()
	recibo := reciboAltaPeticionCentroPrueba()
	for nombre, mutar := range map[string]func(*ReciboAlta){
		"referencia": func(r *ReciboAlta) { r.ReciboRef = "" },
		"version":    func(r *ReciboAlta) { r.Version = 0 },
		"sello":      func(r *ReciboAlta) { r.ReciboRef = "" },
	} {
		copia := recibo
		mutar(&copia)
		if copia.ValidarEstructura() == nil {
			t.Fatalf("recibo divergente aceptado: %s", nombre)
		}
	}
	for _, material := range []MaterialEntregaPeticionCentro{
		{Modo: "preparar", ActorRef: "actor:rrhh", PerfilRef: "perfil:rrhh", PeticionRef: peticion.Referencia, VersionEsperada: 1},
		{Modo: "confirmar", ActorRef: "actor:rrhh", PerfilRef: "perfil:rrhh", PeticionRef: "peticion:otra", VersionEsperada: 2, ReciboAlta: &recibo, AmbitoAltaHMAC: selloEntregaPeticionCentroPrueba + "a"},
	} {
		if material.Validar() == nil {
			t.Fatalf("material divergente aceptado: %#v", material)
		}
	}
}

func TestContratoEntregaPeticionCentroMaterialBandejaYConfirmacionValidos(t *testing.T) {
	peticion := datosPeticionCentroEntregaPrueba()
	if err := (MaterialEntregaPeticionCentro{Modo: "bandeja", ActorRef: "actor:rrhh", PerfilRef: "perfil:rrhh"}).Validar(); err != nil {
		t.Fatal(err)
	}
	if err := (MaterialEntregaPeticionCentro{Modo: "preparar", ActorRef: "actor:rrhh", PerfilRef: "perfil:rrhh", PeticionRef: peticion.Referencia, VersionEsperada: 2, ClaveAltaCandidata: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", AmbitoAltaHMAC: selloEntregaPeticionCentroPrueba}).Validar(); err != nil {
		t.Fatal(err)
	}
	if err := (MaterialEntregaPeticionCentro{Modo: "confirmar", ActorRef: "actor:rrhh", PerfilRef: "perfil:rrhh", PeticionRef: peticion.Referencia, VersionEsperada: 2, ReciboAlta: func() *ReciboAlta { r := reciboAltaPeticionCentroPrueba(); return &r }(), AmbitoAltaHMAC: selloEntregaPeticionCentroPrueba}).Validar(); err != nil {
		t.Fatal(err)
	}
}

func reciboAltaPeticionCentroPrueba() ReciboAlta {
	return ReciboAlta{ExpedienteRef: "expediente:ct:001", NumeroVisible: "2026/5487", Version: 1, ReciboRef: "recibo:alta:001", AuditoriaRef: "auditoria:alta:001", EventoRef: "evento:alta:001", ConfirmadaEn: tiempoEntregaPeticionCentroPrueba}
}

func datosPeticionCentroEntregaPrueba() (datos domain.DatosPeticionCentro) {
	config := configuracionPeticionPuertoPrueba()
	creada := time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)
	peticion, err := domain.NuevaPeticionCentro("peticion:centro:001", config, *solicitudPeticionPuertoPrueba(), creada)
	if err != nil {
		panic(err)
	}
	ratificada, err := peticion.Ratificar(config.Ratificador, 1, "Necesidad confirmada", creada.Add(time.Minute))
	if err != nil {
		panic(err)
	}
	return ratificada.Datos()
}

var tiempoEntregaPeticionCentroPrueba = tiempoCanonicoEntregaPeticionCentroPrueba()

func tiempoCanonicoEntregaPeticionCentroPrueba() (t time.Time) {
	return time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
}
