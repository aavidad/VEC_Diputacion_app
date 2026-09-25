package importacionconvoca

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"
)

var esquemasConvoca = []EsquemaExportacion{EsquemaResumenPersona, EsquemaDetalleMerito}

func TestFormatosConvocaSonCerradosDisjuntosYEnNFC(t *testing.T) {
	vistas := map[string]string{}
	for _, formato := range formatosCabeceras {
		for _, esquema := range esquemasConvoca {
			cabeceras := esquema.CabecerasFormato(formato)
			if len(cabeceras) != esquema.NumeroColumnas() {
				t.Fatalf("%s/%s: %d columnas", formato, esquema, len(cabeceras))
			}
			for _, c := range append(cabeceras, esquema.NombreHoja(formato)) {
				if !norm.NFC.IsNormalString(c) {
					t.Fatalf("%s/%s: literal no NFC %q", formato, esquema, c)
				}
			}
			clave := strings.Join(cabeceras, "\x1f")
			if previo, ok := vistas[clave]; ok {
				t.Fatalf("cabeceras ambiguas entre %s y %s/%s", previo, formato, esquema)
			}
			vistas[clave] = string(formato) + "/" + string(esquema)
			detectado, detectadoFormato, err := DetectarFormato(cabeceras)
			if err != nil || detectado != esquema || detectadoFormato != formato {
				t.Fatalf("detectar %s/%s: %s %s %v", formato, esquema, detectado, detectadoFormato, err)
			}
		}
	}
	if EsquemaResumenPersona.CabecerasFormato("convoca:v3") != nil {
		t.Fatal("formato inexistente con cabeceras")
	}
}

func TestDetectarFormatoRealMapeaDocumentoEnmascaradoYAceptaSoloNFC(t *testing.T) {
	real := EsquemaResumenPersona.CabecerasFormato(FormatoConvocaV2)
	if real[0] != "DNI/NIE enmascarado" || real[6] != "Formación" {
		t.Fatalf("literales reales inesperados: %#v", real)
	}
	descompuestas := make([]string, len(real))
	for i, c := range real {
		descompuestas[i] = norm.NFD.String(c)
	}
	if descompuestas[6] == real[6] {
		t.Fatal("la prueba NFD no descompone")
	}
	if esquema, formato, err := DetectarFormato(descompuestas); err != nil ||
		esquema != EsquemaResumenPersona || formato != FormatoConvocaV2 {
		t.Fatalf("NFD equivalente rechazada: %s %s %v", esquema, formato, err)
	}
	for nombre, alterar := range map[string]func([]string){
		"sin tilde":          func(c []string) { c[6] = "Formacion" },
		"mayusculas":         func(c []string) { c[0] = "DNI/NIE ENMASCARADO" },
		"espacio final":      func(c []string) { c[0] = "DNI/NIE enmascarado " },
		"espacio duro":       func(c []string) { c[0] = "DNI/NIE enmascarado" },
		"compatibilidad":     func(c []string) { c[7] = "Ｔotal" },
		"mezcla v1 y v2":     func(c []string) { c[0] = "DNI/NIE" },
		"utf8 invalido":      func(c []string) { c[1] = "Primer Apellido\xff" },
		"cabecera ajena":     func(c []string) { c[5] = "Experiencia laboral" },
		"documento sin masc": func(c []string) { c[0] = "DNI/NIE sin enmascarar" },
	} {
		alteradas := append([]string(nil), real...)
		alterar(alteradas)
		if _, _, err := DetectarFormato(alteradas); !errors.Is(err, ErrEsquemaExportacionDesconocido) {
			t.Fatalf("%s aceptada: %v", nombre, err)
		}
	}
	detalle := EsquemaDetalleMerito.CabecerasFormato(FormatoConvocaV2)
	detalle[8] = "Descripcion del merito"
	if _, _, err := DetectarFormato(detalle); !errors.Is(err, ErrEsquemaExportacionDesconocido) {
		t.Fatalf("detalle mezclado aceptado: %v", err)
	}
}

func TestValidarNombreHojaConvocaV2EsLiteralYFailClosed(t *testing.T) {
	aceptados := []struct {
		esquema EsquemaExportacion
		formato FormatoCabeceras
		nombre  string
	}{
		{EsquemaResumenPersona, FormatoConvocaV2, "grupo de méritos (Tribunal) (1)"},
		{EsquemaResumenPersona, FormatoConvocaV2, norm.NFD.String("grupo de méritos (Tribunal) (1)")},
		{EsquemaDetalleMerito, FormatoConvocaV2, "méritos (1)"},
		{EsquemaResumenPersona, FormatoConvocaV1, ""},
		{EsquemaResumenPersona, FormatoConvocaV1, "Hoja1"},
		{EsquemaResumenPersona, FormatoConvocaV1, "grupo de méritos (Tribunal) (1)"},
		{EsquemaDetalleMerito, FormatoConvocaV1, "méritos (1)"},
	}
	for _, caso := range aceptados {
		if err := ValidarNombreHoja(caso.esquema, caso.formato, caso.nombre); err != nil {
			t.Fatalf("%s/%s %q rechazada: %v", caso.formato, caso.esquema, caso.nombre, err)
		}
	}
	rechazados := []struct {
		esquema EsquemaExportacion
		formato FormatoCabeceras
		nombre  string
	}{
		{EsquemaResumenPersona, FormatoConvocaV2, "grupo de méritos (Tribunal) (2)"},
		{EsquemaResumenPersona, FormatoConvocaV2, "grupo de méritos (Tribunal)"},
		{EsquemaResumenPersona, FormatoConvocaV2, "Grupo de méritos (Tribunal) (1)"},
		{EsquemaResumenPersona, FormatoConvocaV2, "grupo de meritos (Tribunal) (1)"},
		{EsquemaResumenPersona, FormatoConvocaV2, "méritos (1)"},
		{EsquemaResumenPersona, FormatoConvocaV2, ""},
		{EsquemaDetalleMerito, FormatoConvocaV2, "méritos (2)"},
		{EsquemaDetalleMerito, FormatoConvocaV2, "méritos (1) "},
		{EsquemaDetalleMerito, FormatoConvocaV2, "grupo de méritos (Tribunal) (1)"},
		{EsquemaResumenPersona, FormatoConvocaV1, "méritos (1)"},
		{EsquemaDetalleMerito, FormatoConvocaV1, "grupo de méritos (Tribunal) (1)"},
		{EsquemaResumenPersona, "convoca:v3", "grupo de méritos (Tribunal) (1)"},
		{"otro", FormatoConvocaV2, "méritos (1)"},
	}
	for _, caso := range rechazados {
		if err := ValidarNombreHoja(caso.esquema, caso.formato, caso.nombre); !errors.Is(err, ErrEsquemaExportacionDesconocido) {
			t.Fatalf("%s/%s %q aceptada: %v", caso.formato, caso.esquema, caso.nombre, err)
		}
	}
}

func TestValidarHojaConvocaV2ExigeHojaYValidaIgualQueV1(t *testing.T) {
	filas := []FilaStaging{filaResumen(2, "***0001**", "Sintética", "Uno", "Ana", "Libre", "1", "2", "3")}
	real := HojaStaging{
		Esquema:    EsquemaResumenPersona,
		Cabeceras:  EsquemaResumenPersona.CabecerasFormato(FormatoConvocaV2),
		NombreHoja: "grupo de méritos (Tribunal) (1)",
		Filas:      filas,
	}
	resultado, err := ValidarHoja(real)
	if err != nil || len(resultado.Aceptadas) != 1 || resultado.Rechazadas != 0 ||
		resultado.Aceptadas[0].Identidad.Documento != "***0001**" ||
		resultado.Aceptadas[0].Esquema != EsquemaResumenPersona {
		t.Fatalf("hoja real no aceptada: %#v %v", resultado, err)
	}
	for nombre, hoja := range map[string]HojaStaging{
		"sin nombre de hoja": {Esquema: real.Esquema, Cabeceras: real.Cabeceras, Filas: filas},
		"hoja (2)":           {Esquema: real.Esquema, Cabeceras: real.Cabeceras, NombreHoja: "grupo de méritos (Tribunal) (2)", Filas: filas},
		"esquema cruzado":    {Esquema: EsquemaDetalleMerito, Cabeceras: real.Cabeceras, NombreHoja: "méritos (1)", Filas: filas},
	} {
		if _, err := ValidarHoja(hoja); !errors.Is(err, ErrHojaStagingInvalida) {
			t.Fatalf("%s aceptada: %v", nombre, err)
		}
	}
}
