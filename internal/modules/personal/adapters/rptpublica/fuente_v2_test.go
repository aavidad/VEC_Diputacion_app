package rptpublica

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

func TestFuenteV2LigaElSnapshotYRechazaCambioDeBytes(t *testing.T) {
	catalogo := domain.CatalogoRPTPublicaV2{
		Esquema: domain.EsquemaCandidatoRPTPublicaV2, Estado: domain.EstadoCandidatoRPTPublicaV2,
		Fuente:                    domain.FuenteRPTPublica{Documento: "RPT publicada", Importacion: "rpt-2026", GeneradoEn: "2026-09-17", Aviso: "Sin ocupantes."},
		Resumen:                   domain.ResumenRPTPublica{Puestos: 1, Dotacion: 1, Categorias: 1, Centros: 1},
		Categorias:                []domain.CategoriaRPTPublicaV2{{Clave: "auxiliar", Denominacion: "AUXILIAR", Origen: "categoria", Grupos: []string{"C2"}, Escalas: []string{}, Puestos: 1, Dotacion: 1}},
		Puestos:                   []domain.PuestoRPTPublicoV2{{PuestoRPTPublico: domain.PuestoRPTPublico{Codigo: "430-101-001", Denominacion: "AUXILIAR", CentroCodigo: "101", Centro: "CENTRO", Delegacion: "AREA", Grupos: []string{"C2"}, Escala: "", CategoriaClave: "auxiliar", Dotacion: 1, Tipo: "N", Provision: "C"}, CategoriasClaves: []string{"auxiliar"}, CategoriasPendientes: []domain.CategoriaPendienteRPTPublicaV2{}}},
		CategoriasPendientesGrupo: []string{},
	}
	bruto, err := json.Marshal(catalogo)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "rpt.json")
	if err := os.WriteFile(ruta, bruto, 0600); err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(bruto)
	f, err := NuevaFuenteV2(ruta, "rpt-publicada:2026-05-07", "2026-05-07", hex.EncodeToString(suma[:]))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := f.ObtenerRPTPublicaV2(t.Context())
	if err != nil || snapshot.Validar() != nil || snapshot.HuellaSHA256 != hex.EncodeToString(suma[:]) || snapshot.Catalogo.Puestos[0].CategoriaClave != "auxiliar" {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	if err := os.WriteFile(ruta, append(bruto, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := f.ObtenerRPTPublicaV2(context.Background()); err == nil {
		t.Fatal("la fuente admitió un cambio posterior a la huella fijada")
	}
}

// Permite comprobar el candidato preparado localmente sin incluirlo en Git.
func TestFuenteV2CandidatoLocal(t *testing.T) {
	ruta := os.Getenv("RPT_CANDIDATO_LOCAL")
	if ruta == "" {
		t.Skip("RPT_CANDIDATO_LOCAL no configurado")
	}
	bruto, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(bruto)
	f, err := NuevaFuenteV2(ruta, "rpt-publicada:2026-05-07", "2026-05-07", hex.EncodeToString(suma[:]))
	if err != nil {
		t.Fatal(err)
	}
	s, err := f.ObtenerRPTPublicaV2(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if s.Catalogo.Resumen.Puestos != 842 || s.Catalogo.Resumen.Dotacion != 1714 || len(s.Catalogo.CategoriasPendientesGrupo) != 11 || s.Catalogo.Estado != domain.EstadoCandidatoRPTPublicaV2 {
		t.Fatalf("el candidato dejó de tener su procedencia e incertidumbres: %+v", s.Catalogo.Resumen)
	}
}
