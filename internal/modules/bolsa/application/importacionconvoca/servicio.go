// Package importacionconvoca orquesta la importacion gobernada de una
// exportacion enmascarada de Convoca. Las interfaces viven junto al caso de
// uso para no ampliar artificialmente internal/modules/bolsa/ports (DEC-051).
package importacionconvoca

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

const MaximoBytesExportacion = 16 * 1024 * 1024

var (
	ErrDecodificadorRequerido = errors.New("bolsa: decodificador Convoca requerido")
	ErrRepositorioRequerido   = errors.New("bolsa: repositorio de importaciones Convoca requerido")
	ErrRelojRequerido         = errors.New("bolsa: reloj de importacion Convoca requerido")
	ErrSolicitudInvalida      = errors.New("bolsa: solicitud de importacion Convoca invalida")
	ErrResultadoInseguro      = errors.New("bolsa: resultado de importacion Convoca inseguro")
	ErrImportacionEnConflicto = errors.New("bolsa: importacion Convoca en conflicto")
)

var actorOpaco = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{2,127}$`)

type DecodificadorExportacion interface {
	Decodificar(context.Context, io.ReadSeeker) (dominio.HojaStaging, error)
}

// RepositorioImportaciones confirma el staging y el acta de forma atomica.
// El booleano indica que ya existia el mismo SHA-256 y no se duplico el lote.
type RepositorioImportaciones interface {
	GuardarSiAusente(context.Context, dominio.LoteValidado) (dominio.ActaImportacion, bool, error)
}

type Reloj func() time.Time

type SolicitudImportacion struct {
	NombreFichero        string
	FicheroCustodiadoRef string
	ActorRef             string
	Contenido            []byte
}

type ResultadoImportacion struct {
	Acta        dominio.ActaImportacion
	Reutilizada bool
}

type Servicio struct {
	decodificador DecodificadorExportacion
	repositorio   RepositorioImportaciones
	reloj         Reloj
}

func NuevoServicio(
	decodificador DecodificadorExportacion,
	repositorio RepositorioImportaciones,
	reloj Reloj,
) (*Servicio, error) {
	if decodificador == nil {
		return nil, ErrDecodificadorRequerido
	}
	if repositorio == nil {
		return nil, ErrRepositorioRequerido
	}
	if reloj == nil {
		return nil, ErrRelojRequerido
	}
	return &Servicio{decodificador: decodificador, repositorio: repositorio, reloj: reloj}, nil
}

func (s *Servicio) Importar(
	ctx context.Context,
	solicitud SolicitudImportacion,
) (ResultadoImportacion, error) {
	if err := ctx.Err(); err != nil {
		return ResultadoImportacion{}, err
	}
	if !solicitudValida(solicitud) {
		return ResultadoImportacion{}, ErrSolicitudInvalida
	}
	suma := sha256.Sum256(solicitud.Contenido)
	huella := hex.EncodeToString(suma[:])
	hoja, err := s.decodificador.Decodificar(ctx, bytes.NewReader(solicitud.Contenido))
	if err != nil {
		return ResultadoImportacion{}, err
	}
	staging, err := dominio.ValidarHoja(hoja)
	if err != nil {
		return ResultadoImportacion{}, err
	}
	registradaEn := s.reloj().UTC().Truncate(time.Microsecond)
	if registradaEn.IsZero() {
		return ResultadoImportacion{}, ErrResultadoInseguro
	}
	lote := dominio.LoteValidado{
		Acta: dominio.ActaImportacion{
			ActaRef:              "acta:importacion-convoca:" + huella,
			ImportacionRef:       "importacion:convoca:" + huella,
			HuellaFicheroSHA256:  huella,
			FicheroCustodiadoRef: solicitud.FicheroCustodiadoRef,
			NombreFichero:        solicitud.NombreFichero,
			ActorRef:             solicitud.ActorRef,
			RegistradaEn:         registradaEn,
			Esquema:              hoja.Esquema,
			FilasLeidas:          staging.FilasLeidas,
			FilasAceptadas:       len(staging.Aceptadas),
			FilasRechazadas:      staging.Rechazadas,
			Incidencias:          append([]dominio.Incidencia(nil), staging.Incidencias...),
			Procedencia:          dominio.NuevaProcedenciaNoAutoritativa(),
		},
		Aceptadas: append([]dominio.FilaAceptada(nil), staging.Aceptadas...),
	}
	if lote.Validar() != nil {
		return ResultadoImportacion{}, ErrResultadoInseguro
	}
	actaGuardada, reutilizada, err := s.repositorio.GuardarSiAusente(ctx, lote)
	if err != nil {
		return ResultadoImportacion{}, err
	}
	if actaGuardada.Validar() != nil || !actaGuardada.CoincideExactamente(lote.Acta) {
		return ResultadoImportacion{}, ErrResultadoInseguro
	}
	return ResultadoImportacion{Acta: actaGuardada, Reutilizada: reutilizada}, nil
}

func solicitudValida(s SolicitudImportacion) bool {
	if len(s.Contenido) == 0 || len(s.Contenido) > MaximoBytesExportacion ||
		!referenciaOpacaDurable.MatchString(s.FicheroCustodiadoRef) ||
		strings.TrimSpace(s.NombreFichero) != s.NombreFichero ||
		strings.TrimSpace(s.ActorRef) != s.ActorRef ||
		!actorOpaco.MatchString(s.ActorRef) || !utf8.ValidString(s.NombreFichero) ||
		len(s.NombreFichero) > 255 || filepath.Base(s.NombreFichero) != s.NombreFichero ||
		strings.ContainsAny(s.NombreFichero, `/\`) ||
		!strings.HasSuffix(strings.ToLower(s.NombreFichero), ".xls") {
		return false
	}
	for _, r := range s.NombreFichero {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
