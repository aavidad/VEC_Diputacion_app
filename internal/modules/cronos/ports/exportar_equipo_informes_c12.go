package ports

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrInformeC12Invalido = errors.New("cronos: informe inválido")

type AlcanceInformeC12 string

const (
	InformeC12Empleado AlcanceInformeC12 = "empleado"
	InformeC12Jefatura AlcanceInformeC12 = "jefatura"
	InformeC12RRHH     AlcanceInformeC12 = "rrhh"
)

type PasoInformeC12 string

const (
	PasoPrepararInformeC12  PasoInformeC12 = "preparar"
	PasoConfirmarInformeC12 PasoInformeC12 = "confirmar"
)

type ProveedorInformeC12 interface {
	ProveerMaterialInformeC12(context.Context, domain.MaterialExportacionCronos, PasoInformeC12) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenInformeC12 struct {
	actor     vecdomain.ContextoActor
	alcance   AlcanceInformeC12
	proveedor ProveedorInformeC12
}

func NuevaOrdenInformeC12(actor vecdomain.ContextoActor, alcance AlcanceInformeC12, p ProveedorInformeC12) (OrdenInformeC12, error) {
	if actor.Validar() != nil || dependenciaRemotaNula(p) || (alcance != InformeC12Empleado && alcance != InformeC12Jefatura && alcance != InformeC12RRHH) {
		return OrdenInformeC12{}, ErrDependenciaNoDisponible
	}
	copia, err := actor.Clonar()
	if err != nil {
		return OrdenInformeC12{}, ErrDependenciaNoDisponible
	}
	return OrdenInformeC12{actor: copia, alcance: alcance, proveedor: p}, nil
}
func (o OrdenInformeC12) ContextoActor() (vecdomain.ContextoActor, error) { return o.actor.Clonar() }
func (o OrdenInformeC12) Alcance() AlcanceInformeC12                      { return o.alcance }
func (o OrdenInformeC12) Proveedor() ProveedorInformeC12 {
	if dependenciaRemotaNula(o.proveedor) {
		return nil
	}
	return o.proveedor
}

type PoliticaInformeC12 struct {
	Referencia string `json:"politica_ref"`
	Version    int64  `json:"politica_version"`
	SHA256     string `json:"politica_sha256"`
}
type PreparacionInformeC12 struct {
	PoliticaRef     string                      `json:"politica_ref"`
	PoliticaVersion int64                       `json:"politica_version"`
	PoliticaSHA256  string                      `json:"politica_sha256"`
	Tipo            domain.TipoInformeCronos    `json:"tipo_informe"`
	Formato         domain.FormatoInformeCronos `json:"formato"`
	EmpleadoRef     string                      `json:"empleado_ref"`
	EquipoRef       string                      `json:"equipo_ref"`
	Desde           string                      `json:"desde"`
	Hasta           string                      `json:"hasta"`
	Idioma          string                      `json:"idioma"`
	Campos          []string                    `json:"campos"`
	Datos           [][]json.RawMessage         `json:"datos"`
	FuenteSHA256    string                      `json:"fuente_sha256"`
	PreparacionRef  string                      `json:"preparacion_ref"`
}
type ReciboInformeC12 struct {
	ReciboRef       string    `json:"recibo_ref"`
	AuditoriaRef    string    `json:"auditoria_ref"`
	PreparacionRef  string    `json:"preparacion_ref"`
	DocumentoSHA256 string    `json:"documento_sha256"`
	Tamano          int64     `json:"tamano"`
	ConfirmadaUTC   time.Time `json:"confirmada_utc"`
}
type ResultadoInformeC12 struct {
	Contenido []byte
	Formato   domain.FormatoInformeCronos
	Recibo    ReciboInformeC12
}
type PeticionInformeC12 struct {
	EquipoRef                    string
	Tipo                         domain.TipoInformeCronos
	Formato                      domain.FormatoInformeCronos
	Desde, Hasta, ClaveOperacion string
	Idioma                       string
}
type RepositorioInformeC12 interface {
	PoliticaInforme(context.Context, string, domain.TipoInformeCronos, domain.FormatoInformeCronos) (PoliticaInformeC12, error)
	FotoEquipo(context.Context, string) (domain.FotoEquipoAutoritativa, error)
	PrepararInforme(context.Context, OrdenInformeC12, domain.MaterialExportacionCronos) (PreparacionInformeC12, error)
	ConfirmarInforme(context.Context, OrdenInformeC12, domain.MaterialExportacionCronos, string, int64) (ReciboInformeC12, error)
}
type RenderizadorInformeC12 interface {
	RenderizarInforme(context.Context, domain.FormatoInformeCronos, PreparacionInformeC12) ([]byte, error)
}
type CasoUsoInformeC12 interface {
	ExportarInforme(context.Context, OrdenInformeC12, PeticionInformeC12) (ResultadoInformeC12, error)
}
