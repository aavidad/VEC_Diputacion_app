package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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
	t.Run("mensaje A con compromisos B recomputados", func(t *testing.T) {
		cruzada := solicitud
		cruzada.mensaje = append([]byte(nil), solicitud.mensaje...)
		cruzada.referenciaDecision = "dec_otra234567890abcdef0123456789ab"
		cruzada.huellaDecision = strings.Repeat("a", 64)
		cruzada.referenciaContexto = "rca_otra234567890abcdefghijklmn"
		cruzada.huellaContexto = strings.Repeat("b", 64)
		cruzada.huellaCompromisos = cruzada.calcularHuellaCompromisos()
		if cruzada.Validar() == nil {
			t.Fatal("mensaje A aceptado con compromisos B recomputados")
		}
	})
	t.Run("acción vacía con todos los compromisos recompuestos", func(t *testing.T) {
		cruzada := solicitud
		cruzada.mensaje = append([]byte(nil), solicitud.mensaje...)
		inicio, fin := limitesDecisionAtestacionAutorizacionV3PuertosPrueba(
			t,
			cruzada.mensaje,
		)
		const accionOriginal = "bolsa.expediente.leer"
		original := []byte(`"accion":"` + accionOriginal + `"`)
		reemplazo := append(
			append([]byte(`"accion":"`), bytes.Repeat([]byte(" "), len(accionOriginal))...),
			'"',
		)
		if len(original) != len(reemplazo) {
			t.Fatal("fixture de sustitución no conserva longitud")
		}
		decisionMutada := bytes.Replace(
			cruzada.mensaje[inicio:fin],
			original,
			reemplazo,
			1,
		)
		if bytes.Equal(decisionMutada, cruzada.mensaje[inicio:fin]) {
			t.Fatal("no se encontró la acción del fixture")
		}
		copy(cruzada.mensaje[inicio:fin], decisionMutada)
		huellaDecision := sha256.Sum256(decisionMutada)
		huellaMensaje := sha256.Sum256(cruzada.mensaje)
		cruzada.huellaDecision = hex.EncodeToString(huellaDecision[:])
		cruzada.huellaMensaje = hex.EncodeToString(huellaMensaje[:])
		cruzada.huellaCompromisos = cruzada.calcularHuellaCompromisos()
		if cruzada.Validar() == nil {
			t.Fatal("acción vacía aceptada tras recomponer todos los compromisos")
		}
	})

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

func limitesDecisionAtestacionAutorizacionV3PuertosPrueba(
	t *testing.T,
	mensaje []byte,
) (int, int) {
	t.Helper()
	posicion := len(domain.EsquemaMensajeAtestacionAutorizacionV3) + 1 + 2
	for indice := 0; indice < 3; indice++ {
		if posicion+4 > len(mensaje) {
			t.Fatal("cabecera VEC-AD-3 truncada")
		}
		longitud := int(binary.BigEndian.Uint32(mensaje[posicion : posicion+4]))
		posicion += 4 + longitud
	}
	if posicion+4 > len(mensaje) {
		t.Fatal("bloque de decisión VEC-AD-3 ausente")
	}
	longitud := int(binary.BigEndian.Uint32(mensaje[posicion : posicion+4]))
	inicio := posicion + 4
	fin := inicio + longitud
	if longitud <= 0 || fin > len(mensaje)-8 {
		t.Fatal("bloque de decisión VEC-AD-3 inválido")
	}
	return inicio, fin
}

func cabeceraAtestacionAutorizacionV3PuertosPrueba() domain.CabeceraAtestacionAutorizacionV3 {
	return domain.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite:          "VEC-AD-3-COSE-EDDSA-1",
		ClaveID:        "clave:prueba:vec-ad-3:2026-07",
		Audiencia:      "vec-diputacion/pruebas/contratacion-temporal",
	}
}
