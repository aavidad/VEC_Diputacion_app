package bootstrap

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestProvisionMotivosSeleccionCerradaYPlanLigado(t *testing.T) {
	f := fuenteProvisionCandidatoPrueba()
	f.Preimagen.SecuenciaMotivos = 36
	f.CatalogosMotivos = []string{motivoHistorialMiBolsaDesarrollo().CatalogoID}
	p, err := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), f, "motivos", time.Now())
	if err != nil || len(p.motivos) != 1 || p.motivos[0] != motivoHistorialMiBolsaDesarrollo() {
		t.Fatal("no seleccionó exclusivamente el motivo existente de historial", err)
	}
	r := p.Resumen()
	r.CatalogosMotivos[0] = "catalogo_mutado"
	if p.Resumen().CatalogosMotivos[0] != f.CatalogosMotivos[0] {
		t.Fatal("el resumen comparte memoria con el plan")
	}
	b, err := json.Marshal(p.Resumen())
	if err != nil {
		t.Fatal(err)
	}
	var esquema map[string]any
	if json.Unmarshal(b, &esquema) != nil || len(esquema) != 5 || esquema["fase"] != "motivos" || esquema["catalogos_motivos"] == nil {
		t.Fatal("el plan no muestra su selección y las dos huellas")
	}
	f.CatalogosMotivos = nil
	completo, err := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), f, "motivos", time.Now())
	if err != nil || len(completo.motivos) != 3 || completo.resumen.HuellaSHA256 == p.resumen.HuellaSHA256 {
		t.Fatal("la selección no quedó ligada al plan o perdió el comportamiento previo")
	}
	for _, seleccion := range [][]string{{}, {"desconocido"}, {motivoHistorialMiBolsaDesarrollo().CatalogoID, motivoHistorialMiBolsaDesarrollo().CatalogoID}} {
		f.CatalogosMotivos = seleccion
		if _, err := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), f, "motivos", time.Now()); err == nil {
			t.Fatal("aceptó catálogo vacío, desconocido o duplicado")
		}
	}
	f.CatalogosMotivos = []string{motivoHistorialMiBolsaDesarrollo().CatalogoID}
	for _, fase := range []string{"autorizacion", "contexto", "identidad"} {
		if _, err := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), f, fase, time.Now()); err == nil {
			t.Fatal("la selección habilitó otro gobierno")
		}
	}
}

type txMotivosProvisionPrueba struct {
	pgx.Tx
	args      [][]any
	publicada bool
}

func (t *txMotivosProvisionPrueba) QueryRow(_ context.Context, consulta string, args ...any) pgx.Row {
	t.args = append(t.args, append([]any(nil), args...))
	return filaProvisionPrueba{valores: []any{t.publicada}}
}

func TestProvisionMotivoAusenteYReplayUsanMismaSecuenciaYContenido(t *testing.T) {
	f := fuenteProvisionCandidatoPrueba()
	f.CatalogosMotivos = []string{motivoHistorialMiBolsaDesarrollo().CatalogoID}
	f.Preimagen.SecuenciaMotivos = 36
	p, err := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), f, "motivos", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	tx := &txMotivosProvisionPrueba{publicada: true}
	for i := 0; i < 2; i++ {
		if err := publicarMotivosCandidatoExterno(context.Background(), tx, p); err != nil {
			t.Fatal(err)
		}
	}
	if len(tx.args) != 2 || !reflect.DeepEqual(tx.args[0], tx.args[1]) || tx.args[0][1] != int64(37) || tx.args[0][3] != motivoHistorialMiBolsaDesarrollo().CatalogoID {
		t.Fatal("repitió otro catálogo, otra secuencia o material diferente")
	}
	tx.publicada = false
	if err := publicarMotivosCandidatoExterno(context.Background(), tx, p); err != ErrProvisionCandidatoExterno {
		t.Fatal("ignoró un rechazo de CAS o de contenido del publicador SQL")
	}
}
