package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionVincularEmisionBolsa     = "contratacion_temporal.bolsa.vincular"
	AudienciaVincularEmisionBolsa  = "vec_contratacion_temporal.vinculo_emision_bolsa.v1"
	TipoRecursoVinculoEmisionBolsa = "vinculo_bolsa_expediente"
	FinalidadVinculoEmisionBolsa   = "tramitacion_expediente_contratacion_temporal"
	EsquemaVinculoEmisionBolsa     = "vec.ct.vinculo-emision-bolsa.v1"
)

var (
	ErrVinculoEmisionBolsaInvalido     = errors.New("ct: vínculo de emisión Bolsa inválido")
	ErrVinculoEmisionBolsaNoDisponible = errors.New("ct: vínculo de emisión Bolsa no disponible")
	ErrVinculoEmisionBolsaConflicto    = errors.New("ct: vínculo de emisión Bolsa en conflicto")
)

type SolicitudVinculoEmisionBolsa struct {
	OrganizacionRef   string
	ExpedienteRef     string
	VersionEsperada   uint64
	BolsaRef          string
	LlamamientoRef    string
	ReciboEmisionRef  string
	ClaveIdempotencia string
}

func (s SolicitudVinculoEmisionBolsa) Validar() error {
	if !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(s.BolsaRef) ||
		!domain.ReferenciaOpacaValida(s.LlamamientoRef) ||
		!domain.ReferenciaOpacaValida(s.ReciboEmisionRef) ||
		s.VersionEsperada == 0 || s.VersionEsperada > 9_007_199_254_740_991 ||
		!ClaveIdempotenciaValida(s.ClaveIdempotencia) {
		return ErrVinculoEmisionBolsaInvalido
	}
	return nil
}

// MaterialVinculoEmisionBolsa conserva el orden exacto del canon consumido
// por CT201. No contiene actor ni perfil aportados por el navegador.
type MaterialVinculoEmisionBolsa struct {
	Esquema           string `json:"esquema"`
	OrganizacionRef   string `json:"organizacion_ref"`
	ExpedienteRef     string `json:"expediente_ref"`
	VersionEsperada   uint64 `json:"version_esperada"`
	BolsaRef          string `json:"bolsa_ref"`
	LlamamientoRef    string `json:"llamamiento_ref"`
	ReciboEmisionRef  string `json:"recibo_emision_ref"`
	ClaveIdempotencia string `json:"clave_idempotencia"`
}

func NuevoMaterialVinculoEmisionBolsa(s SolicitudVinculoEmisionBolsa) (MaterialVinculoEmisionBolsa, []byte, string, error) {
	if s.Validar() != nil {
		return MaterialVinculoEmisionBolsa{}, nil, "", ErrVinculoEmisionBolsaInvalido
	}
	m := MaterialVinculoEmisionBolsa{Esquema: EsquemaVinculoEmisionBolsa,
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, VersionEsperada: s.VersionEsperada,
		BolsaRef: s.BolsaRef, LlamamientoRef: s.LlamamientoRef, ReciboEmisionRef: s.ReciboEmisionRef,
		ClaveIdempotencia: s.ClaveIdempotencia}
	b, err := json.Marshal(m)
	if err != nil {
		return MaterialVinculoEmisionBolsa{}, nil, "", ErrVinculoEmisionBolsaInvalido
	}
	h := sha256.Sum256(b)
	return m, b, hex.EncodeToString(h[:]), nil
}

// La autoridad obtiene centro y categoría del expediente ya autorizado.
func NuevoRecursoVinculoEmisionBolsa(s SolicitudVinculoEmisionBolsa, materialSHA256, centroRef, categoriaRef string) (core.RecursoAutorizable, error) {
	if s.Validar() != nil || len(materialSHA256) != 64 ||
		!domain.ReferenciaOpacaValida(centroRef) || !domain.ReferenciaOpacaValida(categoriaRef) {
		return core.RecursoAutorizable{}, ErrVinculoEmisionBolsaInvalido
	}
	r := core.RecursoAutorizable{Referencia: s.ExpedienteRef, ModuloID: "contratacion_temporal",
		Tipo:      TipoRecursoVinculoEmisionBolsa,
		Ambitos:   map[string]string{"organizacion_ref": s.OrganizacionRef, "centro_ref": centroRef, "categoria_ref": categoriaRef},
		Atributos: map[string]string{"material_sha256": materialSHA256}}
	return r, nil
}

type ReciboVinculoEmisionBolsa struct {
	ExpedienteRef    string    `json:"expediente_ref"`
	BolsaRef         string    `json:"bolsa_ref"`
	LlamamientoRef   string    `json:"llamamiento_ref"`
	ReciboEmisionRef string    `json:"recibo_emision_ref"`
	ReciboVinculoRef string    `json:"recibo_vinculo_ref"`
	VinculadoEn      time.Time `json:"vinculado_en"`
	AuditoriaRef     string    `json:"auditoria_ref"`
	EventoRef        string    `json:"evento_ref"`
	Reutilizado      bool      `json:"reutilizado"`
}

func (r ReciboVinculoEmisionBolsa) ValidarPara(s SolicitudVinculoEmisionBolsa) error {
	if s.Validar() != nil || r.ExpedienteRef != s.ExpedienteRef || r.BolsaRef != s.BolsaRef ||
		r.LlamamientoRef != s.LlamamientoRef || r.ReciboEmisionRef != s.ReciboEmisionRef ||
		!domain.ReferenciaOpacaValida(r.ReciboVinculoRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.EventoRef) ||
		!domain.InstanteUTCCanonico(r.VinculadoEn) {
		return ErrVinculoEmisionBolsaNoDisponible
	}
	return nil
}

type AutorizadorVinculoEmisionBolsa interface {
	AutorizarVinculoEmisionBolsa(context.Context, SolicitudVinculoEmisionBolsa, string) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type RepositorioVinculoEmisionBolsa interface {
	RegistrarVinculoEmisionBolsa(context.Context, []byte, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboVinculoEmisionBolsa, error)
}

// LectorAmbitosVinculoEmisionBolsa obtiene sólo centro y categoría de la
// versión actual, después de comprobar organización y bolsa de cobertura.
type LectorAmbitosVinculoEmisionBolsa interface {
	LeerAmbitosVinculoEmisionBolsa(context.Context, SolicitudVinculoEmisionBolsa) (centroRef, categoriaRef string, err error)
}
