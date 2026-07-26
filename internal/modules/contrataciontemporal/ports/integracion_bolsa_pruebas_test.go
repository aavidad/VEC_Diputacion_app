package ports

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	autoridadPeticionBolsaPrueba  = "autoridad:contratacion-temporal"
	autoridadRespuestaBolsaPrueba = "autoridad:bolsa"
	clavePeticionBolsaV1Prueba    = dominioSelloPeticionBolsa + "/v1"
	claveRespuestaBolsaV1Prueba   = dominioSelloRespuestaBolsa + "/v1"
	claveRespuestaBolsaV2Prueba   = dominioSelloRespuestaBolsa + "/v2"
)

var (
	secretoPeticionBolsaV1Prueba  = []byte("clave-peticion-bolsa-v1-prueba")
	secretoRespuestaBolsaV1Prueba = []byte("clave-respuesta-bolsa-v1-prueba")
	secretoRespuestaBolsaV2Prueba = []byte("clave-respuesta-bolsa-v2-prueba")
)

type selladorHMACBolsaPrueba struct {
	claveRef string
	clave    []byte
	falla    bool
	cancelar context.CancelFunc
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
	sello := "hmac-sha256:" + s.claveRef + ":" +
		hex.EncodeToString(mac.Sum(nil))
	if s.cancelar != nil {
		s.cancelar()
	}
	return sello, nil
}

type verificadorHMACBolsaPrueba struct {
	claves   map[string][]byte
	falla    bool
	usos     atomic.Uint32
	cancelar context.CancelFunc
}

func (v *verificadorHMACBolsaPrueba) VerificarDatos(
	ctx context.Context,
	claveRef string,
	material []byte,
	sello string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	v.usos.Add(1)
	if v.falla {
		return ErrIntegracionBolsaNoDisponible
	}
	clave, existe := v.claves[claveRef]
	if !existe {
		return ErrEvidenciaBolsaNoAutenticada
	}
	mac := hmac.New(sha256.New, clave)
	_, _ = mac.Write(material)
	esperado := "hmac-sha256:" + claveRef + ":" +
		hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(esperado), []byte(sello)) {
		return ErrEvidenciaBolsaNoAutenticada
	}
	if v.cancelar != nil {
		v.cancelar()
	}
	return nil
}

func instanteBolsaPrueba() time.Time {
	return time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC)
}

func referenciaBolsaPrueba(referencia, caracter string) ReferenciaVersionadaIntegracionBolsa {
	return ReferenciaVersionadaIntegracionBolsa{
		Referencia: referencia, Version: 3, HuellaSHA256: strings.Repeat(caracter, 64),
	}
}

func selloNominalBolsaPrueba(claveRef, caracter string) string {
	return "hmac-sha256:" + claveRef + ":" + strings.Repeat(caracter, 64)
}

func seudonimoSeleccionBolsaPrueba(
	t *testing.T,
	caracter string,
) SeudonimoSeleccionBolsa {
	t.Helper()
	valor := "hmac-sha256:" + dominioSeudonimoSeleccionBolsa +
		"/v1:" + strings.Repeat(caracter, 64)
	seudonimo, err := NuevoSeudonimoSeleccionBolsa(valor)
	if err != nil {
		t.Fatalf("crear seudónimo de selección: %v", err)
	}
	return seudonimo
}

func selladorPeticionBolsaPrueba() *selladorHMACBolsaPrueba {
	return &selladorHMACBolsaPrueba{
		claveRef: clavePeticionBolsaV1Prueba,
		clave:    secretoPeticionBolsaV1Prueba,
	}
}

func selladorRespuestaBolsaPrueba() *selladorHMACBolsaPrueba {
	return &selladorHMACBolsaPrueba{
		claveRef: claveRespuestaBolsaV1Prueba,
		clave:    secretoRespuestaBolsaV1Prueba,
	}
}

func verificadorRespuestaBolsaPrueba(
	activa string,
	retenidas ...string,
) *VerificadorEvidenciaIntegracionBolsa {
	claves := map[string][]byte{
		claveRespuestaBolsaV1Prueba: secretoRespuestaBolsaV1Prueba,
		claveRespuestaBolsaV2Prueba: secretoRespuestaBolsaV2Prueba,
	}
	verificador, err := NuevoVerificadorEvidenciaIntegracionBolsa(
		autoridadRespuestaBolsaPrueba,
		activa,
		retenidas,
		&verificadorHMACBolsaPrueba{claves: claves},
	)
	if err != nil {
		panic(err)
	}
	return verificador
}

func datosContextoBolsaPrueba(
	instante time.Time,
	operacion string,
	recurso ReferenciaVersionadaIntegracionBolsa,
	accion ReferenciaVersionadaIntegracionBolsa,
) DatosContextoPeticionIntegracionBolsa {
	return DatosContextoPeticionIntegracionBolsa{
		OperacionRef:         operacion,
		OrganizacionRef:      "organizacion:diputacion",
		ExpedienteRef:        "expediente:temporal",
		VersionExpediente:    5,
		CorrelacionRef:       "correlacion:temporal",
		ContratoVersion:      VersionContratoIntegracionBolsa,
		AutoridadSolicitante: autoridadPeticionBolsaPrueba,
		Autorizacion:         referenciaBolsaPrueba("autorizacion:contratacion", "1"),
		Accion:               accion,
		Recurso:              recurso,
		Finalidad:            referenciaBolsaPrueba("finalidad:cobertura", "f"),
		SolicitadaEn:         instante,
		ValidaHasta:          instante.Add(10 * time.Minute),
	}
}

func emitirContextoBolsaPrueba(
	t *testing.T,
	datos DatosContextoPeticionIntegracionBolsa,
) ContextoPeticionIntegracionBolsa {
	t.Helper()
	emisor, err := NuevoEmisorContextoPeticionIntegracionBolsa(
		autoridadPeticionBolsaPrueba,
		clavePeticionBolsaV1Prueba,
		selladorPeticionBolsaPrueba(),
	)
	if err != nil {
		t.Fatalf("crear emisor de contexto: %v", err)
	}
	contexto, err := emisor.Emitir(context.Background(), datos, datos.SolicitadaEn)
	if err != nil {
		t.Fatalf("emitir contexto autenticado: %v", err)
	}
	return contexto
}

func contextoBolsaPrueba(
	t *testing.T,
	instante time.Time,
	operacion string,
	recurso ReferenciaVersionadaIntegracionBolsa,
	accion ReferenciaVersionadaIntegracionBolsa,
) ContextoPeticionIntegracionBolsa {
	t.Helper()
	return emitirContextoBolsaPrueba(
		t,
		datosContextoBolsaPrueba(instante, operacion, recurso, accion),
	)
}

func procedenciaBolsaPrueba(instante time.Time) ProcedenciaIntegracionBolsa {
	return ProcedenciaIntegracionBolsa{
		AutoridadRef:    autoridadRespuestaBolsaPrueba,
		RespuestaRef:    "respuesta:bolsa",
		ContratoVersion: VersionContratoIntegracionBolsa,
		Fuente:          referenciaBolsaPrueba("fuente:bolsa", "8"),
		Evidencia: EvidenciaNominalIntegracionBolsa{
			EvidenciaRef:         "evidencia:bolsa",
			ClaveVerificacionRef: claveRespuestaBolsaV1Prueba,
			SelloHMAC: selloNominalBolsaPrueba(
				claveRespuestaBolsaV1Prueba,
				"7",
			),
			EmitidaEn:    instante.Add(2 * time.Minute),
			ValidaHasta:  instante.Add(8 * time.Minute),
			RetenerHasta: instante.Add(30 * 24 * time.Hour),
		},
	}
}

func solicitudDisponibilidadPrueba(
	t *testing.T,
	instante time.Time,
) SolicitudDisponibilidadBolsa {
	t.Helper()
	necesidad := referenciaBolsaPrueba("necesidad:temporal", "a")
	return SolicitudDisponibilidadBolsa{
		Contexto: contextoBolsaPrueba(
			t,
			instante,
			"operacion:consultar:disponibilidad",
			necesidad,
			referenciaBolsaPrueba("accion:consultar:disponibilidad", "2"),
		),
		Necesidad: necesidad, CategoriaRef: "categoria:auxiliar", MaximoResultados: 100,
	}
}

func resultadoDisponibilidadPrueba(
	t *testing.T,
	instante time.Time,
) ResultadoDisponibilidadBolsa {
	t.Helper()
	solicitud := solicitudDisponibilidadPrueba(t, instante)
	contexto, _ := solicitud.Contexto.datosDurables()
	return ResultadoDisponibilidadBolsa{
		OperacionRef: contexto.OperacionRef, OrganizacionRef: contexto.OrganizacionRef,
		ExpedienteRef: contexto.ExpedienteRef, VersionExpediente: contexto.VersionExpediente,
		CorrelacionRef: contexto.CorrelacionRef, Necesidad: solicitud.Necesidad,
		CategoriaRef:    solicitud.CategoriaRef,
		Resultado:       referenciaBolsaPrueba("resultado:disponibilidad", "d"),
		BolsaEncontrada: true, Bolsa: referenciaBolsaPrueba("bolsa:vigente", "b"),
		Disponible: true, CantidadDisponible: 7, CantidadExacta: true,
		Procedencia: procedenciaBolsaPrueba(instante),
	}
}

func comandoOrdenPrueba(t *testing.T, instante time.Time) ComandoPrepararOrdenBolsa {
	t.Helper()
	bolsa := referenciaBolsaPrueba("bolsa:vigente", "b")
	return ComandoPrepararOrdenBolsa{
		Contexto: contextoBolsaPrueba(
			t,
			instante,
			"operacion:preparar:orden",
			bolsa,
			referenciaBolsaPrueba("accion:preparar:orden", "3"),
		),
		Necesidad:        referenciaBolsaPrueba("necesidad:temporal", "a"),
		Bolsa:            bolsa,
		Politica:         referenciaBolsaPrueba("politica:llamamiento", "c"),
		MaximoPosiciones: 200,
	}
}

func reciboOrdenPrueba(t *testing.T, instante time.Time) ReciboOrdenBolsa {
	t.Helper()
	comando := comandoOrdenPrueba(t, instante)
	contexto, _ := comando.Contexto.datosDurables()
	return ReciboOrdenBolsa{
		OperacionRef: contexto.OperacionRef, OrganizacionRef: contexto.OrganizacionRef,
		ExpedienteRef: contexto.ExpedienteRef, VersionExpediente: contexto.VersionExpediente,
		CorrelacionRef: contexto.CorrelacionRef, Necesidad: comando.Necesidad,
		Bolsa: comando.Bolsa, Politica: comando.Politica,
		Resultado:     referenciaBolsaPrueba("resultado:orden", "d"),
		OrdenGenerada: true, OrdenCompleta: true,
		Orden:             referenciaBolsaPrueba("orden:instantanea", "e"),
		AccionLlamamiento: referenciaBolsaPrueba("accion:solicitar:llamamiento", "4"),
		TotalPosiciones:   82, ReciboRef: "recibo:orden",
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
	comandoOrden := comandoOrdenPrueba(t, instante)
	reciboOrden := reciboOrdenPrueba(t, instante)
	firmarOrdenPrueba(t, sellador, comandoOrden, &reciboOrden)
	verificador := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)
	ahora := instante.Add(3 * time.Minute)
	comprobante, _, err := verificador.VerificarReciboOrden(
		context.Background(), comandoOrden, reciboOrden, ahora,
	)
	if err != nil {
		t.Fatalf("autenticar orden para comando: %v", err)
	}
	datos := datosContextoBolsaPrueba(
		instante,
		"operacion:solicitar:llamamiento",
		reciboOrden.Orden,
		reciboOrden.AccionLlamamiento,
	)
	contexto := emitirContextoBolsaPrueba(t, datos)
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
	contexto, _ := datos.Contexto.datosDurables()
	return ReciboSolicitudLlamamientoBolsa{
		OperacionRef: contexto.OperacionRef, OrganizacionRef: contexto.OrganizacionRef,
		ExpedienteRef: contexto.ExpedienteRef, VersionExpediente: contexto.VersionExpediente,
		CorrelacionRef: contexto.CorrelacionRef, Necesidad: datos.Necesidad,
		Bolsa: datos.Bolsa, Orden: datos.Orden, Politica: datos.Politica,
		Resultado:         referenciaBolsaPrueba("resultado:propuesta", "d"),
		PropuestaGenerada: true, Propuesta: referenciaBolsaPrueba("propuesta:primera", "e"),
		AccionEvento:       referenciaBolsaPrueba("accion:registrar:evento", "5"),
		LlamamientoRef:     "llamamiento:primero",
		SeleccionRef:       seudonimoSeleccionBolsaPrueba(t, "9"),
		RetencionSeleccion: referenciaBolsaPrueba("retencion:seleccion", "6"),
		OrdenSeleccionado:  4, ReciboRef: "recibo:llamamiento",
		AuditoriaRef: "auditoria:llamamiento", EventoRef: "evento:llamamiento",
		ConfirmadaEn: instante.Add(time.Minute), Procedencia: procedenciaBolsaPrueba(instante),
	}
}

func firmarRespuestaBolsaPrueba(
	t *testing.T,
	sellador *selladorHMACBolsaPrueba,
	tipoMaterial string,
	peticionRef string,
	materialPeticion []byte,
	materialRespuesta []byte,
	procedencia *ProcedenciaIntegracionBolsa,
) {
	t.Helper()
	procedencia.Evidencia.ClaveVerificacionRef = sellador.claveRef
	procedencia.Evidencia.SelloHMAC = selloNominalBolsaPrueba(sellador.claveRef, "1")
	evidencia := nuevaEvidenciaDurableBolsa(
		tipoMaterial,
		peticionRef,
		materialPeticion,
		materialRespuesta,
		*procedencia,
	)
	materialFirmado := materialAutenticacionRespuestaBolsa(evidencia, materialRespuesta)
	sello, err := sellador.SellarDatos(context.Background(), materialFirmado)
	if err != nil {
		t.Fatalf("sellar respuesta: %v", err)
	}
	procedencia.Evidencia.SelloHMAC = sello
}

func firmarDisponibilidadPrueba(
	t *testing.T,
	sellador *selladorHMACBolsaPrueba,
	solicitud SolicitudDisponibilidadBolsa,
	resultado *ResultadoDisponibilidadBolsa,
) {
	t.Helper()
	contexto, _ := solicitud.Contexto.datosDurables()
	firmarRespuestaBolsaPrueba(
		t,
		sellador,
		"disponibilidad_volatil",
		contexto.OperacionRef,
		materialSolicitudDisponibilidadBolsa(solicitud),
		materialDisponibilidadBolsa(solicitud, *resultado),
		&resultado.Procedencia,
	)
}

func firmarOrdenPrueba(
	t *testing.T,
	sellador *selladorHMACBolsaPrueba,
	comando ComandoPrepararOrdenBolsa,
	recibo *ReciboOrdenBolsa,
) {
	t.Helper()
	contexto, _ := comando.Contexto.datosDurables()
	firmarRespuestaBolsaPrueba(
		t,
		sellador,
		"recibo_orden",
		contexto.OperacionRef,
		materialComandoOrdenBolsa(comando),
		materialReciboOrdenBolsa(comando, *recibo),
		&recibo.Procedencia,
	)
}

func firmarLlamamientoPrueba(
	t *testing.T,
	sellador *selladorHMACBolsaPrueba,
	comando ComandoSolicitarLlamamientoBolsa,
	recibo *ReciboSolicitudLlamamientoBolsa,
) {
	t.Helper()
	datos, err := comando.datosCanonicos()
	if err != nil {
		t.Fatalf("leer comando para firma: %v", err)
	}
	contexto, _ := datos.Contexto.datosDurables()
	firmarRespuestaBolsaPrueba(
		t,
		sellador,
		"recibo_llamamiento",
		contexto.OperacionRef,
		materialComandoLlamamientoBolsa(comando),
		materialReciboLlamamientoBolsa(comando, *recibo),
		&recibo.Procedencia,
	)
}

func autenticarReciboLlamamientoPrueba(
	t *testing.T,
	comando ComandoSolicitarLlamamientoBolsa,
	recibo ReciboSolicitudLlamamientoBolsa,
	instante time.Time,
) ComprobanteEvidenciaIntegracionBolsa {
	t.Helper()
	comprobante, _, err := verificadorRespuestaBolsaPrueba(
		claveRespuestaBolsaV1Prueba,
	).VerificarReciboLlamamiento(context.Background(), comando, recibo, instante)
	if err != nil {
		t.Fatalf("autenticar recibo de llamamiento: %v", err)
	}
	return comprobante
}

func exigirErrorBolsa(t *testing.T, err error, esperado error) {
	t.Helper()
	if !errors.Is(err, esperado) {
		t.Fatalf("error=%v; esperado=%v", err, esperado)
	}
}
