package calculoexperiencia

import (
	"errors"
	"testing"
)

func TestPresupuestoSalidaRechazaAntesDeMaterializar(t *testing.T) {
	maximoAplicaciones := int(
		(uint64(maximoBytesResultadoV1) - reservaBaseSalidaV1) /
			(costeAplicacionPuntuadaV1 + costeBloqueoPuntuacionV1),
	)
	if err := comprobarPresupuestoSalidaPuntuacionV1(maximoAplicaciones, 0, 0); err != nil {
		t.Fatalf("frontera serializable rechazada: %v", err)
	}
	if err := comprobarPresupuestoSalidaPuntuacionV1(maximoAplicaciones+1, 0, 0); !errors.Is(
		err, ErrFueraDeLimites,
	) {
		t.Fatalf("salida imposible aceptada: %v", err)
	}
}

func TestPresupuestoSalidaEvitaDesbordamientoAritmetico(t *testing.T) {
	presupuesto := nuevoPresupuestoSalidaCalculoV1()
	if err := presupuesto.consumir(1, uint64(maximoBytesResultadoV1)); err != nil {
		t.Fatal(err)
	}
	if err := presupuesto.consumir(int(^uint(0)>>1), ^uint64(0)); !errors.Is(
		err, ErrFueraDeLimites,
	) {
		t.Fatalf("multiplicacion desbordada aceptada: %v", err)
	}
}

func TestCotasSalidaCubrenLosCamposMaximosV1(t *testing.T) {
	minimoAplicacion := uint64(6*maximoCaracteresExactoResultadoV1 +
		3*maximoCaracteresReferencia + 8*maximoCaracteresClave)
	if costeSeleccionAplicacionV1+costeAplicacionPuntuadaV1 < minimoAplicacion {
		t.Fatal("la cota por aplicacion ya no cubre el contrato V1")
	}
	minimoBloqueoPuntuacion := uint64(maximoCaracteresExactoResultadoV1 +
		maximoCaracteresReferencia + 2*maximoCaracteresClave)
	if costeBloqueoPuntuacionV1 < minimoBloqueoPuntuacion {
		t.Fatal("la cota por bloqueo de puntuacion ya no cubre el contrato V1")
	}
	minimoRegla := uint64(13*maximoCaracteresExactoResultadoV1 +
		2*maximoCaracteresClave)
	if costeReglaPuntuadaV1 < minimoRegla {
		t.Fatal("la cota por regla ya no cubre el contrato V1")
	}
	minimoBloqueo := uint64(maximoCaracteresReferencia +
		maximoReferenciasBloqueoV1*(maximoCaracteresClave+3))
	if costeSeleccionBloqueoV1 < minimoBloqueo {
		t.Fatal("la cota por bloqueo ya no cubre el contrato V1")
	}
}
