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
	CategoriaRef         string
	BolsaRef             string
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

// NuevoPreparador expone la misma validación para una transacción que ya posee
// la persistencia y la autorización. No recibe acceso de escritura al staging.
func NuevoPreparador(decodificador DecodificadorExportacion, reloj Reloj) (*Servicio, error) {
	if decodificador == nil {
		return nil, ErrDecodificadorRequerido
	}
	if reloj == nil {
		return nil, ErrRelojRequerido
	}
	return &Servicio{decodificador: decodificador, reloj: reloj}, nil
}

func (s *Servicio) Importar(
	ctx context.Context,
	solicitud SolicitudImportacion,
) (ResultadoImportacion, error) {
	if s == nil || s.repositorio == nil {
		return ResultadoImportacion{}, ErrRepositorioRequerido
	}
	lote, err := s.PrepararLote(ctx, solicitud)
	if err != nil {
		return ResultadoImportacion{}, err
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

// PrepararLote aplica las mismas reglas de importación que el CLI sin escribir
// en PostgreSQL. B1 entrega este lote a su única transacción autorizada.
func (s *Servicio) PrepararLote(ctx context.Context, solicitud SolicitudImportacion) (dominio.LoteValidado, error) {
	if ctx == nil || s == nil || s.decodificador == nil || s.reloj == nil {
		return dominio.LoteValidado{}, ErrDecodificadorRequerido
	}
	if err := ctx.Err(); err != nil {
		return dominio.LoteValidado{}, err
	}
	if !solicitudValida(solicitud) {
		return dominio.LoteValidado{}, ErrSolicitudInvalida
	}
	suma := sha256.Sum256(solicitud.Contenido)
	huella := hex.EncodeToString(suma[:])
	hoja, err := s.decodificador.Decodificar(ctx, bytes.NewReader(solicitud.Contenido))
	if err != nil {
		return dominio.LoteValidado{}, err
	}
	staging, err := dominio.ValidarHoja(hoja)
	if err != nil {
		return dominio.LoteValidado{}, err
	}
	registradaEn := s.reloj().UTC().Truncate(time.Microsecond)
	if registradaEn.IsZero() {
		return dominio.LoteValidado{}, ErrResultadoInseguro
	}
	lote := dominio.LoteValidado{
		Acta: dominio.ActaImportacion{
			CategoriaRef: solicitud.CategoriaRef, BolsaRef: solicitud.BolsaRef,
			ActaRef: ReferenciaActa(huella, solicitud.CategoriaRef), ImportacionRef: "importacion:convoca:" + referenciaContexto(huella, solicitud.CategoriaRef),
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
		return dominio.LoteValidado{}, ErrResultadoInseguro
	}
	return lote, nil
}

func solicitudValida(s SolicitudImportacion) bool {
	if !referenciaOpacaDurable.MatchString(s.CategoriaRef) || (s.BolsaRef != "" && !referenciaOpacaDurable.MatchString(s.BolsaRef)) || len(s.Contenido) == 0 || len(s.Contenido) > MaximoBytesExportacion ||
		!referenciaOpacaDurable.MatchString(s.FicheroCustodiadoRef) ||
		strings.TrimSpace(s.NombreFichero) != s.NombreFichero ||
		strings.TrimSpace(s.ActorRef) != s.ActorRef ||
		!actorOpaco.MatchString(s.ActorRef) || !utf8.ValidString(s.NombreFichero) ||
		len(s.NombreFichero) > 255 || filepath.Base(s.NombreFichero) != s.NombreFichero ||
		strings.ContainsAny(s.NombreFichero, `/\`) ||
		!nombreExportacionConvocaValido(s.NombreFichero) {
		return false
	}
	for _, r := range s.NombreFichero {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func nombreExportacionConvocaValido(nombre string) bool {
	extension := strings.ToLower(filepath.Ext(nombre))
	return len(nombre) > len(extension) && (extension == ".xls" || extension == ".xlsx")
}

// ReferenciaActa es la referencia del acta de un fichero (por su huella
// SHA-256) importado con una categoría. Es determinista: la pantalla de carga
// la conoce antes de importar y la usa como recurso de la decisión.
func ReferenciaActa(huellaSHA256, categoriaRef string) string {
	return "acta:importacion-convoca:" + referenciaContexto(huellaSHA256, categoriaRef)
}

func referenciaContexto(huella, categoria string) string {
	s := sha256.Sum256([]byte(huella + "\x1f" + categoria))
	return hex.EncodeToString(s[:])
}
