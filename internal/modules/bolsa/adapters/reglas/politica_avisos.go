package reglas

import (
	"context"
	"errors"
	"strconv"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecreglas "vec-diputacion-granada/internal/vec/reglas"
)

// Atributos del catálogo que usan los avisos y marcas de Bolsa 000041.
const (
	atributoAntelacionDias = "antelacion_dias"
	atributoVentanaMeses   = "ventana_meses"
	atributoModo           = "modo"
	atributoSituaciones    = "situaciones"
	// entradaPoliticaAvisos nombra en la referencia publicada el conjunto
	// b16, b17 y b19 de una misma versión del catálogo.
	entradaPoliticaAvisos = "avisos"
)

var errReglasAvisos = errors.New("bolsa: reglas de avisos del catalogo incompletas")

// PoliticaAvisos compone los parámetros de avisos y marcas del catálogo:
//   - b19 (duración máxima en vacante, en años o meses) con antelacion_dias:
//     plazo y antelación del aviso de trabajo continuado;
//   - b17 (meses) con ventana_meses: umbral y ventana del encadenamiento;
//   - b16 con modo (aviso o excluir) y situaciones: ya presta servicios.
//
// Una regla ausente, o sin los atributos que la hacen operativa, no se
// configura y rige lo que la base ya tenga. hay=false si no hay ninguna; un
// atributo mal formado es un error, para no ignorar la configuración.
func PoliticaAvisos(ctx context.Context, resolutor *vecreglas.Resolutor) (puertosbolsa.PublicacionPoliticaAvisos, bool, error) {
	var vacia puertosbolsa.PublicacionPoliticaAvisos
	if ctx == nil || !resolutor.Disponible() {
		return vacia, false, nil
	}
	todas, err := resolutor.Reglas(ctx)
	if errors.Is(err, vecreglas.ErrReglasNoConfiguradas) {
		return vacia, false, nil
	}
	if err != nil {
		return vacia, false, err
	}
	porClave := make(map[string]vecreglas.Regla, len(todas))
	for _, regla := range todas {
		porClave[regla.Clave] = regla
	}
	var politica dominiobolsa.PoliticaAvisosBolsa
	var base *vecreglas.Regla
	if regla, ok := porClave[vecreglas.BolsaVacanteDuracionMaxima]; ok {
		if texto, configurada := regla.Atributos[atributoAntelacionDias]; configurada {
			meses := regla.Cantidad
			switch regla.Unidad {
			case vecreglas.UnidadAnios:
				meses *= 12
			case vecreglas.UnidadMeses:
			default:
				return vacia, false, errReglasAvisos
			}
			antelacion, ok := enteroAtributo(texto, 0)
			if !ok {
				return vacia, false, errReglasAvisos
			}
			politica.ContinuadoConfigurado, politica.ContinuadoMeses, politica.ContinuadoAntelacionDias = true, meses, antelacion
			base = &regla
		}
	}
	if regla, ok := porClave[vecreglas.BolsaAvisoEncadenamiento]; ok {
		if texto, configurada := regla.Atributos[atributoVentanaMeses]; configurada {
			ventana, ok := enteroAtributo(texto, 1)
			if !ok || regla.Unidad != vecreglas.UnidadMeses {
				return vacia, false, errReglasAvisos
			}
			politica.EncadenamientoUmbralMeses, politica.EncadenamientoVentanaMeses = regla.Cantidad, ventana
			if base == nil {
				base = &regla
			}
		}
	}
	if regla, ok := porClave[vecreglas.BolsaPrestaServicios]; ok {
		modo, conModo := regla.Atributos[atributoModo]
		situaciones, conSituaciones := regla.Atributos[atributoSituaciones]
		if conModo != conSituaciones {
			return vacia, false, errReglasAvisos
		}
		if conModo {
			politica.PrestaServiciosModo, politica.PrestaServiciosSituaciones = modo, listaAtributo(situaciones)
			if base == nil {
				base = &regla
			}
		}
	}
	if base == nil {
		return vacia, false, nil
	}
	if err := politica.Validar(); err != nil {
		return vacia, false, errors.Join(errReglasAvisos, err)
	}
	// Todas las entradas vienen de la misma versión del catálogo.
	entrada := base.ReferenciaEntrada
	entrada.EntradaClave = entradaPoliticaAvisos
	return puertosbolsa.PublicacionPoliticaAvisos{CatalogoRef: entrada.Referencia(), CatalogoSHA256: base.HuellaCatalogo, Politica: politica}, true, nil
}

// enteroAtributo admite un entero canónico no menor que minimo.
func enteroAtributo(texto string, minimo int) (int, bool) {
	valor, err := strconv.Atoi(texto)
	if err != nil || valor < minimo || valor > 100_000 || strconv.Itoa(valor) != texto {
		return 0, false
	}
	return valor, true
}
