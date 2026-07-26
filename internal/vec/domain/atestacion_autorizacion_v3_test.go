package domain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMensajeAtestacionAutorizacionV3EsDeterministaYSeparaV2(t *testing.T) {
	cabecera, decision, motivo, contexto := escenarioAtestacionAutorizacionV3Prueba(t)

	primero, err := SerializarMensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		motivo,
		contexto,
	)
	if err != nil {
		t.Fatalf("serializar VEC-AD-3: %v", err)
	}
	segundo, err := SerializarMensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		motivo,
		contexto,
	)
	if err != nil || !bytes.Equal(primero, segundo) {
		t.Fatalf("VEC-AD-3 no es determinista: %v", err)
	}
	prefijo := append([]byte(EsquemaMensajeAtestacionAutorizacionV3), 0)
	if !bytes.HasPrefix(primero, prefijo) ||
		bytes.HasPrefix(
			primero,
			append([]byte(EsquemaMensajeAtestacionAutorizacionV2), 0),
		) {
		t.Fatal("dominio criptografico V3 ausente o cruzado con V2")
	}
	if version := binary.BigEndian.Uint16(
		primero[len(prefijo) : len(prefijo)+2],
	); version != VersionFormatoAtestacionAutorizacionV3 {
		t.Fatalf("version = %d", version)
	}
	if longitud := binary.BigEndian.Uint64(primero[len(primero)-8:]); longitud != uint64(len(primero)) {
		t.Fatalf("longitud final = %d; esperada %d", longitud, len(primero))
	}
	huella, err := HuellaSHA256MensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		motivo,
		contexto,
	)
	const longitudEsperada = 6428
	const huellaEsperada = "169da880b22184f46828d9e2e5b51071002c3d8a8d1e858a13d53bb0c919dc8f"
	if err != nil || len(primero) != longitudEsperada || huella != huellaEsperada {
		t.Fatalf(
			"cambio incompatible VEC-AD-3: longitud=%d huella=%q error=%v",
			len(primero),
			huella,
			err,
		)
	}
}

func TestMensajeAtestacionAutorizacionV3LigaContextoMotivoYCabecera(t *testing.T) {
	cabecera, decision, motivo, contexto := escenarioAtestacionAutorizacionV3Prueba(t)
	base, err := SerializarMensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		motivo,
		contexto,
	)
	if err != nil {
		t.Fatal(err)
	}

	otraCabecera := cabecera
	otraCabecera.Audiencia = "vec-diputacion/pruebas/contratacion-temporal/otra"
	otro, err := SerializarMensajeAtestacionAutorizacionV3(
		otraCabecera,
		decision,
		motivo,
		contexto,
	)
	if err != nil || bytes.Equal(base, otro) {
		t.Fatalf("audiencia no quedo ligada: %v", err)
	}

	otroContexto := contexto
	otroContexto.RegistroContextoRef = "rca_otra234567890abcdefghijklmn"
	if _, err := SerializarMensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		motivo,
		otroContexto,
	); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
		t.Fatalf("contexto adulterado aceptado: %v", err)
	}

	otroMotivo := motivo
	otroMotivo.EntradaClave = "motivo_44444444444444444444444444444444"
	if _, err := SerializarMensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		otroMotivo,
		contexto,
	); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
		t.Fatalf("motivo cruzado aceptado: %v", err)
	}
}

func TestMensajeAtestacionAutorizacionV3RechazaDenegacionCerosYCruces(t *testing.T) {
	cabecera, decision, motivo, contexto := escenarioAtestacionAutorizacionV3Prueba(t)
	denegada := clonarDecisionAutorizacionLigadaV3Prueba(t, decision)
	denegada.datos.concedida = false
	denegada.datos.codigo = "denegada"
	denegada.datos.selloSHA256, _ = huellaCanonicaDecisionAutorizacionV3(denegada.datos)

	casos := []struct {
		nombre   string
		cabecera CabeceraAtestacionAutorizacionV3
		decision DecisionAutorizacionLigadaV3
		motivo   ReferenciaEntradaCatalogo
		contexto ResultadoContextoActorRegistradoV2
	}{
		{"cabecera cero", CabeceraAtestacionAutorizacionV3{}, decision, motivo, contexto},
		{"decision cero", cabecera, DecisionAutorizacionLigadaV3{}, motivo, contexto},
		{"motivo cero", cabecera, decision, ReferenciaEntradaCatalogo{}, contexto},
		{"contexto cero", cabecera, decision, motivo, ResultadoContextoActorRegistradoV2{}},
		{"denegacion", cabecera, denegada, motivo, contexto},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := SerializarMensajeAtestacionAutorizacionV3(
				caso.cabecera,
				caso.decision,
				caso.motivo,
				caso.contexto,
			); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
				t.Fatalf("caso inseguro aceptado: %v", err)
			}
		})
	}
}

func TestMensajeAtestacionAutorizacionV3NoModificaEntradas(t *testing.T) {
	cabecera, decision, motivo, contexto := escenarioAtestacionAutorizacionV3Prueba(t)
	representacion := append([]byte(nil), contexto.RepresentacionCanonica...)
	manifiesto := append([]byte(nil), contexto.ManifiestoProcedenciaCanonico...)

	if _, err := SerializarMensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		motivo,
		contexto,
	); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(representacion, contexto.RepresentacionCanonica) ||
		!bytes.Equal(manifiesto, contexto.ManifiestoProcedenciaCanonico) {
		t.Fatal("la serializacion modifico slices del contexto")
	}
}

func escenarioAtestacionAutorizacionV3Prueba(
	t *testing.T,
) (
	CabeceraAtestacionAutorizacionV3,
	DecisionAutorizacionLigadaV3,
	ReferenciaEntradaCatalogo,
	ResultadoContextoActorRegistradoV2,
) {
	t.Helper()
	instante := time.Date(2026, 7, 15, 10, 11, 12, 123_456_000, time.UTC)
	vinculo, contexto, _ := vinculoAutenticacionActorV2Prueba(t, instante)
	base, instantanea, emitida := escenarioDecisionAutorizacionV3Prueba(t)
	datos, err := base.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.VinculoAutenticacionActor = vinculo
	solicitud, err := NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		t.Fatal(err)
	}
	evidencia, err := NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud,
		instantanea,
		"dec_0123456789abcdef0123456789abcdef",
		emitida,
		emitida.Add(4*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	return CabeceraAtestacionAutorizacionV3{
		FormatoVersion: VersionFormatoAtestacionAutorizacionV3,
		Suite:          "VEC-AD-3-COSE-EDDSA-1",
		ClaveID:        "clave:prueba:vec-ad-3:2026-07",
		Audiencia:      "vec-diputacion/pruebas/contratacion-temporal",
	}, decision, datos.ReferenciaMotivo, contexto
}

func TestEsquemaAtestacionAutorizacionV3EsAcotado(t *testing.T) {
	cabecera, decision, motivo, contexto := escenarioAtestacionAutorizacionV3Prueba(t)
	cabecera.Audiencia = strings.Repeat("a", 513)
	if _, err := SerializarMensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		motivo,
		contexto,
	); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
		t.Fatalf("audiencia sobredimensionada aceptada: %v", err)
	}
}
