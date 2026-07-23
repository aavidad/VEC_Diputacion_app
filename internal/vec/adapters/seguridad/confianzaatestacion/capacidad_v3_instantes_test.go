package confianzaatestacion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"testing"
	"time"
)

func TestCanonTemporalCapacidadV3Determinista(t *testing.T) {
	entradasSQL := map[string]string{
		"2026-07-23T09:00:00.000000Z": "2026-07-23T09:00:00Z",
		"2026-07-23T09:00:00.123450Z": "2026-07-23T09:00:00.12345Z",
		"2026-07-23T09:00:00.123456Z": "2026-07-23T09:00:00.123456Z",
	}
	for entrada, salida := range entradasSQL {
		instante, err := parsearInstanteEntradaSQLO205(entrada)
		if err != nil || instante.Format(time.RFC3339Nano) != salida {
			t.Fatalf("frontera SQL temporal %q: %q, %v", entrada, salida, err)
		}
	}
	rechazados := []string{
		"2026-07-23T09:00:00.12345Z",
		"2026-07-23T09:00:00.1234560Z",
		"2026-07-23T09:00:00.123456789Z",
		"2026-07-23T11:00:00.123450+02:00",
		"2026-07-23t09:00:00.123450z",
	}
	for _, valor := range rechazados {
		if _, err := parsearInstanteEntradaSQLO205(valor); err == nil {
			t.Fatalf("entrada temporal SQL no canónica aceptada %q", valor)
		}
	}

	escenario, prueba := escenarioYPruebaConfianzaV3(t)
	for _, desplazamiento := range []time.Duration{
		0,
		123450 * time.Microsecond,
		123456 * time.Microsecond,
	} {
		emitidaEn := escenario.ahora.Add(desplazamiento)
		clave := claveCapacidadAtestacionV3Prueba(
			t,
			EstadoClaveHMACCapacidadAtestacionV3Emision,
			time.Time{},
			bytes.Repeat([]byte{0x73}, 32),
		)
		emisor, err := NuevoEmisorCapacidadesAtestacionAutorizacionV3(
			clave,
			&relojConfianzaAtestacionV3Prueba{ahora: emitidaEn},
		)
		if err != nil {
			t.Fatal(err)
		}
		capacidad, err := emisor.Emitir(
			context.Background(),
			escenario.solicitud,
			escenario.decision,
			escenario.motivo,
			escenario.resultado,
			escenario.atestacion,
			prueba,
		)
		if err != nil {
			t.Fatal(err)
		}
		exportacion, err := capacidad.ExportacionCanonicaParaConsumidor()
		if err != nil {
			t.Fatal(err)
		}
		documento, err := interpretarExportacionCapacidadV3(exportacion)
		if err != nil {
			t.Fatal(err)
		}
		if documento.EmitidaEn != emitidaEn.Format(time.RFC3339Nano) {
			t.Fatalf("emisión no canónica: %q", documento.EmitidaEn)
		}
	}

	evidencia, err := prueba.ExportacionCanonicaParaConsumidor()
	if err != nil {
		t.Fatal(err)
	}
	datos, err := prueba.Datos()
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(evidencia)
	if hex.EncodeToString(suma[:]) != datos.HuellaPruebaSHA256 {
		t.Fatal("la evidencia exportada no coincide con su huella")
	}
	if historica := preimagenHistoricaPruebaV3(datos); !bytes.Equal(
		historica,
		evidencia,
	) {
		t.Fatal("la exportación difiere de la preimagen histórica V3")
	}
}

func preimagenHistoricaPruebaV3(
	d DatosPruebaConfianzaAtestacionAutorizacionV3,
) []byte {
	var salida bytes.Buffer
	for _, campo := range []string{
		"vec.prueba-confianza-atestacion-autorizacion.v3",
		d.ReferenciaDecision, d.HuellaDecisionSHA256, d.HuellaMotivoSHA256,
		d.ReferenciaContexto, d.HuellaContextoSHA256, d.HuellaMensajeSHA256,
		d.HuellaSobreSHA256, d.ClaveID, d.HuellaClaveSPKISHA256,
		strconv.FormatUint(d.RaizVersion, 10),
		d.AlgoritmoCOSE, d.Suite, d.AudienciaDespliegue, string(d.EstadoClave),
		d.VerificadaEn.Format(time.RFC3339Nano),
		d.RaizValidaDesde.Format(time.RFC3339Nano),
		d.RaizValidaHasta.Format(time.RFC3339Nano),
		d.RevisionConfiguracion,
		strconv.FormatUint(d.SecuenciaConfiguracion, 10),
		d.HuellaConfiguracionSHA256,
		d.ConfiguracionPublicadaEn.Format(time.RFC3339Nano),
		d.ConfiguracionExpiraEn.Format(time.RFC3339Nano),
	} {
		var longitud [8]byte
		binary.BigEndian.PutUint64(longitud[:], uint64(len([]byte(campo))))
		_, _ = salida.Write(longitud[:])
		_, _ = salida.WriteString(campo)
	}
	return salida.Bytes()
}
