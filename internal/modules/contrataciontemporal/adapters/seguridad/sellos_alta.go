// Package seguridad implementa adaptadores criptográficos del módulo sin
// incorporar secretos al dominio ni a la aplicación.
package seguridad

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	esquemaHuellaAltaV1 = "vec.contratacion-temporal.huella-alta.v1"
	esquemaAmbitoAltaV1 = "vec.contratacion-temporal.ambito-idempotencia.v1"
	dominioClaveHuella  = "vec.contratacion-temporal.huella-peticion"
	dominioClaveAmbito  = "vec.contratacion-temporal.ambito-idempotencia"
)

var ErrSelladoAltaNoDisponible = errors.New(
	"contratacion temporal: sellado criptografico no disponible",
)

// selladorHMAC es satisfecho por el conector HMAC común de VEC y por futuros
// conectores HSM/KMS. El módulo no recibe ni conserva el material de la clave.
type selladorHMAC interface {
	SellarDatos(context.Context, []byte) (string, error)
}

type DerivadorHuellaAltaHMAC struct {
	sellador      selladorHMAC
	referenciaKey string
}

func NuevoDerivadorHuellaAltaHMAC(
	referenciaClave string,
	sellador selladorHMAC,
) (*DerivadorHuellaAltaHMAC, error) {
	if !referenciaClaveDominioValida(referenciaClave, dominioClaveHuella) ||
		dependenciaNula(sellador) {
		return nil, ErrSelladoAltaNoDisponible
	}
	return &DerivadorHuellaAltaHMAC{
		sellador: sellador, referenciaKey: referenciaClave,
	}, nil
}

func (d *DerivadorHuellaAltaHMAC) DerivarHuellaAlta(
	ctx context.Context,
	material ports.MaterialHuellaAlta,
) (string, error) {
	if ctx == nil || d == nil || dependenciaNula(d.sellador) ||
		!referenciaClaveDominioValida(d.referenciaKey, dominioClaveHuella) ||
		material.Validar() != nil {
		return "", ErrSelladoAltaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	contenido, err := materialCanonicoHuellaAlta(material)
	if err != nil {
		return "", ErrSelladoAltaNoDisponible
	}
	defer borrar(contenido)
	return d.sellarYValidar(ctx, contenido)
}

func (d *DerivadorHuellaAltaHMAC) sellarYValidar(
	ctx context.Context,
	contenido []byte,
) (string, error) {
	sello, err := d.sellador.SellarDatos(ctx, contenido)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", ErrSelladoAltaNoDisponible
	}
	if !selloDelDominio(sello, d.referenciaKey) {
		return "", ErrSelladoAltaNoDisponible
	}
	return sello, nil
}

type SelladorAmbitoIdempotenciaHMAC struct {
	sellador      selladorHMAC
	referenciaKey string
}

func NuevoSelladorAmbitoIdempotenciaHMAC(
	referenciaClave string,
	sellador selladorHMAC,
) (*SelladorAmbitoIdempotenciaHMAC, error) {
	if !referenciaClaveDominioValida(referenciaClave, dominioClaveAmbito) ||
		dependenciaNula(sellador) {
		return nil, ErrSelladoAltaNoDisponible
	}
	return &SelladorAmbitoIdempotenciaHMAC{
		sellador: sellador, referenciaKey: referenciaClave,
	}, nil
}

func (s *SelladorAmbitoIdempotenciaHMAC) SellarAmbitoIdempotencia(
	ctx context.Context,
	solicitud ports.SolicitudSellarAmbitoIdempotencia,
) (string, error) {
	if ctx == nil || s == nil || dependenciaNula(s.sellador) ||
		!referenciaClaveDominioValida(s.referenciaKey, dominioClaveAmbito) ||
		solicitud.Validar() != nil {
		return "", ErrSelladoAltaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	contenido, err := json.Marshal(struct {
		Esquema           string `json:"esquema"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
		OrganizacionRef   string `json:"organizacion_ref"`
		ActorRef          string `json:"actor_ref"`
		PerfilRef         string `json:"perfil_ref"`
	}{
		Esquema:           esquemaAmbitoAltaV1,
		ClaveIdempotencia: solicitud.ClaveIdempotencia,
		OrganizacionRef:   solicitud.OrganizacionRef,
		ActorRef:          solicitud.ActorRef,
		PerfilRef:         solicitud.PerfilRef,
	})
	if err != nil {
		return "", ErrSelladoAltaNoDisponible
	}
	defer borrar(contenido)
	sello, err := s.sellador.SellarDatos(ctx, contenido)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", ErrSelladoAltaNoDisponible
	}
	if !selloDelDominio(sello, s.referenciaKey) {
		return "", ErrSelladoAltaNoDisponible
	}
	return sello, nil
}

func materialCanonicoHuellaAlta(
	material ports.MaterialHuellaAlta,
) ([]byte, error) {
	solicitud, err := material.Solicitud.Clonar()
	if err != nil {
		return nil, err
	}
	if solicitud.DocumentosAdjuntos == nil {
		solicitud.DocumentosAdjuntos = []string{}
	}
	return json.Marshal(struct {
		Esquema         string                 `json:"esquema"`
		OrganizacionRef string                 `json:"organizacion_ref"`
		ActorRef        string                 `json:"actor_ref"`
		PerfilRef       string                 `json:"perfil_ref"`
		Flujo           domain.ReferenciaFlujo `json:"flujo"`
		Solicitud       domain.SolicitudCentro `json:"solicitud"`
	}{
		Esquema:         esquemaHuellaAltaV1,
		OrganizacionRef: material.OrganizacionRef,
		ActorRef:        material.ActorRef,
		PerfilRef:       material.PerfilRef,
		Flujo:           material.Flujo,
		Solicitud:       solicitud,
	})
}

func referenciaClaveDominioValida(valor, dominio string) bool {
	prefijo := dominio + "/v"
	if !strings.HasPrefix(valor, prefijo) || len(valor) > 96 {
		return false
	}
	generacion := strings.TrimPrefix(valor, prefijo)
	if generacion == "" || generacion[0] == '0' {
		return false
	}
	numero, err := strconv.ParseUint(generacion, 10, 32)
	return err == nil && numero > 0
}

func selloDelDominio(sello, referenciaClave string) bool {
	return ports.SelloHMACSHA256Valido(sello) &&
		strings.HasPrefix(sello, "hmac-sha256:"+referenciaClave+":")
}

func dependenciaNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func borrar(contenido []byte) {
	for indice := range contenido {
		contenido[indice] = 0
	}
}

var (
	_ ports.DerivadorHuellaAlta        = (*DerivadorHuellaAltaHMAC)(nil)
	_ ports.SelladorAmbitoIdempotencia = (*SelladorAmbitoIdempotenciaHMAC)(nil)
)
