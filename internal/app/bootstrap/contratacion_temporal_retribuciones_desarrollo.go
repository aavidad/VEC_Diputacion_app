package bootstrap

import (
	"context"
	"errors"
	"math/big"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Las retribuciones de referencia con las que se estima el coste de un
// nombramiento se leen del catálogo ct.retribuciones (paquete de ejemplo en
// data/demo/reglas). Sin catálogo compuesto el coste queda «sin calcular»:
// ningún importe está fijado en el código.

var errRetribucionesCTNoValidas = errors.New("bootstrap: catalogo de retribuciones de contratacion temporal no valido")

const (
	maximoImporteMensualRetribucionDesarrollo = int64(10_000_000) // 100.000 € al mes
	maximoPagasRetribucionDesarrollo          = int64(24)
	maximaCuotaRetribucionDesarrollo          = int64(10_000) // 100 %
	maximoDiasCosteDesarrollo                 = int64(3_660)
)

var patronGrupoRetribucionDesarrollo = regexp.MustCompile(`^[A-Z][A-Z0-9/+.-]{0,19}$`)

// retribucionReferenciaDesarrollo es una fila del catálogo: importes
// mensuales a jornada completa en céntimos, número de pagas anuales de ese
// total y cuota empresarial de Seguridad Social en centésimas de punto.
type retribucionReferenciaDesarrollo struct {
	clave                     string
	grupo                     string
	categoriaRef              string
	sueldo                    int64
	complementos              int64
	pagas                     int64
	seguridadSocialCentesimas int64
	proporcionalJornada       bool
}

// fuenteRetribucionesDesarrollo relee el catálogo en cada cálculo para que una
// nueva versión publicada se aplique sin reiniciar. Un puntero nulo significa
// «sin catálogo».
type fuenteRetribucionesDesarrollo struct {
	resolutor *reglas.Resolutor
}

// nuevaFuenteRetribucionesDesarrollo compone el catálogo declarado y lo valida
// entero al arrancar: un paquete ilegible, ajeno al paquete de ejemplo o con
// una fila no válida impide arrancar en lugar de ignorarse.
func nuevaFuenteRetribucionesDesarrollo(ruta string, reloj reglas.Reloj) (*fuenteRetribucionesDesarrollo, error) {
	ruta = strings.TrimSpace(ruta)
	if ruta == "" {
		return nil, nil
	}
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		return nil, errors.Join(errRetribucionesCTNoValidas, err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: consulta,
		CatalogoID: reglas.CatalogoRetribucionesCT, ModuloID: reglas.ModuloContratacionTemporal,
		Reloj: reloj,
	})
	if err != nil {
		return nil, errors.Join(errRetribucionesCTNoValidas, err)
	}
	fuente := &fuenteRetribucionesDesarrollo{resolutor: resolutor}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	filas, ejemplo, err := fuente.tabla(ctx)
	if err != nil || len(filas) == 0 || !ejemplo {
		return nil, errors.Join(errRetribucionesCTNoValidas, err)
	}
	return fuente, nil
}

// tabla devuelve las filas vigentes y si el catálogo lleva la marca del
// paquete de ejemplo. Cualquier fila vigente no válida invalida la tabla.
func (f *fuenteRetribucionesDesarrollo) tabla(ctx context.Context) ([]retribucionReferenciaDesarrollo, bool, error) {
	if f == nil {
		return nil, false, reglas.ErrReglasNoConfiguradas
	}
	catalogo, _, instante, err := f.resolutor.CatalogoVigente(ctx)
	if err != nil {
		return nil, false, err
	}
	filas := make([]retribucionReferenciaDesarrollo, 0, len(catalogo.Entradas))
	grupos := map[string]bool{}
	categorias := map[string]bool{}
	for _, entrada := range catalogo.Entradas {
		if !entrada.VigenteEn(instante) {
			continue
		}
		fila, ok := retribucionDesdeEntradaDesarrollo(entrada)
		if !ok {
			return nil, false, errRetribucionesCTNoValidas
		}
		if fila.categoriaRef != "" {
			if categorias[fila.categoriaRef] {
				return nil, false, errRetribucionesCTNoValidas
			}
			categorias[fila.categoriaRef] = true
		} else {
			if grupos[fila.grupo] {
				return nil, false, errRetribucionesCTNoValidas
			}
			grupos[fila.grupo] = true
		}
		filas = append(filas, fila)
	}
	return filas, catalogo.FuenteRef == reglas.MarcaPaqueteEjemplo, nil
}

// retribucion elige la fila de la categoría y, si no existe, la de su grupo.
// La fila de una categoría solo vale si su grupo coincide con el del análisis.
// Sin catálogo, o sin fila, devuelve encontrada=false sin error: el coste queda
// «sin calcular». Un catálogo declarado pero no disponible devuelve el error.
func (f *fuenteRetribucionesDesarrollo) retribucion(
	ctx context.Context,
	categoriaRef, grupo string,
) (retribucionReferenciaDesarrollo, bool, error) {
	if f == nil {
		return retribucionReferenciaDesarrollo{}, false, nil
	}
	filas, _, err := f.tabla(ctx)
	if err != nil {
		return retribucionReferenciaDesarrollo{}, false, err
	}
	var porGrupo *retribucionReferenciaDesarrollo
	for indice := range filas {
		fila := &filas[indice]
		if fila.grupo != grupo {
			continue
		}
		if fila.categoriaRef != "" && fila.categoriaRef == categoriaRef {
			return *fila, true, nil
		}
		if fila.categoriaRef == "" {
			porGrupo = fila
		}
	}
	if porGrupo == nil {
		return retribucionReferenciaDesarrollo{}, false, nil
	}
	return *porGrupo, true, nil
}

func retribucionDesdeEntradaDesarrollo(entrada vecdomain.EntradaCatalogoConfigurable) (retribucionReferenciaDesarrollo, bool) {
	a := entrada.Atributos
	fila := retribucionReferenciaDesarrollo{
		clave: entrada.Clave, grupo: a["grupo"], categoriaRef: a["categoria_ref"],
	}
	var ok [4]bool
	fila.sueldo, ok[0] = enteroRetribucionDesarrollo(a["sueldo_mensual_centimos"], 0, maximoImporteMensualRetribucionDesarrollo)
	fila.complementos, ok[1] = enteroRetribucionDesarrollo(a["complementos_mensual_centimos"], 0, maximoImporteMensualRetribucionDesarrollo)
	fila.pagas, ok[2] = enteroRetribucionDesarrollo(a["pagas_anuales"], 1, maximoPagasRetribucionDesarrollo)
	fila.seguridadSocialCentesimas, ok[3] = enteroRetribucionDesarrollo(a["seguridad_social_centesimas"], 0, maximaCuotaRetribucionDesarrollo)
	switch a["proporcional_jornada"] {
	case "si":
		fila.proporcionalJornada = true
	case "no":
	default:
		return retribucionReferenciaDesarrollo{}, false
	}
	_, conCategoria := a["categoria_ref"]
	if !ok[0] || !ok[1] || !ok[2] || !ok[3] || fila.sueldo+fila.complementos <= 0 ||
		!patronGrupoRetribucionDesarrollo.MatchString(fila.grupo) ||
		(conCategoria && (fila.categoriaRef == "" || fila.categoriaRef != strings.TrimSpace(fila.categoriaRef))) {
		return retribucionReferenciaDesarrollo{}, false
	}
	return fila, true
}

// enteroRetribucionDesarrollo admite solo decimales canónicos (sin signo,
// espacios ni ceros a la izquierda) de hasta doce cifras, dentro del rango.
func enteroRetribucionDesarrollo(texto string, minimo, maximo int64) (int64, bool) {
	if texto == "" || len(texto) > 12 || (len(texto) > 1 && texto[0] == '0') {
		return 0, false
	}
	var valor int64
	for _, cifra := range []byte(texto) {
		if cifra < '0' || cifra > '9' {
			return 0, false
		}
		valor = valor*10 + int64(cifra-'0')
	}
	return valor, valor >= minimo && valor <= maximo
}

// costeEstimadoAnalisisDesarrollo calcula el coste empresa del periodo:
//
//	mensual = (sueldo + complementos) × pagas ÷ 12 × (1 + cuota)
//	coste   = mensual × días naturales ÷ 30,4375 × jornada
//
// con los días del periodo ambos inclusive y la jornada en diezmilésimas (solo
// si la fila es proporcional). Se opera con enteros exactos y se redondea una
// sola vez al céntimo, mitad hacia arriba. Devuelve false si el periodo no es
// válido o el resultado no es positivo: el coste queda «sin calcular».
func costeEstimadoAnalisisDesarrollo(
	fila retribucionReferenciaDesarrollo,
	periodo domain.PeriodoPrevisto,
	jornada domain.JornadaDiezmilesimas,
) (domain.Importe, bool) {
	if jornada == 0 || jornada > domain.JornadaCompletaDiezmilesimas || periodo.Fin.Before(periodo.Inicio) {
		return domain.Importe{}, false
	}
	dias := int64(periodo.Fin.Sub(periodo.Inicio).Hours()/24) + 1
	if dias <= 0 || dias > maximoDiasCosteDesarrollo {
		return domain.Importe{}, false
	}
	fraccion := int64(domain.JornadaCompletaDiezmilesimas)
	if fila.proporcionalJornada {
		fraccion = int64(jornada)
	}
	numerador := new(big.Int).SetInt64(fila.sueldo + fila.complementos)
	for _, factor := range []int64{fila.pagas, 10_000 + fila.seguridadSocialCentesimas, dias, fraccion} {
		numerador.Mul(numerador, big.NewInt(factor))
	}
	// 12 meses × 10 000 (cuota) × 304 375 (30,4375 días × 10 000 de jornada).
	divisor := big.NewInt(12 * 10_000 * diasMesDiezmilesimasCosteDesarrollo)
	// redondeo = ⌊(2·n + d) ÷ 2·d⌋
	numerador.Mul(numerador, big.NewInt(2)).Add(numerador, divisor)
	centimos := numerador.Quo(numerador, divisor.Mul(divisor, big.NewInt(2)))
	if !centimos.IsInt64() || centimos.Int64() <= 0 {
		return domain.Importe{}, false
	}
	return domain.Importe{Centimos: centimos.Int64(), Moneda: "EUR"}, true
}
