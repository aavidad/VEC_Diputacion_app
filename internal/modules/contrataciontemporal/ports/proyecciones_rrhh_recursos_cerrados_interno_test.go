package ports

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type codecsEscrituraRecursosConsultaRRHH interface {
	MarshalText() ([]byte, error)
	MarshalBinary() ([]byte, error)
	GobEncode() ([]byte, error)
	MarshalCBOR() ([]byte, error)
	MarshalYAML() (any, error)
}

type codecsLecturaRecursosConsultaRRHH interface {
	UnmarshalText([]byte) error
	UnmarshalBinary([]byte) error
	GobDecode([]byte) error
	UnmarshalCBOR([]byte) error
	UnmarshalYAML(func(any) error) error
}

func TestRecursosConsultaRRHHTienenUnicoRecursoPrivado(t *testing.T) {
	t.Parallel()

	tipo := reflect.TypeOf(RecursosConsultaRRHH{})
	if tipo.NumField() != 2 {
		t.Fatalf("campos inesperados en el envoltorio: %d", tipo.NumField())
	}
	bloqueo := tipo.Field(0)
	if !bloqueo.Anonymous || bloqueo.PkgPath == "" ||
		bloqueo.Type != reflect.TypeOf(bloqueoSerializacionConsultaRRHH{}) {
		t.Fatalf("bloqueo de serialización inesperado: %#v", bloqueo)
	}
	recurso := tipo.Field(1)
	if recurso.Name != "recurso" || recurso.PkgPath == "" ||
		recurso.Type != reflect.TypeOf(dominiovec.RecursoAutorizable{}) ||
		recurso.Tag != "" {
		t.Fatalf("recurso privado inesperado: %#v", recurso)
	}
	for _, nombre := range []string{"Recurso", "Ambitos", "Atributos", "Datos"} {
		if _, existe := tipo.MethodByName(nombre); existe {
			t.Fatalf("se expuso el getter %s por valor", nombre)
		}
		if _, existe := reflect.PointerTo(tipo).MethodByName(nombre); existe {
			t.Fatalf("se expuso el getter %s por puntero", nombre)
		}
	}
}

func TestRecursosConsultaRRHHValidanSoloInvariantesComunes(t *testing.T) {
	t.Parallel()

	valido := recursosConsultaRRHHInternosPrueba()
	if err := valido.validarEstructura(); err != nil {
		t.Fatalf("recurso común válido rechazado: %v", err)
	}

	for nombre, valor := range map[string]RecursosConsultaRRHH{
		"cero": {},
		"modulo_ajeno": {
			recurso: dominiovec.RecursoAutorizable{
				Referencia: "recurso:rrhh:cerrado:001",
				ModuloID:   "modulo_ajeno",
				Tipo:       "consulta_rrhh",
			},
		},
		"recurso_invalido": {
			recurso: dominiovec.RecursoAutorizable{
				Referencia: "*",
				ModuloID:   ModuloContratacion,
				Tipo:       "consulta_rrhh",
			},
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			if err := valor.validarEstructura(); !errors.Is(
				err,
				ErrCapacidadConsultaRRHHInvalida,
			) {
				t.Fatalf("recurso común inválido aceptado: %v", err)
			}
		})
	}
}

func TestRecursosConsultaRRHHBloqueanSerializacionYRegistro(t *testing.T) {
	t.Parallel()

	recursos := recursosConsultaRRHHInternosPrueba()
	for nombre, valor := range map[string]any{
		"valor":   recursos,
		"puntero": &recursos,
	} {
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			comprobarBloqueoEscrituraRecursosConsultaRRHH(t, valor)
			comprobarRedaccionRecursosConsultaRRHH(t, valor)
		})
	}

	comprobarBloqueoLecturaRecursosConsultaRRHH(t, &recursos)
}

func recursosConsultaRRHHInternosPrueba() RecursosConsultaRRHH {
	return RecursosConsultaRRHH{
		recurso: dominiovec.RecursoAutorizable{
			Referencia: "recurso:rrhh:cerrado:001",
			ModuloID:   ModuloContratacion,
			Tipo:       "consulta_rrhh",
			Ambitos: map[string]string{
				"ambito_prueba": "organizacion:rrhh:privada",
			},
			Atributos: map[string]string{
				"atributo_prueba": "valor_interno_no_exportable",
			},
		},
	}
}

func comprobarBloqueoEscrituraRecursosConsultaRRHH(t *testing.T, valor any) {
	t.Helper()

	codecs, ok := valor.(codecsEscrituraRecursosConsultaRRHH)
	if !ok {
		t.Fatalf("%T no conserva todos los bloqueos de escritura", valor)
	}
	comprobarErrorSensibleRecursosConsultaRRHH(
		t, "json", func() error { _, err := json.Marshal(valor); return err }(),
	)
	comprobarErrorSensibleRecursosConsultaRRHH(
		t, "xml", func() error { _, err := xml.Marshal(valor); return err }(),
	)
	comprobarErrorSensibleRecursosConsultaRRHH(
		t, "texto", func() error { _, err := codecs.MarshalText(); return err }(),
	)
	comprobarErrorSensibleRecursosConsultaRRHH(
		t, "binario", func() error { _, err := codecs.MarshalBinary(); return err }(),
	)
	comprobarErrorSensibleRecursosConsultaRRHH(
		t, "gob_directo", func() error { _, err := codecs.GobEncode(); return err }(),
	)
	comprobarErrorSensibleRecursosConsultaRRHH(
		t, "cbor", func() error { _, err := codecs.MarshalCBOR(); return err }(),
	)
	comprobarErrorSensibleRecursosConsultaRRHH(
		t, "yaml", func() error { _, err := codecs.MarshalYAML(); return err }(),
	)

	var destino bytes.Buffer
	comprobarErrorSensibleRecursosConsultaRRHH(
		t,
		"gob",
		gob.NewEncoder(&destino).Encode(valor),
	)
}

func comprobarBloqueoLecturaRecursosConsultaRRHH(
	t *testing.T,
	recursos *RecursosConsultaRRHH,
) {
	t.Helper()

	codecs, ok := any(recursos).(codecsLecturaRecursosConsultaRRHH)
	if !ok {
		t.Fatalf("%T no conserva todos los bloqueos de lectura", recursos)
	}
	comprobarErrorSensibleRecursosConsultaRRHH(
		t, "json_decode", json.Unmarshal([]byte(`{}`), recursos),
	)
	comprobarErrorSensibleRecursosConsultaRRHH(
		t, "xml_decode", xml.Unmarshal([]byte(`<recursos/>`), recursos),
	)
	comprobarErrorSensibleRecursosConsultaRRHH(
		t, "texto_decode", codecs.UnmarshalText([]byte("adulterado")),
	)
	comprobarErrorSensibleRecursosConsultaRRHH(
		t, "binario_decode", codecs.UnmarshalBinary([]byte("adulterado")),
	)
	comprobarErrorSensibleRecursosConsultaRRHH(
		t, "gob_decode", codecs.GobDecode([]byte("adulterado")),
	)
	comprobarErrorSensibleRecursosConsultaRRHH(
		t, "cbor_decode", codecs.UnmarshalCBOR([]byte("adulterado")),
	)
	comprobarErrorSensibleRecursosConsultaRRHH(
		t,
		"yaml_decode",
		codecs.UnmarshalYAML(func(any) error { return nil }),
	)
}

func comprobarRedaccionRecursosConsultaRRHH(t *testing.T, valor any) {
	t.Helper()

	formato := fmt.Sprintf("%v %+v %#v", valor, valor, valor)
	var registro bytes.Buffer
	slog.New(slog.NewJSONHandler(&registro, nil)).Info(
		"recursos_consulta_rrhh",
		"valor",
		valor,
	)
	for _, salida := range []string{formato, registro.String()} {
		if !strings.Contains(salida, "MATERIAL-CONSULTA-RRHH-OPACO") {
			t.Fatalf("representación no redactada: %q", salida)
		}
		for _, sensible := range []string{
			"recurso:rrhh:cerrado:001",
			"organizacion:rrhh:privada",
			"valor_interno_no_exportable",
		} {
			if strings.Contains(salida, sensible) {
				t.Fatalf("representación filtra %q: %q", sensible, salida)
			}
		}
	}
}

func comprobarErrorSensibleRecursosConsultaRRHH(
	t *testing.T,
	nombre string,
	err error,
) {
	t.Helper()
	if !errors.Is(err, ErrMaterialConsultaRRHHSensible) {
		t.Fatalf("%s no quedó bloqueado: %v", nombre, err)
	}
}
