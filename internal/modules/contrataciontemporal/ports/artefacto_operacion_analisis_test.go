package ports

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestArtefactoAnalisisDerivaAutoridadSinAceptarlaDelDTO(t *testing.T) {
	solicitud, datos := artefactoAnalisisPrueba()
	artefacto, err := NuevoArtefactoAnalisisPreparado(solicitud, datos)
	if err != nil {
		t.Fatal(err)
	}
	analisis, err := DerivarAnalisisDesdeArtefacto(solicitud, artefacto)
	if err != nil {
		t.Fatal(err)
	}
	if analisis.ValidacionRC.FuenteRef != datos.FuenteRCRef ||
		analisis.ValidacionRC.ReciboRef != datos.ReciboRCRef ||
		analisis.ValidacionRC.ValidadaEn != datos.ValidadaEn ||
		analisis.CostePrevisto == datos.CostePrevisto ||
		analisis.CostePrevisto.Centimos != datos.CostePrevisto.Centimos ||
		analisis.FuenteCosteRef != datos.FuenteCosteRef {
		t.Fatal("la proyección no derivó o no clonó la autoridad del artefacto")
	}
	for _, nombre := range []string{
		"Analisis", "ValidacionRC", "CostePrevisto", "FuenteCosteRef",
		"ReciboRCRef", "ValidadaEn",
	} {
		if _, existe := reflect.TypeOf(SolicitudPrepararArtefactoAnalisis{}).
			FieldByName(nombre); existe {
			t.Fatalf("el DTO acepta autoridad mediante %s", nombre)
		}
	}
}

func TestArtefactoAnalisisLigaTodasLasCoordenadas(t *testing.T) {
	solicitud, datos := artefactoAnalisisPrueba()
	casos := []struct {
		nombre    string
		modificar func(*DatosArtefactoAnalisis)
	}{
		{"referencia", func(d *DatosArtefactoAnalisis) {
			d.ArtefactoRef = "artefacto:prueba-distinto"
		}},
		{"huella", func(d *DatosArtefactoAnalisis) {
			d.ArtefactoHuellaSHA256 = strings.Repeat("0", 64)
		}},
		{"organizacion", func(d *DatosArtefactoAnalisis) {
			d.OrganizacionRef = "organizacion:prueba-distinta"
		}},
		{"expediente", func(d *DatosArtefactoAnalisis) {
			d.ExpedienteRef = "expediente:prueba-distinto"
		}},
		{"version", func(d *DatosArtefactoAnalisis) {
			d.VersionExpediente++
		}},
		{"entrada", func(d *DatosArtefactoAnalisis) {
			d.DatosFuncionales.EntradaRC.Referencia = "entrada:prueba-distinta"
		}},
		{"resultado_rc", func(d *DatosArtefactoAnalisis) {
			d.ResultadoRC = domain.RCRechazada
		}},
		{"fuente_rc", func(d *DatosArtefactoAnalisis) {
			d.FuenteRCRef = ""
		}},
		{"recibo_rc", func(d *DatosArtefactoAnalisis) {
			d.ReciboRCRef = ""
		}},
		{"instante_rc", func(d *DatosArtefactoAnalisis) {
			d.ValidadaEn = d.PreparadoEn.Add(time.Microsecond)
		}},
		{"coste", func(d *DatosArtefactoAnalisis) {
			d.CostePrevisto.Centimos = d.ImporteRC.Centimos + 1
		}},
		{"fuente_coste", func(d *DatosArtefactoAnalisis) {
			d.FuenteCosteRef = ""
		}},
		{"recibo_coste", func(d *DatosArtefactoAnalisis) {
			d.ReciboCosteRef = ""
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterado := clonarDatosArtefactoAnalisis(datos)
			caso.modificar(&alterado)
			if _, err := NuevoArtefactoAnalisisPreparado(
				solicitud,
				alterado,
			); !errors.Is(err, ErrArtefactoAnalisisNoConfiable) {
				t.Fatalf("coordenada adulterada aceptada: %v", err)
			}
		})
	}
}

func TestArtefactoAnalisisReservaIncrementoEnteroSeguro(t *testing.T) {
	solicitud, datos := artefactoAnalisisPrueba()
	solicitud.VersionExpediente =
		MaximoEnteroSeguroOperacionAnalisis - 1
	datos.VersionExpediente = solicitud.VersionExpediente
	if _, err := NuevoArtefactoAnalisisPreparado(
		solicitud,
		datos,
	); err != nil {
		t.Fatalf("última versión incrementable rechazada: %v", err)
	}
	solicitud.VersionExpediente =
		MaximoEnteroSeguroOperacionAnalisis
	datos.VersionExpediente = solicitud.VersionExpediente
	if _, err := NuevoArtefactoAnalisisPreparado(
		solicitud,
		datos,
	); !errors.Is(err, ErrArtefactoAnalisisNoConfiable) {
		t.Fatalf("versión sin hueco CAS aceptada: %v", err)
	}
}

func TestArtefactoAnalisisBloqueaTodosLosCodecs(t *testing.T) {
	solicitud, datos := artefactoAnalisisPrueba()
	artefacto, err := NuevoArtefactoAnalisisPreparado(solicitud, datos)
	if err != nil {
		t.Fatal(err)
	}
	comprobar := func(nombre string, err error) {
		t.Helper()
		if !errors.Is(err, ErrSerializacionOperacionAnalisisProhibida) {
			t.Fatalf("%s no quedó bloqueado: %v", nombre, err)
		}
	}
	_, err = json.Marshal(artefacto)
	comprobar("json", err)
	_, err = xml.Marshal(artefacto)
	comprobar("xml", err)
	_, err = artefacto.MarshalText()
	comprobar("text", err)
	_, err = artefacto.MarshalBinary()
	comprobar("binary", err)
	var destino bytes.Buffer
	comprobar("gob", gob.NewEncoder(&destino).Encode(artefacto))
	_, err = artefacto.MarshalCBOR()
	comprobar("cbor", err)
	_, err = artefacto.MarshalYAML()
	comprobar("yaml", err)

	var reconstruido ArtefactoAnalisisPreparado
	comprobar("json_decode", json.Unmarshal([]byte(`{}`), &reconstruido))
	comprobar("xml_decode", xml.Unmarshal([]byte(`<a/>`), &reconstruido))
	comprobar("text_decode", reconstruido.UnmarshalText(nil))
	comprobar("binary_decode", reconstruido.UnmarshalBinary(nil))
	comprobar("gob_decode", reconstruido.GobDecode(nil))
	comprobar("cbor_decode", reconstruido.UnmarshalCBOR(nil))
	comprobar("yaml_decode", reconstruido.UnmarshalYAML(func(any) error {
		return nil
	}))
}

func artefactoAnalisisPrueba() (
	SolicitudPrepararArtefactoAnalisis,
	DatosArtefactoAnalisis,
) {
	instante := time.Date(2026, time.July, 23, 10, 0, 0, 0, time.UTC)
	fechaRC := time.Date(2026, time.July, 22, 0, 0, 0, 0, time.UTC)
	importeRC := domain.Importe{Centimos: 5_000_000, Moneda: "EUR"}
	coste := domain.Importe{Centimos: 4_000_000, Moneda: "EUR"}
	funcionales := DatosFuncionalesOperacionAnalisis{
		ModalidadClave: "modalidad.prueba",
		CategoriaRef:   "categoria:prueba-001",
		GrupoSubgrupo:  "C2",
		CausaClave:     "causa.prueba",
		Periodo: domain.PeriodoPrevisto{
			Inicio: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
			Fin:    time.Date(2026, time.October, 31, 0, 0, 0, 0, time.UTC),
		},
		PorcentajeJornada: domain.JornadaCompletaDiezmilesimas,
		EntradaRC: domain.VinculoEntradaRC{
			Referencia:   "entrada:rc-prueba-001",
			HuellaSHA256: strings.Repeat("1", 64),
		},
	}
	solicitud := SolicitudPrepararArtefactoAnalisis{
		ArtefactoRef:      "artefacto:analisis-prueba-001",
		OrganizacionRef:   "organizacion:prueba-001",
		ExpedienteRef:     "expediente:prueba-001",
		VersionExpediente: 1,
		DatosFuncionales:  funcionales,
		SolicitadaEn:      instante.Add(-time.Minute),
	}
	datos := DatosArtefactoAnalisis{
		ArtefactoRef:          solicitud.ArtefactoRef,
		ArtefactoHuellaSHA256: strings.Repeat("2", 64),
		OrganizacionRef:       solicitud.OrganizacionRef,
		ExpedienteRef:         solicitud.ExpedienteRef,
		VersionExpediente:     solicitud.VersionExpediente,
		DatosFuncionales:      funcionales,
		ResultadoRC:           domain.RCValidada,
		FuenteRCRef:           "fuente:rc-prueba-001",
		ReciboRCRef:           "recibo:rc-prueba-001",
		ValidadaEn:            instante.Add(-30 * time.Second),
		FechaRC:               &fechaRC,
		NumeroRC:              "rc:prueba-001",
		ImporteRC:             &importeRC,
		DocumentoRCRef:        "documento:rc-prueba-001",
		CostePrevisto:         &coste,
		FuenteCosteRef:        "fuente:coste-prueba-001",
		ReciboCosteRef:        "recibo:coste-prueba-001",
		CalculadoEn:           instante.Add(-20 * time.Second),
		PreparadoEn:           instante,
	}
	return solicitud, datos
}
