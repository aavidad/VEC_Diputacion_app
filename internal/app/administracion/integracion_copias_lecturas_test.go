package administracion

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	regadapter "vec-diputacion-granada/internal/modules/administracion/adapters/registrocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	ej "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

type destinoLecturasSintetico struct {
	conjunto ej.Conjunto
	err      error
	llamadas int
}

func (d *destinoLecturasSintetico) Recuperar(context.Context, string) (ej.Conjunto, error) {
	d.llamadas++
	return d.conjunto, d.err
}
func (d *destinoLecturasSintetico) Publicar(context.Context, ej.Captura) (ej.Conjunto, error) {
	return ej.Conjunto{}, p.ErrNoDisponible
}
func (d *destinoLecturasSintetico) CerrarVerificacion(context.Context, ej.Conjunto, copias.Verificacion) (ej.Conjunto, error) {
	return ej.Conjunto{}, p.ErrNoDisponible
}
func TestLectorDiarioRealMinimizaYSoloRecuperaConjuntoAutenticado(t *testing.T) {
	b, err := os.ReadFile("../../../cmd/vec-copias-comprobar/testdata/compatible.json")
	if err != nil {
		t.Fatal(err)
	}
	var ejemplo struct {
		Manifiesto copias.Manifiesto `json:"manifiesto"`
	}
	if json.Unmarshal(b, &ejemplo) != nil {
		t.Fatal("fixture")
	}
	m := ejemplo.Manifiesto
	base := t.TempDir()
	dir := filepath.Join(base, "diario")
	root := filepath.Join(base, "estado")
	if os.Mkdir(dir, 0700) != nil || os.Mkdir(root, 0700) != nil {
		t.Fatal("fixture dirs")
	}
	diario, err := regadapter.Abrir(regadapter.Config{Directorio: dir, RaicesRestauradas: []string{root}, LimiteListado: 100})
	if err != nil {
		t.Fatal(err)
	}
	ses := sesionPrueba(t)
	solicitud := operacionescopias.Solicitud{Operacion: m.OperacionRef, Clave: "clave:ejemplo", SHA256: strings.Repeat("a", 64), Conjunto: m.ConjuntoRef, Destino: "destino:ejemplo", Politica: m.PoliticaRef}
	if _, err = diario.Reservar(context.Background(), declaracionCopias(ses), solicitud); err != nil {
		t.Fatal(err)
	}
	destino := &destinoLecturasSintetico{conjunto: ej.Conjunto{Ref: m.ConjuntoRef, IndiceAutenticadoRef: "indice:fixture", Manifiesto: m, ManifiestoSHA256: huellaManifiestoADMIN(m)}}
	autoridad := &autoridadCopiasPrueba{recurso: m.ConjuntoRef}
	auditor := &auditorCopiasPrueba{}
	lector := &LecturasCopias{Autoridad: autoridad, Auditor: auditor, Diario: diario, Destino: destino}
	pagina, err := lector.Listar(context.Background(), ses, "", 25)
	if err != nil || len(pagina.Copias) != 1 || pagina.Copias[0].CopiaRef != m.ConjuntoRef || pagina.Copias[0].Compatibilidad.Estado != "no_comprobable" {
		t.Fatalf("lectura: %+v %v", pagina, err)
	}
	// La declaración inicial CS07 no decide la verificación del manifiesto.
	if pagina.Copias[0].Estado != m.Verificacion.Estado {
		t.Fatal("estado inferido del diario")
	}
	autoridad.err = p.ErrDenegado
	antes := destino.llamadas
	if _, err = lector.Detalle(context.Background(), ses, m.ConjuntoRef); !errors.Is(err, p.ErrDenegado) || destino.llamadas != antes {
		t.Fatal("sin permiso recupera contenido")
	}
	autoridad.err = nil
	destino.err = errors.New("fallo_sintetico")
	if _, err = lector.Listar(context.Background(), ses, "", 25); !errors.Is(err, p.ErrNoDisponible) {
		t.Fatal("fallo autenticacion no bloquea")
	}
	destino.err = nil
	destino.conjunto.Manifiesto.OperacionRef = "operacion:ajena"
	destino.conjunto.ManifiestoSHA256 = huellaManifiestoADMIN(destino.conjunto.Manifiesto)
	if _, err = lector.Listar(context.Background(), ses, "", 25); !errors.Is(err, p.ErrNoDisponible) {
		t.Fatal("conjunto de otra operacion aceptado")
	}
}
