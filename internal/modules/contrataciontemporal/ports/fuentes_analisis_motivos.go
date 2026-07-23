package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var errMotivoFuenteAnalisisInvalido = errors.New(
	"contratacion temporal: motivo catalogado de fuente invalido",
)

const maximoParametrosMotivoFuenteAnalisis = 8

type ParametroMotivoFuenteAnalisis struct {
	Clave domain.ClaveCatalogo
	Valor domain.ClaveCatalogo
}

type VinculoMotivoFuenteAnalisis struct {
	CatalogoRef      string
	CatalogoVersion  uint64
	CatalogoHuella   string
	EntradaClave     domain.ClaveCatalogo
	ClaveMensajeI18N domain.ClaveCatalogo
	Parametros       []ParametroMotivoFuenteAnalisis
}

type MotivoFuenteAnalisis struct {
	datos *VinculoMotivoFuenteAnalisis
}

func NuevoMotivoFuenteAnalisis(
	catalogoRef string,
	catalogoVersion uint64,
	catalogoHuella string,
	entradaClave domain.ClaveCatalogo,
	claveMensajeI18N domain.ClaveCatalogo,
	parametros []ParametroMotivoFuenteAnalisis,
) (MotivoFuenteAnalisis, error) {
	motivo := MotivoFuenteAnalisis{datos: &VinculoMotivoFuenteAnalisis{
		CatalogoRef: catalogoRef, CatalogoVersion: catalogoVersion,
		CatalogoHuella: catalogoHuella, EntradaClave: entradaClave,
		ClaveMensajeI18N: claveMensajeI18N,
		Parametros:       append([]ParametroMotivoFuenteAnalisis(nil), parametros...),
	}}
	if motivo.Validar() != nil {
		return MotivoFuenteAnalisis{}, errMotivoFuenteAnalisisInvalido
	}
	return motivo, nil
}

func (m MotivoFuenteAnalisis) Validar() error {
	if m.datos == nil || !domain.ReferenciaOpacaValida(m.datos.CatalogoRef) ||
		m.datos.CatalogoVersion == 0 ||
		m.datos.CatalogoVersion > maximoEnteroSeguroFuenteAnalisis ||
		!huellaSHA256FuenteAnalisisValida(m.datos.CatalogoHuella) ||
		!m.datos.EntradaClave.Valida() ||
		!m.datos.ClaveMensajeI18N.Valida() ||
		!strings.HasPrefix(
			string(m.datos.ClaveMensajeI18N),
			"contratacion_temporal.rc.",
		) ||
		len(m.datos.Parametros) > maximoParametrosMotivoFuenteAnalisis ||
		!parametrosMotivoFuenteAnalisisValidos(m.datos.Parametros) {
		return errMotivoFuenteAnalisisInvalido
	}
	return nil
}

func parametrosMotivoFuenteAnalisisValidos(
	parametros []ParametroMotivoFuenteAnalisis,
) bool {
	ordenados := append([]ParametroMotivoFuenteAnalisis(nil), parametros...)
	sort.Slice(ordenados, func(i, j int) bool {
		return ordenados[i].Clave < ordenados[j].Clave
	})
	for indice, parametro := range parametros {
		if parametro != ordenados[indice] ||
			(indice > 0 && parametro.Clave == parametros[indice-1].Clave) ||
			!parametro.Clave.Valida() || !parametro.Valor.Valida() {
			return false
		}
	}
	return true
}

func (m MotivoFuenteAnalisis) Datos() (
	VinculoMotivoFuenteAnalisis,
	error,
) {
	if m.Validar() != nil {
		return VinculoMotivoFuenteAnalisis{}, errMotivoFuenteAnalisisInvalido
	}
	datos := *m.datos
	datos.Parametros = append(
		[]ParametroMotivoFuenteAnalisis(nil),
		m.datos.Parametros...,
	)
	return datos, nil
}

func (m MotivoFuenteAnalisis) clonar() MotivoFuenteAnalisis {
	datos, err := m.Datos()
	if err != nil {
		return MotivoFuenteAnalisis{}
	}
	return MotivoFuenteAnalisis{datos: &datos}
}

func (MotivoFuenteAnalisis) String() string {
	return "[MOTIVO-FUENTE-ANALISIS-CATALOGADO-REDACTADO]"
}

func (m MotivoFuenteAnalisis) GoString() string { return m.String() }
func (m MotivoFuenteAnalisis) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, m.String())
}
func (m MotivoFuenteAnalisis) LogValue() slog.Value {
	return slog.StringValue(m.String())
}

type SolicitudVerificarPublicacionMotivoFuenteAnalisis struct {
	Motivo                MotivoFuenteAnalisis
	HuellaRespuestaSHA256 string
	AutoridadRespuestaRef string
	GeneracionRespuesta   uint32
}

func (s SolicitudVerificarPublicacionMotivoFuenteAnalisis) Validar() error {
	if s.Motivo.Validar() != nil ||
		!huellaSHA256FuenteAnalisisValida(s.HuellaRespuestaSHA256) ||
		!domain.ReferenciaOpacaValida(s.AutoridadRespuestaRef) ||
		s.GeneracionRespuesta == 0 {
		return errMotivoFuenteAnalisisInvalido
	}
	return nil
}

func (s SolicitudVerificarPublicacionMotivoFuenteAnalisis) huellaParaPublicador(
	publicadorRef string,
) (
	string,
	error,
) {
	if s.Validar() != nil || !domain.ReferenciaOpacaValida(publicadorRef) {
		return "", errMotivoFuenteAnalisisInvalido
	}
	datos, _ := s.Motivo.Datos()
	canon := nuevoEscritorCanonFuenteAnalisis()
	canon.texto("VEC-CT-PUBLICACION-MOTIVO-RC-V1")
	canon.texto(datos.CatalogoRef)
	canon.entero64(datos.CatalogoVersion)
	canon.texto(datos.CatalogoHuella)
	canon.texto(string(datos.EntradaClave))
	canon.texto(string(datos.ClaveMensajeI18N))
	canon.entero16(uint16(len(datos.Parametros)))
	for _, parametro := range datos.Parametros {
		canon.texto(string(parametro.Clave))
		canon.texto(string(parametro.Valor))
	}
	canon.texto(s.HuellaRespuestaSHA256)
	canon.texto(s.AutoridadRespuestaRef)
	canon.entero64(uint64(s.GeneracionRespuesta))
	canon.texto(publicadorRef)
	contenido, err := canon.resultado()
	if err != nil {
		return "", errMotivoFuenteAnalisisInvalido
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

type ConfirmacionPublicacionMotivoFuenteAnalisis struct {
	datos *DatosConfirmacionPublicacionMotivoFuenteAnalisis
}

func (ConfirmacionPublicacionMotivoFuenteAnalisis) String() string {
	return "[CONFIRMACION-PUBLICACION-MOTIVO-FUENTE-ANALISIS-REDACTADA]"
}

func (c ConfirmacionPublicacionMotivoFuenteAnalisis) GoString() string {
	return c.String()
}
func (c ConfirmacionPublicacionMotivoFuenteAnalisis) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, c.String())
}
func (c ConfirmacionPublicacionMotivoFuenteAnalisis) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

type DatosConfirmacionPublicacionMotivoFuenteAnalisis struct {
	PublicadorRef         string
	PublicacionRef        string
	ReciboVerificacionRef string
	HuellaSolicitudSHA256 string
	VerificadaEn          time.Time
}

func NuevaConfirmacionPublicacionMotivoFuenteAnalisis(
	solicitud SolicitudVerificarPublicacionMotivoFuenteAnalisis,
	publicadorRef string,
	publicacionRef string,
	reciboVerificacionRef string,
	verificadaEn time.Time,
) (ConfirmacionPublicacionMotivoFuenteAnalisis, error) {
	huella, err := solicitud.huellaParaPublicador(publicadorRef)
	datos := DatosConfirmacionPublicacionMotivoFuenteAnalisis{
		PublicadorRef:         publicadorRef,
		PublicacionRef:        publicacionRef,
		ReciboVerificacionRef: reciboVerificacionRef,
		HuellaSolicitudSHA256: huella,
		VerificadaEn:          verificadaEn,
	}
	if err != nil || validarConfirmacionPublicacionMotivo(
		datos,
		solicitud,
		verificadaEn,
	) != nil {
		return ConfirmacionPublicacionMotivoFuenteAnalisis{},
			errMotivoFuenteAnalisisInvalido
	}
	return ConfirmacionPublicacionMotivoFuenteAnalisis{datos: &datos}, nil
}

func validarConfirmacionPublicacionMotivo(
	datos DatosConfirmacionPublicacionMotivoFuenteAnalisis,
	solicitud SolicitudVerificarPublicacionMotivoFuenteAnalisis,
	comprobadaEn time.Time,
) error {
	huella, err := solicitud.huellaParaPublicador(datos.PublicadorRef)
	if err != nil ||
		!domain.ReferenciaOpacaValida(datos.PublicadorRef) ||
		!domain.ReferenciaOpacaValida(datos.PublicacionRef) ||
		!domain.ReferenciaOpacaValida(datos.ReciboVerificacionRef) ||
		!bytes.Equal(
			[]byte(datos.HuellaSolicitudSHA256),
			[]byte(huella),
		) ||
		!instanteFuenteAnalisisCanonico(datos.VerificadaEn) ||
		!instanteFuenteAnalisisCanonico(comprobadaEn) ||
		comprobadaEn.Before(datos.VerificadaEn) {
		return errMotivoFuenteAnalisisInvalido
	}
	return nil
}

func (c ConfirmacionPublicacionMotivoFuenteAnalisis) ValidarPara(
	solicitud SolicitudVerificarPublicacionMotivoFuenteAnalisis,
	comprobadaEn time.Time,
) error {
	if c.datos == nil {
		return errMotivoFuenteAnalisisInvalido
	}
	return validarConfirmacionPublicacionMotivo(*c.datos, solicitud, comprobadaEn)
}

func (c ConfirmacionPublicacionMotivoFuenteAnalisis) Datos() (
	DatosConfirmacionPublicacionMotivoFuenteAnalisis,
	error,
) {
	if c.datos == nil {
		return DatosConfirmacionPublicacionMotivoFuenteAnalisis{},
			errMotivoFuenteAnalisisInvalido
	}
	return *c.datos, nil
}

func (c ConfirmacionPublicacionMotivoFuenteAnalisis) clonar() ConfirmacionPublicacionMotivoFuenteAnalisis {
	if c.datos == nil {
		return ConfirmacionPublicacionMotivoFuenteAnalisis{}
	}
	datos := *c.datos
	return ConfirmacionPublicacionMotivoFuenteAnalisis{datos: &datos}
}

type VerificadorPublicacionMotivoFuenteAnalisis interface {
	PresentadorAutoridadFuenteAnalisis
	VerificarPublicacionMotivoFuenteAnalisis(
		context.Context,
		SolicitudVerificarPublicacionMotivoFuenteAnalisis,
	) (ConfirmacionPublicacionMotivoFuenteAnalisis, error)
}

func verificarMotivoResultadoRC(
	ctx context.Context,
	verificador VerificadorPublicacionMotivoFuenteAnalisis,
	identidadPublicador identidadAutoridadFuenteAnalisis,
	resultado ResultadoValidacionRC,
	reloj RelojFuenteAnalisis,
) (*ConfirmacionPublicacionMotivoFuenteAnalisis, error) {
	datos, err := resultado.Datos()
	if err != nil {
		return nil, ErrResultadoFuenteAnalisisNoConfiable
	}
	if datos.Validacion.Resultado == domain.RCValidada {
		if datos.Motivo.datos != nil {
			return nil, ErrResultadoFuenteAnalisisNoConfiable
		}
		return nil, nil
	}
	solicitud := SolicitudVerificarPublicacionMotivoFuenteAnalisis{
		Motivo:                datos.Motivo,
		HuellaRespuestaSHA256: datos.HuellaRespuestaSHA256,
		AutoridadRespuestaRef: datos.Atestacion.Metadatos.AutoridadRef,
		GeneracionRespuesta:   datos.Atestacion.Metadatos.Generacion,
	}
	confirmacion, errVerificacion := verificador.
		VerificarPublicacionMotivoFuenteAnalisis(ctx, solicitud)
	if errContexto := ctx.Err(); errContexto != nil {
		return nil, errorDisponibilidadFuente(
			ErrVerificacionFuenteAnalisisNoDisponible,
			errContexto,
		)
	}
	datosConfirmacion, errDatos := confirmacion.Datos()
	if errVerificacion != nil || errDatos != nil ||
		datosConfirmacion.PublicadorRef != identidadPublicador.autoridadRef ||
		confirmacion.ValidarPara(solicitud, datosConfirmacion.VerificadaEn) != nil {
		return nil, errorDisponibilidadFuente(
			ErrVerificacionFuenteAnalisisNoDisponible,
			errVerificacion,
		)
	}
	comprobadaEn := reloj.Ahora()
	if errContexto := ctx.Err(); errContexto != nil {
		return nil, errorDisponibilidadFuente(
			ErrVerificacionFuenteAnalisisNoDisponible,
			errContexto,
		)
	}
	if confirmacion.ValidarPara(solicitud, comprobadaEn) != nil {
		return nil, ErrResultadoFuenteAnalisisNoConfiable
	}
	return &confirmacion, nil
}

func materializarMotivoValidacionRC(
	validacion domain.ValidacionRC,
	motivo MotivoFuenteAnalisis,
) (domain.ValidacionRC, error) {
	validacion = clonarValidacionRC(validacion)
	if validacion.Motivo != "" {
		return domain.ValidacionRC{}, errMotivoFuenteAnalisisInvalido
	}
	if validacion.Resultado == domain.RCValidada {
		if motivo.datos != nil {
			return domain.ValidacionRC{}, errMotivoFuenteAnalisisInvalido
		}
		return validacion, nil
	}
	datos, err := motivo.Datos()
	if err != nil {
		return domain.ValidacionRC{}, errMotivoFuenteAnalisisInvalido
	}
	validacion.Motivo = string(datos.ClaveMensajeI18N)
	return validacion, nil
}
