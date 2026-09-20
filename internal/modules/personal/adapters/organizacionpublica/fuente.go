// Package organizacionpublica adapta la proyección pública mínima de la
// estructura organizativa DEMO inmovilizada por huella de paquete.
package organizacionpublica

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"strings"
	personalcatalogos "vec-diputacion-granada/internal/modules/personal/adapters/catalogosvec"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
)

const HuellaPaqueteEstructura2026 = "0e52d878526d6a5e7ee4ab6f525ef92a70144aef665f0b031fca6051564e054c"
const maximoBytesPaqueteEstructuraPublica = 4 << 20

type Fuente struct {
	estructura domain.EstructuraOrganizativaPublica
}

func NuevaFuente(ruta string) (*Fuente, error) {
	if strings.TrimSpace(ruta) == "" {
		return nil, domain.ErrConsultaEstructuraPublicaInvalida
	}
	contenido, err := leerPaqueteInmutable(ruta)
	if err != nil {
		return nil, domain.ErrEstructuraPublicaNoDisponible
	}
	suma := sha256.Sum256(contenido)
	if subtle.ConstantTimeCompare([]byte(hex.EncodeToString(suma[:])), []byte(HuellaPaqueteEstructura2026)) != 1 {
		return nil, domain.ErrEstructuraPublicaNoDisponible
	}
	estructura, err := cargarEstructuraDesdeInstantanea(contenido)
	if err != nil {
		return nil, domain.ErrEstructuraPublicaNoDisponible
	}
	return &Fuente{estructura: estructura}, nil
}
func (f *Fuente) ObtenerEstructuraOrganizativaPublica(ctx context.Context) (domain.EstructuraOrganizativaPublica, error) {
	if ctx == nil || f == nil {
		return domain.EstructuraOrganizativaPublica{}, domain.ErrConsultaEstructuraPublicaInvalida
	}
	if err := ctx.Err(); err != nil {
		return domain.EstructuraOrganizativaPublica{}, err
	}
	if err := f.estructura.Validar(); err != nil {
		return domain.EstructuraOrganizativaPublica{}, domain.ErrEstructuraPublicaNoDisponible
	}
	return f.estructura.Clonar(), nil
}

func leerPaqueteInmutable(ruta string) ([]byte, error) {
	info, err := os.Stat(ruta)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maximoBytesPaqueteEstructuraPublica {
		return nil, domain.ErrEstructuraPublicaNoDisponible
	}
	archivo, err := os.Open(ruta)
	if err != nil {
		return nil, domain.ErrEstructuraPublicaNoDisponible
	}
	defer archivo.Close()
	abierto, err := archivo.Stat()
	if err != nil || !abierto.Mode().IsRegular() || abierto.Size() != info.Size() {
		return nil, domain.ErrEstructuraPublicaNoDisponible
	}
	contenido, err := io.ReadAll(io.LimitReader(archivo, maximoBytesPaqueteEstructuraPublica+1))
	if err != nil || len(contenido) < 1 || len(contenido) > maximoBytesPaqueteEstructuraPublica || int64(len(contenido)) != abierto.Size() {
		return nil, domain.ErrEstructuraPublicaNoDisponible
	}
	return contenido, nil
}

func cargarEstructuraDesdeInstantanea(contenido []byte) (domain.EstructuraOrganizativaPublica, error) {
	// El lector gobernado recibe una instantánea privada de los mismos bytes ya
	// inmovilizados. No reabre la ruta de origen y por tanto una sustitución
	// posterior del fichero no puede cambiar la proyección concedida.
	instantanea, err := os.CreateTemp("", "vec-estructura-publica-")
	if err != nil {
		return domain.EstructuraOrganizativaPublica{}, domain.ErrEstructuraPublicaNoDisponible
	}
	rutaInstantanea := instantanea.Name()
	defer os.Remove(rutaInstantanea)
	if _, err := instantanea.Write(contenido); err != nil || instantanea.Close() != nil {
		instantanea.Close()
		return domain.EstructuraOrganizativaPublica{}, domain.ErrEstructuraPublicaNoDisponible
	}
	// El lector gobernado valida el catálogo y el adaptador de Personal valida el árbol.
	lector, err := fichero.NuevaConsultaCatalogos(rutaInstantanea)
	if err != nil {
		return domain.EstructuraOrganizativaPublica{}, domain.ErrEstructuraPublicaNoDisponible
	}
	consulta, err := personalcatalogos.NuevaConsultaEstructuraOrganizativa(lector, "estructura-organizativa-dipgra", 1)
	if err != nil {
		return domain.EstructuraOrganizativaPublica{}, domain.ErrEstructuraPublicaNoDisponible
	}
	estructura, err := consulta.Obtener(context.Background())
	if err != nil {
		return domain.EstructuraOrganizativaPublica{}, domain.ErrEstructuraPublicaNoDisponible
	}
	var paquete struct {
		Fuente struct {
			Revision      string `json:"revision"`
			ActualizadaEn string `json:"actualizada_en"`
			Demostracion  bool   `json:"demostracion"`
			Aviso         string `json:"aviso"`
		} `json:"fuente"`
	}
	if json.Unmarshal(contenido, &paquete) != nil {
		return domain.EstructuraOrganizativaPublica{}, domain.ErrEstructuraPublicaNoDisponible
	}
	resultado := domain.EstructuraOrganizativaPublica{Esquema: "vec.personal.estructura-organizativa-publica.v1", CatalogoID: estructura.CatalogoID, CatalogoVersion: estructura.CatalogoVersion, CatalogoRevision: estructura.CatalogoRevision, FuenteRef: estructura.FuenteRef, Fuente: domain.FuenteEstructuraPublica{Revision: paquete.Fuente.Revision, ActualizadaEn: paquete.Fuente.ActualizadaEn, Demostracion: paquete.Fuente.Demostracion, Aviso: paquete.Fuente.Aviso, HuellaSHA256: HuellaPaqueteEstructura2026}, Unidades: make([]domain.UnidadEstructuraPublica, 0, len(estructura.Unidades))}
	for _, unidad := range estructura.Unidades {
		resultado.Unidades = append(resultado.Unidades, domain.UnidadEstructuraPublica{Clave: unidad.Clave, Etiqueta: unidad.Etiqueta, Tipo: unidad.Tipo, AdscripcionClave: unidad.AdscripcionClave})
	}
	if err := resultado.Validar(); err != nil {
		return domain.EstructuraOrganizativaPublica{}, domain.ErrEstructuraPublicaNoDisponible
	}
	return resultado, nil
}

var _ ports.ConsultaEstructuraOrganizativaPublica = (*Fuente)(nil)
