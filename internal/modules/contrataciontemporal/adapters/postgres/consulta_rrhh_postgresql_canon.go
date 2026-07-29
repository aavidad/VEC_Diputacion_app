package postgres

import (
	"bytes"
	"errors"
	"strconv"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	cabeceraContenidoCuadroRRHHPostgreSQL  = "VEC-CT-CONTENIDO-CUADRO-RRHH-V1\n"
	cabeceraContenidoDetalleRRHHPostgreSQL = "VEC-CT-CONTENIDO-DETALLE-RRHH-V1\n"
	formatoInstanteCanonicoRRHHPostgreSQL  = "2006-01-02T15:04:05.000000Z"
)

var errCanonConsultaRRHHPostgreSQL = errors.New(
	"contratacion temporal: canon de consulta RRHH no confiable",
)

// contenidoCuadroRRHHDecodificado conserva únicamente la proyección de una
// página y la huella binaria del cursor. El cursor en claro procede de otra
// columna de la fachada y se cruzará después, nunca se reconstruye aquí.
type contenidoCuadroRRHHDecodificado struct {
	paginaSinRecibo ports.PaginaCuadroRRHH
	cursorHuella    [32]byte
}

// contenidoDetalleRRHHDecodificado transporta la entrada nominal mínima que
// admite el constructor del puerto, más las dos claves necesarias para cruzar
// después las columnas probatorias de la fachada.
type contenidoDetalleRRHHDecodificado struct {
	entrada       ports.EntradaDetalleExpedienteRRHHMinimizada
	expedienteRef string
	version       uint64
	actualizadoEn time.Time
}

func decodificarContenidoCuadroRRHHPostgreSQL(
	canon []byte,
) (contenidoCuadroRRHHDecodificado, error) {
	lector, err := nuevoLectorCanonRRHHPostgreSQL(
		canon,
		cabeceraContenidoCuadroRRHHPostgreSQL,
	)
	if err != nil {
		return contenidoCuadroRRHHDecodificado{}, err
	}
	generadaEn, err := lector.instante()
	if err != nil {
		return contenidoCuadroRRHHDecodificado{}, err
	}
	total, err := lector.enteroSinSigno()
	if err != nil || total > ports.LimiteMaximoCuadroRRHH ||
		!lector.cabenMarcos(total, 15) {
		return contenidoCuadroRRHHDecodificado{},
			errCanonConsultaRRHHPostgreSQL
	}
	resumenes := make([]ports.ResumenExpedienteRRHH, int(total))
	for indice := range resumenes {
		resumenes[indice], err = lector.resumen()
		if err != nil {
			return contenidoCuadroRRHHDecodificado{}, err
		}
	}
	if !resumenesCuadroRRHHPostgreSQLValidos(resumenes, generadaEn) {
		return contenidoCuadroRRHHDecodificado{},
			errCanonConsultaRRHHPostgreSQL
	}
	hayMas, err := lector.booleano()
	if err != nil {
		return contenidoCuadroRRHHDecodificado{}, err
	}
	cursorMaterial, err := lector.marco()
	if err != nil || hayMas && len(cursorMaterial) != 32 ||
		!hayMas && len(cursorMaterial) != 0 ||
		hayMas && bytes.Equal(cursorMaterial, make([]byte, 32)) ||
		!lector.consumido() {
		return contenidoCuadroRRHHDecodificado{},
			errCanonConsultaRRHHPostgreSQL
	}
	var cursorHuella [32]byte
	copy(cursorHuella[:], cursorMaterial)
	pagina := ports.PaginaCuadroRRHH{
		GeneradaEn:  generadaEn,
		Expedientes: resumenes,
		HayMas:      hayMas,
	}
	// Sin cursor, el constructor público permite una comprobación byte a
	// byte inmediata. Con cursor, el texto claro llega en otra columna de la
	// fachada y la misma comprobación se aplaza hasta cruzar ambas salidas.
	if !hayMas {
		exportacion, err := pagina.ExportarContenidoCanonicoParaSQL()
		if err != nil || !bytes.Equal(exportacion.BytesCanonicos(), canon) {
			return contenidoCuadroRRHHDecodificado{},
				errCanonConsultaRRHHPostgreSQL
		}
	}
	return contenidoCuadroRRHHDecodificado{
		paginaSinRecibo: pagina,
		cursorHuella:    cursorHuella,
	}, nil
}

func decodificarContenidoDetalleRRHHPostgreSQL(
	canon []byte,
) (contenidoDetalleRRHHDecodificado, error) {
	lector, err := nuevoLectorCanonRRHHPostgreSQL(
		canon,
		cabeceraContenidoDetalleRRHHPostgreSQL,
	)
	if err != nil {
		return contenidoDetalleRRHHDecodificado{}, err
	}
	resumen, err := lector.resumen()
	if err != nil {
		return contenidoDetalleRRHHDecodificado{}, err
	}
	solicitud, err := lector.solicitud()
	if err != nil {
		return contenidoDetalleRRHHDecodificado{}, err
	}
	mascara, err := lector.enteroSinSigno()
	if err != nil || mascara != 0 && mascara != 1 &&
		mascara != 3 && mascara != 7 {
		return contenidoDetalleRRHHDecodificado{},
			errCanonConsultaRRHHPostgreSQL
	}

	analisis, referenciaAnalisis, err := lector.analisis()
	if err != nil {
		return contenidoDetalleRRHHDecodificado{}, err
	}
	cobertura, referenciaCobertura, err := lector.cobertura()
	if err != nil {
		return contenidoDetalleRRHHDecodificado{}, err
	}
	asignacion, referenciaAsignacion, err := lector.asignacion()
	if err != nil {
		return contenidoDetalleRRHHDecodificado{}, err
	}
	mascaraReal := uint64(0)
	if analisis != nil {
		mascaraReal |= 1
	}
	if cobertura != nil {
		mascaraReal |= 2
	}
	if asignacion != nil {
		mascaraReal |= 4
	}
	if mascara != mascaraReal {
		return contenidoDetalleRRHHDecodificado{},
			errCanonConsultaRRHHPostgreSQL
	}

	totalHitos, err := lector.enteroSinSigno()
	if err != nil || totalHitos != resumen.Version ||
		!lector.cabenMarcos(totalHitos, 8) {
		return contenidoDetalleRRHHDecodificado{},
			errCanonConsultaRRHHPostgreSQL
	}
	hitos := make([]ports.HitoExpedienteRRHH, int(totalHitos))
	for indice := range hitos {
		hitos[indice], err = lector.hito()
		if err != nil {
			return contenidoDetalleRRHHDecodificado{}, err
		}
	}
	if !lector.consumido() {
		return contenidoDetalleRRHHDecodificado{},
			errCanonConsultaRRHHPostgreSQL
	}
	entrada, err := ports.NuevaEntradaDetalleExpedienteRRHHMinimizada(
		resumen, solicitud,
		analisis, referenciaAnalisis,
		cobertura, referenciaCobertura,
		asignacion, referenciaAsignacion,
		hitos,
	)
	if err != nil {
		return contenidoDetalleRRHHDecodificado{},
			errCanonConsultaRRHHPostgreSQL
	}
	return contenidoDetalleRRHHDecodificado{
		entrada:       entrada,
		expedienteRef: resumen.ExpedienteRef,
		version:       resumen.Version,
		actualizadoEn: resumen.ActualizadoEn,
	}, nil
}

type lectorCanonRRHHPostgreSQL struct {
	contenido []byte
	posicion  int
}

func nuevoLectorCanonRRHHPostgreSQL(
	canon []byte,
	cabecera string,
) (*lectorCanonRRHHPostgreSQL, error) {
	if len(canon) <= len(cabecera) ||
		len(canon) > ports.LimiteMaximoCanonResultadoRRHH ||
		!bytes.HasPrefix(canon, []byte(cabecera)) {
		return nil, errCanonConsultaRRHHPostgreSQL
	}
	return &lectorCanonRRHHPostgreSQL{
		contenido: canon,
		posicion:  len(cabecera),
	}, nil
}

func (l *lectorCanonRRHHPostgreSQL) marco() ([]byte, error) {
	if l == nil || l.posicion >= len(l.contenido) {
		return nil, errCanonConsultaRRHHPostgreSQL
	}
	inicio := l.posicion
	separadorRelativo := bytes.IndexByte(l.contenido[inicio:], ':')
	if separadorRelativo <= 0 || separadorRelativo > 6 {
		return nil, errCanonConsultaRRHHPostgreSQL
	}
	separador := inicio + separadorRelativo
	digitos := l.contenido[inicio:separador]
	if !decimalCanonicoRRHHPostgreSQL(digitos) {
		return nil, errCanonConsultaRRHHPostgreSQL
	}
	longitud, err := strconv.ParseUint(string(digitos), 10, 32)
	if err != nil || longitud > uint64(ports.LimiteMaximoCanonResultadoRRHH) {
		return nil, errCanonConsultaRRHHPostgreSQL
	}
	inicioValor := separador + 1
	finValor := inicioValor + int(longitud)
	if finValor < inicioValor || finValor >= len(l.contenido) ||
		l.contenido[finValor] != '\n' {
		return nil, errCanonConsultaRRHHPostgreSQL
	}
	l.posicion = finValor + 1
	return l.contenido[inicioValor:finValor], nil
}

func decimalCanonicoRRHHPostgreSQL(valor []byte) bool {
	if len(valor) == 0 || len(valor) > 1 && valor[0] == '0' {
		return false
	}
	for _, caracter := range valor {
		if caracter < '0' || caracter > '9' {
			return false
		}
	}
	return true
}

func (l *lectorCanonRRHHPostgreSQL) texto() (string, error) {
	valor, err := l.marco()
	if err != nil || !utf8.Valid(valor) {
		return "", errCanonConsultaRRHHPostgreSQL
	}
	return string(valor), nil
}

func (l *lectorCanonRRHHPostgreSQL) enteroSinSigno() (uint64, error) {
	texto, err := l.texto()
	if err != nil || !decimalCanonicoRRHHPostgreSQL([]byte(texto)) {
		return 0, errCanonConsultaRRHHPostgreSQL
	}
	valor, err := strconv.ParseUint(texto, 10, 64)
	if err != nil || valor > 9_007_199_254_740_991 {
		return 0, errCanonConsultaRRHHPostgreSQL
	}
	return valor, nil
}

func (l *lectorCanonRRHHPostgreSQL) enteroConSigno() (int64, error) {
	texto, err := l.texto()
	if err != nil || texto == "" || texto == "-0" || texto[0] == '+' {
		return 0, errCanonConsultaRRHHPostgreSQL
	}
	digitos := texto
	if texto[0] == '-' {
		digitos = texto[1:]
	}
	if !decimalCanonicoRRHHPostgreSQL([]byte(digitos)) {
		return 0, errCanonConsultaRRHHPostgreSQL
	}
	valor, err := strconv.ParseInt(texto, 10, 64)
	if err != nil {
		return 0, errCanonConsultaRRHHPostgreSQL
	}
	return valor, nil
}

func (l *lectorCanonRRHHPostgreSQL) booleano() (bool, error) {
	texto, err := l.texto()
	if err != nil || texto != "0" && texto != "1" {
		return false, errCanonConsultaRRHHPostgreSQL
	}
	return texto == "1", nil
}

func (l *lectorCanonRRHHPostgreSQL) instante() (time.Time, error) {
	texto, err := l.texto()
	if err != nil {
		return time.Time{}, errCanonConsultaRRHHPostgreSQL
	}
	instante, err := time.Parse(formatoInstanteCanonicoRRHHPostgreSQL, texto)
	if err != nil || instante.Location() != time.UTC ||
		instante.Format(formatoInstanteCanonicoRRHHPostgreSQL) != texto {
		return time.Time{}, errCanonConsultaRRHHPostgreSQL
	}
	return instante, nil
}

func (l *lectorCanonRRHHPostgreSQL) resumen() (
	ports.ResumenExpedienteRRHH,
	error,
) {
	expedienteRef, e1 := l.texto()
	organizacionRef, e2 := l.texto()
	numeroVisible, e3 := l.texto()
	version, e4 := l.enteroSinSigno()
	flujoRef, e5 := l.texto()
	flujoVersion, e6 := l.enteroSinSigno()
	flujoHuella, e7 := l.texto()
	faseClave, e8 := l.texto()
	estadoClave, e9 := l.texto()
	centroRef, e10 := l.texto()
	categoriaRef, e11 := l.texto()
	modalidadClave, e12 := l.texto()
	unidadRef, e13 := l.texto()
	creadoEn, e14 := l.instante()
	actualizadoEn, e15 := l.instante()
	if algunErrorCanonRRHHPostgreSQL(
		e1, e2, e3, e4, e5, e6, e7, e8, e9, e10, e11, e12, e13, e14, e15,
	) {
		return ports.ResumenExpedienteRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	resumen := ports.ResumenExpedienteRRHH{
		ExpedienteRef: expedienteRef, OrganizacionRef: organizacionRef,
		NumeroVisible: numeroVisible, Version: version,
		FlujoRef: flujoRef, FlujoVersion: flujoVersion,
		FlujoHuella: flujoHuella, FaseClave: domain.ClaveFase(faseClave),
		EstadoClave: domain.EstadoOperativo(estadoClave),
		CentroRef:   centroRef, CategoriaRef: categoriaRef,
		ModalidadClave: domain.ClaveCatalogo(modalidadClave),
		UnidadRef:      unidadRef, CreadoEn: creadoEn, ActualizadoEn: actualizadoEn,
	}
	if resumen.Validar() != nil {
		return ports.ResumenExpedienteRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	return resumen, nil
}

func (l *lectorCanonRRHHPostgreSQL) solicitud() (
	ports.SolicitudOperativaRRHH,
	error,
) {
	grupo, e1 := l.texto()
	motivo, e2 := l.texto()
	inicio, e3 := l.instante()
	fin, e4 := l.instante()
	if algunErrorCanonRRHHPostgreSQL(e1, e2, e3, e4) {
		return ports.SolicitudOperativaRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	return ports.SolicitudOperativaRRHH{
		GrupoSubgrupo: grupo, MotivoClave: domain.ClaveCatalogo(motivo),
		PeriodoInicio: inicio, PeriodoFin: fin,
	}, nil
}

func (l *lectorCanonRRHHPostgreSQL) analisis() (
	*ports.AnalisisOperativoRRHH,
	ports.ReferenciaHitoAnalisisRRHH,
	error,
) {
	presente, err := l.booleano()
	if err != nil {
		return nil, ports.ReferenciaHitoAnalisisRRHH{}, err
	}
	secuencia, err := l.enteroSinSigno()
	if err != nil || !presente && secuencia != 0 {
		return nil, ports.ReferenciaHitoAnalisisRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	if !presente {
		return nil, ports.ReferenciaHitoAnalisisRRHH{}, nil
	}
	referencia, err := ports.NuevaReferenciaHitoAnalisisRRHH(secuencia)
	if err != nil {
		return nil, ports.ReferenciaHitoAnalisisRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	modalidad, e1 := l.texto()
	categoria, e2 := l.texto()
	causa, e3 := l.texto()
	inicio, e4 := l.instante()
	fin, e5 := l.instante()
	jornada, e6 := l.enteroSinSigno()
	resultadoRC, e7 := l.texto()
	costePresente, e8 := l.booleano()
	if algunErrorCanonRRHHPostgreSQL(e1, e2, e3, e4, e5, e6, e7, e8) ||
		jornada > 10_000 {
		return nil, ports.ReferenciaHitoAnalisisRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	var coste *ports.ImporteOperativoRRHH
	if costePresente {
		centimos, errCentimos := l.enteroConSigno()
		moneda, errMoneda := l.texto()
		if errCentimos != nil || errMoneda != nil {
			return nil, ports.ReferenciaHitoAnalisisRRHH{},
				errCanonConsultaRRHHPostgreSQL
		}
		coste = &ports.ImporteOperativoRRHH{
			Centimos: centimos,
			Moneda:   moneda,
		}
	}
	fuenteCoste, err := l.texto()
	if err != nil {
		return nil, ports.ReferenciaHitoAnalisisRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	return &ports.AnalisisOperativoRRHH{
		ModalidadClave: domain.ClaveCatalogo(modalidad),
		CategoriaRef:   categoria, CausaClave: domain.ClaveCatalogo(causa),
		PeriodoInicio: inicio, PeriodoFin: fin,
		PorcentajeJornada: domain.JornadaDiezmilesimas(jornada),
		ResultadoRC:       domain.ResultadoValidacionRC(resultadoRC),
		CostePrevisto:     coste, FuenteCosteRef: fuenteCoste,
	}, referencia, nil
}

func (l *lectorCanonRRHHPostgreSQL) cobertura() (
	*ports.CoberturaOperativaRRHH,
	ports.ReferenciaHitoCoberturaRRHH,
	error,
) {
	presente, err := l.booleano()
	if err != nil {
		return nil, ports.ReferenciaHitoCoberturaRRHH{}, err
	}
	secuencia, err := l.enteroSinSigno()
	if err != nil || !presente && secuencia != 0 {
		return nil, ports.ReferenciaHitoCoberturaRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	if !presente {
		return nil, ports.ReferenciaHitoCoberturaRRHH{}, nil
	}
	referencia, err := ports.NuevaReferenciaHitoCoberturaRRHH(secuencia)
	if err != nil {
		return nil, ports.ReferenciaHitoCoberturaRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	via, e1 := l.texto()
	gobernada, e2 := l.booleano()
	procedimiento, e3 := l.texto()
	bolsa, e4 := l.texto()
	total, e5 := l.enteroSinSigno()
	if algunErrorCanonRRHHPostgreSQL(e1, e2, e3, e4, e5) ||
		total > 32 || !l.cabenMarcos(total, 2) {
		return nil, ports.ReferenciaHitoCoberturaRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	comprobaciones := make([]ports.ComprobacionOperativaRRHH, int(total))
	for indice := range comprobaciones {
		clave, errClave := l.texto()
		resultado, errResultado := l.texto()
		if errClave != nil || errResultado != nil {
			return nil, ports.ReferenciaHitoCoberturaRRHH{},
				errCanonConsultaRRHHPostgreSQL
		}
		comprobaciones[indice] = ports.ComprobacionOperativaRRHH{
			Clave:     domain.ClaveCatalogo(clave),
			Resultado: domain.ResultadoComprobacion(resultado),
		}
	}
	return &ports.CoberturaOperativaRRHH{
		ViaClave: domain.ClaveCatalogo(via), DecisionGobernada: gobernada,
		ProcedimientoRef: procedimiento, BolsaRef: bolsa,
		Comprobaciones: comprobaciones,
	}, referencia, nil
}

func (l *lectorCanonRRHHPostgreSQL) asignacion() (
	*ports.AsignacionOperativaRRHH,
	ports.ReferenciaHitoAsignacionRRHH,
	error,
) {
	presente, err := l.booleano()
	if err != nil {
		return nil, ports.ReferenciaHitoAsignacionRRHH{}, err
	}
	secuencia, err := l.enteroSinSigno()
	if err != nil || !presente && secuencia != 0 {
		return nil, ports.ReferenciaHitoAsignacionRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	if !presente {
		return nil, ports.ReferenciaHitoAsignacionRRHH{}, nil
	}
	referencia, err := ports.NuevaReferenciaHitoAsignacionRRHH(secuencia)
	if err != nil {
		return nil, ports.ReferenciaHitoAsignacionRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	unidad, e1 := l.texto()
	asignadaEn, e2 := l.instante()
	motivo, e3 := l.texto()
	if algunErrorCanonRRHHPostgreSQL(e1, e2, e3) {
		return nil, ports.ReferenciaHitoAsignacionRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	return &ports.AsignacionOperativaRRHH{
		UnidadRef: unidad, AsignadaEn: asignadaEn,
		MotivoClave: domain.ClaveCatalogo(motivo),
	}, referencia, nil
}

func (l *lectorCanonRRHHPostgreSQL) hito() (
	ports.HitoExpedienteRRHH,
	error,
) {
	secuencia, e1 := l.enteroSinSigno()
	version, e2 := l.enteroSinSigno()
	accion, e3 := l.texto()
	realizadaEn, e4 := l.instante()
	faseOrigen, e5 := l.texto()
	faseDestino, e6 := l.texto()
	estadoOrigen, e7 := l.texto()
	estadoDestino, e8 := l.texto()
	if algunErrorCanonRRHHPostgreSQL(e1, e2, e3, e4, e5, e6, e7, e8) {
		return ports.HitoExpedienteRRHH{},
			errCanonConsultaRRHHPostgreSQL
	}
	return ports.HitoExpedienteRRHH{
		Secuencia: secuencia, VersionExpediente: version,
		AccionClave: domain.ClaveCatalogo(accion), RealizadaEn: realizadaEn,
		FaseOrigen:    domain.ClaveFase(faseOrigen),
		FaseDestino:   domain.ClaveFase(faseDestino),
		EstadoOrigen:  domain.EstadoOperativo(estadoOrigen),
		EstadoDestino: domain.EstadoOperativo(estadoDestino),
	}, nil
}

func (l *lectorCanonRRHHPostgreSQL) cabenMarcos(
	total uint64,
	camposPorElemento uint64,
) bool {
	if l == nil || l.posicion > len(l.contenido) ||
		camposPorElemento == 0 {
		return false
	}
	restantes := uint64(len(l.contenido) - l.posicion)
	return total <= restantes/(3*camposPorElemento)
}

func resumenesCuadroRRHHPostgreSQLValidos(
	resumenes []ports.ResumenExpedienteRRHH,
	generadaEn time.Time,
) bool {
	vistos := make(map[string]struct{}, len(resumenes))
	for indice, resumen := range resumenes {
		if resumen.Validar() != nil || resumen.ActualizadoEn.After(generadaEn) {
			return false
		}
		if _, repetido := vistos[resumen.ExpedienteRef]; repetido {
			return false
		}
		vistos[resumen.ExpedienteRef] = struct{}{}
		if indice == 0 {
			continue
		}
		anterior := resumenes[indice-1]
		if anterior.ActualizadoEn.Before(resumen.ActualizadoEn) ||
			anterior.ActualizadoEn.Equal(resumen.ActualizadoEn) &&
				anterior.ExpedienteRef <= resumen.ExpedienteRef {
			return false
		}
	}
	return true
}

func (l *lectorCanonRRHHPostgreSQL) consumido() bool {
	return l != nil && l.posicion == len(l.contenido)
}

func algunErrorCanonRRHHPostgreSQL(errores ...error) bool {
	for _, err := range errores {
		if err != nil {
			return true
		}
	}
	return false
}
