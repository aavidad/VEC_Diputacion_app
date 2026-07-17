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

	"vec-diputacion-granada/internal/vec/domain"
)

func TestOrdenRegistroAutorizacionSolicitudLigadaV2ConservaPreimagenDefensiva(t *testing.T) {
	decision, _ := decisionAutorizacionSolicitudLigadaV2Prueba(t)
	referencia := referenciaMotivoAutorizacionPuertoV2Prueba(claveMotivoAutorizacionPuertoV2Prueba)
	orden, err := NuevaOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(decision, referencia)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := orden.Datos()
	if err != nil || datos.ReferenciaMotivo != referencia ||
		datos.Decision.DecisionRef != decision.DecisionRef {
		t.Fatalf("datos inesperados: %+v err=%v", datos, err)
	}
	datos.Decision.CamposPermitidos[0] = "campo_mutado"
	datos.Decision.PoliticasHuellasSHA256[datos.Decision.PoliticasRefs[0]] = strings.Repeat("9", 64)
	nuevos, err := orden.Datos()
	if err != nil || nuevos.Decision.CamposPermitidos[0] == "campo_mutado" ||
		nuevos.Decision.PoliticasHuellasSHA256[nuevos.Decision.PoliticasRefs[0]] == strings.Repeat("9", 64) {
		t.Fatal("Datos compartio memoria mutable con la orden")
	}
}

func TestOrdenRegistroAutorizacionSolicitudLigadaV2RechazaCrucesYMismatches(t *testing.T) {
	decision, _ := decisionAutorizacionSolicitudLigadaV2Prueba(t)
	referencia := referenciaMotivoAutorizacionPuertoV2Prueba(claveMotivoAutorizacionPuertoV2Prueba)
	v1, _ := decisionAutorizacionReforzadaPrueba(t)
	if _, err := NuevaOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(v1, referencia); !errors.Is(err, ErrOrdenRegistroAutorizacionSolicitudLigadaV2Invalida) {
		t.Fatalf("decision V1 aceptada: %v", err)
	}
	for nombre, mutar := range map[string]func(*domainReferenciaMotivoPrueba){
		"clave": func(p *domainReferenciaMotivoPrueba) {
			p.referencia.EntradaClave = claveMotivoAutorizacionPuertoV2Alternativa
		},
		"version": func(p *domainReferenciaMotivoPrueba) { p.referencia.CatalogoVersion++ },
		"huella catalogo": func(p *domainReferenciaMotivoPrueba) {
			p.referencia.CatalogoHuellaSHA256 = strings.Repeat("8", 64)
		},
		"huella decision": func(p *domainReferenciaMotivoPrueba) {
			p.decision.MotivoHuellaSHA256 = strings.Repeat("7", 64)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			prueba := domainReferenciaMotivoPrueba{decision: decision, referencia: referencia}
			mutar(&prueba)
			if _, err := NuevaOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(
				prueba.decision,
				prueba.referencia,
			); !errors.Is(err, ErrOrdenRegistroAutorizacionSolicitudLigadaV2Invalida) {
				t.Fatalf("preimagen distinta aceptada: %v", err)
			}
		})
	}
}

type domainReferenciaMotivoPrueba struct {
	decision   domain.DecisionAutorizacion
	referencia domain.ReferenciaEntradaCatalogo
}

func TestOrdenRegistroAutorizacionSolicitudLigadaV2BloqueaCodecsYFormato(t *testing.T) {
	decision, _ := decisionAutorizacionSolicitudLigadaV2Prueba(t)
	referencia := referenciaMotivoAutorizacionPuertoV2Prueba(claveMotivoAutorizacionPuertoV2Prueba)
	orden, err := NuevaOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(decision, referencia)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := orden.Datos()
	if err != nil {
		t.Fatal(err)
	}
	for nombre, valor := range map[string]any{"orden": orden, "datos": datos} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("JSON no bloqueado: %v", err)
			}
			if _, err := xml.Marshal(valor); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("XML no bloqueado: %v", err)
			}
			var salida bytes.Buffer
			if err := gob.NewEncoder(&salida).Encode(valor); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("Gob no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalBinary() ([]byte, error) }).MarshalBinary(); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("binario no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalText() ([]byte, error) }).MarshalText(); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("texto no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalCBOR() ([]byte, error) }).MarshalCBOR(); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("CBOR no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalYAML() (any, error) }).MarshalYAML(); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("YAML no bloqueado: %v", err)
			}
			texto := fmt.Sprintf("%v %+v %#v", valor, valor, valor)
			if strings.Contains(texto, decision.DecisionRef) || strings.Contains(texto, referencia.EntradaClave) {
				t.Fatalf("formato filtro contenido: %q", texto)
			}
		})
	}
	for nombre, destino := range map[string]any{
		"orden": &OrdenRegistroDecisionAutorizacionSolicitudLigadaV2{},
		"datos": &DatosOrdenRegistroDecisionAutorizacionSolicitudLigadaV2{},
	} {
		t.Run("decodificar "+nombre, func(t *testing.T) {
			if err := json.Unmarshal([]byte(`{}`), destino); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("JSON no bloqueado: %v", err)
			}
			if err := xml.Unmarshal([]byte(`<orden/>`), destino); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("XML no bloqueado: %v", err)
			}
			if err := destino.(interface{ GobDecode([]byte) error }).GobDecode(nil); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("Gob no bloqueado: %v", err)
			}
			if err := destino.(interface{ UnmarshalBinary([]byte) error }).UnmarshalBinary(nil); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("binario no bloqueado: %v", err)
			}
			if err := destino.(interface{ UnmarshalText([]byte) error }).UnmarshalText(nil); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("texto no bloqueado: %v", err)
			}
			if err := destino.(interface{ UnmarshalCBOR([]byte) error }).UnmarshalCBOR([]byte{0xa0}); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("CBOR no bloqueado: %v", err)
			}
			if err := destino.(interface{ UnmarshalYAML(func(any) error) error }).UnmarshalYAML(
				func(any) error { return nil },
			); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("YAML no bloqueado: %v", err)
			}
		})
	}
}
