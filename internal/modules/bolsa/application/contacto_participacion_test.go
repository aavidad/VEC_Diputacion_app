package application

import "testing"

func TestServicioContactoParticipacionExigeDependencias(t *testing.T) {
	if _, err := NuevoServicioContactoParticipacion(nil, nil, nil, nil); err == nil {
		t.Fatal("servicio incompleto admitido")
	}
}
