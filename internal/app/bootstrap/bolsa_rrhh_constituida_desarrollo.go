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
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// fuenteConstituidaRRHHDesarrollo sirve las lecturas RRHH exclusivamente desde
// bolsas constituidas (migración 000007). Los nombres y documentos enmascarados
// no se almacenan en claro: se recuperan del staging protegido del acta al leer.
type fuenteConstituidaRRHHDesarrollo struct {
	repositorio ports.RepositorioConstitucion
	situaciones ports.RepositorioSituacionParticipacion
	estadosCese ports.ConsultaEstadoCese
	ceseActivo  bool
	orden       *bolsaapplication.ServicioOrdenVigente
	avisos      *bolsaapplication.ServicioAvisosRRHH
	// Bolsa 000041: bandeja con parámetros del catálogo, publicación de la
	// política y marcas por participación (bolsa_parametros_avisos_desarrollo.go).
	consultaAvisos *postgresbolsa.ConsultaAvisosRRHHPostgreSQL
	parametros     *postgresbolsa.RepositorioPoliticaAvisosPostgreSQL
	marcas         ports.ConsultaMarcasParticipaciones
	intentos       ports.PoliticaIntentosContacto
	emisiones      contadorLlamamientosEnCursoBolsa
	// Con B82 y B85, resumenConjunto sirve cuadro y estadísticas con tres
	// consultas en una instantánea. Sin B85 se usa todo el camino legado.
	resumenConjunto ports.LectorResumenBolsas
	// B92 acota las situaciones a la bolsa pedida; la lista exige este lector.
	bolsaConjunto   lectorBolsaRRHHConjunto
	recuperador     constitucion.Recuperador
	categorias      map[string]string
	grupos          map[string][]string
	ahora           func() time.Time

	mu       sync.Mutex
	cache    datasetBolsasRRHHDesarrollo
	cacheada bool
	hasta    time.Time
}

// contadorLlamamientosEnCursoBolsa se usa sólo en el camino legado, una vez
// por bolsa, cuando falta el resumen de conjunto B82/B85.
type contadorLlamamientosEnCursoBolsa interface {
	ContarEnCurso(context.Context, string) (int, error)
}

type lectorBolsaRRHHConjunto interface {
	LeerBolsa(context.Context, string, time.Time) (dominiobolsa.OrdenVigenteBolsa, []ports.SituacionResumenParticipacion, int, error)
}

const validezCacheBolsasConstituidas = 30 * time.Second
const envBolsaCeseCTEnabled = "VEC_BOLSA_CESE_CT_ENABLED"

// El texto original del error puede incluir datos de conexión. La etapa y la
// clasificación bastan para diagnosticar por qué esta fuente queda desactivada.
func registrarFalloFuenteConstituidaRRHHDesarrollo(etapa string, err error) {
	log.Printf("bolsa rrhh: bolsas constituidas no disponibles; etapa=%s causa=%s", etapa, causaFalloPostgreSQLCTDesarrollo(err))
}

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
	ceseActivo, err := selectorCapacidadRRHHDesarrollo(cfg, envBolsaCeseCTEnabled)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("selector_cese", err)
		return nil
	}
	poolBolsa, err := abrirBolsaLlamamientosPostgreSQLDesarrollo(ctx, cfg.ContratacionTemporalPostgreSQL)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("pool_bolsa", err)
		return nil
	}
	repositorio, err := postgresbolsa.NuevoRepositorioConstitucionPostgreSQL(poolBolsa)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("repositorio_constitucion", err)
		poolBolsa.Close()
		return nil
	}
	situaciones, err := postgresbolsa.NuevoRepositorioSituacionParticipacionPostgreSQL(poolBolsa)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("repositorio_situaciones", err)
		poolBolsa.Close()
		return nil
	}
	var estadosCese ports.ConsultaEstadoCese
	if ceseActivo {
		var instalada bool
		if err := poolBolsa.QueryRow(ctx, `SELECT to_regprocedure('vec_bolsa_llamamientos.consultar_estado_cese_bolsa_v1(text,timestamptz)') IS NOT NULL`).Scan(&instalada); err != nil || !instalada {
			registrarFalloFuenteConstituidaRRHHDesarrollo("capacidad_cese", err)
			poolBolsa.Close()
			return nil
		}
		estadosCese, err = postgresbolsa.NuevaConsultaEstadoCesePostgreSQL(poolBolsa)
		if err != nil {
			registrarFalloFuenteConstituidaRRHHDesarrollo("consulta_estado_cese", err)
			poolBolsa.Close()
			return nil
		}
	}
	consultaOrden, err := postgresbolsa.NuevaConsultaOrdenVigentePostgreSQL(poolBolsa)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("consulta_orden", err)
		poolBolsa.Close()
		return nil
	}
	orden, err := bolsaapplication.NuevoServicioOrdenVigente(consultaOrden)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("servicio_orden", err)
		poolBolsa.Close()
		return nil
	}
	nuevaConsultaAvisos := postgresbolsa.NuevaConsultaAvisosRRHHPostgreSQL
	if debeComponerPortalCandidatoDesarrollo(cfg) {
		nuevaConsultaAvisos = postgresbolsa.NuevaConsultaAvisosRRHHConPortalPostgreSQL
	}
	consultaAvisos, err := nuevaConsultaAvisos(poolBolsa)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("consulta_avisos", err)
		poolBolsa.Close()
		return nil
	}
	avisos, err := bolsaapplication.NuevoServicioAvisosRRHH(consultaAvisos, time.Now)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("servicio_avisos", err)
		poolBolsa.Close()
		return nil
	}
	emisiones, err := postgresbolsa.NuevoRepositorioEmisionLlamamientoPostgreSQL(poolBolsa)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("repositorio_emisiones", err)
		poolBolsa.Close()
		return nil
	}
	material, err := cargarMaterialSeguridadDesarrollo(cfg)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("material", err)
		poolBolsa.Close()
		return nil
	}
	defer borrarMaterialImportacionConvoca(material)
	p, err := protector.Nuevo(material.claveKMS)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("protector_staging", err)
		poolBolsa.Close()
		return nil
	}
	poolImportacion, err := abrirPoolImportacionConvoca(ctx, cfg)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("pool_importacion", err)
		poolBolsa.Close()
		return nil
	}
	recuperador, err := importacionpg.NuevoRepositorioRecuperacionPostgreSQL(poolImportacion, p)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("recuperador_staging", err)
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
	} else if strings.TrimSpace(cfg.RPTCatalogoPath) != "" {
		log.Printf("bolsa rrhh: catalogo RPT opcional no legible; causa=%s", causaFalloPostgreSQLCTDesarrollo(err))
	}
	parametros, err := postgresbolsa.NuevoRepositorioPoliticaAvisosPostgreSQL(poolBolsa)
	if err != nil {
		registrarFalloFuenteConstituidaRRHHDesarrollo("parametros_avisos", err)
		poolBolsa.Close()
		poolImportacion.Close()
		return nil
	}
	resumenConjunto := lectorResumenBolsasInstalado(ctx, poolBolsa)
	bolsaConjunto := lectorBolsaRRHHConjuntoInstalado(ctx, poolBolsa)
	return &fuenteConstituidaRRHHDesarrollo{repositorio: repositorio, situaciones: situaciones, estadosCese: estadosCese, ceseActivo: ceseActivo, orden: orden, avisos: avisos, consultaAvisos: consultaAvisos, parametros: parametros, emisiones: emisiones, resumenConjunto: resumenConjunto, bolsaConjunto: bolsaConjunto, recuperador: recuperador, categorias: categorias, grupos: grupos, ahora: time.Now}
}

func (f *fuenteConstituidaRRHHDesarrollo) constituidas(ctx context.Context) (datasetBolsasRRHHDesarrollo, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ahora := f.ahora()
	if f.cacheada && !f.ceseActivo && ahora.Before(f.hasta) {
		return f.cache, true
	}
	datos, err := f.cargar(ctx)
	if err != nil {
		if f.ceseActivo {
			log.Printf("bolsa rrhh: estado de cese no legible; causa=%s", causaFalloPostgreSQLCTDesarrollo(err))
		} else {
			log.Printf("bolsa rrhh: bolsas constituidas no legibles; causa=%s", causaFalloPostgreSQLCTDesarrollo(err))
		}
		if f.cacheada && !f.ceseActivo {
			return f.cache, true
		}
		return datasetBolsasRRHHDesarrollo{}, false
	}
	f.cache, f.cacheada, f.hasta = datos, true, ahora.Add(validezCacheBolsasConstituidas)
	return datos, true
}

// alcanceCargaBolsasRRHH decide cuánto lee una petición. El cuadro y las
// estadísticas solo cuentan personas por estado: no descifran el acta
// protegida (nombres y documentos) ni calculan marcas. La lista de
// candidatos de una bolsa lee solo esa bolsa, con todo su detalle.
type alcanceCargaBolsasRRHH struct {
	bolsa   string
	detalle bool
}

func (f *fuenteConstituidaRRHHDesarrollo) cargar(ctx context.Context) (datasetBolsasRRHHDesarrollo, error) {
	return f.cargarAlcance(ctx, alcanceCargaBolsasRRHH{detalle: true})
}

// cargarResumen sirve el cuadro y las estadísticas de RRHH.
func (f *fuenteConstituidaRRHHDesarrollo) cargarResumen(ctx context.Context) (datasetBolsasRRHHDesarrollo, error) {
	if f != nil && f.resumenConjunto != nil {
		return f.cargarResumenConjunto(ctx)
	}
	return f.cargarAlcance(ctx, alcanceCargaBolsasRRHH{})
}

// cargarBolsa sirve la lista de candidatos de una sola bolsa.
func (f *fuenteConstituidaRRHHDesarrollo) cargarBolsa(ctx context.Context, bolsaRef string) (datasetBolsasRRHHDesarrollo, error) {
	if bolsaRef == "" || f == nil || f.bolsaConjunto == nil {
		return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
	}
	return f.cargarAlcance(ctx, alcanceCargaBolsasRRHH{bolsa: bolsaRef, detalle: true})
}

func (f *fuenteConstituidaRRHHDesarrollo) cargarAlcance(ctx context.Context, alcance alcanceCargaBolsasRRHH) (datasetBolsasRRHHDesarrollo, error) {
	if f == nil || f.repositorio == nil || f.situaciones == nil || f.orden == nil || f.emisiones == nil || f.recuperador == nil || f.ahora == nil {
		return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
	}
	if f.ceseActivo && f.estadosCese == nil {
		return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
	}
	corte := f.ahora()
	vigentes, err := f.repositorio.ListarVigentes(ctx)
	if err != nil {
		return datasetBolsasRRHHDesarrollo{}, err
	}
	datos := datasetBolsasRRHHDesarrollo{GeneradoEn: corte.UTC().Format(time.RFC3339)}
	if alcance.detalle {
		if err := f.cargarMarcasBase(ctx, &datos); err != nil {
			return datasetBolsasRRHHDesarrollo{}, err
		}
	}
	for _, vigente := range vigentes {
		if alcance.bolsa != "" && vigente.Bolsa.BolsaRef != alcance.bolsa {
			continue
		}
		var ordenVigente dominiobolsa.OrdenVigenteBolsa
		var filasResumen []ports.SituacionResumenParticipacion
		var totalCurso int
		if alcance.bolsa != "" && f.bolsaConjunto != nil {
			ordenVigente, filasResumen, totalCurso, err = f.bolsaConjunto.LeerBolsa(ctx, vigente.Bolsa.BolsaRef, corte)
		} else {
			ordenVigente, err = f.orden.Consultar(ctx, vigente.Bolsa.BolsaRef)
		}
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
		if filasResumen == nil {
			totalCurso, err = f.emisiones.ContarEnCurso(ctx, vigente.Bolsa.BolsaRef)
			if err != nil {
				return datasetBolsasRRHHDesarrollo{}, err
			}
		}
		datos.Bolsas[len(datos.Bolsas)-1].LlamamientosEnCurso = totalCurso
		entradas, err := f.repositorio.Entradas(ctx, vigente.Instantanea.InstantaneaRef, vigente.Instantanea.Version)
		if err != nil {
			return datasetBolsasRRHHDesarrollo{}, err
		}
		var filas map[int]struct{ nombre, documento string }
		if alcance.detalle {
			if err := f.cargarMarcas(ctx, &datos, vigente.Bolsa.BolsaRef); err != nil {
				return datasetBolsasRRHHDesarrollo{}, err
			}
			if filas, err = f.filasVisibles(ctx, vigente); err != nil {
				return datasetBolsasRRHHDesarrollo{}, err
			}
		}
		var situacionesLote map[string]ports.SituacionParticipacion
		var cesesLote map[string]ports.EstadoCese
		if filasResumen != nil {
			situacionesLote, cesesLote, err = mapearSituacionesConjuntoBolsa(vigente, entradas, filasResumen)
		} else {
			situacionesLote, cesesLote, err = f.leerSituacionesBolsa(ctx, entradas, corte)
		}
		if err != nil {
			return datasetBolsasRRHHDesarrollo{}, err
		}
		for _, entrada := range entradas {
			posicion, encontradaOrden := posiciones[entrada.ParticipacionRef]
			if !encontradaOrden {
				return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
			}
			visible, encontrada := filas[entrada.FilaNumero]
			if alcance.detalle && !encontrada {
				return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
			}
			situacion, err := f.situacionDeLote(ctx, situacionesLote, entrada.ParticipacionRef)
			if err != nil {
				return datasetBolsasRRHHDesarrollo{}, err
			}
			if f.ceseActivo {
				estadoCese, presente, err := f.estadoCeseDeLote(ctx, cesesLote, entrada.ParticipacionRef, corte)
				if err != nil {
					return datasetBolsasRRHHDesarrollo{}, err
				}
				situacion, err = bolsaapplication.ProyectarSituacionConEstadoCese(situacion, estadoCese, presente, corte)
				if err != nil {
					return datasetBolsasRRHHDesarrollo{}, err
				}
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

// filasVisibles descifra del acta protegida el nombre y el documento
// enmascarado de cada fila. Solo la lista de candidatos los necesita.
func (f *fuenteConstituidaRRHHDesarrollo) filasVisibles(ctx context.Context, vigente ports.ConstitucionVigente) (map[int]struct{ nombre, documento string }, error) {
	lote, _, existe, err := f.recuperador.RecuperarLote(ctx, vigente.Bolsa.HuellaListadoSHA256, vigente.CategoriaRef)
	if err != nil {
		return nil, err
	}
	if !existe {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	filas := make(map[int]struct{ nombre, documento string }, len(lote.Aceptadas))
	for _, fila := range lote.Aceptadas {
		nombre := strings.TrimSpace(strings.Join([]string{fila.Identidad.Nombre, fila.Identidad.PrimerApellido, fila.Identidad.SegundoApellido}, " "))
		filas[fila.Numero] = struct{ nombre, documento string }{strings.Join(strings.Fields(nombre), " "), fila.Identidad.Documento}
	}
	return filas, nil
}

// leerSituacionesBolsa lee de una vez la situación vigente y, si procede, el
// estado de cese de todas las participaciones de una bolsa: dos idas y vueltas
// por bolsa en lugar de dos por participación. Si el adaptador no ofrece
// lectura por lotes devuelve mapas nulos y la lectura sigue siendo individual.
func (f *fuenteConstituidaRRHHDesarrollo) leerSituacionesBolsa(ctx context.Context, entradas []ports.EntradaConstitucion, corte time.Time) (map[string]ports.SituacionParticipacion, map[string]ports.EstadoCese, error) {
	refs := make([]string, 0, len(entradas))
	for _, entrada := range entradas {
		refs = append(refs, entrada.ParticipacionRef)
	}
	var situaciones map[string]ports.SituacionParticipacion
	if lector, ok := f.situaciones.(ports.LectorSituacionesVigentes); ok {
		var err error
		if situaciones, err = lector.SituacionesVigentes(ctx, refs); err != nil {
			return nil, nil, err
		}
	}
	var ceses map[string]ports.EstadoCese
	if consulta, ok := f.estadosCese.(ports.ConsultaEstadosCese); ok && f.ceseActivo {
		var err error
		if ceses, err = consulta.ConsultarEstadosCese(ctx, refs, corte); err != nil {
			return nil, nil, err
		}
	}
	return situaciones, ceses, nil
}

func (f *fuenteConstituidaRRHHDesarrollo) situacionDeLote(ctx context.Context, lote map[string]ports.SituacionParticipacion, ref string) (ports.SituacionParticipacion, error) {
	if lote == nil {
		return f.situaciones.SituacionVigente(ctx, ref)
	}
	situacion, ok := lote[ref]
	if !ok {
		return ports.SituacionParticipacion{}, ports.ErrSituacionParticipacionNoEncontrada
	}
	return situacion, nil
}

func (f *fuenteConstituidaRRHHDesarrollo) estadoCeseDeLote(ctx context.Context, lote map[string]ports.EstadoCese, ref string, corte time.Time) (ports.EstadoCese, bool, error) {
	if lote == nil {
		return f.estadosCese.ConsultarEstadoCese(ctx, ref, corte)
	}
	estado, presente := lote[ref]
	return estado, presente, nil
}

func (f *fuenteConstituidaRRHHDesarrollo) denominacion(categoriaRef string) string {
	if etiqueta, ok := f.categorias[categoriaRef]; ok && etiqueta != "" {
		return etiqueta
	}
	return strings.ToUpper(strings.ReplaceAll(strings.TrimPrefix(categoriaRef, "categoria:rpt:"), "-", " "))
}
