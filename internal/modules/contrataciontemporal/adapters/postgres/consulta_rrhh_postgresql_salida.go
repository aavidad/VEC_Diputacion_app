package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/diagnostico"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	reglasdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// salidaCierreConsultaRRHH conserva únicamente los escalares probatorios
// devueltos por CT-000045. No contiene la capacidad, el cursor de entrada ni
// identificadores de personas.
type salidaCierreConsultaRRHH struct {
	esquema                      string
	accesoRef                    string
	secuencia                    int64
	anteriorSHA256               string
	huellaSHA256                 string
	vinculoIdentidadHuellaSHA256 string
	alcanceHuellaSHA256          string
	registradaEn                 time.Time
	auditoriaRef                 string
	auditoriaHuellaSHA256        string
	consumoHuellaSHA256          string
	contenidoHuellaSHA256        string
	resultadoHuellaSHA256        string
	cursorHuellaSHA256           string
	generadaEn                   time.Time
	expedienteRef                string
	versionExpediente            int64
	total                        int16
	reciboSelloSHA256            string
}

type salidaCuadroConsultaRRHH struct {
	contenidoCanonico []byte
	cursorSiguiente   string
	cierre            salidaCierreConsultaRRHH
	totalFiltrado     int64
	enTramitacion     int64
	conIncidencia     int64
	enLlamamiento     int64
	// CT-000110: entrada en la fase actual, en el orden del canon.
	faseDesdeExpedientes []string
	faseDesdeInstantes   []time.Time
	// CT-000125: si cada expediente consta como urgente, en el mismo orden.
	urgentes          []bool
	instantaneasRegla []byte
	basesRegla        []byte
	ajustesRegla      []byte
}

// El límite incluye los tres contextos de la página. SQL limita conjuntamente
// los dos diccionarios; aquí se vuelve a imponer antes de decodificarlos.
const maximoBytesContextoPlazosRRHH = 4 << 20

type vinculoInstantaneaRRHH struct {
	Estado              string     `json:"estado"`
	ExpedienteRef       string     `json:"expediente_ref"`
	VersionExpediente   uint64     `json:"version_expediente"`
	Fase                string     `json:"fase"`
	FaseDesde           time.Time  `json:"fase_desde"`
	CatalogoBaseID      string     `json:"catalogo_base_id"`
	BaseVersion         int        `json:"base_version"`
	BaseHuellaSHA256    string     `json:"base_huella_sha256"`
	CatalogoAjustesID   string     `json:"catalogo_ajustes_id"`
	AjustesEncontrados  *bool      `json:"ajustes_encontrados"`
	AjustesVersion      int        `json:"ajustes_version"`
	AjustesHuellaSHA256 string     `json:"ajustes_huella_sha256"`
	AjustesVigenteDesde *time.Time `json:"ajustes_vigente_desde"`
	CapturadaEn         time.Time  `json:"capturada_en"`
}

type definicionBaseRRHH struct {
	CatalogoBaseID   string `json:"catalogo_base_id"`
	BaseVersion      int    `json:"base_version"`
	BaseHuellaSHA256 string `json:"base_huella_sha256"`
	BaseCanonico     string `json:"base_canonico"`
}

type definicionAjustesRRHH struct {
	CatalogoAjustesID   string `json:"catalogo_ajustes_id"`
	AjustesVersion      int    `json:"ajustes_version"`
	AjustesHuellaSHA256 string `json:"ajustes_huella_sha256"`
	AjustesCanonico     string `json:"ajustes_canonico"`
}

type claveContextoPlazosRRHH struct {
	id      string
	version int
	huella  string
}

// instantaneasAlineadas rehidrata únicamente los contextos citados por esta
// página. Los tres bloques provienen de la misma lectura SQL v5 y nunca se
// sustituyen por las definiciones actualmente vigentes.
func (s salidaCuadroConsultaRRHH) instantaneasAlineadas(
	resumenes []ports.ResumenExpedienteRRHH, fasesDesde []time.Time,
) ([]*reglas.InstantaneaPersistidaRegla, error) {
	fallo := ports.ErrResultadoConsultaRRHHNoConfiable
	if len(s.instantaneasRegla) == 0 || len(s.basesRegla) == 0 ||
		len(s.ajustesRegla) == 0 ||
		len(s.instantaneasRegla) > maximoBytesContextoPlazosRRHH ||
		len(s.basesRegla)+len(s.ajustesRegla) > maximoBytesContextoPlazosRRHH ||
		len(fasesDesde) != len(resumenes) {
		return nil, fallo
	}
	var vinculos []vinculoInstantaneaRRHH
	var bases []definicionBaseRRHH
	var ajustes []definicionAjustesRRHH
	if err := decodificarArrayPlazosRRHH(s.instantaneasRegla, &vinculos); err != nil {
		return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
	}
	if err := decodificarArrayPlazosRRHH(s.basesRegla, &bases); err != nil {
		return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
	}
	if err := decodificarArrayPlazosRRHH(s.ajustesRegla, &ajustes); err != nil {
		return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
	}
	if len(vinculos) != len(resumenes) ||
		len(bases) > len(resumenes) || len(ajustes) > len(resumenes) {
		return nil, fallo
	}
	basesPorClave := make(map[claveContextoPlazosRRHH][]byte, len(bases))
	usosBase := make(map[claveContextoPlazosRRHH]int, len(bases))
	for _, base := range bases {
		clave := claveContextoPlazosRRHH{base.CatalogoBaseID, base.BaseVersion, base.BaseHuellaSHA256}
		if clave.id == "" || clave.version < 1 || len(base.BaseCanonico) == 0 ||
			len(base.BaseCanonico) > 1<<20 || basesPorClave[clave] != nil {
			return nil, fallo
		}
		var catalogo reglasdomain.CatalogoConfigurable
		if err := json.Unmarshal([]byte(base.BaseCanonico), &catalogo); err != nil {
			return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
		}
		if err := reglas.ValidarCatalogoBaseReglas(catalogo); err != nil {
			return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
		}
		canonico, huella, err := reglas.CanonicoCatalogoBaseReglas(catalogo)
		if err != nil {
			return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
		}
		if !bytes.Equal(canonico, []byte(base.BaseCanonico)) ||
			catalogo.ID != clave.id || catalogo.Version != clave.version ||
			huella != clave.huella {
			return nil, fallo
		}
		basesPorClave[clave] = canonico
	}
	ajustesPorClave := make(map[claveContextoPlazosRRHH][]byte, len(ajustes))
	usosAjustes := make(map[claveContextoPlazosRRHH]int, len(ajustes))
	for _, ajuste := range ajustes {
		clave := claveContextoPlazosRRHH{ajuste.CatalogoAjustesID, ajuste.AjustesVersion, ajuste.AjustesHuellaSHA256}
		if clave.id == "" || clave.version < 0 || len(ajuste.AjustesCanonico) == 0 ||
			len(ajuste.AjustesCanonico) > 16<<10 || ajustesPorClave[clave] != nil {
			return nil, fallo
		}
		var datos map[string]map[string]string
		if err := json.Unmarshal([]byte(ajuste.AjustesCanonico), &datos); err != nil {
			return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
		}
		if datos == nil {
			return nil, fallo
		}
		canonico, err := reglas.CanonicoAjustes(datos)
		if err != nil {
			return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: err}
		}
		huella, errHuella := reglas.HuellaAjustes(datos)
		if errHuella != nil {
			return nil, &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: fallo, Causa: errHuella}
		}
		if !bytes.Equal(canonico, []byte(ajuste.AjustesCanonico)) || huella != clave.huella {
			return nil, fallo
		}
		ajustesPorClave[clave] = canonico
	}
	resultado := make([]*reglas.InstantaneaPersistidaRegla, len(resumenes))
	for indice, vinculo := range vinculos {
		if vinculo.ExpedienteRef != resumenes[indice].ExpedienteRef ||
			vinculo.VersionExpediente != resumenes[indice].Version ||
			vinculo.Fase != string(resumenes[indice].FaseClave) ||
			!vinculo.FaseDesde.Equal(fasesDesde[indice]) {
			return nil, fallo
		}
		if vinculo.Estado == "legado_sin_instantanea" {
			if vinculo.CatalogoBaseID != "" || vinculo.CatalogoAjustesID != "" ||
				vinculo.BaseVersion != 0 || vinculo.AjustesVersion != 0 ||
				vinculo.BaseHuellaSHA256 != "" || vinculo.AjustesHuellaSHA256 != "" ||
				vinculo.AjustesEncontrados != nil || vinculo.AjustesVigenteDesde != nil ||
				!vinculo.CapturadaEn.IsZero() {
				return nil, fallo
			}
			continue
		}
		if vinculo.Estado != "capturada" || vinculo.AjustesEncontrados == nil ||
			vinculo.CapturadaEn.IsZero() || vinculo.CapturadaEn.After(s.cierre.generadaEn) ||
			vinculo.CatalogoAjustesID != reglas.CatalogoAjustesDe(vinculo.CatalogoBaseID) {
			return nil, fallo
		}
		claveBase := claveContextoPlazosRRHH{vinculo.CatalogoBaseID, vinculo.BaseVersion, vinculo.BaseHuellaSHA256}
		claveAjustes := claveContextoPlazosRRHH{vinculo.CatalogoAjustesID, vinculo.AjustesVersion, vinculo.AjustesHuellaSHA256}
		base, existeBase := basesPorClave[claveBase]
		ajuste, existeAjuste := ajustesPorClave[claveAjustes]
		if !existeBase || !existeAjuste {
			return nil, fallo
		}
		usosBase[claveBase]++
		usosAjustes[claveAjustes]++
		var desdeAjustes time.Time
		if vinculo.AjustesVigenteDesde != nil {
			desdeAjustes = vinculo.AjustesVigenteDesde.UTC()
		}
		instantanea := &reglas.InstantaneaPersistidaRegla{
			CatalogoBaseID: vinculo.CatalogoBaseID, CatalogoBaseVersion: vinculo.BaseVersion,
			CatalogoBaseHuella:   vinculo.BaseHuellaSHA256,
			CatalogoBaseCanonico: append([]byte(nil), base...),
			CatalogoAjustesID:    vinculo.CatalogoAjustesID,
			AjustesEncontrados:   *vinculo.AjustesEncontrados,
			VersionAjustes:       vinculo.AjustesVersion,
			HuellaAjustes:        vinculo.AjustesHuellaSHA256,
			CanonicoAjustes:      append([]byte(nil), ajuste...),
			AjustesVigenteDesde:  desdeAjustes,
			PreparadaEn:          vinculo.CapturadaEn.UTC(),
			Fase:                 vinculo.Fase, FaseDesde: vinculo.FaseDesde.UTC(),
		}
		if _, err := reglas.RehidratarInstantaneaRegla(*instantanea); err != nil &&
			!errors.Is(err, reglas.ErrReglaNoEncontrada) {
			return nil, fallo
		}
		resultado[indice] = instantanea
	}
	for clave := range basesPorClave {
		if usosBase[clave] == 0 {
			return nil, fallo
		}
	}
	for clave := range ajustesPorClave {
		if usosAjustes[clave] == 0 {
			return nil, fallo
		}
	}
	return resultado, nil
}

func decodificarArrayPlazosRRHH[T any](contenido []byte, destino *[]T) error {
	if len(contenido) < 2 || contenido[0] != '[' {
		return ports.ErrResultadoConsultaRRHHNoConfiable
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return err
	}
	if len(*destino) == 0 && !bytes.Equal(contenido, []byte("[]")) {
		return ports.ErrResultadoConsultaRRHHNoConfiable
	}
	var resto any
	if err := decodificador.Decode(&resto); err != io.EOF {
		return ports.ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

// urgentesAlineados devuelve la urgencia declarada de cada resumen, alineada
// con los expedientes ya comprobados por fasesDesde. Una cardinalidad distinta
// es no confiable; sin ningún urgente devuelve nil.
func (s salidaCuadroConsultaRRHH) urgentesAlineados(
	resumenes []ports.ResumenExpedienteRRHH,
) ([]bool, error) {
	if len(s.urgentes) != len(resumenes) {
		return nil, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	for _, urgente := range s.urgentes {
		if urgente {
			return append([]bool(nil), s.urgentes...), nil
		}
	}
	return nil, nil
}

// fasesDesde alinea la fecha de entrada en fase con los resúmenes ya
// analizados; cualquier desajuste de orden o cardinalidad es no confiable.
func (s salidaCuadroConsultaRRHH) fasesDesde(
	resumenes []ports.ResumenExpedienteRRHH,
) ([]time.Time, error) {
	if len(s.faseDesdeExpedientes) != len(resumenes) ||
		len(s.faseDesdeInstantes) != len(resumenes) {
		return nil, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	if len(resumenes) == 0 {
		return nil, nil
	}
	fases := make([]time.Time, len(resumenes))
	for indice, resumen := range resumenes {
		if s.faseDesdeExpedientes[indice] != resumen.ExpedienteRef {
			return nil, ports.ErrResultadoConsultaRRHHNoConfiable
		}
		fases[indice] = s.faseDesdeInstantes[indice].UTC()
	}
	return fases, nil
}

func (s salidaCuadroConsultaRRHH) construirTotales() (*ports.TotalesCuadroRRHH, error) {
	const maximoEnteroJSONSeguro = int64(9_007_199_254_740_991)
	if s.totalFiltrado < 0 || s.enTramitacion < 0 ||
		s.conIncidencia < 0 || s.enLlamamiento < 0 ||
		s.totalFiltrado > maximoEnteroJSONSeguro ||
		s.enTramitacion > s.totalFiltrado ||
		s.conIncidencia > s.totalFiltrado ||
		s.enLlamamiento > s.totalFiltrado {
		return nil, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	return &ports.TotalesCuadroRRHH{
		Total:         uint64(s.totalFiltrado),
		EnTramitacion: uint64(s.enTramitacion),
		ConIncidencia: uint64(s.conIncidencia),
		EnLlamamiento: uint64(s.enLlamamiento),
	}, nil
}

// timestamptz conserva un instante absoluto, pero pgx puede entregarlo con
// time.Local. Cambiar su representación a UTC no altera el instante ni su
// precisión; las validaciones del recibo siguen rechazando valores inválidos.
func (s *salidaCierreConsultaRRHH) normalizarInstantesSQL() {
	s.registradaEn = s.registradaEn.UTC()
	s.generadaEn = s.generadaEn.UTC()
}

type salidaDetalleConsultaRRHH struct {
	contenidoCanonico []byte
	cierre            salidaCierreConsultaRRHH
}

// analizadorCanonConsultaRRHH es el límite privado con el analizador del
// canon binario CT-000042. El analizador no recibe ni fabrica autoridad.
type analizadorCanonConsultaRRHH interface {
	analizarCuadro(
		[]byte,
		string,
		time.Time,
		uint16,
	) (ports.PaginaCuadroRRHH, error)
	analizarDetalle(
		[]byte,
		time.Time,
		string,
		uint64,
	) (ports.EntradaDetalleExpedienteRRHHMinimizada, error)
}

type analizadorCanonConsultaRRHHPostgreSQL struct{}

func (analizadorCanonConsultaRRHHPostgreSQL) analizarCuadro(
	contenido []byte,
	cursor string,
	generadaEn time.Time,
	total uint16,
) (ports.PaginaCuadroRRHH, error) {
	decodificado, err := decodificarContenidoCuadroRRHHPostgreSQL(contenido)
	if err != nil ||
		!decodificado.paginaSinRecibo.GeneradaEn.Equal(generadaEn) ||
		len(decodificado.paginaSinRecibo.Expedientes) != int(total) {
		return ports.PaginaCuadroRRHH{},
			&diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaResultadoSQL, Sentinela: ports.ErrResultadoConsultaRRHHNoConfiable, Causa: err}
	}
	pagina := decodificado.paginaSinRecibo
	if pagina.HayMas {
		materialCursor, err := base64.RawURLEncoding.Strict().DecodeString(cursor)
		if err != nil {
			return ports.PaginaCuadroRRHH{},
				&diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaCursorDecod, Sentinela: ports.ErrResultadoConsultaRRHHNoConfiable, Causa: err}
		}
		defer clear(materialCursor)
		huellaCursor := sha256.Sum256([]byte(cursor))
		if len(materialCursor) != sha256.Size ||
			base64.RawURLEncoding.EncodeToString(materialCursor) != cursor ||
			!bytes.Equal(huellaCursor[:], decodificado.cursorHuella[:]) {
			return ports.PaginaCuadroRRHH{},
				&diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaCursorHuella, Sentinela: ports.ErrResultadoConsultaRRHHNoConfiable}
		}
		pagina.CursorSiguiente = cursor
	} else if cursor != "" ||
		decodificado.cursorHuella != ([sha256.Size]byte{}) {
		return ports.PaginaCuadroRRHH{},
			&diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaCursorDecod, Sentinela: ports.ErrResultadoConsultaRRHHNoConfiable}
	}
	return pagina, nil
}

func (analizadorCanonConsultaRRHHPostgreSQL) analizarDetalle(
	contenido []byte,
	generadaEn time.Time,
	expedienteRef string,
	version uint64,
) (ports.EntradaDetalleExpedienteRRHHMinimizada, error) {
	decodificado, err := decodificarContenidoDetalleRRHHPostgreSQL(contenido)
	if err != nil || decodificado.expedienteRef != expedienteRef ||
		decodificado.version != version {
		return ports.EntradaDetalleExpedienteRRHHMinimizada{},
			ports.ErrResultadoConsultaRRHHNoConfiable
	}
	reconstruido, err :=
		decodificado.entrada.ExportarContenidoCanonicoParaSQL(generadaEn)
	if err != nil || !bytes.Equal(reconstruido.BytesCanonicos(), contenido) {
		return ports.EntradaDetalleExpedienteRRHHMinimizada{},
			ports.ErrResultadoConsultaRRHHNoConfiable
	}
	return decodificado.entrada, nil
}

func (s salidaCierreConsultaRRHH) construirRecibo(
	ordenContexto ports.ContextoConsultaRRHH,
	capacidad ports.CapacidadConsultaRRHH,
) (ports.ReciboLecturaRRHH, error) {
	secuencia, version, total, err := s.enterosSeguros()
	if err != nil {
		return ports.ReciboLecturaRRHH{},
			ports.ErrResultadoConsultaRRHHNoConfiable
	}
	registro, err := ports.NuevoResultadoRegistradorAccesoRRHHV2(
		s.esquema,
		s.accesoRef,
		secuencia,
		s.anteriorSHA256,
		s.huellaSHA256,
		s.vinculoIdentidadHuellaSHA256,
		s.alcanceHuellaSHA256,
		s.registradaEn,
	)
	if err != nil {
		return ports.ReciboLecturaRRHH{},
			ports.ErrResultadoConsultaRRHHNoConfiable
	}
	evidencia, err := ports.NuevaEvidenciaConsumoResultadoRRHHV2(
		s.auditoriaRef,
		s.auditoriaHuellaSHA256,
		s.consumoHuellaSHA256,
		s.contenidoHuellaSHA256,
		s.resultadoHuellaSHA256,
		s.cursorHuellaSHA256,
		s.generadaEn,
		s.expedienteRef,
		version,
		total,
	)
	if err != nil {
		return ports.ReciboLecturaRRHH{},
			ports.ErrResultadoConsultaRRHHNoConfiable
	}
	return ports.NuevoReciboLecturaRRHHV2(
		ordenContexto,
		capacidad,
		registro,
		evidencia,
		s.reciboSelloSHA256,
	)
}

func (s salidaCierreConsultaRRHH) enterosSeguros() (
	uint64,
	uint64,
	uint16,
	error,
) {
	const maximoEnteroJSONSeguro = int64(9_007_199_254_740_991)
	if s.secuencia < 1 || s.secuencia > maximoEnteroJSONSeguro ||
		s.versionExpediente < 0 ||
		s.versionExpediente > maximoEnteroJSONSeguro ||
		s.total < 0 || s.total > ports.LimiteMaximoCuadroRRHH {
		return 0, 0, 0, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	return uint64(s.secuencia), uint64(s.versionExpediente),
		uint16(s.total), nil
}
