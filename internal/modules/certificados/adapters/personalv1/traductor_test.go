package personalv1

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/certificados/domain"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

const muestra = "testdata/servicios-personal-v1.ensayo.json"

func TestTraducirConvierteElPeriodoSemiabiertoYConservaLaProcedencia(t *testing.T) {
	f, err := FicheroMuestra{Ruta: muestra}.Obtener(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if f.Esquema != domain.EsquemaFuentePersonalV1Ensayo || f.Cobertura != "parcial" || len(f.Servicios) != 4 || f.Corte.ConocidoEn != "2026-10-01T08:00:00Z" {
		t.Fatalf("%+v", f)
	}
	// [2022-01-10, 2024-01-10) termina el día anterior; Hasta vacía sigue abierto.
	if s := f.Servicios[2]; s.Inicio != "2022-01-10" || s.Fin != "2024-01-09" || s.Dias != nil || s.Certeza != "acreditado" ||
		s.ActoRef != "ensayo:resolucion-2024-118" || s.ClaseVersion != 2 || !s.SustentaCertificacion() {
		t.Fatalf("%+v", s)
	}
	if s := f.Servicios[3]; s.Fin != "" || !s.SustentaCertificacion() {
		t.Fatalf("periodo abierto: %+v", s)
	}
	if f.Servicios[0].SustentaCertificacion() || f.Servicios[1].SustentaCertificacion() {
		t.Fatal("un servicio declarado o comprobado no sustenta un certificado")
	}
}

func TestTraducirRechazaPeriodosImposibles(t *testing.T) {
	r := personalports.ResultadoServiciosParaCertificadosV1{Cobertura: "completa",
		Corte:     personaldomain.CorteEmpleadoB2{VigenteEn: "2026-10-01", ConocidoEn: time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)},
		Servicios: []personalports.ServicioParaCertificadosV1{{Periodo: personalports.PeriodoPersonalNominalV1{Desde: "2024-01-10", Hasta: "2024-01-10"}}}}
	if _, err := Traducir(r, "x", "ensayo:x", true); err != domain.ErrEntrada {
		t.Fatalf("periodo vacío aceptado: %v", err)
	}
	r.Corte.ConocidoEn = time.Time{}
	r.Servicios = nil
	if _, err := Traducir(r, "x", "ensayo:x", true); err != domain.ErrEntrada {
		t.Fatalf("corte sin instante aceptado: %v", err)
	}
}

func TestFicheroMuestraRechazaMuestrasNoValidas(t *testing.T) {
	raw, err := os.ReadFile(muestra)
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string]func(string) string{
		"no_sintetica":      func(s string) string { return strings.Replace(s, `"sintetica": true`, `"sintetica": false`, 1) },
		"estado_ajeno":      func(s string) string { return strings.Replace(s, `"estado": "declarado"`, `"estado": "propuesto"`, 1) },
		"certeza_ajena":     func(s string) string { return strings.Replace(s, `"certeza": "pendiente"`, `"certeza": "probable"`, 1) },
		"cobertura_ajena":   func(s string) string { return strings.Replace(s, `"cobertura": "parcial"`, `"cobertura": "casi"`, 1) },
		"con_dias":          func(s string) string { return strings.Replace(s, `"version": 7,`, `"version": 7, "dias": 30,`, 1) },
		"hasta_antes":       func(s string) string { return strings.Replace(s, `"hasta": "2019-09-01"`, `"hasta": "2019-02-01"`, 1) },
		"inicio_tras_corte": func(s string) string { return strings.Replace(s, `"desde": "2024-01-10"`, `"desde": "2026-11-01"`, 1) },
		"sin_acto": func(s string) string {
			return strings.Replace(s, `"acto_ref": "ensayo:declaracion-2019"`, `"acto_ref": ""`, 1)
		},
		"esquema_ensayo":  func(s string) string { return strings.Replace(s, EsquemaMuestra, domain.EsquemaFuenteEnsayo, 1) },
		"clave_duplicada": func(s string) string { return strings.Replace(s, `"version": 7,`, `"version": 7, "version": 8,`, 1) },
	}
	dir := t.TempDir()
	for nombre, cambiar := range casos {
		alterada := cambiar(string(raw))
		if alterada == string(raw) {
			t.Fatalf("%s: el caso no cambia la muestra", nombre)
		}
		ruta := filepath.Join(dir, nombre+".json")
		if err := os.WriteFile(ruta, []byte(alterada), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := (FicheroMuestra{Ruta: ruta}).Obtener(context.Background()); err == nil {
			t.Errorf("%s: aceptada", nombre)
		}
	}
	var m Muestra
	if json.Unmarshal(raw, &m) != nil || m.Nombre == "" {
		t.Fatal("muestra ilegible")
	}
}

func TestServicioConFinPrevistoTrasElCorteQuedaEnCurso(t *testing.T) {
	r := personalports.ResultadoServiciosParaCertificadosV1{Cobertura: "completa",
		Corte: personaldomain.CorteEmpleadoB2{VigenteEn: "2026-10-01", ConocidoEn: time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)},
		Servicios: []personalports.ServicioParaCertificadosV1{{ServicioRef: "s:1", ClaseRef: "c", ClaseVersion: 1, Estado: "reconocido",
			Periodo:     personalports.PeriodoPersonalNominalV1{Desde: "2026-03-01", Hasta: "2027-01-01"},
			Procedencia: personalports.ProcedenciaPersonalNominalV1{ActoRef: "a:1", Certeza: "acreditado"}}}}
	f, err := Traducir(r, "Elena Martín Robles", "ensayo:x", true)
	if err != nil || f.ValidarEnsayo() != nil {
		t.Fatal(err)
	}
	if s := f.Servicios[0]; !s.EnCursoAlCorte || s.Fin != "2026-10-01" {
		t.Fatalf("%+v", s)
	}
	f.Servicios[0].Fin = "2026-09-30"
	if f.ValidarEnsayo() == nil {
		t.Fatal("en curso con un fin distinto de la fecha de referencia")
	}
}
