package almacen

import "time"

// ReferenciaObjetoAlmacen identifica de forma opaca una version inmutable.
type ReferenciaObjetoAlmacen struct {
	Referencia string
	Version    string
}

func (r ReferenciaObjetoAlmacen) Validar() error {
	if !ReferenciaOpacaValida(r.Referencia, 512) || !ReferenciaOpacaValida(r.Version, 256) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

// ObjetoAlmacenado es la proyeccion tecnica verificable de un objeto.
type ObjetoAlmacenado struct {
	Objeto               ReferenciaObjetoAlmacen
	ConectorID           string
	Zona                 Zona
	MIME                 string
	Tamano               int64
	HuellaSHA256         string
	EvidenciaCreacionRef string
	AlmacenadoEn         time.Time
	RetenidoHasta        time.Time
	Inmovilizado         bool
	Eliminado            bool
}

func (o ObjetoAlmacenado) Validar() error {
	if err := o.Objeto.Validar(); err != nil {
		return err
	}
	if !ReferenciaOpacaValida(o.ConectorID, 128) || !o.Zona.Valida() || !TextoSeguro(o.MIME, 255) ||
		o.Tamano < 1 || !SHA256HexadecimalValido(o.HuellaSHA256) ||
		!ReferenciaOpacaValida(o.EvidenciaCreacionRef, 512) || o.AlmacenadoEn.IsZero() ||
		(!o.RetenidoHasta.IsZero() && !o.RetenidoHasta.After(o.AlmacenadoEn)) ||
		(o.Eliminado && o.Inmovilizado) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

// SolicitudSellarIdempotenciaCarga contiene solo los campos exactos que el
// sellador debe ligar mediante una clave exclusiva del servidor.
type SolicitudSellarIdempotenciaCarga struct {
	OperacionRef     string
	PrincipalRef     string
	RecursoRef       string
	MIME             string
	Tamano           int64
	HuellaSHA256     string
	ClaveSolicitante string
}

func (s SolicitudSellarIdempotenciaCarga) Validar() error {
	if !ReferenciaOpacaValida(s.OperacionRef, 512) ||
		!ReferenciaOpacaValida(s.PrincipalRef, 512) ||
		!ReferenciaOpacaValida(s.RecursoRef, 512) || !TextoSeguro(s.MIME, 255) ||
		s.Tamano < 1 || !SHA256HexadecimalValido(s.HuellaSHA256) ||
		!ReferenciaOpacaValida(s.ClaveSolicitante, 512) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}
