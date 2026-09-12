package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	adminapp "vec-diputacion-granada/internal/modules/administracion/application"
	admindomain "vec-diputacion-granada/internal/modules/administracion/domain"
	adminports "vec-diputacion-granada/internal/modules/administracion/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestPayloadConfiguracionCorreoAdministracionV3LigaSobreSinSecretoClaro(t *testing.T) {
	var err error
	aad := sha256.Sum256([]byte("aad-cifrado-sintetico"))
	p := adminports.PreparacionConfiguracionCorreo{
		Entrada: admindomain.ActualizacionConfiguracionCorreo{
			VistaConfiguracionCorreo: admindomain.VistaConfiguracionCorreo{Configurada: true, Host: "smtp.intranet.example", Puerto: 465, NombreServidor: "smtp.intranet.example", ReferenciaCA: "ca:smtp:interna", RemitenteFijo: "noreply@example.test", Usuario: "smtp-admin", ModoTLS: admindomain.ModoTLSCorreoImplicito, ModoAutenticacion: admindomain.ModoAutenticacionCorreoPlain, TiempoMaximoMillis: 5000, SecretoConfigurado: true},
			VersionEsperada:          0,
		},
		Sustituir:       true,
		SobreNuevo:      adminports.SobreSecretoConfiguracionCorreo{Version: 1, ClaveRef: "kms:desarrollo:smtp:v1", Nonce: bytes.Repeat([]byte{1}, 12), Cifrado: bytes.Repeat([]byte{2}, 32)},
		HuellaAADSHA256: hex.EncodeToString(aad[:]),
	}
	p.PayloadNegocio, err = adminapp.PayloadNegocioConfiguracionCorreo(p, auditoriaConfiguracionCorreoV3Prueba())
	if err != nil {
		t.Fatal(err)
	}
	payload, err := payloadPreparadoConfiguracionCorreoV3(p, auditoriaConfiguracionCorreoV3Prueba())
	if err != nil {
		t.Fatalf("preparacion cifrada valida rechazada: %v", err)
	}
	defer borrarBytes(payload)
	if !bytes.Equal(payload, p.PayloadNegocio) || bytes.Contains(payload, []byte("secreto-smtp-sintetico-no-publicable")) {
		t.Fatalf("payload no ligado o expone secreto: %q", payload)
	}
}

func auditoriaConfiguracionCorreoV3Prueba() vecdomain.AuditEntry {
	return vecdomain.AuditEntry{ActorID: "per_admin_sintetico", Action: accionConfiguracionCorreoAdministracionV3, ModuleID: "vec.module.administracion", SubjectRef: referenciaConfiguracionCorreoAdministracionV3, Result: "accepted", CorrelationRef: "configuracion-correo-sintetica", OccurredAt: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)}
}

func TestPayloadConfiguracionCorreoAdministracionV3FallaCerradoAnteCruceDeOperacionSecreta(t *testing.T) {
	for nombre, modificar := range map[string]func(*adminports.PreparacionConfiguracionCorreo){
		"retiene secreto claro": func(p *adminports.PreparacionConfiguracionCorreo) {
			secreto, err := admindomain.NuevoSecretoCorreo([]byte("secreto-sintetico"))
			if err != nil {
				t.Fatal(err)
			}
			p.Entrada.SecretoNuevo = &secreto
		},
		"sobre sin AAD":                       func(p *adminports.PreparacionConfiguracionCorreo) { p.HuellaAADSHA256 = "" },
		"version de secreto no ligada al CAS": func(p *adminports.PreparacionConfiguracionCorreo) { p.SobreNuevo.Version = 2 },
		"omite sustitucion con sobre":         func(p *adminports.PreparacionConfiguracionCorreo) { p.Sustituir = false },
	} {
		t.Run(nombre, func(t *testing.T) {
			p := preparacionConfiguracionCorreoV3Prueba(t)
			modificar(&p)
			if _, err := payloadPreparadoConfiguracionCorreoV3(p, auditoriaConfiguracionCorreoV3Prueba()); err == nil {
				t.Fatal("la preparacion cruzada fue aceptada")
			}
		})
	}
}

func preparacionConfiguracionCorreoV3Prueba(t *testing.T) adminports.PreparacionConfiguracionCorreo {
	t.Helper()
	var err error
	h := sha256.Sum256([]byte("aad-prueba"))
	p := adminports.PreparacionConfiguracionCorreo{Entrada: admindomain.ActualizacionConfiguracionCorreo{VistaConfiguracionCorreo: admindomain.VistaConfiguracionCorreo{Configurada: true, Host: "smtp.example.test", Puerto: 465, NombreServidor: "smtp.example.test", ReferenciaCA: "ca:smtp", RemitenteFijo: "noreply@example.test", Usuario: "smtp", ModoTLS: admindomain.ModoTLSCorreoImplicito, ModoAutenticacion: admindomain.ModoAutenticacionCorreoPlain, TiempoMaximoMillis: 1000, SecretoConfigurado: true}, VersionEsperada: 0}, Sustituir: true, SobreNuevo: adminports.SobreSecretoConfiguracionCorreo{Version: 1, ClaveRef: "kms:smtp:v1", Nonce: bytes.Repeat([]byte{1}, 12), Cifrado: bytes.Repeat([]byte{2}, 16)}, HuellaAADSHA256: hex.EncodeToString(h[:])}
	p.PayloadNegocio, err = adminapp.PayloadNegocioConfiguracionCorreo(p, auditoriaConfiguracionCorreoV3Prueba())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPayloadConfiguracionCorreoAdministracionRechazaPreimagenAjena(t *testing.T) {
	p := preparacionConfiguracionCorreoV3Prueba(t)
	a := auditoriaConfiguracionCorreoV3Prueba()
	if _, err := payloadPreparadoConfiguracionCorreoV3(p, a); err != nil {
		t.Fatal(err)
	}
	otra := preparacionConfiguracionCorreoV3Prueba(t)
	otra.Entrada.Host = "otro.example.test"
	otroPayload, err := adminapp.PayloadNegocioConfiguracionCorreo(otra, a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := payloadPreparadoConfiguracionCorreoV3(adminports.PreparacionConfiguracionCorreo{Entrada: otra.Entrada, SobreNuevo: otra.SobreNuevo, Sustituir: otra.Sustituir, HuellaAADSHA256: otra.HuellaAADSHA256, PayloadNegocio: otroPayload}, a); err != nil {
		t.Fatal("segundo control valido rechazado")
	}
	p.PayloadNegocio = otroPayload
	if _, err := payloadPreparadoConfiguracionCorreoV3(p, a); err == nil {
		t.Fatal("se admite preimagen de otra configuracion")
	}
	p = preparacionConfiguracionCorreoV3Prueba(t)
	a.ActorID = "per_otro_admin"
	if _, err := payloadPreparadoConfiguracionCorreoV3(p, a); err == nil {
		t.Fatal("se admite auditoria de otro actor")
	}
	p.PayloadNegocio = []byte(`{"esquema":"vec.administracion.configuracion-correo.persistencia.v1"}`)
	if _, err := payloadPreparadoConfiguracionCorreoV3(p, a); err == nil {
		t.Fatal("se admite payload incompleto")
	}
}
