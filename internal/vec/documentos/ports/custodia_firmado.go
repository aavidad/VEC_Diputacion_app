package ports

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Custodia del documento firmado de un expediente (5.06). Un módulo productor
// (Contratación temporal) entrega el PDF ya firmado y verificado; Documentos lo
// custodia con custodia VEC y consume en SQL la autorización V3 de esta acción
// propia (AD3-113, Documentos 000009) en la misma transacción que la fila del
// documento, su fila de firmado, la auditoría y el outbox. La firma es de
// prueba (GrxFirma): el documento sigue con el proveedor corporativo pendiente.
const (
	AccionCustodiarFirmado    = "documentos.firmado.custodiar"
	FinalidadCustodiarFirmado = "custodiar_documento_firmado"
	// MIMEDocumentoFirmado es el único formato custodiado por esta ruta.
	MIMEDocumentoFirmado = "application/pdf"
)

// CustodiaFirmado es la orden del módulo productor. FirmaOperacionRef es la
// referencia opaca de su operación de firma y HuellaOriginalSHA256 la del
// borrador que se firmó; Documentos no lee tablas del productor. No lleva
// autorización: la preimagen incluye la conservación que resuelve la política
// en ese instante, así que la V3 se pide después (AutorizadorCustodiaFirmado).
type CustodiaFirmado struct {
	ID, ClaveIdempotencia, ModuloID, ExpedienteRef, TipoRef string
	Version                                                 uint64
	Contenido                                               []byte
	HuellaOriginalSHA256                                    string
	FirmaOperacionRef                                       string
	SolicitudPolitica                                       vecports.SolicitudPoliticaConservacionDocumental
}

// AutorizadorCustodiaFirmado obtiene la V3 de documentos.firmado.custodiar
// ligada a la preimagen exacta (CustodiaFirmadoPersistente.PreimagenCustodia),
// al documento y al expediente. La identidad procede del canal autenticado de
// la petición, nunca de estos argumentos. Cada llamada debe dar una decisión
// nueva: una recuperación tras un fallo siempre pide otra. Un error envuelto
// en ErrCapacidadNoDisponible es dependencia caída; cualquier otro, denegación.
type AutorizadorCustodiaFirmado interface {
	AutorizarCustodiaFirmado(ctx context.Context, preimagen []byte, documentoID, expedienteRef string) (AutorizacionV3, error)
}

// CustodiaFirmadoPersistente se confirma tras verificar el recibo del almacén.
type CustodiaFirmadoPersistente struct {
	ID, ClaveIdempotencia, ModuloID, ExpedienteRef, TipoRef string
	Version                                                 uint64
	HuellaSHA256                                            string
	Tamano                                                  int64
	HuellaOriginalSHA256                                    string
	FirmaOperacionRef                                       string
	Objeto                                                  vecports.ResultadoOperacionObjeto
	Politica                                                vecports.ResultadoPoliticaConservacionDocumental
	Autorizacion                                            AutorizacionV3
}

// PreimagenCustodia fija los campos que autoriza V3 y coteja
// vec_documentos.custodiar_firmado_v1 (claves exactas).
func (c CustodiaFirmadoPersistente) PreimagenCustodia() ([]byte, error) {
	if c.Politica.Validar() != nil || !domain.HuellaValida(c.HuellaSHA256) ||
		!domain.HuellaValida(c.HuellaOriginalSHA256) || c.HuellaOriginalSHA256 == c.HuellaSHA256 ||
		!domain.ReferenciaOpacaValida(c.ID) || !domain.ReferenciaOpacaValida(c.ClaveIdempotencia) ||
		!domain.IdentificadorTecnicoValido(c.ModuloID) || !domain.ReferenciaExpedienteModuloValida(c.ModuloID, c.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(c.TipoRef) || !domain.ReferenciaOpacaValida(c.FirmaOperacionRef) ||
		c.Version == 0 || c.Tamano < 1 {
		return nil, ErrSolicitudInvalida
	}
	p := c.Politica.Politica()
	s := p.Solicitud()
	return json.Marshal(struct {
		Accion            string `json:"accion"`
		ID                string `json:"id"`
		Clave             string `json:"clave_idempotencia"`
		Modulo            string `json:"modulo_id"`
		Expediente        string `json:"expediente_ref"`
		Tipo              string `json:"tipo_ref"`
		Version           uint64 `json:"version"`
		MIME              string `json:"mime"`
		Tamano            int64  `json:"tamano"`
		Huella            string `json:"huella_sha256"`
		HuellaOriginal    string `json:"huella_original_sha256"`
		FirmaOperacion    string `json:"firma_operacion_ref"`
		Politica          string `json:"politica_ref"`
		VersionPolitica   uint64 `json:"version_politica"`
		HuellaPolitica    string `json:"huella_politica_sha256"`
		Proteccion        string `json:"proteccion"`
		ConservacionHasta string `json:"conservacion_hasta"`
		EstadoPolitica    string `json:"estado_politica"`
	}{AccionCustodiarFirmado, c.ID, c.ClaveIdempotencia, c.ModuloID, c.ExpedienteRef, c.TipoRef, c.Version,
		MIMEDocumentoFirmado, c.Tamano, c.HuellaSHA256, c.HuellaOriginalSHA256, c.FirmaOperacionRef,
		s.PoliticaRef(), s.VersionPolitica(), hex.EncodeToString(s.HuellaPoliticaSHA256()),
		string(p.Proteccion()), p.ConservacionHasta().UTC().Format(time.RFC3339Nano), EstadoPolitica(p)})
}

// RepositorioCustodiaFirmado confirma la custodia consumiendo la V3 en SQL.
type RepositorioCustodiaFirmado interface {
	ConfirmarCustodiaFirmado(context.Context, CustodiaFirmadoPersistente) (domain.Documento, error)
}

// FabricaContextoCustodia obtiene la escritura en el almacén con una
// concesión V3 registrada de documentos.firmado.custodiar ligada a la
// decisión que consumirá el SQL (no la consume: esa la consume ConfirmarCustodia).
type FabricaContextoCustodia interface {
	ContextoCustodiaFirmado(context.Context, CustodiaFirmadoPersistente) (vecports.ContextoOperacionAlmacen, error)
}

// ReservaTiposFirmado indica los tipos documentales reservados a la custodia
// de firmados: el alta genérica no los admite.
type ReservaTiposFirmado interface {
	CustodiaFirmadoReservada(tipoRef string) bool
}
