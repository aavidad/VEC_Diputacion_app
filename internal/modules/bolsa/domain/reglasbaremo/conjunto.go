package reglasbaremo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"vec-diputacion-granada/internal/shared/baremacion"
)

const (
	esquemaConjuntoReglasBaremo     = "vec.bolsa.conjunto_reglas_baremo.v1"
	maximoSeccionesPorConjunto      = 64
	maximoGruposConcurrencia        = 256
	maximoReglasExperienciaConjunto = 1024
	maximoCriteriosConjunto         = 4096
	maximoValoresCriterioConjunto   = 16_384
	maximoBytesRepresentacion       = 4 * 1024 * 1024
	maximoCaracteresClave           = 128
	maximoCaracteresReferencia      = 512
)

var (
	patronClave      = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	patronReferencia = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]*$`)
)

// ConjuntoReglasBaremo es la version inmutable de las reglas de una
// convocatoria. Este primer corte solo contiene reglas de experiencia; no
// calcula ni publica resultados.
type ConjuntoReglasBaremo struct {
	identidad           IdentidadConjuntoReglasBaremo
	bases               ReferenciaVersionada
	fechaCorteInclusiva baremacion.FechaCivil
	secciones           []SeccionBaremo
	gruposConcurrencia  []GrupoConcurrenciaExperiencia
	reglasExperiencia   []ReglaExperiencia
}

// NuevoConjuntoReglasBaremo construye, valida y ordena una instantanea. Las
// colecciones recibidas se copian y dejan de pertenecer al agregado.
func NuevoConjuntoReglasBaremo(
	identidad IdentidadConjuntoReglasBaremo,
	bases ReferenciaVersionada,
	fechaCorte baremacion.FechaCivil,
	secciones []SeccionBaremo,
	gruposConcurrencia []GrupoConcurrenciaExperiencia,
	reglasExperiencia []ReglaExperiencia,
) (ConjuntoReglasBaremo, error) {
	if len(secciones) == 0 || len(secciones) > maximoSeccionesPorConjunto {
		return ConjuntoReglasBaremo{}, nuevoError("secciones", CodigoFueraDeLimites)
	}
	if len(reglasExperiencia) == 0 || len(reglasExperiencia) > maximoReglasExperienciaConjunto {
		return ConjuntoReglasBaremo{}, nuevoError("reglas_experiencia", CodigoFueraDeLimites)
	}
	if len(gruposConcurrencia) == 0 || len(gruposConcurrencia) > maximoGruposConcurrencia {
		return ConjuntoReglasBaremo{}, nuevoError("grupos_concurrencia", CodigoFueraDeLimites)
	}
	if err := validarVolumenReglas(reglasExperiencia); err != nil {
		return ConjuntoReglasBaremo{}, err
	}
	conjunto := ConjuntoReglasBaremo{
		identidad:           identidad,
		bases:               bases,
		fechaCorteInclusiva: fechaCorte,
		secciones:           append([]SeccionBaremo(nil), secciones...),
		gruposConcurrencia:  append([]GrupoConcurrenciaExperiencia(nil), gruposConcurrencia...),
		reglasExperiencia:   clonarReglas(reglasExperiencia),
	}
	if err := conjunto.validar(false); err != nil {
		return ConjuntoReglasBaremo{}, err
	}

	sort.Slice(conjunto.secciones, func(i, j int) bool {
		if conjunto.secciones[i].orden != conjunto.secciones[j].orden {
			return conjunto.secciones[i].orden < conjunto.secciones[j].orden
		}
		return conjunto.secciones[i].clave < conjunto.secciones[j].clave
	})
	ordenSeccion := make(map[string]uint32, len(conjunto.secciones))
	for _, seccion := range conjunto.secciones {
		ordenSeccion[seccion.clave] = seccion.orden
	}
	sort.Slice(conjunto.gruposConcurrencia, func(i, j int) bool {
		if conjunto.gruposConcurrencia[i].orden != conjunto.gruposConcurrencia[j].orden {
			return conjunto.gruposConcurrencia[i].orden < conjunto.gruposConcurrencia[j].orden
		}
		return conjunto.gruposConcurrencia[i].clave < conjunto.gruposConcurrencia[j].clave
	})
	sort.Slice(conjunto.reglasExperiencia, func(i, j int) bool {
		izquierda, derecha := conjunto.reglasExperiencia[i], conjunto.reglasExperiencia[j]
		if ordenSeccion[izquierda.seccionClave] != ordenSeccion[derecha.seccionClave] {
			return ordenSeccion[izquierda.seccionClave] < ordenSeccion[derecha.seccionClave]
		}
		if izquierda.orden != derecha.orden {
			return izquierda.orden < derecha.orden
		}
		return izquierda.clave < derecha.clave
	})

	if err := conjunto.validar(true); err != nil {
		return ConjuntoReglasBaremo{}, err
	}
	if _, err := conjunto.RepresentacionCanonica(); err != nil {
		return ConjuntoReglasBaremo{}, err
	}
	return conjunto, nil
}

// Identidad devuelve la identidad por valor.
func (c ConjuntoReglasBaremo) Identidad() IdentidadConjuntoReglasBaremo { return c.identidad }

// Bases devuelve la version y huella exactas de las bases publicadas.
func (c ConjuntoReglasBaremo) Bases() ReferenciaVersionada { return c.bases }

// FechaCorte devuelve el ultimo dia civil incluido en el computo. La
// inclusividad forma parte de la invariante V1 y no depende de una convencion
// del calculador.
func (c ConjuntoReglasBaremo) FechaCorte() baremacion.FechaCivil { return c.fechaCorteInclusiva }

// Secciones devuelve una copia ordenada.
func (c ConjuntoReglasBaremo) Secciones() []SeccionBaremo {
	return append([]SeccionBaremo(nil), c.secciones...)
}

// GruposConcurrenciaExperiencia devuelve una copia ordenada de las politicas
// que resuelven coincidencias de reglas y solapes temporales.
func (c ConjuntoReglasBaremo) GruposConcurrenciaExperiencia() []GrupoConcurrenciaExperiencia {
	return append([]GrupoConcurrenciaExperiencia(nil), c.gruposConcurrencia...)
}

// ReglasExperiencia devuelve una copia profunda y ordenada.
func (c ConjuntoReglasBaremo) ReglasExperiencia() []ReglaExperiencia {
	return clonarReglas(c.reglasExperiencia)
}

// SeccionPorClave busca sin exponer la coleccion interna.
func (c ConjuntoReglasBaremo) SeccionPorClave(clave string) (SeccionBaremo, bool) {
	if !claveValida(clave) {
		return SeccionBaremo{}, false
	}
	for _, seccion := range c.secciones {
		if seccion.clave == clave {
			return seccion, true
		}
	}
	return SeccionBaremo{}, false
}

// GrupoConcurrenciaPorClave busca sin exponer la coleccion interna.
func (c ConjuntoReglasBaremo) GrupoConcurrenciaPorClave(
	clave string,
) (GrupoConcurrenciaExperiencia, bool) {
	if !claveValida(clave) {
		return GrupoConcurrenciaExperiencia{}, false
	}
	for _, grupo := range c.gruposConcurrencia {
		if grupo.clave == clave {
			return grupo, true
		}
	}
	return GrupoConcurrenciaExperiencia{}, false
}

// ReglaExperienciaPorClave busca y devuelve una copia profunda.
func (c ConjuntoReglasBaremo) ReglaExperienciaPorClave(clave string) (ReglaExperiencia, bool) {
	if !claveValida(clave) {
		return ReglaExperiencia{}, false
	}
	for _, regla := range c.reglasExperiencia {
		if regla.clave == clave {
			return regla.clonar(), true
		}
	}
	return ReglaExperiencia{}, false
}

// Validar vuelve a comprobar las invariantes de la instantanea.
func (c ConjuntoReglasBaremo) Validar() error { return c.validar(true) }

func (c ConjuntoReglasBaremo) validar(exigirOrdenCanonico bool) error {
	if err := c.identidad.validar(); err != nil {
		return err
	}
	if err := c.bases.validar("bases"); err != nil {
		return err
	}
	if !c.fechaCorteInclusiva.EsValida() {
		return nuevoError("fecha_corte", CodigoValorInvalido)
	}
	if _, err := c.fechaCorteInclusiva.Siguiente(); err != nil {
		return nuevoError("fecha_corte", CodigoFueraDeLimites)
	}
	if len(c.secciones) == 0 || len(c.secciones) > maximoSeccionesPorConjunto {
		return nuevoError("secciones", CodigoFueraDeLimites)
	}
	if len(c.reglasExperiencia) == 0 || len(c.reglasExperiencia) > maximoReglasExperienciaConjunto {
		return nuevoError("reglas_experiencia", CodigoFueraDeLimites)
	}
	if len(c.gruposConcurrencia) == 0 || len(c.gruposConcurrencia) > maximoGruposConcurrencia {
		return nuevoError("grupos_concurrencia", CodigoFueraDeLimites)
	}
	if err := validarVolumenReglas(c.reglasExperiencia); err != nil {
		return err
	}

	seccionesPorClave := make(map[string]SeccionBaremo, len(c.secciones))
	clavesSeccion := make(map[string]struct{}, len(c.secciones))
	ordenesSeccion := make(map[uint32]struct{}, len(c.secciones))
	referencias := map[string]struct{}{
		c.identidad.referencia:      {},
		c.identidad.convocatoriaRef: {},
		c.identidad.expedienteRef:   {},
	}
	if _, duplicada := referencias[c.bases.referencia]; duplicada {
		return nuevoError("bases.referencia", CodigoValorDuplicado)
	}
	referencias[c.bases.referencia] = struct{}{}

	for indice, seccion := range c.secciones {
		if err := seccion.validar(); err != nil {
			return err
		}
		if _, duplicada := clavesSeccion[seccion.clave]; duplicada {
			return nuevoError("seccion.clave", CodigoValorDuplicado)
		}
		if _, duplicado := ordenesSeccion[seccion.orden]; duplicado {
			return nuevoError("seccion.orden", CodigoValorDuplicado)
		}
		if _, duplicada := referencias[seccion.definicion.referencia]; duplicada {
			return nuevoError("seccion.definicion.referencia", CodigoValorDuplicado)
		}
		if exigirOrdenCanonico && indice > 0 &&
			c.secciones[indice-1].orden >= seccion.orden {
			return nuevoError("secciones.orden", CodigoInvarianteQuebrada)
		}
		clavesSeccion[seccion.clave] = struct{}{}
		ordenesSeccion[seccion.orden] = struct{}{}
		referencias[seccion.definicion.referencia] = struct{}{}
		seccionesPorClave[seccion.clave] = seccion
	}

	gruposPorClave := make(map[string]GrupoConcurrenciaExperiencia, len(c.gruposConcurrencia))
	ordenesGrupo := make(map[uint32]struct{}, len(c.gruposConcurrencia))
	for indice, grupo := range c.gruposConcurrencia {
		if err := grupo.validar(); err != nil {
			return err
		}
		if _, duplicada := gruposPorClave[grupo.clave]; duplicada {
			return nuevoError("grupo_concurrencia.clave", CodigoValorDuplicado)
		}
		if _, duplicado := ordenesGrupo[grupo.orden]; duplicado {
			return nuevoError("grupo_concurrencia.orden", CodigoValorDuplicado)
		}
		if _, duplicada := referencias[grupo.definicion.referencia]; duplicada {
			return nuevoError("grupo_concurrencia.definicion.referencia", CodigoValorDuplicado)
		}
		if exigirOrdenCanonico && indice > 0 &&
			c.gruposConcurrencia[indice-1].orden >= grupo.orden {
			return nuevoError("grupos_concurrencia.orden", CodigoInvarianteQuebrada)
		}
		gruposPorClave[grupo.clave] = grupo
		ordenesGrupo[grupo.orden] = struct{}{}
		referencias[grupo.definicion.referencia] = struct{}{}
	}

	clavesRegla := make(map[string]struct{}, len(c.reglasExperiencia))
	ordenesPorSeccion := make(map[string]map[uint32]struct{}, len(c.secciones))
	reglasPorSeccion := make(map[string]int, len(c.secciones))
	reglasPorGrupo := make(map[string]int, len(c.gruposConcurrencia))
	prioridadesPorGrupo := make(map[string]map[uint32]struct{}, len(c.gruposConcurrencia))
	ordenSecciones := make(map[string]uint32, len(c.secciones))
	for _, seccion := range c.secciones {
		ordenSecciones[seccion.clave] = seccion.orden
	}
	for indice, regla := range c.reglasExperiencia {
		if err := regla.validar(); err != nil {
			return err
		}
		seccion, existe := seccionesPorClave[regla.seccionClave]
		if !existe {
			return nuevoError("regla.seccion_clave", CodigoSeccionDesconocida)
		}
		if _, existe := gruposPorClave[regla.grupoConcurrenciaClave]; !existe {
			return nuevoError("regla.grupo_concurrencia_clave", CodigoGrupoDesconocido)
		}
		if _, duplicada := clavesRegla[regla.clave]; duplicada {
			return nuevoError("regla.clave", CodigoValorDuplicado)
		}
		if _, duplicada := referencias[regla.definicion.referencia]; duplicada {
			return nuevoError("regla.definicion.referencia", CodigoValorDuplicado)
		}
		ordenes, existe := ordenesPorSeccion[regla.seccionClave]
		if !existe {
			ordenes = make(map[uint32]struct{})
			ordenesPorSeccion[regla.seccionClave] = ordenes
		}
		if _, duplicado := ordenes[regla.orden]; duplicado {
			return nuevoError("regla.orden", CodigoValorDuplicado)
		}
		prioridades, existe := prioridadesPorGrupo[regla.grupoConcurrenciaClave]
		if !existe {
			prioridades = make(map[uint32]struct{})
			prioridadesPorGrupo[regla.grupoConcurrenciaClave] = prioridades
		}
		if _, duplicada := prioridades[regla.prioridadConcurrencia]; duplicada {
			return nuevoError("regla.prioridad_concurrencia", CodigoValorDuplicado)
		}
		if limite, limitado := regla.maximoPuntos.Valor(); limitado &&
			limite.Micropuntos() > seccion.puntosMaximos.Micropuntos() {
			return nuevoError("regla.maximo_puntos", CodigoFueraDeLimites)
		}
		if exigirOrdenCanonico && indice > 0 {
			anterior := c.reglasExperiencia[indice-1]
			ordenSeccionAnterior := ordenSecciones[anterior.seccionClave]
			ordenSeccionActual := ordenSecciones[regla.seccionClave]
			if ordenSeccionAnterior > ordenSeccionActual ||
				(ordenSeccionAnterior == ordenSeccionActual && anterior.orden >= regla.orden) {
				return nuevoError("reglas_experiencia.orden", CodigoInvarianteQuebrada)
			}
		}
		clavesRegla[regla.clave] = struct{}{}
		referencias[regla.definicion.referencia] = struct{}{}
		ordenes[regla.orden] = struct{}{}
		prioridades[regla.prioridadConcurrencia] = struct{}{}
		reglasPorSeccion[regla.seccionClave]++
		reglasPorGrupo[regla.grupoConcurrenciaClave]++
	}
	for clave := range seccionesPorClave {
		if reglasPorSeccion[clave] == 0 {
			return nuevoError("seccion.reglas", CodigoPoliticaIncompleta)
		}
	}
	for clave := range gruposPorClave {
		if reglasPorGrupo[clave] == 0 {
			return nuevoError("grupo_concurrencia.reglas", CodigoPoliticaIncompleta)
		}
	}
	return nil
}

// RepresentacionCanonica devuelve un contrato JSON estable de esquema V1.
// No serializa directamente los campos privados del agregado.
func (c ConjuntoReglasBaremo) RepresentacionCanonica() ([]byte, error) {
	if err := c.validar(true); err != nil {
		return nil, err
	}
	material := materialConjunto{
		Esquema: esquemaConjuntoReglasBaremo,
		Identidad: materialIdentidad{
			Referencia:      c.identidad.referencia,
			Version:         c.identidad.version,
			ConvocatoriaRef: c.identidad.convocatoriaRef,
			ExpedienteRef:   c.identidad.expedienteRef,
		},
		Bases:               materialDeReferencia(c.bases),
		FechaCorteInclusiva: c.fechaCorteInclusiva,
		Secciones:           make([]materialSeccion, len(c.secciones)),
		GruposConcurrencia:  make([]materialGrupoConcurrencia, len(c.gruposConcurrencia)),
		ReglasExperiencia:   make([]materialReglaExperiencia, len(c.reglasExperiencia)),
	}
	for indice, seccion := range c.secciones {
		material.Secciones[indice] = materialSeccion{
			Clave:         seccion.clave,
			Definicion:    materialDeReferencia(seccion.definicion),
			Orden:         seccion.orden,
			PuntosMinimos: seccion.puntosMinimos,
			PuntosMaximos: seccion.puntosMaximos,
		}
	}
	for indice, grupo := range c.gruposConcurrencia {
		material.GruposConcurrencia[indice] = materialDeGrupoConcurrencia(grupo)
	}
	for indice, regla := range c.reglasExperiencia {
		material.ReglasExperiencia[indice] = materialDeRegla(regla)
	}
	contenido, err := json.Marshal(material)
	if err != nil || len(contenido) == 0 || len(contenido) > maximoBytesRepresentacion {
		return nil, nuevoError("representacion_canonica", CodigoFueraDeLimites)
	}
	return append([]byte(nil), contenido...), nil
}

// HuellaSHA256 calcula la huella hexadecimal minuscula de los bytes canonicos.
func (c ConjuntoReglasBaremo) HuellaSHA256() (string, error) {
	contenido, err := c.RepresentacionCanonica()
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

// ReferenciaVersionada devuelve la identidad del conjunto enlazada a su
// contenido exacto.
func (c ConjuntoReglasBaremo) ReferenciaVersionada() (ReferenciaVersionada, error) {
	huella, err := c.HuellaSHA256()
	if err != nil {
		return ReferenciaVersionada{}, err
	}
	return NuevaReferenciaVersionada(c.identidad.referencia, c.identidad.version, huella)
}

// MarshalJSON usa exactamente el contrato canonico, sin una segunda forma de
// serializacion accidental.
func (c ConjuntoReglasBaremo) MarshalJSON() ([]byte, error) {
	return c.RepresentacionCanonica()
}

type materialConjunto struct {
	Esquema             string                      `json:"esquema"`
	Identidad           materialIdentidad           `json:"identidad"`
	Bases               materialReferencia          `json:"bases"`
	FechaCorteInclusiva baremacion.FechaCivil       `json:"fecha_corte_inclusiva"`
	Secciones           []materialSeccion           `json:"secciones"`
	GruposConcurrencia  []materialGrupoConcurrencia `json:"grupos_concurrencia_experiencia"`
	ReglasExperiencia   []materialReglaExperiencia  `json:"reglas_experiencia"`
}

type materialIdentidad struct {
	Referencia      string `json:"referencia"`
	Version         uint64 `json:"version"`
	ConvocatoriaRef string `json:"convocatoria_ref"`
	ExpedienteRef   string `json:"expediente_ref"`
}

type materialReferencia struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type materialSeccion struct {
	Clave         string             `json:"clave"`
	Definicion    materialReferencia `json:"definicion"`
	Orden         uint32             `json:"orden"`
	PuntosMinimos baremacion.Puntos  `json:"puntos_minimos"`
	PuntosMaximos baremacion.Puntos  `json:"puntos_maximos"`
}

type materialCriterio struct {
	Clave    string             `json:"clave"`
	Catalogo materialReferencia `json:"catalogo"`
	Valores  []string           `json:"valores"`
}

type materialPoliticaUnidadTemporal struct {
	UnidadBase            UnidadTemporal          `json:"unidad_base"`
	UnidadPuntuable       UnidadTemporal          `json:"unidad_puntuable"`
	UnidadesBasePorUnidad baremacion.Racional     `json:"unidades_base_por_unidad"`
	ExtremoFinal          TratamientoExtremoFinal `json:"extremo_final"`
}

type materialPoliticaJornada struct {
	Modo   ModoJornada                 `json:"modo"`
	Umbral *baremacion.FraccionJornada `json:"umbral,omitempty"`
}

type materialPoliticaSolape struct {
	Modo   ModoSolape                  `json:"modo"`
	Limite *baremacion.FraccionJornada `json:"limite,omitempty"`
}

type materialPoliticaCoincidenciaReglas struct {
	Modo ModoCoincidenciaReglas `json:"modo"`
}

type materialPoliticaRepartoExceso struct {
	Modo                    ModoRepartoExceso       `json:"modo"`
	DesempateEntreReglas    CriterioDesempateExceso `json:"desempate_entre_reglas"`
	RepartoDentroMismaRegla ModoRepartoDentroRegla  `json:"reparto_dentro_misma_regla"`
}

type materialGrupoConcurrencia struct {
	Clave              string                             `json:"clave"`
	Definicion         materialReferencia                 `json:"definicion"`
	Orden              uint32                             `json:"orden"`
	CoincidenciaReglas materialPoliticaCoincidenciaReglas `json:"coincidencia_reglas"`
	Solape             materialPoliticaSolape             `json:"solape"`
	RepartoExceso      *materialPoliticaRepartoExceso     `json:"reparto_exceso,omitempty"`
}

type materialPoliticaRestos struct {
	Modo ModoRestos `json:"modo"`
}

type materialPoliticaRedondeo struct {
	Momento MomentoRedondeo         `json:"momento"`
	Modo    baremacion.ModoRedondeo `json:"modo"`
}

type materialLimitePuntos struct {
	Modo  modoLimite         `json:"modo"`
	Valor *baremacion.Puntos `json:"valor,omitempty"`
}

type materialLimiteUnidades struct {
	Modo  modoLimite           `json:"modo"`
	Valor *baremacion.Racional `json:"valor,omitempty"`
}

type materialReglaExperiencia struct {
	Clave                  string                         `json:"clave"`
	Definicion             materialReferencia             `json:"definicion"`
	SeccionClave           string                         `json:"seccion_clave"`
	Orden                  uint32                         `json:"orden"`
	Criterios              []materialCriterio             `json:"criterios"`
	GrupoConcurrenciaClave string                         `json:"grupo_concurrencia_clave"`
	PrioridadConcurrencia  uint32                         `json:"prioridad_concurrencia"`
	UnidadTemporal         materialPoliticaUnidadTemporal `json:"unidad_temporal"`
	Jornada                materialPoliticaJornada        `json:"jornada"`
	Restos                 materialPoliticaRestos         `json:"restos"`
	Redondeo               materialPoliticaRedondeo       `json:"redondeo"`
	PuntosPorUnidad        baremacion.Puntos              `json:"puntos_por_unidad"`
	MaximoUnidades         materialLimiteUnidades         `json:"maximo_unidades"`
	MaximoPuntos           materialLimitePuntos           `json:"maximo_puntos"`
}

func materialDeReferencia(referencia ReferenciaVersionada) materialReferencia {
	return materialReferencia{
		Referencia:   referencia.referencia,
		Version:      referencia.version,
		HuellaSHA256: referencia.huellaSHA256,
	}
}

func materialDeGrupoConcurrencia(grupo GrupoConcurrenciaExperiencia) materialGrupoConcurrencia {
	material := materialGrupoConcurrencia{
		Clave:              grupo.clave,
		Definicion:         materialDeReferencia(grupo.definicion),
		Orden:              grupo.orden,
		CoincidenciaReglas: materialPoliticaCoincidenciaReglas{Modo: grupo.coincidenciaReglas.modo},
		Solape:             materialPoliticaSolape{Modo: grupo.solape.modo},
	}
	if grupo.solape.tieneLimite {
		limite := grupo.solape.limite
		material.Solape.Limite = &limite
	}
	if grupo.tieneRepartoExceso {
		material.RepartoExceso = &materialPoliticaRepartoExceso{
			Modo:                    grupo.repartoExceso.modo,
			DesempateEntreReglas:    grupo.repartoExceso.desempateEntreReglas,
			RepartoDentroMismaRegla: grupo.repartoExceso.repartoDentroMismaRegla,
		}
	}
	return material
}

func materialDeRegla(regla ReglaExperiencia) materialReglaExperiencia {
	criterios := make([]materialCriterio, len(regla.criterios))
	for indice, criterio := range regla.criterios {
		criterios[indice] = materialCriterio{
			Clave:    criterio.clave,
			Catalogo: materialDeReferencia(criterio.catalogo),
			Valores:  append([]string(nil), criterio.valores...),
		}
	}

	jornada := materialPoliticaJornada{Modo: regla.jornada.modo}
	if regla.jornada.tieneUmbral {
		umbral := regla.jornada.umbral
		jornada.Umbral = &umbral
	}
	maximoUnidades := materialLimiteUnidades{Modo: regla.maximoUnidades.modo}
	if regla.maximoUnidades.EstaLimitado() {
		valor := regla.maximoUnidades.valor
		maximoUnidades.Valor = &valor
	}
	maximoPuntos := materialLimitePuntos{Modo: regla.maximoPuntos.modo}
	if regla.maximoPuntos.EstaLimitado() {
		valor := regla.maximoPuntos.valor
		maximoPuntos.Valor = &valor
	}

	return materialReglaExperiencia{
		Clave:                  regla.clave,
		Definicion:             materialDeReferencia(regla.definicion),
		SeccionClave:           regla.seccionClave,
		Orden:                  regla.orden,
		Criterios:              criterios,
		GrupoConcurrenciaClave: regla.grupoConcurrenciaClave,
		PrioridadConcurrencia:  regla.prioridadConcurrencia,
		UnidadTemporal: materialPoliticaUnidadTemporal{
			UnidadBase:            regla.unidadTemporal.unidadBase,
			UnidadPuntuable:       regla.unidadTemporal.unidadPuntuable,
			UnidadesBasePorUnidad: regla.unidadTemporal.unidadesBasePorUnidad,
			ExtremoFinal:          regla.unidadTemporal.extremoFinal,
		},
		Jornada: jornada,
		Restos:  materialPoliticaRestos{Modo: regla.restos.modo},
		Redondeo: materialPoliticaRedondeo{
			Momento: regla.redondeo.momento,
			Modo:    regla.redondeo.modo,
		},
		PuntosPorUnidad: regla.puntosPorUnidad,
		MaximoUnidades:  maximoUnidades,
		MaximoPuntos:    maximoPuntos,
	}
}

func clonarReglas(origen []ReglaExperiencia) []ReglaExperiencia {
	clon := make([]ReglaExperiencia, len(origen))
	for indice := range origen {
		clon[indice] = origen[indice].clonar()
	}
	return clon
}

func validarVolumenReglas(reglas []ReglaExperiencia) error {
	totalCriterios := 0
	totalValores := 0
	for _, regla := range reglas {
		if len(regla.criterios) > maximoCriteriosConjunto-totalCriterios {
			return nuevoError("reglas_experiencia.criterios", CodigoFueraDeLimites)
		}
		totalCriterios += len(regla.criterios)
		for _, criterio := range regla.criterios {
			if len(criterio.valores) > maximoValoresCriterioConjunto-totalValores {
				return nuevoError("reglas_experiencia.valores", CodigoFueraDeLimites)
			}
			totalValores += len(criterio.valores)
		}
	}
	return nil
}

func claveValida(valor string) bool {
	return valor == strings.TrimSpace(valor) && len(valor) > 0 &&
		len(valor) <= maximoCaracteresClave && patronClave.MatchString(valor)
}

func referenciaValida(valor string) bool {
	return valor == strings.TrimSpace(valor) && len(valor) > 0 &&
		len(valor) <= maximoCaracteresReferencia && patronReferencia.MatchString(valor)
}

func huellaSHA256Valida(valor string) bool {
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) {
		return false
	}
	_, err := hex.DecodeString(valor)
	return err == nil
}
