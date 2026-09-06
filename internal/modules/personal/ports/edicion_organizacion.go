package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
	"unicode"

	core "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrSolicitudCambioOrganizacionInvalida    = errors.New("personal: solicitud de cambio de organizacion invalida")
	ErrCambioOrganizacionNoDisponible         = errors.New("personal: cambio de organizacion no disponible")
	ErrCambioOrganizacionDenegado             = errors.New("personal: cambio de organizacion denegado")
	ErrClaveCambioOrganizacionUsada           = errors.New("personal: clave de cambio de organizacion usada")
	ErrRevisionCambioOrganizacionEnConflicto  = errors.New("personal: revision de organizacion en conflicto")
	ErrResultadoCambioOrganizacionNoConfiable = errors.New("personal: recibo de cambio de organizacion no confiable")
)

const (
	AccionCambioOrganizacion           = "personal.organizacion.actualizar"
	TipoCambioOrganizacion             = "estructura_organizativa"
	AudienciaCambioOrganizacion        = "vec_contratacion_temporal.confirmar_alta_atestada.v1"
	IDCatalogoOrganizacion             = "estructura-organizativa-dipgra"
	MaximoBytesCatalogoOrganizacion    = 1 << 20
	MaximoReferenciaReciboOrganizacion = 512
)

type UnidadCambioOrganizacion struct {
	Clave            string `json:"clave"`
	Etiqueta         string `json:"etiqueta"`
	Tipo             string `json:"tipo"`
	AdscripcionClave string `json:"adscripcion_clave"`
}

type SolicitudCambioOrganizacion struct {
	CatalogoVersion   int                      `json:"catalogo_version"`
	CatalogoRevision  int                      `json:"catalogo_revision"`
	HuellaEsperada    string                   `json:"huella_esperada"`
	ClaveIdempotencia string                   `json:"clave_idempotencia"`
	Unidad            UnidadCambioOrganizacion `json:"unidad"`
	Motivo            string                   `json:"motivo"`
}

func (s SolicitudCambioOrganizacion) Validar() error {
	texto := func(v string, max int, obligatorio bool) bool {
		if obligatorio && (v == "" || strings.TrimSpace(v) != v) || len(v) > max {
			return false
		}
		for _, r := range v {
			if unicode.IsControl(r) {
				return false
			}
		}
		return true
	}
	if s.CatalogoVersion < 1 || s.CatalogoRevision < 1 || !sha256Hex(s.HuellaEsperada) ||
		!uuidV4Canonica(s.ClaveIdempotencia) ||
		!texto(s.Unidad.Clave, 128, true) || !texto(s.Unidad.Etiqueta, 256, true) ||
		!texto(s.Unidad.Tipo, 64, true) || !texto(s.Unidad.AdscripcionClave, 128, false) ||
		!texto(s.Motivo, 4096, true) {
		return ErrSolicitudCambioOrganizacionInvalida
	}
	return nil
}

type MaterialCambioOrganizacion struct {
	Solicitud        SolicitudCambioOrganizacion `json:"solicitud"`
	ActorID          string                      `json:"actor_id"`
	CatalogoCanonico string                      `json:"catalogo_canonico"`
}

func (m MaterialCambioOrganizacion) Catalogo() (core.CatalogoConfigurable, error) {
	if m.CatalogoCanonico == "" || len(m.CatalogoCanonico) > MaximoBytesCatalogoOrganizacion {
		return core.CatalogoConfigurable{}, ErrSolicitudCambioOrganizacionInvalida
	}
	var c core.CatalogoConfigurable
	dec := json.NewDecoder(strings.NewReader(m.CatalogoCanonico))
	dec.DisallowUnknownFields()
	if dec.Decode(&c) != nil {
		return core.CatalogoConfigurable{}, ErrSolicitudCambioOrganizacionInvalida
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return core.CatalogoConfigurable{}, ErrSolicitudCambioOrganizacionInvalida
	}
	canon, err := c.ClonarCanonico()
	if err != nil {
		return core.CatalogoConfigurable{}, ErrSolicitudCambioOrganizacionInvalida
	}
	b, err := json.Marshal(canon)
	if err != nil || !bytes.Equal(b, []byte(m.CatalogoCanonico)) {
		return core.CatalogoConfigurable{}, ErrSolicitudCambioOrganizacionInvalida
	}
	return canon, nil
}

func (m MaterialCambioOrganizacion) Validar() error {
	if m.Solicitud.Validar() != nil || m.ActorID == "" || strings.TrimSpace(m.ActorID) != m.ActorID ||
		strings.IndexFunc(m.ActorID, unicode.IsControl) >= 0 {
		return ErrSolicitudCambioOrganizacionInvalida
	}
	catalogo, err := m.Catalogo()
	if err != nil || catalogo.ID != IDCatalogoOrganizacion || catalogo.Version != m.Solicitud.CatalogoVersion ||
		m.Solicitud.CatalogoRevision == int(^uint(0)>>1) || catalogo.Revision != m.Solicitud.CatalogoRevision+1 ||
		catalogo.Estado != core.EstadoCatalogoBorrador || catalogo.UltimaModificacionPor != m.ActorID ||
		catalogo.MotivoModificacion != m.Solicitud.Motivo {
		return ErrSolicitudCambioOrganizacionInvalida
	}
	clave := m.Solicitud.Unidad.Clave
	for _, entrada := range catalogo.Entradas {
		if entrada.Clave == clave {
			if entrada.Etiqueta != m.Solicitud.Unidad.Etiqueta || entrada.Atributos["tipo"] != m.Solicitud.Unidad.Tipo || entrada.Atributos["adscripcion_clave"] != m.Solicitud.Unidad.AdscripcionClave || entrada.Atributos["modificada_localmente"] != "si" {
				return ErrSolicitudCambioOrganizacionInvalida
			}
			return nil
		}
	}
	return ErrSolicitudCambioOrganizacionInvalida
}

type ReciboCambioOrganizacion struct {
	ReciboRef        string    `json:"recibo_ref"`
	RegistradoEn     time.Time `json:"registrado_en"`
	CatalogoVersion  int       `json:"catalogo_version"`
	CatalogoRevision int       `json:"catalogo_revision"`
	HuellaAnterior   string    `json:"huella_anterior"`
	HuellaPosterior  string    `json:"huella_posterior"`
	EstadoLocal      string    `json:"estado_local"`
}

func (r ReciboCambioOrganizacion) ValidarPara(m MaterialCambioOrganizacion) error {
	c, err := m.Catalogo()
	if err != nil || !referenciaReciboValida(r.ReciboRef) || r.RegistradoEn.IsZero() || r.RegistradoEn.Location() != time.UTC || r.RegistradoEn != r.RegistradoEn.Truncate(time.Microsecond) || r.RegistradoEn.Before(c.UltimaModificacionEn.UTC()) ||
		r.CatalogoVersion != c.Version || r.CatalogoRevision != c.Revision ||
		!sha256Hex(r.HuellaAnterior) || !sha256Hex(r.HuellaPosterior) ||
		(r.EstadoLocal != "registrado" && r.EstadoLocal != "replay_confirmado") {
		return ErrResultadoCambioOrganizacionNoConfiable
	}
	if r.HuellaAnterior != m.Solicitud.HuellaEsperada {
		return ErrResultadoCambioOrganizacionNoConfiable
	}
	posterior, err := c.HuellaSHA256()
	if err != nil || r.HuellaPosterior != posterior {
		return ErrResultadoCambioOrganizacionNoConfiable
	}
	return nil
}

type RepositorioCambiosOrganizacion interface {
	GuardarCambio(context.Context, SolicitudCambioOrganizacion) (ReciboCambioOrganizacion, error)
}

func sha256Hex(v string) bool {
	if len(v) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil && strings.ToLower(v) == v
}

func uuidV4Canonica(v string) bool {
	if len(v) != 36 || v[8] != '-' || v[13] != '-' || v[18] != '-' || v[23] != '-' ||
		v[14] != '4' || !strings.ContainsRune("89ab", rune(v[19])) {
		return false
	}
	for i, r := range v {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return v != "00000000-0000-4000-8000-000000000000"
}

func referenciaReciboValida(v string) bool {
	if v == "" || len(v) > MaximoReferenciaReciboOrganizacion || strings.TrimSpace(v) != v {
		return false
	}
	return !strings.ContainsRune(v, '\x00') && strings.IndexFunc(v, unicode.IsControl) < 0
}
