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

// fuenteConstituidaRRHHDesarrollo sirve las lecturas RRHH exclusivamente desde
// bolsas constituidas (migración 000007). Los nombres y documentos enmascarados
// no se almacenan en claro: se recuperan del staging protegido del acta al leer.
type fuenteConstituidaRRHHDesarrollo struct {
	repositorio ports.RepositorioConstitucion
	situaciones ports.RepositorioSituacionParticipacion
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
	situaciones, err := postgresbolsa.NuevoRepositorioSituacionParticipacionPostgreSQL(poolBolsa)
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
	return &fuenteConstituidaRRHHDesarrollo{repositorio: repositorio, situaciones: situaciones, recuperador: recuperador, categorias: categorias, grupos: grupos, ahora: time.Now}
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
	if f == nil || f.repositorio == nil || f.situaciones == nil || f.recuperador == nil || f.ahora == nil {
		return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
	}
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
		if !existe {
			return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
		}
		filas := map[int]struct{ nombre, documento string }{}
		for _, fila := range lote.Aceptadas {
			nombre := strings.TrimSpace(strings.Join([]string{fila.Identidad.Nombre, fila.Identidad.PrimerApellido, fila.Identidad.SegundoApellido}, " "))
			filas[fila.Numero] = struct{ nombre, documento string }{strings.Join(strings.Fields(nombre), " "), fila.Identidad.Documento}
		}
		for _, entrada := range entradas {
			visible, encontrada := filas[entrada.FilaNumero]
			if !encontrada {
				return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
			}
			situacion, err := f.situaciones.SituacionVigente(ctx, entrada.ParticipacionRef)
			if err != nil {
				return datasetBolsasRRHHDesarrollo{}, err
			}
			var disponible *string
			if situacion.FechaDisponible != nil {
				valor := situacion.FechaDisponible.UTC().Format(time.RFC3339)
				disponible = &valor
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
				Nombre: visible.nombre, Documento: visible.documento, Estado: situacion.Situacion, EstadoDesde: situacion.Desde.UTC().Format(time.RFC3339), Disponible: disponible,
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
