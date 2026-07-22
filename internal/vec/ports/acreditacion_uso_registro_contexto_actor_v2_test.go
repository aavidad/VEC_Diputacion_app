package ports

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestOrdenAcreditacionUsoRegistroContextoActorV2EsCompletaDefensivaYOpaca(t *testing.T) {
	t.Parallel()
	resultado, emitida, validaHasta := escenarioOrdenAcreditacionContextoActorV2Prueba(t)
	orden, err := NuevaOrdenAcreditacionUsoRegistroContextoActorV2(resultado, emitida, validaHasta)
	if err != nil {
		t.Fatalf("crear orden: %v", err)
	}
	datos, err := orden.Datos()
	if err != nil || datos.Resultado.Contexto.Instantanea.PersonaVersion != 3 ||
		datos.Resultado.Contexto.Instantanea.PerfilVersion != 4 ||
		!datos.EmitidaEn.Equal(emitida) || !datos.ValidaHasta.Equal(validaHasta) {
		t.Fatalf("datos incompletos: %#v, %v", datos, err)
	}

	resultado.RepresentacionCanonica[0] ^= 0xff
	resultado.ManifiestoProcedenciaCanonico[0] ^= 0xff
	datos.Resultado.RepresentacionCanonica[0] ^= 0xff
	datos.Resultado.ManifiestoProcedenciaCanonico[0] ^= 0xff
	otra, err := orden.Datos()
	if err != nil || otra.Resultado.Validar() != nil {
		t.Fatalf("la orden no conservo copia defensiva: %v", err)
	}

	for nombre, valor := range map[string]any{"orden": orden, "datos": otra} {
		if _, err = json.Marshal(valor); !errors.Is(err, ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida) {
			t.Fatalf("%s JSON: %v", nombre, err)
		}
		if _, err = xml.Marshal(valor); !errors.Is(err, ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida) {
			t.Fatalf("%s XML: %v", nombre, err)
		}
	}
	if _, err = orden.MarshalText(); !errors.Is(err, ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida) {
		t.Fatalf("texto: %v", err)
	}
	if _, err = orden.MarshalBinary(); !errors.Is(err, ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida) {
		t.Fatalf("binario: %v", err)
	}
	if _, err = orden.GobEncode(); !errors.Is(err, ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida) {
		t.Fatalf("gob: %v", err)
	}
	if _, err = orden.MarshalCBOR(); !errors.Is(err, ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida) {
		t.Fatalf("CBOR: %v", err)
	}
	if _, err = orden.MarshalYAML(); !errors.Is(err, ErrSerializacionAcreditacionUsoRegistroContextoActorV2Prohibida) {
		t.Fatalf("YAML: %v", err)
	}
	if texto := orden.String(); strings.Contains(texto, resultado.RegistroContextoRef) ||
		strings.Contains(texto, resultado.Contexto.PersonaRef) {
		t.Fatalf("formato filtro referencias: %q", texto)
	}
}

func TestOrdenAcreditacionUsoRegistroContextoActorV2RechazaVentanasInvalidas(t *testing.T) {
	t.Parallel()
	resultado, emitida, validaHasta := escenarioOrdenAcreditacionContextoActorV2Prueba(t)
	casos := []struct {
		nombre string
		desde  time.Time
		hasta  time.Time
	}{
		{"antes_del_recibo", resultado.ResueltoEnAutoritativo.Add(-time.Microsecond), validaHasta},
		{"vacia", emitida, emitida},
		{"fuera_del_contexto", emitida, resultado.Contexto.Instantanea.VigenteHasta.Add(time.Microsecond)},
		{"no_utc", emitida.In(time.FixedZone("otro", 3600)), validaHasta},
		{"submicrosegundo", emitida.Add(time.Nanosecond), validaHasta},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			if _, err := NuevaOrdenAcreditacionUsoRegistroContextoActorV2(
				resultado, caso.desde, caso.hasta,
			); !errors.Is(err, ErrOrdenAcreditacionUsoRegistroContextoActorV2Invalida) {
				t.Fatalf("ventana aceptada: %v", err)
			}
		})
	}
	var cero OrdenAcreditacionUsoRegistroContextoActorV2
	if _, err := cero.Datos(); !errors.Is(err, ErrOrdenAcreditacionUsoRegistroContextoActorV2Invalida) {
		t.Fatalf("valor cero aceptado: %v", err)
	}
}

func escenarioOrdenAcreditacionContextoActorV2Prueba(
	t *testing.T,
) (domain.ResultadoContextoActorRegistradoV2, time.Time, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 7, 22, 10, 0, 0, 123_456_000, time.UTC)
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: domain.AuthMethodCertificate,
		Garantia: domain.AuthAssuranceHigh,
	}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 5,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: 7,
		PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 3,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 4,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(30 * time.Minute),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatalf("crear actor: %v", err)
	}
	representacion, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := domain.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef: "prc_0123456789abcdefghijkl", ProcedenciaVersion: 1,
		ProcedenciaHuellaSHA256: strings.Repeat("a", 64),
		ProcedenciaAutoridad:    domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := domain.ManifiestoProcedenciaContextoActorV1{
		Esquema:           domain.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: domain.ProcedenciaCuentaContextoActorV1{
			CuentaRef: instantanea.CuentaRef, Version: instantanea.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: domain.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantanea.PersonaRef, Version: instantanea.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: domain.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantanea.PerfilActivoRef, Version: instantanea.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: domain.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantanea.VinculoRef, Version: instantanea.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []domain.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	manifiestoCanonico, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(manifiestoCanonico)
	if err != nil {
		t.Fatal(err)
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn", Contexto: actor,
		RepresentacionCanonica: representacion, HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico:     manifiestoCanonico,
		ManifiestoProcedenciaHuellaSHA256: huellaManifiesto,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
	return resultado, ahora, ahora.Add(time.Minute)
}
