package ports

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestSolicitudFirmaAtestacionV2ConservaMensajeYCompromisos(t *testing.T) {
	decision := decisionFirmaAtestacionV2PuertoPrueba(t)
	referenciaMotivo := referenciaMotivoAutorizacionPuertoV2Prueba(
		claveMotivoAutorizacionPuertoV2Prueba,
	)
	cabecera := cabeceraFirmaAtestacionV2Prueba()
	solicitud, err := NuevaSolicitudFirmaAtestacionAutorizacionV2(
		cabecera,
		decision,
		referenciaMotivo,
	)
	if err != nil {
		t.Fatalf("crear solicitud V2: %v", err)
	}
	mensaje, err := solicitud.Mensaje()
	if err != nil {
		t.Fatal(err)
	}
	esperado, err := domain.SerializarMensajeAtestacionAutorizacionV2(
		cabecera,
		decision,
		referenciaMotivo,
	)
	if err != nil || !bytes.Equal(mensaje, esperado) {
		t.Fatalf("mensaje distinto del contrato V2: err=%v", err)
	}
	mensaje[0] ^= 0xff
	recuperado, err := solicitud.Mensaje()
	if err != nil || !bytes.Equal(recuperado, esperado) {
		t.Fatalf("el llamador modifico la solicitud: err=%v", err)
	}

	referenciaDecision, err := solicitud.ReferenciaDecision()
	huellaSolicitud, errSolicitud := solicitud.HuellaSolicitudLigadaSHA256()
	huellaMotivo, errMotivo := solicitud.HuellaMotivoCatalogoSHA256()
	if err != nil || errSolicitud != nil || errMotivo != nil ||
		referenciaDecision != decision.DecisionRef ||
		huellaSolicitud != decision.SolicitudHuellaSHA256 ||
		huellaMotivo != decision.MotivoHuellaSHA256 {
		t.Fatalf(
			"compromisos V2 no ligados: ref=%q solicitud=%q motivo=%q errores=%v/%v/%v",
			referenciaDecision,
			huellaSolicitud,
			huellaMotivo,
			err,
			errSolicitud,
			errMotivo,
		)
	}
}

func TestSolicitudFirmaAtestacionV2RechazaCrucesYManipulaciones(t *testing.T) {
	decision := decisionFirmaAtestacionV2PuertoPrueba(t)
	referenciaMotivo := referenciaMotivoAutorizacionPuertoV2Prueba(
		claveMotivoAutorizacionPuertoV2Prueba,
	)
	solicitud, err := NuevaSolicitudFirmaAtestacionAutorizacionV2(
		cabeceraFirmaAtestacionV2Prueba(),
		decision,
		referenciaMotivo,
	)
	if err != nil {
		t.Fatal(err)
	}
	candidatas := []struct {
		nombre string
		mutar  func(*SolicitudFirmaAtestacionAutorizacionV2)
	}{
		{"cabecera", func(s *SolicitudFirmaAtestacionAutorizacionV2) { s.cabecera.ClaveID = "clave:otra" }},
		{"mensaje", func(s *SolicitudFirmaAtestacionAutorizacionV2) { s.mensaje[0] ^= 1 }},
		{"huella_mensaje", func(s *SolicitudFirmaAtestacionAutorizacionV2) { s.huellaMensaje = strings.Repeat("0", 64) }},
		{"decision", func(s *SolicitudFirmaAtestacionAutorizacionV2) { s.referenciaDecision += ":otra" }},
		{"solicitud", func(s *SolicitudFirmaAtestacionAutorizacionV2) { s.huellaSolicitudLigada = strings.Repeat("1", 64) }},
		{"motivo", func(s *SolicitudFirmaAtestacionAutorizacionV2) { s.huellaMotivoCatalogo = strings.Repeat("2", 64) }},
	}
	for _, caso := range candidatas {
		t.Run(caso.nombre, func(t *testing.T) {
			candidata := solicitud
			candidata.mensaje = append([]byte(nil), solicitud.mensaje...)
			caso.mutar(&candidata)
			if candidata.Validar() == nil {
				t.Fatal("solicitud V2 manipulada aceptada")
			}
		})
	}

	referenciaAjena := referenciaMotivo
	referenciaAjena.EntradaClave = "motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := NuevaSolicitudFirmaAtestacionAutorizacionV2(
		cabeceraFirmaAtestacionV2Prueba(),
		decision,
		referenciaAjena,
	); !errors.Is(err, ErrSolicitudFirmaAtestacionInvalida) {
		t.Fatalf("motivo ajeno aceptado: %v", err)
	}
	if _, err := NuevaSolicitudFirmaAtestacionAutorizacionV2(
		domain.CabeceraAtestacionAutorizacionV2{},
		decision,
		referenciaMotivo,
	); !errors.Is(err, ErrSolicitudFirmaAtestacionInvalida) {
		t.Fatalf("cabecera cero aceptada: %v", err)
	}
}

func TestResultadoYAtestacionV2QuedanLigadosAUnaSolicitud(t *testing.T) {
	decision := decisionFirmaAtestacionV2PuertoPrueba(t)
	referenciaMotivo := referenciaMotivoAutorizacionPuertoV2Prueba(
		claveMotivoAutorizacionPuertoV2Prueba,
	)
	primera, err := NuevaSolicitudFirmaAtestacionAutorizacionV2(
		cabeceraFirmaAtestacionV2Prueba(),
		decision,
		referenciaMotivo,
	)
	if err != nil {
		t.Fatal(err)
	}
	decision.DecisionRef += ":otra"
	segunda, err := NuevaSolicitudFirmaAtestacionAutorizacionV2(
		cabeceraFirmaAtestacionV2Prueba(),
		decision,
		referenciaMotivo,
	)
	if err != nil {
		t.Fatal(err)
	}

	firma := []byte{1, 2, 3, 4}
	resultado, err := NuevoResultadoFirmaAtestacionAutorizacionV2(
		primera,
		firma,
		"evidencia:firma:v2:1",
		time.Date(2026, 7, 17, 8, 0, 0, 123_000, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	firma[0] = 9
	recuperada, _ := resultado.Firma()
	if recuperada[0] != 1 || resultado.ValidarPara(segunda) == nil {
		t.Fatal("el resultado compartio memoria o acepto otra solicitud")
	}

	atestacion, err := NuevaAtestacionAutorizacionV2(primera, resultado)
	if err != nil || atestacion.ValidarPara(primera) != nil || atestacion.ValidarPara(segunda) == nil {
		t.Fatalf("ligadura de atestacion V2 incorrecta: %v", err)
	}
	solicitudRecuperada, _ := atestacion.Solicitud()
	solicitudRecuperada.mensaje[0] ^= 1
	if atestacion.Validar() != nil {
		t.Fatal("la solicitud devuelta compartio memoria con la atestacion")
	}
	resultadoRecuperado, _ := atestacion.Resultado()
	resultadoRecuperado.firma[0] ^= 1
	if atestacion.Validar() != nil {
		t.Fatal("el resultado devuelto compartio memoria con la atestacion")
	}
}

func TestResultadoFirmaAtestacionV2RechazaFormaInvalida(t *testing.T) {
	decision := decisionFirmaAtestacionV2PuertoPrueba(t)
	solicitud, err := NuevaSolicitudFirmaAtestacionAutorizacionV2(
		cabeceraFirmaAtestacionV2Prueba(),
		decision,
		referenciaMotivoAutorizacionPuertoV2Prueba(claveMotivoAutorizacionPuertoV2Prueba),
	)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre    string
		firma     []byte
		evidencia string
		instante  time.Time
	}{
		{"firma_vacia", nil, "evidencia:firma:v2", time.Now().UTC().Truncate(time.Microsecond)},
		{"evidencia_vacia", []byte{1}, "", time.Now().UTC().Truncate(time.Microsecond)},
		{"instante_no_canonico", []byte{1}, "evidencia:firma:v2", time.Now().UTC().Truncate(time.Microsecond).Add(time.Nanosecond)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevoResultadoFirmaAtestacionAutorizacionV2(
				solicitud,
				caso.firma,
				caso.evidencia,
				caso.instante,
			); !errors.Is(err, ErrResultadoFirmaAtestacionInvalido) {
				t.Fatalf("resultado invalido aceptado: %v", err)
			}
		})
	}
}

func TestFirmanteAtestacionesAutorizacionV2ConservaContrato(t *testing.T) {
	var _ FirmanteAtestacionesAutorizacionV2 = firmanteAtestacionV2ContratoPrueba{}
}

func TestAtestacionAutorizacionV2BloqueaCodecsYRedactaFormato(t *testing.T) {
	decision := decisionFirmaAtestacionV2PuertoPrueba(t)
	solicitud, err := NuevaSolicitudFirmaAtestacionAutorizacionV2(
		cabeceraFirmaAtestacionV2Prueba(),
		decision,
		referenciaMotivoAutorizacionPuertoV2Prueba(claveMotivoAutorizacionPuertoV2Prueba),
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := NuevoResultadoFirmaAtestacionAutorizacionV2(
		solicitud,
		[]byte("firma-privada-no-registrar"),
		"evidencia:privada:no-registrar",
		time.Date(2026, 7, 17, 8, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	atestacion, err := NuevaAtestacionAutorizacionV2(solicitud, resultado)
	if err != nil {
		t.Fatal(err)
	}

	for _, valor := range []any{solicitud, resultado, atestacion} {
		if _, err := json.Marshal(valor); !errors.Is(
			err,
			ErrSerializacionAtestacionAutorizacionV2Prohibida,
		) {
			t.Fatalf("JSON no bloqueado para %T: %v", valor, err)
		}
		if _, err := xml.Marshal(valor); !errors.Is(
			err,
			ErrSerializacionAtestacionAutorizacionV2Prohibida,
		) {
			t.Fatalf("XML no bloqueado para %T: %v", valor, err)
		}
		texto := fmt.Sprintf("%v|%+v|%#v", valor, valor, valor)
		if strings.Contains(texto, decision.DecisionRef) ||
			strings.Contains(texto, "firma-privada") ||
			strings.Contains(texto, "evidencia:privada") ||
			!strings.Contains(texto, "REDACTADA") {
			t.Fatalf("formato no redactado para %T: %s", valor, texto)
		}
		registro := slog.AnyValue(valor).Resolve().String()
		if strings.Contains(registro, decision.DecisionRef) || !strings.Contains(registro, "REDACTADA") {
			t.Fatalf("slog no redactado para %T: %s", valor, registro)
		}
	}
}

type firmanteAtestacionV2ContratoPrueba struct{}

func (firmanteAtestacionV2ContratoPrueba) FirmarAtestacionAutorizacionV2(
	context.Context,
	SolicitudFirmaAtestacionAutorizacionV2,
) (ResultadoFirmaAtestacionAutorizacionV2, error) {
	return ResultadoFirmaAtestacionAutorizacionV2{}, nil
}

func cabeceraFirmaAtestacionV2Prueba() domain.CabeceraAtestacionAutorizacionV2 {
	return domain.CabeceraAtestacionAutorizacionV2{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV2,
		Suite:          "VEC-AD-PRUEBA-2",
		ClaveID:        "clave:prueba:2026-07",
		Audiencia:      "vec/pruebas/autorizacion-v2",
	}
}

func decisionFirmaAtestacionV2PuertoPrueba(t *testing.T) domain.DecisionAutorizacion {
	t.Helper()
	decision, _ := decisionAutorizacionSolicitudLigadaV2Prueba(t)
	sort.Strings(decision.PoliticasEvaluadasRefs)
	sort.Strings(decision.PoliticasRefs)
	sort.Strings(decision.CamposPermitidos)
	sort.Strings(decision.Obligaciones)
	if err := decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2(); err != nil {
		t.Fatalf("decision canonica V2 de prueba invalida: %v", err)
	}
	return decision
}
