package postgres

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestRecursoCambioOrganizacionLigaMaterialYReferenciaCore(t *testing.T) {
	tm := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	c := core.CatalogoConfigurable{ID: personalports.IDCatalogoOrganizacion, Version: 1, Revision: 2, ModuloID: "personal", Nombre: "Estructura", FuenteRef: "fuente:dipgra", MotivoCreacion: "inicial", Estado: core.EstadoCatalogoBorrador, CreadoPor: "actor:rrhh:001", CreadoEn: tm, UltimaModificacionPor: "actor:rrhh:001", UltimaModificacionEn: tm, MotivoModificacion: "motivo", Entradas: []core.EntradaCatalogoConfigurable{{Clave: "local-018f47a2-6b31-4c80-8a95-4d2e707c5a11", Etiqueta: "Centro", Orden: 0, VigenteDesde: tm, Atributos: map[string]string{"tipo": "centro", "modificada_localmente": "si"}}}}
	canon, err := c.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	material := personalports.MaterialCambioOrganizacion{Solicitud: personalports.SolicitudCambioOrganizacion{CatalogoVersion: 1, CatalogoRevision: 1, HuellaEsperada: strings.Repeat("a", 64), ClaveIdempotencia: "018f47a2-6b31-4c80-8a95-4d2e707c5a11", Unidad: personalports.UnidadCambioOrganizacion{Clave: "local-018f47a2-6b31-4c80-8a95-4d2e707c5a11", Etiqueta: "Centro", Tipo: "centro"}, Motivo: "motivo"}, ActorID: "actor:rrhh:001"}
	b, err := json.Marshal(canon)
	material.CatalogoCanonico = string(b)
	if err != nil {
		t.Fatal(err)
	}
	r, err := RecursoCambioOrganizacion(material)
	if err != nil {
		t.Fatal(err)
	}
	if r.Referencia != "estructura-organizativa-dipgra:1" || r.ModuloID != "personal" || r.Tipo != personalports.TipoCambioOrganizacion || r.Ambitos["organizacion_ref"] != "organizacion:desarrollo:dipgra" || r.Atributos["material_sha256"] == "" {
		t.Fatalf("recurso incorrecto: %#v", r)
	}
}
