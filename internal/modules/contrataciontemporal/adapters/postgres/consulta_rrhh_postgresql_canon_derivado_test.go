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
