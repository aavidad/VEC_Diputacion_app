package auditoria

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"strings"
	"testing"
	"time"

	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridad "vec-diputacion-granada/internal/vec/adapters/documentos/seguridad"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const contextoActorAuditoriaPrueba = `{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":"per_rrrrrrrrrrrrrrrrrrrrrr","metodo":"certificado","garantia":"alto","perfil_activo_ref":"prf_pppppppppppppppppppppp","persona_ref":"per_rrrrrrrrrrrrrrrrrrrrrr","contexto_actor_ref":"vca_vvvvvvvvvvvvvvvvvvvvvv","contexto_version":3,"cuenta_ref":"cta_aaaaaaaaaaaaaaaaaaaaaa","cuenta_version":6,"persona_version":4,"perfil_version":5,"estado":"activo","vigente_desde":"2026-07-15T09:30:00.123000Z","vigente_hasta":"2026-07-15T11:30:00.123000Z","resuelto_en":"2026-07-15T10:30:00.123000Z","vinculos":[{"vinculo_ref":"vin_cccccccccccccccccccccc","version":7,"tipo":"candidato","referencia":"can_cccccccccccccccccccccc","estado":"activo","vigente_desde":"2026-07-15T09:30:00.123000Z","vigente_hasta":"2026-07-15T11:30:00.123000Z"},{"vinculo_ref":"vin_eeeeeeeeeeeeeeeeeeeeee","version":9,"tipo":"empleado","referencia":"emp_eeeeeeeeeeeeeeeeeeeeee","estado":"activo","vigente_desde":"2026-07-15T09:30:00.123000Z","vigente_hasta":"2026-07-15T11:30:00.123000Z"}]}`

type lectorAtestacionPrueba struct {
	versionRol, correlacion string
	err                     error
}

func (l lectorAtestacionPrueba) Referencias([]byte) (string, string, error) {
	return l.versionRol, l.correlacion, l.err
}

func solicitudAuditoriaPrueba(t *testing.T) (ports.SolicitudDespacharCorreoLlamamiento, ports.SolicitudRegistrarResultadoCorreoLlamamiento) {
	t.Helper()
	despacho := ports.SolicitudDespacharCorreoLlamamiento{OrganizacionRef: "org-001", ExpedienteRef: "exp-001", LlamamientoRef: "llamamiento-001", ComunicacionRef: "comunicacion-001", IntencionEnvioRef: "intencion-001"}
	huella, err := despacho.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	reserva := ports.ReservaIntentoCorreoLlamamiento{IntentoRef: "intento-001", MessageID: "<intento-001@vec.invalid>", FechaOrigen: time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC), SolicitudHuella: huella, Estado: ports.CorreoLlamamientoIniciado}
	resultado, err := ports.NuevaSolicitudRegistrarResultadoCorreoLlamamiento(despacho, reserva, ports.CorreoLlamamientoAceptadoPorRelay, ports.PlantillaCorreoLlamamientoV1)
	if err != nil {
		t.Fatal(err)
	}
	return despacho, resultado
}

func capacidadDespachoAuditoriaPrueba(t *testing.T, s ports.SolicitudDespacharCorreoLlamamiento, contexto []byte) ports.CapacidadDespachoCorreoLlamamiento {
	return capacidadDespachoAuditoriaPruebaConVersiones(t, s, contexto, 4, 5)
}

func capacidadDespachoAuditoriaPruebaConVersiones(t *testing.T, s ports.SolicitudDespacharCorreoLlamamiento, contexto []byte, personaVersion, perfilVersion uint64) ports.CapacidadDespachoCorreoLlamamiento {
	t.Helper()
	recurso, err := ctapplication.RecursoDespachoCorreoLlamamiento(s)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	vencimiento := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision-001", strings.Repeat("a", 64), strings.Repeat("a", 64), "contexto-001", strings.Repeat("a", 64), ctapplication.AccionDespacharCorreoLlamamiento, s.IntencionEnvioRef, huella, ctapplication.AudienciaDespachoCorreoLlamamientoV3, vencimiento.Add(-5*time.Second), vencimiento)
	if err != nil {
		t.Fatal(err)
	}
	clave := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(clave.Public())
	if err != nil {
		t.Fatal(err)
	}
	material, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{5}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen, []byte("decision"), []byte("motivo"), contexto, personaVersion, perfilVersion, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := ctapplication.NuevaCapacidadDespachoCorreoLlamamiento(s, material, vencimiento.Add(-time.Second))
	if err != nil {
		t.Fatal(err)
	}
	return capacidad
}

func TestPreparadorAuditoriaResultadoCorreoLlamamientoUsaHMACComunYCorrelacionNueva(t *testing.T) {
	despacho, resultado := solicitudAuditoriaPrueba(t)
	seudonimizador, err := seguridad.NuevoSelladorHMAC("prueba", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	preparador, err := nuevoPreparadorResultadoCorreoLlamamiento(seudonimizador, seguridadvec.GeneradorReferenciasCriptograficas{}, lectorAtestacionPrueba{versionRol: "rol-version-rrhh-1", correlacion: "correlacion_despacho_0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	auditoria, err := preparador.PrepararAuditoriaResultadoCorreoLlamamiento(context.Background(), resultado, capacidadDespachoAuditoriaPrueba(t, despacho, []byte(contextoActorAuditoriaPrueba)), time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	canonico, err := auditoria.SerializarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(canonico), `"actor_id":"hmac-sha256:prueba:`) || !strings.Contains(string(canonico), `"actor_profile":"prf_pppppppppppppppppppppp"`) || !strings.Contains(string(canonico), `"actor_roles":["rol-version-rrhh-1"]`) || !strings.Contains(string(canonico), `"occurred_at":"2026-09-12T10:00:00.000000Z"`) {
		t.Fatalf("AuditEntry incompleto: %s", canonico)
	}
	correlacion, err := auditoria.CorrelacionRef()
	if err != nil || correlacion == "correlacion_despacho_0123456789abcdef0123456789abcdef" || !strings.HasPrefix(correlacion, "correlacion_") {
		t.Fatalf("correlacion=%q err=%v", correlacion, err)
	}
	if _, err := auditoria.HuellaSHA256(); err != nil {
		t.Fatal(err)
	}
}

func TestPreparadorAuditoriaResultadoCorreoLlamamientoDeniegaContextoOAtestacionNoRehidratable(t *testing.T) {
	despacho, resultado := solicitudAuditoriaPrueba(t)
	seudonimizador, err := seguridad.NuevoSelladorHMAC("prueba", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	preparador, err := NuevoPreparadorResultadoCorreoLlamamiento(seudonimizador, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = preparador.PrepararAuditoriaResultadoCorreoLlamamiento(context.Background(), resultado, capacidadDespachoAuditoriaPrueba(t, despacho, []byte(contextoActorAuditoriaPrueba)), time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("payload V3 no atestado aceptado")
	}
	preparador, err = nuevoPreparadorResultadoCorreoLlamamiento(seudonimizador, seguridadvec.GeneradorReferenciasCriptograficas{}, lectorAtestacionPrueba{versionRol: "rol-version-rrhh-1", correlacion: "correlacion_despacho_0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = preparador.PrepararAuditoriaResultadoCorreoLlamamiento(context.Background(), resultado, capacidadDespachoAuditoriaPrueba(t, despacho, []byte(`{}`)), time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("contexto no canónico aceptado")
	}
	if _, err = preparador.PrepararAuditoriaResultadoCorreoLlamamiento(context.Background(), resultado, capacidadDespachoAuditoriaPruebaConVersiones(t, despacho, []byte(contextoActorAuditoriaPrueba), 99, 5), time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("versiones de material y contexto cruzadas aceptadas")
	}
}

func TestPreparadorAuditoriaResultadoCorreoLlamamientoRechazaDependenciasAusentes(t *testing.T) {
	if _, err := NuevoPreparadorResultadoCorreoLlamamiento(nil, seguridadvec.GeneradorReferenciasCriptograficas{}); err == nil {
		t.Fatal("seudonimizador ausente aceptado")
	}
	seudonimizador, err := seguridad.NuevoSelladorHMAC("prueba", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NuevoPreparadorResultadoCorreoLlamamiento(seudonimizador, nil); err == nil {
		t.Fatal("generador ausente aceptado")
	}
	// Evita que la prueba anterior se convierta en una comparación de cadenas
	// prefabricadas: el HMAC procede del adaptador común y cambia con el sujeto.
	uno, _ := seudonimizador.SeudonimizarSujetoAlmacen(context.Background(), debeSolicitudSeudonimo(t, "per_rrrrrrrrrrrrrrrrrrrrrr"))
	dos, _ := seudonimizador.SeudonimizarSujetoAlmacen(context.Background(), debeSolicitudSeudonimo(t, "per_ssssssssssssssssssssss"))
	if uno == dos || !seudonimoHMACValido(uno) {
		t.Fatal("HMAC común no ligado al sujeto")
	}
}

func debeSolicitudSeudonimo(t *testing.T, sujeto string) vecports.SolicitudSeudonimizarSujetoAlmacen {
	t.Helper()
	s, err := vecports.NuevaSolicitudSeudonimizarSujetoAlmacen(sujeto, ambitoSeudonimizacionAuditoriaResultadoCorreoLlamamiento)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
