package bootstrap

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestConjuntoSieteConservaSeisYSegregaRolesNuevos(t *testing.T) {
	seis, _ := AudienciasConjuntoCapacidadesAdmin(6)
	siete, ok := AudienciasConjuntoCapacidadesAdmin(7)
	if !ok || len(siete) != len(seis)+2 {
		t.Fatal("conjunto 7 incompleto")
	}
	for i := range seis {
		if siete[i] != seis[i] {
			t.Fatalf("cambia la entrada histórica %d", i)
		}
	}
	esperadas := []AudienciaCapacidadAdmin{
		{pg.AudienciaGobiernoRolProponer, SegmentoGobiernoRolPropuesta, EmisorGobiernoRolesAdmin},
		{pg.AudienciaGobiernoRolAprobar, SegmentoGobiernoRolCierre, EmisorGobiernoRolesAdmin},
	}
	for i, e := range esperadas {
		if siete[len(seis)+i] != e {
			t.Fatalf("entrada nueva %d distinta de AUT60", i)
		}
		for _, anterior := range seis {
			if e.Audiencia == anterior.Audiencia || e.Segmento == anterior.Segmento || e.EmisorID == anterior.EmisorID {
				t.Fatal("capacidad nueva comparte identidad con una anterior")
			}
		}
	}
}

func TestOverlayRolesNuevosExigeSuConjuntoYAcuse(t *testing.T) {
	for _, conjunto := range []uint64{6, 7} {
		s := prepararSalidaVersionBolsaPrueba(t, conjunto)
		d := destinoVersionBolsaPrueba(s.dir)
		motivo := d.Motivos[pg.AudienciaVersionarRolBolsaProponer]
		d.Motivos = map[string]domain.ReferenciaEntradaCatalogo{
			pg.AudienciaGobiernoRolProponer: motivo, pg.AudienciaGobiernoRolAprobar: motivo,
		}
		err := EscribirOverlayGobiernoRolesAdmin(s.raiz, d, "acuse-aplicar.json", s.reloj)
		if conjunto == 6 {
			if err == nil {
				t.Fatal("exporta claves de roles nuevos desde conjunto 6")
			}
			if _, err := s.raiz.Stat("gobierno-roles.json"); !os.IsNotExist(err) {
				t.Fatal("escribe configuración tras denegar")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		b, err := s.raiz.ReadFile("gobierno-roles.json")
		var o overlayVersionBolsa
		if err != nil || json.Unmarshal(b, &o) != nil || len(o.Confianza.EntradasCapacidad) != 2 || bytes.Contains(b, []byte("secreto_hmac")) {
			t.Fatal("configuración de roles inválida o con secreto en línea")
		}
		for _, e := range o.Confianza.EntradasCapacidad {
			secreto, err := os.ReadFile(e.MaterialArchivo)
			info, statErr := os.Stat(e.MaterialArchivo)
			if err != nil || statErr != nil || info.Mode().Perm() != 0600 ||
				e.EmisorID != EmisorGobiernoRolesAdmin || !bytes.Equal(secreto, s.claves[e.Audiencia]) ||
				filepath.Dir(e.MaterialArchivo) != s.dir || len(secreto) != 32 {
				t.Fatal("material nominal no coincide con el confirmado")
			}
		}
		if EscribirOverlayGobiernoRolesAdmin(s.raiz, d, "acuse-aplicar.json", s.reloj) == nil {
			t.Fatal("sobrescribe una exportación anterior")
		}
	}
}
