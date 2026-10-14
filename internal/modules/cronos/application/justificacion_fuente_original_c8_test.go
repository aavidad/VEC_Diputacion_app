package application

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type capturaOriginalC8Prueba struct {
	captura ports.CapturaLecturaOriginalC8
}

func (c capturaOriginalC8Prueba) CapturarLecturaOriginalPropia(context.Context, ports.OrdenOriginalC8, string) (ports.CapturaLecturaOriginalC8, error) {
	return c.captura, nil
}
func (c capturaOriginalC8Prueba) CapturarLecturaOriginalRRHH(context.Context, ports.OrdenOriginalC8, string) (ports.CapturaLecturaOriginalC8, error) {
	return c.captura, nil
}
func (c capturaOriginalC8Prueba) CapturarBandejaOriginal(context.Context, ports.OrdenOriginalC8) (ports.CapturaLecturaOriginalC8, error) {
	return c.captura, nil
}

type lectorFuenteOriginalC8Prueba struct{ lectura ports.LecturaOriginalC8 }

func (l lectorFuenteOriginalC8Prueba) ConsultarOriginal(context.Context, ports.CapturaLecturaOriginalC8, string) (ports.LecturaOriginalC8, error) {
	return l.lectura, nil
}
func (l lectorFuenteOriginalC8Prueba) ConsultarBandejaOriginal(context.Context, ports.CapturaLecturaOriginalC8) (ports.BandejaOriginalC8, error) {
	return ports.BandejaOriginalC8{}, ports.ErrJustificacionNoDisponible
}

func TestFuenteOriginalC8RecuperaHistoriaExactaSinAceptarTitularAjeno(t *testing.T) {
	_, vieja, _, _, _, _, peticion := escenarioJustificacion(t)
	actor, err := vieja.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	empleados, err := actor.Referencias("empleado")
	if err != nil || len(empleados) != 1 {
		t.Fatal("empleado no acreditado")
	}
	orden, err := ports.NuevaOrdenOriginalC8(actor, &proveedorOriginalC8Prueba{})
	if err != nil {
		t.Fatal(err)
	}
	ref := peticion.SolicitudRef
	capturaJSON, _ := json.Marshal(map[string]any{"accion": "cronos.justificacion.propia.consultar", "registro_contexto_ref": "rca_" + strings.Repeat("C", 22)})
	captura := capturaOriginalC8Prueba{ports.CapturaLecturaOriginalC8{Orden: capturaJSON}}
	base := ports.LecturaOriginalC8{SolicitudRef: ref, EmpleadoRef: empleados[0], PersonaTitularRef: actor.PersonaRef,
		EstadoPermiso: domain.EstadoPermisoConcedido, VersionPermiso: 3, PendienteJustificar: true, JustificanteExigido: true,
		Historia: []ports.HechoOriginalC8{{Version: 1, Estado: domain.JustificacionPendiente,
			DocumentoID: "ref:" + strings.Repeat("a", 64), DocumentoVersion: 1, SHA256: strings.Repeat("b", 64),
			ReciboDocumentoRef: "recibo:documentos:abc", CustodioID: "almacen.local",
			ReciboRef: "recibo:cronos:00000000-0000-4000-8000-000000000001", InstanteUTC: time.Now().UTC().Truncate(time.Microsecond)}},
	}
	f, err := NuevaFuenteSolicitudOriginalC8(captura, lectorFuenteOriginalC8Prueba{base})
	if err != nil {
		t.Fatal(err)
	}
	s, err := f.PrepararOriginal(context.Background(), orden, ref, ports.LecturaOriginalPropia)
	if err != nil || s.Actual == nil || s.Actual.Version != 1 || s.Actual.Original.DocumentoID != base.Historia[0].DocumentoID {
		t.Fatalf("historia original: %+v, %v", s, err)
	}
	base.Historia[0].Version = 2
	f, _ = NuevaFuenteSolicitudOriginalC8(captura, lectorFuenteOriginalC8Prueba{base})
	if _, err = f.PrepararOriginal(context.Background(), orden, ref, ports.LecturaOriginalPropia); err == nil {
		t.Fatal("aceptó una historia con versión ausente")
	}
	base.Historia[0].Version = 1
	base.PersonaTitularRef = "per_" + strings.Repeat("Z", 22)
	f, _ = NuevaFuenteSolicitudOriginalC8(captura, lectorFuenteOriginalC8Prueba{base})
	if _, err = f.PrepararOriginal(context.Background(), orden, ref, ports.LecturaOriginalPropia); err == nil {
		t.Fatal("aceptó un titular distinto del actor propio")
	}
}
