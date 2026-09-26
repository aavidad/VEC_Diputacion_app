package application

import (
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// EntornoV3Prueba expone a las pruebas externas del módulo una Orden, un
// emisor de material V3 real (PDP y criptografía de prueba) y su reloj.
func EntornoV3Prueba(t *testing.T, superficie dominiovec.SuperficieAutenticacionActorV1) (Orden, ports.EmisorMaterialV3, ports.Reloj) {
	t.Helper()
	e := nuevoEntornoSeleccion(t, superficie)
	return e.orden, e.emisor, &relojAutorizacionServicioPrueba{ahora: e.ahora}
}

// ProtectorPrueba expone el protector reversible de prueba.
func ProtectorPrueba() ports.ProtectorDatosSolicitud { return protectorPrueba{} }
