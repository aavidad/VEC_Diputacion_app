package ports

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestAtestacionAutorizacionV3EsNominalLigadaYDefensiva(t *testing.T) {
	e := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	cabecera := cabeceraAtestacionAutorizacionV3PuertosPrueba()
	solicitud, err := NuevaSolicitudFirmaAtestacionAutorizacionV3(
		cabecera,
		e.decision,
		e.motivo,
		e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	mensaje, err := solicitud.Mensaje()
	if err != nil {
		t.Fatal(err)
	}
	mensaje[0] ^= 1
	segunda, _ := solicitud.Mensaje()
	if bytes.Equal(mensaje, segunda) {
		t.Fatal("Mensaje expuso el slice interno")
	}
	huella, _ := solicitud.HuellaMensajeSHA256()
	resultado, err := NuevoResultadoFirmaAtestacionAutorizacionV3(
		solicitud,
		[]byte("firma-opaca-vec-ad-3"),
		"evidencia_firma_0123456789abcdef",
		e.ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	atestacion, err := NuevaAtestacionAutorizacionV3(solicitud, resultado)
	if err != nil || atestacion.ValidarPara(solicitud) != nil {
		t.Fatalf("atestacion V3 invalida: %v", err)
	}
	copiaResultado, _ := atestacion.Resultado()
	firma, _ := copiaResultado.Firma()
	firma[0] ^= 1
	otraFirma, _ := copiaResultado.Firma()
	if bytes.Equal(firma, otraFirma) {
		t.Fatal("Resultado expuso la firma interna")
	}
	if obtenida, _ := resultado.HuellaMensajeSHA256(); obtenida != huella {
		t.Fatalf("huella de resultado = %q; esperada %q", obtenida, huella)
	}
	if referencia, _ := solicitud.ReferenciaContextoActor(); referencia != e.resultado.RegistroContextoRef {
		t.Fatalf("referencia de contexto = %q", referencia)
	}
	if contexto, _ := solicitud.HuellaContextoActorSHA256(); contexto != e.resultado.HuellaSHA256 {
		t.Fatalf("huella de contexto = %q", contexto)
	}
}

func TestAtestacionAutorizacionV3RechazaCrucesYAdulteraciones(t *testing.T) {
	e := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	cabecera := cabeceraAtestacionAutorizacionV3PuertosPrueba()
	solicitud, err := NuevaSolicitudFirmaAtestacionAutorizacionV3(
		cabecera,
		e.decision,
		e.motivo,
		e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	adulteraciones := []struct {
		nombre string
		mutar  func(*SolicitudFirmaAtestacionAutorizacionV3)
	}{
		{"mensaje", func(s *SolicitudFirmaAtestacionAutorizacionV3) { s.mensaje[0] ^= 1 }},
		{"huella mensaje", func(s *SolicitudFirmaAtestacionAutorizacionV3) { s.huellaMensaje = strings.Repeat("a", 64) }},
		{"decision", func(s *SolicitudFirmaAtestacionAutorizacionV3) {
			s.referenciaDecision = "dec_otra234567890abcdef0123456789ab"
		}},
		{"huella decision", func(s *SolicitudFirmaAtestacionAutorizacionV3) { s.huellaDecision = strings.Repeat("a", 64) }},
		{"huella motivo", func(s *SolicitudFirmaAtestacionAutorizacionV3) { s.huellaMotivoCatalogo = strings.Repeat("a", 64) }},
		{"contexto", func(s *SolicitudFirmaAtestacionAutorizacionV3) {
			s.referenciaContexto = "rca_otra234567890abcdefghijklmn"
		}},
		{"huella contexto", func(s *SolicitudFirmaAtestacionAutorizacionV3) { s.huellaContexto = strings.Repeat("a", 64) }},
	}
	for _, caso := range adulteraciones {
		t.Run(caso.nombre, func(t *testing.T) {
			copia := solicitud
			copia.mensaje = append([]byte(nil), solicitud.mensaje...)
			caso.mutar(&copia)
			if copia.Validar() == nil {
				t.Fatal("solicitud adulterada aceptada")
			}
		})
	}

	otraCabecera := cabecera
	otraCabecera.Audiencia = "vec-diputacion/pruebas/otra"
	otraSolicitud, err := NuevaSolicitudFirmaAtestacionAutorizacionV3(
		otraCabecera,
		e.decision,
		e.motivo,
		e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := NuevoResultadoFirmaAtestacionAutorizacionV3(
		solicitud,
		[]byte("firma-opaca"),
		"evidencia_firma_0123456789abcdef",
		e.ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	if resultado.ValidarPara(otraSolicitud) == nil {
		t.Fatal("firma VEC-AD-3 aceptada para otra audiencia")
	}
}

func TestAtestacionAutorizacionV3BloqueaCodecsYFiltraFormato(t *testing.T) {
	e := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	solicitud, err := NuevaSolicitudFirmaAtestacionAutorizacionV3(
		cabeceraAtestacionAutorizacionV3PuertosPrueba(),
		e.decision,
		e.motivo,
		e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(solicitud); !errors.Is(
		err,
		ErrSerializacionAtestacionAutorizacionV3Prohibida,
	) {
		t.Fatalf("JSON no bloqueado: %v", err)
	}
	if _, err := solicitud.MarshalText(); !errors.Is(
		err,
		ErrSerializacionAtestacionAutorizacionV3Prohibida,
	) {
		t.Fatalf("texto no bloqueado: %v", err)
	}
	const marca = "[ATESTACION-AUTORIZACION-V3-REDACTADA-NO-AUTORITATIVA]"
	for _, valor := range []string{
		fmt.Sprint(solicitud),
		fmt.Sprintf("%+v", solicitud),
		fmt.Sprintf("%#v", solicitud),
	} {
		if !strings.Contains(valor, marca) || strings.Contains(valor, "dec_0123456789") {
			t.Fatalf("formato filtro contenido: %q", valor)
		}
	}
	var registro bytes.Buffer
	slog.New(slog.NewTextHandler(&registro, nil)).Info("prueba", "valor", solicitud)
	if !strings.Contains(registro.String(), marca) ||
		strings.Contains(registro.String(), "dec_0123456789") {
		t.Fatalf("slog filtro contenido: %s", registro.String())
	}
}

func cabeceraAtestacionAutorizacionV3PuertosPrueba() domain.CabeceraAtestacionAutorizacionV3 {
	return domain.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite:          "VEC-AD-3-COSE-EDDSA-1",
		ClaveID:        "clave:prueba:vec-ad-3:2026-07",
		Audiencia:      "vec-diputacion/pruebas/contratacion-temporal",
	}
}
