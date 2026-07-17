package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const contextoActorCanonicoGoldenV1 = `{"esquema":"vec.contexto-actor.vinculado.v1","principal_ref":"per_rrrrrrrrrrrrrrrrrrrrrr","metodo":"certificado","garantia":"alto","perfil_activo_ref":"prf_pppppppppppppppppppppp","persona_ref":"per_rrrrrrrrrrrrrrrrrrrrrr","contexto_actor_ref":"vca_vvvvvvvvvvvvvvvvvvvvvv","contexto_version":3,"cuenta_ref":"cta_aaaaaaaaaaaaaaaaaaaaaa","persona_version":4,"perfil_version":5,"estado":"activo","vigente_desde":"2026-07-15T09:30:00.123000Z","vigente_hasta":"2026-07-15T11:30:00.123000Z","resuelto_en":"2026-07-15T10:30:00.123000Z","vinculos":[{"vinculo_ref":"vin_cccccccccccccccccccccc","version":7,"tipo":"candidato","referencia":"can_cccccccccccccccccccccc","estado":"activo","vigente_desde":"2026-07-15T09:30:00.123000Z","vigente_hasta":"2026-07-15T11:30:00.123000Z"},{"vinculo_ref":"vin_eeeeeeeeeeeeeeeeeeeeee","version":9,"tipo":"empleado","referencia":"emp_eeeeeeeeeeeeeeeeeeeeee","estado":"activo","vigente_desde":"2026-07-15T09:30:00.123000Z","vigente_hasta":"2026-07-15T11:30:00.123000Z"}]}`

const huellaContextoActorCanonicoGoldenV1 = "c57ac236b6ee6b8beedf77a720ac6c878500da7ba5fe5b8c7fb31fa9655c3023"

func TestRepresentacionContextoActorV1ConservaVectorGoldenYHuellaHistorica(t *testing.T) {
	actor := contextoActorCanonicoV1Prueba(t)
	representacion, err := actor.RepresentacionCanonicaVinculadaV1()
	if err != nil {
		t.Fatalf("representacion canonica: %v", err)
	}
	if string(representacion) != contextoActorCanonicoGoldenV1 {
		t.Fatalf("vector V1 alterado\nobtenido: %s\nesperado: %s", representacion, contextoActorCanonicoGoldenV1)
	}
	huella, err := actor.HuellaSHA256VinculadaV1()
	if err != nil || huella != huellaContextoActorCanonicoGoldenV1 {
		t.Fatalf("huella V1 alterada: obtenida=%q esperada=%q error=%v", huella, huellaContextoActorCanonicoGoldenV1, err)
	}

	representacion[0] = '['
	segunda, err := actor.RepresentacionCanonicaVinculadaV1()
	if err != nil || string(segunda) != contextoActorCanonicoGoldenV1 {
		t.Fatal("los bytes canónicos compartieron memoria con el llamador")
	}
}

func TestRehidratarContextoActorV1RecuperaSoloElDocumentoCanonico(t *testing.T) {
	actor, err := RehidratarContextoActorVinculadoV1([]byte(contextoActorCanonicoGoldenV1))
	if err != nil {
		t.Fatalf("rehidratar golden: %v", err)
	}
	representacion, err := actor.RepresentacionCanonicaVinculadaV1()
	if err != nil || string(representacion) != contextoActorCanonicoGoldenV1 {
		t.Fatalf("rehidratacion no reversible: %s, %v", representacion, err)
	}
	huella, err := actor.HuellaSHA256VinculadaV1()
	if err != nil || huella != huellaContextoActorCanonicoGoldenV1 {
		t.Fatalf("huella rehidratada distinta: %q, %v", huella, err)
	}
}

func TestRehidratarContextoActorV1RechazaVariantesNoCanonicas(t *testing.T) {
	base := contextoActorCanonicoGoldenV1
	var documento contextoActorCanonicoV1
	if err := json.Unmarshal([]byte(base), &documento); err != nil {
		t.Fatalf("preparar documento: %v", err)
	}
	documento.Vinculos[0], documento.Vinculos[1] = documento.Vinculos[1], documento.Vinculos[0]
	desordenada, err := json.Marshal(documento)
	if err != nil {
		t.Fatalf("preparar orden alternativo: %v", err)
	}

	cabeceraDuplicada := strings.Replace(base, `{"esquema":`, `{"esquema":"vec.contexto-actor.vinculado.v1","esquema":`, 1)
	vinculoDuplicado := strings.Replace(base,
		`"vinculo_ref":"vin_cccccccccccccccccccccc"`,
		`"vinculo_ref":"vin_cccccccccccccccccccccc","vinculo_ref":"vin_cccccccccccccccccccccc"`, 1)
	cabeceraDesconocida := strings.Replace(base, `{"esquema":`, `{"extension":true,"esquema":`, 1)
	vinculoDesconocido := strings.Replace(base,
		`"vinculo_ref":"vin_cccccccccccccccccccccc"`,
		`"extension":true,"vinculo_ref":"vin_cccccccccccccccccccccc"`, 1)

	casos := []struct {
		nombre string
		valor  []byte
	}{
		{"vacio", nil},
		{"espacio inicial", []byte(" " + base)},
		{"clave de cabecera duplicada", []byte(cabeceraDuplicada)},
		{"clave de vinculo duplicada", []byte(vinculoDuplicado)},
		{"clave de cabecera desconocida", []byte(cabeceraDesconocida)},
		{"clave de vinculo desconocida", []byte(vinculoDesconocido)},
		{"clave escapada", []byte(strings.Replace(base, `"esquema"`, `"esqu\u0065ma"`, 1))},
		{"esquema ajeno", []byte(strings.Replace(base, esquemaHuellaContextoActorV1, "vec.contexto-actor.vinculado.v2", 1))},
		{"milisegundos", []byte(strings.Replace(base, ".123000Z", ".123Z", 1))},
		{"submicrosegundo", []byte(strings.Replace(base, ".123000Z", ".1230001Z", 1))},
		{"zona alternativa", []byte(strings.Replace(base, ".123000Z", ".123000+00:00", 1))},
		{"numero no canonico", []byte(strings.Replace(base, `"contexto_version":3`, `"contexto_version":3.0`, 1))},
		{"vinculos nulos", []byte(reemplazarVinculosContextoActorV1(t, base, "null"))},
		{"vinculos desordenados", desordenada},
		{"json concatenado", []byte(base + `{}`)},
		{"demasiado grande", bytes.Repeat([]byte{'x'}, TamanoMaximoRepresentacionContextoActorV1+1)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			actor, err := RehidratarContextoActorVinculadoV1(caso.valor)
			if !errors.Is(err, ErrRepresentacionContextoActorV1Invalida) || actor.Validar() == nil {
				t.Fatalf("variante aceptada: actor=%#v error=%v", actor, err)
			}
		})
	}
}

func TestRehidratarContextoActorV1RechazaProfundidadAjenaAntesDeInterpretar(t *testing.T) {
	valor := []byte(`{"a":[[[[["b"]]]]]}`)
	if _, err := RehidratarContextoActorVinculadoV1(valor); !errors.Is(err, ErrRepresentacionContextoActorV1Invalida) {
		t.Fatalf("profundidad no acotada: %v", err)
	}
}

func TestRehidratarContextoActorV1CortaVinculosExcesivosAntesDeAsignar(t *testing.T) {
	arrayExcesivo := "[" + strings.Repeat("{},", maximoVinculosContextoActor) + "{}]"
	alterado := []byte(reemplazarVinculosContextoActorV1(
		t, contextoActorCanonicoGoldenV1, arrayExcesivo,
	))
	if validarJSONContextoActorV1SinDuplicados(alterado) == nil {
		t.Fatal("la prelectura acepto mas vinculos que el dominio")
	}
	if _, err := RehidratarContextoActorVinculadoV1(alterado); !errors.Is(err, ErrRepresentacionContextoActorV1Invalida) {
		t.Fatalf("array excesivo aceptado: %v", err)
	}
}

func FuzzRehidratarContextoActorVinculadoV1(f *testing.F) {
	f.Add([]byte(contextoActorCanonicoGoldenV1))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"vinculos":null}`))
	f.Fuzz(func(t *testing.T, contenido []byte) {
		actor, err := RehidratarContextoActorVinculadoV1(contenido)
		if err != nil {
			return
		}
		canonicos, err := actor.RepresentacionCanonicaVinculadaV1()
		if err != nil || !bytes.Equal(canonicos, contenido) || actor.Validar() != nil {
			t.Fatalf("entrada aceptada sin reversibilidad exacta: canonicos=%q error=%v", canonicos, err)
		}
	})
}

func contextoActorCanonicoV1Prueba(t *testing.T) ContextoActor {
	t.Helper()
	instante := instanteContextoActorPrueba()
	actor, err := NuevoContextoActor(
		solicitudContextoActorPrueba().Cuenta,
		instantaneaContextoActorPrueba(instante),
		instante,
	)
	if err != nil {
		t.Fatalf("crear contexto actor: %v", err)
	}
	return actor
}

func reemplazarVinculosContextoActorV1(t *testing.T, base, sustituto string) string {
	t.Helper()
	inicio := strings.Index(base, `"vinculos":[`)
	if inicio < 0 {
		t.Fatal("vector sin vinculos")
	}
	inicioValor := inicio + len(`"vinculos":`)
	profundidad := 0
	for indice := inicioValor; indice < len(base); indice++ {
		switch base[indice] {
		case '[':
			profundidad++
		case ']':
			profundidad--
			if profundidad == 0 {
				return base[:inicioValor] + sustituto + base[indice+1:]
			}
		}
	}
	t.Fatal("vector con array incompleto")
	return ""
}
