package domain

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
	"unicode/utf8"
)

var (
	// ErrParseoHistoricoAtestacionAutorizacionV1Invalido identifica una
	// representacion VEC-AD-1 que no es exacta, canonica y completa. El parseo
	// solo produce datos nominales: superar esta validacion no prueba una firma
	// ni concede autoridad alguna.
	ErrParseoHistoricoAtestacionAutorizacionV1Invalido = errors.New("vec: parseo historico VEC-AD-1 invalido")

	// ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
	// evita que una proyeccion con identificadores de persona y sesion termine
	// accidentalmente en registros o serializadores generales. VEC-AD-1 solo se
	// vuelve a emitir dentro del comprobador canonico privado de este archivo.
	ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida = errors.New("vec: serializacion de proyeccion historica VEC-AD-1 prohibida")
)

const representacionRedactadaProyeccionHistoricaAtestacionAutorizacionV1 = "[PROYECCION-HISTORICA-VEC-AD-1-NO-AUTORITATIVA]"

// DatosDecisionHistoricaAtestacionAutorizacionV1 enumera los treinta datos de
// DecisionAutorizacion distintos del vinculo y conserva el bloque de vinculo
// como DatosVinculoAutenticacionActorV1. Deliberadamente no contiene ni puede
// reconstruir VinculoAutenticacionActorV1, una DecisionAutorizacion o cualquier
// otra capacidad opaca.
//
// Sus campos son una proyeccion defensiva para comparaciones exactas tras haber
// verificado la firma por otra capa. No son una autorizacion y no deben usarse
// como entrada de un PDP, un constructor de capacidades o una mutacion durable.
type DatosDecisionHistoricaAtestacionAutorizacionV1 struct {
	DecisionRef                           string                           `json:"-"`
	Concedida                             bool                             `json:"-"`
	Codigo                                string                           `json:"-"`
	PrincipalID                           string                           `json:"-"`
	PerfilActivoRef                       string                           `json:"-"`
	Accion                                string                           `json:"-"`
	RecursoRef                            string                           `json:"-"`
	ModuloID                              string                           `json:"-"`
	TipoRecurso                           string                           `json:"-"`
	ContextoRecursoHuellaSHA256           string                           `json:"-"`
	Finalidad                             string                           `json:"-"`
	CorrelacionRef                        string                           `json:"-"`
	VinculoAutenticacionActor             DatosVinculoAutenticacionActorV1 `json:"-"`
	AsignacionRef                         string                           `json:"-"`
	AsignacionHuellaSHA256                string                           `json:"-"`
	VersionRolRef                         string                           `json:"-"`
	VersionRolHuellaSHA256                string                           `json:"-"`
	ControlVigenciaVersionRolRef          string                           `json:"-"`
	ControlVigenciaVersionRolRevision     uint64                           `json:"-"`
	ControlVigenciaVersionRolHuellaSHA256 string                           `json:"-"`
	RevisionCatalogoPoliticas             uint64                           `json:"-"`
	CatalogoPoliticasHuellaSHA256         string                           `json:"-"`
	PoliticasEvaluadasRefs                []string                         `json:"-"`
	PoliticasEvaluadasHuellasSHA256       map[string]string                `json:"-"`
	PoliticasRefs                         []string                         `json:"-"`
	PoliticasHuellasSHA256                map[string]string                `json:"-"`
	GarantiaMinima                        AuthAssurance                    `json:"-"`
	CamposPermitidos                      []string                         `json:"-"`
	Obligaciones                          []string                         `json:"-"`
	EmitidaEn                             time.Time                        `json:"-"`
	ValidaHasta                           time.Time                        `json:"-"`
}

// ProyeccionHistoricaAtestacionAutorizacionV1 es el resultado nominal y no
// autoritativo del parser estricto. Los campos internos impiden fabricar un
// valor valido mediante un literal desde otro paquete. Datos devuelve siempre
// copias de listas y mapas.
type ProyeccionHistoricaAtestacionAutorizacionV1 struct {
	cabecera CabeceraAtestacionAutorizacionV1
	datos    *DatosDecisionHistoricaAtestacionAutorizacionV1
}

// ParsearMensajeAtestacionAutorizacionV1NoAutoritativo interpreta el formato
// historico VEC-AD-1. Antes de reservar memoria limita el mensaje, cada texto y
// los conteos de listas y mapas; despues exige orden UTF-8 estricto, semantica
// de concesion reforzada y una reserializacion byte a byte identica.
//
// Esta funcion NO verifica COSE, una firma, la vigencia actual, revocaciones ni
// el consumo unico de una decision. Su resultado nunca debe tratarse como
// autoridad por el mero hecho de haber sido parseado.
func ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(
	contenido []byte,
) (ProyeccionHistoricaAtestacionAutorizacionV1, error) {
	if len(contenido) == 0 || len(contenido) > TamanoMaximoMensajeAtestacionAutorizacionV1 {
		return ProyeccionHistoricaAtestacionAutorizacionV1{}, errorParseoHistoricoAtestacionAutorizacionV1()
	}

	lector := lectorHistoricoAtestacionAutorizacionV1{contenido: contenido}
	lector.exigirBytes([]byte(EsquemaMensajeAtestacionAutorizacionV1))
	lector.exigirByte(0)
	cabecera := CabeceraAtestacionAutorizacionV1{
		FormatoVersion: lector.leerUint16(),
		Suite:          lector.leerTexto(128),
		ClaveID:        lector.leerTexto(512),
		Audiencia:      lector.leerTexto(512),
	}

	datos := DatosDecisionHistoricaAtestacionAutorizacionV1{
		DecisionRef:                           lector.leerTexto(512),
		Concedida:                             lector.leerBooleano(),
		Codigo:                                lector.leerTexto(128),
		PrincipalID:                           lector.leerTexto(512),
		PerfilActivoRef:                       lector.leerTexto(512),
		Accion:                                lector.leerTexto(256),
		RecursoRef:                            lector.leerTexto(512),
		ModuloID:                              lector.leerTexto(128),
		TipoRecurso:                           lector.leerTexto(128),
		ContextoRecursoHuellaSHA256:           lector.leerTexto(64),
		Finalidad:                             lector.leerTexto(512),
		CorrelacionRef:                        lector.leerTexto(512),
		VinculoAutenticacionActor:             leerDatosVinculoAutenticacionActorV1Historico(&lector),
		AsignacionRef:                         lector.leerTexto(512),
		AsignacionHuellaSHA256:                lector.leerTexto(64),
		VersionRolRef:                         lector.leerTexto(512),
		VersionRolHuellaSHA256:                lector.leerTexto(64),
		ControlVigenciaVersionRolRef:          lector.leerTexto(512),
		ControlVigenciaVersionRolRevision:     lector.leerUint64(),
		ControlVigenciaVersionRolHuellaSHA256: lector.leerTexto(64),
		RevisionCatalogoPoliticas:             lector.leerUint64(),
		CatalogoPoliticasHuellaSHA256:         lector.leerTexto(64),
		PoliticasEvaluadasRefs:                lector.leerListaUTF8Estricta(),
		PoliticasEvaluadasHuellasSHA256:       lector.leerMapaUTF8Estricto(),
		PoliticasRefs:                         lector.leerListaUTF8Estricta(),
		PoliticasHuellasSHA256:                lector.leerMapaUTF8Estricto(),
		GarantiaMinima:                        AuthAssurance(lector.leerTexto(32)),
		CamposPermitidos:                      lector.leerListaUTF8Estricta(),
		Obligaciones:                          lector.leerListaUTF8Estricta(),
		EmitidaEn:                             lector.leerInstante(),
		ValidaHasta:                           lector.leerInstante(),
	}
	longitudDeclarada := lector.leerUint64()
	if lector.err != nil || lector.posicion != len(contenido) || longitudDeclarada != uint64(len(contenido)) ||
		cabecera.Validar() != nil || validarDatosDecisionHistoricaAtestacionAutorizacionV1(datos) != nil {
		return ProyeccionHistoricaAtestacionAutorizacionV1{}, errorParseoHistoricoAtestacionAutorizacionV1()
	}

	canonico, err := serializarProyeccionHistoricaAtestacionAutorizacionV1(cabecera, datos)
	if err != nil || !bytes.Equal(canonico, contenido) {
		return ProyeccionHistoricaAtestacionAutorizacionV1{}, errorParseoHistoricoAtestacionAutorizacionV1()
	}

	copia := clonarDatosDecisionHistoricaAtestacionAutorizacionV1(datos)
	return ProyeccionHistoricaAtestacionAutorizacionV1{cabecera: cabecera, datos: &copia}, nil
}

// Cabecera devuelve la configuracion nominal incluida en VEC-AD-1. No indica
// que esa suite, clave o audiencia haya sido verificada o sea de confianza.
func (p ProyeccionHistoricaAtestacionAutorizacionV1) Cabecera() (CabeceraAtestacionAutorizacionV1, error) {
	if p.validarNoAutoritativa() != nil {
		return CabeceraAtestacionAutorizacionV1{}, errorParseoHistoricoAtestacionAutorizacionV1()
	}
	return p.cabecera, nil
}

// Datos devuelve una copia defensiva de todos los datos nominales de decision.
// En particular devuelve DatosVinculoAutenticacionActorV1 y nunca la capacidad
// opaca VinculoAutenticacionActorV1. El bloqueo de serializacion protege esta
// proyeccion completa; DatosVinculoAutenticacionActorV1, una vez extraido de
// ella, conserva deliberadamente la serializacion historica definida por su
// propio contrato y debe tratarse como dato personal sensible.
func (p ProyeccionHistoricaAtestacionAutorizacionV1) Datos() (DatosDecisionHistoricaAtestacionAutorizacionV1, error) {
	if p.validarNoAutoritativa() != nil {
		return DatosDecisionHistoricaAtestacionAutorizacionV1{}, errorParseoHistoricoAtestacionAutorizacionV1()
	}
	return clonarDatosDecisionHistoricaAtestacionAutorizacionV1(*p.datos), nil
}

func (p ProyeccionHistoricaAtestacionAutorizacionV1) validarNoAutoritativa() error {
	if p.datos == nil || p.cabecera.Validar() != nil ||
		validarDatosDecisionHistoricaAtestacionAutorizacionV1(*p.datos) != nil {
		return errorParseoHistoricoAtestacionAutorizacionV1()
	}
	return nil
}

func leerDatosVinculoAutenticacionActorV1Historico(
	lector *lectorHistoricoAtestacionAutorizacionV1,
) DatosVinculoAutenticacionActorV1 {
	return DatosVinculoAutenticacionActorV1{
		BloqueVersion:                lector.leerUint16(),
		AutenticacionRef:             lector.leerTexto(512),
		AutenticacionHuellaSHA256:    lector.leerTexto(64),
		AsercionRef:                  lector.leerTexto(512),
		SesionRef:                    lector.leerTexto(512),
		ControlSesionRef:             lector.leerTexto(512),
		ControlSesionRevision:        lector.leerUint64(),
		ControlSesionHuellaSHA256:    lector.leerTexto(64),
		CuentaRef:                    lector.leerTexto(512),
		CuentaOrdinariaRef:           lector.leerTexto(512),
		PrincipalID:                  lector.leerTexto(512),
		PerfilActivoRef:              lector.leerTexto(512),
		CuentaPrivilegiada:           lector.leerBooleano(),
		Superficie:                   SuperficieAutenticacionActorV1(lector.leerTexto(64)),
		MetodoObservado:              AuthMethod(lector.leerTexto(64)),
		GarantiaObservada:            AuthAssurance(lector.leerTexto(32)),
		PoliticaGarantiaRef:          lector.leerTexto(512),
		PoliticaGarantiaHuellaSHA256: lector.leerTexto(64),
		AutenticacionVerificadaEn:    lector.leerInstante(),
		SesionEmitidaEn:              lector.leerInstante(),
		SesionValidaHasta:            lector.leerInstante(),
		SesionRevalidadaEn:           lector.leerInstante(),
		ContextoActorRef:             lector.leerTexto(512),
		ContextoActorVersion:         lector.leerUint64(),
		ContextoActorHuellaSHA256:    lector.leerTexto(64),
	}
}

func validarDatosDecisionHistoricaAtestacionAutorizacionV1(
	d DatosDecisionHistoricaAtestacionAutorizacionV1,
) error {
	vinculo := d.VinculoAutenticacionActor
	if vinculo.Validar() != nil ||
		!textoAutorizacionSinComodinSeguro(d.DecisionRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.Codigo, 128, false) ||
		!textoAutorizacionSinComodinSeguro(d.PrincipalID, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.PerfilActivoRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.Accion, 256, false) ||
		!textoAutorizacionSinComodinSeguro(d.RecursoRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.ModuloID, 128, false) ||
		!textoAutorizacionSinComodinSeguro(d.TipoRecurso, 128, false) ||
		!huellaSHA256AutorizacionValida(d.ContextoRecursoHuellaSHA256) ||
		!textoAutorizacionSinComodinSeguro(d.Finalidad, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.CorrelacionRef, 512, false) ||
		d.PrincipalID != vinculo.PrincipalID || d.PerfilActivoRef != vinculo.PerfilActivoRef ||
		!d.Concedida || d.Codigo != "concedida" ||
		!textoAutorizacionSinComodinSeguro(d.AsignacionRef, 512, false) ||
		!huellaSHA256AutorizacionValida(d.AsignacionHuellaSHA256) ||
		!textoAutorizacionSinComodinSeguro(d.VersionRolRef, 512, false) ||
		!huellaSHA256AutorizacionValida(d.VersionRolHuellaSHA256) ||
		d.ControlVigenciaVersionRolRef != d.VersionRolRef ||
		d.ControlVigenciaVersionRolRevision == 0 ||
		!huellaSHA256AutorizacionValida(d.ControlVigenciaVersionRolHuellaSHA256) ||
		d.RevisionCatalogoPoliticas == 0 ||
		!huellaSHA256AutorizacionValida(d.CatalogoPoliticasHuellaSHA256) ||
		!d.GarantiaMinima.Valida() || !CumpleGarantiaAutenticacion(vinculo.GarantiaObservada, d.GarantiaMinima) ||
		!listaAutorizacionValida(d.PoliticasEvaluadasRefs, false, false) ||
		!listaAutorizacionValida(d.PoliticasRefs, false, false) ||
		!listaAutorizacionValida(d.CamposPermitidos, false, false) ||
		!listaAutorizacionValida(d.Obligaciones, false, false) ||
		!listaAtestacionAutorizacionCanonica(d.PoliticasEvaluadasRefs) ||
		!listaAtestacionAutorizacionCanonica(d.PoliticasRefs) ||
		!listaAtestacionAutorizacionCanonica(d.CamposPermitidos) ||
		!listaAtestacionAutorizacionCanonica(d.Obligaciones) ||
		!instanteAutorizacionCanonico(d.EmitidaEn) || !instanteAutorizacionCanonico(d.ValidaHasta) ||
		!d.ValidaHasta.After(d.EmitidaEn) || d.EmitidaEn.Before(vinculo.SesionRevalidadaEn) ||
		d.ValidaHasta.After(vinculo.SesionValidaHasta) ||
		d.ValidaHasta.Sub(d.EmitidaEn) > VigenciaMaximaDecisionAutorizacion {
		return errorParseoHistoricoAtestacionAutorizacionV1()
	}

	huellaCatalogo, err := huellaCatalogoPoliticasDesdeEvidencias(
		d.PoliticasEvaluadasRefs,
		d.PoliticasEvaluadasHuellasSHA256,
	)
	if err != nil || huellaCatalogo != d.CatalogoPoliticasHuellaSHA256 ||
		len(d.PoliticasHuellasSHA256) != len(d.PoliticasRefs) {
		return errorParseoHistoricoAtestacionAutorizacionV1()
	}
	for _, referencia := range d.PoliticasRefs {
		huellaAplicada, aplicada := d.PoliticasHuellasSHA256[referencia]
		huellaEvaluada, evaluada := d.PoliticasEvaluadasHuellasSHA256[referencia]
		if !aplicada || !evaluada || !huellaSHA256AutorizacionValida(huellaAplicada) ||
			huellaAplicada != huellaEvaluada {
			return errorParseoHistoricoAtestacionAutorizacionV1()
		}
	}
	return nil
}

func serializarProyeccionHistoricaAtestacionAutorizacionV1(
	cabecera CabeceraAtestacionAutorizacionV1,
	d DatosDecisionHistoricaAtestacionAutorizacionV1,
) ([]byte, error) {
	if cabecera.Validar() != nil || validarDatosDecisionHistoricaAtestacionAutorizacionV1(d) != nil {
		return nil, errorParseoHistoricoAtestacionAutorizacionV1()
	}

	escritor := nuevoEscritorAtestacionAutorizacionV1()
	escritor.escribirBytes([]byte(EsquemaMensajeAtestacionAutorizacionV1))
	escritor.escribirByte(0)
	escritor.escribirUint16(cabecera.FormatoVersion)
	escritor.escribirTexto(cabecera.Suite)
	escritor.escribirTexto(cabecera.ClaveID)
	escritor.escribirTexto(cabecera.Audiencia)
	escritor.escribirTexto(d.DecisionRef)
	escritor.escribirBooleano(d.Concedida)
	escritor.escribirTexto(d.Codigo)
	escritor.escribirTexto(d.PrincipalID)
	escritor.escribirTexto(d.PerfilActivoRef)
	escritor.escribirTexto(d.Accion)
	escritor.escribirTexto(d.RecursoRef)
	escritor.escribirTexto(d.ModuloID)
	escritor.escribirTexto(d.TipoRecurso)
	escritor.escribirTexto(d.ContextoRecursoHuellaSHA256)
	escritor.escribirTexto(d.Finalidad)
	escritor.escribirTexto(d.CorrelacionRef)
	escribirDatosVinculoAutenticacionActorV1Historico(escritor, d.VinculoAutenticacionActor)
	escritor.escribirTexto(d.AsignacionRef)
	escritor.escribirTexto(d.AsignacionHuellaSHA256)
	escritor.escribirTexto(d.VersionRolRef)
	escritor.escribirTexto(d.VersionRolHuellaSHA256)
	escritor.escribirTexto(d.ControlVigenciaVersionRolRef)
	escritor.escribirUint64(d.ControlVigenciaVersionRolRevision)
	escritor.escribirTexto(d.ControlVigenciaVersionRolHuellaSHA256)
	escritor.escribirUint64(d.RevisionCatalogoPoliticas)
	escritor.escribirTexto(d.CatalogoPoliticasHuellaSHA256)
	escritor.escribirLista(d.PoliticasEvaluadasRefs)
	escritor.escribirMapa(d.PoliticasEvaluadasHuellasSHA256)
	escritor.escribirLista(d.PoliticasRefs)
	escritor.escribirMapa(d.PoliticasHuellasSHA256)
	escritor.escribirTexto(string(d.GarantiaMinima))
	escritor.escribirLista(d.CamposPermitidos)
	escritor.escribirLista(d.Obligaciones)
	escritor.escribirInstante(d.EmitidaEn)
	escritor.escribirInstante(d.ValidaHasta)
	if escritor.err != nil || escritor.buffer.Len() > TamanoMaximoMensajeAtestacionAutorizacionV1-8 {
		return nil, errorParseoHistoricoAtestacionAutorizacionV1()
	}
	longitud := uint64(escritor.buffer.Len() + 8)
	escritor.escribirUint64(longitud)
	if escritor.err != nil || escritor.buffer.Len() != int(longitud) {
		return nil, errorParseoHistoricoAtestacionAutorizacionV1()
	}
	return append([]byte(nil), escritor.buffer.Bytes()...), nil
}

func escribirDatosVinculoAutenticacionActorV1Historico(
	escritor *escritorAtestacionAutorizacionV1,
	v DatosVinculoAutenticacionActorV1,
) {
	escritor.escribirUint16(v.BloqueVersion)
	escritor.escribirTexto(v.AutenticacionRef)
	escritor.escribirTexto(v.AutenticacionHuellaSHA256)
	escritor.escribirTexto(v.AsercionRef)
	escritor.escribirTexto(v.SesionRef)
	escritor.escribirTexto(v.ControlSesionRef)
	escritor.escribirUint64(v.ControlSesionRevision)
	escritor.escribirTexto(v.ControlSesionHuellaSHA256)
	escritor.escribirTexto(v.CuentaRef)
	escritor.escribirTexto(v.CuentaOrdinariaRef)
	escritor.escribirTexto(v.PrincipalID)
	escritor.escribirTexto(v.PerfilActivoRef)
	escritor.escribirBooleano(v.CuentaPrivilegiada)
	escritor.escribirTexto(string(v.Superficie))
	escritor.escribirTexto(string(v.MetodoObservado))
	escritor.escribirTexto(string(v.GarantiaObservada))
	escritor.escribirTexto(v.PoliticaGarantiaRef)
	escritor.escribirTexto(v.PoliticaGarantiaHuellaSHA256)
	escritor.escribirInstante(v.AutenticacionVerificadaEn)
	escritor.escribirInstante(v.SesionEmitidaEn)
	escritor.escribirInstante(v.SesionValidaHasta)
	escritor.escribirInstante(v.SesionRevalidadaEn)
	escritor.escribirTexto(v.ContextoActorRef)
	escritor.escribirUint64(v.ContextoActorVersion)
	escritor.escribirTexto(v.ContextoActorHuellaSHA256)
}

func clonarDatosDecisionHistoricaAtestacionAutorizacionV1(
	d DatosDecisionHistoricaAtestacionAutorizacionV1,
) DatosDecisionHistoricaAtestacionAutorizacionV1 {
	copia := d
	copia.PoliticasEvaluadasRefs = append([]string(nil), d.PoliticasEvaluadasRefs...)
	copia.PoliticasRefs = append([]string(nil), d.PoliticasRefs...)
	copia.CamposPermitidos = append([]string(nil), d.CamposPermitidos...)
	copia.Obligaciones = append([]string(nil), d.Obligaciones...)
	copia.PoliticasEvaluadasHuellasSHA256 = clonarMapaTextoHistoricoAtestacionAutorizacionV1(d.PoliticasEvaluadasHuellasSHA256)
	copia.PoliticasHuellasSHA256 = clonarMapaTextoHistoricoAtestacionAutorizacionV1(d.PoliticasHuellasSHA256)
	return copia
}

func clonarMapaTextoHistoricoAtestacionAutorizacionV1(origen map[string]string) map[string]string {
	copia := make(map[string]string, len(origen))
	for clave, valor := range origen {
		copia[clave] = valor
	}
	return copia
}

type lectorHistoricoAtestacionAutorizacionV1 struct {
	contenido []byte
	posicion  int
	err       error
}

func (l *lectorHistoricoAtestacionAutorizacionV1) tomar(cantidad int) []byte {
	if l.err != nil {
		return nil
	}
	if cantidad < 0 || l.posicion < 0 || l.posicion > len(l.contenido) ||
		cantidad > len(l.contenido)-l.posicion {
		l.err = errorParseoHistoricoAtestacionAutorizacionV1()
		return nil
	}
	inicio := l.posicion
	l.posicion += cantidad
	return l.contenido[inicio:l.posicion]
}

func (l *lectorHistoricoAtestacionAutorizacionV1) exigirBytes(esperado []byte) {
	if l.err != nil {
		return
	}
	if recibido := l.tomar(len(esperado)); l.err != nil || !bytes.Equal(recibido, esperado) {
		l.err = errorParseoHistoricoAtestacionAutorizacionV1()
	}
}

func (l *lectorHistoricoAtestacionAutorizacionV1) exigirByte(esperado byte) {
	if recibido := l.tomar(1); l.err != nil || len(recibido) != 1 || recibido[0] != esperado {
		l.err = errorParseoHistoricoAtestacionAutorizacionV1()
	}
}

func (l *lectorHistoricoAtestacionAutorizacionV1) leerUint16() uint16 {
	contenido := l.tomar(2)
	if l.err != nil {
		return 0
	}
	return binary.BigEndian.Uint16(contenido)
}

func (l *lectorHistoricoAtestacionAutorizacionV1) leerUint32() uint32 {
	contenido := l.tomar(4)
	if l.err != nil {
		return 0
	}
	return binary.BigEndian.Uint32(contenido)
}

func (l *lectorHistoricoAtestacionAutorizacionV1) leerUint64() uint64 {
	contenido := l.tomar(8)
	if l.err != nil {
		return 0
	}
	return binary.BigEndian.Uint64(contenido)
}

func (l *lectorHistoricoAtestacionAutorizacionV1) leerTexto(maximo int) string {
	longitud := l.leerUint32()
	if l.err != nil || maximo < 0 || uint64(longitud) > uint64(maximo) ||
		uint64(longitud) > uint64(len(l.contenido)-l.posicion) {
		l.err = errorParseoHistoricoAtestacionAutorizacionV1()
		return ""
	}
	contenido := l.tomar(int(longitud))
	if l.err != nil || !utf8.Valid(contenido) {
		l.err = errorParseoHistoricoAtestacionAutorizacionV1()
		return ""
	}
	return string(contenido)
}

func (l *lectorHistoricoAtestacionAutorizacionV1) leerBooleano() bool {
	contenido := l.tomar(1)
	if l.err != nil || len(contenido) != 1 || contenido[0] > 1 {
		l.err = errorParseoHistoricoAtestacionAutorizacionV1()
		return false
	}
	return contenido[0] == 1
}

func (l *lectorHistoricoAtestacionAutorizacionV1) leerListaUTF8Estricta() []string {
	cantidad := l.leerUint32()
	if l.err != nil || cantidad > maximoElementosAutorizacion {
		l.err = errorParseoHistoricoAtestacionAutorizacionV1()
		return nil
	}
	valores := make([]string, 0, int(cantidad))
	for indice := uint32(0); indice < cantidad; indice++ {
		valor := l.leerTexto(512)
		if l.err != nil || (len(valores) > 0 && bytes.Compare([]byte(valores[len(valores)-1]), []byte(valor)) >= 0) {
			l.err = errorParseoHistoricoAtestacionAutorizacionV1()
			return nil
		}
		valores = append(valores, valor)
	}
	return valores
}

func (l *lectorHistoricoAtestacionAutorizacionV1) leerMapaUTF8Estricto() map[string]string {
	cantidad := l.leerUint32()
	if l.err != nil || cantidad > maximoElementosAutorizacion {
		l.err = errorParseoHistoricoAtestacionAutorizacionV1()
		return nil
	}
	valores := make(map[string]string, int(cantidad))
	anterior := ""
	for indice := uint32(0); indice < cantidad; indice++ {
		clave := l.leerTexto(512)
		valor := l.leerTexto(64)
		if l.err != nil || (indice > 0 && bytes.Compare([]byte(anterior), []byte(clave)) >= 0) {
			l.err = errorParseoHistoricoAtestacionAutorizacionV1()
			return nil
		}
		if _, repetida := valores[clave]; repetida {
			l.err = errorParseoHistoricoAtestacionAutorizacionV1()
			return nil
		}
		valores[clave] = valor
		anterior = clave
	}
	return valores
}

func (l *lectorHistoricoAtestacionAutorizacionV1) leerInstante() time.Time {
	microsegundos := l.leerUint64()
	if l.err != nil {
		return time.Time{}
	}
	return time.UnixMicro(int64(microsegundos)).UTC()
}

func errorParseoHistoricoAtestacionAutorizacionV1() error {
	return errors.Join(
		ErrParseoHistoricoAtestacionAutorizacionV1Invalido,
		ErrMensajeAtestacionAutorizacionInvalido,
	)
}

func (DatosDecisionHistoricaAtestacionAutorizacionV1) String() string {
	return representacionRedactadaProyeccionHistoricaAtestacionAutorizacionV1
}

func (d DatosDecisionHistoricaAtestacionAutorizacionV1) GoString() string { return d.String() }

func (d DatosDecisionHistoricaAtestacionAutorizacionV1) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}

func (d DatosDecisionHistoricaAtestacionAutorizacionV1) LogValue() slog.Value {
	return slog.StringValue(d.String())
}

func (DatosDecisionHistoricaAtestacionAutorizacionV1) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (*DatosDecisionHistoricaAtestacionAutorizacionV1) UnmarshalJSON([]byte) error {
	return ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (DatosDecisionHistoricaAtestacionAutorizacionV1) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (*DatosDecisionHistoricaAtestacionAutorizacionV1) UnmarshalText([]byte) error {
	return ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (DatosDecisionHistoricaAtestacionAutorizacionV1) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (*DatosDecisionHistoricaAtestacionAutorizacionV1) UnmarshalBinary([]byte) error {
	return ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (DatosDecisionHistoricaAtestacionAutorizacionV1) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (*DatosDecisionHistoricaAtestacionAutorizacionV1) GobDecode([]byte) error {
	return ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (DatosDecisionHistoricaAtestacionAutorizacionV1) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (DatosDecisionHistoricaAtestacionAutorizacionV1) MarshalYAML() (any, error) {
	return nil, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (DatosDecisionHistoricaAtestacionAutorizacionV1) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (ProyeccionHistoricaAtestacionAutorizacionV1) String() string {
	return representacionRedactadaProyeccionHistoricaAtestacionAutorizacionV1
}

func (p ProyeccionHistoricaAtestacionAutorizacionV1) GoString() string { return p.String() }

func (p ProyeccionHistoricaAtestacionAutorizacionV1) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}

func (p ProyeccionHistoricaAtestacionAutorizacionV1) LogValue() slog.Value {
	return slog.StringValue(p.String())
}

func (ProyeccionHistoricaAtestacionAutorizacionV1) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (*ProyeccionHistoricaAtestacionAutorizacionV1) UnmarshalJSON([]byte) error {
	return ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (ProyeccionHistoricaAtestacionAutorizacionV1) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (*ProyeccionHistoricaAtestacionAutorizacionV1) UnmarshalText([]byte) error {
	return ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (ProyeccionHistoricaAtestacionAutorizacionV1) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (*ProyeccionHistoricaAtestacionAutorizacionV1) UnmarshalBinary([]byte) error {
	return ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (ProyeccionHistoricaAtestacionAutorizacionV1) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (*ProyeccionHistoricaAtestacionAutorizacionV1) GobDecode([]byte) error {
	return ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (ProyeccionHistoricaAtestacionAutorizacionV1) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (ProyeccionHistoricaAtestacionAutorizacionV1) MarshalYAML() (any, error) {
	return nil, ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}

func (ProyeccionHistoricaAtestacionAutorizacionV1) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
}
