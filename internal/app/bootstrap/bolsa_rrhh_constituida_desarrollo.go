package bootstrap

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/config"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	importacionpg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgresimportacionconvoca"
	protector "vec-diputacion-granada/internal/modules/bolsa/adapters/protectorstagingdesarrollo"
	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// fuenteConstituidaRRHHDesarrollo sirve las lecturas RRHH de bolsas desde las
// bolsas constituidas (migración 000007) cuando existen. Para cada categoría
// constituida sustituye la bolsa del dataset sintético; el resto sigue viniendo
// del dataset. Los nombres y documentos enmascarados no se almacenan en claro:
// se recuperan del staging protegido del acta al leer, y se cachean brevemente.
type fuenteConstituidaRRHHDesarrollo struct {
	repositorio ports.RepositorioConstitucion
	recuperador constitucion.Recuperador
	categorias  map[string]string
	grupos      map[string][]string
	ahora       func() time.Time

	mu       sync.Mutex
	cache    datasetBolsasRRHHDesarrollo
	cacheada bool
	hasta    time.Time
}

const validezCacheBolsasConstituidas = 30 * time.Second

func nuevaFuenteConstituidaRRHHDesarrollo(ctx context.Context, cfg config.Config) *fuenteConstituidaRRHHDesarrollo {
	if ctx == nil || !cfg.DevelopmentEnabledByDoubleKey() {
		return nil
	}
	poolBolsa, err := abrirBolsaLlamamientosPostgreSQLDesarrollo(ctx, cfg.ContratacionTemporalPostgreSQL)
	if err != nil {
		log.Printf("bolsa rrhh: bolsas constituidas no disponibles; etapa=pool_bolsa")
		return nil
	}
	repositorio, err := postgresbolsa.NuevoRepositorioConstitucionPostgreSQL(poolBolsa)
	if err != nil {
		poolBolsa.Close()
		return nil
	}
	material, err := cargarMaterialSeguridadDesarrollo(cfg)
	if err != nil {
		poolBolsa.Close()
		log.Printf("bolsa rrhh: bolsas constituidas no disponibles; etapa=material")
		return nil
	}
	defer borrarMaterialImportacionConvoca(material)
	p, err := protector.Nuevo(material.claveKMS)
	if err != nil {
		poolBolsa.Close()
		return nil
	}
	poolImportacion, err := abrirPoolImportacionConvoca(ctx, cfg)
	if err != nil {
		poolBolsa.Close()
		log.Printf("bolsa rrhh: bolsas constituidas no disponibles; etapa=pool_importacion")
		return nil
	}
	recuperador, err := importacionpg.NuevoRepositorioRecuperacionPostgreSQL(poolImportacion, p)
	if err != nil {
		poolBolsa.Close()
		poolImportacion.Close()
		return nil
	}
	categorias, grupos := map[string]string{}, map[string][]string{}
	if lista, err := cargarCategoriasRPTDesarrollo(cfg.RPTCatalogoPath); err == nil {
		for _, categoria := range lista {
			categorias[categoria.Referencia] = categoria.Etiqueta
			for _, grupo := range categoria.GruposSubgrupos {
				grupos[categoria.Referencia] = append(grupos[categoria.Referencia], grupo.Clave)
			}
		}
	}
	return &fuenteConstituidaRRHHDesarrollo{repositorio: repositorio, recuperador: recuperador, categorias: categorias, grupos: grupos, ahora: time.Now}
}

// fusionar devuelve el dataset base con las categorías constituidas
// sustituidas por su bolsa constituida y sus participaciones.
func (f *fuenteConstituidaRRHHDesarrollo) fusionar(ctx context.Context, base datasetBolsasRRHHDesarrollo) datasetBolsasRRHHDesarrollo {
	if f == nil {
		return base
	}
	constituidas, ok := f.constituidas(ctx)
	if !ok || len(constituidas.Bolsas) == 0 {
		return base
	}
	sustituidas := map[string]struct{}{}
	for _, bolsa := range constituidas.Bolsas {
		sustituidas[bolsa.CategoriaRef] = struct{}{}
	}
	resultado := datasetBolsasRRHHDesarrollo{GeneradoEn: constituidas.GeneradoEn}
	retiradas := map[string]struct{}{}
	for _, bolsa := range base.Bolsas {
		if _, sustituida := sustituidas[bolsa.CategoriaRef]; sustituida {
			retiradas[bolsa.Referencia] = struct{}{}
			continue
		}
		resultado.Bolsas = append(resultado.Bolsas, bolsa)
	}
	candidaturasRetiradas := map[string]struct{}{}
	for _, candidatura := range base.Candidaturas {
		if _, retirada := retiradas[candidatura.BolsaRef]; retirada {
			candidaturasRetiradas[candidatura.Referencia] = struct{}{}
			continue
		}
		resultado.Candidaturas = append(resultado.Candidaturas, candidatura)
	}
	for _, llamamiento := range base.Llamamientos {
		if _, retirada := candidaturasRetiradas[llamamiento.Candidatura]; retirada {
			continue
		}
		resultado.Llamamientos = append(resultado.Llamamientos, llamamiento)
	}
	resultado.Bolsas = append(resultado.Bolsas, constituidas.Bolsas...)
	resultado.Candidaturas = append(resultado.Candidaturas, constituidas.Candidaturas...)
	return resultado
}

func (f *fuenteConstituidaRRHHDesarrollo) constituidas(ctx context.Context) (datasetBolsasRRHHDesarrollo, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ahora := f.ahora()
	if f.cacheada && ahora.Before(f.hasta) {
		return f.cache, true
	}
	datos, err := f.cargar(ctx)
	if err != nil {
		log.Printf("bolsa rrhh: bolsas constituidas no legibles: %v", err)
		if f.cacheada {
			return f.cache, true
		}
		return datasetBolsasRRHHDesarrollo{}, false
	}
	f.cache, f.cacheada, f.hasta = datos, true, ahora.Add(validezCacheBolsasConstituidas)
	return datos, true
}

func (f *fuenteConstituidaRRHHDesarrollo) cargar(ctx context.Context) (datasetBolsasRRHHDesarrollo, error) {
	vigentes, err := f.repositorio.ListarVigentes(ctx)
	if err != nil {
		return datasetBolsasRRHHDesarrollo{}, err
	}
	datos := datasetBolsasRRHHDesarrollo{GeneradoEn: f.ahora().UTC().Format(time.RFC3339)}
	for _, vigente := range vigentes {
		desde := vigente.ConfirmadaEn.UTC().Format(time.RFC3339)
		datos.Bolsas = append(datos.Bolsas, struct {
			Referencia   string  `json:"bolsa_ref"`
			CategoriaRef string  `json:"categoria_ref"`
			Categoria    string  `json:"categoria"`
			TipoLista    string  `json:"tipo_lista"`
			VigenteDesde string  `json:"vigente_desde"`
			VigenteHasta *string `json:"vigente_hasta"`
		}{
			Referencia: vigente.Bolsa.BolsaRef, CategoriaRef: vigente.CategoriaRef,
			Categoria: f.denominacion(vigente.CategoriaRef), TipoLista: "definitiva", VigenteDesde: desde,
		})
		entradas, err := f.repositorio.Entradas(ctx, vigente.Instantanea.InstantaneaRef, vigente.Instantanea.Version)
		if err != nil {
			return datasetBolsasRRHHDesarrollo{}, err
		}
		lote, _, existe, err := f.recuperador.RecuperarLote(ctx, vigente.Bolsa.HuellaListadoSHA256, vigente.CategoriaRef)
		if err != nil {
			return datasetBolsasRRHHDesarrollo{}, err
		}
		filas := map[int]struct{ nombre, documento string }{}
		if existe {
			for _, fila := range lote.Aceptadas {
				nombre := strings.TrimSpace(strings.Join([]string{fila.Identidad.Nombre, fila.Identidad.PrimerApellido, fila.Identidad.SegundoApellido}, " "))
				filas[fila.Numero] = struct{ nombre, documento string }{strings.Join(strings.Fields(nombre), " "), fila.Identidad.Documento}
			}
		}
		for _, entrada := range entradas {
			visible := filas[entrada.FilaNumero]
			if visible.nombre == "" {
				visible.nombre = "Participación " + entrada.ParticipacionRef[len(entrada.ParticipacionRef)-6:]
			}
			datos.Candidaturas = append(datos.Candidaturas, struct {
				Referencia  string  `json:"candidatura_ref"`
				BolsaRef    string  `json:"bolsa_ref"`
				Orden       int     `json:"orden"`
				Nombre      string  `json:"nombre_visible"`
				Documento   string  `json:"documento_enmascarado"`
				Estado      string  `json:"estado_clave"`
				EstadoDesde string  `json:"estado_desde"`
				Disponible  *string `json:"disponible_desde"`
			}{
				Referencia: entrada.ParticipacionRef, BolsaRef: vigente.Bolsa.BolsaRef, Orden: int(entrada.Orden),
				Nombre: visible.nombre, Documento: visible.documento, Estado: constitucion.EstadoInicial, EstadoDesde: desde,
			})
		}
	}
	return datos, nil
}

func (f *fuenteConstituidaRRHHDesarrollo) denominacion(categoriaRef string) string {
	if etiqueta, ok := f.categorias[categoriaRef]; ok && etiqueta != "" {
		return etiqueta
	}
	return strings.ToUpper(strings.ReplaceAll(strings.TrimPrefix(categoriaRef, "categoria:rpt:"), "-", " "))
}
