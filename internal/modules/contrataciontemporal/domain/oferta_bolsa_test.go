package domain

import (
	"strings"
	"testing"
	"time"
)

func expedienteConCreditoParaOfertaPrueba(t *testing.T) (Expediente, DefinicionCircuitoRRHH) {
	t.Helper()
	e := expedienteConAsignacion(t)
	inicial := e.Actuaciones[0].FaseDestino
	d, err := NuevaDefinicionCircuitoRRHH("flujo:ct:rrhh:oferta-prueba", 2, inicial,
		[]TransicionCircuitoRRHH{
			{Clave: "peticion_firmada", Tipo: HitoPeticionFirmada, Origen: inicial, Destino: "autorizacion_rrhh", RequiereDocumento: true, RequiereFirma: true, PerfilClave: "tecnico_rrhh", FirmasRequeridas: []ClaveCatalogo{"tecnico", "delegacion"}},
			{Clave: "autorizacion_rrhh", Tipo: HitoAutorizacionRRHH, Origen: "autorizacion_rrhh", Destino: "credito", RequiereDocumento: true, PerfilClave: "tecnico_rrhh", AutorizanteCargoClave: "direccion_rrhh"},
			{Clave: "credito_comprobado", Tipo: HitoCreditoComprobado, Origen: "credito", Destino: "oferta", RequiereDocumento: true, PerfilClave: "tecnico_rrhh"},
			{Clave: "oferta_emitida", Tipo: HitoOfertaEmitida, Origen: "oferta", Destino: "adjudicacion", PerfilClave: "tecnico_rrhh"},
		})
	if err != nil {
		t.Fatal(err)
	}
	analisis, cobertura := e.Actuaciones[1], e.Actuaciones[2]
	h := func(a Actuacion) HitoCircuitoRRHH {
		return HitoCircuitoRRHH{ActuacionClave: a.AccionClave, ActorRef: a.ActorRef,
			PerfilClave: "tecnico_rrhh", PerfilRef: "perfil:tecnico-rrhh:prueba",
			UnidadRef: a.UnidadRef, ReciboRef: a.ReciboRef, RegistradoEn: a.RealizadaEn}
	}
	peticion := h(analisis)
	peticion.Secuencia, peticion.VersionExpedienteEntrada = 1, 1
	peticion.Clave, peticion.Tipo = "peticion_firmada", HitoPeticionFirmada
	peticion.Origen, peticion.Destino = inicial, "autorizacion_rrhh"
	peticion.DocumentoRef, peticion.HuellaDocumentoSHA256 = "documento:peticion:prueba", strings.Repeat("a", 64)
	peticion.Firmas = []FirmaCircuitoRRHH{
		{CargoClave: "tecnico", FirmaRef: "firma:tecnico:prueba", FirmanteRef: "persona:tecnico:prueba", HuellaDocumentoSHA256: peticion.HuellaDocumentoSHA256},
		{CargoClave: "delegacion", FirmaRef: "firma:delegacion:prueba", FirmanteRef: "persona:delegacion:prueba", HuellaDocumentoSHA256: peticion.HuellaDocumentoSHA256},
	}
	autorizacion := h(analisis)
	autorizacion.Secuencia, autorizacion.VersionExpedienteEntrada = 2, 1
	autorizacion.Clave, autorizacion.Tipo = "autorizacion_rrhh", HitoAutorizacionRRHH
	autorizacion.Origen, autorizacion.Destino = "autorizacion_rrhh", "credito"
	autorizacion.DocumentoRef, autorizacion.HuellaDocumentoSHA256 = "documento:autorizacion:prueba", strings.Repeat("b", 64)
	autorizacion.ActoAutorizacionRef, autorizacion.AutorizanteRef = "acto:autorizacion:prueba", "persona:direccion:prueba"
	autorizacion.CargoAutorizanteClave = "direccion_rrhh"
	credito := h(cobertura)
	credito.Secuencia, credito.VersionExpedienteEntrada = 3, 2
	credito.Clave, credito.Tipo = "credito_comprobado", HitoCreditoComprobado
	credito.Origen, credito.Destino = "credito", "oferta"
	credito.CreditoRef = e.Analisis.ValidacionRC.ReciboRef
	credito.DocumentoRef = e.Analisis.ValidacionRC.DocumentoRef
	credito.HuellaDocumentoSHA256 = strings.Repeat("c", 64)
	e.Flujo = d.Flujo
	e.Circuito = &CircuitoAdministrativo{Definicion: d.Flujo, EstadoActual: "oferta", Hitos: []HitoCircuitoRRHH{peticion, autorizacion, credito}}
	if err := e.Validar(); err != nil {
		t.Fatalf("preimagen con crédito: %v", err)
	}
	return e, d
}

func TestVincularOfertaBolsaConservaCreditoYAnadeHitoSinElegirPersona(t *testing.T) {
	e, d := expedienteConCreditoParaOfertaPrueba(t)
	fecha := e.ActualizadoEn.Add(time.Minute)
	act := DatosActuacion{AccionClave: AccionVincularOfertaBolsa, ActorRef: "persona:rrhh:prueba",
		UnidadRef: e.Asignacion.UnidadRef, ReciboRef: "recibo:oferta-ct:prueba",
		RealizadaEn: fecha, FaseDestino: e.FaseActual, EstadoDestino: e.EstadoActual}
	siguiente, err := e.VincularOfertaBolsa(e.Version, d, "oferta:publicada:prueba", "perfil:tecnico-rrhh:prueba", act)
	if err != nil || siguiente.Validar() != nil {
		t.Fatalf("vincular oferta real: %v", err)
	}
	if siguiente.Version != e.Version+1 || len(siguiente.Circuito.Hitos) != 4 ||
		siguiente.Circuito.Hitos[3].OfertaRef != "oferta:publicada:prueba" ||
		siguiente.Circuito.Hitos[2].CreditoRef != e.Analisis.ValidacionRC.ReciboRef ||
		len(e.Circuito.Hitos) != 3 {
		t.Fatal("la oferta alteró el crédito o perdió la historia")
	}
	if _, err := siguiente.VincularOfertaBolsa(e.Version, d, "oferta:publicada:prueba", "perfil:tecnico-rrhh:prueba", act); err == nil {
		t.Fatal("una clave repetida no debe crear otra versión")
	}
}
