package ports

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func ordenDenegacionFronteraIdentidadV1Valida() OrdenDenegacionFronteraIdentidadV1 {
	return OrdenDenegacionFronteraIdentidadV1{
		CorrelacionRef: "correlacion_0123456789abcdef0123456789abcdef",
		Superficie:     string(SuperficieDenegacionFronteraIdentidadV1ExternaPersonal),
		RutaExacta:     RutaExactaDenegacionFronteraIdentidadV1ParticipacionesPropias,
		Accion:         string(AccionDenegacionFronteraIdentidadV1ConsultarParticipacionesPropias),
		Motivo:         string(MotivoDenegacionFronteraIdentidadV1AutenticacionRequerida),
		CanalRef:       "tls-exportador:sha256:" + strings.Repeat("a", 64),
	}
}

func TestOrdenDenegacionFronteraIdentidadV1EsMinimaYCerrada(t *testing.T) {
	orden := ordenDenegacionFronteraIdentidadV1Valida()
	if err := orden.Validar(); err != nil {
		t.Fatalf("orden valida: %v", err)
	}
	for nombre, mutar := range map[string]func(*OrdenDenegacionFronteraIdentidadV1){
		"correlacion libre": func(o *OrdenDenegacionFronteraIdentidadV1) { o.CorrelacionRef = "corr_libre" },
		"superficie libre":  func(o *OrdenDenegacionFronteraIdentidadV1) { o.Superficie = "publica_anonima" },
		"ruta con consulta": func(o *OrdenDenegacionFronteraIdentidadV1) { o.RutaExacta += "?dato=privado" },
		"ruta ajena":        func(o *OrdenDenegacionFronteraIdentidadV1) { o.RutaExacta = "/api/vec/bolsa/otra" },
		"accion libre":      func(o *OrdenDenegacionFronteraIdentidadV1) { o.Accion = "bolsa.otra.accion" },
		"motivo libre":      func(o *OrdenDenegacionFronteraIdentidadV1) { o.Motivo = "detalle_privado" },
		"canal no opaco": func(o *OrdenDenegacionFronteraIdentidadV1) {
			o.CanalRef = "tls-exportador:sha256:" + strings.Repeat("A", 64)
		},
		"actor en preautenticacion": func(o *OrdenDenegacionFronteraIdentidadV1) { o.ActorRef = "per_actor_opaco_001" },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := orden
			mutar(&alterada)
			if !errors.Is(alterada.Validar(), ErrOrdenDenegacionFronteraIdentidadV1Invalida) {
				t.Fatalf("orden invalida aceptada: %#v", alterada)
			}
		})
	}
}

func TestOrdenDenegacionFronteraIdentidadV1AdmiteSuperficiesDelCatalogo(t *testing.T) {
	for _, superficie := range []SuperficieDenegacionFronteraIdentidadV1{
		SuperficieDenegacionFronteraIdentidadV1ExternaPersonal,
		SuperficieDenegacionFronteraIdentidadV1InternaCorporativa,
		SuperficieDenegacionFronteraIdentidadV1AdministracionPrivilegiada,
	} {
		orden := ordenDenegacionFronteraIdentidadV1Valida()
		orden.Superficie = string(superficie)
		orden.CanalRef = ""
		orden.Motivo = string(MotivoDenegacionFronteraIdentidadV1AccesoDenegado)
		if err := orden.Validar(); err != nil {
			t.Fatalf("superficie %q rechazada: %v", superficie, err)
		}
	}
}

func TestOrdenDenegacionFronteraIdentidadV1NoTransportaMaterialSensibleONoServidor(t *testing.T) {
	tipo := reflect.TypeOf(OrdenDenegacionFronteraIdentidadV1{})
	prohibidos := map[string]struct{}{
		"cabecera": {}, "ip": {}, "cuerpo": {}, "token": {}, "cert": {}, "dni": {},
		"correo": {}, "sesion": {}, "asercion": {}, "instante": {}, "time": {},
	}
	for indice := 0; indice < tipo.NumField(); indice++ {
		campo := tipo.Field(indice)
		nombre := strings.ToLower(campo.Name)
		for prohibido := range prohibidos {
			if strings.Contains(nombre, prohibido) {
				t.Fatalf("campo prohibido en la orden minimizada: %s", campo.Name)
			}
		}
		if campo.Type == reflect.TypeOf(time.Time{}) {
			t.Fatalf("la orden no admite instante: %s", campo.Name)
		}
	}
}

var _ RegistradorDenegacionFronteraIdentidadV1 = registradorDenegacionFronteraIdentidadV1Prueba{}

type registradorDenegacionFronteraIdentidadV1Prueba struct{}

func (registradorDenegacionFronteraIdentidadV1Prueba) RegistrarDenegacionFronteraIdentidadV1(
	context.Context,
	OrdenDenegacionFronteraIdentidadV1,
) error {
	return nil
}
