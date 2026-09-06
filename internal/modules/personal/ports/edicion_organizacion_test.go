package ports

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

func catalogoOrganizacionPuertoPrueba() core.CatalogoConfigurable {
	t := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	return core.CatalogoConfigurable{ID: IDCatalogoOrganizacion, Version: 1, Revision: 2, ModuloID: "personal", Nombre: "Estructura", FuenteRef: "fuente:dipgra", MotivoCreacion: "inicial", Estado: core.EstadoCatalogoBorrador, CreadoPor: "actor:rrhh:001", CreadoEn: t, UltimaModificacionPor: "actor:rrhh:001", UltimaModificacionEn: t, MotivoModificacion: "ajuste", Entradas: []core.EntradaCatalogoConfigurable{{Clave: "centro-1", Etiqueta: "Centro 1", Orden: 0, VigenteDesde: t, Atributos: map[string]string{"tipo": "centro"}}}}
}

func TestMaterialCambioOrganizacionConservaCanonYRechazaExtras(t *testing.T) {
	c := catalogoOrganizacionPuertoPrueba()
	canon, err := c.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(canon)
	if err != nil {
		t.Fatal(err)
	}
	s := SolicitudCambioOrganizacion{CatalogoVersion: 1, CatalogoRevision: 2, HuellaEsperada: strings.Repeat("a", 64), ClaveIdempotencia: "018f47a2-6b31-4c80-8a95-4d2e707c5a11", Unidad: UnidadCambioOrganizacion{Clave: "local-018f47a2-6b31-4c80-8a95-4d2e707c5a11", Etiqueta: "Centro nuevo", Tipo: "centro"}, Motivo: "cambio"}
	m := MaterialCambioOrganizacion{Solicitud: s, ActorID: "actor:rrhh:001", CatalogoCanonico: string(b)}
	obtenido, err := m.Catalogo()
	if err != nil || obtenido.Referencia() != "estructura-organizativa-dipgra:1" {
		t.Fatalf("catalogo: %v %#v", err, obtenido)
	}
	m.CatalogoCanonico = string(b) + " "
	if _, err := m.Catalogo(); err == nil {
		t.Fatal("acepto canon con bytes adicionales")
	}
}

func TestSolicitudCambioOrganizacionValidaUUIDV4Canonica(t *testing.T) {
	s := SolicitudCambioOrganizacion{CatalogoVersion: 1, CatalogoRevision: 1, HuellaEsperada: strings.Repeat("a", 64), ClaveIdempotencia: "018f47a2-6b31-4c80-8a95-4d2e707c5a11", Unidad: UnidadCambioOrganizacion{Clave: "local-018f47a2-6b31-4c80-8a95-4d2e707c5a11", Etiqueta: "Centro", Tipo: "centro"}, Motivo: "motivo"}
	if err := s.Validar(); err != nil {
		t.Fatal(err)
	}
	// Las unidades existentes mantienen su clave pública; no se recrean como
	// unidades locales para poder cambiar una etiqueta o adscripción.
	s.Unidad.Clave = "centro-520"
	if err := s.Validar(); err != nil {
		t.Fatalf("rechazó la edición de una unidad existente: %v", err)
	}
	s.ClaveIdempotencia = strings.ToUpper(s.ClaveIdempotencia)
	if err := s.Validar(); err == nil {
		t.Fatal("acepto UUID no canonica")
	}
}
