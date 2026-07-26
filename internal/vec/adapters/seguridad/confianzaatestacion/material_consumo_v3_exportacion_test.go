package confianzaatestacion

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/xml"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

type piezasExportacionMaterialV3Prueba struct {
	capacidad []byte
	resumen   ports.ResumenCapacidadAtestacionAutorizacionV3
	decision  []byte
	motivo    []byte
	contexto  []byte
	persona   uint64
	perfil    uint64
	payload   []byte
	cose      []byte
	evidencia []byte
	spki      []byte
}

func piezasExportacionMaterialV3Desde(
	e ports.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) piezasExportacionMaterialV3Prueba {
	return piezasExportacionMaterialV3Prueba{
		capacidad: e.CapacidadCanonica(),
		resumen:   e.ResumenCapacidad(),
		decision:  e.DecisionCanonica(),
		motivo:    e.MotivoCanonico(),
		contexto:  e.ContextoActorCanonico(),
		persona:   e.PersonaVersion(),
		perfil:    e.PerfilVersion(),
		payload:   e.PayloadVECAD3(),
		cose:      e.SobreCOSESign1(),
		evidencia: e.EvidenciaVerificacion(),
		spki:      e.RaizPublicaSPKI(),
	}
}

func construirExportacionMaterialV3Prueba(
	p piezasExportacionMaterialV3Prueba,
) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		p.capacidad,
		p.resumen,
		p.decision,
		p.motivo,
		p.contexto,
		p.persona,
		p.perfil,
		p.payload,
		p.cose,
		p.evidencia,
		p.spki,
	)
}

func TestExportacionMaterialConsumoV3CopiaTodasLasEntradas(
	t *testing.T,
) {
	t.Parallel()
	escenario := nuevoEscenarioMaterialConsumoV3Prueba(t)
	base, _ := escenario.material.ExportarMaterialParaConsumidor()
	piezas := piezasExportacionMaterialV3Desde(base)
	esperadas := piezasExportacionMaterialV3Desde(base)
	exportacion, err := construirExportacionMaterialV3Prueba(piezas)
	if err != nil {
		t.Fatal(err)
	}
	for _, contenido := range [][]byte{
		piezas.capacidad,
		piezas.decision,
		piezas.motivo,
		piezas.contexto,
		piezas.payload,
		piezas.cose,
		piezas.evidencia,
		piezas.spki,
	} {
		contenido[len(contenido)/2] ^= 0xff
	}
	obtenida := piezasExportacionMaterialV3Desde(exportacion)
	for indice, par := range [][2][]byte{
		{esperadas.capacidad, obtenida.capacidad},
		{esperadas.decision, obtenida.decision},
		{esperadas.motivo, obtenida.motivo},
		{esperadas.contexto, obtenida.contexto},
		{esperadas.payload, obtenida.payload},
		{esperadas.cose, obtenida.cose},
		{esperadas.evidencia, obtenida.evidencia},
		{esperadas.spki, obtenida.spki},
	} {
		if !bytes.Equal(par[0], par[1]) {
			t.Fatalf("entrada %d no fue copiada", indice)
		}
	}
}

func TestExportacionMaterialConsumoV3ImponeLimitesExactosSQL(
	t *testing.T,
) {
	t.Parallel()
	escenario := nuevoEscenarioMaterialConsumoV3Prueba(t)
	exportacion, _ := escenario.material.ExportarMaterialParaConsumidor()
	base := piezasExportacionMaterialV3Desde(exportacion)
	casos := map[string]func(*piezasExportacionMaterialV3Prueba){
		"capacidad_corta": func(p *piezasExportacionMaterialV3Prueba) {
			p.capacidad = make([]byte, ports.TamanoMinimoCapacidadCanonicaV3-1)
		},
		"capacidad_larga": func(p *piezasExportacionMaterialV3Prueba) {
			p.capacidad = make([]byte, ports.TamanoMaximoCapacidadCanonicaV3+1)
		},
		"decision_vacia": func(p *piezasExportacionMaterialV3Prueba) {
			p.decision = nil
		},
		"decision_larga": func(p *piezasExportacionMaterialV3Prueba) {
			p.decision = make([]byte, ports.TamanoMaximoDecisionCanonicaV3+1)
		},
		"motivo_vacio": func(p *piezasExportacionMaterialV3Prueba) {
			p.motivo = nil
		},
		"motivo_largo": func(p *piezasExportacionMaterialV3Prueba) {
			p.motivo = make([]byte, ports.TamanoMaximoMotivoCanonicoV3+1)
		},
		"contexto_vacio": func(p *piezasExportacionMaterialV3Prueba) {
			p.contexto = nil
		},
		"contexto_largo": func(p *piezasExportacionMaterialV3Prueba) {
			p.contexto = make([]byte, ports.TamanoMaximoContextoActorCanonicoV3+1)
		},
		"persona_cero": func(p *piezasExportacionMaterialV3Prueba) {
			p.persona = 0
		},
		"persona_fuera": func(p *piezasExportacionMaterialV3Prueba) {
			p.persona = ports.VersionMaximaExactaMaterialConsumoV3 + 1
		},
		"perfil_cero": func(p *piezasExportacionMaterialV3Prueba) {
			p.perfil = 0
		},
		"perfil_fuera": func(p *piezasExportacionMaterialV3Prueba) {
			p.perfil = ports.VersionMaximaExactaMaterialConsumoV3 + 1
		},
		"payload_vacio": func(p *piezasExportacionMaterialV3Prueba) {
			p.payload = nil
		},
		"payload_largo": func(p *piezasExportacionMaterialV3Prueba) {
			p.payload = make([]byte, ports.TamanoMaximoPayloadVECAD3+1)
		},
		"cose_vacio": func(p *piezasExportacionMaterialV3Prueba) {
			p.cose = nil
		},
		"cose_largo": func(p *piezasExportacionMaterialV3Prueba) {
			p.cose = make([]byte, ports.TamanoMaximoSobreCOSESign1V3+1)
		},
		"evidencia_vacia": func(p *piezasExportacionMaterialV3Prueba) {
			p.evidencia = nil
		},
		"evidencia_larga": func(p *piezasExportacionMaterialV3Prueba) {
			p.evidencia = make([]byte, ports.TamanoMaximoEvidenciaVerificacionV3+1)
		},
		"spki_corta": func(p *piezasExportacionMaterialV3Prueba) {
			p.spki = make([]byte, ports.TamanoRaizPublicaSPKIEd25519V3-1)
		},
		"spki_larga": func(p *piezasExportacionMaterialV3Prueba) {
			p.spki = make([]byte, ports.TamanoRaizPublicaSPKIEd25519V3+1)
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			piezas := base
			mutar(&piezas)
			if _, err := construirExportacionMaterialV3Prueba(piezas); !errors.Is(
				err,
				ports.ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida,
			) {
				t.Fatalf("límite %s aceptado: %v", nombre, err)
			}
		})
	}
}

func TestExportacionMaterialConsumoV3RechazaSPKIHostil(
	t *testing.T,
) {
	t.Parallel()
	escenario := nuevoEscenarioMaterialConsumoV3Prueba(t)
	exportacion, _ := escenario.material.ExportarMaterialParaConsumidor()
	base := piezasExportacionMaterialV3Desde(exportacion)
	privadaX25519, err := ecdh.X25519().NewPrivateKey(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}
	spkiX25519, err := x509.MarshalPKIXPublicKey(privadaX25519.PublicKey())
	if err != nil || len(spkiX25519) != ports.TamanoRaizPublicaSPKIEd25519V3 {
		t.Fatalf("preparar SPKI X25519 de 44 bytes: %v (%d)", err, len(spkiX25519))
	}
	privadaRSA, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	spkiRSA, err := x509.MarshalPKIXPublicKey(&privadaRSA.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	noCanonica := bytes.Clone(base.spki)
	noCanonica[1] ^= 1
	for nombre, spki := range map[string][]byte{
		"x25519_44_bytes": spkiX25519,
		"rsa":             spkiRSA,
		"der_no_canonico": noCanonica,
		"hostil_44_bytes": bytes.Repeat([]byte{0xff}, 44),
	} {
		t.Run(nombre, func(t *testing.T) {
			piezas := base
			piezas.spki = spki
			if _, err := construirExportacionMaterialV3Prueba(piezas); !errors.Is(
				err,
				ports.ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida,
			) {
				t.Fatalf("SPKI %s aceptado: %v", nombre, err)
			}
		})
	}
}

func TestConstructorEstructuralMaterialConsumoV3NoConcedeAutoridad(
	t *testing.T,
) {
	t.Parallel()
	escenario := nuevoEscenarioMaterialConsumoV3Prueba(t)
	exportacion, _ := escenario.material.ExportarMaterialParaConsumidor()
	piezas := piezasExportacionMaterialV3Desde(exportacion)
	piezas.decision = bytes.Clone(piezas.motivo)
	cruce, err := construirExportacionMaterialV3Prueba(piezas)
	if err != nil || cruce.ValidarEstructura() != nil {
		t.Fatalf("el transporte debe limitar, no autorizar: %v", err)
	}
	if _, autoriza := any(cruce).(ports.ExportadorMaterialConsumoAutorizacionAtestadaV3); autoriza {
		t.Fatal("el transporte estructural se convirtió en autoridad")
	}
}

type bloqueadorCodecsMaterialConsumoV3Prueba interface {
	MarshalJSON() ([]byte, error)
	UnmarshalJSON([]byte) error
	MarshalText() ([]byte, error)
	UnmarshalText([]byte) error
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
	GobEncode() ([]byte, error)
	GobDecode([]byte) error
	MarshalCBOR() ([]byte, error)
	UnmarshalCBOR([]byte) error
	MarshalYAML() (any, error)
	UnmarshalYAML(func(any) error) error
}

func TestMaterialConsumoV3BloqueaTambienDecodificacion(
	t *testing.T,
) {
	t.Parallel()
	escenario := nuevoEscenarioMaterialConsumoV3Prueba(t)
	exportacion, _ := escenario.material.ExportarMaterialParaConsumidor()
	for nombre, valor := range map[string]bloqueadorCodecsMaterialConsumoV3Prueba{
		"material":    &escenario.material,
		"exportacion": &exportacion,
	} {
		t.Run(nombre, func(t *testing.T) {
			errores := []error{}
			_, err := valor.MarshalJSON()
			errores = append(errores, err, valor.UnmarshalJSON([]byte(`{}`)))
			_, err = valor.MarshalText()
			errores = append(errores, err, valor.UnmarshalText([]byte("x")))
			_, err = valor.MarshalBinary()
			errores = append(errores, err, valor.UnmarshalBinary([]byte{1}))
			_, err = valor.GobEncode()
			errores = append(errores, err, valor.GobDecode([]byte{1}))
			_, err = valor.MarshalCBOR()
			errores = append(errores, err, valor.UnmarshalCBOR([]byte{0xa0}))
			_, err = valor.MarshalYAML()
			errores = append(errores, err, valor.UnmarshalYAML(func(any) error {
				return nil
			}))
			for indice, err := range errores {
				if err == nil {
					t.Fatalf("codec %d aceptó material", indice)
				}
			}
		})
	}
	var material MaterialConsumoAutorizacionAtestadaV3
	var vacia ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	for nombre, destino := range map[string]any{
		"material":    &material,
		"exportacion": &vacia,
	} {
		if json.Unmarshal([]byte(`{}`), destino) == nil ||
			xml.Unmarshal([]byte(`<material/>`), destino) == nil {
			t.Fatalf("un codec genérico decodificó %s", nombre)
		}
	}
	if salida := strings.ToLower(material.String()); strings.Contains(
		salida,
		"capacidad",
	) && !strings.Contains(salida, "redactada") {
		t.Fatal("valor cero no permanece redactado")
	}
}
