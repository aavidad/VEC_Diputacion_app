package bootstrap

import (
	"context"
	"testing"

	bolsadominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type lectorContactosContadoPrueba struct {
	llamadas *int
	total    int
}

// Imita al repositorio real: devuelve siempre el cursor del último contacto,
// también en la última página incompleta.
func (l lectorContactosContadoPrueba) ListarContactosBolsa(_ context.Context, bolsa, cursor string, limite int) (ports.PaginaContactosParticipacion, error) {
	*l.llamadas++
	desde := 0
	if cursor != "" {
		desde = limite * (*l.llamadas - 1)
	}
	var pagina ports.PaginaContactosParticipacion
	for i := desde; i < l.total && i < desde+limite; i++ {
		pagina.Contactos = append(pagina.Contactos, bolsadominio.ContactoParticipacion{ContactoRef: "contacto:" + string(rune('a'+i%26)), BolsaRef: bolsa})
	}
	if len(pagina.Contactos) > 0 {
		pagina.CursorSiguiente = pagina.Contactos[len(pagina.Contactos)-1].ContactoRef
	}
	return pagina, nil
}

// Cada página consume una autorización V3: una página incompleta es la última
// y no se pide otra vacía.
func TestBolsasRRHHContactosNoPidePaginaVaciaTrasUltima(t *testing.T) {
	for _, caso := range []struct{ total, llamadas int }{{0, 1}, {3, 1}, {100, 2}, {150, 2}, {250, 3}} {
		llamadas := 0
		h := &bolsasRRHHDesarrollo{contactos: lectorContactosContadoPrueba{llamadas: &llamadas, total: caso.total}}
		vista := &bolsasRRHHDesarrolloDatos{}
		if !h.cargarContactos(context.Background(), vista, "bolsa:prueba") || len(vista.datos.Contactos) != caso.total {
			t.Fatalf("total=%d: contactos=%d", caso.total, len(vista.datos.Contactos))
		}
		if llamadas != caso.llamadas {
			t.Fatalf("total=%d: %d páginas autorizadas; se esperaban %d", caso.total, llamadas, caso.llamadas)
		}
	}
}
