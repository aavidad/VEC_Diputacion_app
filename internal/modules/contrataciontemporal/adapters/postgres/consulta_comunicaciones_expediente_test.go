package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
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
