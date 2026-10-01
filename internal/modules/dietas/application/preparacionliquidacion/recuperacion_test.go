package preparacionliquidacion

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

func instantaneaRecuperacion(t *testing.T, fixture string) domain.InstantaneaLiquidacionPropuesta {
	t.Helper()
	datos, err := os.ReadFile("../../../../../cmd/vec-dietas/testdata/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	var e Entrada
	if err := json.Unmarshal(datos, &e); err != nil {
		t.Fatal(err)
	}
	p, err := Preparar(e)
	if err != nil {
		t.Fatal(err)
	}
	return p.Instantanea()
}

func TestRecuperarRoundtripYCopias(t *testing.T) {
	for _, fixture := range []string{"preparacion_liquidacion.json", "preparacion_liquidacion_gastos.json"} {
		t.Run(fixture, func(t *testing.T) {
			s := instantaneaRecuperacion(t, fixture)
			original, err := json.Marshal(s)
			if err != nil {
				t.Fatal(err)
			}
			var importada domain.InstantaneaLiquidacionPropuesta
			if err := json.Unmarshal(original, &importada); err != nil {
				t.Fatal(err)
			}
			p, err := Recuperar(importada)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(s, p.Instantanea()) {
				t.Fatal("instantánea discordante")
			}
			despues, _ := json.Marshal(importada)
			if string(original) != string(despues) {
				t.Fatal("importación mutada")
			}
			importada.Catalogo.Reglas[0].Referencia = "regla:ajena"
			importada.Catalogo.Fuentes[0] = "fuente:ajena"
			*importada.Documento.Lineas[0].IndiceTramo = 31
			importada.Lineas[0].OriginalCentimos++
			if !reflect.DeepEqual(s, p.Instantanea()) {
				t.Fatal("preparación comparte datos importados")
			}
		})
	}
}

func TestRecuperarRechazaManipulacionInclusoConHuellaRecalculada(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*domain.InstantaneaLiquidacionPropuesta)
	}{
		{"esquema", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Esquema = "otro" }},
		{"procedencia", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Procedencia = "registrada" }},
		{"liquidable", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Liquidable = true }},
		{"catalogo_ref", func(s *domain.InstantaneaLiquidacionPropuesta) { s.CatalogoRef += "x" }},
		{"catalogo_version", func(s *domain.InstantaneaLiquidacionPropuesta) { s.CatalogoVersion += "x" }},
		{"catalogo_sha", func(s *domain.InstantaneaLiquidacionPropuesta) { s.CatalogoSHA256 = strings.Repeat("0", 64) }},
		{"documento_sha", func(s *domain.InstantaneaLiquidacionPropuesta) { s.DocumentoSHA256 = strings.Repeat("0", 64) }},
		{"documento", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Documento.TotalOrientativoCentimos++ }},
		{"catalogo", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Catalogo.Reglas[0].TopeCentimos++ }},
		{"total_original", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Totales.OriginalCentimos++ }},
		{"total_reconocido", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Totales.ReconocidoPropuestoCentimos++ }},
		{"total_rechazado", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Totales.RechazadoCentimos++ }},
		{"linea_original", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Lineas[0].OriginalCentimos++ }},
		{"linea_rechazo", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Lineas[0].RechazadoCentimos++ }},
		{"linea_exceso", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Lineas[0].ReconocidoPropuestoCentimos++ }},
		{"linea_regla", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Lineas[0].ReglaRef = "regla:ajena" }},
		{"linea_motivo", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Lineas[0].MotivoCodigo = "inventado" }},
		{"indice_duplicado", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Lineas[1].Indice = s.Lineas[0].Indice }},
		{"orden_lineas", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Lineas[0], s.Lineas[1] = s.Lineas[1], s.Lineas[0] }},
		{"sin_lineas", func(s *domain.InstantaneaLiquidacionPropuesta) { s.Lineas = nil }},
		{"exceso_lineas", func(s *domain.InstantaneaLiquidacionPropuesta) {
			s.Lineas = make([]domain.LineaLiquidacionPropuesta, 103)
		}},
	}
	for _, caso := range casos {
		for _, rehash := range []bool{false, true} {
			t.Run(caso.nombre+map[bool]string{false: "", true: "_rehash"}[rehash], func(t *testing.T) {
				s := instantaneaRecuperacion(t, "preparacion_liquidacion.json")
				caso.mutar(&s)
				if rehash {
					s.SnapshotSHA256 = ""
					var err error
					s.SnapshotSHA256, err = domain.HuellaDatosLiquidacion(s)
					if err != nil {
						t.Fatal(err)
					}
				}
				if p, err := Recuperar(s); p != nil || !errors.Is(err, domain.ErrPreparacionLiquidacion) {
					t.Fatalf("manipulación aceptada: %v", err)
				}
			})
		}
	}
	for _, hash := range []string{"", strings.Repeat("0", 64)} {
		s := instantaneaRecuperacion(t, "preparacion_liquidacion.json")
		s.SnapshotSHA256 = hash
		if _, err := Recuperar(s); !errors.Is(err, domain.ErrPreparacionLiquidacion) {
			t.Fatal("huella final ausente o incorrecta aceptada")
		}
	}
}

func TestRecuperarRevalidaRutaAunqueSeRecalculenAmbasHuellas(t *testing.T) {
	s := instantaneaRecuperacion(t, "preparacion_liquidacion.json")
	s.Documento.Lineas[1].KilometrosBase = "99.0000"
	var err error
	s.DocumentoSHA256, err = domain.HuellaDatosLiquidacion(s.Documento)
	if err != nil {
		t.Fatal(err)
	}
	s.SnapshotSHA256 = ""
	s.SnapshotSHA256, err = domain.HuellaDatosLiquidacion(s)
	if err != nil {
		t.Fatal(err)
	}
	if p, err := Recuperar(s); p != nil || !errors.Is(err, domain.ErrPreparacionLiquidacion) {
		t.Fatalf("ruta incoherente aceptada con huellas concordantes: %v", err)
	}
}
