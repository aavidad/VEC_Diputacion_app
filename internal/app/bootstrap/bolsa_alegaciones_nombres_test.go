package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type repositorioAlegacionesSinSQLPrueba struct {
	ports.RepositorioConstitucion
	llamadas *int
}

func (r repositorioAlegacionesSinSQLPrueba) ListarVigentes(context.Context) ([]ports.ConstitucionVigente, error) {
	*r.llamadas++
	return nil, nil
}
func (r repositorioAlegacionesSinSQLPrueba) Entradas(context.Context, string, uint64) ([]ports.EntradaConstitucion, error) {
	*r.llamadas++
	return nil, nil
}

type recuperadorAlegacionesAusentePrueba struct{}

func (recuperadorAlegacionesAusentePrueba) RecuperarLote(context.Context, string, string) (importacion.LoteValidado, importacionapp.EstadoImportacion, bool, error) {
	return importacion.LoteValidado{}, importacionapp.EstadoImportacion{}, false, nil
}

func TestNombresAlegacionesB7SinSQLAdicionalYFalloReal(t *testing.T) {
	fuente, _, recuperaciones := fuenteNombrePropioPrueba(t, nil)
	sql := new(int)
	fuente.repositorio = repositorioAlegacionesSinSQLPrueba{llamadas: sql}
	lote := loteNombrePropioPrueba()
	metadato := func(ref string, fila int) *inscripcion.MetadatoNombreAlegacionB7 {
		return &inscripcion.MetadatoNombreAlegacionB7{ActaRef: lote.Acta.ActaRef, ParticipacionRef: ref,
			HuellaListadoSHA256: strings.Repeat("ab", 32), CategoriaRef: lote.Acta.CategoriaRef, FilaNumero: fila}
	}
	pagina := []inscripcion.Alegacion{
		{Objeto: inscripcion.ObjetoAlegacion{Tipo: "acto_participacion", ActoRef: lote.Acta.ActaRef, ParticipacionRef: "part_b7_2"}, MetadatoNombreB7: metadato("part_b7_2", 2)},
		{Objeto: inscripcion.ObjetoAlegacion{Tipo: "acto_participacion", ActoRef: lote.Acta.ActaRef, ParticipacionRef: "part_b7_3"}, MetadatoNombreB7: metadato("part_b7_3", 3)},
		{Objeto: inscripcion.ObjetoAlegacion{Tipo: "decision_inscripcion", ActoRef: "decision_sin_b7"}},
	}
	nombres, err := fuente.nombresAlegacionesB7(context.Background(), pagina)
	if err != nil || nombres["part_b7_2"] != "Lucía Moreno Castillo" || nombres["part_b7_3"] != "Antonio Reyes Álvarez" ||
		len(nombres) != 2 || *recuperaciones != 1 || *sql != 0 {
		t.Fatalf("nombres=%d recuperaciones=%d SQL=%d error=%v", len(nombres), *recuperaciones, *sql, err)
	}
	fuente.recuperador = recuperadorAlegacionesAusentePrueba{}
	nombres, err = fuente.nombresAlegacionesB7(context.Background(), pagina)
	if err != nil || len(nombres) != 0 || *sql != 0 {
		t.Fatal("ausencia legítima produjo nombre o consulta SQL")
	}
	fuente.recuperador = recuperadorNombrePropioPrueba{lote: lote, err: errors.New("dato personal ficticio"), llamado: recuperaciones}
	nombres, err = fuente.nombresAlegacionesB7(context.Background(), pagina)
	if !errors.Is(err, inscripcion.ErrNoDisponible) || len(nombres) != 0 || *sql != 0 ||
		strings.Contains(err.Error(), "dato personal") {
		t.Fatal("fallo real tragado o filtró dato personal")
	}
}
