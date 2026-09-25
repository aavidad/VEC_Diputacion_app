package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// ConfiguracionDocumentacionFormalizacion nombra las reglas del catálogo que
// gobiernan la formalización y cómo se llama en Documentos cada documento.
// Todo procede de la composición: cambiar el catálogo o esta configuración
// cambia la conducta sin tocar el código.
type ConfiguracionDocumentacionFormalizacion struct {
	Reglas ports.ReglasFormalizacion
	Tipos  ports.CatalogoTiposDocumentalesFormalizacion
	Reloj  ports.Reloj
	// ClavePlazoDocumentacion y ClaveDocumentos son obligatorias.
	ClavePlazoDocumentacion string
	ClaveDocumentos         string
	// ClavePlazoIncorporacion es opcional: sin ella no se informa el margen.
	ClavePlazoIncorporacion string
	// PrefijoTipoDocumental y SufijoTipoDocumental forman el tipo documental
	// de cada documento exigido: prefijo + clave + sufijo.
	PrefijoTipoDocumental string
	SufijoTipoDocumental  string
}

// SolicitudDocumentacionFormalizacion parte de la aceptación ya confirmada.
// La modalidad solo selecciona la lista si el catálogo la distingue.
type SolicitudDocumentacionFormalizacion struct {
	ExpedienteRef string
	AceptadaEn    time.Time
	Modalidad     string
}

const dominioExpedienteDocumentalFormalizacion = "vec.contratacion_temporal.formalizacion.documentacion.v1"

// ReferenciaExpedienteDocumentalFormalizacion agrupa en Documentos la
// documentación de formalización de un expediente con una referencia opaca
// derivada, como hace Dietas con sus comisiones: Documentos no recibe la
// referencia del expediente de Contratación temporal.
func ReferenciaExpedienteDocumentalFormalizacion(expedienteRef string) (string, error) {
	if !domain.ReferenciaOpacaValida(expedienteRef) {
		return "", ports.ErrSolicitudDocumentacionFormalizacion
	}
	suma := sha256.Sum256([]byte(dominioExpedienteDocumentalFormalizacion + "\x00" + expedienteRef))
	return "ref:" + hex.EncodeToString(suma[:]), nil
}

// EstadoPlazoFormalizacion: en_curso, ultimo_dia (hoy es el último día en
// hora peninsular) o vencido. No se inventa un umbral de «próximo a vencer».
type EstadoPlazoFormalizacion string

const (
	PlazoFormalizacionEnCurso   EstadoPlazoFormalizacion = "en_curso"
	PlazoFormalizacionUltimoDia EstadoPlazoFormalizacion = "ultimo_dia"
	PlazoFormalizacionVencido   EstadoPlazoFormalizacion = "vencido"
)

type PlazoFormalizacion struct {
	Regla       ports.ReglaFormalizacion
	Vencimiento ports.VencimientoFormalizacion
	Estado      EstadoPlazoFormalizacion
}

type DocumentoExigidoFormalizacion struct {
	Clave          string
	TipoDocumental string
	// Registrable es falso si el tipo no tiene política de conservación: la
	// anotación en Documentos se rechazaría y la interfaz no la ofrece.
	Registrable bool
}

type DocumentacionFormalizacion struct {
	// ExpedienteDocumentalRef es la agrupación de Documentos donde constan
	// las anotaciones de esta documentación.
	ExpedienteDocumentalRef string
	AceptadaEn              time.Time
	Modalidad               string
	PorModalidad            bool
	Documentos              []DocumentoExigidoFormalizacion
	ReglaDocumentos         ports.ReglaFormalizacion
	PlazoDocumentacion      PlazoFormalizacion
	PlazoIncorporacion      *PlazoFormalizacion
	ConsultadaEn            time.Time
}

type ServicioDocumentacionFormalizacion struct {
	cfg  ConfiguracionDocumentacionFormalizacion
	zona *time.Location
}

var (
	ErrConfiguracionDocumentacionFormalizacion = errors.New("contratacion temporal: configuracion de documentacion de formalizacion invalida")
	claveDocumentoFormalizacion                = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	tipoDocumentalFormalizacion                = regexp.MustCompile(`^[a-z][a-z0-9_.]{2,127}$`)
)

const maximoDocumentosFormalizacion = 64

func NuevoServicioDocumentacionFormalizacion(cfg ConfiguracionDocumentacionFormalizacion) (*ServicioDocumentacionFormalizacion, error) {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil || dependenciaNula(cfg.Reglas) || dependenciaNula(cfg.Tipos) || dependenciaNula(cfg.Reloj) ||
		strings.TrimSpace(cfg.ClavePlazoDocumentacion) == "" || strings.TrimSpace(cfg.ClaveDocumentos) == "" ||
		!tipoDocumentalFormalizacion.MatchString(cfg.PrefijoTipoDocumental+"x"+cfg.SufijoTipoDocumental) {
		return nil, ErrConfiguracionDocumentacionFormalizacion
	}
	return &ServicioDocumentacionFormalizacion{cfg: cfg, zona: zona}, nil
}

// Consultar calcula la documentación exigida y sus plazos desde la aceptación.
// No registra nada ni afirma que la documentación esté aportada: el estado
// de cada documento lo conserva Documentos.
func (s *ServicioDocumentacionFormalizacion) Consultar(ctx context.Context, solicitud SolicitudDocumentacionFormalizacion) (DocumentacionFormalizacion, error) {
	var vacia DocumentacionFormalizacion
	if s == nil || ctx == nil {
		return vacia, ports.ErrReglasFormalizacionNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	ahora := s.cfg.Reloj.Ahora().UTC()
	aceptada := solicitud.AceptadaEn.UTC()
	agrupacion, err := ReferenciaExpedienteDocumentalFormalizacion(solicitud.ExpedienteRef)
	if err != nil || aceptada.IsZero() || aceptada.Year() < 2000 || aceptada.After(ahora) ||
		(solicitud.Modalidad != "" && !claveDocumentoFormalizacion.MatchString(solicitud.Modalidad)) {
		return vacia, ports.ErrSolicitudDocumentacionFormalizacion
	}
	lista, err := s.cfg.Reglas.ReglaFormalizacion(ctx, s.cfg.ClaveDocumentos)
	if err != nil {
		return vacia, normalizarErrorReglasFormalizacion(ctx, err)
	}
	elementos, porModalidad := lista.Elementos, false
	if porLista, ok := lista.ElementosPorModalidad[solicitud.Modalidad]; ok && solicitud.Modalidad != "" {
		elementos, porModalidad = porLista, true
	}
	documentos, err := s.documentosExigidos(elementos)
	if err != nil {
		return vacia, err
	}
	plazo, err := s.plazo(ctx, s.cfg.ClavePlazoDocumentacion, aceptada, ahora)
	if err != nil {
		return vacia, err
	}
	resultado := DocumentacionFormalizacion{
		ExpedienteDocumentalRef: agrupacion, AceptadaEn: aceptada, Modalidad: solicitud.Modalidad, PorModalidad: porModalidad,
		Documentos: documentos, ReglaDocumentos: lista, PlazoDocumentacion: plazo, ConsultadaEn: ahora,
	}
	if s.cfg.ClavePlazoIncorporacion != "" {
		incorporacion, err := s.plazo(ctx, s.cfg.ClavePlazoIncorporacion, aceptada, ahora)
		if err != nil {
			return vacia, err
		}
		resultado.PlazoIncorporacion = &incorporacion
	}
	return resultado, nil
}

func (s *ServicioDocumentacionFormalizacion) documentosExigidos(elementos []string) ([]DocumentoExigidoFormalizacion, error) {
	if len(elementos) == 0 || len(elementos) > maximoDocumentosFormalizacion {
		return nil, ports.ErrReglasFormalizacionNoDisponibles
	}
	vistos := make(map[string]bool, len(elementos))
	documentos := make([]DocumentoExigidoFormalizacion, 0, len(elementos))
	for _, clave := range elementos {
		tipo := s.cfg.PrefijoTipoDocumental + clave + s.cfg.SufijoTipoDocumental
		if !claveDocumentoFormalizacion.MatchString(clave) || vistos[clave] || !tipoDocumentalFormalizacion.MatchString(tipo) {
			return nil, ports.ErrReglasFormalizacionNoDisponibles
		}
		vistos[clave] = true
		documentos = append(documentos, DocumentoExigidoFormalizacion{
			Clave: clave, TipoDocumental: tipo, Registrable: s.cfg.Tipos.TipoDocumentalCatalogado(tipo),
		})
	}
	return documentos, nil
}

func (s *ServicioDocumentacionFormalizacion) plazo(ctx context.Context, clave string, inicio, ahora time.Time) (PlazoFormalizacion, error) {
	regla, vencimiento, err := s.cfg.Reglas.VencimientoFormalizacion(ctx, clave, inicio)
	if err != nil {
		return PlazoFormalizacion{}, normalizarErrorReglasFormalizacion(ctx, err)
	}
	if vencimiento.UltimoDia == "" || !vencimiento.VenceAntesDe.After(inicio) {
		return PlazoFormalizacion{}, ports.ErrReglasFormalizacionNoDisponibles
	}
	estado := PlazoFormalizacionEnCurso
	switch {
	case !ahora.Before(vencimiento.VenceAntesDe):
		estado = PlazoFormalizacionVencido
	case ahora.In(s.zona).Format(time.DateOnly) == vencimiento.UltimoDia:
		estado = PlazoFormalizacionUltimoDia
	}
	vencimiento.VenceAntesDe = vencimiento.VenceAntesDe.UTC()
	return PlazoFormalizacion{Regla: regla, Vencimiento: vencimiento, Estado: estado}, nil
}

func normalizarErrorReglasFormalizacion(ctx context.Context, err error) error {
	if errContexto := ctx.Err(); errContexto != nil {
		return errContexto
	}
	if errors.Is(err, ports.ErrReglasFormalizacionNoConfiguradas) {
		return ports.ErrReglasFormalizacionNoConfiguradas
	}
	return ports.ErrReglasFormalizacionNoDisponibles
}
