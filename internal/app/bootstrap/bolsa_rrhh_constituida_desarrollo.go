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
	bolsaapplication "vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// fuenteConstituidaRRHHDesarrollo sirve las lecturas RRHH exclusivamente desde
// bolsas constituidas (migración 000007). Los nombres y documentos enmascarados
// no se almacenan en claro: se recuperan del staging protegido del acta al leer.
type fuenteConstituidaRRHHDesarrollo struct {
	repositorio ports.RepositorioConstitucion
	situaciones ports.RepositorioSituacionParticipacion
	orden       *bolsaapplication.ServicioOrdenVigente
	avisos      *bolsaapplication.ServicioAvisosRRHH
	// Bolsa 000041: bandeja con parámetros del catálogo, publicación de la
	// política y marcas por participación (bolsa_parametros_avisos_desarrollo.go).
	consultaAvisos *postgresbolsa.ConsultaAvisosRRHHPostgreSQL
	parametros     *postgresbolsa.RepositorioPoliticaAvisosPostgreSQL
	marcas         ports.ConsultaMarcasParticipaciones
	intentos       ports.PoliticaIntentosContacto
	emisiones      *postgresbolsa.RepositorioEmisionLlamamientoPostgreSQL
	recuperador    constitucion.Recuperador
	categorias     map[string]string
	grupos         map[string][]string
	ahora          func() time.Time

	mu       sync.Mutex
	cache    datasetBolsasRRHHDesarrollo
	cacheada bool
	hasta    time.Time
}

const validezCacheBolsasConstituidas = 30 * time.Second

func (f *fuenteConstituidaRRHHDesarrollo) invalidar() {
	if f == nil {
		return
	}
	f.mu.Lock()
	f.cacheada = false
	f.mu.Unlock()
}

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
	consultaOrden, err := postgresbolsa.NuevaConsultaOrdenVigentePostgreSQL(poolBolsa)
	if err != nil {
		poolBolsa.Close()
		return nil
	}
	orden, err := bolsaapplication.NuevoServicioOrdenVigente(consultaOrden)
	if err != nil {
		poolBolsa.Close()
		return nil
	}
	nuevaConsultaAvisos := postgresbolsa.NuevaConsultaAvisosRRHHPostgreSQL
	if debeComponerPortalCandidatoDesarrollo(cfg) {
		nuevaConsultaAvisos = postgresbolsa.NuevaConsultaAvisosRRHHConPortalPostgreSQL
	}
	consultaAvisos, err := nuevaConsultaAvisos(poolBolsa)
	if err != nil {
		poolBolsa.Close()
		return nil
	}
	avisos, err := bolsaapplication.NuevoServicioAvisosRRHH(consultaAvisos, time.Now)
	if err != nil {
		poolBolsa.Close()
		return nil
	}
	emisiones, err := postgresbolsa.NuevoRepositorioEmisionLlamamientoPostgreSQL(poolBolsa)
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
	parametros, err := postgresbolsa.NuevoRepositorioPoliticaAvisosPostgreSQL(poolBolsa)
	if err != nil {
		log.Printf("bolsa rrhh: bolsas constituidas no disponibles; etapa=parametros_avisos: %v", err)
		poolBolsa.Close()
		poolImportacion.Close()
		return nil
	}
	return &fuenteConstituidaRRHHDesarrollo{repositorio: repositorio, situaciones: situaciones, orden: orden, avisos: avisos, consultaAvisos: consultaAvisos, parametros: parametros, emisiones: emisiones, recuperador: recuperador, categorias: categorias, grupos: grupos, ahora: time.Now}
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
	if f == nil || f.repositorio == nil || f.situaciones == nil || f.orden == nil || f.emisiones == nil || f.recuperador == nil || f.ahora == nil {
		return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
	}
	vigentes, err := f.repositorio.ListarVigentes(ctx)
	if err != nil {
		return datasetBolsasRRHHDesarrollo{}, err
	}
	datos := datasetBolsasRRHHDesarrollo{GeneradoEn: f.ahora().UTC().Format(time.RFC3339)}
	if err := f.cargarMarcasBase(ctx, &datos); err != nil {
		return datasetBolsasRRHHDesarrollo{}, err
	}
	for _, vigente := range vigentes {
		ordenVigente, err := f.orden.Consultar(ctx, vigente.Bolsa.BolsaRef)
		if err != nil {
			return datasetBolsasRRHHDesarrollo{}, err
		}
		posiciones := make(map[string]struct {
			acta    int
			vigente *int
			razon   string
		}, len(ordenVigente.Posiciones))
		for _, posicion := range ordenVigente.Posiciones {
			var actual *int
			if posicion.OrdenVigente != nil {
				v := int(*posicion.OrdenVigente)
				actual = &v
			}
			posiciones[posicion.ParticipacionRef] = struct {
				acta    int
				vigente *int
				razon   string
			}{int(posicion.OrdenActa), actual, posicion.Razon}
		}
		politica := politicaOrdenRRHHDesarrollo{Referencia: ordenVigente.Politica.PoliticaRef, Criterio: ordenVigente.Politica.Criterio, TipoLista: ordenVigente.Politica.TipoLista, Reposicion: ordenVigente.Politica.Reposicion, Rotulo: ordenVigente.Politica.Rotulo, Actor: ordenVigente.Politica.Actor, VigenteDesde: ordenVigente.Politica.VigenteDesde.UTC().Format(time.RFC3339), Version: ordenVigente.Politica.Version, Provisional: ordenVigente.Politica.Provisional}
		desde := vigente.ConfirmadaEn.UTC().Format(time.RFC3339)
		datos.Bolsas = append(datos.Bolsas, struct {
			Referencia          string                      `json:"bolsa_ref"`
			CategoriaRef        string                      `json:"categoria_ref"`
			Categoria           string                      `json:"categoria"`
			TipoLista           string                      `json:"tipo_lista"`
			VigenteDesde        string                      `json:"vigente_desde"`
			VigenteHasta        *string                     `json:"vigente_hasta"`
			LlamamientosEnCurso int                         `json:"llamamientos_en_curso"`
			PoliticaOrden       politicaOrdenRRHHDesarrollo `json:"politica_orden"`
		}{
			Referencia: vigente.Bolsa.BolsaRef, CategoriaRef: vigente.CategoriaRef,
			Categoria: f.denominacion(vigente.CategoriaRef), TipoLista: politica.TipoLista, VigenteDesde: desde, PoliticaOrden: politica,
		})
		totalCurso, err := f.emisiones.ContarEnCurso(ctx, vigente.Bolsa.BolsaRef)
		if err != nil {
			return datasetBolsasRRHHDesarrollo{}, err
		}
		datos.Bolsas[len(datos.Bolsas)-1].LlamamientosEnCurso = totalCurso
		if err := f.cargarMarcas(ctx, &datos, vigente.Bolsa.BolsaRef); err != nil {
			return datasetBolsasRRHHDesarrollo{}, err
		}
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
			posicion, encontradaOrden := posiciones[entrada.ParticipacionRef]
			if !encontradaOrden {
				return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
			}
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
				Orden       *int    `json:"orden"`
				OrdenActa   int     `json:"orden_acta"`
				RazonOrden  string  `json:"razon_orden"`
				Nombre      string  `json:"nombre_visible"`
				Documento   string  `json:"documento_enmascarado"`
				Estado      string  `json:"estado_clave"`
				EstadoDesde string  `json:"estado_desde"`
				Disponible  *string `json:"disponible_desde"`
			}{
				Referencia: entrada.ParticipacionRef, BolsaRef: vigente.Bolsa.BolsaRef, Orden: posicion.vigente, OrdenActa: posicion.acta, RazonOrden: posicion.razon,
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
