package administracion

import (
	"errors"
	"net/http"
	"testing"
)

func TestMontajeADMINRechazaDependenciasAusentes(t *testing.T) {
	for _, construir := range []func(Configuracion, DependenciasPerfiles) (*http.Server, error){NuevoServidorConLecturas, NuevoServidorConPerfiles} {
		servidor, err := construir(Configuracion{}, DependenciasPerfiles{})
		if servidor != nil || !errors.Is(err, ErrConfiguracion) {
			t.Fatalf("montaje sin autoridades: servidor=%v err=%v", servidor, err)
		}
	}
}
