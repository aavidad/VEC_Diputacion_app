package ports

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

const (
	claveMotivoAutorizacionPuertoV2Prueba           = "motivo_22222222222222222222222222222222"
	claveMotivoAutorizacionPuertoV2Alternativa      = "motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	referenciaCorrelacionAutorizacionPuertoV2Prueba = "correlacion_22222222222222222222222222222222"
)

func TestEvidenciaSolicitudLigadaV2NoConvierteV1YLigaMotivo(t *testing.T) {
	v1, verificadaEn := decisionAutorizacionReforzadaPrueba(t)
	if _, err := NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(v1, verificadaEn); !errors.Is(err, ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("decision V1 convertida a capacidad V2: %v", err)
	}

	v2, verificadaEn := decisionAutorizacionSolicitudLigadaV2Prueba(t)
	motivo := referenciaMotivoAutorizacionPuertoV2Prueba(claveMotivoAutorizacionPuertoV2Prueba)
	evidencia, err := NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(v2, verificadaEn)
	if err != nil {
		t.Fatalf("crear evidencia V2: %v", err)
	}
	if _, err := NuevaEvidenciaUsoDecisionAutorizacion(v2, verificadaEn); !errors.Is(err, ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("decision V2 degradada a capacidad V1: %v", err)
	}
	if err := evidencia.ValidarMotivo(motivo); err != nil {
		t.Fatalf("motivo exacto no reconocido: %v", err)
	}
	if err := evidencia.ValidarMotivo(referenciaMotivoAutorizacionPuertoV2Prueba(claveMotivoAutorizacionPuertoV2Alternativa)); !errors.Is(err, ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("motivo distinto aceptado: %v", err)
	}
	datos, err := evidencia.Datos()
	if err != nil || datos.EsquemaHuella != EsquemaHuellaDecisionAutorizacionReforzadaV2 {
		t.Fatalf("datos V2 invalidos: esquema=%q err=%v", datos.EsquemaHuella, err)
	}
	if err := (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{
		Decision: v2,
	}).ValidarMotivo(motivo); !errors.Is(err, ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("proyeccion fabricada valido motivo: %v", err)
	}
}

func TestEvidenciaSolicitudLigadaV2RechazaCombinacionesParciales(t *testing.T) {
	decision, verificadaEn := decisionAutorizacionSolicitudLigadaV2Prueba(t)
	casos := []struct {
		nombre string
		mutar  func(*domain.DecisionAutorizacion)
	}{
		{"sin esquema solicitud", func(d *domain.DecisionAutorizacion) { d.EsquemaHuellaSolicitud = "" }},
		{"sin huella solicitud", func(d *domain.DecisionAutorizacion) { d.SolicitudHuellaSHA256 = "" }},
		{"sin esquema motivo", func(d *domain.DecisionAutorizacion) { d.EsquemaHuellaMotivo = "" }},
		{"sin huella motivo", func(d *domain.DecisionAutorizacion) { d.MotivoHuellaSHA256 = "" }},
		{"huella solicitud nula", func(d *domain.DecisionAutorizacion) { d.SolicitudHuellaSHA256 = strings.Repeat("0", 64) }},
		{"huella motivo nula", func(d *domain.DecisionAutorizacion) { d.MotivoHuellaSHA256 = strings.Repeat("0", 64) }},
		{"correlacion legible", func(d *domain.DecisionAutorizacion) { d.CorrelacionRef = "correlacion:expediente:1234" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			candidata := clonarDecisionAutorizacionCanonica(decision)
			caso.mutar(&candidata)
			if _, err := NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(candidata, verificadaEn); !errors.Is(err, ErrEvidenciaUsoDecisionAutorizacionInvalida) {
				t.Fatalf("combinacion parcial aceptada: %v", err)
			}
		})
	}
}

func TestEvidenciaSolicitudLigadaV2BloqueaTodosLosCodecs(t *testing.T) {
	decision, verificadaEn := decisionAutorizacionSolicitudLigadaV2Prueba(t)
	evidencia, err := NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(decision, verificadaEn)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := evidencia.Datos()
	if err != nil {
		t.Fatal(err)
	}
	for nombre, valor := range map[string]any{"evidencia": evidencia, "datos": datos} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("JSON no bloqueado: %v", err)
			}
			if _, err := xml.Marshal(valor); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("XML no bloqueado: %v", err)
			}
			var destino bytes.Buffer
			if err := gob.NewEncoder(&destino).Encode(valor); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("Gob no bloqueado: %v", err)
			}
			if codificador, ok := valor.(interface{ MarshalBinary() ([]byte, error) }); !ok {
				t.Fatal("falta bloqueo binario")
			} else if _, err := codificador.MarshalBinary(); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("binario no bloqueado: %v", err)
			}
			if codificador, ok := valor.(interface{ MarshalText() ([]byte, error) }); !ok {
				t.Fatal("falta bloqueo de texto")
			} else if _, err := codificador.MarshalText(); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("texto no bloqueado: %v", err)
			}
			if codificador, ok := valor.(interface{ MarshalCBOR() ([]byte, error) }); !ok {
				t.Fatal("falta bloqueo CBOR")
			} else if _, err := codificador.MarshalCBOR(); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("CBOR no bloqueado: %v", err)
			}
			if codificador, ok := valor.(interface{ MarshalYAML() (any, error) }); !ok {
				t.Fatal("falta bloqueo YAML")
			} else if _, err := codificador.MarshalYAML(); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("YAML no bloqueado: %v", err)
			}
			if salida := fmt.Sprintf("%v", valor); strings.Contains(salida, decision.DecisionRef) {
				t.Fatalf("formato filtro contenido: %q", salida)
			}
		})
	}
	for nombre, destino := range map[string]interface{ UnmarshalYAML(func(any) error) error }{
		"evidencia": &EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{},
		"datos":     &DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{},
	} {
		if err := destino.UnmarshalYAML(func(any) error { return nil }); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
			t.Fatalf("%s: decodificacion YAML no bloqueada: %v", nombre, err)
		}
	}
	for nombre, destino := range map[string]interface{ UnmarshalCBOR([]byte) error }{
		"evidencia": &EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{},
		"datos":     &DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{},
	} {
		if err := destino.UnmarshalCBOR([]byte{0xa0}); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
			t.Fatalf("%s: decodificacion CBOR no bloqueada: %v", nombre, err)
		}
	}
}

func TestRepresentacionDecisionSolicitudLigadaV2TieneVectorIndependiente(t *testing.T) {
	decision, _ := decisionAutorizacionSolicitudLigadaV2Prueba(t)
	contenido, err := RepresentacionCanonicaDecisionAutorizacionReforzadaV2(decision)
	if err != nil {
		t.Fatal(err)
	}
	suma, err := huellaDecisionAutorizacionReforzadaV2(decision)
	if err != nil {
		t.Fatal(err)
	}
	const huellaEsperada = "cdaa301baf2443a596ac3d17db1814479aee3ddda772f98a9f325a9c8b1eb553"
	if suma != huellaEsperada {
		t.Fatalf("vector V2 cambio: huella=%s bytes=%d", suma, len(contenido))
	}
	var objeto map[string]json.RawMessage
	if json.Unmarshal(contenido, &objeto) != nil || len(objeto) != 34 ||
		string(objeto["esquema"]) != `"`+EsquemaHuellaDecisionAutorizacionReforzadaV2+`"` {
		t.Fatalf("perfil V2 inesperado: campos=%d", len(objeto))
	}
}

func decisionAutorizacionSolicitudLigadaV2Prueba(
	t *testing.T,
) (domain.DecisionAutorizacion, time.Time) {
	t.Helper()
	decision, verificadaEn := decisionAutorizacionReforzadaPrueba(t)
	decision.CorrelacionRef = referenciaCorrelacionAutorizacionPuertoV2Prueba
	recurso := domain.RecursoAutorizable{
		Referencia: decision.RecursoRef, ModuloID: decision.ModuloID, Tipo: decision.TipoRecurso,
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	decision.ContextoRecursoHuellaSHA256 = huellaContexto
	ligarDecisionAutorizacionReforzadaPrueba(
		t,
		&decision,
		recurso,
		referenciaMotivoAutorizacionPuertoV2Prueba(claveMotivoAutorizacionPuertoV2Prueba),
	)
	if err := decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2(); err != nil {
		t.Fatalf("decision V2 de prueba invalida: %v", err)
	}
	return decision, verificadaEn
}

func referenciaMotivoAutorizacionPuertoV2Prueba(clave string) domain.ReferenciaEntradaCatalogo {
	return domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 4,
		CatalogoHuellaSHA256: "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		EntradaClave:         clave,
	}
}
