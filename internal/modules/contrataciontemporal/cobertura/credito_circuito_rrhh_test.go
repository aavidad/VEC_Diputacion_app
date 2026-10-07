package cobertura

import (
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDecisionCoberturaAdjuntaCreditoEnSuMismaVersion(t *testing.T) {
	anterior, siguiente, definicion := expedienteConCreditoCircuitoPrueba(t)
	credito, err := anterior.DatosCreditoCircuitoRRHH()
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudEvidenciaCreditoCircuitoRRHH(
		credito, siguiente.Actuaciones[len(siguiente.Actuaciones)-1].ActorRef, "perfil:rrhh:prueba",
	)
	if err != nil {
		t.Fatal(err)
	}
	evidencia := ports.EvidenciaCreditoCircuitoRRHH{
		Solicitud: solicitud, Definicion: definicion, DocumentoVersion: 3,
		HuellaDocumentoSHA256: strings.Repeat("c", 64),
	}
	if _, err := adjuntarCreditoCircuitoDecisionCobertura(anterior, siguiente, "perfil:rrhh:prueba", nil); err == nil {
		t.Fatal("el circuito avanzó sin fuente documental")
	}
	conCredito, err := adjuntarCreditoCircuitoDecisionCobertura(anterior, siguiente, "perfil:rrhh:prueba", &evidencia)
	if err != nil || conCredito.Validar() != nil {
		t.Fatalf("hito de crédito acreditado: %v", err)
	}
	if conCredito.Version != siguiente.Version || len(conCredito.Circuito.Hitos) != 3 {
		t.Fatal("el hito creó una versión adicional o perdió historia")
	}
	hito := conCredito.Circuito.Hitos[2]
	ultima := siguiente.Actuaciones[len(siguiente.Actuaciones)-1]
	if hito.CreditoRef != anterior.Analisis.ValidacionRC.ReciboRef ||
		hito.DocumentoRef != anterior.Analisis.ValidacionRC.DocumentoRef ||
		hito.DocumentoVersion != 3 || hito.HuellaDocumentoSHA256 != strings.Repeat("c", 64) ||
		hito.ReciboRef != ultima.ReciboRef || hito.VersionExpedienteEntrada != anterior.Version {
		t.Fatal("el hito no quedó ligado al crédito, documento y recibo originales")
	}
	alterada := evidencia.Clonar()
	alterada.Solicitud.DocumentoRef = "documento:otro:credito"
	if _, err := adjuntarCreditoCircuitoDecisionCobertura(anterior, siguiente, "perfil:rrhh:prueba", &alterada); err == nil {
		t.Fatal("una huella de otro documento avanzó")
	}
}

func expedienteConCreditoCircuitoPrueba(t *testing.T) (domain.Expediente, domain.Expediente, domain.DefinicionCircuitoRRHH) {
	t.Helper()
	base := expedienteConAnalisisDurableO3Prueba(t, domain.RCNoRequerida)
	fecha := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	importe := domain.Importe{Centimos: 4_000_000, Moneda: "EUR"}
	base.Analisis.ValidacionRC.Resultado = domain.RCValidada
	base.Analisis.ValidacionRC.Motivo = ""
	base.Analisis.ValidacionRC.FechaRC = &fecha
	base.Analisis.ValidacionRC.Numero = "rc:prueba:credito"
	base.Analisis.ValidacionRC.Importe = &importe
	base.Analisis.ValidacionRC.DocumentoRef = "documento:prueba:credito"
	definicion, err := domain.NuevaDefinicionCircuitoRRHH("flujo:ct:rrhh:credito-prueba", 2, "solicitud", []domain.TransicionCircuitoRRHH{
		{Clave: "peticion_firmada", Tipo: domain.HitoPeticionFirmada, Origen: "solicitud", Destino: "autorizacion_rrhh", RequiereDocumento: true, RequiereFirma: true, PerfilClave: "tecnico_rrhh", FirmasRequeridas: []domain.ClaveCatalogo{"tecnico", "delegacion"}},
		{Clave: "autorizacion_rrhh", Tipo: domain.HitoAutorizacionRRHH, Origen: "autorizacion_rrhh", Destino: "credito", RequiereDocumento: true, PerfilClave: "tecnico_rrhh", AutorizanteCargoClave: "direccion_rrhh"},
		{Clave: "credito_comprobado", Tipo: domain.HitoCreditoComprobado, Origen: "credito", Destino: "oferta", RequiereDocumento: true, PerfilClave: "tecnico_rrhh"},
	})
	if err != nil {
		t.Fatal(err)
	}
	base.Flujo = definicion.Flujo
	circuito, err := domain.NuevoCircuitoAdministrativo(definicion)
	if err != nil {
		t.Fatal(err)
	}
	base.Circuito = &circuito
	if err := base.Validar(); err != nil {
		t.Fatal(err)
	}
	actAnalisis := base.Actuaciones[len(base.Actuaciones)-1]
	comun := domain.HitoCircuitoRRHH{
		ActuacionClave: actAnalisis.AccionClave, ActorRef: actAnalisis.ActorRef,
		PerfilClave: "tecnico_rrhh", PerfilRef: "perfil:rrhh:prueba", UnidadRef: actAnalisis.UnidadRef,
		ReciboRef: actAnalisis.ReciboRef, RegistradoEn: actAnalisis.RealizadaEn,
	}
	peticion := comun
	peticion.Clave = "peticion_firmada"
	peticion.DocumentoRef = "documento:peticion:prueba"
	peticion.HuellaDocumentoSHA256 = strings.Repeat("a", 64)
	peticion.Firmas = []domain.FirmaCircuitoRRHH{
		{CargoClave: "tecnico", FirmaRef: "firma:tecnico:prueba", FirmanteRef: "actor:tecnico:prueba", HuellaDocumentoSHA256: peticion.HuellaDocumentoSHA256},
		{CargoClave: "delegacion", FirmaRef: "firma:delegacion:prueba", FirmanteRef: "actor:delegacion:prueba", HuellaDocumentoSHA256: peticion.HuellaDocumentoSHA256},
	}
	autorizacion := comun
	autorizacion.Clave = "autorizacion_rrhh"
	autorizacion.DocumentoRef = "documento:autorizacion:prueba"
	autorizacion.HuellaDocumentoSHA256 = strings.Repeat("b", 64)
	autorizacion.ActoAutorizacionRef = "acto:autorizacion:prueba"
	autorizacion.AutorizanteRef = "actor:direccion:prueba"
	autorizacion.CargoAutorizanteClave = "direccion_rrhh"
	base, err = base.AdjuntarHitosCircuito(definicion, base.Version, []domain.HitoCircuitoRRHH{peticion, autorizacion})
	if err != nil {
		t.Fatalf("adjuntar antecedentes: %v", err)
	}
	via := domain.DecisionViaCobertura{
		ViaClave: "bolsa_vigente", ProcedimientoRef: "procedimiento:bolsa:prueba",
		Comprobaciones: []domain.ComprobacionCobertura{{
			Clave: "bolsa_disponible", Resultado: domain.ComprobacionAfirmativa,
			FuenteRef: "fuente:bolsa:prueba", ReciboRef: "recibo:bolsa:prueba",
			EvaluadaEn: actAnalisis.RealizadaEn,
		}},
		Motivacion: "Bolsa disponible en la prueba.",
	}
	siguiente, err := base.RegistrarViaCobertura(base.Version, via, domain.DatosActuacion{
		AccionClave: "cobertura.decidida", ActorRef: "actor:rrhh:prueba",
		UnidadRef: "unidad:rrhh:prueba", ReciboRef: "recibo:cobertura:prueba",
		RealizadaEn: actAnalisis.RealizadaEn.Add(time.Minute), FaseDestino: "asignacion_unidad",
		EstadoDestino: domain.EstadoEnCurso,
	})
	if err != nil {
		t.Fatalf("registrar vía: %v", err)
	}
	return base, siguiente, definicion
}
