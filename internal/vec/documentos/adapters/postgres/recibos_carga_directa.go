package postgres

import (
	"context"
	"encoding/json"
	"time"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

// RepositorioRecibosCargasDirectas comparte la misma fachada y transacción D17
// con la carga. El alta y consumo legados permanecen cerrados en producción.
type RepositorioRecibosCargasDirectas struct{ base *Repositorio }

func NuevoRepositorioRecibosCargasDirectas(base *Repositorio) (*RepositorioRecibosCargasDirectas, error) {
	if base == nil || base.db == nil {
		return nil, ErrRepositorioNoDisponible
	}
	return &RepositorioRecibosCargasDirectas{base: base}, nil
}

func (*RepositorioRecibosCargasDirectas) RegistrarReciboCargaDirecta(context.Context, vecports.RegistroReciboCargaDirecta) (vecports.ResultadoRegistroReciboCargaDirecta, error) {
	return vecports.ResultadoRegistroReciboCargaDirecta{}, vecports.ErrReciboCargaDirectaNoDisponible
}
func (*RepositorioRecibosCargasDirectas) ConsumirReciboCargaDirecta(context.Context, vecports.OrdenConsumoReciboCargaDirecta) (vecports.ResultadoConsumoReciboCargaDirecta, error) {
	return vecports.ResultadoConsumoReciboCargaDirecta{}, vecports.ErrReciboCargaDirectaNoDisponible
}
func (*RepositorioRecibosCargasDirectas) ConsultarConsumoReciboCargaDirecta(context.Context, string) (vecports.ConsumoReciboCargaDirectaPersistido, error) {
	return vecports.ConsumoReciboCargaDirectaPersistido{}, vecports.ErrReciboCargaDirectaNoDisponible
}

func (r *RepositorioRecibosCargasDirectas) RegistrarReciboCargaDirectaV3(ctx context.Context, s vecports.RegistroReciboCargaDirectaV3) (vecports.ResultadoRegistroReciboCargaDirecta, error) {
	if r == nil || r.base == nil || s.Validar(time.Now().UTC()) != nil {
		return vecports.ResultadoRegistroReciboCargaDirecta{}, vecports.ErrReciboCargaDirectaNoValido
	}
	registro, err := json.Marshal(struct {
		IndiceHMAC             string    `json:"indice_hmac"`
		GrupoHMAC              string    `json:"grupo_hmac"`
		VinculoHMAC            string    `json:"vinculo_hmac"`
		EvidenciaAltaRef       string    `json:"evidencia_alta_ref"`
		AutorizacionEmisionRef string    `json:"autorizacion_emision_ref"`
		ExpiraEn               time.Time `json:"expira_en"`
	}{s.Registro.IndiceHMAC, s.Registro.GrupoHMAC, s.Registro.VinculoHMAC,
		s.Registro.EvidenciaAltaRef, s.Registro.AutorizacionEmisionRef, s.Registro.ExpiraEn})
	if err != nil {
		return vecports.ResultadoRegistroReciboCargaDirecta{}, vecports.ErrReciboCargaDirectaNoValido
	}
	auth, err := autorizacionCargaJSON(s.AutorizacionV3)
	if err != nil {
		return vecports.ResultadoRegistroReciboCargaDirecta{}, vecports.ErrReciboCargaDirectaNoValido
	}
	m, err := materialCargaV3(s.AutorizacionV3)
	if err != nil {
		return vecports.ResultadoRegistroReciboCargaDirecta{}, err
	}
	raw, err := r.base.transaccion(ctx,
		"SELECT vec_documentos.registrar_recibo_carga_directa_v1($1,$2,$3::jsonb,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)",
		s.Preimagen, registro, string(auth), m.capacidad, m.decision, m.motivo, m.contexto,
		m.persona, m.perfil, m.payload, m.sobre, m.evidencia, m.raiz)
	if err != nil {
		return vecports.ResultadoRegistroReciboCargaDirecta{}, err
	}
	var salida struct {
		IndiceHMAC             string    `json:"indice_hmac"`
		GrupoHMAC              string    `json:"grupo_hmac"`
		AutorizacionEmisionRef string    `json:"autorizacion_emision_ref"`
		RegistradoEn           time.Time `json:"registrado_en"`
		Estado                 string    `json:"estado_operativo"`
		EntregaAcreditada      bool      `json:"entrega_acreditada"`
	}
	if json.Unmarshal(raw, &salida) != nil || salida.Estado != "recibo_registrado" || salida.EntregaAcreditada {
		return vecports.ResultadoRegistroReciboCargaDirecta{}, ErrRepositorioNoDisponible
	}
	resultado := vecports.ResultadoRegistroReciboCargaDirecta{IndiceHMAC: salida.IndiceHMAC,
		GrupoHMAC: salida.GrupoHMAC, AutorizacionEmisionRef: salida.AutorizacionEmisionRef,
		RegistradoEn: salida.RegistradoEn}
	if resultado.ValidarContra(s.Registro) != nil {
		return vecports.ResultadoRegistroReciboCargaDirecta{}, ErrRepositorioNoDisponible
	}
	return resultado, nil
}

func (r *RepositorioRecibosCargasDirectas) ConsumirReciboCargaDirectaV3(
	ctx context.Context, s vecports.ConsumoReciboCargaDirectaV3,
) (vecports.ResultadoConsumoReciboCargaDirecta, error) {
	if r == nil || r.base == nil || s.Validar(time.Now().UTC()) != nil {
		return vecports.ResultadoConsumoReciboCargaDirecta{}, vecports.ErrReciboCargaDirectaNoValido
	}
	orden, err := json.Marshal(struct {
		IndiceHMAC               string    `json:"indice_hmac"`
		GrupoHMAC                string    `json:"grupo_hmac"`
		VinculoHMAC              string    `json:"vinculo_hmac"`
		EvidenciaConsumoRef      string    `json:"evidencia_consumo_ref"`
		IntencionConfirmacionRef string    `json:"intencion_confirmacion_ref"`
		HuellaIntencionHMAC      string    `json:"huella_intencion_hmac"`
		RegistradoEn             time.Time `json:"registrado_en"`
		ValidaHasta              time.Time `json:"valida_hasta"`
	}{s.Orden.IndiceHMAC, s.Orden.GrupoHMAC, s.Orden.VinculoHMAC,
		s.Orden.EvidenciaConsumoRef, s.Orden.IntencionConfirmacionRef,
		s.Orden.HuellaIntencionHMAC, s.Orden.RegistradoEn, s.Orden.ValidaHasta})
	if err != nil {
		return vecports.ResultadoConsumoReciboCargaDirecta{}, vecports.ErrReciboCargaDirectaNoValido
	}
	auth, err := autorizacionCargaJSON(s.AutorizacionV3)
	if err != nil {
		return vecports.ResultadoConsumoReciboCargaDirecta{}, vecports.ErrReciboCargaDirectaNoValido
	}
	m, err := materialCargaV3(s.AutorizacionV3)
	if err != nil {
		return vecports.ResultadoConsumoReciboCargaDirecta{}, err
	}
	raw, err := r.base.transaccion(ctx,
		"SELECT vec_documentos.consumir_recibo_carga_directa_v3_v1($1,$2,$3::jsonb,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)",
		s.Preimagen, orden, string(auth), m.capacidad, m.decision, m.motivo, m.contexto,
		m.persona, m.perfil, m.payload, m.sobre, m.evidencia, m.raiz)
	if err != nil {
		return vecports.ResultadoConsumoReciboCargaDirecta{}, err
	}
	var salida struct {
		IndiceHMAC               string    `json:"indice_hmac"`
		GrupoHMAC                string    `json:"grupo_hmac"`
		VinculoHMAC              string    `json:"vinculo_hmac"`
		EvidenciaConsumoRef      string    `json:"evidencia_consumo_ref"`
		IntencionConfirmacionRef string    `json:"intencion_confirmacion_ref"`
		HuellaIntencionHMAC      string    `json:"huella_intencion_hmac"`
		RegistradoEn             time.Time `json:"registrado_en"`
		ConsumidoEn              time.Time `json:"consumido_en"`
		ExpiraEn                 time.Time `json:"expira_en"`
		ValidaHasta              time.Time `json:"valida_hasta"`
		Estado                   string    `json:"estado_operativo"`
	}
	if json.Unmarshal(raw, &salida) != nil || salida.Estado != "confirmacion_iniciada" ||
		!salida.ValidaHasta.Equal(s.Orden.ValidaHasta) {
		return vecports.ResultadoConsumoReciboCargaDirecta{}, ErrRepositorioNoDisponible
	}
	resultado := vecports.ResultadoConsumoReciboCargaDirecta{
		IndiceHMAC: salida.IndiceHMAC, GrupoHMAC: salida.GrupoHMAC, VinculoHMAC: salida.VinculoHMAC,
		EvidenciaConsumoRef:      salida.EvidenciaConsumoRef,
		IntencionConfirmacionRef: salida.IntencionConfirmacionRef,
		HuellaIntencionHMAC:      salida.HuellaIntencionHMAC,
		RegistradoEn:             salida.RegistradoEn, ConsumidoEn: salida.ConsumidoEn, ExpiraEn: salida.ExpiraEn}
	if resultado.ValidarContra(s.Orden) != nil {
		return vecports.ResultadoConsumoReciboCargaDirecta{}, ErrRepositorioNoDisponible
	}
	return resultado, nil
}

var _ vecports.RepositorioRecibosCargaDirecta = (*RepositorioRecibosCargasDirectas)(nil)
var _ vecports.RepositorioRegistroReciboCargaDirectaV3 = (*RepositorioRecibosCargasDirectas)(nil)
var _ vecports.RepositorioConsultaConsumoReciboCargaDirecta = (*RepositorioRecibosCargasDirectas)(nil)
var _ vecports.RepositorioConsumoReciboCargaDirectaV3 = (*RepositorioRecibosCargasDirectas)(nil)
