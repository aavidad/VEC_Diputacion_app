package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionReservarOriginalFirmable  = "documentos.original_firmable.reservar"
	AccionConfirmarOriginalFirmable = "documentos.original_firmable.confirmar"
	FinalidadOriginalFirmable       = "custodiar_original_firmable"
)

// OrdenCustodiarOriginalFirmable solo la emite una fuente documental
// autenticada; el servicio calcula la huella y obtiene autorizaciones V3.
type OrdenCustodiarOriginalFirmable struct {
	ID, ClaveIdempotencia, ModuloID, ExpedienteRef, TipoRef string
	Version                                                 uint64
	MIME                                                    string
	Contenido                                               []byte
	SolicitudPolitica                                       vecports.SolicitudPoliticaConservacionDocumental
}

// ReservaOriginalFirmable fija el material exacto antes de escribir en el
// almacén. El repositorio emite una clave distinta por intento pendiente.
type ReservaOriginalFirmable struct {
	ID, ClaveIdempotencia, ModuloID, ExpedienteRef, TipoRef string
	Version                                                 uint64
	MIME, HuellaSHA256                                      string
	Tamano                                                  int64
	Politica                                                vecports.ResultadoPoliticaConservacionDocumental
	Autorizacion                                            AutorizacionV3
}

func (r ReservaOriginalFirmable) Preimagen() ([]byte, error) {
	if r.Politica.Validar() != nil || !domain.ReferenciaOpacaValida(r.ID) ||
		!domain.ReferenciaOpacaValida(r.ClaveIdempotencia) ||
		!domain.IdentificadorTecnicoValido(r.ModuloID) ||
		!domain.ReferenciaOpacaValida(r.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(r.TipoRef) || r.Version == 0 || r.Version > 9007199254740991 ||
		r.MIME != "application/pdf" || !domain.HuellaValida(r.HuellaSHA256) || r.Tamano < 1 || r.Tamano > 1<<20 {
		return nil, ErrSolicitudInvalida
	}
	p := r.Politica.Politica()
	s := p.Solicitud()
	if s.ExpedienteRef() != r.ExpedienteRef || s.TipoDocumentalRef() != r.TipoRef {
		return nil, ErrSolicitudInvalida
	}
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
		Politica          string `json:"politica_ref"`
		VersionPolitica   uint64 `json:"version_politica"`
		HuellaPolitica    string `json:"huella_politica_sha256"`
		Proteccion        string `json:"proteccion"`
		ConservacionHasta string `json:"conservacion_hasta"`
		EstadoPolitica    string `json:"estado_politica"`
	}{AccionReservarOriginalFirmable, r.ID, r.ClaveIdempotencia, r.ModuloID,
		r.ExpedienteRef, r.TipoRef, r.Version, r.MIME, r.Tamano, r.HuellaSHA256,
		s.PoliticaRef(), s.VersionPolitica(), hex.EncodeToString(s.HuellaPoliticaSHA256()),
		string(p.Proteccion()), p.ConservacionHasta().UTC().Format(time.RFC3339Nano), EstadoPolitica(p)})
}

type IntentoOriginalFirmable struct {
	ReservaRef, Estado, DocumentoID, HuellaSHA256, ClaveAlmacenRef string
	Numero                                                         uint64
}

func (i IntentoOriginalFirmable) ValidarContra(r ReservaOriginalFirmable) error {
	if !domain.ReferenciaOpacaValida(i.ReservaRef) || i.DocumentoID != r.ID ||
		i.HuellaSHA256 != r.HuellaSHA256 || !domain.ReferenciaOpacaValida(i.ClaveAlmacenRef) ||
		i.Numero == 0 || i.Numero > 9007199254740991 ||
		(i.Estado != "pendiente" && i.Estado != "confirmado") {
		return ErrCapacidadNoDisponible
	}
	return nil
}

// ObjetoOriginalFirmable incluye la clave del intento junto al recibo de
// AlmacenObjetos. Se construye exclusivamente desde un resultado validado.
type ObjetoOriginalFirmable struct {
	ClaveAlmacenRef          string     `json:"clave_almacen_ref"`
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
}

func NuevoObjetoOriginalFirmable(intento IntentoOriginalFirmable, resultado vecports.ResultadoOperacionObjeto) (ObjetoOriginalFirmable, error) {
	if resultado.Validar() != nil || !domain.ReferenciaOpacaValida(intento.ClaveAlmacenRef) ||
		resultado.Objeto.HuellaSHA256 != intento.HuellaSHA256 {
		return ObjetoOriginalFirmable{}, ErrSolicitudInvalida
	}
	recibo, err := json.Marshal(resultado.Evidencia)
	if err != nil {
		return ObjetoOriginalFirmable{}, ErrSolicitudInvalida
	}
	suma := sha256.Sum256(recibo)
	var retencion *time.Time
	if !resultado.Objeto.RetenidoHasta.IsZero() {
		v := resultado.Objeto.RetenidoHasta.UTC()
		retencion = &v
	}
	return ObjetoOriginalFirmable{
		ClaveAlmacenRef: intento.ClaveAlmacenRef,
		ObjetoRef:       resultado.Objeto.Objeto.Referencia, ObjetoVersion: resultado.Objeto.Objeto.Version,
		ConectorRef: resultado.Objeto.ConectorID, ReciboObjetoRef: resultado.Evidencia.OperacionRef,
		ReciboObjetoHuellaSHA256: hex.EncodeToString(suma[:]), RetenidoHasta: retencion,
		Inmovilizado: resultado.Objeto.Inmovilizado, MIME: resultado.Objeto.MIME,
		Tamano: resultado.Objeto.Tamano, HuellaSHA256: resultado.Objeto.HuellaSHA256,
	}, nil
}

type ConfirmacionOriginalFirmable struct {
	Intento      IntentoOriginalFirmable
	Objeto       ObjetoOriginalFirmable
	Autorizacion AutorizacionV3
}

func (c ConfirmacionOriginalFirmable) Preimagen() ([]byte, error) {
	if !domain.ReferenciaOpacaValida(c.Intento.ReservaRef) || c.Intento.Estado != "pendiente" ||
		!domain.ReferenciaOpacaValida(c.Intento.DocumentoID) || !domain.HuellaValida(c.Intento.HuellaSHA256) ||
		c.Intento.Numero == 0 || !domain.ReferenciaOpacaValida(c.Intento.ClaveAlmacenRef) ||
		c.Objeto.ClaveAlmacenRef != c.Intento.ClaveAlmacenRef ||
		c.Objeto.HuellaSHA256 != c.Intento.HuellaSHA256 ||
		!domain.ReferenciaValida(c.Objeto.ObjetoRef) || !domain.VersionObjetoValida(c.Objeto.ObjetoVersion) ||
		!domain.IdentificadorTecnicoValido(c.Objeto.ConectorRef) ||
		!domain.ReferenciaValida(c.Objeto.ReciboObjetoRef) ||
		!domain.HuellaValida(c.Objeto.ReciboObjetoHuellaSHA256) ||
		c.Objeto.MIME != "application/pdf" || c.Objeto.Tamano < 1 || c.Objeto.Tamano > 1<<20 ||
		(c.Objeto.RetenidoHasta != nil && (c.Objeto.RetenidoHasta.IsZero() || c.Objeto.RetenidoHasta.Location() != time.UTC)) {
		return nil, ErrSolicitudInvalida
	}
	return json.Marshal(struct {
		Accion     string                 `json:"accion"`
		ReservaRef string                 `json:"reserva_ref"`
		IntentoNum uint64                 `json:"intento_num"`
		Clave      string                 `json:"clave_almacen_ref"`
		ID         string                 `json:"id"`
		Huella     string                 `json:"huella_sha256"`
		Objeto     ObjetoOriginalFirmable `json:"objeto"`
	}{AccionConfirmarOriginalFirmable, c.Intento.ReservaRef, c.Intento.Numero,
		c.Intento.ClaveAlmacenRef, c.Intento.DocumentoID, c.Intento.HuellaSHA256, c.Objeto})
}

// ValidarOriginalFirmable admite exclusivamente las dos acciones nuevas.
// El material V3 sigue requiriendo consumo en la fachada SQL correspondiente.
func (a AutorizacionV3) ValidarOriginalFirmable(accion string, ahora time.Time) error {
	if accion != AccionReservarOriginalFirmable && accion != AccionConfirmarOriginalFirmable {
		return ErrSolicitudInvalida
	}
	if a.Material.ValidarEstructura() != nil || a.Accion != accion ||
		a.Finalidad != FinalidadOriginalFirmable || !domain.ReferenciaOpacaValida(a.RecursoRef) ||
		!domain.ReferenciaOpacaValida(a.AmbitoRef) || !domain.ReferenciaValida(a.PrincipalID) ||
		!domain.ReferenciaValida(a.PerfilActivoRef) || !domain.ReferenciaValida(a.CorrelacionRef) ||
		ahora.IsZero() {
		return ErrSolicitudInvalida
	}
	resumen := a.Material.ResumenCapacidad()
	if resumen.ValidarEstructura() != nil || resumen.AudienciaConsumo() != AudienciaV3 ||
		resumen.Operacion() != accion || resumen.EfectoRef() != a.RecursoRef ||
		ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return ErrSolicitudInvalida
	}
	return nil
}

// AutorizarOriginalFirmable obtiene decisiones nominales para cada preimagen.
// El contexto de escritura es otra concesión PDP propia del almacén.
type AutorizarOriginalFirmable interface {
	AutorizarReservaOriginal(context.Context, []byte, string, string) (AutorizacionV3, error)
	AutorizarConfirmacionOriginal(context.Context, []byte, string, string) (AutorizacionV3, error)
	ContextoEscrituraOriginal(context.Context, ReservaOriginalFirmable, IntentoOriginalFirmable) (vecports.ContextoOperacionAlmacen, error)
}

type RepositorioOriginalFirmable interface {
	ReservarOriginalFirmable(context.Context, ReservaOriginalFirmable) (IntentoOriginalFirmable, error)
	ConfirmarOriginalFirmable(context.Context, ConfirmacionOriginalFirmable) (domain.Documento, error)
}
