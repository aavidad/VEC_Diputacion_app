package composicion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

type ProveedorFormatosExportacionServiciosPropios struct {
	formatos map[string]domain.FormatoExportacionServiciosPropios
}

// Los bytes vienen de los catálogos comunes leídos por el montaje. No hay
// idioma predeterminado: una traducción ausente mantiene la operación cerrada.
func NuevoProveedorFormatosExportacionServiciosPropios(catalogos map[string][]byte) (*ProveedorFormatosExportacionServiciosPropios, error) {
	if len(catalogos) == 0 || len(catalogos) > 100 {
		return nil, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	p := &ProveedorFormatosExportacionServiciosPropios{formatos: map[string]domain.FormatoExportacionServiciosPropios{}}
	for idioma, b := range catalogos {
		if len(b) == 0 || len(b) > 64<<10 {
			return nil, domain.ErrExportacionServiciosPropiosNoDisponible
		}
		var datos struct {
			Formato struct {
				Referencia    string `json:"referencia"`
				Version       string `json:"version"`
				NombreArchivo string `json:"nombre_archivo"`
			} `json:"formato"`
			CSV struct {
				Inicio string `json:"fecha_inicio"`
				Fin    string `json:"fecha_fin"`
				Clase  string `json:"clase"`
				Dias   string `json:"dias"`
				Estado string `json:"estado"`
			} `json:"csv"`
			General map[string]string `json:"general"`
			Estados map[string]string `json:"estados"`
		}
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if dec.Decode(&datos) != nil || dec.Decode(new(any)) != io.EOF {
			return nil, domain.ErrExportacionServiciosPropiosNoDisponible
		}
		version, err := strconv.ParseUint(datos.Formato.Version, 10, 64)
		if err != nil || strconv.FormatUint(version, 10) != datos.Formato.Version {
			return nil, domain.ErrExportacionServiciosPropiosNoDisponible
		}
		h := sha256.Sum256(b)
		f, err := domain.NuevoFormatoExportacionServiciosPropios(domain.DatosFormatoExportacionServiciosPropios{Referencia: datos.Formato.Referencia, Version: version, NombreArchivo: datos.Formato.NombreArchivo, Idioma: idioma, CatalogoSHA256: hex.EncodeToString(h[:]), Cabeceras: []string{datos.CSV.Inicio, datos.CSV.Fin, datos.CSV.Clase, datos.CSV.Dias, datos.CSV.Estado}, Estados: datos.Estados})
		if err != nil {
			return nil, domain.ErrExportacionServiciosPropiosNoDisponible
		}
		p.formatos[idioma] = f
	}
	return p, nil
}
func (p *ProveedorFormatosExportacionServiciosPropios) FormatoParaIdioma(ctx context.Context, idioma string) (domain.FormatoExportacionServiciosPropios, error) {
	if p == nil || ctx == nil || ctx.Err() != nil {
		return domain.FormatoExportacionServiciosPropios{}, domain.ErrExportacionServiciosPropiosNoDisponible
	}
	f, ok := p.formatos[idioma]
	if !ok {
		return domain.FormatoExportacionServiciosPropios{}, domain.ErrExportacionServiciosPropiosInvalida
	}
	return domain.NuevoFormatoExportacionServiciosPropios(f.Datos())
}

var _ ports.ProveedorFormatoExportacionServiciosPropios = (*ProveedorFormatosExportacionServiciosPropios)(nil)
