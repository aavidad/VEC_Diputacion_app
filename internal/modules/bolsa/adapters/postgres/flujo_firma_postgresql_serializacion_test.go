package postgres

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestFlujoFirmaPostgreSQLSerializaSinExponerCifradoEnJSON(t *testing.T) {
	expediente := expedienteFlujoFirmaPostgreSQLPrueba(t)
	documento, cifrado, err := serializarExpedienteFlujoFirmaPostgreSQL(expediente)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(documento, bytes.Repeat([]byte{0x42}, 8)) ||
		!bytes.Equal(cifrado, bytes.Repeat([]byte{0x42}, 48)) {
		t.Fatal("el documento mezcló el cifrado con los metadatos JSON")
	}
	restaurado, err := decodificarExpedienteFlujoFirmaPostgreSQL(documento, cifrado)
	if err != nil {
		t.Fatal(err)
	}
	esperada, err := puertosbolsa.RepresentacionCanonicaExpedienteFlujoFirmaBaremacion(expediente)
	if err != nil {
		t.Fatal(err)
	}
	obtenida, err := puertosbolsa.RepresentacionCanonicaExpedienteFlujoFirmaBaremacion(restaurado)
	if err != nil || !bytes.Equal(esperada.Revelar(), obtenida.Revelar()) {
		t.Fatalf("la rehidratación alteró el expediente: %v", err)
	}
}

func TestFlujoFirmaPostgreSQLRechazaRespuestaAmbiguaOAlterada(t *testing.T) {
	expediente := expedienteFlujoFirmaPostgreSQLPrueba(t)
	documento, cifrado, err := serializarExpedienteFlujoFirmaPostgreSQL(expediente)
	if err != nil {
		t.Fatal(err)
	}
	duplicado := append([]byte(`{"esquema":"forjado",`), documento[1:]...)
	desconocido := append([]byte(nil), documento[:len(documento)-1]...)
	desconocido = append(desconocido, []byte(`,"campo_ajeno":true}`)...)
	cifradoAlterado := append([]byte(nil), cifrado...)
	cifradoAlterado[len(cifradoAlterado)-1] ^= 0xff
	for nombre, caso := range map[string]struct {
		documento []byte
		cifrado   []byte
	}{
		"clave duplicada":   {duplicado, cifrado},
		"campo desconocido": {desconocido, cifrado},
		"cifrado alterado":  {documento, cifradoAlterado},
		"documento vacío":   {nil, cifrado},
		"cifrado vacío":     {documento, nil},
	} {
		t.Run(nombre, func(t *testing.T) {
			resultado, err := decodificarExpedienteFlujoFirmaPostgreSQL(
				caso.documento,
				caso.cifrado,
			)
			if !errors.Is(err, puertosbolsa.ErrEstadoFlujoFirmaAlterado) ||
				resultado.Validar() == nil {
				t.Fatalf("respuesta no fiable aceptada: resultado=%+v error=%v", resultado, err)
			}
		})
	}
}

func TestFlujoFirmaPostgreSQLRechazaEnterosEInstantesNoCanonicos(t *testing.T) {
	expediente := expedienteFlujoFirmaPostgreSQLPrueba(t)
	documento, cifrado, err := serializarExpedienteFlujoFirmaPostgreSQL(expediente)
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string][]byte{
		"versión con cero": bytes.Replace(
			documento,
			[]byte(`"version":"1"`),
			[]byte(`"version":"01"`),
			1,
		),
		"zona no UTC": bytes.Replace(
			documento,
			[]byte(`"creado_en":"2026-07-24T20:00:00.123456Z"`),
			[]byte(`"creado_en":"2026-07-24T22:00:00.123456+02:00"`),
			1,
		),
	}
	for nombre, alterado := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := decodificarExpedienteFlujoFirmaPostgreSQL(
				alterado,
				cifrado,
			); !errors.Is(err, puertosbolsa.ErrEstadoFlujoFirmaAlterado) {
				t.Fatalf("representación no canónica aceptada: %v", err)
			}
		})
	}
}

func expedienteFlujoFirmaPostgreSQLPrueba(
	t *testing.T,
) puertosbolsa.ExpedienteFlujoFirmaBaremacion {
	t.Helper()
	estado, err := puertosbolsa.NuevoEstadoProtegidoFlujoFirmaBaremacion(
		puertosbolsa.AlgoritmoProteccionEstadoAES256GCM,
		"clave-estado-firma-postgresql-v1",
		bytes.Repeat([]byte{0x31}, 12),
		bytes.Repeat([]byte{0x42}, 48),
	)
	if err != nil {
		t.Fatal(err)
	}
	instante := time.Date(2026, time.July, 24, 20, 0, 0, 123_456_000, time.UTC)
	expediente := puertosbolsa.ExpedienteFlujoFirmaBaremacion{
		FlujoRef: "flujo-firma-postgresql-001", Version: 1,
		IndiceIdempotenciaHMAC: hmacFlujoFirmaPostgreSQLPrueba("1"),
		HuellaSolicitudHMAC:    hmacFlujoFirmaPostgreSQLPrueba("2"),
		VinculoActorHMAC:       hmacFlujoFirmaPostgreSQLPrueba("3"),
		PerfilActorClave:       "tecnico_rrhh",
		ProcesoRef:             "proceso-firma-postgresql-001",
		SolicitudRef:           "solicitud-firma-postgresql-001",
		BaremacionMeritoRef:    "baremacion-firma-postgresql-001",
		DecisionRef:            "decision-firma-postgresql-001",
		Estado:                 puertosbolsa.EstadoExpedienteFirmaPreparando,
		EstadoProtegido:        estado,
		CreadoEn:               instante,
		ActualizadoEn:          instante,
		SelloEstadoHMAC:        hmacFlujoFirmaPostgreSQLPrueba("4"),
	}
	if err := expediente.Validar(); err != nil {
		t.Fatalf("fixture inválido: %v", err)
	}
	return expediente
}

func hmacFlujoFirmaPostgreSQLPrueba(caracter string) string {
	return "hmac-sha256:flujo_firma_postgresql_v1:" +
		strings.Repeat(caracter, 64)
}
