package domain

import (
	"time"
)

// Incidencias técnicas de la capa de supervisión (M1 del consenso de
// supervisión del 25/09/2026).
//
// Este fichero es la única autoridad del catálogo cerrado y versionado de
// incidencias técnicas. Una incidencia describe un fallo técnico de la
// plataforma mediante una lista blanca de campos cerrados; nunca transporta
// texto libre, errores de bibliotecas, SQL, direcciones IP, URI, rutas,
// nombres, referencias de persona o expediente, cabeceras, cuerpos ni pilas.
// Por construcción es apta para entregarse íntegra a Sistemas o a una
// herramienta externa sin datos personales, tampoco indirectos.
//
// Cualquier valor que no pertenezca al catálogo se sanea a un valor fijo: el
// código desconocido, o un componente/etapa no admitidos para el código, se
// transforman en RECOLECCION_DEGRADADA/supervision/validacion. La severidad y
// el mensaje nunca los decide el llamante: proceden del catálogo.

// EsquemaIncidenciaTecnica identifica la versión del formato serializado.
const EsquemaIncidenciaTecnica = "vec.incidencia_tecnica.v1"

// VersionCatalogoIncidenciasTecnicas se incrementa con cualquier cambio de
// códigos, severidades, componentes, etapas o plantillas.
const VersionCatalogoIncidenciasTecnicas = 1

// RecuentoMaximoIncidenciaTecnica acota el recuento declarado de una incidencia.
const RecuentoMaximoIncidenciaTecnica uint32 = 1 << 20

// CodigoIncidenciaTecnica es un código estable del catálogo.
type CodigoIncidenciaTecnica string

// Códigos del catálogo v1.
const (
	IncidenciaArranqueFallido           CodigoIncidenciaTecnica = "ARRANQUE_FALLIDO"
	IncidenciaCatalogoModulosInvalido   CodigoIncidenciaTecnica = "CATALOGO_MODULOS_INVALIDO"
	IncidenciaHTTPInternoFallido        CodigoIncidenciaTecnica = "HTTP_INTERNO_FALLIDO"
	IncidenciaPanicoControlado          CodigoIncidenciaTecnica = "PANICO_CONTROLADO"
	IncidenciaGobiernoV3NoDisponible    CodigoIncidenciaTecnica = "GOBIERNO_V3_NO_DISPONIBLE"
	IncidenciaPostgresNoDisponible      CodigoIncidenciaTecnica = "POSTGRES_NO_DISPONIBLE"
	IncidenciaSMTPNoDisponible          CodigoIncidenciaTecnica = "SMTP_NO_DISPONIBLE"
	IncidenciaOSRMNoDisponible          CodigoIncidenciaTecnica = "OSRM_NO_DISPONIBLE"
	IncidenciaAuditoriaNoRegistrada     CodigoIncidenciaTecnica = "AUDITORIA_NO_REGISTRADA"
	IncidenciaModuloWebNoCargado        CodigoIncidenciaTecnica = "MODULO_WEB_NO_CARGADO"
	IncidenciaClienteFalloNoClasificado CodigoIncidenciaTecnica = "CLIENTE_FALLO_NO_CLASIFICADO"
	IncidenciaRecoleccionDegradada      CodigoIncidenciaTecnica = "RECOLECCION_DEGRADADA"
	IncidenciaAlertaNoEntregada         CodigoIncidenciaTecnica = "ALERTA_NO_ENTREGADA"
)

// SeveridadIncidenciaTecnica es asignada exclusivamente por el catálogo.
type SeveridadIncidenciaTecnica string

const (
	SeveridadIncidenciaAviso   SeveridadIncidenciaTecnica = "aviso"
	SeveridadIncidenciaError   SeveridadIncidenciaTecnica = "error"
	SeveridadIncidenciaCritica SeveridadIncidenciaTecnica = "critica"
)

// ComponenteIncidenciaTecnica identifica la pieza técnica afectada.
type ComponenteIncidenciaTecnica string

const (
	ComponenteIncidenciaServidor        ComponenteIncidenciaTecnica = "servidor"
	ComponenteIncidenciaComposicion     ComponenteIncidenciaTecnica = "composicion"
	ComponenteIncidenciaHTTP            ComponenteIncidenciaTecnica = "http"
	ComponenteIncidenciaCatalogoModulos ComponenteIncidenciaTecnica = "catalogo_modulos"
	ComponenteIncidenciaPortalWeb       ComponenteIncidenciaTecnica = "portal_web"
	ComponenteIncidenciaGobiernoV3      ComponenteIncidenciaTecnica = "gobierno_v3"
	ComponenteIncidenciaPostgreSQL      ComponenteIncidenciaTecnica = "postgresql"
	ComponenteIncidenciaSMTP            ComponenteIncidenciaTecnica = "smtp"
	ComponenteIncidenciaOSRM            ComponenteIncidenciaTecnica = "osrm"
	ComponenteIncidenciaAuditoria       ComponenteIncidenciaTecnica = "auditoria"
	ComponenteIncidenciaSupervision     ComponenteIncidenciaTecnica = "supervision"
	ComponenteIncidenciaAlertas         ComponenteIncidenciaTecnica = "alertas"
)

// EtapaIncidenciaTecnica identifica la operación/etapa cerrada en la que se
// produjo el fallo.
type EtapaIncidenciaTecnica string

const (
	EtapaIncidenciaConfiguracion EtapaIncidenciaTecnica = "configuracion"
	EtapaIncidenciaComposicion   EtapaIncidenciaTecnica = "composicion"
	EtapaIncidenciaEscucha       EtapaIncidenciaTecnica = "escucha"
	EtapaIncidenciaValidacion    EtapaIncidenciaTecnica = "validacion"
	EtapaIncidenciaCarga         EtapaIncidenciaTecnica = "carga"
	EtapaIncidenciaPeticion      EtapaIncidenciaTecnica = "peticion"
	EtapaIncidenciaEjecucion     EtapaIncidenciaTecnica = "ejecucion"
	EtapaIncidenciaRenovacion    EtapaIncidenciaTecnica = "renovacion"
	EtapaIncidenciaConexion      EtapaIncidenciaTecnica = "conexion"
	EtapaIncidenciaConsulta      EtapaIncidenciaTecnica = "consulta"
	EtapaIncidenciaEnvio         EtapaIncidenciaTecnica = "envio"
	EtapaIncidenciaRegistro      EtapaIncidenciaTecnica = "registro"
	EtapaIncidenciaEmision       EtapaIncidenciaTecnica = "emision"
	EtapaIncidenciaEscritura     EtapaIncidenciaTecnica = "escritura"
	EtapaIncidenciaEntrega       EtapaIncidenciaTecnica = "entrega"
)

// EntornoIncidenciaTecnica es la lista cerrada de entornos admitidos.
type EntornoIncidenciaTecnica string

const (
	EntornoIncidenciaDesarrollo   EntornoIncidenciaTecnica = "desarrollo"
	EntornoIncidenciaPruebas      EntornoIncidenciaTecnica = "pruebas"
	EntornoIncidenciaPresentacion EntornoIncidenciaTecnica = "presentacion"
	EntornoIncidenciaProduccion   EntornoIncidenciaTecnica = "produccion"
	EntornoIncidenciaDesconocido  EntornoIncidenciaTecnica = "desconocido"
)

// VersionBinarioDesconocida sustituye a cualquier versión no conforme.
const VersionBinarioDesconocida = "desconocida"

// DefinicionIncidenciaTecnica es la entrada inmutable del catálogo para un código.
type DefinicionIncidenciaTecnica struct {
	Codigo      CodigoIncidenciaTecnica
	Severidad   SeveridadIncidenciaTecnica
	Componentes []ComponenteIncidenciaTecnica
	Etapas      []EtapaIncidenciaTecnica
	// Plantilla es un texto fijo sin marcadores de sustitución.
	Plantilla string
}

// DefinicionIncidenciaTecnicaDe devuelve una copia de la definición del
// código o false si el código no pertenece al catálogo.
func DefinicionIncidenciaTecnicaDe(codigo CodigoIncidenciaTecnica) (DefinicionIncidenciaTecnica, bool) {
	switch codigo {
	case IncidenciaArranqueFallido:
		return definicionIncidencia(codigo, SeveridadIncidenciaCritica,
			"El servidor no ha podido arrancar.",
			[]ComponenteIncidenciaTecnica{ComponenteIncidenciaServidor, ComponenteIncidenciaComposicion},
			[]EtapaIncidenciaTecnica{EtapaIncidenciaConfiguracion, EtapaIncidenciaComposicion, EtapaIncidenciaEscucha}), true
	case IncidenciaCatalogoModulosInvalido:
		return definicionIncidencia(codigo, SeveridadIncidenciaError,
			"El catálogo de módulos no es válido.",
			[]ComponenteIncidenciaTecnica{ComponenteIncidenciaCatalogoModulos, ComponenteIncidenciaPortalWeb},
			[]EtapaIncidenciaTecnica{EtapaIncidenciaValidacion, EtapaIncidenciaCarga}), true
	case IncidenciaHTTPInternoFallido:
		return definicionIncidencia(codigo, SeveridadIncidenciaError,
			"Una petición ha terminado con error interno.",
			[]ComponenteIncidenciaTecnica{ComponenteIncidenciaHTTP},
			[]EtapaIncidenciaTecnica{EtapaIncidenciaPeticion}), true
	case IncidenciaPanicoControlado:
		return definicionIncidencia(codigo, SeveridadIncidenciaCritica,
			"Se ha contenido un pánico del proceso.",
			[]ComponenteIncidenciaTecnica{ComponenteIncidenciaHTTP, ComponenteIncidenciaServidor},
			[]EtapaIncidenciaTecnica{EtapaIncidenciaPeticion, EtapaIncidenciaEjecucion}), true
	case IncidenciaGobiernoV3NoDisponible:
		return definicionIncidencia(codigo, SeveridadIncidenciaCritica,
			"La configuración de gobierno V3 no está disponible.",
			[]ComponenteIncidenciaTecnica{ComponenteIncidenciaGobiernoV3},
			[]EtapaIncidenciaTecnica{EtapaIncidenciaRenovacion, EtapaIncidenciaConsulta}), true
	case IncidenciaPostgresNoDisponible:
		return definicionIncidencia(codigo, SeveridadIncidenciaCritica,
			"PostgreSQL no está disponible.",
			[]ComponenteIncidenciaTecnica{ComponenteIncidenciaPostgreSQL},
			[]EtapaIncidenciaTecnica{EtapaIncidenciaConexion, EtapaIncidenciaConsulta}), true
	case IncidenciaSMTPNoDisponible:
		return definicionIncidencia(codigo, SeveridadIncidenciaError,
			"El servicio de correo saliente no está disponible.",
			[]ComponenteIncidenciaTecnica{ComponenteIncidenciaSMTP},
			[]EtapaIncidenciaTecnica{EtapaIncidenciaConexion, EtapaIncidenciaEnvio}), true
	case IncidenciaOSRMNoDisponible:
		return definicionIncidencia(codigo, SeveridadIncidenciaAviso,
			"El servicio de rutas OSRM no está disponible.",
			[]ComponenteIncidenciaTecnica{ComponenteIncidenciaOSRM},
			[]EtapaIncidenciaTecnica{EtapaIncidenciaConexion, EtapaIncidenciaConsulta}), true
	case IncidenciaAuditoriaNoRegistrada:
		return definicionIncidencia(codigo, SeveridadIncidenciaCritica,
			"No se ha podido registrar una entrada de auditoría.",
			[]ComponenteIncidenciaTecnica{ComponenteIncidenciaAuditoria},
			[]EtapaIncidenciaTecnica{EtapaIncidenciaRegistro}), true
	case IncidenciaModuloWebNoCargado:
		return definicionIncidencia(codigo, SeveridadIncidenciaError,
			"Un módulo web no se ha cargado.",
			[]ComponenteIncidenciaTecnica{ComponenteIncidenciaPortalWeb},
			[]EtapaIncidenciaTecnica{EtapaIncidenciaCarga}), true
	case IncidenciaClienteFalloNoClasificado:
		return definicionIncidencia(codigo, SeveridadIncidenciaAviso,
			"El cliente ha declarado un fallo no clasificado.",
			[]ComponenteIncidenciaTecnica{ComponenteIncidenciaPortalWeb},
			[]EtapaIncidenciaTecnica{EtapaIncidenciaEjecucion}), true
	case IncidenciaRecoleccionDegradada:
		return definicionIncidencia(codigo, SeveridadIncidenciaAviso,
			"La recogida de incidencias técnicas está degradada.",
			[]ComponenteIncidenciaTecnica{ComponenteIncidenciaSupervision},
			[]EtapaIncidenciaTecnica{EtapaIncidenciaValidacion, EtapaIncidenciaEmision, EtapaIncidenciaEscritura}), true
	case IncidenciaAlertaNoEntregada:
		return definicionIncidencia(codigo, SeveridadIncidenciaError,
			"Un aviso técnico no se ha entregado.",
			[]ComponenteIncidenciaTecnica{ComponenteIncidenciaAlertas},
			[]EtapaIncidenciaTecnica{EtapaIncidenciaEntrega}), true
	default:
		return DefinicionIncidenciaTecnica{}, false
	}
}

func definicionIncidencia(codigo CodigoIncidenciaTecnica, severidad SeveridadIncidenciaTecnica, plantilla string, componentes []ComponenteIncidenciaTecnica, etapas []EtapaIncidenciaTecnica) DefinicionIncidenciaTecnica {
	return DefinicionIncidenciaTecnica{Codigo: codigo, Severidad: severidad, Componentes: componentes, Etapas: etapas, Plantilla: plantilla}
}

// CodigosIncidenciaTecnica devuelve una copia nueva del catálogo de códigos.
func CodigosIncidenciaTecnica() []CodigoIncidenciaTecnica {
	return []CodigoIncidenciaTecnica{
		IncidenciaArranqueFallido,
		IncidenciaCatalogoModulosInvalido,
		IncidenciaHTTPInternoFallido,
		IncidenciaPanicoControlado,
		IncidenciaGobiernoV3NoDisponible,
		IncidenciaPostgresNoDisponible,
		IncidenciaSMTPNoDisponible,
		IncidenciaOSRMNoDisponible,
		IncidenciaAuditoriaNoRegistrada,
		IncidenciaModuloWebNoCargado,
		IncidenciaClienteFalloNoClasificado,
		IncidenciaRecoleccionDegradada,
		IncidenciaAlertaNoEntregada,
	}
}

// SolicitudIncidenciaTecnica es lo único que un llamante puede aportar.
// Severidad, instante, entorno, versión, correlación y mensaje los fija la
// capa de emisión y el catálogo, nunca el llamante.
type SolicitudIncidenciaTecnica struct {
	Codigo     CodigoIncidenciaTecnica
	Componente ComponenteIncidenciaTecnica
	Etapa      EtapaIncidenciaTecnica
	// Recuento de ocurrencias agregadas; 0 equivale a 1.
	Recuento uint32
}

// ClasificacionIncidenciaTecnica es una solicitud ya saneada contra el
// catálogo. Sus valores son siempre constantes del catálogo, de modo que no
// retiene memoria aportada por el llamante.
type ClasificacionIncidenciaTecnica struct {
	Codigo     CodigoIncidenciaTecnica
	Severidad  SeveridadIncidenciaTecnica
	Componente ComponenteIncidenciaTecnica
	Etapa      EtapaIncidenciaTecnica
	Recuento   uint32
	Plantilla  string
}

// ClasificarIncidenciaTecnica sanea una solicitud. Devuelve saneada=true si
// hubo que sustituir el código, el componente o la etapa por el valor fijo
// RECOLECCION_DEGRADADA/supervision/validacion. El recuento se acota a
// [1, RecuentoMaximoIncidenciaTecnica] sin considerarse saneamiento.
func ClasificarIncidenciaTecnica(s SolicitudIncidenciaTecnica) (ClasificacionIncidenciaTecnica, bool) {
	recuento := s.Recuento
	if recuento == 0 {
		recuento = 1
	}
	if recuento > RecuentoMaximoIncidenciaTecnica {
		recuento = RecuentoMaximoIncidenciaTecnica
	}
	def, ok := DefinicionIncidenciaTecnicaDe(s.Codigo)
	if ok {
		componente, okC := canonicoEn(def.Componentes, s.Componente)
		etapa, okE := canonicoEn(def.Etapas, s.Etapa)
		if okC && okE {
			return ClasificacionIncidenciaTecnica{
				Codigo: def.Codigo, Severidad: def.Severidad, Componente: componente,
				Etapa: etapa, Recuento: recuento, Plantilla: def.Plantilla,
			}, false
		}
	}
	return clasificacionNoConforme(recuento), true
}

// ClasificacionRecoleccionDegradada construye la incidencia que la propia
// capa de emisión usa para declarar pérdidas o entradas no conformes.
func ClasificacionRecoleccionDegradada(etapa EtapaIncidenciaTecnica, recuento uint32) ClasificacionIncidenciaTecnica {
	c, _ := ClasificarIncidenciaTecnica(SolicitudIncidenciaTecnica{
		Codigo: IncidenciaRecoleccionDegradada, Componente: ComponenteIncidenciaSupervision,
		Etapa: etapa, Recuento: recuento,
	})
	return c
}

func clasificacionNoConforme(recuento uint32) ClasificacionIncidenciaTecnica {
	def, _ := DefinicionIncidenciaTecnicaDe(IncidenciaRecoleccionDegradada)
	return ClasificacionIncidenciaTecnica{
		Codigo: def.Codigo, Severidad: def.Severidad, Componente: ComponenteIncidenciaSupervision,
		Etapa: EtapaIncidenciaValidacion, Recuento: recuento, Plantilla: def.Plantilla,
	}
}

// canonicoEn devuelve el valor del catálogo igual al pedido, no el del
// llamante, para no retener su memoria.
func canonicoEn[T ~string](permitidos []T, valor T) (T, bool) {
	for _, p := range permitidos {
		if p == valor {
			return p, true
		}
	}
	var cero T
	return cero, false
}

// NormalizarEntornoIncidenciaTecnica reduce cualquier texto a la lista cerrada.
func NormalizarEntornoIncidenciaTecnica(valor string) EntornoIncidenciaTecnica {
	switch EntornoIncidenciaTecnica(valor) {
	case EntornoIncidenciaDesarrollo:
		return EntornoIncidenciaDesarrollo
	case EntornoIncidenciaPruebas:
		return EntornoIncidenciaPruebas
	case EntornoIncidenciaPresentacion:
		return EntornoIncidenciaPresentacion
	case EntornoIncidenciaProduccion:
		return EntornoIncidenciaProduccion
	default:
		return EntornoIncidenciaDesconocido
	}
}

// NormalizarVersionBinario admite solo una revisión hexadecimal en minúsculas
// de 7 a 40 caracteres o una versión semántica simple vMAYOR.MENOR.PARCHE; lo
// demás se sustituye por VersionBinarioDesconocida.
func NormalizarVersionBinario(valor string) string {
	if esRevisionHex(valor) || esVersionSemantica(valor) {
		return valor
	}
	return VersionBinarioDesconocida
}

func esRevisionHex(v string) bool {
	return len(v) >= 7 && len(v) <= 40 && esHexMinusculas(v)
}

func esVersionSemantica(v string) bool {
	if len(v) < 6 || len(v) > 16 || v[0] != 'v' {
		return false
	}
	partes, digitos := 1, 0
	for i := 1; i < len(v); i++ {
		switch c := v[i]; {
		case c >= '0' && c <= '9':
			digitos++
			if digitos > 4 {
				return false
			}
		case c == '.' && digitos > 0:
			partes++
			digitos = 0
		default:
			return false
		}
	}
	return partes == 3 && digitos > 0
}

// EsCorrelacionTecnicaValida exige 32 caracteres hexadecimales en minúsculas
// (128 bits aleatorios generados por la capa de emisión).
func EsCorrelacionTecnicaValida(v string) bool {
	return len(v) == 32 && esHexMinusculas(v)
}

func esHexMinusculas(v string) bool {
	for i := 0; i < len(v); i++ {
		c := v[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// IncidenciaTecnica es la proyección completa y cerrada de una incidencia. Es
// la única forma que puede persistirse o transmitirse al canal técnico.
type IncidenciaTecnica struct {
	Esquema        string
	Instante       time.Time
	Codigo         CodigoIncidenciaTecnica
	Severidad      SeveridadIncidenciaTecnica
	Componente     ComponenteIncidenciaTecnica
	Etapa          EtapaIncidenciaTecnica
	Entorno        EntornoIncidenciaTecnica
	VersionBinario string
	Correlacion    string
	Recuento       uint32
	Mensaje        string
}

// NuevaIncidenciaTecnica compone la incidencia final a partir de una
// clasificación ya saneada. Vuelve a sanear entorno, versión y correlación;
// una correlación no conforme se sustituye por ceros.
func NuevaIncidenciaTecnica(c ClasificacionIncidenciaTecnica, instante time.Time, entorno EntornoIncidenciaTecnica, version, correlacion string) IncidenciaTecnica {
	// Revalidar la clasificación impide que un valor construido a mano
	// fuera de ClasificarIncidenciaTecnica cuele campos no catalogados.
	c, _ = ClasificarIncidenciaTecnica(SolicitudIncidenciaTecnica{Codigo: c.Codigo, Componente: c.Componente, Etapa: c.Etapa, Recuento: c.Recuento})
	if !EsCorrelacionTecnicaValida(correlacion) {
		correlacion = "00000000000000000000000000000000"
	}
	return IncidenciaTecnica{
		Esquema:        EsquemaIncidenciaTecnica,
		Instante:       instante.UTC().Truncate(time.Millisecond),
		Codigo:         c.Codigo,
		Severidad:      c.Severidad,
		Componente:     c.Componente,
		Etapa:          c.Etapa,
		Entorno:        NormalizarEntornoIncidenciaTecnica(string(entorno)),
		VersionBinario: NormalizarVersionBinario(version),
		Correlacion:    correlacion,
		Recuento:       c.Recuento,
		Mensaje:        c.Plantilla,
	}
}
