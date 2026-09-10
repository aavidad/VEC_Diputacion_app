package incorporacionejercicio

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	esquemaPlanesPreparacionV2 = "vec.contratacion-temporal.planes-incorporacion.v2"
	maximoBytesPlanesV2        = 256 << 10
	maximoPlanesV2             = 256
	maximaProfundidadPlanesV2  = 16
)

type documentoPlanesPreparacionV2 struct {
	Esquema    string                     `json:"esquema"`
	Referencia string                     `json:"referencia"`
	Version    uint64                     `json:"version"`
	Planes     []PlanPreparacionDurableV2 `json:"planes"`
}

// fuentePlanesPreparacionV2 es una instantánea administrativa, no un permiso
// ni un registro de alta. Las solicitudes e idempotencias proceden del documento.
type fuentePlanesPreparacionV2 struct {
	planes map[[2]string]PlanPreparacionDurableV2
}

var _ FuentePlanesPreparacionV2 = (*fuentePlanesPreparacionV2)(nil)

// NuevaFuentePlanesPreparacionV2 coteja los bytes exactos con la referencia
// fijada por composición confiable antes de interpretar la configuración.
func NuevaFuentePlanesPreparacionV2(contenido []byte, esperada ports.ReferenciaVersionadaPersonalRPT) (*fuentePlanesPreparacionV2, error) {
	fallo := ports.ErrComposicionIncorporacionAplicacion
	if len(contenido) == 0 || len(contenido) > maximoBytesPlanesV2 || esperada.Validar() != nil || !utf8.Valid(contenido) {
		return nil, fallo
	}
	suma := sha256.Sum256(contenido)
	if hex.EncodeToString(suma[:]) != esperada.HuellaSHA256 {
		return nil, fallo
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.UseNumber()
	if validarValorPlanesV2(dec, reflect.TypeFor[documentoPlanesPreparacionV2](), 0) != nil {
		return nil, fallo
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fallo
	}
	var documento documentoPlanesPreparacionV2
	dec = json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if dec.Decode(&documento) != nil || documento.Esquema != esquemaPlanesPreparacionV2 || documento.Referencia != esperada.Referencia || documento.Version != esperada.Version || len(documento.Planes) == 0 || len(documento.Planes) > maximoPlanesV2 {
		return nil, fallo
	}
	f := &fuentePlanesPreparacionV2{planes: make(map[[2]string]PlanPreparacionDurableV2, len(documento.Planes))}
	for _, p := range documento.Planes {
		if p.Validar() != nil {
			return nil, fallo
		}
		clave := [2]string{p.OrganizacionRef, p.SolicitudPersonal.ExpedienteRef}
		if _, existe := f.planes[clave]; existe {
			return nil, fallo
		}
		f.planes[clave] = p.Copia()
	}
	return f, nil
}

// ResolverPlan no genera material nuevo ni interpreta el plan como autoridad.
func (f *fuentePlanesPreparacionV2) ResolverPlan(ctx context.Context, org, exp string) (PlanPreparacionDurableV2, error) {
	var cero PlanPreparacionDurableV2
	if f == nil || ctx == nil {
		return cero, ports.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if !dom.ReferenciaOpacaValida(org) || !dom.ReferenciaOpacaValida(exp) {
		return cero, ports.ErrComposicionIncorporacionAplicacion
	}
	p, existe := f.planes[[2]string{org, exp}]
	if !existe {
		return cero, ports.ErrComposicionIncorporacionAplicacion
	}
	return p.Copia(), nil
}

// Cierra las claves con el contrato real (incluidos nombres sin etiqueta JSON),
// sin la comparación insensible a mayúsculas de encoding/json. El recorrido
// previo impide duplicados, nulos escalares y colecciones/profundidad excesivas.
func validarValorPlanesV2(dec *json.Decoder, tipo reflect.Type, profundidad int) error {
	fallo := ports.ErrComposicionIncorporacionAplicacion
	if profundidad > maximaProfundidadPlanesV2 {
		return fallo
	}
	token, err := dec.Token()
	if err != nil {
		return fallo
	}
	if tipo == reflect.TypeFor[time.Time]() {
		if _, ok := token.(string); !ok {
			return fallo
		}
		return nil // time.Time valida el valor al decodificar el documento.
	}
	switch tipo.Kind() {
	case reflect.Struct:
		if token != json.Delim('{') {
			return fallo
		}
		campos := make(map[string]reflect.Type, tipo.NumField())
		for i := 0; i < tipo.NumField(); i++ {
			campo := tipo.Field(i)
			nombre := strings.Split(campo.Tag.Get("json"), ",")[0]
			if nombre == "" {
				nombre = campo.Name
			}
			if campo.IsExported() && nombre != "-" {
				campos[nombre] = campo.Type
			}
		}
		for dec.More() {
			clave, err := dec.Token()
			nombre, ok := clave.(string)
			campo, existe := campos[nombre]
			if err != nil || !ok || !existe {
				return fallo
			}
			delete(campos, nombre)
			if validarValorPlanesV2(dec, campo, profundidad+1) != nil {
				return fallo
			}
		}
		fin, err := dec.Token()
		if err != nil || fin != json.Delim('}') {
			return fallo
		}
	case reflect.Slice:
		if token == nil {
			return nil
		} // Una lista opcional nula equivale a vacía.
		if token != json.Delim('[') {
			return fallo
		}
		for n := 0; dec.More(); n++ {
			if n >= maximoPlanesV2 || validarValorPlanesV2(dec, tipo.Elem(), profundidad+1) != nil {
				return fallo
			}
		}
		fin, err := dec.Token()
		if err != nil || fin != json.Delim(']') {
			return fallo
		}
	default:
		if _, compuesto := token.(json.Delim); compuesto || token == nil {
			return fallo
		}
	}
	return nil
}
