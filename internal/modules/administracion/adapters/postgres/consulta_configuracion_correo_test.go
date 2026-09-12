package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	adminapp "vec-diputacion-granada/internal/modules/administracion/application"
	admindomain "vec-diputacion-granada/internal/modules/administracion/domain"
	adminports "vec-diputacion-granada/internal/modules/administracion/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Estas pruebas usan el callback pgx local: prueban orden, límites y la
// frontera de entrega, pero no sustituyen la comprobación PostgreSQL de ACL,
// V3 ni T13/3 que pertenece a la migración en un clúster real.
func TestConsultaConfiguracionCorreoAuditadaConfirmaAntesDeEntregar(t *testing.T) {
	preparada := auditoriaConsultaCorreoPrueba()
	vista := vistaConsultaCorreoPrueba(7)
	recibo := reciboConsultaCorreoPrueba(preparada, 7)
	tx := &transaccionConfiguracionCorreoFlujoPrueba{fila: respuestaConsultaCorreoPrueba(t, vista, recibo)}
	pool := &iniciadorConfiguracionCorreoFlujoPrueba{tx: tx}
	r := &ConsultaConfiguracionCorreoPostgreSQL{pool: pool}
	orden := ordenConsultaCorreoPrueba(t, preparada)
	original := bytes.Clone(orden.Preparacion.PayloadNegocio)
	resultado, err := r.ConsultarConfiguracionCorreoAuditada(context.Background(), orden)
	if err != nil {
		t.Fatal(err)
	}
	if resultado.Vista.Version != 7 || resultado.ReciboAuditoria.ID != recibo.ID || tx.commits != 1 || tx.consultas != 1 || len(pool.opciones) != 1 || pool.opciones[0] != (pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite}) || len(tx.argumentos) != 9 || !bytes.Equal(orden.Preparacion.PayloadNegocio, original) {
		t.Fatalf("consulta no quedó ligada a commit/material: resultado=%+v consultas=%d commits=%d opciones=%#v argumentos=%d", resultado, tx.consultas, tx.commits, pool.opciones, len(tx.argumentos))
	}
}

func TestConsultaConfiguracionCorreoAuditadaNoEntregaSiFallaCommit(t *testing.T) {
	preparada := auditoriaConsultaCorreoPrueba()
	tx := &transaccionConfiguracionCorreoFlujoPrueba{fila: respuestaConsultaCorreoPrueba(t, vistaConsultaCorreoPrueba(7), reciboConsultaCorreoPrueba(preparada, 7)), errCommit: errors.New("commit caído")}
	r := &ConsultaConfiguracionCorreoPostgreSQL{pool: &iniciadorConfiguracionCorreoFlujoPrueba{tx: tx}}
	orden := ordenConsultaCorreoPrueba(t, preparada)
	original := bytes.Clone(orden.Preparacion.PayloadNegocio)
	if resultado, err := r.ConsultarConfiguracionCorreoAuditada(context.Background(), orden); !errors.Is(err, ErrConfiguracionCorreoNoDisponible) || resultado.Vista.Configurada || resultado.ReciboAuditoria.ID != "" || tx.commits != 1 || !bytes.Equal(orden.Preparacion.PayloadNegocio, original) {
		t.Fatalf("fallo de commit expuso datos: resultado=%+v err=%v commits=%d", resultado, err, tx.commits)
	}
}

func TestConsultaConfiguracionCorreoAuditadaConservaPayloadDelLlamadorSiFallaConsulta(t *testing.T) {
	preparada := auditoriaConsultaCorreoPrueba()
	tx := &transaccionConfiguracionCorreoFlujoPrueba{errScan: errors.New("consulta caída")}
	r := &ConsultaConfiguracionCorreoPostgreSQL{pool: &iniciadorConfiguracionCorreoFlujoPrueba{tx: tx}}
	orden := ordenConsultaCorreoPrueba(t, preparada)
	original := bytes.Clone(orden.Preparacion.PayloadNegocio)
	if _, err := r.ConsultarConfiguracionCorreoAuditada(context.Background(), orden); !errors.Is(err, ErrConfiguracionCorreoNoDisponible) || !bytes.Equal(orden.Preparacion.PayloadNegocio, original) {
		t.Fatalf("el fallo de consulta alteró el payload del llamador: err=%v payload=%q", err, orden.Preparacion.PayloadNegocio)
	}
}

func TestConsultaConfiguracionCorreoAuditadaRechazaReciboConVersionCruzada(t *testing.T) {
	preparada := auditoriaConsultaCorreoPrueba()
	tx := &transaccionConfiguracionCorreoFlujoPrueba{fila: respuestaConsultaCorreoPrueba(t, vistaConsultaCorreoPrueba(7), reciboConsultaCorreoPrueba(preparada, 6))}
	r := &ConsultaConfiguracionCorreoPostgreSQL{pool: &iniciadorConfiguracionCorreoFlujoPrueba{tx: tx}}
	if resultado, err := r.ConsultarConfiguracionCorreoAuditada(context.Background(), ordenConsultaCorreoPrueba(t, preparada)); !errors.Is(err, ErrConfiguracionCorreoNoDisponible) || resultado.Vista.Configurada || tx.commits != 0 {
		t.Fatalf("versión auditada cruzada llegó al llamador: resultado=%+v err=%v commits=%d", resultado, err, tx.commits)
	}
}

func TestConsultaConfiguracionCorreoAuditadaRechazaCamposSecretosYNoAbreTx(t *testing.T) {
	preparada := auditoriaConsultaCorreoPrueba()
	orden := ordenConsultaCorreoPrueba(t, preparada)
	orden.Preparacion.PayloadNegocio = append(orden.Preparacion.PayloadNegocio[:len(orden.Preparacion.PayloadNegocio)-1], []byte(`,"secreto":"prohibido"}`)...)
	pool := &iniciadorConfiguracionCorreoFlujoPrueba{tx: &transaccionConfiguracionCorreoFlujoPrueba{}}
	r := &ConsultaConfiguracionCorreoPostgreSQL{pool: pool}
	if _, err := r.ConsultarConfiguracionCorreoAuditada(context.Background(), orden); !errors.Is(err, ErrConfiguracionCorreoNoDisponible) || pool.inicios != 0 {
		t.Fatalf("payload con secreto abrió una transacción: err=%v inicios=%d", err, pool.inicios)
	}
}

func TestResultadoConsultaCorreoRechazaSecretoEnVistaYAusenciaVersionCero(t *testing.T) {
	preparada := auditoriaConsultaCorreoPrueba()
	base := respuestaConsultaCorreoPrueba(t, admindomain.VistaConfiguracionCorreo{}, reciboConsultaCorreoPrueba(preparada, 0))
	if _, err := resultadoConsultaConfiguracionCorreoDesdeJSON(base, preparada); err != nil {
		t.Fatalf("ausencia version cero válida rechazada: %v", err)
	}
	var sobre map[string]any
	if err := json.Unmarshal(base, &sobre); err != nil {
		t.Fatal(err)
	}
	sobre["configuracion"].(map[string]any)["secreto"] = "prohibido"
	conSecreto, _ := json.Marshal(sobre)
	if _, err := resultadoConsultaConfiguracionCorreoDesdeJSON(conSecreto, preparada); err == nil {
		t.Fatal("vista con secreto aceptada")
	}
}

func ordenConsultaCorreoPrueba(t *testing.T, auditoria vecdomain.AuditEntry) adminports.OrdenConsultaConfiguracionCorreoAutorizada {
	t.Helper()
	payload, err := adminapp.PayloadConsultaConfiguracionCorreo(auditoria)
	if err != nil {
		t.Fatal(err)
	}
	return adminports.OrdenConsultaConfiguracionCorreoAutorizada{Preparacion: adminports.PreparacionConsultaConfiguracionCorreo{Auditoria: auditoria, PayloadNegocio: payload}, Material: materialConfiguracionCorreoFlujoPrueba()}
}

func auditoriaConsultaCorreoPrueba() vecdomain.AuditEntry {
	return vecdomain.AuditEntry{ActorID: "hmac-sha256:bolsa_accesos_v1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ActorProfile: "perfil_admin_sintetico", ActorRoles: []string{"admin_sintetico"}, AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh, Purpose: "administrar_integraciones", Action: "administracion.configuracion_correo.consultar", ModuleID: "vec.module.administracion", SubjectRef: referenciaConfiguracionCorreo, Result: "permitido", CorrelationRef: "correlacion_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", OccurredAt: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)}
}

func reciboConsultaCorreoPrueba(preparada vecdomain.AuditEntry, version int) vecdomain.AuditEntry {
	r := preparada
	r.ID = "acc_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	r.Seq = 8
	r.ObjectVersion = version
	r.AuthorizationRef = "decision_sintetica"
	r.Metadata = map[string]string{"consumo_ref": "consumo_sintetico"}
	r.IntegrityAlgorithm = "sha256-chain-v1"
	r.PrevSignature = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	r.Signature = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	return r
}

func vistaConsultaCorreoPrueba(version uint64) admindomain.VistaConfiguracionCorreo {
	return admindomain.VistaConfiguracionCorreo{Configurada: true, Host: "smtp.intranet.local", Puerto: 465, NombreServidor: "smtp.intranet.local", ReferenciaCA: "ca:correo-interno:v1", RemitenteFijo: "rrhh@diputacion.example", Usuario: "rrhh-smtp", ModoTLS: admindomain.ModoTLSCorreoImplicito, ModoAutenticacion: admindomain.ModoAutenticacionCorreoXOAUTH2, TiempoMaximoMillis: 5000, SecretoConfigurado: true, Version: version}
}

func respuestaConsultaCorreoPrueba(t *testing.T, vista admindomain.VistaConfiguracionCorreo, recibo vecdomain.AuditEntry) []byte {
	t.Helper()
	b, err := json.Marshal(struct {
		Configuracion admindomain.VistaConfiguracionCorreo `json:"configuracion"`
		Auditoria     vecdomain.AuditEntry                 `json:"auditoria"`
	}{vista, recibo})
	if err != nil {
		t.Fatal(err)
	}
	return b
}
