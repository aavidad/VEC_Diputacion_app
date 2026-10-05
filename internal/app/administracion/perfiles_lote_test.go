package administracion

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

type relojLotePrueba struct{}

func (relojLotePrueba) Ahora() time.Time { return time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC) }

// El lote sólo se monta junto a las lecturas nominales de usuarios y con su
// catálogo y servicio; si falta algo, no hay servidor.
func TestMontajeADMINLoteSoloConUsuariosMetadatosYAutoridad(t *testing.T) {
	// El host es válido: el rechazo viene de la comprobación del lote.
	if _, ok := analizarHostAdmin("admin.example.test"); !ok {
		t.Fatal("host de prueba inválido")
	}
	base := DependenciasPerfiles{ContextoConexion: func(ctx context.Context, _ net.Conn) context.Context { return ctx },
		Reloj: relojLotePrueba{}}
	for nombre, deps := range map[string]DependenciasPerfiles{
		"sin_metadatos": func() DependenciasPerfiles {
			d := base
			d.Lote = &LoteADMIN{Organizacion: "org_prueba"}
			return d
		}(),
		"sin_servicio": func() DependenciasPerfiles {
			d := base
			d.SoloUsuariosMetadatos = true
			d.Lote = &LoteADMIN{Organizacion: "org_prueba"}
			return d
		}(),
	} {
		servidor, err := NuevoServidorConLecturas(Configuracion{Host: "admin.example.test"}, deps)
		if servidor != nil || !errors.Is(err, ErrConfiguracion) {
			t.Fatalf("%s: lote montado sin requisitos: %v", nombre, err)
		}
	}
}
