package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var _ ports.RepositorioCustodiaFirmado = (*Repositorio)(nil)

// ConfirmarCustodiaFirmado llama a vec_documentos.custodiar_firmado_v1
// (Documentos 000009), que consume la V3 documentos.firmado.custodiar en la
// misma transacción SERIALIZABLE que documento, firmado, auditoría y outbox.
// Una transacción por custodia: el disparador diferido lee el expediente de
// cada fila, pero así el fallo de una no arrastra a otra.
func (r *Repositorio) ConfirmarCustodiaFirmado(ctx context.Context, c ports.CustodiaFirmadoPersistente) (domain.Documento, error) {
	preimagen, err := c.PreimagenCustodia()
	if err != nil || c.Objeto.Validar() != nil || c.Objeto.Objeto.HuellaSHA256 != c.HuellaSHA256 ||
		c.Objeto.Objeto.MIME != ports.MIMEDocumentoFirmado || c.Objeto.Objeto.Tamano != c.Tamano ||
		c.Objeto.Evidencia.Objeto != c.Objeto.Objeto.Objeto || c.Autorizacion.RecursoRef != c.ID ||
		c.Autorizacion.AmbitoRef != c.ExpedienteRef || !retencionCoherenteCon(c.Politica, c.Objeto) {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	material, err := validarAutorizacion(c.Autorizacion, ports.AccionCustodiarFirmado, preimagen)
	if err != nil {
		return domain.Documento{}, err
	}
	auth, _ := autorizacionJSON(c.Autorizacion)
	objeto, err := objetoJSON(c.Objeto, ports.MIMEDocumentoFirmado, c.Tamano, c.HuellaSHA256)
	if err != nil {
		return domain.Documento{}, ErrRepositorioNoDisponible
	}
	raw, err := r.transaccion(ctx,
		"SELECT vec_documentos.custodiar_firmado_v1($1,$2::jsonb,$3::jsonb,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)",
		preimagen, string(objeto), string(auth), material.capacidad, material.decision, material.motivo,
		material.contexto, material.persona, material.perfil, material.payload, material.sobre,
		material.evidencia, material.raiz)
	if err != nil {
		return domain.Documento{}, err
	}
	return decodificarDocumento(raw)
}

// objetoJSON es el recibo del conector que cotejan las fachadas SQL.
func objetoJSON(o vecports.ResultadoOperacionObjeto, mime string, tamano int64, huella string) ([]byte, error) {
	recibo, err := json.Marshal(o.Evidencia)
	if err != nil {
		return nil, err
	}
	suma := sha256.Sum256(recibo)
	return json.Marshal(struct {
		ObjetoRef                string     `json:"objeto_ref"`
		ObjetoVersion            string     `json:"objeto_version"`
		ConectorRef              string     `json:"conector_ref"`
		ReciboObjetoRef          string     `json:"recibo_objeto_ref"`
		ReciboObjetoHuellaSHA256 string     `json:"recibo_objeto_huella_sha256"`
		RetenidoHasta            *time.Time `json:"retenido_hasta"`
		Inmovilizado             bool       `json:"inmovilizado"`
		MIME                     string     `json:"mime"`
		Tamano                   int64      `json:"tamano"`
		HuellaSHA256             string     `json:"huella_sha256"`
	}{o.Objeto.Objeto.Referencia, o.Objeto.Objeto.Version, o.Objeto.ConectorID, o.Evidencia.OperacionRef,
		hex.EncodeToString(suma[:]), retenidoHasta(o.Objeto.RetenidoHasta), o.Objeto.Inmovilizado, mime, tamano, huella})
}

// retencionCoherenteCon aplica la misma regla que retencionCoherente.
func retencionCoherenteCon(p vecports.ResultadoPoliticaConservacionDocumental, o vecports.ResultadoOperacionObjeto) bool {
	pol := p.Politica()
	obj := o.Objeto
	if pol.Provisional() {
		return obj.RetenidoHasta.IsZero() && !obj.Inmovilizado && pol.Proteccion() == "conservacion"
	}
	return !obj.RetenidoHasta.IsZero() && !obj.RetenidoHasta.Before(pol.ConservacionHasta()) &&
		(pol.Proteccion() != "bloqueo" || obj.Inmovilizado)
}
