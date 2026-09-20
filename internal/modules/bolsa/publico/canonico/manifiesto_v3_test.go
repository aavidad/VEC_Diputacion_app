package canonico

import (
	"testing"
	"time"
)

func manifiestoV3Prueba(t *testing.T) ManifiestoPublicoV3 {
	t.Helper()
	base := manifiestoPrueba(t)
	instante := base.Fuente.ActualizadaEn
	return ManifiestoPublicoV3{
		Esquema: EsquemaManifiestoPublicoV3, Fuente: base.Fuente, Catalogos: base.Catalogos,
		Categorias: base.Categorias, Convocatorias: base.Convocatorias,
		BolsasV1: BolsasManifiestoV1{GeneradoEn: instante, Bolsas: []BolsaManifiestoV1{{
			BolsaRef: "bolsa:administrativo:2026-09-18", Categoria: "Administrativo", CategoriaClave: "administrativo",
			Grupos: []string{"C1", "C2"}, TipoLista: "definitiva", VigenteDesde: instante,
			Total: 2, Posiciones: []PosicionBolsaManifiestoV1{
				{Orden: 1, DocumentoEnmascarado: "***0001**", EstadoClave: "disponible"},
				{Orden: 2, DocumentoEnmascarado: "***0002**", EstadoClave: "ocupado"},
			},
		}}},
	}
}

func TestManifiestoPublicoV3OrdenaBolsasYGruposYDetectaAlteracion(t *testing.T) {
	manifiesto := manifiestoV3Prueba(t)
	manifiesto.BolsasV1.Bolsas = append(manifiesto.BolsasV1.Bolsas, BolsaManifiestoV1{
		BolsaRef: "bolsa:auxiliar:2026-09-18", Categoria: "Auxiliar", CategoriaClave: "auxiliar",
		Grupos: []string{"C2"}, TipoLista: "definitiva", VigenteDesde: manifiesto.Fuente.ActualizadaEn,
	})
	manifiesto.BolsasV1.Bolsas[0], manifiesto.BolsasV1.Bolsas[1] = manifiesto.BolsasV1.Bolsas[1], manifiesto.BolsasV1.Bolsas[0]
	manifiesto.BolsasV1.Bolsas[1].Grupos[0], manifiesto.BolsasV1.Bolsas[1].Grupos[1] = "C2", "C1"
	huella, err := manifiesto.HuellaSHA256()
	if err != nil || huella == "" {
		t.Fatalf("huella V3: %q %v", huella, err)
	}
	canonico, err := manifiesto.canonico()
	if err != nil || canonico.BolsasV1.Bolsas[0].BolsaRef != "bolsa:administrativo:2026-09-18" || canonico.BolsasV1.Bolsas[0].Grupos[0] != "C1" {
		t.Fatalf("orden canonico de grupos: %#v; %v", canonico.BolsasV1.Bolsas, err)
	}
	alterado := manifiesto
	alterado.BolsasV1.Bolsas = append([]BolsaManifiestoV1(nil), manifiesto.BolsasV1.Bolsas...)
	alterado.BolsasV1.Bolsas[1].Posiciones = append([]PosicionBolsaManifiestoV1(nil), manifiesto.BolsasV1.Bolsas[1].Posiciones...)
	alterado.BolsasV1.Bolsas[1].Posiciones[1].EstadoClave = "excluido"
	huellaAlterada, err := alterado.HuellaSHA256()
	if err != nil || huellaAlterada == huella {
		t.Fatalf("alteracion B10 no cambia ancla: %q %v", huellaAlterada, err)
	}
	if !canonico.BolsasV1.GeneradoEn.Equal(time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)) {
		t.Fatal("instante B10 no canonico")
	}
}
