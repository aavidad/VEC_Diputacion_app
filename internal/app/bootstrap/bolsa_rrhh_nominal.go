package bootstrap

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	bolsaapplication "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	vidaSeleccionTextoRRHHBolsa         = 2 * time.Minute
	maximoSeleccionesTextoRRHHBolsa     = 16
	maximoReferenciasSeleccionRRHHBolsa = 80000
)

type seleccionTextoRRHHBolsa struct {
	ids                                          []string
	principalID, personaRef, perfilRef, bolsaRef string
	filtroSHA256, snapshotSHA256                 string
	expira                                       time.Time
}

type lecturasNominalesRRHHBolsaDesarrollo struct {
	fuente               *fuenteConstituidaRRHHDesarrollo
	preparador           *preparadorBorradorLlamamientoDesarrollo
	bolsas, estadisticas *bolsaapplication.ServicioConsultaResumenRRHHNominal
	candidatos           *bolsaapplication.ServicioConsultaCandidatosRRHHNominal
	mu                   sync.Mutex
	selecciones          map[string]seleccionTextoRRHHBolsa
}

// ConfigurarLecturasNominalesRRHHBolsaDesarrollo se invoca en composición
// después de crear la fuente y el PDP de Bolsa. Si falta una audiencia o una
// sesión nominal, las tres rutas quedan indisponibles, sin lector histórico.
func ConfigurarLecturasNominalesRRHHBolsaDesarrollo(ctx context.Context, fuente *fuenteConstituidaRRHHDesarrollo,
	mutador http.Handler, gobierno *pgxpool.Pool, emisores map[string]*emisorMaterialRenovableCTDesarrollo) error {
	m, ok := mutador.(*manejadorParticipacionBolsaDesarrollo)
	if !ok || ctx == nil || ctx.Err() != nil || fuente == nil || fuente.poolNominal == nil ||
		fuente.recuperadorSelectivo == nil || gobierno == nil || m.preparador == nil ||
		m.preparador.sesion == nil || m.preparador.soporte == nil || len(emisores) != 3 {
		return ErrComposicionDesarrolloIncompleta
	}
	resumen, ok := fuente.resumenConjunto.(puertosbolsa.LectorResumenBolsasNominal)
	if !ok {
		return ErrComposicionDesarrolloIncompleta
	}
	var instaladas bool
	if err := fuente.poolNominal.QueryRow(ctx, `SELECT to_regprocedure('vec_bolsa_llamamientos.consultar_resumen_rrhh_nominal_v1(text,boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
		AND to_regprocedure('vec_bolsa_llamamientos.consultar_candidatos_rrhh_nominal_v1(text,text,text,text,text,text,integer,text,text[],boolean,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
		AND to_regprocedure('vec_bolsa_llamamientos.finalizar_barrido_rrhh_nominal_v1(text,text,text,text[])') IS NOT NULL`).Scan(&instaladas); err != nil || !instaladas {
		return ErrComposicionDesarrolloIncompleta
	}
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(time.Now())
	if !vigente || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno,
		[]dominiovec.ReferenciaEntradaCatalogo{
			motivoRRHHBolsasBolsaDesarrollo(), motivoRRHHEstadisticasBolsaDesarrollo(), motivoRRHHCandidatosBolsaDesarrollo(),
		}, desde) != nil {
		return ErrComposicionDesarrolloIncompleta
	}
	lectorCandidatos, err := postgresbolsa.NuevoLectorCandidatosRRHHPostgreSQL(fuente.poolNominal, fuente.recuperadorSelectivo, fuente.intentos)
	if err != nil {
		return ErrComposicionDesarrolloIncompleta
	}
	nuevoResumen := func(accion string) (*bolsaapplication.ServicioConsultaResumenRRHHNominal, error) {
		emisor := emisores[accion]
		if emisor == nil {
			return nil, ErrComposicionDesarrolloIncompleta
		}
		return bolsaapplication.NuevoServicioConsultaResumenRRHHNominal(resumen, emisor, time.Now)
	}
	bolsas, err := nuevoResumen(puertosbolsa.AccionRRHHBolsasConsultar)
	if err != nil {
		return ErrComposicionDesarrolloIncompleta
	}
	estadisticas, err := nuevoResumen(puertosbolsa.AccionRRHHEstadisticasConsultar)
	if err != nil {
		return ErrComposicionDesarrolloIncompleta
	}
	emisorCandidatos := emisores[puertosbolsa.AccionRRHHCandidatosConsultar]
	if emisorCandidatos == nil {
		return ErrComposicionDesarrolloIncompleta
	}
	candidatos, err := bolsaapplication.NuevoServicioConsultaCandidatosRRHHNominal(lectorCandidatos, emisorCandidatos, time.Now)
	if err != nil {
		return ErrComposicionDesarrolloIncompleta
	}
	fuente.nominal = &lecturasNominalesRRHHBolsaDesarrollo{fuente: fuente, preparador: m.preparador,
		bolsas: bolsas, estadisticas: estadisticas, candidatos: candidatos,
		selecciones: make(map[string]seleccionTextoRRHHBolsa)}
	return nil
}

func motivoRRHHBolsasBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return motivoConsultaNominalBolsa("bolsas")
}
func motivoRRHHEstadisticasBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return motivoConsultaNominalBolsa("estadisticas")
}
func motivoRRHHCandidatosBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return motivoConsultaNominalBolsa("candidatos")
}
func motivoConsultaNominalBolsa(nombre string) dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_consulta_rrhh_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-consulta-rrhh-bolsa-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-rrhh-"+nombre+"-consultar")}
}

func (n *lecturasNominalesRRHHBolsaDesarrollo) prepararOrden(ctx context.Context, accion string) (puertosbolsa.OrdenConsultaResumenRRHH, string, error) {
	var vacio puertosbolsa.OrdenConsultaResumenRRHH
	if n == nil || n.preparador == nil || n.preparador.soporte == nil {
		return vacio, "", ErrComposicionDesarrolloIncompleta
	}
	seguridad, err := n.preparador.contextoRevalidado(ctx)
	if err != nil {
		return vacio, "", err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, n.preparador.generar)
	if err != nil {
		return vacio, "", ErrComposicionDesarrolloIncompleta
	}
	motivo := motivoRRHHBolsasBolsaDesarrollo()
	switch accion {
	case puertosbolsa.AccionRRHHEstadisticasConsultar:
		motivo = motivoRRHHEstadisticasBolsaDesarrollo()
	case puertosbolsa.AccionRRHHCandidatosConsultar:
		motivo = motivoRRHHCandidatosBolsaDesarrollo()
	case puertosbolsa.AccionRRHHBolsasConsultar:
	default:
		return vacio, "", ErrComposicionDesarrolloIncompleta
	}
	vinculo, err := seguridad.Vinculo.Datos()
	if err != nil || vinculo.PrincipalID == "" || seguridad.Resultado.Contexto.PersonaRef == "" {
		return vacio, "", ErrComposicionDesarrolloIncompleta
	}
	return puertosbolsa.OrdenConsultaResumenRRHH{Accion: accion, UnidadRef: n.preparador.soporte.unidadRef,
		AmbitoRef: n.preparador.soporte.ambitoRef, Resultado: seguridad.Resultado, Vinculo: seguridad.Vinculo,
		Motivo: motivo, Correlacion: correlacion, CeseActivo: n.fuente.ceseActivo}, vinculo.PrincipalID, nil
}

func (n *lecturasNominalesRRHHBolsaDesarrollo) consultarResumen(ctx context.Context, accion string) (map[string]any, error) {
	orden, _, err := n.prepararOrden(ctx, accion)
	if err != nil {
		return nil, err
	}
	servicio := n.bolsas
	if accion == puertosbolsa.AccionRRHHEstadisticasConsultar {
		servicio = n.estadisticas
	}
	if servicio == nil {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	resumen, err := servicio.Consultar(ctx, orden)
	if err != nil {
		return nil, err
	}
	if accion == puertosbolsa.AccionRRHHBolsasConsultar {
		bolsas := make([]map[string]any, 0, len(resumen.Bolsas))
		for _, b := range resumen.Bolsas {
			politica := politicaOrdenRRHHDesarrollo{Referencia: b.Politica.PoliticaRef,
				Criterio: b.Politica.Criterio, TipoLista: b.Politica.TipoLista, Reposicion: b.Politica.Reposicion,
				Rotulo: b.Politica.Rotulo, Actor: b.Politica.Actor,
				VigenteDesde: b.Politica.VigenteDesde.UTC().Format(time.RFC3339),
				Version:      b.Politica.Version, Provisional: b.Politica.Provisional}
			conteos := mapaEstadosVacio()
			for estado, total := range b.PorEstado {
				conteos[estado] = total
			}
			var hasta *string
			if b.VigenteHasta != nil {
				v := b.VigenteHasta.UTC().Format(time.RFC3339)
				hasta = &v
			}
			bolsas = append(bolsas, salidaBolsaRRHH(b.BolsaRef, b.CategoriaRef,
				n.fuente.denominacion(b.CategoriaRef), politica.TipoLista,
				b.ConfirmadaEn.UTC().Format(time.RFC3339), hasta, conteos, b.LlamamientosEnCurso, politica))
		}
		return map[string]any{"esquema": "vec.bolsa.rrhh.bolsas.v1",
			"generado_en": resumen.GeneradoEn.UTC().Format(time.RFC3339), "bolsas": bolsas}, nil
	}
	if accion != puertosbolsa.AccionRRHHEstadisticasConsultar || resumen.Estadisticas == nil {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	s := resumen.Estadisticas
	porBolsa := make([]map[string]any, 0, len(s.PorBolsa))
	for _, b := range s.PorBolsa {
		conteos := mapaEstadosVacio()
		for estado, total := range b.PorEstado {
			conteos[estado] = total
		}
		porBolsa = append(porBolsa, map[string]any{"bolsa_ref": b.BolsaRef,
			"categoria": n.fuente.denominacion(b.CategoriaRef), "tipo_lista": b.TipoLista,
			"vigente": b.Vigente, "total": b.Total, "por_estado": conteos})
	}
	porEstado := mapaEstadosVacio()
	for estado, total := range s.PersonasPorEstado {
		porEstado[estado] = total
	}
	return map[string]any{"esquema": "vec.bolsa.rrhh.estadisticas.v1",
		"generado_en": resumen.GeneradoEn.UTC().Format(time.RFC3339),
		"bolsas":      map[string]any{"total": s.BolsasTotal, "vigentes": s.BolsasVigentes, "sustituidas": s.BolsasSustituidas},
		"personas":    map[string]any{"total": s.PersonasTotal, "por_estado": porEstado},
		"llamamientos": map[string]any{"en_curso": s.LlamamientosEnCurso, "historico_disponible": false,
			"total": nil, "por_canal": nil, "por_resultado": nil}, "por_bolsa": porBolsa}, nil
}

func huellaFiltroTextoRRHHBolsa(bolsaRef, estado, texto string) string {
	suma := sha256.Sum256([]byte(bolsaRef + "\x1f" + estado + "\x1f" + texto))
	return hex.EncodeToString(suma[:])
}

func nuevoIDSeleccionTextoRRHHBolsa() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}

func (n *lecturasNominalesRRHHBolsaDesarrollo) invalidarSelecciones() {
	if n == nil {
		return
	}
	n.mu.Lock()
	n.selecciones = make(map[string]seleccionTextoRRHHBolsa)
	n.mu.Unlock()
}

func (n *lecturasNominalesRRHHBolsaDesarrollo) guardarSeleccion(principalID string,
	orden puertosbolsa.OrdenConsultaResumenRRHH, bolsaRef, filtro, snapshot string, ids []string) (string, error) {
	id, err := nuevoIDSeleccionTextoRRHHBolsa()
	if err != nil || len(ids) > 5000 {
		return "", ErrComposicionDesarrolloIncompleta
	}
	ahora := time.Now()
	n.mu.Lock()
	defer n.mu.Unlock()
	total := len(ids)
	for clave, s := range n.selecciones {
		if ahora.After(s.expira) {
			delete(n.selecciones, clave)
		} else {
			total += len(s.ids)
		}
	}
	if len(n.selecciones) >= maximoSeleccionesTextoRRHHBolsa || total > maximoReferenciasSeleccionRRHHBolsa {
		// Cota firme de memoria: las selecciones anteriores caducan juntas.
		n.selecciones = make(map[string]seleccionTextoRRHHBolsa)
	}
	n.selecciones[id] = seleccionTextoRRHHBolsa{ids: append([]string(nil), ids...),
		principalID: principalID, personaRef: orden.Resultado.Contexto.PersonaRef,
		perfilRef: orden.Resultado.Contexto.PerfilActivoRef, bolsaRef: bolsaRef,
		filtroSHA256: filtro, snapshotSHA256: snapshot, expira: ahora.Add(vidaSeleccionTextoRRHHBolsa)}
	return id, nil
}

func cursorTextoRRHHBolsa(id string, indice int, snapshot string, pagina []string) string {
	suma := sha256.Sum256([]byte(strings.Join(pagina, "\x1f")))
	return "txt." + id + "." + strconv.Itoa(indice) + "." + snapshot + "." + hex.EncodeToString(suma[:])
}
func cursorSQLRRHHBolsa(ref, snapshot string) string {
	return "sql." + hex.EncodeToString([]byte(ref)) + "." + snapshot
}

func (n *lecturasNominalesRRHHBolsaDesarrollo) consultarCandidatos(ctx context.Context, bolsaRef string, consulta consultaCandidatosRRHH) (map[string]any, error) {
	orden, principalID, err := n.prepararOrden(ctx, puertosbolsa.AccionRRHHCandidatosConsultar)
	if err != nil {
		return nil, err
	}
	q := puertosbolsa.ConsultaCandidatosRRHHNominal{BolsaRef: bolsaRef, Estado: consulta.estado,
		Texto: consulta.texto, Limite: consulta.limite, CursorToken: consulta.cursor, CeseActivo: n.fuente.ceseActivo}
	var seleccionID string
	var indice int
	var seleccion seleccionTextoRRHHBolsa
	if consulta.texto == "" {
		q.Modo = puertosbolsa.ModoPaginaCandidatosRRHH
		if consulta.cursor != "" {
			partes := strings.Split(consulta.cursor, ".")
			if len(partes) != 3 || partes[0] != "sql" || len(partes[2]) != 64 {
				return nil, ErrComposicionDesarrolloIncompleta
			}
			ref, e := hex.DecodeString(partes[1])
			if e != nil || len(ref) == 0 || len(ref) > 512 {
				return nil, ErrComposicionDesarrolloIncompleta
			}
			q.CursorRef, q.SnapshotSHA256 = string(ref), partes[2]
		}
	} else if consulta.cursor == "" {
		q.Modo = puertosbolsa.ModoBarridoTextoCandidatosRRHH
	} else {
		partes := strings.Split(consulta.cursor, ".")
		if len(partes) != 5 || partes[0] != "txt" || len(partes[1]) != 32 || len(partes[3]) != 64 || len(partes[4]) != 64 {
			return nil, ErrComposicionDesarrolloIncompleta
		}
		seleccionID = partes[1]
		indice, err = strconv.Atoi(partes[2])
		if err != nil || indice < 1 {
			return nil, ErrComposicionDesarrolloIncompleta
		}
		n.mu.Lock()
		seleccion, err = n.recuperarSeleccionBloqueada(seleccionID)
		n.mu.Unlock()
		if err != nil || !seleccionTextoRRHHBolsaValida(seleccion, principalID,
			orden.Resultado.Contexto.PersonaRef, orden.Resultado.Contexto.PerfilActivoRef,
			bolsaRef, huellaFiltroTextoRRHHBolsa(bolsaRef, consulta.estado, consulta.texto), partes[3], indice) {
			return nil, ErrComposicionDesarrolloIncompleta
		}
		fin := indice + consulta.limite
		if fin > len(seleccion.ids) {
			fin = len(seleccion.ids)
		}
		q.Modo = puertosbolsa.ModoPaginaTextoCandidatosRRHH
		q.SnapshotSHA256 = seleccion.snapshotSHA256
		q.SeleccionRefs = append([]string(nil), seleccion.ids[indice:fin]...)
		firma := sha256.Sum256([]byte(strings.Join(q.SeleccionRefs, "\x1f")))
		if hex.EncodeToString(firma[:]) != partes[4] {
			return nil, ErrComposicionDesarrolloIncompleta
		}
	}
	pagina, err := n.candidatos.Consultar(ctx, orden, q)
	if err != nil {
		return nil, err
	}
	if q.Modo == puertosbolsa.ModoBarridoTextoCandidatosRRHH {
		pagina.TotalFiltrado = len(pagina.SeleccionRefs)
		if pagina.HayMas {
			seleccionID, err = n.guardarSeleccion(principalID, orden, bolsaRef,
				huellaFiltroTextoRRHHBolsa(bolsaRef, consulta.estado, consulta.texto), pagina.SnapshotSHA256, pagina.SeleccionRefs)
			if err != nil {
				return nil, err
			}
			inicio := len(pagina.Candidatos)
			fin := inicio + consulta.limite
			if fin > len(pagina.SeleccionRefs) {
				fin = len(pagina.SeleccionRefs)
			}
			pagina.CursorRefSiguiente = cursorTextoRRHHBolsa(seleccionID, inicio, pagina.SnapshotSHA256, pagina.SeleccionRefs[inicio:fin])
		}
	} else if q.Modo == puertosbolsa.ModoPaginaTextoCandidatosRRHH {
		pagina.TotalFiltrado = len(seleccion.ids)
		pagina.HayMas = indice+len(pagina.Candidatos) < len(seleccion.ids)
		pagina.CursorRefSiguiente = ""
		if pagina.HayMas {
			inicio := indice + len(pagina.Candidatos)
			fin := inicio + consulta.limite
			if fin > len(seleccion.ids) {
				fin = len(seleccion.ids)
			}
			pagina.CursorRefSiguiente = cursorTextoRRHHBolsa(seleccionID, inicio, pagina.SnapshotSHA256, seleccion.ids[inicio:fin])
		}
	} else if pagina.HayMas {
		pagina.CursorRefSiguiente = cursorSQLRRHHBolsa(pagina.CursorRefSiguiente, pagina.SnapshotSHA256)
	}
	return n.respuestaCandidatosNominal(pagina)
}

func (n *lecturasNominalesRRHHBolsaDesarrollo) recuperarSeleccionBloqueada(id string) (seleccionTextoRRHHBolsa, error) {
	s, ok := n.selecciones[id]
	if !ok || time.Now().After(s.expira) {
		delete(n.selecciones, id)
		return seleccionTextoRRHHBolsa{}, ErrComposicionDesarrolloIncompleta
	}
	s.ids = append([]string(nil), s.ids...)
	return s, nil
}

func seleccionTextoRRHHBolsaValida(s seleccionTextoRRHHBolsa, principalID, personaRef, perfilRef,
	bolsaRef, filtroSHA256, snapshotSHA256 string, indice int) bool {
	return s.principalID == principalID && s.personaRef == personaRef && s.perfilRef == perfilRef &&
		s.bolsaRef == bolsaRef && s.filtroSHA256 == filtroSHA256 && s.snapshotSHA256 == snapshotSHA256 &&
		indice >= 1 && indice < len(s.ids) && time.Now().Before(s.expira)
}

func (n *lecturasNominalesRRHHBolsaDesarrollo) respuestaCandidatosNominal(p puertosbolsa.PaginaCandidatosRRHHNominal) (map[string]any, error) {
	datos := datasetBolsasRRHHDesarrollo{GeneradoEn: p.GeneradoEn.UTC().Format(time.RFC3339), Contactos: p.Contactos}
	if n.fuente.marcas != nil {
		datos.Marcas = p.Marcas
		datos.PoliticaIntentos = p.PoliticaIntentos
	}
	for _, c := range p.Candidatos {
		var disponible *string
		if c.DisponibleDesde != nil {
			v := c.DisponibleDesde.UTC().Format(time.RFC3339)
			disponible = &v
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
		}{c.ParticipacionRef, p.BolsaRef, c.Orden, c.OrdenActa, c.RazonOrden, c.NombreVisible, c.DocumentoEnmascarado, c.Estado, c.EstadoDesde.UTC().Format(time.RFC3339), disponible})
	}
	vista := &bolsasRRHHDesarrolloDatos{datos: datos}
	candidatos := make([]map[string]any, 0, len(datos.Candidaturas))
	for _, c := range datos.Candidaturas {
		candidatos = append(candidatos, vista.salidaCandidata(c))
	}
	contactos := make([]map[string]any, 0, len(p.Contactos))
	for _, c := range p.Contactos {
		contactos = append(contactos, map[string]any{"contacto_ref": c.ContactoRef, "participacion_ref": c.ParticipacionRef,
			"llamamiento_ref": nuloBootstrap(c.LlamamientoRef), "canal": c.Canal, "instante": c.Instante.UTC().Format(time.RFC3339Nano),
			"actor_ref": c.Actor, "resultado": c.Resultado, "anotacion": c.Anotacion})
	}
	conteos := mapaEstadosVacio()
	for estado, total := range p.PorEstado {
		if _, ok := conteos[estado]; !ok || total < 0 {
			return nil, ErrComposicionDesarrolloIncompleta
		}
		conteos[estado] = total
	}
	politica := politicaOrdenRRHHDesarrollo{Referencia: p.Politica.PoliticaRef, Criterio: p.Politica.Criterio, TipoLista: p.Politica.TipoLista,
		Reposicion: p.Politica.Reposicion, Rotulo: p.Politica.Rotulo, Actor: p.Politica.Actor,
		VigenteDesde: p.Politica.VigenteDesde.UTC().Format(time.RFC3339), Version: p.Politica.Version, Provisional: p.Politica.Provisional}
	var vigenteHasta *string
	if p.VigenteHasta != nil {
		v := p.VigenteHasta.UTC().Format(time.RFC3339)
		vigenteHasta = &v
	}
	bolsa := salidaBolsaRRHH(p.BolsaRef, p.CategoriaRef, n.fuente.denominacion(p.CategoriaRef), p.TipoLista,
		p.VigenteDesde.UTC().Format(time.RFC3339), vigenteHasta, conteos, p.LlamamientosEnCurso, politica)
	var ultimo any
	if p.TurnoUltimo != nil && p.TurnoUltimo.Contacto != nil {
		c := p.TurnoUltimo.Contacto
		ultimo = map[string]any{"participacion_ref": p.TurnoUltimo.Candidato.ParticipacionRef,
			"nombre_visible": p.TurnoUltimo.Candidato.NombreVisible, "orden": p.TurnoUltimo.Candidato.Orden,
			"comunicado_en": c.Instante.UTC().Format(time.RFC3339Nano), "canal": c.Canal, "resultado": c.Resultado}
	}
	var siguiente any
	estadoSiguiente := bolsaapplication.EstadoSiguienteSinDisponibles
	if p.TurnoSiguiente != nil {
		siguiente = map[string]any{"participacion_ref": p.TurnoSiguiente.ParticipacionRef,
			"nombre_visible": p.TurnoSiguiente.NombreVisible, "orden": p.TurnoSiguiente.Orden}
		estadoSiguiente = bolsaapplication.EstadoSiguientePrimeroDisponible
	}
	turno := map[string]any{"politica_ref": p.Politica.PoliticaRef, "politica_version": p.Politica.Version,
		"provisional": p.Politica.Provisional, "ultimo_llamado": ultimo, "siguiente": siguiente, "estado_siguiente": estadoSiguiente}
	var cursor any
	if p.HayMas {
		cursor = p.CursorRefSiguiente
	}
	return map[string]any{"esquema": "vec.bolsa.rrhh.candidatos.v2", "generado_en": p.GeneradoEn.UTC().Format(time.RFC3339),
		"bolsa": bolsa, "candidatos": candidatos, "contactos": contactos, "turno": turno,
		"hay_mas": p.HayMas, "cursor_siguiente": cursor, "total_filtrado": p.TotalFiltrado}, nil
}
