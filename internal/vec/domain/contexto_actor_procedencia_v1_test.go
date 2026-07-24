package domain

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

func TestManifiestoProcedenciaContextoActorV1ConservaMaximoUint64(t *testing.T) {
	manifiesto := manifiestoProcedenciaContextoActorPrueba(math.MaxUint64)
	contenido, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatalf("representar uint64 maximo: %v", err)
	}
	rehidratado, err := RehidratarManifiestoProcedenciaContextoActorV1(contenido)
	if err != nil || rehidratado.Cuenta.Version != math.MaxUint64 ||
		rehidratado.Persona.Version != math.MaxUint64 ||
		rehidratado.Perfil.Version != math.MaxUint64 ||
		rehidratado.Contexto.Version != math.MaxUint64 ||
		rehidratado.Vinculos[0].Version != math.MaxUint64 ||
		rehidratado.Cuenta.ProcedenciaVersion != math.MaxUint64 {
		t.Fatalf("uint64 maximo no preservado: %#v err=%v", rehidratado, err)
	}
}

func TestManifiestoProcedenciaContextoActorV1RechazaBordesYAdulteraciones(t *testing.T) {
	base := manifiestoProcedenciaContextoActorPrueba(1)
	contenido, err := base.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	casos := [][]byte{
		append(append([]byte(nil), contenido...), ' '),
		[]byte(strings.Replace(string(contenido), `"esquema":`, `"extra":1,"esquema":`, 1)),
		[]byte(strings.Replace(string(contenido), `"version":1`, `"version":0`, 1)),
		[]byte(strings.Replace(string(contenido), `"version":1`, `"version":18446744073709551616`, 1)),
		[]byte(strings.Replace(string(contenido), `"autoridad_efectiva":"autoridad_maestra_acreditada"`, `"autoridad_efectiva":"no_autoritativa"`, 1)),
	}
	for i, caso := range casos {
		if _, err := RehidratarManifiestoProcedenciaContextoActorV1(caso); err == nil {
			t.Fatalf("adulteracion %d aceptada: %s", i, caso)
		}
	}

	duplicado := bytes.Replace(contenido, []byte(`"esquema":`),
		[]byte(`"esquema":"vec.contexto-actor.procedencia-manifiesto.v1","esquema":`), 1)
	if _, err := RehidratarManifiestoProcedenciaContextoActorV1(duplicado); err == nil {
		t.Fatal("clave repetida aceptada")
	}
}

func manifiestoProcedenciaContextoActorPrueba(version uint64) ManifiestoProcedenciaContextoActorV1 {
	ref := func(prefijo, caracter string) string { return prefijo + strings.Repeat(caracter, 24) }
	acreditacion := AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef: ref("prc_", "p"), ProcedenciaVersion: version,
		ProcedenciaHuellaSHA256: strings.Repeat("a", 64),
		ProcedenciaAutoridad:    AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	return ManifiestoProcedenciaContextoActorV1{
		Esquema:           EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: ProcedenciaCuentaContextoActorV1{
			CuentaRef: ref("cta_", "a"), Version: version,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: ProcedenciaPersonaContextoActorV1{
			PersonaRef: ref("per_", "b"), Version: version,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: ProcedenciaPerfilContextoActorV1{
			PerfilRef: ref("prf_", "c"), Version: version,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: ProcedenciaVinculoContextoActorV1{
			VinculoRef: ref("vca_", "d"), Version: version,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []ProcedenciaVinculoReferenciaContextoActorV1{{
			VinculoRef: ref("vin_", "e"), Version: version,
			Tipo: TipoReferenciaContextoActorCandidato, Referencia: ref("can_", "f"),
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		}},
	}
}
