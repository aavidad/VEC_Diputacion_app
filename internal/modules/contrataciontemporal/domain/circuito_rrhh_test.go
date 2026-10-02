package domain

import (
	"strings"
	"testing"
)

func TestCircuitoRRHHLigaDosHitosAlActoSinVersionArtificial(t *testing.T) {
	base := expedienteConAnalisisRehidratado(t, RCValidada)
	inicial := base.Actuaciones[0].FaseDestino
	definicion, err := NuevaDefinicionCircuitoRRHH(
		"flujo:ct:rrhh:prueba", 2, inicial,
		[]TransicionCircuitoRRHH{
			{Clave: "peticion_firmada", Tipo: HitoPeticionFirmada, Origen: inicial,
				Destino: "autorizacion_rrhh", RequiereDocumento: true, RequiereFirma: true,
				PerfilClave: "tecnico", FirmasRequeridas: []ClaveCatalogo{"tecnico", "delegacion"}},
			{Clave: "autorizacion_rrhh", Tipo: HitoAutorizacionRRHH, Origen: "autorizacion_rrhh",
				Destino: "credito", RequiereDocumento: true, PerfilClave: "direccion_rrhh"},
			{Clave: "oferta_emitida", Tipo: HitoOfertaEmitida, Origen: "credito",
				Destino: "adjudicacion", PerfilClave: "rrhh"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	base.Flujo = definicion.Flujo
	circuito, err := NuevoCircuitoAdministrativo(definicion)
	if err != nil {
		t.Fatal(err)
	}
	base.Circuito = &circuito
	if err := base.Validar(); err != nil {
		t.Fatalf("alta ligada al triple: %v", err)
	}
	act := base.Actuaciones[len(base.Actuaciones)-1]
	ahora := act.RealizadaEn
	peticion := HitoCircuitoRRHH{
		Clave: "peticion_firmada", ActuacionClave: act.AccionClave, ActorRef: act.ActorRef,
		PerfilClave: "tecnico", PerfilRef: "perfil:tecnico:sintetico",
		UnidadRef: act.UnidadRef, DocumentoRef: "documento:peticion:sintetica", HuellaDocumentoSHA256: strings.Repeat("a", 64),
		Firmas: []FirmaCircuitoRRHH{
			{CargoClave: "tecnico", FirmaRef: "firma:tecnico:sintetica", HuellaDocumentoSHA256: strings.Repeat("a", 64)},
			{CargoClave: "delegacion", FirmaRef: "firma:delegacion:sintetica", HuellaDocumentoSHA256: strings.Repeat("a", 64)},
		},
		ReciboRef: act.ReciboRef, AuditoriaRef: "auditoria:peticion:sintetica",
		EventoRef: "evento:peticion:sintetica", RegistradoEn: ahora,
	}
	autorizacion := HitoCircuitoRRHH{
		Clave: "autorizacion_rrhh", ActuacionClave: act.AccionClave, ActorRef: act.ActorRef,
		PerfilClave: "direccion_rrhh", PerfilRef: "perfil:direccion_rrhh:sintetico",
		UnidadRef: act.UnidadRef, DocumentoRef: "documento:autorizacion:sintetica", HuellaDocumentoSHA256: strings.Repeat("b", 64),
		ReciboRef: act.ReciboRef, AuditoriaRef: "auditoria:autorizacion:sintetica",
		EventoRef: "evento:autorizacion:sintetica", RegistradoEn: ahora,
	}
	conHitos, err := base.AdjuntarHitosCircuito(definicion, base.Version, []HitoCircuitoRRHH{peticion, autorizacion})
	if err != nil || conHitos.Validar() != nil {
		t.Fatalf("dos hitos en v2: %v", err)
	}
	if conHitos.Version != base.Version || len(conHitos.Circuito.Hitos) != 2 {
		t.Fatal("adjuntar hitos creó otra versión o perdió la historia")
	}
	oferta := HitoCircuitoRRHH{
		Clave: "oferta_emitida", ActuacionClave: act.AccionClave, ActorRef: act.ActorRef,
		PerfilClave: "rrhh", PerfilRef: "perfil:rrhh:sintetico",
		UnidadRef: act.UnidadRef, OfertaRef: "oferta:sintetica", ReciboRef: act.ReciboRef,
		AuditoriaRef: "auditoria:oferta:sintetica", EventoRef: "evento:oferta:sintetica",
		RegistradoEn: ahora,
	}
	if _, err := conHitos.AdjuntarHitosCircuito(definicion, conHitos.Version, []HitoCircuitoRRHH{oferta}); err == nil {
		t.Fatal("una oferta sin credito avanzó")
	}
	adulterado := conHitos.Clonar()
	adulterado.Circuito.Definicion.HuellaSHA256 = strings.Repeat("b", 64)
	if adulterado.Validar() == nil {
		t.Fatal("el circuito aceptó otro triple")
	}
}
