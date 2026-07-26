// Package fichero aporta adaptadores locales de solo lectura para paquetes de
// demostracion. No constituye la autoridad productiva ni realiza importaciones
// implicitas al arrancar la aplicacion.
package fichero

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	versionEsquemaPaqueteCatalogo = 1
	maximoBytesPaqueteCatalogo    = 4 << 20
)

var (
	_ ports.ConsultaCatalogosConfigurables        = (*ConsultaCatalogos)(nil)
	_ ports.ConsultaCatalogosConfigurablesAcotada = (*ConsultaCatalogos)(nil)
	_ ports.ConsultaMetadatosFuenteCatalogos      = (*ConsultaCatalogos)(nil)

	patronRevisionPaquete = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,79}$`)
	patronSHA256Paquete   = regexp.MustCompile(`^[a-f0-9]{64}$`)
	patronDNIPaquete      = regexp.MustCompile(`(?i)(?:^|[^0-9])[0-9]{8}[A-Z](?:$|[^A-Z0-9])`)
	patronNIEPaquete      = regexp.MustCompile(`(?i)(?:^|[^A-Z0-9])[XYZ][0-9]{7}[A-Z](?:$|[^A-Z0-9])`)
	patronNIFPaquete      = regexp.MustCompile(`(?i)(?:^|[^A-Z0-9])[ABCDEFGHJNPQRSUVW][0-9]{7}[0-9A-J](?:$|[^A-Z0-9])`)
	patronCorreoPaquete   = regexp.MustCompile(`(?i)[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}`)
	patronTelefonoPaquete = regexp.MustCompile(`(?:^|[^0-9])(?:\+34[ .-]?)?[6789][0-9 .-]{7,12}[0-9](?:$|[^0-9])`)
	patronIBANPaquete     = regexp.MustCompile(`(?i)(?:^|[^A-Z0-9])ES[0-9]{22}(?:$|[^A-Z0-9])`)
	patronClavePaquete    = regexp.MustCompile(`(?i)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`)
)

type metadatosPaqueteCatalogo struct {
	Revision      string    `json:"revision"`
	ActualizadaEn time.Time `json:"actualizada_en"`
	Demostracion  bool      `json:"demostracion"`
	Aviso         string    `json:"aviso"`
	OrigenSHA256  string    `json:"origen_sha256"`
}

type paqueteCatalogo struct {
	VersionEsquema int                         `json:"version_esquema"`
	Fuente         metadatosPaqueteCatalogo    `json:"fuente"`
	Catalogo       domain.CatalogoConfigurable `json:"catalogo"`
}

// ConsultaCatalogos mantiene una unica instantanea canonica e inmutable. Un
// paquete de fichero solo sirve como adaptador DEMO explicito; PostgreSQL,
// Oracle u otro repositorio implementan el mismo puerto del nucleo.
type ConsultaCatalogos struct {
	catalogo  domain.CatalogoConfigurable
	metadatos ports.MetadatosFuenteCatalogos
}

func NuevaConsultaCatalogos(ruta string) (*ConsultaCatalogos, error) {
	ruta = strings.TrimSpace(ruta)
	if ruta == "" {
		return nil, fmt.Errorf("%w: ruta vacia", domain.ErrCatalogoConfigurableInvalido)
	}
	fichero, err := os.Open(ruta)
	if err != nil {
		return nil, fmt.Errorf("%w: no se puede abrir el paquete DEMO", domain.ErrCatalogoConfigurableInvalido)
	}
	defer fichero.Close()
	informacion, err := fichero.Stat()
	if err != nil || !informacion.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: el paquete no es un fichero regular", domain.ErrCatalogoConfigurableInvalido)
	}
	contenido, err := io.ReadAll(io.LimitReader(fichero, maximoBytesPaqueteCatalogo+1))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoBytesPaqueteCatalogo {
		return nil, fmt.Errorf("%w: tamano no permitido", domain.ErrCatalogoConfigurableInvalido)
	}
	return nuevaConsultaCatalogosDesdeJSON(contenido)
}

func nuevaConsultaCatalogosDesdeJSON(contenido []byte) (*ConsultaCatalogos, error) {
	if patronDNIPaquete.Match(contenido) || patronNIEPaquete.Match(contenido) || patronNIFPaquete.Match(contenido) || patronCorreoPaquete.Match(contenido) ||
		patronTelefonoPaquete.Match(contenido) || patronIBANPaquete.Match(contenido) || patronClavePaquete.Match(contenido) {
		return nil, fmt.Errorf("%w: posible dato personal o secreto", domain.ErrCatalogoConfigurableInvalido)
	}
	if err := validarJSONCatalogoSinClavesDuplicadas(contenido); err != nil {
		return nil, fmt.Errorf("%w: contenido JSON rechazado", domain.ErrCatalogoConfigurableInvalido)
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var paquete paqueteCatalogo
	if err := decodificador.Decode(&paquete); err != nil {
		return nil, fmt.Errorf("%w: esquema no valido", domain.ErrCatalogoConfigurableInvalido)
	}
	if err := exigirFinJSONCatalogo(decodificador); err != nil {
		return nil, fmt.Errorf("%w: mas de un documento JSON", domain.ErrCatalogoConfigurableInvalido)
	}
	if err := validarPaqueteCatalogo(paquete); err != nil {
		return nil, err
	}
	canonico, err := paquete.Catalogo.ClonarCanonico()
	if err != nil {
		return nil, fmt.Errorf("%w: catalogo no valido", domain.ErrCatalogoConfigurableInvalido)
	}
	return &ConsultaCatalogos{
		catalogo: canonico,
		metadatos: ports.MetadatosFuenteCatalogos{
			Revision:      paquete.Fuente.Revision,
			ActualizadaEn: paquete.Fuente.ActualizadaEn,
			Demostracion:  paquete.Fuente.Demostracion,
			Aviso:         paquete.Fuente.Aviso,
		},
	}, nil
}

func (c *ConsultaCatalogos) ObtenerCatalogo(ctx context.Context, id string, version int) (domain.CatalogoConfigurable, error) {
	if ctx == nil || c == nil || id != strings.TrimSpace(id) || id == "" || version < 1 {
		return domain.CatalogoConfigurable{}, ports.ErrCatalogoNoEncontrado
	}
	if err := ctx.Err(); err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	if c.catalogo.ID != id || c.catalogo.Version != version {
		return domain.CatalogoConfigurable{}, ports.ErrCatalogoNoEncontrado
	}
	return c.catalogo.ClonarCanonico()
}

func (c *ConsultaCatalogos) ObtenerCatalogoAcotado(
	ctx context.Context,
	id string,
	version int,
	limites ports.LimitesConsultaCatalogosAcotada,
) (ports.ResultadoConsultaCatalogoAcotado, error) {
	if ctx == nil || c == nil ||
		id != strings.TrimSpace(id) || id == "" || version < 1 {
		return ports.ResultadoConsultaCatalogoAcotado{},
			ports.ErrCatalogoNoEncontrado
	}
	if limites.Validar() != nil {
		return ports.ResultadoConsultaCatalogoAcotado{},
			ports.ErrLimitesConsultaCatalogosInvalidos
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoConsultaCatalogoAcotado{}, err
	}
	if c.catalogo.ID != id || c.catalogo.Version != version {
		return ports.ResultadoConsultaCatalogoAcotado{},
			ports.ErrCatalogoNoEncontrado
	}
	medida, medible := ports.MedirCatalogoConfigurable(c.catalogo)
	_, cabe := (ports.ConsumoConsultaCatalogosAcotada{}).Agregar(
		medida,
		limites,
	)
	if !medible || !cabe {
		return ports.ResultadoConsultaCatalogoAcotado{Truncado: true}, nil
	}
	clon, err := c.catalogo.ClonarCanonico()
	if err != nil {
		return ports.ResultadoConsultaCatalogoAcotado{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoConsultaCatalogoAcotado{}, err
	}
	return ports.ResultadoConsultaCatalogoAcotado{Catalogo: clon}, nil
}

func (c *ConsultaCatalogos) ListarVersionesCatalogo(ctx context.Context, id string) ([]domain.CatalogoConfigurable, error) {
	if ctx == nil || c == nil || id != strings.TrimSpace(id) || id == "" {
		return nil, ports.ErrCatalogoNoEncontrado
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.catalogo.ID != id {
		return nil, ports.ErrCatalogoNoEncontrado
	}
	clon, err := c.catalogo.ClonarCanonico()
	if err != nil {
		return nil, err
	}
	return []domain.CatalogoConfigurable{clon}, nil
}

func (c *ConsultaCatalogos) ListarVersionesCatalogoAcotado(
	ctx context.Context,
	id string,
	limites ports.LimitesConsultaCatalogosAcotada,
) (ports.ResultadoConsultaCatalogosAcotada, error) {
	if ctx == nil || c == nil ||
		id != strings.TrimSpace(id) || id == "" {
		return ports.ResultadoConsultaCatalogosAcotada{},
			ports.ErrCatalogoNoEncontrado
	}
	if limites.Validar() != nil {
		return ports.ResultadoConsultaCatalogosAcotada{},
			ports.ErrLimitesConsultaCatalogosInvalidos
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoConsultaCatalogosAcotada{}, err
	}
	if c.catalogo.ID != id {
		return ports.ResultadoConsultaCatalogosAcotada{},
			ports.ErrCatalogoNoEncontrado
	}
	medida, medible := ports.MedirCatalogoConfigurable(c.catalogo)
	_, cabe := (ports.ConsumoConsultaCatalogosAcotada{}).Agregar(
		medida,
		limites,
	)
	if !medible || !cabe {
		return ports.ResultadoConsultaCatalogosAcotada{Truncado: true}, nil
	}
	clon, err := c.catalogo.ClonarCanonico()
	if err != nil {
		return ports.ResultadoConsultaCatalogosAcotada{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoConsultaCatalogosAcotada{}, err
	}
	return ports.ResultadoConsultaCatalogosAcotada{
		Catalogos: []domain.CatalogoConfigurable{clon},
	}, nil
}

func (c *ConsultaCatalogos) ObtenerMetadatosFuenteCatalogos(ctx context.Context) (ports.MetadatosFuenteCatalogos, error) {
	if ctx == nil || c == nil {
		return ports.MetadatosFuenteCatalogos{}, domain.ErrCatalogoConfigurableInvalido
	}
	if err := ctx.Err(); err != nil {
		return ports.MetadatosFuenteCatalogos{}, err
	}
	return c.metadatos, nil
}

func validarPaqueteCatalogo(paquete paqueteCatalogo) error {
	fuente := paquete.Fuente
	if paquete.VersionEsquema != versionEsquemaPaqueteCatalogo || !fuente.Demostracion ||
		!strings.Contains(strings.ToUpper(fuente.Aviso), "DEMOSTRACI") ||
		!patronRevisionPaquete.MatchString(fuente.Revision) ||
		!patronSHA256Paquete.MatchString(fuente.OrigenSHA256) ||
		fuente.ActualizadaEn.IsZero() || fuente.ActualizadaEn.Location() != time.UTC ||
		fuente.ActualizadaEn.Nanosecond()%1000 != 0 || fuente.Aviso != strings.TrimSpace(fuente.Aviso) {
		return fmt.Errorf("%w: metadatos DEMO no validos", domain.ErrCatalogoConfigurableInvalido)
	}
	if _, err := paquete.Catalogo.ClonarCanonico(); err != nil {
		return fmt.Errorf("%w: catalogo no valido", domain.ErrCatalogoConfigurableInvalido)
	}
	return nil
}

func validarJSONCatalogoSinClavesDuplicadas(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	if err := validarValorJSONCatalogo(decodificador); err != nil {
		return err
	}
	return exigirFinJSONCatalogo(decodificador)
}

func validarValorJSONCatalogo(decodificador *json.Decoder) error {
	token, err := decodificador.Token()
	if err != nil {
		return err
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimitador {
	case '{':
		vistas := make(map[string]struct{})
		for decodificador.More() {
			claveToken, err := decodificador.Token()
			if err != nil {
				return err
			}
			clave, ok := claveToken.(string)
			if !ok {
				return errors.New("clave JSON no valida")
			}
			if _, existe := vistas[clave]; existe {
				return errors.New("clave JSON duplicada")
			}
			vistas[clave] = struct{}{}
			if err := validarValorJSONCatalogo(decodificador); err != nil {
				return err
			}
		}
		_, err = decodificador.Token()
		return err
	case '[':
		for decodificador.More() {
			if err := validarValorJSONCatalogo(decodificador); err != nil {
				return err
			}
		}
		_, err = decodificador.Token()
		return err
	default:
		return errors.New("delimitador JSON no valido")
	}
}

func exigirFinJSONCatalogo(decodificador *json.Decoder) error {
	var extra any
	err := decodificador.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return errors.New("contenido JSON adicional")
}
