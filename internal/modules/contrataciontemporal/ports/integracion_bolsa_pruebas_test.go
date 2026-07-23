package ports

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

type selladorHMACBolsaPrueba struct {
	clave []byte
	falla bool
}

func (s *selladorHMACBolsaPrueba) SellarDatos(
	ctx context.Context,
	material []byte,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if s.falla {
		return "", ErrIntegracionBolsaNoDisponible
	}
	mac := hmac.New(sha256.New, s.clave)
	_, _ = mac.Write(material)
	return "hmac-sha256:" + dominioSelloRespuestaBolsa + "/v1:" +
		hex.EncodeToString(mac.Sum(nil)), nil
}

func instanteBolsaPrueba() time.Time {
	return time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC)
}

func referenciaBolsaPrueba(referencia, caracter string) ReferenciaVersionadaIntegracionBolsa {
	return ReferenciaVersionadaIntegracionBolsa{
		Referencia: referencia, Version: 3, HuellaSHA256: strings.Repeat(caracter, 64),
	}
}

func selloNominalBolsaPrueba(dominio, caracter string) string {
	return "hmac-sha256:" + dominio + "/v1:" + strings.Repeat(caracter, 64)
}

func contextoBolsaPrueba(instante time.Time) ContextoPeticionIntegracionBolsa {
	return ContextoPeticionIntegracionBolsa{
		OperacionRef: "operacion:bolsa:temporal", OrganizacionRef: "organizacion:diputacion",
		ExpedienteRef: "expediente:temporal", VersionExpediente: 5,
		CorrelacionRef: "correlacion:temporal", ContratoVersion: VersionContratoIntegracionBolsa,
		Finalidad:         referenciaBolsaPrueba("finalidad:cobertura", "f"),
		SelloPeticionHMAC: selloNominalBolsaPrueba(dominioSelloPeticionBolsa, "9"),
		SolicitadaEn:      instante, ValidaHasta: instante.Add(10 * time.Minute),
	}
}

func procedenciaBolsaPrueba(instante time.Time) ProcedenciaIntegracionBolsa {
	return ProcedenciaIntegracionBolsa{
		AutoridadRef: "autoridad:bolsa", RespuestaRef: "respuesta:bolsa",
		ContratoVersion: VersionContratoIntegracionBolsa,
		Fuente:          referenciaBolsaPrueba("fuente:bolsa", "8"),
		Evidencia: EvidenciaNominalIntegracionBolsa{
			EvidenciaRef: "evidencia:bolsa",
			SelloHMAC:    selloNominalBolsaPrueba(dominioSelloRespuestaBolsa, "7"),
			EmitidaEn:    instante.Add(2 * time.Minute),
			ValidaHasta:  instante.Add(8 * time.Minute),
		},
	}
}

func solicitudDisponibilidadPrueba(instante time.Time) SolicitudDisponibilidadBolsa {
	return SolicitudDisponibilidadBolsa{
		Contexto:     contextoBolsaPrueba(instante),
		Necesidad:    referenciaBolsaPrueba("necesidad:temporal", "a"),
		CategoriaRef: "categoria:auxiliar", MaximoResultados: 100,
	}
}

func resultadoDisponibilidadPrueba(instante time.Time) ResultadoDisponibilidadBolsa {
	solicitud := solicitudDisponibilidadPrueba(instante)
	return ResultadoDisponibilidadBolsa{
		OperacionRef: solicitud.Contexto.OperacionRef, OrganizacionRef: solicitud.Contexto.OrganizacionRef,
		ExpedienteRef: solicitud.Contexto.ExpedienteRef, VersionExpediente: solicitud.Contexto.VersionExpediente,
		CorrelacionRef: solicitud.Contexto.CorrelacionRef, Necesidad: solicitud.Necesidad,
		CategoriaRef:    solicitud.CategoriaRef,
		Resultado:       referenciaBolsaPrueba("resultado:disponibilidad", "d"),
		BolsaEncontrada: true, Bolsa: referenciaBolsaPrueba("bolsa:vigente", "b"),
		Disponible: true, CantidadDisponible: 7, CantidadExacta: true,
		Procedencia: procedenciaBolsaPrueba(instante),
	}
}

func comandoOrdenPrueba(instante time.Time) ComandoPrepararOrdenBolsa {
	return ComandoPrepararOrdenBolsa{
		Contexto:         contextoBolsaPrueba(instante),
		Necesidad:        referenciaBolsaPrueba("necesidad:temporal", "a"),
		Bolsa:            referenciaBolsaPrueba("bolsa:vigente", "b"),
		Politica:         referenciaBolsaPrueba("politica:llamamiento", "c"),
		MaximoPosiciones: 200,
	}
}

func reciboOrdenPrueba(instante time.Time) ReciboOrdenBolsa {
	comando := comandoOrdenPrueba(instante)
	return ReciboOrdenBolsa{
		OperacionRef: comando.Contexto.OperacionRef, OrganizacionRef: comando.Contexto.OrganizacionRef,
		ExpedienteRef: comando.Contexto.ExpedienteRef, VersionExpediente: comando.Contexto.VersionExpediente,
		CorrelacionRef: comando.Contexto.CorrelacionRef, Necesidad: comando.Necesidad,
		Bolsa: comando.Bolsa, Politica: comando.Politica,
		Resultado:     referenciaBolsaPrueba("resultado:orden", "d"),
		OrdenGenerada: true, OrdenCompleta: true,
		Orden:           referenciaBolsaPrueba("orden:instantanea", "e"),
		TotalPosiciones: 82, ReciboRef: "recibo:orden",
		AuditoriaRef: "auditoria:orden", EventoRef: "evento:orden",
		ConfirmadaEn: instante.Add(time.Minute), Procedencia: procedenciaBolsaPrueba(instante),
	}
}

func comandoLlamamientoPrueba(
	t *testing.T,
	instante time.Time,
	sellador *selladorHMACBolsaPrueba,
) ComandoSolicitarLlamamientoBolsa {
	t.Helper()
	comandoOrden := comandoOrdenPrueba(instante)
	reciboOrden := reciboOrdenPrueba(instante)
	firmarOrdenPrueba(t, sellador, comandoOrden, &reciboOrden)
	verificador, err := NuevoVerificadorEvidenciaIntegracionBolsa(sellador)
	if err != nil {
		t.Fatalf("crear verificador para comando: %v", err)
	}
	ahora := instante.Add(3 * time.Minute)
	comprobante, err := verificador.VerificarReciboOrden(
		context.Background(), comandoOrden, reciboOrden, ahora,
	)
	if err != nil {
		t.Fatalf("autenticar orden para comando: %v", err)
	}
	contexto := contextoBolsaPrueba(instante)
	contexto.OperacionRef = "operacion:solicitar:llamamiento"
	comando, err := NuevoComandoSolicitarLlamamientoBolsa(
		PreparacionComandoSolicitarLlamamientoBolsa{
			Contexto: contexto, ComandoOrden: comandoOrden, ReciboOrden: reciboOrden,
			ComprobanteOrden: comprobante, MaximaPosicionEvaluable: 50,
		},
		ahora,
	)
	if err != nil {
		t.Fatalf("crear comando de llamamiento: %v", err)
	}
	return comando
}

func reciboLlamamientoPrueba(
	t *testing.T,
	comando ComandoSolicitarLlamamientoBolsa,
	instante time.Time,
) ReciboSolicitudLlamamientoBolsa {
	t.Helper()
	datos, err := comando.DatosEn(instante.Add(3 * time.Minute))
	if err != nil {
		t.Fatalf("extraer comando de llamamiento: %v", err)
	}
	return ReciboSolicitudLlamamientoBolsa{
		OperacionRef: datos.Contexto.OperacionRef, OrganizacionRef: datos.Contexto.OrganizacionRef,
		ExpedienteRef: datos.Contexto.ExpedienteRef, VersionExpediente: datos.Contexto.VersionExpediente,
		CorrelacionRef: datos.Contexto.CorrelacionRef, Necesidad: datos.Necesidad,
		Bolsa: datos.Bolsa, Orden: datos.Orden, Politica: datos.Politica,
		Resultado:         referenciaBolsaPrueba("resultado:propuesta", "d"),
		PropuestaGenerada: true, Propuesta: referenciaBolsaPrueba("propuesta:primera", "e"),
		LlamamientoRef: "llamamiento:primero", SeleccionRef: "seleccion:seudonimizada",
		RetencionSeleccion: referenciaBolsaPrueba("retencion:seleccion", "6"),
		OrdenSeleccionado:  4, ReciboRef: "recibo:llamamiento",
		AuditoriaRef: "auditoria:llamamiento", EventoRef: "evento:llamamiento",
		ConfirmadaEn: instante.Add(time.Minute), Procedencia: procedenciaBolsaPrueba(instante),
	}
}

func firmarDisponibilidadPrueba(
	t *testing.T,
	sellador *selladorHMACBolsaPrueba,
	solicitud SolicitudDisponibilidadBolsa,
	resultado *ResultadoDisponibilidadBolsa,
) {
	t.Helper()
	sello, err := sellador.SellarDatos(context.Background(), materialDisponibilidadBolsa(solicitud, *resultado))
	if err != nil {
		t.Fatalf("sellar disponibilidad: %v", err)
	}
	resultado.Procedencia.Evidencia.SelloHMAC = sello
}

func firmarOrdenPrueba(
	t *testing.T,
	sellador *selladorHMACBolsaPrueba,
	comando ComandoPrepararOrdenBolsa,
	recibo *ReciboOrdenBolsa,
) {
	t.Helper()
	sello, err := sellador.SellarDatos(context.Background(), materialReciboOrdenBolsa(comando, *recibo))
	if err != nil {
		t.Fatalf("sellar orden: %v", err)
	}
	recibo.Procedencia.Evidencia.SelloHMAC = sello
}

func firmarLlamamientoPrueba(
	t *testing.T,
	sellador *selladorHMACBolsaPrueba,
	comando ComandoSolicitarLlamamientoBolsa,
	recibo *ReciboSolicitudLlamamientoBolsa,
) {
	t.Helper()
	sello, err := sellador.SellarDatos(context.Background(), materialReciboLlamamientoBolsa(comando, *recibo))
	if err != nil {
		t.Fatalf("sellar llamamiento: %v", err)
	}
	recibo.Procedencia.Evidencia.SelloHMAC = sello
}
