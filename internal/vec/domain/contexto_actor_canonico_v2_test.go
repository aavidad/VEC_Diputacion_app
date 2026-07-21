package domain

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const contextoActorCanonicoGoldenV2 = `{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":"per_rrrrrrrrrrrrrrrrrrrrrr","metodo":"certificado","garantia":"alto","perfil_activo_ref":"prf_pppppppppppppppppppppp","persona_ref":"per_rrrrrrrrrrrrrrrrrrrrrr","contexto_actor_ref":"vca_vvvvvvvvvvvvvvvvvvvvvv","contexto_version":3,"cuenta_ref":"cta_aaaaaaaaaaaaaaaaaaaaaa","cuenta_version":6,"persona_version":4,"perfil_version":5,"estado":"activo","vigente_desde":"2026-07-15T09:30:00.123000Z","vigente_hasta":"2026-07-15T11:30:00.123000Z","resuelto_en":"2026-07-15T10:30:00.123000Z","vinculos":[{"vinculo_ref":"vin_cccccccccccccccccccccc","version":7,"tipo":"candidato","referencia":"can_cccccccccccccccccccccc","estado":"activo","vigente_desde":"2026-07-15T09:30:00.123000Z","vigente_hasta":"2026-07-15T11:30:00.123000Z"},{"vinculo_ref":"vin_eeeeeeeeeeeeeeeeeeeeee","version":9,"tipo":"empleado","referencia":"emp_eeeeeeeeeeeeeeeeeeeeee","estado":"activo","vigente_desde":"2026-07-15T09:30:00.123000Z","vigente_hasta":"2026-07-15T11:30:00.123000Z"}]}`

const huellaContextoActorCanonicoGoldenV2 = "18e12e87244ad1d33bbd2ab1d6344bae8a7c6819723d51a3da501d7560cf4798"

func TestRepresentacionContextoActorV2ComprometeVersionCuenta(t *testing.T) {
	actor := contextoActorCanonicoV2Prueba(t, 6)
	representacion, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil || string(representacion) != contextoActorCanonicoGoldenV2 {
		t.Fatalf("vector V2 inesperado: %s, %v", representacion, err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil || huella != huellaContextoActorCanonicoGoldenV2 {
		t.Fatalf("huella V2 inesperada: %q, %v", huella, err)
	}
	if _, err := actor.RepresentacionCanonicaVinculadaV1(); !errors.Is(err, ErrContextoActorInvalido) {
		t.Fatalf("V1 omitio silenciosamente cuenta_version: %v", err)
	}

	rehidratado, err := RehidratarContextoActorVinculadoV2(representacion)
	if err != nil || rehidratado.Instantanea.CuentaVersion != 6 {
		t.Fatalf("rehidratacion V2 perdio cuenta_version: %#v, %v", rehidratado, err)
	}
	segunda, err := rehidratado.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(segunda, representacion) {
		t.Fatalf("V2 no fue reversible: %s, %v", segunda, err)
	}
}

func TestRehidratarContextoActorV2RechazaCuentaVersionAusenteInvalidaONoCanonica(t *testing.T) {
	base := contextoActorCanonicoGoldenV2
	cases := []struct {
		nombre string
		valor  string
	}{
		{"version ausente", strings.Replace(base, `,"cuenta_version":6`, "", 1)},
		{"version cero", strings.Replace(base, `"cuenta_version":6`, `"cuenta_version":0`, 1)},
		{"version excede uint64", strings.Replace(base, `"cuenta_version":6`, `"cuenta_version":18446744073709551616`, 1)},
		{"esquema V1", strings.Replace(base, EsquemaRepresentacionContextoActorV2, esquemaHuellaContextoActorV1, 1)},
		{"campo desconocido", strings.Replace(base, `"cuenta_version":6`, `"cuenta_version":6,"extra":true`, 1)},
		{"espacio final", base + " "},
	}
	for _, caso := range cases {
		t.Run(caso.nombre, func(t *testing.T) {
			actor, err := RehidratarContextoActorVinculadoV2([]byte(caso.valor))
			if !errors.Is(err, ErrRepresentacionContextoActorV2Invalida) || actor.Validar() == nil {
				t.Fatalf("V2 inseguro aceptado: actor=%#v err=%v", actor, err)
			}
		})
	}
}

func TestContextoActorV2ConservaCuentaVersionMaxima(t *testing.T) {
	const maxima = ^uint64(0)
	actor := contextoActorCanonicoV2Prueba(t, maxima)
	contenido, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	rehidratado, err := RehidratarContextoActorVinculadoV2(contenido)
	if err != nil || rehidratado.Instantanea.CuentaVersion != maxima {
		t.Fatalf("uint64 maximo perdido: %#v, %v", rehidratado.Instantanea, err)
	}
}

func contextoActorCanonicoV2Prueba(t *testing.T, cuentaVersion uint64) ContextoActor {
	t.Helper()
	instantanea := instantaneaContextoActorPrueba(instanteContextoActorPrueba())
	instantanea.CuentaVersion = cuentaVersion
	actor, err := NuevoContextoActor(
		solicitudContextoActorPrueba().Cuenta,
		instantanea,
		instanteContextoActorPrueba(),
	)
	if err != nil {
		t.Fatalf("crear contexto actor V2: %v", err)
	}
	return actor
}
