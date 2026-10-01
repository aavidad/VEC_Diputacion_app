// Package simulaciondevengo ensaya datos importados sin efectos duraderos.
package simulaciondevengo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

const Esquema = "vec_dietas_simulacion_v1"

var ErrEntrada = errors.New("entrada_invalida")
var ErrCatalogo = errors.New("catalogo_discordante")
var ErrVigencia = errors.New("vigencia_invalida")
var ErrCalculo = errors.New("calculo_no_disponible")
var ErrSerializacion = errors.New("serializacion_invalida")

type Civil struct {
	Fecha string `json:"fecha"`
	Hora  string `json:"hora"`
}
type Seleccion struct {
	Grupo                      int    `json:"grupo"`
	VersionTarifaRef           string `json:"version_tarifa_ref"`
	ReglaRef                   string `json:"regla_ref"`
	ReglaHuellaDeclaradaSHA256 string `json:"regla_huella_declarada_sha256"`
}
type Tarifa struct {
	VersionRef              string `json:"version_ref"`
	Rotulo                  string `json:"rotulo"`
	PaisISO2                string `json:"pais_iso2"`
	Grupo                   int    `json:"grupo"`
	VigenteDesde            string `json:"vigente_desde"`
	VigenteHasta            string `json:"vigente_hasta"`
	ManutencionCentimos     int64  `json:"manutencion_centimos"`
	AlojamientoTopeCentimos int64  `json:"alojamiento_tope_centimos"`
}

func (t Tarifa) dominio() domain.TarifaNacionalProvisional {
	return domain.TarifaNacionalProvisional{VersionRef: t.VersionRef, Rotulo: t.Rotulo, PaisISO2: t.PaisISO2, Grupo: t.Grupo, VigenteDesde: t.VigenteDesde, VigenteHasta: t.VigenteHasta, ManutencionCentimos: t.ManutencionCentimos, AlojamientoTopeCentimos: t.AlojamientoTopeCentimos}
}

type Entrada struct {
	Esquema   string                         `json:"esquema"`
	Inicio    Civil                          `json:"inicio"`
	Fin       Civil                          `json:"fin"`
	Seleccion Seleccion                      `json:"seleccion"`
	Catalogo  []Tarifa                       `json:"catalogo"`
	Regla     domain.ReglaDevengoProvisional `json:"regla"`
}
type Resultado struct {
	Tramos                         []domain.TramoDietaProvisional `json:"tramos"`
	ManutencionCentimos            int64                          `json:"manutencion_centimos"`
	AlojamientoTopeCentimos        int64                          `json:"alojamiento_tope_centimos"`
	TotalMaximoOrientativoCentimos int64                          `json:"total_maximo_orientativo_centimos"`
}
type Huellas struct {
	Namespace                     string `json:"namespace"`
	EntradaSHA256                 string `json:"entrada_sha256"`
	ResultadoSHA256               string `json:"resultado_sha256"`
	ConfiguracionSHA256           string `json:"configuracion_sha256"`
	ReglaImportadaDeclaradaSHA256 string `json:"regla_importada_declarada_sha256"`
}
type Salida struct {
	Esquema     string    `json:"esquema"`
	Procedencia string    `json:"procedencia"`
	Liquidable  bool      `json:"liquidable"`
	Entrada     Entrada   `json:"entrada"`
	Resultado   Resultado `json:"resultado"`
	Huellas     Huellas   `json:"huellas"`
}

// Simular no publica versiones ni acredita la procedencia de los datos importados.
func Simular(e Entrada) (*Salida, error) {
	if e.Esquema != Esquema || len(e.Catalogo) < 1 || len(e.Catalogo) > 3 {
		return nil, ErrEntrada
	}
	r, s := e.Regla, e.Seleccion
	if r.Validar() != nil || s.Grupo < 1 || s.Grupo > 3 || s.VersionTarifaRef != r.VersionTarifaRef || s.ReglaRef != r.ReglaRef || s.ReglaHuellaDeclaradaSHA256 != r.HuellaSHA256 {
		return nil, ErrCatalogo
	}
	zona, err := time.LoadLocation(r.Configuracion.Zona)
	if err != nil {
		return nil, ErrEntrada
	}
	inicio, err := domain.ResolverInstanteCivil(e.Inicio.Fecha, e.Inicio.Hora, zona)
	if err != nil {
		return nil, ErrEntrada
	}
	fin, err := domain.ResolverInstanteCivil(e.Fin.Fecha, e.Fin.Hora, zona)
	if err != nil || !fin.After(inicio) {
		return nil, ErrEntrada
	}
	if err := validarVigencia(r.VigenteDesde, r.VigenteHasta, e.Inicio.Fecha, e.Fin.Fecha); err != nil {
		return nil, err
	}
	grupos := map[int]bool{}
	var elegido *domain.CalculoDietasProvisional
	for _, t := range e.Catalogo {
		if grupos[t.Grupo] || t.VersionRef != s.VersionTarifaRef || t.PaisISO2 != r.PaisISO2 {
			return nil, ErrCatalogo
		}
		grupos[t.Grupo] = true
		c, err := domain.CalcularTramosNacionalesProvisionales(inicio, fin, zona, t.dominio(), r)
		if err != nil {
			return nil, ErrCalculo
		}
		if t.Grupo == s.Grupo {
			elegido = &c
		}
	}
	if elegido == nil {
		return nil, ErrCatalogo
	}
	// Copiar el catálogo para que cambios del llamador no alteren la instantánea.
	e.Catalogo = append([]Tarifa(nil), e.Catalogo...)
	resultado := Resultado{elegido.Tramos, elegido.ManutencionCentimos, elegido.AlojamientoTopeCentimos, elegido.TotalMaximoOrientativo}
	entradaSHA256, err := huella(e)
	if err != nil {
		return nil, err
	}
	resultadoSHA256, err := huella(resultado)
	if err != nil {
		return nil, err
	}
	configuracionSHA256, err := huella(r.Configuracion)
	if err != nil {
		return nil, err
	}
	return &Salida{Esquema: Esquema, Procedencia: "propuesta_sin_publicar", Liquidable: false, Entrada: e, Resultado: resultado, Huellas: Huellas{Namespace: Esquema, EntradaSHA256: entradaSHA256, ResultadoSHA256: resultadoSHA256, ConfiguracionSHA256: configuracionSHA256, ReglaImportadaDeclaradaSHA256: r.HuellaSHA256}}, nil
}
func validarVigencia(desde, hasta, inicio, fin string) error {
	d, err := time.Parse("2006-01-02", desde)
	if err != nil || d.Format("2006-01-02") != desde || inicio < desde {
		return ErrVigencia
	}
	if hasta == "" {
		return nil
	}
	h, err := time.Parse("2006-01-02", hasta)
	if err != nil || h.Format("2006-01-02") != hasta || !h.After(d) || fin >= hasta {
		return ErrVigencia
	}
	return nil
}

// huella usa la serialización JSON de los DTO tipados, no jsonb::text de PostgreSQL.
func huella(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", ErrSerializacion
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
