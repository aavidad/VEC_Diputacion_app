package application

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

var (
	ErrConsultaDocumentosBolsaInvalida     = errors.New("bolsa documentos: consulta invalida")
	ErrConsultaDocumentosBolsaNoDisponible = errors.New("bolsa documentos: consulta no disponible")
	ErrDocumentoBolsaNoEncontrado          = errors.New("bolsa documentos: documento no encontrado")
)

// DocumentoBolsaRegistral proyecta exclusivamente los datos durables de la
// constitución. No afirma custodia del fichero original ni firma.
type DocumentoBolsaRegistral struct {
	DocumentoRef         string    `json:"documento_ref"`
	Tipo                 string    `json:"tipo"`
	ActaRef              string    `json:"acta_ref"`
	BolsaRef             string    `json:"bolsa_ref"`
	VersionBolsa         uint64    `json:"version_bolsa"`
	InstantaneaRef       string    `json:"instantanea_ref"`
	VersionInstantanea   uint64    `json:"version_instantanea"`
	ListadoDefinitivoRef string    `json:"listado_definitivo_ref"`
	VersionListado       uint64    `json:"version_listado"`
	HuellaListadoSHA256  string    `json:"huella_listado_sha256"`
	ConfirmadaEn         time.Time `json:"confirmada_en"`
	TotalParticipaciones uint64    `json:"total_participaciones"`
}

type ConsultaDocumentosBolsa struct {
	BolsaRef string
	Tipo     string
	Texto    string
	Cursor   string
	Limite   int
}

type PaginaDocumentosBolsa struct {
	Documentos      []DocumentoBolsaRegistral
	Total           int
	CursorSiguiente *string
}

type ServicioDocumentosBolsa struct {
	lector ports.LectorConstitucionVigenteBolsa
}

func NuevoServicioDocumentosBolsa(lector ports.LectorConstitucionVigenteBolsa) (*ServicioDocumentosBolsa, error) {
	if lector == nil {
		return nil, ErrConsultaDocumentosBolsaNoDisponible
	}
	return &ServicioDocumentosBolsa{lector: lector}, nil
}

func (s *ServicioDocumentosBolsa) Consultar(ctx context.Context, q ConsultaDocumentosBolsa) (PaginaDocumentosBolsa, error) {
	vacia := PaginaDocumentosBolsa{}
	if ctx == nil || ctx.Err() != nil || s == nil || s.lector == nil {
		return vacia, ErrConsultaDocumentosBolsaNoDisponible
	}
	if !referenciaBolsaDocumentoValida(q.BolsaRef) || (q.Tipo != "" && q.Tipo != "constitucion") ||
		len(q.Texto) > 120 || q.Texto != strings.TrimSpace(q.Texto) ||
		q.Limite < 1 || q.Limite > 100 || len(q.Cursor) > 512 || strings.ContainsAny(q.Cursor, "/\x00\r\n") {
		return vacia, ErrConsultaDocumentosBolsaInvalida
	}
	vigentes, err := s.lector.ListarVigentesBolsa(ctx, q.BolsaRef)
	if err != nil {
		return vacia, err
	}
	if len(vigentes) > 1 {
		return vacia, ErrConsultaDocumentosBolsaNoDisponible
	}
	if len(vigentes) == 0 {
		return vacia, ErrDocumentoBolsaNoEncontrado
	}
	v := vigentes[0]
	if v.Bolsa.BolsaRef != q.BolsaRef || !referenciaDocumentoBolsaValida(v.ActaRef) ||
		!referenciaDocumentoBolsaValida(v.Instantanea.InstantaneaRef) ||
		!referenciaDocumentoBolsaValida(v.Instantanea.ListadoDefinitivoRef) ||
		v.Bolsa.Version == 0 || v.Instantanea.Version == 0 || v.Instantanea.VersionListado == 0 ||
		!huellaListadoBolsaValida(v.Instantanea.HuellaListadoSHA256) || v.ConfirmadaEn.IsZero() {
		return vacia, ErrConsultaDocumentosBolsaNoDisponible
	}
	if q.Cursor != "" && q.Cursor != v.ActaRef {
		return vacia, ErrConsultaDocumentosBolsaInvalida
	}
	doc := DocumentoBolsaRegistral{DocumentoRef: v.ActaRef, Tipo: "constitucion", ActaRef: v.ActaRef,
		BolsaRef: v.Bolsa.BolsaRef, VersionBolsa: v.Bolsa.Version,
		InstantaneaRef: v.Instantanea.InstantaneaRef, VersionInstantanea: v.Instantanea.Version,
		ListadoDefinitivoRef: v.Instantanea.ListadoDefinitivoRef, VersionListado: v.Instantanea.VersionListado,
		HuellaListadoSHA256: v.Instantanea.HuellaListadoSHA256,
		ConfirmadaEn:        v.ConfirmadaEn.UTC(), TotalParticipaciones: v.TotalParticipaciones}
	resultado := PaginaDocumentosBolsa{Documentos: []DocumentoBolsaRegistral{}, Total: 0}
	if q.Texto != "" && !strings.Contains(strings.ToLower(doc.ActaRef+" "+doc.InstantaneaRef+" constitucion"), strings.ToLower(q.Texto)) {
		if q.Cursor != "" {
			return vacia, ErrConsultaDocumentosBolsaInvalida
		}
		return resultado, nil
	}
	resultado.Total = 1
	if q.Cursor == "" {
		resultado.Documentos = append(resultado.Documentos, doc)
	}
	return resultado, nil
}

func referenciaDocumentoBolsaValida(ref string) bool {
	if ref == "" || len(ref) > 512 || strings.Contains(ref, "..") {
		return false
	}
	for _, c := range ref {
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') &&
			c != ':' && c != '.' && c != '_' && c != '~' && c != '-' {
			return false
		}
	}
	return true
}

func referenciaBolsaDocumentoValida(ref string) bool {
	if !strings.HasPrefix(ref, "bolsa:") || len(ref) > 200 || strings.Contains(ref, "..") ||
		len(ref) <= len("bolsa:") {
		return false
	}
	for _, c := range ref {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') &&
			c != ':' && c != '.' && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func huellaListadoBolsaValida(valor string) bool {
	if len(valor) != 64 || strings.ToLower(valor) != valor {
		return false
	}
	bytes, err := hex.DecodeString(valor)
	return err == nil && len(bytes) == 32
}
