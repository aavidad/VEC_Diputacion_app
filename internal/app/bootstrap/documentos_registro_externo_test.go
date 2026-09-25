package bootstrap

import (
	"strings"
	"testing"

	core "vec-diputacion-granada/internal/vec/domain"
)

func TestRegistroExternoDocumentosEsOpcionalYExigeMotivoDelMismoCatalogo(t *testing.T) {
	var ausente *registroExternoDocumentosDesarrollo
	if !ausente.valido("vec.autorizacion.motivos") {
		t.Fatal("sin registro externo el material sigue siendo válido")
	}
	rutas, err := rutaRegistroExternoDocumentos(nil, nil, autoridadConsultaDocumentos{}, nil, nil)
	if err != nil || len(rutas) != 0 {
		t.Fatalf("sin configuración no se publica la ruta: %v %d", err, len(rutas))
	}
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "vec.autorizacion.motivos", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "registrar_documento_externo"}
	admitidos := []admitidoRegistroExternoDesarrollo{{PrefijoTipo: "contratacion_temporal.formalizacion.", ModuloID: "contratacion_temporal", CustodioID: "registro_entrada"}}
	valida := &registroExternoDocumentosDesarrollo{Motivo: motivo, Admitidos: admitidos}
	if !valida.valido("vec.autorizacion.motivos") {
		t.Fatal("configuración completa rechazada")
	}
	for nombre, c := range map[string]*registroExternoDocumentosDesarrollo{
		"otro catalogo": {Motivo: motivo, Admitidos: admitidos},
		"sin admitidos": {Motivo: motivo},
		"sin motivo":    {Admitidos: admitidos},
	} {
		catalogo := "vec.autorizacion.motivos"
		if nombre == "otro catalogo" {
			catalogo = "otro.catalogo"
		}
		if c.valido(catalogo) {
			t.Errorf("%s aceptada", nombre)
		}
	}
}
