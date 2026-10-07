package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	DominioConsultaCuadroRRHHSesionV2 = "vec.contratacion_temporal.consulta_rrhh.cuadro.v2"
	DominioFamiliaCuadroRRHHSesionV2  = "vec.contratacion_temporal.filtros_rrhh.cuadro.v2"
)

// ConsultaCuadroRRHHSesionV2 describe únicamente el corte y la paginación.
// La identidad y el alcance se obtienen del puerto común de sesión al consumirla.
type ConsultaCuadroRRHHSesionV2 struct {
	bloqueoSerializacionConsultaRRHH
	texto        string
	centroRef    string
	categoriaRef string
	estados      [6]domain.EstadoOperativo
	numEstados   uint8
	fases        [32]domain.ClaveFase
	numFases     uint8
	limite       uint16
	cursor       string
}

func NuevaConsultaCuadroRRHHSesionV2(
	texto, centroRef, categoriaRef string,
	estados []domain.EstadoOperativo,
	fases []domain.ClaveFase,
	limite uint16, cursor string,
) (ConsultaCuadroRRHHSesionV2, error) {
	if len(estados) > 6 || len(fases) > 32 {
		return ConsultaCuadroRRHHSesionV2{}, ErrSolicitudConsultaRRHHInvalida
	}
	c := ConsultaCuadroRRHHSesionV2{
		texto: texto, centroRef: centroRef, categoriaRef: categoriaRef,
		numEstados: uint8(len(estados)), numFases: uint8(len(fases)),
		limite: limite, cursor: cursor,
	}
	copy(c.estados[:], estados)
	copy(c.fases[:], fases)
	sort.Slice(c.estados[:c.numEstados], func(i, j int) bool { return c.estados[i] < c.estados[j] })
	sort.Slice(c.fases[:c.numFases], func(i, j int) bool { return c.fases[i] < c.fases[j] })
	if c.validar() != nil {
		return ConsultaCuadroRRHHSesionV2{}, ErrSolicitudConsultaRRHHInvalida
	}
	return c, nil
}

func (c ConsultaCuadroRRHHSesionV2) validar() error {
	if c.texto != strings.TrimSpace(c.texto) || !utf8.ValidString(c.texto) ||
		!patronTextoCuadroRRHH.MatchString(c.texto) ||
		(c.centroRef != "" && !domain.ReferenciaOpacaValida(c.centroRef)) ||
		(c.categoriaRef != "" && !domain.ReferenciaOpacaValida(c.categoriaRef)) ||
		c.numEstados > 6 || c.numFases > 32 ||
		c.limite < 1 || c.limite > LimiteMaximoCuadroRRHH ||
		(c.cursor != "" && !cursorRRHHValido(c.cursor)) {
		return ErrSolicitudConsultaRRHHInvalida
	}
	for i, estado := range c.estados[:c.numEstados] {
		if !estado.Valido() || (i > 0 && c.estados[i-1] >= estado) {
			return ErrSolicitudConsultaRRHHInvalida
		}
	}
	for i, fase := range c.fases[:c.numFases] {
		if !fase.Valida() || (i > 0 && c.fases[i-1] >= fase) {
			return ErrSolicitudConsultaRRHHInvalida
		}
	}
	return nil
}

func (c ConsultaCuadroRRHHSesionV2) Texto() string        { return c.texto }
func (c ConsultaCuadroRRHHSesionV2) CentroRef() string    { return c.centroRef }
func (c ConsultaCuadroRRHHSesionV2) CategoriaRef() string { return c.categoriaRef }
func (c ConsultaCuadroRRHHSesionV2) Limite() uint16       { return c.limite }
func (c ConsultaCuadroRRHHSesionV2) Cursor() string       { return c.cursor }
func (c ConsultaCuadroRRHHSesionV2) EstadosClave() []domain.EstadoOperativo {
	return append([]domain.EstadoOperativo(nil), c.estados[:c.numEstados]...)
}
func (c ConsultaCuadroRRHHSesionV2) FasesClave() []domain.ClaveFase {
	return append([]domain.ClaveFase(nil), c.fases[:c.numFases]...)
}

type canonFiltrosCuadroRRHHSesionV2 struct {
	Dominio      string   `json:"dominio"`
	Version      uint16   `json:"version"`
	Texto        string   `json:"texto"`
	CentroRef    string   `json:"centro_ref"`
	CategoriaRef string   `json:"categoria_ref"`
	EstadosClave []string `json:"estados_clave"`
	FasesClave   []string `json:"fases_clave"`
	Limite       uint16   `json:"limite"`
}

type canonConsultaCuadroRRHHSesionV2 struct {
	Dominio      string   `json:"dominio"`
	Version      uint16   `json:"version"`
	Texto        string   `json:"texto"`
	CentroRef    string   `json:"centro_ref"`
	CategoriaRef string   `json:"categoria_ref"`
	EstadosClave []string `json:"estados_clave"`
	FasesClave   []string `json:"fases_clave"`
	Limite       uint16   `json:"limite"`
	Cursor       string   `json:"cursor"`
}

func (c ConsultaCuadroRRHHSesionV2) canon(familia bool) ([]byte, error) {
	if c.validar() != nil {
		return nil, ErrSolicitudConsultaRRHHInvalida
	}
	dominio := DominioConsultaCuadroRRHHSesionV2
	if familia {
		dominio = DominioFamiliaCuadroRRHHSesionV2
	}
	valor := canonFiltrosCuadroRRHHSesionV2{
		Dominio: dominio, Version: 2, Texto: c.texto,
		CentroRef: c.centroRef, CategoriaRef: c.categoriaRef,
		EstadosClave: make([]string, c.numEstados),
		FasesClave:   make([]string, c.numFases), Limite: c.limite,
	}
	for i, estado := range c.estados[:c.numEstados] {
		valor.EstadosClave[i] = string(estado)
	}
	for i, fase := range c.fases[:c.numFases] {
		valor.FasesClave[i] = string(fase)
	}
	if familia {
		return json.Marshal(valor)
	}
	return json.Marshal(canonConsultaCuadroRRHHSesionV2{
		Dominio: valor.Dominio, Version: valor.Version, Texto: valor.Texto,
		CentroRef: valor.CentroRef, CategoriaRef: valor.CategoriaRef,
		EstadosClave: valor.EstadosClave, FasesClave: valor.FasesClave,
		Limite: valor.Limite, Cursor: c.cursor,
	})
}

func (c ConsultaCuadroRRHHSesionV2) CanonConsulta() ([]byte, error) {
	return c.canon(false)
}
func (c ConsultaCuadroRRHHSesionV2) CanonFamilia() ([]byte, error) {
	return c.canon(true)
}
func (c ConsultaCuadroRRHHSesionV2) HuellaConsulta() (string, error) {
	canon, err := c.CanonConsulta()
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(canon)
	return hex.EncodeToString(suma[:]), nil
}
func (c ConsultaCuadroRRHHSesionV2) HuellaFamilia() (string, error) {
	canon, err := c.CanonFamilia()
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(canon)
	return hex.EncodeToString(suma[:]), nil
}

func (ConsultaCuadroRRHHSesionV2) String() string   { return "[consulta-cuadro-rrhh-sesion-redactada]" }
func (ConsultaCuadroRRHHSesionV2) GoString() string { return "[consulta-cuadro-rrhh-sesion-redactada]" }
func (c ConsultaCuadroRRHHSesionV2) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (c ConsultaCuadroRRHHSesionV2) LogValue() slog.Value {
	return slog.StringValue(c.String())
}
func (ConsultaCuadroRRHHSesionV2) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}
