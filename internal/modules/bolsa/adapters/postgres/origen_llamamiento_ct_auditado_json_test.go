package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func origenAuditadoCanonicoPrueba(t *testing.T) (ports.OrigenLlamamientoCT, []byte) {
	t.Helper()
	ahora := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	huella := func(b byte) string { return strings.Repeat(string(b), 64) }
	bolsa, err := domain.NuevaBolsaConstituida(domain.AltaBolsaConstituida{
		BolsaRef: "bolsa:ct:prueba", Version: 1, ProcesoRef: "proceso:ct:prueba",
		CategoriaRef: "categoria:ct:prueba", ListadoDefinitivoRef: "listado:ct:prueba",
		VersionListado: 1, HuellaListadoSHA256: huella('a'),
		ResolucionConstitucionRef: "resolucion:ct:prueba", HuellaResolucionSHA256: huella('b'),
		ConstituidaEn: ahora.Add(-48 * time.Hour), VigenteDesde: ahora.Add(-24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	participacion, err := domain.NuevaParticipacionBolsa(domain.AltaParticipacionBolsa{
		ParticipacionRef: "participacion:ct:prueba", BolsaRef: bolsa.BolsaRef,
		SujetoRef: "sujeto:ct:prueba", Version: 1, AltaEn: ahora.Add(-12 * time.Hour),
		Situaciones: []domain.SituacionParticipacionBolsa{{
			Secuencia: 1, EstadoClave: "estado_gobernado", EstadoVersion: 1,
			HuellaEstadoSHA256: huella('c'), CausaClave: "causa_gobernada", CausaVersion: 1,
			HuellaCausaSHA256: huella('d'), DecisionRef: "decision:ct:prueba",
			HuellaDecisionSHA256: huella('e'), Desde: ahora.Add(-12 * time.Hour),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	instantanea, err := domain.NuevaInstantaneaOrdenBolsa(domain.AltaInstantaneaOrdenBolsa{
		InstantaneaRef: "instantanea:ct:prueba", Version: 1, Bolsa: bolsa,
		ReferidaEn: ahora, GeneradaEn: ahora,
		Entradas: []domain.EntradaOrdenBolsa{{Orden: 1, Participacion: participacion}},
	})
	if err != nil {
		t.Fatal(err)
	}
	canonBolsa, err := json.Marshal(bolsa)
	if err != nil {
		t.Fatal(err)
	}
	canonInstantanea, err := json.Marshal(instantanea)
	if err != nil {
		t.Fatal(err)
	}
	huellaBytes := func(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
	orden := uint64(1)
	origen := ports.OrigenLlamamientoCT{
		ActaRef: "acta:ct:prueba", CategoriaRef: bolsa.CategoriaRef,
		BolsaRef: bolsa.BolsaRef, VersionBolsa: bolsa.Version,
		HuellaBolsaSHA256: huellaBytes(canonBolsa), BolsaCanonica: canonBolsa,
		InstantaneaRef: instantanea.InstantaneaRef, VersionInstantanea: instantanea.Version,
		HuellaInstantaneaSHA256: huellaBytes(canonInstantanea), InstantaneaCanonica: canonInstantanea,
		PoliticaRef: "politica:ct:prueba", VersionPolitica: 1,
		HuellaPoliticaSHA256: huella('f'), Corte: ahora,
		Entradas: []ports.EntradaOrigenLlamamientoCT{{
			OrdenActa: 1, OrdenVigente: &orden, ParticipacionRef: participacion.ParticipacionRef,
			CandidatoRef: "candidato:ct:prueba", Situacion: "disponible", Razon: "elegible", Elegible: true,
		}},
	}
	if err := validarLecturaOrigenCT(origen); err != nil {
		t.Fatalf("fixture de origen invalido: %v", err)
	}
	sobre, err := json.Marshal(map[string]any{
		"origen": origen, "reserva": nil,
		"acuse": map[string]any{"auditoria_ref": "auditoria:ct:prueba", "secuencia": 1,
			"huella_sha256": huella('a'), "correlacion_ref": "correlacion:ct:prueba", "registrada_en": ahora},
	})
	if err != nil {
		t.Fatal(err)
	}
	return origen, sobre
}

func sustituirCanonicoOrigenCTPrueba(t *testing.T, sobre []byte, campo string, valor any) []byte {
	t.Helper()
	var exterior map[string]json.RawMessage
	if err := json.Unmarshal(sobre, &exterior); err != nil {
		t.Fatal(err)
	}
	var origen map[string]json.RawMessage
	if err := json.Unmarshal(exterior["origen"], &origen); err != nil {
		t.Fatal(err)
	}
	dato, err := json.Marshal(valor)
	if err != nil {
		t.Fatal(err)
	}
	origen[campo] = dato
	exterior["origen"], err = json.Marshal(origen)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := json.Marshal(exterior)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func TestOrigenCTAuditadoDecodificaHexPostgreSQLYBase64Historico(t *testing.T) {
	esperado, historico := origenAuditadoCanonicoPrueba(t)
	nuevo := sustituirCanonicoOrigenCTPrueba(t, historico, "bolsa_canonica", `\x`+hex.EncodeToString(esperado.BolsaCanonica))
	nuevo = sustituirCanonicoOrigenCTPrueba(t, nuevo, "instantanea_canonica", `\x`+hex.EncodeToString(esperado.InstantaneaCanonica))
	for nombre, raw := range map[string][]byte{"historico_base64": historico, "nuevo_bytea_hex": nuevo} {
		salida, err := decodificarRespuestaOrigenCTAuditado(raw)
		if err != nil || !bytes.Equal(salida.Origen.BolsaCanonica, esperado.BolsaCanonica) ||
			!bytes.Equal(salida.Origen.InstantaneaCanonica, esperado.InstantaneaCanonica) ||
			salida.Origen.HuellaBolsaSHA256 != esperado.HuellaBolsaSHA256 ||
			salida.Origen.HuellaInstantaneaSHA256 != esperado.HuellaInstantaneaSHA256 ||
			validarLecturaOrigenCT(salida.Origen) != nil {
			t.Fatalf("%s no conservó bytes y huellas: %v", nombre, err)
		}
	}
}

func TestOrigenCTAuditadoRechazaCanonicosCorruptos(t *testing.T) {
	esperado, sobre := origenAuditadoCanonicoPrueba(t)
	casos := []struct {
		nombre, campo string
		valor         any
	}{
		{"base64_invalido", "bolsa_canonica", "%%%"},
		{"hex_invalido", "bolsa_canonica", `\x0z`},
		{"hex_truncado", "bolsa_canonica", `\x` + hex.EncodeToString(esperado.BolsaCanonica[:len(esperado.BolsaCanonica)-1])},
		{"nulo", "bolsa_canonica", nil},
		{"vacio", "instantanea_canonica", ""},
		{"numero", "instantanea_canonica", 42},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := decodificarRespuestaOrigenCTAuditado(sustituirCanonicoOrigenCTPrueba(t, sobre, caso.campo, caso.valor))
			if !errors.Is(err, ports.ErrOrigenLlamamientoCTNoDisponible) {
				t.Fatalf("canonico corrupto admitido: %v", err)
			}
		})
	}
	alterado := sustituirCanonicoOrigenCTPrueba(t, sobre, "bolsa_canonica", `\x7b7d`)
	salida, err := decodificarRespuestaOrigenCTAuditado(alterado)
	if err != nil || validarLecturaOrigenCT(salida.Origen) == nil {
		t.Fatalf("canonico JSON distinto atravesó el cotejo de huella: %v", err)
	}
}

func TestOrigenCTAuditadoAceptaMasDeUnMiBEnAmbosFormatos(t *testing.T) {
	canonico := append([]byte(`{"dato":"`), bytes.Repeat([]byte{'a'}, 1<<20)...)
	canonico = append(canonico, '"', '}')
	if !json.Valid(canonico) || len(canonico) <= 1<<20 {
		t.Fatal("el vector debe superar el limite incorrecto anterior")
	}
	base64JSON, err := json.Marshal(base64.StdEncoding.EncodeToString(canonico))
	if err != nil {
		t.Fatal(err)
	}
	hexJSON, err := json.Marshal(`\x` + hex.EncodeToString(canonico))
	if err != nil {
		t.Fatal(err)
	}
	for nombre, codificado := range map[string][]byte{"base64": base64JSON, "bytea_hex": hexJSON} {
		bolsa, err := decodificarCanonicoOrigenCTAuditado(codificado, maximoBolsaCanonicaOrigenCT)
		if err != nil || !bytes.Equal(bolsa, canonico) {
			t.Fatalf("%s rechazó bolsa válida >1MiB: %v", nombre, err)
		}
		instantanea, err := decodificarCanonicoOrigenCTAuditado(codificado, maximoInstantaneaCanonicaOrigenCT)
		if err != nil || !bytes.Equal(instantanea, canonico) || sha256.Sum256(bolsa) != sha256.Sum256(instantanea) {
			t.Fatalf("%s cambió los bytes o la huella: %v", nombre, err)
		}
		if _, err := decodificarCanonicoOrigenCTAuditado(codificado, len(canonico)-1); !errors.Is(err, errCanonicoOrigenCTAuditado) {
			t.Fatalf("%s permitió superar la cota del campo: %v", nombre, err)
		}
	}
}

func TestOrigenCTAuditadoRespetaLimitesB7PorCampoSinAsignarGigantes(t *testing.T) {
	if maximoBolsaCanonicaOrigenCT != 8388608 || maximoInstantaneaCanonicaOrigenCT != 33554432 {
		t.Fatal("los limites deben corresponder a bolsa e instantanea de B7")
	}
	for _, maximo := range []int{maximoBolsaCanonicaOrigenCT, maximoInstantaneaCanonicaOrigenCT} {
		if !longitudCodificadaCanonicoOrigenCTValida(2*maximo, maximo, true) ||
			longitudCodificadaCanonicoOrigenCTValida(2*maximo+2, maximo, true) {
			t.Fatalf("limite hexadecimal incorrecto para %d bytes", maximo)
		}
		longitudBase64 := base64.StdEncoding.EncodedLen(maximo)
		if !longitudCodificadaCanonicoOrigenCTValida(longitudBase64, maximo, false) ||
			longitudCodificadaCanonicoOrigenCTValida(longitudBase64+4, maximo, false) {
			t.Fatalf("limite base64 incorrecto para %d bytes", maximo)
		}
	}
}

func TestReplayOrigenCTConservaActaYExigePrimerRango(t *testing.T) {
	primero := uint64(1)
	entrada := ports.EntradaOrigenLlamamientoCT{
		OrdenActa: 5, OrdenVigente: &primero, Elegible: true,
		ParticipacionRef: "participacion:quinta", CandidatoRef: "candidato:quinta",
	}
	reserva := ports.ReservaOrigenCT{
		OrdenSeleccionado: 5, ParticipacionRef: entrada.ParticipacionRef,
		CandidatoRef: entrada.CandidatoRef,
	}
	if !reservaCoincideEntradaOrigenCTAuditado(reserva, entrada) {
		t.Fatal("replay correcto del acta 5 fue rechazado")
	}
	reserva.OrdenSeleccionado = 1
	if reservaCoincideEntradaOrigenCTAuditado(reserva, entrada) {
		t.Fatal("el rango vigente se confundió con la posición original del acta")
	}
	reserva.OrdenSeleccionado = 5
	segundo := uint64(2)
	entrada.OrdenVigente = &segundo
	if reservaCoincideEntradaOrigenCTAuditado(reserva, entrada) {
		t.Fatal("el replay aceptó un rango vigente que no es el primero")
	}
	entrada.OrdenVigente = &primero
	reserva.ParticipacionRef = "participacion:ajena"
	if reservaCoincideEntradaOrigenCTAuditado(reserva, entrada) {
		t.Fatal("el replay aceptó otra participación con la misma posición")
	}
}
