package interna

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type renderOriginalContadoR5 struct{ llamadas int }

func (r *renderOriginalContadoR5) RenderizarBorrador(context.Context, ctports.TipoBorradorRRHH, ctports.DetalleExpedienteRRHH) ([]byte, error) {
	r.llamadas++
	return []byte("%PDF-prueba"), nil
}

func TestOriginalR5NoRenderizaSinTresEtiquetas(t *testing.T) {
	delegado := &renderOriginalContadoR5{}
	r := renderizadorEtiquetadoOriginalR5{siguiente: delegado, etiquetar: func(ref string) string {
		if ref == "unidad:sin_publicar" {
			return ""
		}
		return "Nombre publicado"
	}}
	detalle := ctports.DetalleExpedienteRRHH{Resumen: ctports.ResumenExpedienteRRHH{CentroRef: "centro:rpt:101", CategoriaRef: "categoria:rpt:c2"},
		Asignacion: &ctports.AsignacionOperativaRRHH{UnidadRef: "unidad:sin_publicar"}}
	if _, err := r.RenderizarBorrador(context.Background(), ctports.BorradorResolucion, detalle); err == nil || delegado.llamadas != 0 {
		t.Fatalf("se renderizó sin nombre de unidad: err=%v llamadas=%d", err, delegado.llamadas)
	}
	detalle.Asignacion.UnidadRef = "unidad:publicada"
	if contenido, err := r.RenderizarBorrador(context.Background(), ctports.BorradorResolucion, detalle); err != nil || string(contenido) != "%PDF-prueba" || delegado.llamadas != 1 {
		t.Fatalf("la publicación completa no permitió renderizar: err=%v llamadas=%d", err, delegado.llamadas)
	}
}

func TestRecursoGobernadoConservaFalloCierreOriginalR5(t *testing.T) {
	causa := errors.New("cierre de almacén")
	llamadas := 0
	r := &recursoGobiernoInterno{originalR5: func() error { llamadas++; return causa }}
	primero := r.cerrarSinPropiedad()
	segundo := r.cerrarSinPropiedad()
	if llamadas != 1 || !errors.Is(primero, causa) || !errors.Is(primero, ErrDependenciasProductivasNoDisponibles) ||
		!errors.Is(segundo, causa) {
		t.Fatalf("cierre sin causa o repetido: llamadas=%d, primero=%v, segundo=%v", llamadas, primero, segundo)
	}
}

func TestOriginalR5SoloSePublicaConMaterialYAutoridades(t *testing.T) {
	directorio := t.TempDir()
	m, seleccionado, err := montarOriginalR5Gobernado(context.Background(), directorio,
		internactproveedores.Material{}, nil, nil, relojGobiernoInterno{})
	if err != nil || seleccionado || m.manejador != nil {
		t.Fatalf("sin inventario se publicó original: seleccionado=%v, err=%v", seleccionado, err)
	}
	if err := os.WriteFile(filepath.Join(directorio, "ct_firma_r5.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	m, seleccionado, err = montarOriginalR5Gobernado(context.Background(), directorio,
		internactproveedores.Material{}, nil, nil, relojGobiernoInterno{})
	if !seleccionado || err == nil || m.manejador != nil {
		t.Fatalf("inventario incompleto publicó original: seleccionado=%v, err=%v", seleccionado, err)
	}
}
