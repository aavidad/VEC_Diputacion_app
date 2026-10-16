package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type proveedorComunicacionesExpedientePrueba func(context.Context, ports.ConsultaComunicacionesExpediente) (AmbitoConsultaComunicacionesExpediente, error)

func (p proveedorComunicacionesExpedientePrueba) AutorizarConsultaComunicacionesExpediente(ctx context.Context, c ports.ConsultaComunicacionesExpediente) (AmbitoConsultaComunicacionesExpediente, error) {
	return p(ctx, c)
}

func TestConsultaComunicacionesExpedienteSelecPendienteConfirmaAntesDeExponer(t *testing.T) {
	c := ports.ConsultaComunicacionesExpediente{ExpedienteRef: "expediente:ct231", Limite: 10}
	org := "organizacion:ct231"
	a := autorizacionComunicacionesExpedientePrueba(t, c, org)
	proveedor := proveedorComunicacionesExpedientePrueba(func(_ context.Context, _ ports.ConsultaComunicacionesExpediente) (AmbitoConsultaComunicacionesExpediente, error) {
		return a, nil
	})
	pagina := ports.PaginaComunicacionesExpediente{ExpedienteRef: c.ExpedienteRef,
		Comunicaciones: []ports.ComunicacionExpediente{{
			OrganizacionRef: org, ExpedienteRef: c.ExpedienteRef,
			LlamamientoRef: "llamamiento:ct231", Estado: ports.EstadoSeleccionConfirmadaPendienteAviso,
			Version: 1, RegistradaEn: time.Date(2026, 10, 16, 4, 4, 0, 123456000, time.UTC),
			AntecedenteTipo: "seleccion_confirmada", ReciboAntecedenteRef: "recibo:ct231",
		}}}
	contenido, err := json.Marshal(struct {
		Encontrado bool `json:"encontrado"`
		ports.PaginaComunicacionesExpediente
	}{true, pagina})
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre string
		raw    []byte
		ok     bool
	}{
		{"pendiente", contenido, true},
		{"mezcla_ct54", append([]byte(nil), []byte(strings.Replace(string(contenido), `"version":1`, `"version":1,"comunicacion_ref":"comunicacion:falsa"`, 1))...), false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &transaccionEjecucionSeleccionO6Prueba{fila: filaEjecucionSeleccionO6Prueba{valores: []any{string(caso.raw)}}}
			lector := &LectorComunicacionesExpedientePostgreSQL{pool: &iniciadorEjecucionSeleccionO6Prueba{tx: tx}, proveedor: proveedor}
			salida, err := lector.ConsultarComunicacionesExpediente(context.Background(), c)
			if caso.ok {
				if err != nil || tx.confirmaciones != 1 || len(salida.Comunicaciones) != 1 ||
					salida.Comunicaciones[0].Estado != ports.EstadoSeleccionConfirmadaPendienteAviso {
					t.Fatalf("fila válida no confirmada: %v, commits=%d", err, tx.confirmaciones)
				}
			} else if !errors.Is(err, ports.ErrResultadoComunicacionesExpedienteNoConfiable) || tx.confirmaciones != 0 {
				t.Fatalf("mezcla expuesta: %v, commits=%d", err, tx.confirmaciones)
			}
		})
	}
}

func autorizacionComunicacionesExpedientePrueba(t *testing.T, c ports.ConsultaComunicacionesExpediente, org string) AmbitoConsultaComunicacionesExpediente {
	t.Helper()
	r, err := RecursoConsultaComunicacionesExpediente(c, org)
	if err != nil {
		t.Fatal(err)
	}
	h, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hash := strings.Repeat("a", 64)
	ahora := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:ct140", hash, hash, "contexto:ct140", hash,
		AccionConsultaComunicacionesExpediente, c.ExpedienteRef, h,
		AudienciaConsultaComunicacionesExpediente, ahora, ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public())
	if err != nil {
		t.Fatal(err)
	}
	a, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		[]byte(strings.Repeat("x", 512)), resumen, []byte("{}"), []byte("{}"), []byte("{}"),
		1, 1, []byte("unidad"), []byte("unidad"), []byte("unidad"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return AmbitoConsultaComunicacionesExpediente{OrganizacionRef: org, Autorizacion: a}
}

func TestConsultaComunicacionesExpediente404ConfirmaConsumo(t *testing.T) {
	c := ports.ConsultaComunicacionesExpediente{ExpedienteRef: "expediente:ct140", Limite: 10}
	org := "organizacion:ct140"
	a := autorizacionComunicacionesExpedientePrueba(t, c, org)
	proveedor := proveedorComunicacionesExpedientePrueba(func(_ context.Context, entrada ports.ConsultaComunicacionesExpediente) (AmbitoConsultaComunicacionesExpediente, error) {
		if entrada != c {
			t.Fatal("consulta cambiada")
		}
		return a, nil
	})
	casos := []struct {
		nombre, salida string
		confirmaciones int
		errorEsperado  error
	}{
		{"ausencia", "{\"encontrado\":false,\"expediente_ref\":\"expediente:ct140\",\"comunicaciones\":[],\"siguiente_cursor\":\"\"}", 1, ports.ErrConsultaComunicacionesExpedienteNoEncontrado},
		{"ausencia_con_datos", "{\"encontrado\":false,\"expediente_ref\":\"expediente:ct140\",\"comunicaciones\":[{}],\"siguiente_cursor\":\"\"}", 0, ports.ErrResultadoComunicacionesExpedienteNoConfiable},
		{"sin_marca", "{\"expediente_ref\":\"expediente:ct140\",\"comunicaciones\":[]}", 0, ports.ErrResultadoComunicacionesExpedienteNoConfiable},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &transaccionEjecucionSeleccionO6Prueba{fila: filaEjecucionSeleccionO6Prueba{valores: []any{caso.salida}}}
			l := &LectorComunicacionesExpedientePostgreSQL{pool: &iniciadorEjecucionSeleccionO6Prueba{tx: tx}, proveedor: proveedor}
			p, err := l.ConsultarComunicacionesExpediente(context.Background(), c)
			if !errors.Is(err, caso.errorEsperado) || p.ExpedienteRef != "" || tx.confirmaciones != caso.confirmaciones {
				t.Fatalf("resultado=%+v error=%v commits=%d", p, err, tx.confirmaciones)
			}
		})
	}
}
