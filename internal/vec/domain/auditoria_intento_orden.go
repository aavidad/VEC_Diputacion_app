package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

var (
	ErrOrdenIntentoAuditoriaInvalida = errors.New("vec: orden de intento de auditoria invalida")
	ErrAcuseIntentoAuditoriaInvalido = errors.New("vec: acuse de intento de auditoria invalido")
	ErrIntentoAuditoriaNoDisponible  = errors.New("vec: registro de intento de auditoria no disponible")
	ErrIntentoAuditoriaConflicto     = errors.New("vec: referencia de intento de auditoria en conflicto")
)

// OrdenIntentoAuditoria es un hecho fallido con identidad ya acreditada. No
// concede autorización ni reemplaza la auditoría del efecto confirmado. Se
// crea después de que el repositorio de negocio haya cerrado su transacción.
// Una misma orden conserva su referencia al reintentar el registro.
type OrdenIntentoAuditoria struct {
	intentoRef        string
	resultadoContexto ResultadoContextoActorRegistradoV2
	vinculo           VinculoAutenticacionActorV2
	datos             DatosIntentoAuditoria
}

// NuevaReferenciaIntentoAuditoria crea una clave opaca del servidor. No debe
// derivarse de una referencia, identidad o correlación enviada por el cliente.
func NuevaReferenciaIntentoAuditoria() (string, error) {
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return "", ErrIntentoAuditoriaNoDisponible
	}
	return "intento_" + hex.EncodeToString(aleatorio[:]), nil
}

func NuevaOrdenIntentoAuditoria(
	intentoRef string,
	resultadoContexto ResultadoContextoActorRegistradoV2,
	vinculo VinculoAutenticacionActorV2,
	datos DatosIntentoAuditoria,
) (OrdenIntentoAuditoria, error) {
	datosVinculo, errVinculo := vinculo.Datos()
	if !referenciaIntentoAuditoriaValida(intentoRef) ||
		errVinculo != nil || datos.Canal != string(datosVinculo.Superficie) ||
		vinculo.ValidarPara(resultadoContexto) != nil || datos.Validar() != nil {
		return OrdenIntentoAuditoria{}, ErrOrdenIntentoAuditoriaInvalida
	}
	copia, err := resultadoContexto.Clonar()
	if err != nil {
		return OrdenIntentoAuditoria{}, ErrOrdenIntentoAuditoriaInvalida
	}
	return OrdenIntentoAuditoria{
		intentoRef: intentoRef, resultadoContexto: copia,
		vinculo: vinculo, datos: datos,
	}, nil
}

// DatosOrdenIntentoAuditoria es la vista defensiva para el adaptador. Ninguna
// de estas referencias procede de la solicitud HTTP sin la frontera confiable.
type DatosOrdenIntentoAuditoria struct {
	IntentoRef        string
	ResultadoContexto ResultadoContextoActorRegistradoV2
	Vinculo           VinculoAutenticacionActorV2
	Datos             DatosIntentoAuditoria
}

func (o OrdenIntentoAuditoria) Datos() (DatosOrdenIntentoAuditoria, error) {
	datosVinculo, errVinculo := o.vinculo.Datos()
	if !referenciaIntentoAuditoriaValida(o.intentoRef) ||
		errVinculo != nil || o.datos.Canal != string(datosVinculo.Superficie) ||
		o.vinculo.ValidarPara(o.resultadoContexto) != nil || o.datos.Validar() != nil {
		return DatosOrdenIntentoAuditoria{}, ErrOrdenIntentoAuditoriaInvalida
	}
	copia, err := o.resultadoContexto.Clonar()
	if err != nil {
		return DatosOrdenIntentoAuditoria{}, ErrOrdenIntentoAuditoriaInvalida
	}
	return DatosOrdenIntentoAuditoria{
		IntentoRef: o.intentoRef, ResultadoContexto: copia,
		Vinculo: o.vinculo, Datos: o.datos,
	}, nil
}

func (OrdenIntentoAuditoria) String() string     { return "[ORDEN-INTENTO-AUDITORIA-OPACA]" }
func (o OrdenIntentoAuditoria) GoString() string { return o.String() }
func (o OrdenIntentoAuditoria) Format(s fmt.State, _ rune) {
	if _, err := io.WriteString(s, o.String()); err != nil {
		panic(ErrOrdenIntentoAuditoriaInvalida)
	}
}
func (o OrdenIntentoAuditoria) LogValue() slog.Value { return slog.StringValue(o.String()) }
func (OrdenIntentoAuditoria) MarshalJSON() ([]byte, error) {
	return nil, ErrOrdenIntentoAuditoriaInvalida
}
func (*OrdenIntentoAuditoria) UnmarshalJSON([]byte) error { return ErrOrdenIntentoAuditoriaInvalida }
func (OrdenIntentoAuditoria) MarshalText() ([]byte, error) {
	return nil, ErrOrdenIntentoAuditoriaInvalida
}
func (*OrdenIntentoAuditoria) UnmarshalText([]byte) error { return ErrOrdenIntentoAuditoriaInvalida }
func (OrdenIntentoAuditoria) MarshalBinary() ([]byte, error) {
	return nil, ErrOrdenIntentoAuditoriaInvalida
}
func (*OrdenIntentoAuditoria) UnmarshalBinary([]byte) error { return ErrOrdenIntentoAuditoriaInvalida }
func (OrdenIntentoAuditoria) GobEncode() ([]byte, error) {
	return nil, ErrOrdenIntentoAuditoriaInvalida
}
func (*OrdenIntentoAuditoria) GobDecode([]byte) error { return ErrOrdenIntentoAuditoriaInvalida }

func referenciaIntentoAuditoriaValida(valor string) bool {
	if len(valor) != len("intento_")+32 || !strings.HasPrefix(valor, "intento_") {
		return false
	}
	for _, r := range valor[len("intento_"):] {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
