package domain

import "testing"

func TestCrearPropuestaDecisionCoberturaLimitaPeriodoACienAnios(t *testing.T) {
	t.Parallel()

	datos := datosPropuestaDecisionCoberturaPrueba(t)
	datos.Periodo.Fin = datos.Periodo.Inicio.AddDate(
		maximoAniosPeriodoAnalisis,
		0,
		0,
	)

	propuesta, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatalf("crear con límite exacto de cien años: %v", err)
	}
	if _, err := RestaurarPropuestaDecisionCobertura(
		propuesta.Publicacion(),
		datos.Catalogo,
		datos.Politica,
	); err != nil {
		t.Fatalf("restaurar con límite exacto de cien años: %v", err)
	}

	datos.Periodo.Fin = datos.Periodo.Fin.AddDate(0, 0, 1)
	if _, err := CrearPropuestaDecisionCobertura(datos); err != ErrDatoInvalido {
		t.Fatalf("periodo superior a cien años: obtenido %v; esperado %v", err, ErrDatoInvalido)
	}
}

func TestCanonPropuestaDecisionCoberturaRechazaPeriodoSuperiorACienAnios(
	t *testing.T,
) {
	t.Parallel()

	datos := datosPropuestaDecisionCoberturaPrueba(t)
	propuesta, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatalf("preparar propuesta válida: %v", err)
	}

	publicacion := propuesta.Publicacion()
	publicacion.Periodo.Fin = publicacion.Periodo.Inicio.AddDate(
		maximoAniosPeriodoAnalisis,
		0,
		1,
	)
	if _, err := calcularHuellaPropuestaDecisionCobertura(publicacion); err != ErrDatoInvalido {
		t.Fatalf("canon con periodo superior a cien años: obtenido %v; esperado %v", err, ErrDatoInvalido)
	}
	if _, err := RestaurarPropuestaDecisionCobertura(
		publicacion,
		datos.Catalogo,
		datos.Politica,
	); err != ErrDatoInvalido {
		t.Fatalf("restaurar periodo superior a cien años: obtenido %v; esperado %v", err, ErrDatoInvalido)
	}
}
