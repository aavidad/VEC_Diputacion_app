package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestPanelInternoExigePDPV2AntesDeConsultar(t *testing.T) {
	servicio, consulta, exigidor := nuevoServicioPanelPrueba(t)
	orden := ordenPanelPrueba(t)
	exigidor.observar = func(recurso dominiovec.RecursoAutorizable) {
		if consulta.llamadas != 0 || recurso.ModuloID != puertosbolsa.ModuloPanelInternoBolsa ||
			recurso.Tipo != puertosbolsa.TipoRecursoPanelInternoBolsa ||
			recurso.Ambitos["organizacion_ref"] != orden.Selector.OrganizacionRef ||
			recurso.Ambitos["unidad_gestion_ref"] != orden.Selector.UnidadGestionRef {
			t.Fatalf("PDP fuera de orden o alcance: repo=%d recurso=%+v", consulta.llamadas, recurso)
		}
	}
	consulta.antes = func(solicitud puertosbolsa.SolicitudConsultaPanelInterno) {
		if exigidor.llamadas != 1 {
			t.Fatalf("consulta anterior al PDP: %s", describirLlamadasPanel(consulta.llamadas, exigidor.llamadas))
		}
		evidencia, err := solicitud.Autorizacion()
		selector, errSelector := solicitud.Selector()
		if err != nil || errSelector != nil || selector != orden.Selector ||
			evidencia.ValidarEn(instantePanelInternoPrueba) != nil {
			t.Fatalf("solicitud durable sin capacidad exacta: selector=%+v err=%v", selector, errors.Join(err, errSelector))
		}
	}

	resultado, err := servicio.Consultar(context.Background(), orden)
	if err != nil || resultado.Esquema != puertosbolsa.EsquemaPanelInternoBolsaV1 ||
		consulta.llamadas != 1 || exigidor.llamadas != 1 {
		t.Fatalf("consulta productiva: %s resultado=%+v error=%v",
			describirLlamadasPanel(consulta.llamadas, exigidor.llamadas), resultado, err)
	}
}

func TestPanelInternoRechazaContextosNoProductivosAntesDelPDP(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*testing.T, *OrdenConsultaPanelInterno)
	}{
		{
			nombre: "metodo de demostracion",
			mutar: func(t *testing.T, orden *OrdenConsultaPanelInterno) {
				orden.ContextoActor = nuevoContextoPanelPrueba(
					t, dominiovec.AuthMethodDemo, dominiovec.AuthAssuranceHigh,
				)
				orden.VinculoAutenticacionActor = dominiovec.VinculoAutenticacionActorV1{}
			},
		},
		{
			nombre: "superficie externa",
			mutar: func(t *testing.T, orden *OrdenConsultaPanelInterno) {
				actor, vinculo := nuevoContextoYVinculoPanelPrueba(
					t,
					dominiovec.AuthMethodCertificate,
					dominiovec.AuthAssuranceHigh,
					dominiovec.SuperficieAutenticacionExternaPersonalV1,
				)
				orden.ContextoActor, orden.VinculoAutenticacionActor = actor, vinculo
			},
		},
		{
			nombre: "garantia baja",
			mutar: func(t *testing.T, orden *OrdenConsultaPanelInterno) {
				actor, vinculo := nuevoContextoYVinculoPanelPrueba(
					t,
					dominiovec.AuthMethodCertificate,
					dominiovec.AuthAssuranceLow,
					dominiovec.SuperficieAutenticacionInternaCorporativaV1,
				)
				orden.ContextoActor, orden.VinculoAutenticacionActor = actor, vinculo
			},
		},
		{
			nombre: "ambito ambiguo",
			mutar: func(_ *testing.T, orden *OrdenConsultaPanelInterno) {
				orden.Selector = puertosbolsa.SelectorPanelInterno{
					Clase:            puertosbolsa.AmbitoPanelOrganizacion,
					OrganizacionRef:  "org_0123456789abcdef",
					UnidadGestionRef: "uni_fedcba9876543210",
				}
			},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio, consulta, exigidor := nuevoServicioPanelPrueba(t)
			orden := ordenPanelPrueba(t)
			caso.mutar(t, &orden)
			_, err := servicio.Consultar(context.Background(), orden)
			if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) ||
				!errors.Is(err, ErrOrdenPanelInternoInvalida) ||
				consulta.llamadas != 0 || exigidor.llamadas != 0 {
				t.Fatalf("contexto inseguro aceptado: %s error=%v",
					describirLlamadasPanel(consulta.llamadas, exigidor.llamadas), err)
			}
		})
	}
}

func TestPanelInternoRechazaOrigenDeDemostracion(t *testing.T) {
	servicio, consulta, exigidor := nuevoServicioPanelPrueba(t)
	consulta.resultado.Origen.Demostracion = true
	_, err := servicio.Consultar(context.Background(), ordenPanelPrueba(t))
	if !errors.Is(err, ErrDatosPanelInternoNoConfiables) ||
		!errors.Is(err, puertosbolsa.ErrResultadoPanelInternoInvalido) ||
		consulta.llamadas != 1 || exigidor.llamadas != 1 {
		t.Fatalf("origen demo aceptado: %s error=%v",
			describirLlamadasPanel(consulta.llamadas, exigidor.llamadas), err)
	}
}

func TestDTOPanelInternoNoContienePIINiColeccionGlobalDeCandidatos(t *testing.T) {
	prohibidos := []string{
		"dni", "nif", "nie", "nombre", "apellido", "email", "correo", "telefono",
		"direccion", "nacimiento", "persona", "candidato", "interesado", "principal",
	}
	visitados := make(map[reflect.Type]bool)
	var revisar func(reflect.Type, string)
	revisar = func(tipo reflect.Type, ruta string) {
		for tipo.Kind() == reflect.Pointer || tipo.Kind() == reflect.Slice || tipo.Kind() == reflect.Array {
			tipo = tipo.Elem()
		}
		if tipo.Kind() != reflect.Struct || tipo.PkgPath() == "time" || visitados[tipo] {
			return
		}
		visitados[tipo] = true
		for indice := 0; indice < tipo.NumField(); indice++ {
			campo := tipo.Field(indice)
			nombre := strings.ToLower(campo.Name + " " + campo.Tag.Get("json"))
			for _, prohibido := range prohibidos {
				if strings.Contains(nombre, prohibido) {
					t.Errorf("PII o listado global en %s.%s: %q", ruta, campo.Name, nombre)
				}
			}
			revisar(campo.Type, ruta+"."+campo.Name)
		}
	}
	revisar(reflect.TypeOf(puertosbolsa.InstantaneaPanelInterno{}), "InstantaneaPanelInterno")
}
