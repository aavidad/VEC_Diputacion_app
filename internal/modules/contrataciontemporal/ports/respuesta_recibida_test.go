package ports

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func solicitudRespuestaRecibidaPrueba() SolicitudRegistrarRespuestaRecibida {
	return SolicitudRegistrarRespuestaRecibida{
		ClaveIdempotencia:           "e53cb792-4c62-4daf-8c80-d5d18521748a",
		OrganizacionRef:             "org:desarrollo",
		ExpedienteRef:               "expediente:sintetico",
		LlamamientoRef:              "llamamiento:sintetico",
		ComunicacionRef:             "comunicacion:sintetica",
		VersionComunicacionEsperada: 2,
		Respuesta:                   RespuestaLlamamientoAceptada,
		CorreoRef:                   "correo:declarado-sintetico",
		CorreoSHA256:                strings.Repeat("a", 64),
		RecibidaEn:                  time.Date(2026, 9, 5, 12, 0, 0, 123456000, time.UTC),
	}
}

func resultadoRespuestaRecibidaPrueba() RespuestaRecibidaRegistrada {
	solicitud := solicitudRespuestaRecibidaPrueba()
	return RespuestaRecibidaRegistrada{
		Solicitud:       solicitud,
		JustificanteRef: "justificante:sintetico",
		ReciboRef:       "recibo:sintetico",
		AuditoriaRef:    "auditoria:sintetica",
		RegistradaEn:    solicitud.RecibidaEn.Add(time.Minute),
		Estado:          EstadoRespuestaRecibidaRegistrada,
	}
}

func TestRespuestaRecibidaValidaDeclaracionSinReloj(t *testing.T) {
	for _, respuesta := range []RespuestaLlamamiento{RespuestaLlamamientoAceptada, RespuestaLlamamientoRenunciada} {
		solicitud := solicitudRespuestaRecibidaPrueba()
		solicitud.Respuesta = respuesta
		if err := solicitud.Validar(); err != nil {
			t.Fatal(err)
		}
		// La validez temporal respecto de ahora pertenece a SQL, no al puerto.
		solicitud.RecibidaEn = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
		if err := solicitud.Validar(); err != nil {
			t.Fatalf("el puerto añadió un reloj: %v", err)
		}
	}
}

func TestRespuestaRecibidaRechazaMaterialInvalido(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*SolicitudRegistrarRespuestaRecibida)
	}{
		{"clave", func(s *SolicitudRegistrarRespuestaRecibida) { s.ClaveIdempotencia = "otra" }},
		{"organizacion", func(s *SolicitudRegistrarRespuestaRecibida) { s.OrganizacionRef = "" }},
		{"expediente", func(s *SolicitudRegistrarRespuestaRecibida) { s.ExpedienteRef = "expediente con espacios" }},
		{"llamamiento", func(s *SolicitudRegistrarRespuestaRecibida) { s.LlamamientoRef = "" }},
		{"comunicacion", func(s *SolicitudRegistrarRespuestaRecibida) { s.ComunicacionRef = "" }},
		{"version_cero", func(s *SolicitudRegistrarRespuestaRecibida) { s.VersionComunicacionEsperada = 0 }},
		{"version_anterior", func(s *SolicitudRegistrarRespuestaRecibida) { s.VersionComunicacionEsperada = 1 }},
		{"version_terminal", func(s *SolicitudRegistrarRespuestaRecibida) { s.VersionComunicacionEsperada = 3 }},
		{"respuesta_vacia", func(s *SolicitudRegistrarRespuestaRecibida) { s.Respuesta = "" }},
		{"expiracion_no_es_correo", func(s *SolicitudRegistrarRespuestaRecibida) { s.Respuesta = RespuestaLlamamientoExpirada }},
		{"correo_direccion", func(s *SolicitudRegistrarRespuestaRecibida) { s.CorreoRef = "persona@example.invalid" }},
		{"correo_largo", func(s *SolicitudRegistrarRespuestaRecibida) { s.CorreoRef = strings.Repeat("a", 161) }},
		{"huella_corta", func(s *SolicitudRegistrarRespuestaRecibida) { s.CorreoSHA256 = "abc" }},
		{"huella_cero", func(s *SolicitudRegistrarRespuestaRecibida) { s.CorreoSHA256 = strings.Repeat("0", 64) }},
		{"huella_mayuscula", func(s *SolicitudRegistrarRespuestaRecibida) { s.CorreoSHA256 = strings.Repeat("A", 64) }},
		{"huella_no_hex", func(s *SolicitudRegistrarRespuestaRecibida) { s.CorreoSHA256 = strings.Repeat("g", 64) }},
		{"fecha_cero", func(s *SolicitudRegistrarRespuestaRecibida) { s.RecibidaEn = time.Time{} }},
		{"fecha_no_utc", func(s *SolicitudRegistrarRespuestaRecibida) {
			s.RecibidaEn = s.RecibidaEn.In(time.FixedZone("Local", 0))
		}},
		{"fecha_submicro", func(s *SolicitudRegistrarRespuestaRecibida) { s.RecibidaEn = s.RecibidaEn.Add(time.Nanosecond) }},
		{"fecha_no_json", func(s *SolicitudRegistrarRespuestaRecibida) {
			s.RecibidaEn = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := solicitudRespuestaRecibidaPrueba()
			caso.mutar(&solicitud)
			if err := solicitud.Validar(); !errors.Is(err, ErrSolicitudRespuestaRecibidaInvalida) {
				t.Fatalf("material inválido aceptado: %v", err)
			}
		})
	}
}

func TestRespuestaRecibidaJSONMaterialExacto(t *testing.T) {
	solicitud := solicitudRespuestaRecibidaPrueba()
	tipo := reflect.TypeOf(solicitud)
	if tipo.NumField() != 10 {
		t.Fatal("el material debe tener exactamente diez campos")
	}
	for campo := 0; campo < tipo.NumField(); campo++ {
		if tipo.Field(campo).Tag != "" {
			t.Fatalf("campo con etiquetas: %s", tipo.Field(campo).Name)
		}
	}
	material, err := json.Marshal(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	esperado := "{\"ClaveIdempotencia\":\"e53cb792-4c62-4daf-8c80-d5d18521748a\",\"OrganizacionRef\":\"org:desarrollo\",\"ExpedienteRef\":\"expediente:sintetico\",\"LlamamientoRef\":\"llamamiento:sintetico\",\"ComunicacionRef\":\"comunicacion:sintetica\",\"VersionComunicacionEsperada\":2,\"Respuesta\":\"aceptacion\",\"CorreoRef\":\"correo:declarado-sintetico\",\"CorreoSHA256\":\"" + strings.Repeat("a", 64) + "\",\"RecibidaEn\":\"2026-09-05T12:00:00.123456Z\"}"
	if string(material) != esperado {
		t.Fatal("el material cambió nombres, orden, representación o envoltorio")
	}
}

func TestRespuestaRecibidaValidaResultadoYReplaySinAvanzarVersion(t *testing.T) {
	solicitud := solicitudRespuestaRecibidaPrueba()
	for _, estado := range []string{EstadoRespuestaRecibidaRegistrada, EstadoRespuestaRecibidaReplay} {
		resultado := resultadoRespuestaRecibidaPrueba()
		resultado.Estado = estado
		// Igualdad temporal permitida; un replay mantiene la fecha original.
		resultado.RegistradaEn = solicitud.RecibidaEn
		if err := resultado.ValidarPara(solicitud); err != nil {
			t.Fatal(err)
		}
	}
	// Cada uno de los diez campos está ligado al resultado, no solo la clave.
	for campo := 0; campo < reflect.TypeOf(solicitud).NumField(); campo++ {
		resultado := resultadoRespuestaRecibidaPrueba()
		valor := reflect.ValueOf(&resultado.Solicitud).Elem().Field(campo)
		switch valor.Kind() {
		case reflect.String:
			valor.SetString(valor.String() + "otro")
		case reflect.Uint64:
			valor.SetUint(3)
		default:
			valor.Set(reflect.ValueOf(solicitud.RecibidaEn.Add(time.Microsecond)))
		}
		if err := resultado.ValidarPara(solicitud); !errors.Is(err, ErrResultadoRespuestaRecibidaNoConfiable) {
			t.Fatalf("campo desligado aceptado: %s", reflect.TypeOf(solicitud).Field(campo).Name)
		}
	}
	casos := []struct {
		nombre string
		mutar  func(*RespuestaRecibidaRegistrada)
	}{
		{"justificante", func(r *RespuestaRecibidaRegistrada) { r.JustificanteRef = "" }},
		{"recibo", func(r *RespuestaRecibidaRegistrada) { r.ReciboRef = "" }},
		{"auditoria", func(r *RespuestaRecibidaRegistrada) { r.AuditoriaRef = "" }},
		{"estado_terminal", func(r *RespuestaRecibidaRegistrada) { r.Estado = "aceptada" }},
		{"estado_vacio", func(r *RespuestaRecibidaRegistrada) { r.Estado = "" }},
		{"registro_cero", func(r *RespuestaRecibidaRegistrada) { r.RegistradaEn = time.Time{} }},
		{"registro_no_utc", func(r *RespuestaRecibidaRegistrada) { r.RegistradaEn = r.RegistradaEn.In(time.FixedZone("Local", 0)) }},
		{"registro_submicro", func(r *RespuestaRecibidaRegistrada) { r.RegistradaEn = r.RegistradaEn.Add(time.Nanosecond) }},
		{"registro_anterior", func(r *RespuestaRecibidaRegistrada) { r.RegistradaEn = solicitud.RecibidaEn.Add(-time.Microsecond) }},
		{"solicitud_invalida", func(r *RespuestaRecibidaRegistrada) { r.Solicitud.CorreoRef = "" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resultado := resultadoRespuestaRecibidaPrueba()
			caso.mutar(&resultado)
			if err := resultado.ValidarPara(solicitud); !errors.Is(err, ErrResultadoRespuestaRecibidaNoConfiable) {
				t.Fatalf("resultado inválido aceptado: %v", err)
			}
		})
	}
	invalida := solicitud
	invalida.CorreoSHA256 = ""
	resultado := resultadoRespuestaRecibidaPrueba()
	resultado.Solicitud = invalida
	if err := resultado.ValidarPara(invalida); !errors.Is(err, ErrResultadoRespuestaRecibidaNoConfiable) {
		t.Fatal("coincidencia de solicitudes inválidas aceptada")
	}
}
