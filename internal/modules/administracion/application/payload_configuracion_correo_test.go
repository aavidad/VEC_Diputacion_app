package application

import (
	"bytes"
	"strings"
	"testing"
	"time"

	d "vec-diputacion-granada/internal/modules/administracion/domain"
	p "vec-diputacion-granada/internal/modules/administracion/ports"
	v "vec-diputacion-granada/internal/vec/domain"
)

func TestPayloadNegocioConfiguracionCorreoVectorCanonico(t *testing.T) {
	entrada := actualizacionCorreo(nil, 4)
	auditoria := v.AuditEntry{ID: "audit-1", Seq: 7, ActorID: "admin", Action: "administracion.configuracion_correo.actualizar", ModuleID: "vec.module.administracion", SubjectRef: "configuracion:smtp:diputacion", Result: "accepted", CorrelationRef: "correo-1", OccurredAt: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC), Signature: "firma"}
	preparacion := p.PreparacionConfiguracionCorreo{Entrada: entrada}
	obtenido, err := PayloadNegocioConfiguracionCorreo(preparacion, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	const esperado = `{"esquema":"vec.administracion.configuracion-correo.persistencia.v1","configuracion":{"host":"smtp.intranet.local","puerto":465,"server_name":"smtp.intranet.local","referencia_ca":"ca:v1","remitente_fijo":"rrhh@diputacion.example","usuario":"rrhh","modo_tls":"tls_implicito","modo_autenticacion":"xoauth2","tiempo_maximo_ms":1,"version_esperada":4},"sobre_secreto":null,"auditoria":{"id":"audit-1","seq":7,"actor_id":"admin","action":"administracion.configuracion_correo.actualizar","module_id":"vec.module.administracion","subject_ref":"configuracion:smtp:diputacion","result":"accepted","correlation_ref":"correo-1","occurred_at":"2026-09-12T12:00:00Z","signature":"firma"}}`
	if string(obtenido) != esperado {
		t.Fatalf("vector cambió:\n%s", obtenido)
	}
	repetido, err := PayloadNegocioConfiguracionCorreo(preparacion, auditoria)
	if err != nil || !bytes.Equal(obtenido, repetido) {
		t.Fatal("bytes no estables")
	}
}

func TestPayloadNegocioConfiguracionCorreoRechazaSecretoClaro(t *testing.T) {
	secreto, err := d.NuevoSecretoCorreo([]byte("secreto-sintetico"))
	if err != nil {
		t.Fatal(err)
	}
	entrada := actualizacionCorreo(&secreto, 4)
	_, err = PayloadNegocioConfiguracionCorreo(p.PreparacionConfiguracionCorreo{Entrada: entrada}, v.AuditEntry{ActorID: "admin", Action: "x", ModuleID: "m", SubjectRef: "s", Result: "accepted", OccurredAt: time.Now().UTC()})
	if err == nil {
		t.Fatal("se aceptó secreto claro")
	}
}

func TestPayloadNegocioConfiguracionCorreoVectorSobre(t *testing.T) {
	entrada := actualizacionCorreo(nil, 4)
	preparacion := p.PreparacionConfiguracionCorreo{Entrada: entrada, Sustituir: true, SobreNuevo: p.SobreSecretoConfiguracionCorreo{Version: 5, ClaveRef: "kms:v1", Nonce: []byte{1}, Cifrado: []byte("cifrado")}, HuellaAADSHA256: strings.Repeat("a", 64)}
	payload, err := PayloadNegocioConfiguracionCorreo(preparacion, v.AuditEntry{ActorID: "a", Action: "x", ModuleID: "m", SubjectRef: "s", Result: "accepted", OccurredAt: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	const esperado = `"sobre_secreto":{"version":5,"clave_ref":"kms:v1","nonce":"AQ==","cifrado":"Y2lmcmFkbw==","huella_aad_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	if !bytes.Contains(payload, []byte(esperado)) {
		t.Fatal("representación canónica del sobre alterada")
	}
	preparacion.SobreNuevo.Version = 6
	if _, err := PayloadNegocioConfiguracionCorreo(preparacion, v.AuditEntry{ActorID: "a", Action: "x", ModuleID: "m", SubjectRef: "s", Result: "accepted", OccurredAt: time.Now().UTC()}); err == nil {
		t.Fatal("sobre no ligado a versión admitido")
	}
}
