package application

import (
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// Vector de la fila semilla B57. Fija el contrato entre Go y la función SQL
// huella_causa_participacion_v1, incluidos UTF-8 y booleanos canónicos.
func TestHuellaYReciboCausaParticipacionB57(t *testing.T) {
	for _, caso := range []struct {
		causa  puertosbolsa.CausaParticipacionCatalogada
		huella string
		recibo string
	}{
		{puertosbolsa.CausaParticipacionCatalogada{Codigo: "gestion_situacion", Version: 1, Etiqueta: "Cambio de situación registrado", AplicaSituacion: true, Publicable: true, Activa: true},
			"cba47651311628ed9f05be47b12f0da5ded5f5c47b6075899045ca5803fa8f52",
			"recibo:causa:3f64b6bff0953bc2644bd1bc613f3cd2c86b0ed88acd8ec9892686d5acb317c9"},
		{puertosbolsa.CausaParticipacionCatalogada{Codigo: "prueba_utf8", Version: 1, Etiqueta: "Cambio: áéñ;|:", AplicaSituacion: true, Publicable: true, Activa: true},
			"9fc9fb53485b81871467f6093badad8dc2dfdd871448aec773a96ba9570389f4",
			"recibo:causa:47c9864995a5ba3722123964766f9cfeafa7fc51cea1da806cfbac2f95547f85"},
	} {
		if got := huellaCausaParticipacion(caso.causa); got != caso.huella {
			t.Fatalf("huella B57=%s", got)
		}
		if got := reciboCausaParticipacion(caso.causa); got != caso.recibo {
			t.Fatalf("recibo B57=%s", got)
		}
	}
}
