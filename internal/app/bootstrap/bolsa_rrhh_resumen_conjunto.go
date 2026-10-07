package bootstrap

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	bolsaapplication "vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// lectorResumenBolsasInstalado exige Bolsa 000082 para la fuente RRHH. La
// composición deja la fuente indisponible si falta, en vez de servir miles de
// participaciones mediante consultas individuales sin avisar.
func lectorResumenBolsasInstalado(ctx context.Context, pool *pgxpool.Pool) ports.LectorResumenBolsas {
	var instalada bool
	if err := pool.QueryRow(ctx, `SELECT to_regprocedure('vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1(timestamptz)') IS NOT NULL
		AND to_regprocedure('vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1(timestamptz)') IS NOT NULL`).Scan(&instalada); err != nil {
		log.Printf("bolsa rrhh: no se pudo comprobar el resumen de conjunto (Bolsa 000082); fuente indisponible: %v", err)
		return nil
	}
	if !instalada {
		log.Printf("bolsa rrhh: resumen de conjunto (Bolsa 000082) no instalado; fuente indisponible")
		return nil
	}
	lector, err := postgresbolsa.NuevoLectorResumenBolsasPostgreSQL(pool)
	if err != nil {
		log.Printf("bolsa rrhh: lector de resumen de conjunto no disponible; fuente indisponible: %v", err)
		return nil
	}
	return lector
}

// cargarResumenConjunto produce el mismo conjunto que cargarAlcance sin
// detalle, con dos consultas de conjunto en lugar de una por bolsa y por
// participación: situaciones y ceses de todas las participaciones, y la
// política de orden vigente de cada bolsa. La proyección del cese es la misma.
// No descarga la instantánea canónica de cada bolsa (ListarVigentes), que con
// miles de participaciones son megas por petición.
func (f *fuenteConstituidaRRHHDesarrollo) cargarResumenConjunto(ctx context.Context) (datasetBolsasRRHHDesarrollo, error) {
	if f == nil || f.emisiones == nil || f.resumenConjunto == nil || f.ahora == nil {
		return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
	}
	corte := f.ahora()
	filas, politicas, err := f.resumenConjunto.LeerResumen(ctx, corte)
	if err != nil {
		return datasetBolsasRRHHDesarrollo{}, err
	}
	// Agrupa por bolsa conservando el orden de llegada (el de las
	// constituciones). Una bolsa partida en dos tramos es un resultado
	// incoherente y se rechaza.
	var orden []string
	porBolsa := map[string][]ports.SituacionResumenParticipacion{}
	for _, fila := range filas {
		if _, vista := porBolsa[fila.BolsaRef]; !vista {
			orden = append(orden, fila.BolsaRef)
		} else if orden[len(orden)-1] != fila.BolsaRef {
			return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
		}
		porBolsa[fila.BolsaRef] = append(porBolsa[fila.BolsaRef], fila)
	}
	datos := datasetBolsasRRHHDesarrollo{GeneradoEn: corte.UTC().Format(time.RFC3339)}
	for _, bolsaRef := range orden {
		participaciones := porBolsa[bolsaRef]
		primera := participaciones[0]
		p, existe := politicas[bolsaRef]
		// Igual que la lectura del orden vigente: sin política la bolsa no se
		// puede servir.
		if !existe || p.Validar() != nil {
			return datasetBolsasRRHHDesarrollo{}, ErrComposicionDesarrolloIncompleta
		}
		politica := politicaOrdenRRHHDesarrollo{Referencia: p.PoliticaRef, Criterio: p.Criterio, TipoLista: p.TipoLista, Reposicion: p.Reposicion, Rotulo: p.Rotulo, Actor: p.Actor, VigenteDesde: p.VigenteDesde.UTC().Format(time.RFC3339), Version: p.Version, Provisional: p.Provisional}
		totalCurso, err := f.emisiones.ContarEnCurso(ctx, bolsaRef)
		if err != nil {
			return datasetBolsasRRHHDesarrollo{}, err
		}
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
			Referencia: bolsaRef, CategoriaRef: primera.CategoriaRef, Categoria: f.denominacion(primera.CategoriaRef),
			TipoLista: politica.TipoLista, VigenteDesde: primera.ConfirmadaEn.UTC().Format(time.RFC3339),
			LlamamientosEnCurso: totalCurso, PoliticaOrden: politica,
		})
		for _, fila := range participaciones {
			if fila.Situacion == nil {
				return datasetBolsasRRHHDesarrollo{}, ports.ErrSituacionParticipacionNoEncontrada
			}
			situacion := *fila.Situacion
			if f.ceseActivo {
				var estado ports.EstadoCese
				if fila.Cese != nil {
					estado = *fila.Cese
				}
				if situacion, err = bolsaapplication.ProyectarSituacionConEstadoCese(situacion, estado, fila.Cese != nil, corte); err != nil {
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
				Referencia: fila.ParticipacionRef, BolsaRef: bolsaRef, OrdenActa: int(fila.Orden),
				Estado: situacion.Situacion, EstadoDesde: situacion.Desde.UTC().Format(time.RFC3339), Disponible: disponible,
			})
		}
	}
	return datos, nil
}
