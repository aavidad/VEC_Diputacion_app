package domain

import (
	"testing"
	"time"
)

func FuzzRehidratarFuenteAutoridadV1(f *testing.F) {
	instante := time.Date(2026, time.July, 17, 9, 30, 0, 0, time.UTC)
	fuente, err := NuevaFuenteAutoridadBorradorV1(DatosAltaFuenteAutoridadV1{
		ID: "fuente_fuzz_v1", Contenido: contenidoFuenteAutoridadPrueba(),
		CreadaPor: "per_creador_fuzz_fuente_0000001", CreadaEn: instante,
		MotivoCreacionCodigo: "alta_fuzz",
	})
	if err != nil {
		f.Fatal(err)
	}
	estado, err := fuente.EstadoPersistibleV1()
	if err != nil {
		f.Fatal(err)
	}
	f.Add(estado)
	f.Add([]byte(`{}`))
	f.Add([]byte{0xff, 0xfe})
	f.Fuzz(func(t *testing.T, datos []byte) {
		_, _ = RehidratarFuenteAutoridadV1(datos)
	})
}

func FuzzRehidratarSolicitudTransicionFuenteAutoridadV1(f *testing.F) {
	instante := time.Date(2026, time.July, 17, 9, 30, 0, 0, time.UTC)
	fuente, err := NuevaFuenteAutoridadBorradorV1(DatosAltaFuenteAutoridadV1{
		ID: "solicitud_fuzz_v1", Contenido: contenidoFuenteAutoridadPrueba(),
		CreadaPor: "per_creador_fuzz_solicitud_0001", CreadaEn: instante,
		MotivoCreacionCodigo: "alta_fuzz",
	})
	if err != nil {
		f.Fatal(err)
	}
	solicitud, err := fuente.PrepararSolicitudTransicionV1(DatosPreparacionTransicionFuenteAutoridadV1{
		EstadoNuevo: EstadoFuenteAutoridadPublicada,
		ActorRef:    "per_publicador_fuzz_solicitud_001", MotivoCodigo: "publicacion_fuzz",
		SolicitudRef: "solicitud:fuzz:v1", PreparadaEn: instante.Add(time.Hour),
		ExpiraEn: instante.Add(2 * time.Hour),
	})
	if err != nil {
		f.Fatal(err)
	}
	bytesSolicitud, err := solicitud.BytesCanonicos()
	if err != nil {
		f.Fatal(err)
	}
	f.Add(bytesSolicitud)
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, datos []byte) {
		_, _ = RehidratarSolicitudTransicionFuenteAutoridadV1(datos)
	})
}
