package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func fuenteGobiernoRPTPrueba() FuenteGobiernoCategoriaRPT {
	b := []byte(`{"clase":"ejercicio","categoria":"grupo.a"}`)
	h := sha256.Sum256(b)
	desde := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return FuenteGobiernoCategoriaRPT{
		Bytes: b, SHA256: hex.EncodeToString(h[:]), FuenteRef: "fuente:ejercicio:rpt",
		Clase: "ejercicio", ProcedenciaRef: "procedencia:ensayo", CustodiaRef: "custodia:ensayo",
		OrganizacionRef: "organizacion:ensayo", VigenteDesde: desde, VigenteHasta: desde.AddDate(2, 0, 0),
	}
}

func TestGobiernoRPTFuenteConfiableCopiadaYLimitada(t *testing.T) {
	fuente := fuenteGobiernoRPTPrueba()
	pool := &iniciadorLecturaRPTPrueba{tx: &transaccionLecturaRPTPrueba{}}
	g, err := nuevoGestorGobiernoCategoriaRPTPostgreSQL(pool, descriptorRPTPrueba, fuente)
	if err != nil {
		t.Fatal(err)
	}
	fuente.Bytes[0] = 'X'
	fuente.FuenteRef = "fuente:otra"
	if g.fuenteBytes[0] != '{' || g.fuenteMeta.FuenteRef != "fuente:ejercicio:rpt" {
		t.Fatal("la configuración externa modificó la fuente copiada")
	}
	for nombre, cambiar := range map[string]func(*FuenteGobiernoCategoriaRPT){
		"huella distinta":    func(f *FuenteGobiernoCategoriaRPT) { f.SHA256 = strings.Repeat("0", 64) },
		"clase malformada":   func(f *FuenteGobiernoCategoriaRPT) { f.Clase = "RPT legal" },
		"sin custodia":       func(f *FuenteGobiernoCategoriaRPT) { f.CustodiaRef = "" },
		"vigencia invertida": func(f *FuenteGobiernoCategoriaRPT) { f.VigenteHasta = f.VigenteDesde },
	} {
		t.Run(nombre, func(t *testing.T) {
			mala := fuenteGobiernoRPTPrueba()
			cambiar(&mala)
			if _, err := nuevoGestorGobiernoCategoriaRPTPostgreSQL(pool, descriptorRPTPrueba, mala); !errors.Is(err, ports.ErrGobiernoCategoriaRPTInvalido) {
				t.Fatalf("fuente no confiable aceptada: %v", err)
			}
		})
	}
	futura := fuenteGobiernoRPTPrueba()
	futura.Clase = "rpt_legal"
	if _, err := nuevoGestorGobiernoCategoriaRPTPostgreSQL(pool, descriptorRPTPrueba, futura); err != nil {
		t.Fatalf("el tipo de fuente cerró una admisión SQL futura: %v", err)
	}
}

func TestGobiernoRPTMaterialNoAceptaFuenteDeOrden(t *testing.T) {
	g, err := nuevoGestorGobiernoCategoriaRPTPostgreSQL(&iniciadorLecturaRPTPrueba{tx: &transaccionLecturaRPTPrueba{}}, descriptorRPTPrueba, fuenteGobiernoRPTPrueba())
	if err != nil {
		t.Fatal(err)
	}
	m := ports.MaterialPropuestaGobiernoCategoriaRPT{
		PropuestaRef: "propuesta:rpt:uno", ReciboRef: "recibo:rpt:uno", HuellaSHA256: strings.Repeat("a", 64),
		Contenido: domain.ContenidoGobiernoCategoriaRPT{
			Accion: domain.AccionGobiernoCategoriaRPTPublicar, CatalogoID: descriptorRPTPrueba.CatalogoID,
			ModuloID: descriptorRPTPrueba.ModuloID, FuenteRef: "fuente:ejercicio:rpt",
			PreimagenesControl: map[string]domain.PreimagenControlGobiernoCategoriaRPT{},
		},
	}
	w, err := g.materialPropuesta(m)
	if err != nil {
		t.Fatal(err)
	}
	bruto, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(bruto, &campos); err != nil {
		t.Fatal(err)
	}
	if len(campos) != 8 || campos["fuente_meta"] == nil || campos["fuente_sha256"] == nil || campos["fuente_bytes"] != nil {
		t.Fatalf("material del wrapper alterado: claves=%d", len(campos))
	}
	m.Contenido.FuenteRef = "fuente:http:inyectada"
	if _, err := g.materialPropuesta(m); !errors.Is(err, ports.ErrGobiernoCategoriaRPTInvalido) {
		t.Fatalf("fuente de orden aceptada: %v", err)
	}
}

func TestGobiernoRPTAvanceInvalidoNoConsultaSQL(t *testing.T) {
	pool := &iniciadorLecturaRPTPrueba{tx: &transaccionLecturaRPTPrueba{}}
	g, err := nuevoGestorGobiernoCategoriaRPTPostgreSQL(pool, descriptorRPTPrueba, fuenteGobiernoRPTPrueba())
	if err != nil {
		t.Fatal(err)
	}
	m := ports.MaterialAvanceGobiernoCategoriaRPT{
		PropuestaRef: "propuesta:rpt:uno", CatalogoID: descriptorRPTPrueba.CatalogoID,
		ModuloID: descriptorRPTPrueba.ModuloID, HuellaSHA256: strings.Repeat("a", 64),
		ReciboRef: "recibo:rpt:dos", RevisionEsperada: 2,
	}
	if _, err := g.PrepararAprobacionGobiernoCategoriaRPT(t.Context(), m); !errors.Is(err, ports.ErrGobiernoCategoriaRPTInvalido) || pool.llamadas != 0 {
		t.Fatalf("avance con revisión ajena llegó a SQL: %v, %d", err, pool.llamadas)
	}
}

func TestGobiernoRPTPrepararAvanceSoloMetadatosYHuellaPostgreSQL(t *testing.T) {
	huella := strings.Repeat("a", 64)
	tx := &transaccionLecturaRPTPrueba{huellaMaterial: strings.Repeat("b", 64)}
	tx.respuesta = jsonRPTPrueba(t, preparacionAvanceGobiernoRPTWire{
		PropuestaRef: "propuesta:rpt:uno", CatalogoID: descriptorRPTPrueba.CatalogoID,
		ModuloID: descriptorRPTPrueba.ModuloID, HuellaSHA256: huella, Revision: 1,
		CategoriaID: "grupo.a", OrganizacionRef: "organizacion:ensayo",
		FuenteSHA256: fuenteGobiernoRPTPrueba().SHA256,
	})
	pool := &iniciadorLecturaRPTPrueba{tx: tx}
	g, err := nuevoGestorGobiernoCategoriaRPTPostgreSQL(pool, descriptorRPTPrueba, fuenteGobiernoRPTPrueba())
	if err != nil {
		t.Fatal(err)
	}
	m := ports.MaterialAvanceGobiernoCategoriaRPT{
		PropuestaRef: "propuesta:rpt:uno", CatalogoID: descriptorRPTPrueba.CatalogoID,
		ModuloID: descriptorRPTPrueba.ModuloID, HuellaSHA256: huella,
		ReciboRef: "recibo:rpt:dos", RevisionEsperada: 1,
	}
	w, err := g.materialAvance(m, "aprobar")
	if err != nil || w.FuenteSHA256 != fuenteGobiernoRPTPrueba().SHA256 || w.OrganizacionRef != "organizacion:ensayo" {
		t.Fatalf("avance no vincula fuente privada: %#v, %v", w, err)
	}
	p, err := g.PrepararAprobacionGobiernoCategoriaRPT(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	if pool.opciones.AccessMode != "read only" || tx.consulta != prepararAvanceGobiernoRPTSQL ||
		!tx.configurada || !tx.confirmada || tx.consultasCanon != 1 || tx.fachadas != 1 ||
		p.Recurso.Atributos["material_sha256"] != strings.Repeat("b", 64) ||
		p.HuellaPropuesta != huella || p.Accion != ports.AccionAprobarGobiernoCategoriaRPT ||
		!reflect.DeepEqual(tx.argumentos, []any{m.PropuestaRef, m.HuellaSHA256, m.RevisionEsperada, "aprobar"}) {
		t.Fatalf("preparación sin contrato cerrado: %#v, %#v", p, tx)
	}
}
