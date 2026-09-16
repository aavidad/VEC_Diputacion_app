package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func intentoAutenticacionAdministracionPrueba() IntentoAutenticacionAdministracionV1 {
	return IntentoAutenticacionAdministracionV1{
		IntentoRef:     "hmac-sha256:intento-autenticacion-v1:" + strings.Repeat("b", 64),
		CorrelacionRef: "correlacion_0123456789abcdef0123456789abcdef",
		InstanteUTC:    time.Date(2026, time.September, 12, 10, 11, 12, 123000000, time.UTC),
		Superficie:     SuperficieAutenticacionAdministracionPrivilegiadaV1,
		Resultado:      ResultadoIntentoAutenticacionAdministracionPermitidoV1,
		Motivo:         MotivoIntentoAutenticacionAdministracionPermisoConcedidoV1,
		ActorEstado:    EstadoActorIntentoAutenticacionAcreditadoV1,
		HMACActor:      "hmac-sha256:actor-v1:" + strings.Repeat("a", 64),
	}
}

func TestIntentoAutenticacionAdministracionV1ValidaEstadosCerrados(t *testing.T) {
	casos := []struct {
		nombre  string
		aplicar func(*IntentoAutenticacionAdministracionV1)
		valido  bool
	}{
		{"permitido acreditado", func(*IntentoAutenticacionAdministracionV1) {}, true},
		{"sin certificado no acreditado", func(i *IntentoAutenticacionAdministracionV1) {
			i.Resultado, i.Motivo, i.ActorEstado, i.HMACActor = ResultadoIntentoAutenticacionAdministracionDenegadoV1, MotivoIntentoAutenticacionAdministracionCertificadoAusenteV1, EstadoActorIntentoAutenticacionNoAcreditadoV1, ""
		}, true},
		{"certificado no verificado no acreditado", func(i *IntentoAutenticacionAdministracionV1) {
			i.Resultado, i.Motivo, i.ActorEstado, i.HMACActor = ResultadoIntentoAutenticacionAdministracionDenegadoV1, MotivoIntentoAutenticacionAdministracionCertificadoNoVerificadoV1, EstadoActorIntentoAutenticacionNoAcreditadoV1, ""
		}, true},
		{"certificado revocado no acreditado", func(i *IntentoAutenticacionAdministracionV1) {
			i.Resultado, i.Motivo, i.ActorEstado, i.HMACActor = ResultadoIntentoAutenticacionAdministracionDenegadoV1, MotivoIntentoAutenticacionAdministracionCertificadoRevocadoV1, EstadoActorIntentoAutenticacionNoAcreditadoV1, ""
		}, true},
		{"canal no valido no acreditado", func(i *IntentoAutenticacionAdministracionV1) {
			i.Resultado, i.Motivo, i.ActorEstado, i.HMACActor = ResultadoIntentoAutenticacionAdministracionDenegadoV1, MotivoIntentoAutenticacionAdministracionCanalNoValidoV1, EstadoActorIntentoAutenticacionNoAcreditadoV1, ""
		}, true},
		{"sesion revocada acreditado", func(i *IntentoAutenticacionAdministracionV1) {
			i.Resultado, i.Motivo = ResultadoIntentoAutenticacionAdministracionDenegadoV1, MotivoIntentoAutenticacionAdministracionSesionRevocadaV1
		}, true},
		{"permiso denegado acreditado", func(i *IntentoAutenticacionAdministracionV1) {
			i.Resultado, i.Motivo = ResultadoIntentoAutenticacionAdministracionDenegadoV1, MotivoIntentoAutenticacionAdministracionPermisoDenegadoV1
		}, true},
		{"infraestructura sin actor", func(i *IntentoAutenticacionAdministracionV1) {
			i.Resultado, i.Motivo, i.ActorEstado, i.HMACActor = ResultadoIntentoAutenticacionAdministracionErrorV1, MotivoIntentoAutenticacionAdministracionErrorInfraestructuraV1, EstadoActorIntentoAutenticacionNoAcreditadoV1, ""
		}, true},
		{"certificado ausente acreditado", func(i *IntentoAutenticacionAdministracionV1) {
			i.Motivo = MotivoIntentoAutenticacionAdministracionCertificadoAusenteV1
		}, false},
		{"sesion revocada sin actor", func(i *IntentoAutenticacionAdministracionV1) {
			i.Resultado, i.Motivo, i.ActorEstado, i.HMACActor = ResultadoIntentoAutenticacionAdministracionDenegadoV1, MotivoIntentoAutenticacionAdministracionSesionRevocadaV1, EstadoActorIntentoAutenticacionNoAcreditadoV1, ""
		}, false},
		{"infraestructura con actor", func(i *IntentoAutenticacionAdministracionV1) {
			i.Resultado, i.Motivo = ResultadoIntentoAutenticacionAdministracionErrorV1, MotivoIntentoAutenticacionAdministracionErrorInfraestructuraV1
		}, true},
		{"hmac para no acreditado", func(i *IntentoAutenticacionAdministracionV1) {
			i.ActorEstado = EstadoActorIntentoAutenticacionNoAcreditadoV1
		}, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			intento := intentoAutenticacionAdministracionPrueba()
			caso.aplicar(&intento)
			err := intento.Validar()
			if caso.valido && err != nil {
				t.Fatalf("Validar() error = %v", err)
			}
			if !caso.valido && !errors.Is(err, ErrIntentoAutenticacionAdministracionInvalido) {
				t.Fatalf("Validar() error = %v; se esperaba %v", err, ErrIntentoAutenticacionAdministracionInvalido)
			}
		})
	}
}

func TestIntentoAutenticacionAdministracionV1RechazaReferenciasSuperficieEInstanteNoCanonicos(t *testing.T) {
	casos := []struct {
		nombre  string
		aplicar func(*IntentoAutenticacionAdministracionV1)
	}{
		{"referencia no opaca", func(i *IntentoAutenticacionAdministracionV1) { i.IntentoRef = "intento_operador" }},
		{"correlacion no opaca", func(i *IntentoAutenticacionAdministracionV1) { i.CorrelacionRef = "correlacion_OPERADOR" }},
		{"superficie distinta", func(i *IntentoAutenticacionAdministracionV1) { i.Superficie = "interna_corporativa" }},
		{"instante no utc", func(i *IntentoAutenticacionAdministracionV1) {
			i.InstanteUTC = i.InstanteUTC.In(time.FixedZone("prueba", 3600))
		}},
		{"instante no microsegundo", func(i *IntentoAutenticacionAdministracionV1) { i.InstanteUTC = i.InstanteUTC.Add(time.Nanosecond) }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			intento := intentoAutenticacionAdministracionPrueba()
			caso.aplicar(&intento)
			if err := intento.Validar(); !errors.Is(err, ErrIntentoAutenticacionAdministracionInvalido) {
				t.Fatalf("Validar() error = %v; se esperaba %v", err, ErrIntentoAutenticacionAdministracionInvalido)
			}
		})
	}
}
