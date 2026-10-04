package adminperfiles

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	is "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

type fuenteIDsPruebaADMIN struct{ ids IdentificadoresFuenteADMIN }

func (f fuenteIDsPruebaADMIN) ResolverIdentificadoresADMIN(context.Context, ReferenciaFuenteIdentificadoresADMIN) (IdentificadoresFuenteADMIN, error) {
	return f.ids, nil
}

type seudIDsPruebaADMIN struct {
	resultado is.SeudonimosAlta
	recibidos is.IdentificadoresAlta
}

func (s *seudIDsPruebaADMIN) SeudonimizarAlta(_ context.Context, ids is.IdentificadoresAlta) (is.SeudonimosAlta, error) {
	s.recibidos = ids
	return s.resultado, nil
}

func TestIdentificadoresADMINExigeTresHMACYCoordenadasOriginales(t *testing.T) {
	r := ReferenciaFuenteIdentificadoresADMIN{PersonaRef: "per_" + strings.Repeat("a", 32), CuentaRef: "cta_" + strings.Repeat("b", 32), CuentaOrdinariaRef: "cta_" + strings.Repeat("c", 32),
		CertificadoSHA256: strings.Repeat("d", 64), CASHA256: strings.Repeat("e", 64), EspacioIdentidad: "https://sintetico.example.invalid", EsquemaHMAC: is.EsquemaHMACSHA256V1,
		DominioHMACRef: "idh_" + strings.Repeat("f", 32), ClaveHMACID: "clave-sintetica-v1", ClaveHMACVersion: 1, FuenteRef: "fuente:ids:sintetica", FuenteSHA256: strings.Repeat("1", 64),
		SujetoHMAC: [32]byte{1}, CuentaHMAC: [32]byte{2}, CuentaOrdinariaHMAC: [32]byte{3}}
	ids := IdentificadoresFuenteADMIN{SujetoID: "sujeto-sintetico-original", CuentaID: "admin-sintetica-original", CuentaOrdinariaID: "ordinaria-sintetica-original",
		EspacioIdentidad: r.EspacioIdentidad, DominioHMACRef: r.DominioHMACRef, ClaveHMACID: r.ClaveHMACID, ClaveHMACVersion: 1, FuenteRef: r.FuenteRef, FuenteSHA256: r.FuenteSHA256}
	base := is.SeudonimosAlta{Esquema: r.EsquemaHMAC, EspacioIdentidad: r.EspacioIdentidad, DominioRef: r.DominioHMACRef, ClaveID: r.ClaveHMACID, ClaveVersion: 1,
		SujetoIDHMAC: r.SujetoHMAC, CuentaIDHMAC: r.CuentaHMAC, CuentaOrdinariaIDHMAC: r.CuentaOrdinariaHMAC}
	for _, caso := range []struct {
		nombre    string
		cambiar   func(*IdentificadoresFuenteADMIN, *is.SeudonimosAlta)
		permitido bool
	}{
		{"fuente original", func(*IdentificadoresFuenteADMIN, *is.SeudonimosAlta) {}, true},
		{"sujeto distinto", func(_ *IdentificadoresFuenteADMIN, s *is.SeudonimosAlta) { s.SujetoIDHMAC[0]++ }, false},
		{"cuenta distinta", func(_ *IdentificadoresFuenteADMIN, s *is.SeudonimosAlta) { s.CuentaIDHMAC[0]++ }, false},
		{"ordinaria distinta", func(_ *IdentificadoresFuenteADMIN, s *is.SeudonimosAlta) { s.CuentaOrdinariaIDHMAC[0]++ }, false},
		{"otra generación", func(_ *IdentificadoresFuenteADMIN, s *is.SeudonimosAlta) { s.ClaveVersion++ }, false},
		{"otra procedencia", func(i *IdentificadoresFuenteADMIN, _ *is.SeudonimosAlta) { i.FuenteSHA256 = strings.Repeat("2", 64) }, false},
		{"sin preimagen", func(i *IdentificadoresFuenteADMIN, _ *is.SeudonimosAlta) { i.SujetoID = "" }, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			i, s := ids, base
			caso.cambiar(&i, &s)
			seud := &seudIDsPruebaADMIN{resultado: s}
			v, err := cotejarIdentificadoresFuenteADMIN(context.Background(), fuenteIDsPruebaADMIN{i}, seud, r)
			if (err == nil) != caso.permitido {
				t.Fatal("cotejo no coincide con autoridad esperada")
			}
			if caso.permitido && (v != ids || seud.recibidos.SujetoID != ids.SujetoID || seud.recibidos.CuentaID != ids.CuentaID || seud.recibidos.CuentaOrdinariaID != ids.CuentaOrdinariaID) {
				t.Fatal("se alteraron los identificadores originales")
			}
		})
	}
}

func TestIS16RechazaAcuseAjenoIncompletoODuplicado(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	corr := "correlacion_" + strings.Repeat("a", 32)
	a := acuseIS16{Referencia: "aud_v3_ap2_" + strings.Repeat("b", 32), Secuencia: 1, Huella: strings.Repeat("c", 64), Correlacion: corr, RegistradaEn: ahora}
	bruto := []byte(`{"estado":"denegado","datos":null}`)
	for _, caso := range []struct {
		nombre    string
		cambiar   func(*acuseIS16)
		permitido bool
	}{
		{"denegación confirmada", func(*acuseIS16) {}, true},
		{"otra correlación", func(a *acuseIS16) { a.Correlacion = "correlacion_" + strings.Repeat("d", 32) }, false},
		{"sin secuencia", func(a *acuseIS16) { a.Secuencia = 0 }, false},
		{"sin huella", func(a *acuseIS16) { a.Huella = "" }, false},
		{"fecha futura", func(a *acuseIS16) { a.RegistradaEn = ahora.Add(time.Second) }, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			v := a
			caso.cambiar(&v)
			b, _ := json.Marshal(v)
			_, decision, err := leerResultadoIS16(bruto, b, corr, ahora)
			if (err == nil) != caso.permitido || (caso.permitido && decision == nil) {
				t.Fatal("se aceptó un acuse no acreditado")
			}
		})
	}
	b, _ := json.Marshal(a)
	for _, v := range [][]byte{[]byte(`{"estado":"permitido","estado":"denegado","datos":null}`), []byte(`{"estado":"denegado","datos":null,"extra":1}`)} {
		if _, _, err := leerResultadoIS16(v, b, corr, ahora); err == nil {
			t.Fatal("se aceptó resultado ambiguo")
		}
	}
}

func TestADMINNoSerializaIdentificadoresNiMaterialSQL(t *testing.T) {
	c := CuentaADMIN{SujetoID: "ORIGINAL-PRIVADO", CuentaID: "cuenta-privada", materialCuentaSQL: "MATERIAL-PRIVADO"}
	ids := IdentificadoresFuenteADMIN{SujetoID: c.SujetoID}
	for _, v := range []any{c, ids} {
		b, _ := json.Marshal(v)
		if strings.Contains(string(b), "PRIVADO") || strings.Contains(fmt.Sprintf("%+v %#v", v, v), "ORIGINAL-PRIVADO") {
			t.Fatal("identificadores expuestos")
		}
	}
}
