package bootstrap

import (
	"context"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Documentación de la formalización (dudas 9 y 18). Las reglas proceden del
// catálogo de Bolsa porque es el Reglamento de bolsas el que fija el plazo y
// la documentación tras aceptar una oferta; el tipo documental de cada
// documento es el que el catálogo de conservación admite para anotarlo en
// Documentos. Sin catálogo de reglas la ruta responde reglas_no_configuradas.
const (
	prefijoTipoDocumentalFormalizacion = "contratacion_temporal.formalizacion."
	sufijoTipoDocumentalFormalizacion  = ".v1"
	prefijoModalidadReglaFormalizacion = "valor_"
)

func rutaDocumentacionFormalizacionDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaDocumentacionFormalizacion
}

func nuevaRutaDocumentacionFormalizacionDesarrollo(resolutor *reglas.Resolutor) (vechttp.RutaExacta, error) {
	reloj := relojCalendariosDesarrollo{}
	tipos, err := conservacion.NuevoCatalogoProvisional(reloj)
	if err != nil {
		return vechttp.RutaExacta{}, err
	}
	servicio, err := ctapp.NuevoServicioDocumentacionFormalizacion(ctapp.ConfiguracionDocumentacionFormalizacion{
		Reglas: reglasFormalizacionDesarrollo{resolutor: resolutor}, Tipos: tiposDocumentalesFormalizacion{catalogo: tipos},
		Reloj: reloj, ClavePlazoDocumentacion: reglas.BolsaPlazoDocumentacion, ClaveDocumentos: reglas.BolsaDocumentosIncorporacion,
		ClavePlazoIncorporacion: reglas.BolsaPlazoIncorporacion,
		PrefijoTipoDocumental:   prefijoTipoDocumentalFormalizacion, SufijoTipoDocumental: sufijoTipoDocumentalFormalizacion,
	})
	if err != nil {
		return vechttp.RutaExacta{}, err
	}
	manejador, err := httpinterno.NuevoManejadorDocumentacionFormalizacion(servicio)
	if err != nil {
		return vechttp.RutaExacta{}, err
	}
	return vechttp.RutaExacta{Ruta: httpinterno.RutaDocumentacionFormalizacion, Manejador: manejador}, nil
}

// reglasFormalizacionDesarrollo traduce el resolutor común al puerto neutral
// de Contratación temporal. Un resolutor nulo significa «sin catálogo».
type reglasFormalizacionDesarrollo struct {
	resolutor *reglas.Resolutor
}

func (a reglasFormalizacionDesarrollo) ReglaFormalizacion(ctx context.Context, clave string) (ctports.ReglaFormalizacion, error) {
	regla, err := a.resolutor.Regla(ctx, clave)
	if err != nil {
		return ctports.ReglaFormalizacion{}, errorReglasFormalizacion(err)
	}
	return copiarReglaFormalizacion(regla), nil
}

func (a reglasFormalizacionDesarrollo) VencimientoFormalizacion(ctx context.Context, clave string, inicio time.Time) (ctports.ReglaFormalizacion, ctports.VencimientoFormalizacion, error) {
	regla, vencimiento, err := a.resolutor.Vencimiento(ctx, clave, inicio, "")
	if err != nil {
		return ctports.ReglaFormalizacion{}, ctports.VencimientoFormalizacion{}, errorReglasFormalizacion(err)
	}
	return copiarReglaFormalizacion(regla), ctports.VencimientoFormalizacion{
		UltimoDia: vencimiento.UltimoDia, VenceAntesDe: vencimiento.VenceAntesDe, Prorrogado: vencimiento.Prorrogado,
	}, nil
}

func errorReglasFormalizacion(err error) error {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, reglas.ErrReglasNoConfiguradas):
		return ctports.ErrReglasFormalizacionNoConfiguradas
	default:
		return ctports.ErrReglasFormalizacionNoDisponibles
	}
}

// copiarReglaFormalizacion conserva la procedencia exacta. Una regla de lista
// puede distinguir modalidades con atributos valor_<modalidad>.
func copiarReglaFormalizacion(r reglas.Regla) ctports.ReglaFormalizacion {
	copia := ctports.ReglaFormalizacion{
		Clave: r.Clave, Unidad: string(r.Unidad), Cantidad: r.Cantidad, Computo: string(r.Computo),
		Elementos: r.Elementos(), Origen: string(r.Origen), Ejemplo: r.EsEjemplo(),
		Articulo: r.Articulo, Norma: r.Norma, Duda: r.Duda, ParteEjemplo: r.ParteEjemplo,
		Referencia: r.Referencia, HuellaCatalogo: r.HuellaCatalogo,
	}
	if r.Unidad != reglas.UnidadLista {
		return copia
	}
	for atributo, valor := range r.Atributos {
		modalidad, ok := strings.CutPrefix(atributo, prefijoModalidadReglaFormalizacion)
		if !ok || modalidad == "" || valor == "" {
			continue
		}
		if copia.ElementosPorModalidad == nil {
			copia.ElementosPorModalidad = map[string][]string{}
		}
		copia.ElementosPorModalidad[modalidad] = strings.Split(valor, ",")
	}
	return copia
}

type tiposDocumentalesFormalizacion struct {
	catalogo *conservacion.Catalogo
}

func (t tiposDocumentalesFormalizacion) TipoDocumentalCatalogado(tipo string) bool {
	_, err := t.catalogo.TipoDocumentalRef(tipo)
	return err == nil
}
