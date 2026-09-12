package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func solicitudCorreoLlamamientoPrueba() SolicitudDespacharCorreoLlamamiento {
	return SolicitudDespacharCorreoLlamamiento{
		OrganizacionRef: "org-001", ExpedienteRef: "exp-001", LlamamientoRef: "llamamiento-001",
		ComunicacionRef: "comunicacion-001", IntencionEnvioRef: "intencion-001",
	}
}

func TestSolicitudCorreoLlamamientoCanonizaCincoReferencias(t *testing.T) {
	solicitud := solicitudCorreoLlamamientoPrueba()
	material, err := solicitud.SerializarCanonico()
	if err != nil || !strings.Contains(string(material), `"intencion-001"`) {
		t.Fatalf("serialización=%s error=%v", material, err)
	}
	huella, err := solicitud.HuellaSHA256()
	if err != nil || len(huella) != 64 || strings.Trim(huella, "0") == "" {
		t.Fatalf("huella=%q error=%v", huella, err)
	}
	solicitud.ComunicacionRef = "comunicacion-002"
	otraHuella, err := solicitud.HuellaSHA256()
	if err != nil || otraHuella == huella {
		t.Fatal("una de las cinco referencias no liga la huella")
	}
	solicitud.LlamamientoRef = ""
	if _, err := solicitud.SerializarCanonico(); err != ErrSolicitudCorreoLlamamientoInvalida {
		t.Fatalf("error=%v", err)
	}
}

func TestReservaYCapacidadFinalizacionRechazanCrucesYReplay(t *testing.T) {
	solicitud := solicitudCorreoLlamamientoPrueba()
	huella, err := solicitud.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	reserva := ReservaIntentoCorreoLlamamiento{
		IntentoRef: "intento-001", MessageID: "<intento-001@vec.invalid>",
		FechaOrigen:     time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC),
		SolicitudHuella: huella, Estado: CorreoLlamamientoIniciado,
	}
	if err := reserva.ValidarPara(solicitud); err != nil {
		t.Fatal(err)
	}
	capacidad, err := NuevaCapacidadFinalizacionIntentoCorreoLlamamiento(reserva, []byte(strings.Repeat("s", 32)))
	if err != nil || capacidad.ValidarPara(reserva) != nil {
		t.Fatal(err)
	}
	reserva.YaReservado = true
	if capacidad.ValidarPara(reserva) == nil {
		t.Fatal("capacidad de reserva nueva admitida para replay")
	}
}

func TestAuditoriaResultadoCorreoLlamamientoCanonizaAuditEntry17(t *testing.T) {
	solicitud := solicitudCorreoLlamamientoPrueba()
	huella, err := solicitud.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	reserva := ReservaIntentoCorreoLlamamiento{IntentoRef: "intento-correo-prueba", MessageID: "<m@vec.invalid>", FechaOrigen: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC), SolicitudHuella: huella, Estado: CorreoLlamamientoIniciado}
	resultado, err := NuevaSolicitudRegistrarResultadoCorreoLlamamiento(solicitud, reserva, CorreoLlamamientoAceptadoPorRelay, PlantillaCorreoLlamamientoV1)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	auditoria, err := NuevaAuditoriaResultadoCorreoLlamamiento(DatosAuditoriaResultadoCorreoLlamamiento{ActorID: "hmac-sha256:prueba:" + strings.Repeat("a", 64), ActorProfile: "perfil-rrhh", VersionRolRef: "rol-version-rrhh-1", AuthMethod: "certificado", AuthAssurance: "alto", Correlacion: correlacion, Solicitud: resultado, OcurridoEn: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := auditoria.SerializarCanonico()
	if err != nil || !strings.Contains(string(contenido), `"actor_roles":["rol-version-rrhh-1"]`) || !strings.Contains(string(contenido), `"occurred_at":"2026-09-12T12:00:00.000000Z"`) {
		t.Fatalf("AuditEntry17=%s err=%v", contenido, err)
	}

	texto, err := correlacion.ValorCanonico()
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"id":"","seq":0,"signature":"","actor_id":"hmac-sha256:prueba:` + strings.Repeat("a", 64) + `","actor_profile":"perfil-rrhh","actor_roles":["rol-version-rrhh-1"],"auth_method":"certificado","auth_assurance":"alto","purpose":"gestionar_contratacion_temporal","action":"contratacion_temporal.llamamiento.correo.registrar_resultado","module_id":"vec.module.contratacion_temporal","subject_ref":"intento-correo-prueba","object_version":2,"expediente_ref":"exp-001","result":"accepted","correlation_ref":"` + texto + `","occurred_at":"2026-09-12T12:00:00.000000Z"}`
	if string(contenido) != esperado {
		t.Fatal("cambió el contrato de bytes Audit17")
	}
	hash, err := auditoria.HuellaSHA256()
	if err != nil || hash != sha256HexCorreoPrueba([]byte(esperado)) {
		t.Fatal("huella distinta de Audit17 original")
	}
	original, err := auditoria.CorrelacionPara(resultado)
	if err != nil || original != correlacion {
		t.Fatal("capacidad nominal original no conservada")
	}
	otra := resultado
	otra.IntentoRef = "intento-ajeno"
	if _, err := auditoria.CorrelacionPara(otra); err == nil {
		t.Fatal("correlación transportada para otra solicitud")
	}
	sinCorrelacion := auditoria.datos
	sinCorrelacion.Correlacion = vecdomain.ReferenciaCorrelacionAutorizacionV2{}
	if _, err := NuevaAuditoriaResultadoCorreoLlamamiento(sinCorrelacion); err == nil {
		t.Fatal("auditoría sin correlación nominal aceptada")
	}
	if _, err := NuevaAuditoriaResultadoCorreoLlamamiento(DatosAuditoriaResultadoCorreoLlamamiento{ActorID: "actor-declarado", ActorProfile: "perfil-rrhh", VersionRolRef: "rol-version-rrhh-1", AuthMethod: "certificado", AuthAssurance: "alto", Correlacion: correlacion, Solicitud: resultado, OcurridoEn: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)}); err == nil {
		t.Fatal("actor sin HMAC aceptado")
	}
}

func TestSolicitudResultadoCorreoLlamamientoCanonizaJSON10ConEstadoTexto(t *testing.T) {
	solicitud := solicitudCorreoLlamamientoPrueba()
	huella, err := solicitud.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	reserva := ReservaIntentoCorreoLlamamiento{IntentoRef: "intento-001", MessageID: "<m@vec.invalid>", FechaOrigen: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC), SolicitudHuella: huella, Estado: CorreoLlamamientoIniciado}
	for estado, texto := range map[EstadoCorreoLlamamiento]string{CorreoLlamamientoNoAceptadoTransitorio: "no_aceptado_transitorio", CorreoLlamamientoNoAceptadoPermanente: "no_aceptado_permanente", CorreoLlamamientoIndeterminado: "indeterminado", CorreoLlamamientoAceptadoPorRelay: "aceptado_por_relay"} {
		t.Run(texto, func(t *testing.T) {
			resultado, err := NuevaSolicitudRegistrarResultadoCorreoLlamamiento(solicitud, reserva, estado, PlantillaCorreoLlamamientoV1)
			if err != nil {
				t.Fatal(err)
			}
			contenido, err := resultado.SerializarCanonico()
			if err != nil {
				t.Fatal(err)
			}
			esperado := `{"OrganizacionRef":"org-001","ExpedienteRef":"exp-001","LlamamientoRef":"llamamiento-001","ComunicacionRef":"comunicacion-001","IntencionEnvioRef":"intencion-001","IntentoRef":"intento-001","SolicitudHuella":"` + huella + `","Estado":"` + texto + `","PlantillaRef":"llamamiento_rrhh_v1","VersionEsperada":1}`
			if string(contenido) != esperado {
				t.Fatalf("JSON10=%s", contenido)
			}
			hash, err := resultado.HuellaSHA256()
			if err != nil || hash != sha256HexCorreoPrueba([]byte(esperado)) {
				t.Fatalf("huella=%q err=%v", hash, err)
			}
		})
	}
}

func sha256HexCorreoPrueba(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}
