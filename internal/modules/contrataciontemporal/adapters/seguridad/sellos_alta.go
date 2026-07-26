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

// ConfiguracionSelladorHMAC conserva únicamente la referencia pública de la
// generación y un conector opaco. Nunca recibe ni expone la clave.
type ConfiguracionSelladorHMAC struct {
	referenciaClave string
	sellador        selladorHMAC
}

func NuevaConfiguracionSelladorHMAC(
	referenciaClave string,
	sellador selladorHMAC,
) (ConfiguracionSelladorHMAC, error) {
	if referenciaClave == "" || dependenciaNula(sellador) {
		return ConfiguracionSelladorHMAC{}, ErrSelladoAltaNoDisponible
	}
	return ConfiguracionSelladorHMAC{
		referenciaClave: referenciaClave,
		sellador:        sellador,
	}, nil
}

type llaveroHMAC struct {
	dominio      string
	generaciones []ConfiguracionSelladorHMAC
}

func nuevoLlaveroHMAC(
	dominio string,
	activa ConfiguracionSelladorHMAC,
	retenidas []ConfiguracionSelladorHMAC,
) (*llaveroHMAC, error) {
	if len(retenidas)+1 > ports.MaximoGeneracionesHMACAlta ||
		!referenciaClaveDominioValida(activa.referenciaClave, dominio) ||
		dependenciaNula(activa.sellador) {
		return nil, ErrSelladoAltaNoDisponible
	}
	configuraciones := make([]ConfiguracionSelladorHMAC, 0, len(retenidas)+1)
	configuraciones = append(configuraciones, activa)
	configuraciones = append(configuraciones, retenidas...)
	anterior := generacionReferencia(activa.referenciaClave, dominio)
	vistas := map[string]struct{}{activa.referenciaClave: {}}
	for indice := 1; indice < len(configuraciones); indice++ {
		configuracion := configuraciones[indice]
		generacion := generacionReferencia(configuracion.referenciaClave, dominio)
		if generacion == 0 || generacion >= anterior ||
			!referenciaClaveDominioValida(configuracion.referenciaClave, dominio) ||
			dependenciaNula(configuracion.sellador) {
			return nil, ErrSelladoAltaNoDisponible
		}
		if _, repetida := vistas[configuracion.referenciaClave]; repetida {
			return nil, ErrSelladoAltaNoDisponible
		}
		vistas[configuracion.referenciaClave] = struct{}{}
		anterior = generacion
	}
	return &llaveroHMAC{
		dominio:      dominio,
		generaciones: append([]ConfiguracionSelladorHMAC(nil), configuraciones...),
	}, nil
}

func (l *llaveroHMAC) sellar(
	ctx context.Context,
	contenido []byte,
) (ports.ColeccionSellosHMAC, error) {
	if ctx == nil || l == nil || len(l.generaciones) == 0 ||
		len(l.generaciones) > ports.MaximoGeneracionesHMACAlta {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	sellos := make([]string, 0, len(l.generaciones))
	for _, configuracion := range l.generaciones {
		if err := ctx.Err(); err != nil {
			return ports.ColeccionSellosHMAC{}, err
		}
		sello, err := configuracion.sellador.SellarDatos(ctx, contenido)
		if err != nil {
			if ctx.Err() != nil {
				return ports.ColeccionSellosHMAC{}, ctx.Err()
			}
			return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
		}
		if !selloDelDominio(sello, configuracion.referenciaClave) {
			return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
		}
		sellos = append(sellos, sello)
	}
	coleccion, err := ports.NuevaColeccionSellosHMAC(sellos[0], sellos[1:])
	if err != nil || coleccion.ValidarDominio(l.dominio) != nil {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	return coleccion, nil
}

type DerivadorHuellaAltaHMAC struct {
	llavero *llaveroHMAC
}

func NuevoDerivadorHuellaAltaHMAC(
	referenciaClave string,
	sellador selladorHMAC,
) (*DerivadorHuellaAltaHMAC, error) {
	configuracion, err := NuevaConfiguracionSelladorHMAC(
		referenciaClave,
		sellador,
	)
	if err != nil {
		return nil, ErrSelladoAltaNoDisponible
	}
	return NuevoDerivadorHuellaAltaHMACRotable(configuracion, nil)
}

func NuevoDerivadorHuellaAltaHMACRotable(
	activa ConfiguracionSelladorHMAC,
	retenidas []ConfiguracionSelladorHMAC,
) (*DerivadorHuellaAltaHMAC, error) {
	llavero, err := nuevoLlaveroHMAC(dominioClaveHuella, activa, retenidas)
	if err != nil {
		return nil, err
	}
	return &DerivadorHuellaAltaHMAC{llavero: llavero}, nil
}

func (d *DerivadorHuellaAltaHMAC) DerivarHuellaAlta(
	ctx context.Context,
	material ports.MaterialHuellaAlta,
) (ports.ColeccionSellosHMAC, error) {
	if ctx == nil || d == nil || d.llavero == nil || material.Validar() != nil {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	contenido, err := materialCanonicoHuellaAlta(material)
	if err != nil {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	defer borrar(contenido)
	return d.llavero.sellar(ctx, contenido)
}

type SelladorAmbitoIdempotenciaHMAC struct {
	llavero *llaveroHMAC
}

func NuevoSelladorAmbitoIdempotenciaHMAC(
	referenciaClave string,
	sellador selladorHMAC,
) (*SelladorAmbitoIdempotenciaHMAC, error) {
	configuracion, err := NuevaConfiguracionSelladorHMAC(
		referenciaClave,
		sellador,
	)
	if err != nil {
		return nil, ErrSelladoAltaNoDisponible
	}
	return NuevoSelladorAmbitoIdempotenciaHMACRotable(configuracion, nil)
}

func NuevoSelladorAmbitoIdempotenciaHMACRotable(
	activa ConfiguracionSelladorHMAC,
	retenidas []ConfiguracionSelladorHMAC,
) (*SelladorAmbitoIdempotenciaHMAC, error) {
	llavero, err := nuevoLlaveroHMAC(dominioClaveAmbito, activa, retenidas)
	if err != nil {
		return nil, err
	}
	return &SelladorAmbitoIdempotenciaHMAC{llavero: llavero}, nil
}

func (s *SelladorAmbitoIdempotenciaHMAC) SellarAmbitoIdempotencia(
	ctx context.Context,
	solicitud ports.SolicitudSellarAmbitoIdempotencia,
) (ports.ColeccionSellosHMAC, error) {
	if ctx == nil || s == nil || s.llavero == nil || solicitud.Validar() != nil {
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ColeccionSellosHMAC{}, err
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
		return ports.ColeccionSellosHMAC{}, ErrSelladoAltaNoDisponible
	}
	defer borrar(contenido)
	return s.llavero.sellar(ctx, contenido)
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

func generacionReferencia(valor, dominio string) uint64 {
	if !referenciaClaveDominioValida(valor, dominio) {
		return 0
	}
	generacion := strings.TrimPrefix(valor, dominio+"/v")
	numero, err := strconv.ParseUint(generacion, 10, 32)
	if err != nil {
		return 0
	}
	return numero
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
