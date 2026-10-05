package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func configuracionFuentesLotePrueba() ConfiguracionProveedorAmbitosLote {
	org, unidad := "org_prueba", "unidad:prueba"
	return ConfiguracionProveedorAmbitosLote{OrganizacionRef: org, Unidades: []AmbitosFuenteLote{{
		OrganizacionRef: org, UnidadRef: unidad,
		Descriptores: []DimensionFuenteLote{
			{Dimension: "organizacion_ref", Valores: []string{org}, Fuente: FuenteDescriptorLote{
				Referencia: "fuente:organizacion:1", Version: 1, HuellaSHA256: strings.Repeat("a", 64)}},
			{Dimension: "unidad_ref", Valores: []string{unidad}, Fuente: FuenteDescriptorLote{
				Referencia: "fuente:unidad:1", Version: 1, HuellaSHA256: strings.Repeat("b", 64)}},
		},
	}}}
}

func TestFuenteLotePrivadaFijaOrganizacionYDefiendeCopia(t *testing.T) {
	c := configuracionFuentesLotePrueba()
	p, err := NuevaFuenteAmbitosLotePrivada(c)
	if err != nil {
		t.Fatal(err)
	}
	c.Unidades[0].Descriptores[1].Fuente.HuellaSHA256 = strings.Repeat("c", 64)
	primera, err := p.ResolverUnidadLote(context.Background(), "org_prueba", "unidad:prueba")
	if err != nil || primera.Descriptores[1].Fuente.HuellaSHA256 != strings.Repeat("b", 64) {
		t.Fatal("configuracion mutable altero fuente")
	}
	primera.Descriptores[0].Valores[0] = "org_ajena"
	segunda, err := p.ResolverUnidadLote(context.Background(), "org_prueba", "unidad:prueba")
	if err != nil || segunda.Descriptores[0].Valores[0] != "org_prueba" {
		t.Fatal("copia devuelta altero fuente")
	}
	if _, err := p.ResolverUnidadLote(context.Background(), "org_ajena", "unidad:prueba"); err == nil {
		t.Fatal("organizacion del llamador sustituyo configuracion")
	}
}

func TestMaterialLoteLigaCadaCambioAFuenteSinPrestamo(t *testing.T) {
	s, _, _ := solicitudLoteOrdinarioPrueba(t)
	p, err := NuevaFuenteAmbitosLotePrivada(configuracionFuentesLotePrueba())
	if err != nil {
		t.Fatal(err)
	}
	b := materialFuentesPrivadasLote(context.Background(), p, "org_prueba", s)
	var material materialFuentesLote
	if len(b) == 0 || json.Unmarshal(b, &material) != nil || material.SolicitudSHA256 != s.HuellaSolicitudSHA256 ||
		len(material.AmbitosPorCambio) != len(s.Cambios) || material.AmbitosPorCambio[0][1].Valores[0] != "unidad:prueba" {
		t.Fatal("material de fuente no ligado")
	}
	if materialFuentesPrivadasLote(context.Background(), fuenteLotePrueba{}, "org_prueba", s) != nil {
		t.Fatal("fallo de fuente invento descriptor")
	}
}

func TestFuenteLoteRechazaDuplicadoYReferenciaAjena(t *testing.T) {
	c := configuracionFuentesLotePrueba()
	c.Unidades = append(c.Unidades, c.Unidades[0])
	if _, err := NuevaFuenteAmbitosLotePrivada(c); err == nil {
		t.Fatal("unidad duplicada")
	}
	c = configuracionFuentesLotePrueba()
	c.Unidades[0].Descriptores[1].Valores[0] = "unidad:otra"
	if _, err := NuevaFuenteAmbitosLotePrivada(c); err == nil {
		t.Fatal("fuente ajena")
	}
}
