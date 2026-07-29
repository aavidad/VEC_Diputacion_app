package ports

import (
	"context"
	"errors"
	"reflect"
	"testing"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type emisorMaterialAutorizacionAtestadaV3Prueba struct {
	marca byte
}

func (emisorMaterialAutorizacionAtestadaV3Prueba) EmitirMaterialAutorizacionAtestadaV3(
	context.Context,
	dominiovec.SolicitudAutorizacionLigadaV3,
	dominiovec.ResultadoContextoActorRegistradoV2,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3,
	error,
) {
	return dominiovec.DecisionAutorizacionLigadaV3{},
		puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
		nil,
		nil
}

var _ emisorMaterialAutorizacionAtestadaV3 = (*emisorMaterialAutorizacionAtestadaV3Prueba)(nil)

func TestEmisorMaterialAutorizacionAtestadaV3ConservaFirmaExacta(t *testing.T) {
	t.Parallel()

	tipo := reflect.TypeOf((*emisorMaterialAutorizacionAtestadaV3)(nil)).Elem()
	if tipo.Kind() != reflect.Interface || tipo.NumMethod() != 1 {
		t.Fatalf("contrato estructural inesperado: %v", tipo)
	}
	metodo, existe := tipo.MethodByName("EmitirMaterialAutorizacionAtestadaV3")
	if !existe {
		t.Fatal("falta el único método estructural A2.2")
	}
	firma := reflect.TypeOf(
		func(
			context.Context,
			dominiovec.SolicitudAutorizacionLigadaV3,
			dominiovec.ResultadoContextoActorRegistradoV2,
		) (
			dominiovec.DecisionAutorizacionLigadaV3,
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
			puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3,
			error,
		) {
			panic("la firma de reflexión no debe ejecutarse")
		},
	)
	if metodo.Type != firma {
		t.Fatalf("firma estructural distinta de A2.2: %v", metodo.Type)
	}
}

func TestEmisoresMaterialConsultaRRHHConservanPropiedadNominalOpaca(
	t *testing.T,
) {
	t.Parallel()

	cuadro := &emisorMaterialAutorizacionAtestadaV3Prueba{marca: 1}
	detalle := &emisorMaterialAutorizacionAtestadaV3Prueba{marca: 2}
	par, err := nuevosEmisoresMaterialConsultaRRHH(cuadro, detalle)
	if err != nil {
		t.Fatalf("crear par privado: %v", err)
	}
	if par.cuadro.emisor != cuadro || par.detalle.emisor != detalle {
		t.Fatal("el propietario no conservó cada instancia en su campo nominal")
	}

	tipo := reflect.TypeOf(par)
	if tipo.Name() != "emisoresMaterialConsultaRRHH" ||
		tipo.PkgPath() == "" || tipo.NumField() != 2 ||
		tipo.NumMethod() != 0 || reflect.PointerTo(tipo).NumMethod() != 0 {
		t.Fatalf("propietario privado inesperado: %v", tipo)
	}
	campoCuadro := tipo.Field(0)
	campoDetalle := tipo.Field(1)
	if campoCuadro.Name != "cuadro" || campoCuadro.PkgPath == "" ||
		campoDetalle.Name != "detalle" || campoDetalle.PkgPath == "" ||
		campoCuadro.Tag != "" || campoDetalle.Tag != "" ||
		campoCuadro.Type == campoDetalle.Type {
		t.Fatalf(
			"campos nominales intercambiables o expuestos: %#v, %#v",
			campoCuadro,
			campoDetalle,
		)
	}
	comprobarEnvoltorioEmisorMaterialConsultaRRHH(
		t,
		campoCuadro.Type,
		"emisorMaterialCuadroRRHH",
	)
	comprobarEnvoltorioEmisorMaterialConsultaRRHH(
		t,
		campoDetalle.Type,
		"emisorMaterialDetalleRRHH",
	)
}

func TestNuevosEmisoresMaterialConsultaRRHHCierraDependenciasAmbiguas(
	t *testing.T,
) {
	t.Parallel()

	validoA := &emisorMaterialAutorizacionAtestadaV3Prueba{marca: 1}
	validoB := &emisorMaterialAutorizacionAtestadaV3Prueba{marca: 2}
	var nuloTipado *emisorMaterialAutorizacionAtestadaV3Prueba

	casos := []struct {
		nombre  string
		cuadro  emisorMaterialAutorizacionAtestadaV3
		detalle emisorMaterialAutorizacionAtestadaV3
	}{
		{"cuadro_nulo", nil, validoB},
		{"detalle_nulo", validoA, nil},
		{"cuadro_nulo_tipado", nuloTipado, validoB},
		{"detalle_nulo_tipado", validoA, nuloTipado},
		{"misma_instancia", validoA, validoA},
		{
			"cuadro_sin_identidad_fisica",
			emisorMaterialAutorizacionAtestadaV3Prueba{marca: 1},
			validoB,
		},
		{
			"detalle_sin_identidad_fisica",
			validoA,
			emisorMaterialAutorizacionAtestadaV3Prueba{marca: 2},
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			par, err := nuevosEmisoresMaterialConsultaRRHH(
				caso.cuadro,
				caso.detalle,
			)
			if !errors.Is(
				err,
				errEmisoresMaterialConsultaRRHHNoDisponibles,
			) {
				t.Fatalf("dependencia ambigua aceptada: %+v, %v", par, err)
			}
			if !reflect.ValueOf(par).IsZero() {
				t.Fatalf("el fallo devolvió un propietario parcial: %+v", par)
			}
		})
	}

	par, err := nuevosEmisoresMaterialConsultaRRHH(validoA, validoB)
	if err != nil {
		t.Fatalf("dos instancias físicas distintas fueron rechazadas: %v", err)
	}
	if par.cuadro.emisor == par.detalle.emisor {
		t.Fatal("dos instancias distintas colapsaron en un solo emisor")
	}

	otroMismoContenido := &emisorMaterialAutorizacionAtestadaV3Prueba{marca: 1}
	if _, err := nuevosEmisoresMaterialConsultaRRHH(
		validoA,
		otroMismoContenido,
	); err != nil {
		t.Fatalf("la igualdad de datos se confundió con identidad física: %v", err)
	}
}

func comprobarEnvoltorioEmisorMaterialConsultaRRHH(
	t *testing.T,
	tipo reflect.Type,
	nombre string,
) {
	t.Helper()

	if tipo.Name() != nombre || tipo.PkgPath() == "" ||
		tipo.NumField() != 1 || tipo.NumMethod() != 0 ||
		reflect.PointerTo(tipo).NumMethod() != 0 {
		t.Fatalf("envoltorio nominal inesperado: %v", tipo)
	}
	campo := tipo.Field(0)
	contrato := reflect.TypeOf(
		(*emisorMaterialAutorizacionAtestadaV3)(nil),
	).Elem()
	if campo.Name != "emisor" || campo.PkgPath == "" ||
		campo.Type != contrato || campo.Tag != "" {
		t.Fatalf("dependencia estructural expuesta o alterada: %#v", campo)
	}
}
