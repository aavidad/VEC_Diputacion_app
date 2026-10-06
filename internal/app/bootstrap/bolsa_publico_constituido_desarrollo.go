package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"time"

	bolsapublico "vec-diputacion-granada/internal/modules/bolsa/publico/httpapi"
)

// fuenteBolsasPublicasDesarrollo proyecta las bolsas constituidas (B1→B2) a la
// consulta pública B10: solo bolsas reales, y de cada aspirante únicamente el
// orden, el documento enmascarado y la situación. Los nombres que la fuente
// RRHH recupera del staging no salen de aquí.
// concurrenciaBolsasPublicasDesarrollo acota las lecturas públicas en vivo.
const concurrenciaBolsasPublicasDesarrollo = 4

type fuenteBolsasPublicasDesarrollo struct {
	fuente *fuenteConstituidaRRHHDesarrollo
}

var _ bolsapublico.FuenteBolsasPublicas = (*fuenteBolsasPublicasDesarrollo)(nil)

func nuevoManejadorBolsasPublicasDesarrollo(fuente *fuenteConstituidaRRHHDesarrollo) (http.Handler, error) {
	if fuente == nil {
		return nil, nil
	}
	// La fuente calcula cada consulta en vivo con las conexiones de RRHH:
	// tope de lecturas simultáneas (429 con Retry-After al llenarse).
	return bolsapublico.NuevoManejadorBolsasPublicasConTope(&fuenteBolsasPublicasDesarrollo{fuente: fuente}, concurrenciaBolsasPublicasDesarrollo)
}

func (f *fuenteBolsasPublicasDesarrollo) BolsasPublicas(ctx context.Context) ([]bolsapublico.BolsaPublica, time.Time, error) {
	// El listado solo cuenta personas por bolsa: el resumen, sin descifrar
	// el acta protegida.
	datos, generadoEn, err := f.datos(ctx, "", func(c context.Context) (datasetBolsasRRHHDesarrollo, error) {
		return f.fuente.cargarResumen(c)
	})
	if err != nil {
		return nil, time.Time{}, err
	}
	totales := map[string]int{}
	for _, candidatura := range datos.Candidaturas {
		totales[candidatura.BolsaRef]++
	}
	bolsas := make([]bolsapublico.BolsaPublica, 0, len(datos.Bolsas))
	for _, bolsa := range datos.Bolsas {
		publica, err := f.proyectarBolsa(bolsa, totales[bolsa.Referencia])
		if err != nil {
			return nil, time.Time{}, err
		}
		bolsas = append(bolsas, publica)
	}
	sort.SliceStable(bolsas, func(i, j int) bool { return bolsas[i].Categoria < bolsas[j].Categoria })
	return bolsas, generadoEn, nil
}

func (f *fuenteBolsasPublicasDesarrollo) ListaPublica(ctx context.Context, bolsaRef string) (bolsapublico.BolsaPublica, []bolsapublico.PosicionPublica, time.Time, error) {
	// La lista de una bolsa lee solo esa bolsa (orden y documento enmascarado).
	datos, generadoEn, err := f.datos(ctx, bolsaRef, func(c context.Context) (datasetBolsasRRHHDesarrollo, error) {
		return f.fuente.cargarBolsa(c, bolsaRef)
	})
	if err != nil {
		return bolsapublico.BolsaPublica{}, nil, time.Time{}, err
	}
	posiciones := make([]bolsapublico.PosicionPublica, 0, 64)
	for _, candidatura := range datos.Candidaturas {
		if candidatura.BolsaRef == bolsaRef {
			posiciones = append(posiciones, bolsapublico.PosicionPublica{
				Orden: candidatura.OrdenActa, DocumentoEnmascarado: candidatura.Documento,
				EstadoClave: estadoBolsaPublico(candidatura.Estado, candidatura.Disponible, generadoEn),
			})
		}
	}
	sort.SliceStable(posiciones, func(i, j int) bool { return posiciones[i].Orden < posiciones[j].Orden })
	for _, bolsa := range datos.Bolsas {
		if bolsa.Referencia == bolsaRef {
			publica, err := f.proyectarBolsa(bolsa, len(posiciones))
			if err != nil {
				return bolsapublico.BolsaPublica{}, nil, time.Time{}, err
			}
			return publica, posiciones, generadoEn, nil
		}
	}
	return bolsapublico.BolsaPublica{}, nil, time.Time{}, bolsapublico.ErrBolsaPublicaNoEncontrada
}

func estadoBolsaPublico(estado string, disponibleDesde *string, generadoEn time.Time) string {
	switch estado {
	case "trabajando", "pendiente_incorporacion":
		return "ocupado"
	case "renuncia":
		return "renuncia_pendiente"
	case "disponible_desde":
		if disponibleDesde != nil {
			if desde, err := time.Parse(time.RFC3339, *disponibleDesde); err == nil && !desde.After(generadoEn) {
				return "disponible"
			}
		}
		return "no_disponible"
	case "disponible", "no_disponible", "excluido":
		return estado
	default:
		return "no_disponible"
	}
}

// datos lee el conjunto de la petición dentro de su propio plazo. No hay
// caché: cada petición lee el estado vigente. Un fallo deja su causa en el
// registro y se devuelve tal cual, para que un plazo agotado responda 504 y
// no un 503 genérico.
func (f *fuenteBolsasPublicasDesarrollo) datos(ctx context.Context, bolsaRef string, cargar func(context.Context) (datasetBolsasRRHHDesarrollo, error)) (datasetBolsasRRHHDesarrollo, time.Time, error) {
	if f == nil || f.fuente == nil || ctx == nil || cargar == nil {
		return datasetBolsasRRHHDesarrollo{}, time.Time{}, ErrComposicionDesarrolloIncompleta
	}
	if err := ctx.Err(); err != nil {
		return datasetBolsasRRHHDesarrollo{}, time.Time{}, err
	}
	datos, err := cargar(ctx)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("bolsa pública: lectura no disponible", "bolsa", bolsaRef, "causa", causaFalloBolsaRRHHDesarrollo(err))
		}
		return datasetBolsasRRHHDesarrollo{}, time.Time{}, err
	}
	generadoEn, err := time.Parse(time.RFC3339, datos.GeneradoEn)
	if err != nil {
		return datasetBolsasRRHHDesarrollo{}, time.Time{}, ErrComposicionDesarrolloIncompleta
	}
	return datos, generadoEn, nil
}

func (f *fuenteBolsasPublicasDesarrollo) proyectarBolsa(bolsa struct {
	Referencia          string                      `json:"bolsa_ref"`
	CategoriaRef        string                      `json:"categoria_ref"`
	Categoria           string                      `json:"categoria"`
	TipoLista           string                      `json:"tipo_lista"`
	VigenteDesde        string                      `json:"vigente_desde"`
	VigenteHasta        *string                     `json:"vigente_hasta"`
	LlamamientosEnCurso int                         `json:"llamamientos_en_curso"`
	PoliticaOrden       politicaOrdenRRHHDesarrollo `json:"politica_orden"`
}, total int) (bolsapublico.BolsaPublica, error) {
	desde, err := time.Parse(time.RFC3339, bolsa.VigenteDesde)
	if err != nil {
		return bolsapublico.BolsaPublica{}, ErrComposicionDesarrolloIncompleta
	}
	publica := bolsapublico.BolsaPublica{
		BolsaRef: bolsa.Referencia, Categoria: bolsa.Categoria, Grupos: append([]string(nil), f.fuente.grupos[bolsa.CategoriaRef]...),
		TipoLista: bolsa.TipoLista, VigenteDesde: desde, Total: total,
	}
	if bolsa.VigenteHasta != nil {
		hasta, err := time.Parse(time.RFC3339, *bolsa.VigenteHasta)
		if err != nil {
			return bolsapublico.BolsaPublica{}, ErrComposicionDesarrolloIncompleta
		}
		publica.VigenteHasta = &hasta
	}
	return publica, nil
}
