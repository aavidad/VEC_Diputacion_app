package ports

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"sort"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var (
	errMotivoFuenteAnalisisInvalido = errors.New(
		"contratacion temporal: motivo catalogado de fuente invalido",
	)
	patronHuellaCatalogoMotivoFuente = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type ClaveParametroMotivoFuenteAnalisis string

const (
	ParametroMotivoResultado ClaveParametroMotivoFuenteAnalisis = "resultado"
	ParametroMotivoCausa     ClaveParametroMotivoFuenteAnalisis = "causa"
	ParametroMotivoRegla     ClaveParametroMotivoFuenteAnalisis = "regla"
)

type ParametroMotivoFuenteAnalisis struct {
	Clave ClaveParametroMotivoFuenteAnalisis
	Valor domain.ClaveCatalogo
}

type datosMotivoFuenteAnalisis struct {
	CatalogoRef      string
	CatalogoVersion  uint64
	CatalogoHuella   string
	EntradaClave     domain.ClaveCatalogo
	ClaveMensajeI18N domain.ClaveCatalogo
	Parametros       []ParametroMotivoFuenteAnalisis
}

type MotivoFuenteAnalisis struct {
	datos *datosMotivoFuenteAnalisis
}

func NuevoMotivoFuenteAnalisis(
	catalogoRef string,
	catalogoVersion uint64,
	catalogoHuella string,
	entradaClave domain.ClaveCatalogo,
	claveMensajeI18N domain.ClaveCatalogo,
	parametros []ParametroMotivoFuenteAnalisis,
) (MotivoFuenteAnalisis, error) {
	motivo := MotivoFuenteAnalisis{datos: &datosMotivoFuenteAnalisis{
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
		!patronHuellaCatalogoMotivoFuente.MatchString(m.datos.CatalogoHuella) ||
		m.datos.CatalogoHuella == strings.Repeat("0", 64) ||
		!m.datos.EntradaClave.Valida() || !m.datos.ClaveMensajeI18N.Valida() ||
		len(m.datos.Parametros) > 3 ||
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
			!parametroMotivoFuenteAnalisisPermitido(parametro) {
			return false
		}
	}
	return true
}

func parametroMotivoFuenteAnalisisPermitido(
	parametro ParametroMotivoFuenteAnalisis,
) bool {
	permitidos := map[ClaveParametroMotivoFuenteAnalisis]map[domain.ClaveCatalogo]struct{}{
		ParametroMotivoResultado: {
			"no_requerida": {}, "rechazada": {},
		},
		ParametroMotivoCausa: {
			"no_consta_rc": {}, "rc_insuficiente": {},
			"rc_incoherente": {}, "regla_presupuestaria": {},
		},
		ParametroMotivoRegla: {
			"rc_exigida": {}, "rc_no_requerida": {}, "importe_suficiente": {},
		},
	}
	valores, existe := permitidos[parametro.Clave]
	if !existe {
		return false
	}
	_, existe = valores[parametro.Valor]
	return existe
}

func (m MotivoFuenteAnalisis) Datos() (
	string,
	uint64,
	string,
	domain.ClaveCatalogo,
	domain.ClaveCatalogo,
	[]ParametroMotivoFuenteAnalisis,
	error,
) {
	if m.Validar() != nil {
		return "", 0, "", "", "", nil, errMotivoFuenteAnalisisInvalido
	}
	return m.datos.CatalogoRef, m.datos.CatalogoVersion,
		m.datos.CatalogoHuella, m.datos.EntradaClave,
		m.datos.ClaveMensajeI18N,
		append([]ParametroMotivoFuenteAnalisis(nil), m.datos.Parametros...), nil
}

func (m MotivoFuenteAnalisis) clonar() MotivoFuenteAnalisis {
	if m.datos == nil {
		return MotivoFuenteAnalisis{}
	}
	datos := *m.datos
	datos.Parametros = append([]ParametroMotivoFuenteAnalisis(nil), m.datos.Parametros...)
	return MotivoFuenteAnalisis{datos: &datos}
}

func (MotivoFuenteAnalisis) String() string {
	return "[MOTIVO-FUENTE-ANALISIS-CATALOGADO-REDACTADO]"
}

func (m MotivoFuenteAnalisis) GoString() string {
	return m.String()
}

func (m MotivoFuenteAnalisis) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, m.String())
}

func (m MotivoFuenteAnalisis) LogValue() slog.Value {
	return slog.StringValue(m.String())
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
	if motivo.Validar() != nil {
		return domain.ValidacionRC{}, errMotivoFuenteAnalisisInvalido
	}
	_, _, _, _, claveI18N, _, _ := motivo.Datos()
	validacion.Motivo = string(claveI18N)
	return validacion, nil
}
