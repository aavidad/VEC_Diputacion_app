package postgres

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"
)

// Canon sintético V5 producido por el exportador de puertos con un acto hijo
// de secuencia 6 y seis actuaciones originales del padre.
const canonDetalleDerivadoV5Base64 = "VkVDLUNULUNPTlRFTklETy1ERVRBTExFLVJSSEgtVjUKMjM6ZXhwZWRpZW50ZTpycmhoOmhpam86MDEKMzE6b3JnYW5pemFjaW9uOmRpcHV0YWNp" +
	"b24tZ3JhbmFkYQoxNDoyMDI2L0NULU1JTi1QMQoxOjYKMjE6Zmx1am86cnJoaDptaW5pbWl6YWRvCjE6MQo2NDphYWFhYWFhYWFhYWFhYWFhYWFh" +
	"YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhCjEzOmZpc2NhbGl6YWNpb24KODplbl9jdXJzbwoyMjpjZW50cm86" +
	"cnJoaDptaW5pbWl6YWRvCjI1OmNhdGVnb3JpYTpycmhoOm1pbmltaXphZGEKMTE6aW50ZXJpbmlkYWQKMjI6dW5pZGFkOnJyaGg6bWluaW1pemFk" +
	"YQoyNzoyMDI2LTA3LTI2VDA4OjAzOjAwLjAwMDAwMFoKMjc6MjAyNi0wNy0yNlQwODowMzowMC4wMDAwMDBaCjI6QzIKMTE6c3VzdGl0dWNpb24K" +
	"Mjc6MjAyNi0wOC0yNlQwMDowMDowMC4wMDAwMDBaCjMzOmZlY2hhOjIwMjYtMDktMjZUMDA6MDA6MDAuMDAwMDAwWgoyOjE1CjE6MQoxOjIKMTE6" +
	"aW50ZXJpbmlkYWQKMjU6Y2F0ZWdvcmlhOnJyaGg6bWluaW1pemFkYQoxMTpzdXN0aXR1Y2lvbgoyNzoyMDI2LTA4LTI2VDAwOjAwOjAwLjAwMDAw" +
	"MFoKMzM6ZmVjaGE6MjAyNi0wOS0yNlQwMDowMDowMC4wMDAwMDBaCjU6MTAwMDAKMTI6bm9fcmVxdWVyaWRhCjE6MQo2OjEyNTAwMAozOkVVUgoy" +
	"MzpmdWVudGU6Y29zdGU6bWluaW1pemFkYQowOgoxOjEKMTozCjU6Ym9sc2EKMTowCjI5OnByb2NlZGltaWVudG86cnJoaDptaW5pbWl6YWRvCjIx" +
	"OmJvbHNhOnJyaGg6bWluaW1pemFkYQoxOjEKMTQ6ZGlzcG9uaWJpbGlkYWQKMTA6YWZpcm1hdGl2YQoxOjEKMTo0CjIyOnVuaWRhZDpycmhoOm1p" +
	"bmltaXphZGEKMjc6MjAyNi0wNy0yNlQwODowMDowMC4wMDAwMDBaCjA6CjE6MQoxOjYKMTowCjk6ZmF2b3JhYmxlCjE6MAoyNzoyMDI2LTA3LTI2" +
	"VDA4OjAyOjAwLjAwMDAwMFoKMTowCjE6MQoxOjYKMTo2CjQyOmNvbnRyYXRhY2lvbl90ZW1wb3JhbC5ub21icmFtaWVudG8uZGVyaXZhcgoyNzoy" +
	"MDI2LTA3LTI2VDA4OjAzOjAwLjAwMDAwMFoKMTM6ZmlzY2FsaXphY2lvbgoxMzpmaXNjYWxpemFjaW9uCjg6ZW5fY3Vyc28KODplbl9jdXJzbwox" +
	"NjpkZXJpdmFjaW9uOmN0OjAxCjI2OmV4cGVkaWVudGU6cnJoaDptaW5pbWl6YWRvCjExOjIwMjYvQ1QtTUlOCjEyOm9mZXJ0YTpjdDowMQoxOjEK" +
	"MTI6cmVjaWJvOmN0OjAxCjY0OmFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWEK" +
	"Mjc6MjAyNi0wNy0yNlQwNzo1NzowMC4wMDAwMDBaCjI3OjIwMjYtMDctMjZUMDg6MDI6MDAuMDAwMDAwWgoxOjYKMToxCjE6MQoyMjphY3R1YWNp" +
	"b24ubWluaW1pemFkYS4xCjI3OjIwMjYtMDctMjZUMDc6NTc6MDAuMDAwMDAwWgowOgo5OnNvbGljaXR1ZAo5OnBlbmRpZW50ZQo4OmVuX2N1cnNv" +
	"CjE6MgoxOjIKMjI6YWN0dWFjaW9uLm1pbmltaXphZGEuMgoyNzoyMDI2LTA3LTI2VDA3OjU4OjAwLjAwMDAwMFoKOTpzb2xpY2l0dWQKMTM6Z2Vz" +
	"dGlvbl9ib2xzYQo4OmVuX2N1cnNvCjg6ZW5fY3Vyc28KMTozCjE6MwoyMjphY3R1YWNpb24ubWluaW1pemFkYS4zCjI3OjIwMjYtMDctMjZUMDc6" +
	"NTk6MDAuMDAwMDAwWgoxMzpnZXN0aW9uX2JvbHNhCjE3OmFzaWduYWNpb25fdW5pZGFkCjg6ZW5fY3Vyc28KODplbl9jdXJzbwoxOjQKMTo0CjIy" +
	"OmFjdHVhY2lvbi5taW5pbWl6YWRhLjQKMjc6MjAyNi0wNy0yNlQwODowMDowMC4wMDAwMDBaCjE3OmFzaWduYWNpb25fdW5pZGFkCjE0OnVuaWRh" +
	"ZF9nZXN0b3JhCjg6ZW5fY3Vyc28KODplbl9jdXJzbwoxOjUKMTo1CjQ2OmNvbnRyYXRhY2lvbl90ZW1wb3JhbC5pbmZvcm1lX2p1cmlkaWNvLmdl" +
	"bmVyYXIKMjc6MjAyNi0wNy0yNlQwODowMTowMC4wMDAwMDBaCjE0OnVuaWRhZF9nZXN0b3JhCjE2OmluZm9ybWVfanVyaWRpY28KODplbl9jdXJz" +
	"bwo4OmVuX2N1cnNvCjE6NgoxOjYKNDU6Y29udHJhdGFjaW9uX3RlbXBvcmFsLmZpc2NhbGl6YWNpb24ucmVnaXN0cmFyCjI3OjIwMjYtMDctMjZU" +
	"MDg6MDI6MDAuMDAwMDAwWgoxNjppbmZvcm1lX2p1cmlkaWNvCjEzOmZpc2NhbGl6YWNpb24KODplbl9jdXJzbwo4OmVuX2N1cnNvCg=="

func TestDecodificarDetalleDerivadoV5ReproduceCanonYRechazaMezclas(t *testing.T) {
	canon, err := base64.StdEncoding.DecodeString(canonDetalleDerivadoV5Base64)
	if err != nil {
		t.Fatal(err)
	}
	d, err := decodificarContenidoDetalleRRHHPostgreSQL(canon)
	if err != nil || d.version != 6 {
		t.Fatalf("analizar V5: %v", err)
	}
	_, err = (analizadorCanonConsultaRRHHPostgreSQL{}).analizarDetalle(canon, d.actualizadoEn.Add(time.Minute), d.expedienteRef, d.version)
	if err != nil {
		t.Fatalf("reproducir V5: %v", err)
	}
	accion := []byte("contratacion_temporal.fiscalizacion.registrar")
	alterado := bytes.Replace(canon, accion, bytes.Repeat([]byte("x"), len(accion)), 1)
	if bytes.Equal(alterado, canon) {
		t.Fatal("no se localizó fiscalización del padre")
	}
	if _, err := (analizadorCanonConsultaRRHHPostgreSQL{}).analizarDetalle(alterado, d.actualizadoEn.Add(time.Minute), d.expedienteRef, d.version); err == nil {
		t.Fatal("se admitió hito padre adulterado")
	}
	recibo := []byte("recibo:ct:01")
	alterado = bytes.Replace(canon, recibo, bytes.Repeat([]byte("!"), len(recibo)), 1)
	if bytes.Equal(alterado, canon) {
		t.Fatal("no se localizó recibo")
	}
	if _, err := (analizadorCanonConsultaRRHHPostgreSQL{}).analizarDetalle(alterado, d.actualizadoEn.Add(time.Minute), d.expedienteRef, d.version); err == nil {
		t.Fatal("se admitió recibo inválido")
	}
}

// Mismo exportador de puertos con propuesta real en hito propio 7.
const canonDetalleDerivadoV6Base64 = "VkVDLUNULUNPTlRFTklETy1ERVRBTExFLVJSSEgtVjYKMjM6ZXhwZWRpZW50ZTpycmhoOmhpam86MDEKMzE6b3JnYW5pemFjaW9uOmRpcHV0YWNp" +
	"b24tZ3JhbmFkYQoxNDoyMDI2L0NULU1JTi1QMQoxOjcKMjE6Zmx1am86cnJoaDptaW5pbWl6YWRvCjE6MQo2NDphYWFhYWFhYWFhYWFhYWFhYWFh" +
	"YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhCjEyOm5vbWJyYW1pZW50bwo4OmVuX2N1cnNvCjIyOmNlbnRybzpy" +
	"cmhoOm1pbmltaXphZG8KMjU6Y2F0ZWdvcmlhOnJyaGg6bWluaW1pemFkYQoxMTppbnRlcmluaWRhZAoyMjp1bmlkYWQ6cnJoaDptaW5pbWl6YWRh" +
	"CjI3OjIwMjYtMDctMjZUMDg6MDM6MDAuMDAwMDAwWgoyNzoyMDI2LTA3LTI2VDA4OjA0OjAwLjAwMDAwMFoKMjpDMgoxMTpzdXN0aXR1Y2lvbgoy" +
	"NzoyMDI2LTA4LTI2VDAwOjAwOjAwLjAwMDAwMFoKMzM6ZmVjaGE6MjAyNi0wOS0yNlQwMDowMDowMC4wMDAwMDBaCjI6MTUKMToxCjE6MgoxMTpp" +
	"bnRlcmluaWRhZAoyNTpjYXRlZ29yaWE6cnJoaDptaW5pbWl6YWRhCjExOnN1c3RpdHVjaW9uCjI3OjIwMjYtMDgtMjZUMDA6MDA6MDAuMDAwMDAw" +
	"WgozMzpmZWNoYToyMDI2LTA5LTI2VDAwOjAwOjAwLjAwMDAwMFoKNToxMDAwMAoxMjpub19yZXF1ZXJpZGEKMToxCjY6MTI1MDAwCjM6RVVSCjIz" +
	"OmZ1ZW50ZTpjb3N0ZTptaW5pbWl6YWRhCjA6CjE6MQoxOjMKNTpib2xzYQoxOjAKMjk6cHJvY2VkaW1pZW50bzpycmhoOm1pbmltaXphZG8KMjE6" +
	"Ym9sc2E6cnJoaDptaW5pbWl6YWRhCjE6MQoxNDpkaXNwb25pYmlsaWRhZAoxMDphZmlybWF0aXZhCjE6MQoxOjQKMjI6dW5pZGFkOnJyaGg6bWlu" +
	"aW1pemFkYQoyNzoyMDI2LTA3LTI2VDA4OjAwOjAwLjAwMDAwMFoKMDoKMToxCjE6NgoxOjAKOTpmYXZvcmFibGUKMTowCjI3OjIwMjYtMDctMjZU" +
	"MDg6MDI6MDAuMDAwMDAwWgoxOjAKMToyCjE6NgoxOjYKNDI6Y29udHJhdGFjaW9uX3RlbXBvcmFsLm5vbWJyYW1pZW50by5kZXJpdmFyCjI3OjIw" +
	"MjYtMDctMjZUMDg6MDM6MDAuMDAwMDAwWgoxMzpmaXNjYWxpemFjaW9uCjEzOmZpc2NhbGl6YWNpb24KODplbl9jdXJzbwo4OmVuX2N1cnNvCjE6" +
	"NwoxOjcKMzM6cmVnaXN0cmFyX3Byb3B1ZXN0YV9mb3JtYWxpemFjaW9uCjI3OjIwMjYtMDctMjZUMDg6MDQ6MDAuMDAwMDAwWgoxMzpmaXNjYWxp" +
	"emFjaW9uCjEyOm5vbWJyYW1pZW50bwo4OmVuX2N1cnNvCjg6ZW5fY3Vyc28KMTY6ZGVyaXZhY2lvbjpjdDowMQoyNjpleHBlZGllbnRlOnJyaGg6" +
	"bWluaW1pemFkbwoxMToyMDI2L0NULU1JTgoxMjpvZmVydGE6Y3Q6MDEKMToxCjEyOnJlY2libzpjdDowMQo2NDphYWFhYWFhYWFhYWFhYWFhYWFh" +
	"YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhCjI3OjIwMjYtMDctMjZUMDc6NTc6MDAuMDAwMDAwWgoyNzoyMDI2" +
	"LTA3LTI2VDA4OjAyOjAwLjAwMDAwMFoKMTo2CjE6MQoxOjEKMjI6YWN0dWFjaW9uLm1pbmltaXphZGEuMQoyNzoyMDI2LTA3LTI2VDA3OjU3OjAw" +
	"LjAwMDAwMFoKMDoKOTpzb2xpY2l0dWQKOTpwZW5kaWVudGUKODplbl9jdXJzbwoxOjIKMToyCjIyOmFjdHVhY2lvbi5taW5pbWl6YWRhLjIKMjc6" +
	"MjAyNi0wNy0yNlQwNzo1ODowMC4wMDAwMDBaCjk6c29saWNpdHVkCjEzOmdlc3Rpb25fYm9sc2EKODplbl9jdXJzbwo4OmVuX2N1cnNvCjE6Mwox" +
	"OjMKMjI6YWN0dWFjaW9uLm1pbmltaXphZGEuMwoyNzoyMDI2LTA3LTI2VDA3OjU5OjAwLjAwMDAwMFoKMTM6Z2VzdGlvbl9ib2xzYQoxNzphc2ln" +
	"bmFjaW9uX3VuaWRhZAo4OmVuX2N1cnNvCjg6ZW5fY3Vyc28KMTo0CjE6NAoyMjphY3R1YWNpb24ubWluaW1pemFkYS40CjI3OjIwMjYtMDctMjZU" +
	"MDg6MDA6MDAuMDAwMDAwWgoxNzphc2lnbmFjaW9uX3VuaWRhZAoxNDp1bmlkYWRfZ2VzdG9yYQo4OmVuX2N1cnNvCjg6ZW5fY3Vyc28KMTo1CjE6" +
	"NQo0Njpjb250cmF0YWNpb25fdGVtcG9yYWwuaW5mb3JtZV9qdXJpZGljby5nZW5lcmFyCjI3OjIwMjYtMDctMjZUMDg6MDE6MDAuMDAwMDAwWgox" +
	"NDp1bmlkYWRfZ2VzdG9yYQoxNjppbmZvcm1lX2p1cmlkaWNvCjg6ZW5fY3Vyc28KODplbl9jdXJzbwoxOjYKMTo2CjQ1OmNvbnRyYXRhY2lvbl90" +
	"ZW1wb3JhbC5maXNjYWxpemFjaW9uLnJlZ2lzdHJhcgoyNzoyMDI2LTA3LTI2VDA4OjAyOjAwLjAwMDAwMFoKMTY6aW5mb3JtZV9qdXJpZGljbwox" +
	"MzpmaXNjYWxpemFjaW9uCjg6ZW5fY3Vyc28KODplbl9jdXJzbwoxNTpwcm9wdWVzdGE6Y3Q6MDEKMTk6cmVjaWJvOnByb3B1ZXN0YTowMQoxOjcK" +
	"Mjc6MjAyNi0wNy0yNlQwODowNDowMC4wMDAwMDBaCg=="

func TestDecodificarDetalleDerivadoV6PropuestaEstricto(t *testing.T) {
	canon, err := base64.StdEncoding.DecodeString(canonDetalleDerivadoV6Base64)
	if err != nil {
		t.Fatal(err)
	}
	d, err := decodificarContenidoDetalleRRHHPostgreSQL(canon)
	if err != nil || d.version != 7 {
		t.Fatalf("decodificar V6: %v", err)
	}
	if _, err := (analizadorCanonConsultaRRHHPostgreSQL{}).analizarDetalle(
		canon, d.actualizadoEn.Add(time.Minute), d.expedienteRef, d.version); err != nil {
		t.Fatalf("reproducir V6: %v", err)
	}
	casos := map[string][]byte{
		"referencia": bytes.Replace(canon, []byte("propuesta:ct:01"), bytes.Repeat([]byte("!"), len("propuesta:ct:01")), 1),
		"recibo":     bytes.Replace(canon, []byte("recibo:propuesta:01"), bytes.Repeat([]byte("!"), len("recibo:propuesta:01")), 1),
	}
	version := bytes.Clone(canon)
	indice := bytes.LastIndex(version, []byte("1:7\n"))
	if indice < 0 {
		t.Fatal("falta versión de propuesta")
	}
	version[indice+2] = '8'
	casos["versión"] = version
	fecha := bytes.Clone(canon)
	marca := []byte(d.actualizadoEn.Format("2006-01-02T15:04:05.000000Z"))
	indice = bytes.LastIndex(fecha, marca)
	if indice < 0 {
		t.Fatal("falta fecha de propuesta")
	}
	fecha[indice+18] = '1'
	casos["fecha"] = fecha
	for nombre, alterado := range casos {
		t.Run(nombre, func(t *testing.T) {
			if bytes.Equal(alterado, canon) {
				t.Fatal("la adulteración no cambió el canon")
			}
			if _, err := (analizadorCanonConsultaRRHHPostgreSQL{}).analizarDetalle(
				alterado, d.actualizadoEn.Add(time.Minute), d.expedienteRef, d.version); err == nil {
				t.Fatal("se admitió propuesta adulterada")
			}
		})
	}
}
