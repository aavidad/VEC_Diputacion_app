package domain

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type relojCanceladorVinculoV2Prueba struct {
	ahora   time.Time
	despues func()
}

func (r *relojCanceladorVinculoV2Prueba) Ahora() time.Time {
	if r.despues != nil {
		r.despues()
	}
	return r.ahora
}

func TestCrearVinculoAutenticacionActorV2ConResultadoConservaParExacto(
	t *testing.T,
) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	fuente := resultadoContextoActorRegistradoV2ConContenedoresVaciosPrueba(
		t,
		ahora,
	)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	revalidador := &revalidadorAutenticacionV2Prueba{
		resultado: autenticacion,
	}
	resolutor := &resolutorContextoRegistradoV2Prueba{resultado: fuente}

	vinculo, resultado, err := CrearVinculoAutenticacionActorV2ConResultado(
		context.Background(),
		revalidador,
		solicitudRevalidacionVinculoPrueba(autenticacion),
		resolutor,
		solicitudContextoVinculoV2Prueba(fuente),
		&relojVinculoV2Prueba{ahora: ahora},
	)
	if err != nil {
		t.Fatalf("crear par nominal V2: %v", err)
	}
	if revalidador.invocaciones != 1 || resolutor.invocaciones != 1 {
		t.Fatalf(
			"autoridades invocadas más de una vez: autenticación=%d contexto=%d",
			revalidador.invocaciones,
			resolutor.invocaciones,
		)
	}
	if !reflect.DeepEqual(resultado, fuente) {
		t.Fatal("el resultado devuelto no es el resultado registrado exacto")
	}
	if vinculo.ValidarPara(resultado) != nil ||
		!vinculo.VigenteEn(ahora, resultado) {
		t.Fatal("el vínculo no quedó ligado al resultado devuelto")
	}

	if len(resultado.RepresentacionCanonica) == 0 ||
		len(resultado.ManifiestoProcedenciaCanonico) == 0 ||
		resultado.Contexto.Principal.Roles == nil ||
		resultado.Contexto.Principal.Permissions == nil ||
		resultado.Contexto.Principal.Attributes == nil ||
		resultado.Contexto.Instantanea.Vinculos == nil ||
		&resultado.RepresentacionCanonica[0] ==
			&resolutor.resultado.RepresentacionCanonica[0] ||
		&resultado.ManifiestoProcedenciaCanonico[0] ==
			&resolutor.resultado.ManifiestoProcedenciaCanonico[0] {
		t.Fatal("el resultado comparte memoria mutable con la autoridad")
	}

	canonDevuelto := append([]byte(nil), resultado.RepresentacionCanonica...)
	manifiestoDevuelto := append(
		[]byte(nil),
		resultado.ManifiestoProcedenciaCanonico...,
	)
	resolutor.resultado.RepresentacionCanonica[0] ^= 1
	resolutor.resultado.ManifiestoProcedenciaCanonico[0] ^= 1
	if !bytes.Equal(resultado.RepresentacionCanonica, canonDevuelto) ||
		!bytes.Equal(
			resultado.ManifiestoProcedenciaCanonico,
			manifiestoDevuelto,
		) ||
		resultado.Validar() != nil ||
		vinculo.ValidarPara(resultado) != nil {
		t.Fatal("una mutación posterior de la autoridad alcanzó el resultado")
	}

	resolutor.resultado.RepresentacionCanonica[0] ^= 1
	resolutor.resultado.ManifiestoProcedenciaCanonico[0] ^= 1
	resultado.RepresentacionCanonica[0] ^= 1
	resultado.ManifiestoProcedenciaCanonico[0] ^= 1
	if resolutor.resultado.Validar() != nil {
		t.Fatal("una mutación del resultado devuelto alcanzó la autoridad")
	}
	acreditarContextosActorSinMemoriaCompartida(
		t,
		&resolutor.resultado.Contexto,
		&resultado.Contexto,
	)
}

func TestResultadoContextoActorRegistradoV2ClonaContenedoresMutables(
	t *testing.T,
) {
	origen := resultadoContextoActorRegistradoV2ConContenedoresVaciosPrueba(
		t,
		instanteVinculoAutenticacionActorV2Prueba(),
	)
	clon, err := origen.Clonar()
	if err != nil {
		t.Fatalf("clonar resultado registrado V2: %v", err)
	}
	if !reflect.DeepEqual(clon, origen) {
		t.Fatal("el clon no conserva el resultado registrado exacto")
	}
	if clon.Contexto.Principal.Roles == nil ||
		clon.Contexto.Principal.Permissions == nil ||
		clon.Contexto.Principal.Attributes == nil ||
		clon.Contexto.Instantanea.Vinculos == nil {
		t.Fatal("el clon perdió contenedores vacíos no nulos")
	}
	acreditarContextosActorSinMemoriaCompartida(
		t,
		&origen.Contexto,
		&clon.Contexto,
	)
}

func TestCrearVinculoAutenticacionActorV2ConResultadoFallaCerrado(
	t *testing.T,
) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	fuente := resultadoContextoActorRegistradoV2Prueba(t, ahora)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	solicitudAutenticacion := solicitudRevalidacionVinculoPrueba(autenticacion)
	solicitudContexto := solicitudContextoVinculoV2Prueba(fuente)
	errFuente := errors.New("autoridad no disponible")

	contextoCancelado, cancelarAntes := context.WithCancel(
		context.Background(),
	)
	cancelarAntes()

	casos := []struct {
		nombre      string
		ctx         context.Context
		revalidador *revalidadorAutenticacionV2Prueba
		resolutor   *resolutorContextoRegistradoV2Prueba
		reloj       RelojVinculoAutenticacionActorV2
		solicitudA  SolicitudRevalidacionAutenticacionActorV1
		solicitudC  SolicitudContextoActor
		llamadasA   int
		llamadasC   int
	}{
		{
			nombre: "contexto ausente",
			ctx:    nil,
			revalidador: &revalidadorAutenticacionV2Prueba{
				resultado: autenticacion,
			},
			resolutor: &resolutorContextoRegistradoV2Prueba{
				resultado: fuente,
			},
			reloj:      &relojVinculoV2Prueba{ahora: ahora},
			solicitudA: solicitudAutenticacion,
			solicitudC: solicitudContexto,
		},
		{
			nombre: "cancelación previa",
			ctx:    contextoCancelado,
			revalidador: &revalidadorAutenticacionV2Prueba{
				resultado: autenticacion,
			},
			resolutor: &resolutorContextoRegistradoV2Prueba{
				resultado: fuente,
			},
			reloj:      &relojVinculoV2Prueba{ahora: ahora},
			solicitudA: solicitudAutenticacion,
			solicitudC: solicitudContexto,
		},
		{
			nombre: "revalidación falla",
			ctx:    context.Background(),
			revalidador: &revalidadorAutenticacionV2Prueba{
				err: errFuente,
			},
			resolutor: &resolutorContextoRegistradoV2Prueba{
				resultado: fuente,
			},
			reloj:      &relojVinculoV2Prueba{ahora: ahora},
			solicitudA: solicitudAutenticacion,
			solicitudC: solicitudContexto,
			llamadasA:  1,
		},
		{
			nombre: "resolución falla",
			ctx:    context.Background(),
			revalidador: &revalidadorAutenticacionV2Prueba{
				resultado: autenticacion,
			},
			resolutor: &resolutorContextoRegistradoV2Prueba{
				err: errFuente,
			},
			reloj:      &relojVinculoV2Prueba{ahora: ahora},
			solicitudA: solicitudAutenticacion,
			solicitudC: solicitudContexto,
			llamadasA:  1,
			llamadasC:  1,
		},
		{
			nombre: "resultado registrado inválido",
			ctx:    context.Background(),
			revalidador: &revalidadorAutenticacionV2Prueba{
				resultado: autenticacion,
			},
			resolutor:  &resolutorContextoRegistradoV2Prueba{},
			reloj:      &relojVinculoV2Prueba{ahora: ahora},
			solicitudA: solicitudAutenticacion,
			solicitudC: solicitudContexto,
			llamadasA:  1,
			llamadasC:  1,
		},
		{
			nombre: "reloj vacío",
			ctx:    context.Background(),
			revalidador: &revalidadorAutenticacionV2Prueba{
				resultado: autenticacion,
			},
			resolutor: &resolutorContextoRegistradoV2Prueba{
				resultado: fuente,
			},
			reloj:      &relojVinculoV2Prueba{},
			solicitudA: solicitudAutenticacion,
			solicitudC: solicitudContexto,
			llamadasA:  1,
			llamadasC:  1,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			vinculo, resultado, err :=
				CrearVinculoAutenticacionActorV2ConResultado(
					caso.ctx,
					caso.revalidador,
					caso.solicitudA,
					caso.resolutor,
					caso.solicitudC,
					caso.reloj,
				)
			if !errors.Is(err, ErrVinculoAutenticacionActorV2Invalido) ||
				vinculo.Validar() == nil ||
				resultado.Validar() == nil {
				t.Fatalf("fallo produjo nominales utilizables: %v", err)
			}
			if caso.revalidador.invocaciones != caso.llamadasA ||
				caso.resolutor.invocaciones != caso.llamadasC {
				t.Fatalf(
					"cardinalidad inesperada: autenticación=%d contexto=%d",
					caso.revalidador.invocaciones,
					caso.resolutor.invocaciones,
				)
			}
		})
	}
}

func TestCrearVinculoAutenticacionActorV2ConResultadoRecompruebaCancelacion(
	t *testing.T,
) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	fuente := resultadoContextoActorRegistradoV2Prueba(t, ahora)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	for _, frontera := range []string{
		"revalidación",
		"resolución",
		"reloj",
	} {
		t.Run(frontera, func(t *testing.T) {
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			revalidador := &revalidadorAutenticacionV2Prueba{
				resultado: autenticacion,
			}
			resolutor := &resolutorContextoRegistradoV2Prueba{
				resultado: fuente,
			}
			var reloj RelojVinculoAutenticacionActorV2 = &relojVinculoV2Prueba{ahora: ahora}
			switch frontera {
			case "revalidación":
				revalidador.despues = cancelar
			case "resolución":
				resolutor.despues = cancelar
			case "reloj":
				reloj = &relojCanceladorVinculoV2Prueba{
					ahora:   ahora,
					despues: cancelar,
				}
			}

			vinculo, resultado, err :=
				CrearVinculoAutenticacionActorV2ConResultado(
					ctx,
					revalidador,
					solicitudRevalidacionVinculoPrueba(autenticacion),
					resolutor,
					solicitudContextoVinculoV2Prueba(fuente),
					reloj,
				)
			if !errors.Is(err, context.Canceled) ||
				vinculo.Validar() == nil ||
				resultado.Validar() == nil {
				t.Fatalf("cancelación competida no vació el par: %v", err)
			}
			if revalidador.invocaciones != 1 {
				t.Fatalf(
					"revalidaciones inesperadas: %d",
					revalidador.invocaciones,
				)
			}
			esperadasContexto := 1
			if frontera == "revalidación" {
				esperadasContexto = 0
			}
			if resolutor.invocaciones != esperadasContexto {
				t.Fatalf(
					"resoluciones inesperadas: %d",
					resolutor.invocaciones,
				)
			}
		})
	}
}

func TestCrearVinculoAutenticacionActorV2ConResultadoRechazaNulosTipados(
	t *testing.T,
) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	fuente := resultadoContextoActorRegistradoV2Prueba(t, ahora)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	var revalidador *revalidadorAutenticacionV2Prueba
	var resolutor *resolutorContextoRegistradoV2Prueba
	var reloj *relojVinculoV2Prueba
	casos := []struct {
		revalidador RevalidadorAutenticacionActorV1
		resolutor   ResolutorContextoActorRegistradoV2
		reloj       RelojVinculoAutenticacionActorV2
	}{
		{
			revalidador,
			&resolutorContextoRegistradoV2Prueba{resultado: fuente},
			&relojVinculoV2Prueba{ahora: ahora},
		},
		{
			&revalidadorAutenticacionV2Prueba{resultado: autenticacion},
			resolutor,
			&relojVinculoV2Prueba{ahora: ahora},
		},
		{
			&revalidadorAutenticacionV2Prueba{resultado: autenticacion},
			&resolutorContextoRegistradoV2Prueba{resultado: fuente},
			reloj,
		},
	}
	for indice, caso := range casos {
		vinculo, resultado, err :=
			CrearVinculoAutenticacionActorV2ConResultado(
				context.Background(),
				caso.revalidador,
				solicitudRevalidacionVinculoPrueba(autenticacion),
				caso.resolutor,
				solicitudContextoVinculoV2Prueba(fuente),
				caso.reloj,
			)
		if !errors.Is(err, ErrVinculoAutenticacionActorV2Invalido) ||
			vinculo.Validar() == nil ||
			resultado.Validar() == nil {
			t.Fatalf("nulo tipado %d produjo un par utilizable: %v", indice, err)
		}
	}
}

func TestCrearVinculoAutenticacionActorV2WrapperConservaContrato(
	t *testing.T,
) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	fuente := resultadoContextoActorRegistradoV2ConContenedoresVaciosPrueba(
		t,
		ahora,
	)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	revalidador := &revalidadorAutenticacionV2Prueba{
		resultado: autenticacion,
	}
	resolutor := &resolutorContextoRegistradoV2Prueba{resultado: fuente}

	vinculo, err := CrearVinculoAutenticacionActorV2(
		context.Background(),
		revalidador,
		solicitudRevalidacionVinculoPrueba(autenticacion),
		resolutor,
		solicitudContextoVinculoV2Prueba(fuente),
		&relojVinculoV2Prueba{ahora: ahora},
	)
	if err != nil || vinculo.ValidarPara(fuente) != nil {
		t.Fatalf("el wrapper alteró el contrato anterior: %#v, %v", vinculo, err)
	}
	if revalidador.invocaciones != 1 || resolutor.invocaciones != 1 {
		t.Fatalf(
			"el wrapper duplicó autoridades: autenticación=%d contexto=%d",
			revalidador.invocaciones,
			resolutor.invocaciones,
		)
	}

	vinculo, err = CrearVinculoAutenticacionActorV2(
		context.Background(),
		&revalidadorAutenticacionV2Prueba{resultado: autenticacion},
		solicitudRevalidacionVinculoPrueba(autenticacion),
		&resolutorContextoRegistradoV2Prueba{resultado: fuente},
		solicitudContextoVinculoV2Prueba(fuente),
		&relojVinculoV2Prueba{},
	)
	if err != ErrVinculoAutenticacionActorV2Invalido ||
		vinculo.Validar() == nil {
		t.Fatalf("el wrapper alteró el error de ventana inválida: %v", err)
	}
}

func resultadoContextoActorRegistradoV2ConContenedoresVaciosPrueba(
	t *testing.T,
	ahora time.Time,
) ResultadoContextoActorRegistradoV2 {
	t.Helper()
	resultado := resultadoContextoActorRegistradoV2Prueba(t, ahora)
	resultado.Contexto.Principal.Roles = make([]string, 0, 1)
	resultado.Contexto.Principal.Permissions = make([]string, 0, 1)
	resultado.Contexto.Principal.Attributes = make(map[string]string, 1)
	resultado.Contexto.Instantanea.Vinculos = make(
		[]VinculoReferenciaContextoActor,
		0,
		1,
	)
	if resultado.Validar() != nil {
		t.Fatal("el resultado con contenedores vacíos no nulos no es válido")
	}
	return resultado
}

func acreditarContextosActorSinMemoriaCompartida(
	t *testing.T,
	origen *ContextoActor,
	clon *ContextoActor,
) {
	t.Helper()
	origen.Principal.Roles = append(origen.Principal.Roles, "rol_origen")
	clon.Principal.Roles = append(clon.Principal.Roles, "rol_clon")
	origen.Principal.Permissions = append(
		origen.Principal.Permissions,
		"permiso.origen",
	)
	clon.Principal.Permissions = append(
		clon.Principal.Permissions,
		"permiso.clon",
	)
	origen.Principal.Attributes["origen"] = "atributo_origen"
	clon.Principal.Attributes["clon"] = "atributo_clon"
	origen.Instantanea.Vinculos = append(
		origen.Instantanea.Vinculos,
		VinculoReferenciaContextoActor{VinculoRef: "vinculo_origen"},
	)
	clon.Instantanea.Vinculos = append(
		clon.Instantanea.Vinculos,
		VinculoReferenciaContextoActor{VinculoRef: "vinculo_clon"},
	)

	if origen.Principal.Roles[0] != "rol_origen" ||
		clon.Principal.Roles[0] != "rol_clon" ||
		origen.Principal.Permissions[0] != "permiso.origen" ||
		clon.Principal.Permissions[0] != "permiso.clon" ||
		origen.Instantanea.Vinculos[0].VinculoRef != "vinculo_origen" ||
		clon.Instantanea.Vinculos[0].VinculoRef != "vinculo_clon" {
		t.Fatal("origen y clon comparten memoria de slices mutables")
	}
	if _, existe := clon.Principal.Attributes["origen"]; existe {
		t.Fatal("una mutación del mapa origen alcanzó el clon")
	}
	if _, existe := origen.Principal.Attributes["clon"]; existe {
		t.Fatal("una mutación del mapa clon alcanzó el origen")
	}
}
