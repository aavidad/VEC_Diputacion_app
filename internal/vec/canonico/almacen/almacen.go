// Package almacen concentra las reglas puras y deterministas del contrato de
// almacenamiento. No conoce puertos, adaptadores ni errores de transporte.
package almacen

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	DuracionMaximaInstruccionesCargaDirecta = 10 * time.Minute
	LongitudMaximaDestinoCargaDirecta       = 8192
	MaximoCabecerasCargaDirecta             = 32
	MaximoOrigenesCargaDirecta              = 32
)

// Las acciones son una lista positiva cerrada de operaciones tecnicas.
const (
	AccionEscribir               = "escribir"
	AccionLeer                   = "leer"
	AccionPrepararCargaDirecta   = "preparar_carga_directa"
	AccionConfirmarCargaDirecta  = "confirmar_carga_directa"
	AccionAbandonarCargaDirecta  = "abandonar_carga_directa"
	AccionPromover               = "promover"
	AccionAplicarRetencion       = "aplicar_retencion"
	AccionInmovilizar            = "inmovilizar"
	AccionLevantarInmovilizacion = "levantar_inmovilizacion"
	AccionEliminar               = "eliminar"
	AccionAnalizarContenido      = "analizar_contenido"
)

type Zona string

const (
	ZonaCuarentena Zona = "cuarentena"
	ZonaAdmitida   Zona = "admitida"
)

func (z Zona) Valida() bool {
	return z == ZonaCuarentena || z == ZonaAdmitida
}

type MetodoCargaDirecta string

const (
	MetodoCargaDirectaPUT  MetodoCargaDirecta = "PUT"
	MetodoCargaDirectaPOST MetodoCargaDirecta = "POST"
)

func (m MetodoCargaDirecta) Valido() bool {
	return m == MetodoCargaDirectaPUT || m == MetodoCargaDirectaPOST
}

type CabeceraCargaDirecta struct {
	Nombre string
	Valor  string
}

type Capacidades struct {
	ConectorID                  string
	EscrituraEnFlujo            bool
	LecturaEnFlujo              bool
	ReferenciasOpacas           bool
	IntegridadSHA256            bool
	Versionado                  bool
	Retencion                   bool
	BloqueoLegal                bool
	PromocionAtomica            bool
	RetencionAtomicaEnPromocion bool
	CargaDirectaTemporal        bool
	CifradoEnTransito           bool
	CifradoEnReposo             bool
	CifradoPorObjeto            bool
	TamanoMaximoObjeto          int64
	PreservaObjetoOriginal      bool
	OrigenesCargaDirecta        []string
}

type Requisitos struct {
	EscrituraEnFlujo            bool
	LecturaEnFlujo              bool
	ReferenciasOpacas           bool
	IntegridadSHA256            bool
	Versionado                  bool
	Retencion                   bool
	BloqueoLegal                bool
	PromocionAtomica            bool
	RetencionAtomicaEnPromocion bool
	CargaDirectaTemporal        bool
	CifradoEnTransito           bool
	CifradoEnReposo             bool
	CifradoPorObjeto            bool
	TamanoMinimoObjeto          int64
	PreservaObjetoOriginal      bool
}

// CapacidadesSatisfacen aplica la lista cerrada de requisitos de despliegue.
func CapacidadesSatisfacen(capacidades Capacidades, requisitos Requisitos) bool {
	return ReferenciaOpacaValida(capacidades.ConectorID, 128) &&
		capacidades.TamanoMaximoObjeto >= 1 &&
		(!requisitos.EscrituraEnFlujo || capacidades.EscrituraEnFlujo) &&
		(!requisitos.LecturaEnFlujo || capacidades.LecturaEnFlujo) &&
		(!requisitos.ReferenciasOpacas || capacidades.ReferenciasOpacas) &&
		(!requisitos.IntegridadSHA256 || capacidades.IntegridadSHA256) &&
		(!requisitos.Versionado || capacidades.Versionado) &&
		(!requisitos.Retencion || capacidades.Retencion) &&
		(!requisitos.BloqueoLegal || capacidades.BloqueoLegal) &&
		(!requisitos.PromocionAtomica || capacidades.PromocionAtomica) &&
		(!requisitos.RetencionAtomicaEnPromocion || capacidades.RetencionAtomicaEnPromocion) &&
		(!requisitos.CargaDirectaTemporal || capacidades.CargaDirectaTemporal) &&
		(!requisitos.CifradoEnTransito || capacidades.CifradoEnTransito) &&
		(!requisitos.CifradoEnReposo || capacidades.CifradoEnReposo) &&
		(!requisitos.CifradoPorObjeto || capacidades.CifradoPorObjeto) &&
		(!requisitos.PreservaObjetoOriginal || capacidades.PreservaObjetoOriginal) &&
		requisitos.TamanoMinimoObjeto >= 0 &&
		capacidades.TamanoMaximoObjeto >= requisitos.TamanoMinimoObjeto &&
		(!requisitos.CargaDirectaTemporal || OrigenesCargaDirectaValidos(capacidades.OrigenesCargaDirecta))
}

type DatosInstruccionesCargaDirecta struct {
	ConectorID   string
	SesionRef    string
	Metodo       MetodoCargaDirecta
	Destino      string
	Cabeceras    []CabeceraCargaDirecta
	EmitidaEn    time.Time
	ExpiraEn     time.Time
	TamanoMaximo int64
}

// InstruccionesCargaDirectaValidas valida la concesion completa sin revelar
// ni transformar ninguno de sus valores.
func InstruccionesCargaDirectaValidas(datos DatosInstruccionesCargaDirecta) bool {
	return ReferenciaOpacaValida(datos.ConectorID, 128) &&
		ReferenciaOpacaValida(datos.SesionRef, 512) && datos.Metodo.Valido() &&
		DestinoCargaDirectaValido(datos.Destino) && !datos.EmitidaEn.IsZero() && !datos.ExpiraEn.IsZero() &&
		datos.EmitidaEn.Location() == time.UTC && datos.ExpiraEn.Location() == time.UTC &&
		datos.ExpiraEn.After(datos.EmitidaEn) &&
		datos.ExpiraEn.Sub(datos.EmitidaEn) <= DuracionMaximaInstruccionesCargaDirecta &&
		datos.TamanoMaximo >= 1 && CabecerasCargaDirectaValidas(datos.Cabeceras)
}

func DestinoCargaDirectaValido(destino string) bool {
	if destino == "" || destino != strings.TrimSpace(destino) || len(destino) > LongitudMaximaDestinoCargaDirecta {
		return false
	}
	analizado, err := url.Parse(destino)
	return err == nil && analizado.Scheme == "https" && analizado.Host != "" && analizado.User == nil &&
		analizado.Fragment == "" && analizado.Opaque == "" && !analizado.ForceQuery &&
		analizado.Host == strings.ToLower(analizado.Host) && analizado.String() == destino
}

func OrigenDestinoCargaDirecta(destino string) string {
	analizado, err := url.Parse(destino)
	if err != nil {
		return ""
	}
	return strings.ToLower(analizado.Scheme + "://" + analizado.Host)
}

func OrigenesCargaDirectaValidos(origenes []string) bool {
	// Compatibilidad historica: esta regla valida la forma canonica del origen,
	// no su esquema. Los destinos utilizables siguen exigiendo HTTPS. Endurecer
	// tambien esta declaracion requiere una decision de migracion independiente.
	if len(origenes) == 0 || len(origenes) > MaximoOrigenesCargaDirecta {
		return false
	}
	vistos := make(map[string]struct{}, len(origenes))
	for _, origen := range origenes {
		if origen == "" || origen != strings.TrimSpace(origen) || strings.HasSuffix(origen, "/") ||
			OrigenDestinoCargaDirecta(origen) != origen {
			return false
		}
		if _, repetido := vistos[origen]; repetido {
			return false
		}
		vistos[origen] = struct{}{}
	}
	return true
}

func CabecerasCargaDirectaValidas(cabeceras []CabeceraCargaDirecta) bool {
	if len(cabeceras) > MaximoCabecerasCargaDirecta {
		return false
	}
	vistas := make(map[string]struct{}, len(cabeceras))
	for _, cabecera := range cabeceras {
		if cabecera.Nombre != strings.TrimSpace(cabecera.Nombre) ||
			cabecera.Nombre != strings.ToLower(cabecera.Nombre) ||
			!NombreCabeceraCargaDirectaValido(cabecera.Nombre) ||
			!ValorCabeceraCargaDirectaValido(cabecera.Valor) {
			return false
		}
		if _, repetida := vistas[cabecera.Nombre]; repetida {
			return false
		}
		vistas[cabecera.Nombre] = struct{}{}
	}
	return true
}

func NombreCabeceraCargaDirectaValido(nombre string) bool {
	switch nombre {
	case "content-type", "content-md5", "digest", "if-none-match", "x-checksum-sha256",
		"x-amz-checksum-sha256", "x-amz-content-sha256",
		"x-amz-sdk-checksum-algorithm", "x-amz-server-side-encryption",
		"x-amz-server-side-encryption-aws-kms-key-id", "x-amz-server-side-encryption-bucket-key-enabled",
		"x-amz-meta-vec-esquema", "x-amz-meta-vec-conector", "x-amz-meta-vec-zona",
		"x-amz-meta-vec-tamano", "x-amz-meta-vec-sha256", "x-amz-meta-vec-evidencia",
		"x-amz-meta-vec-almacenado-en", "x-amz-meta-vec-idempotencia-sha256",
		"x-amz-meta-vec-vinculo-sesion-sha256", "x-amz-meta-vec-final-referencia",
		"x-amz-meta-vec-mime", "x-amz-meta-vec-expira-en", "x-amz-meta-vec-preparacion-sha256",
		"x-goog-content-sha256", "x-ms-blob-type", "x-ms-content-crc64":
		return true
	default:
		return false
	}
}

func ValorCabeceraCargaDirectaValido(valor string) bool {
	if valor == "" || len(valor) > 2048 || valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) {
			return false
		}
	}
	return true
}

func TextoSeguro(valor string, maximo int) bool {
	if maximo < 1 || valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximo || !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) {
			return false
		}
	}
	return true
}

func ReferenciaOpacaValida(valor string, maximo int) bool {
	if !TextoSeguro(valor, maximo) {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsSpace(caracter) {
			return false
		}
	}
	return true
}

func SHA256HexadecimalValido(valor string) bool {
	if valor != strings.TrimSpace(valor) || valor != strings.ToLower(valor) || len(valor) != 64 {
		return false
	}
	decodificado, err := hex.DecodeString(valor)
	return err == nil && len(decodificado) == sha256.Size
}

type DatosHuellaPreparacionCargaDirecta struct {
	Esquema                string
	OperacionRef           string
	CorrelacionRef         string
	AutorizacionRef        string
	Finalidad              string
	Clasificacion          string
	AccionNegocio          string
	AccionTecnica          string
	CargaRef               string
	SujetoSeudonimoHMAC    string
	RecursoRef             string
	ModuloID               string
	HuellaSolicitudHMAC    string
	EfectoRef              string
	HuellaPlanEfectoSHA256 string
	PasoRef                string
	HuellaDecisionSHA256   string
	ClaveIdempotencia      string
	MIME                   string
	Tamano                 int64
	HuellaSHA256           string
	ExpiraEn               time.Time
}

// HuellaPreparacionCargaDirecta usa campos en orden fijo y prefijo decimal de
// longitud para que ninguna concatenacion ambigua produzca la misma huella.
func HuellaPreparacionCargaDirecta(datos DatosHuellaPreparacionCargaDirecta) string {
	valores := []string{
		datos.Esquema, datos.OperacionRef, datos.CorrelacionRef, datos.AutorizacionRef,
		datos.Finalidad, datos.Clasificacion, datos.AccionNegocio, datos.AccionTecnica,
		datos.CargaRef, datos.SujetoSeudonimoHMAC, datos.RecursoRef, datos.ModuloID,
		datos.HuellaSolicitudHMAC, datos.EfectoRef, datos.HuellaPlanEfectoSHA256,
		datos.PasoRef, datos.HuellaDecisionSHA256, datos.ClaveIdempotencia, datos.MIME,
		strconv.FormatInt(datos.Tamano, 10), datos.HuellaSHA256,
		datos.ExpiraEn.UTC().Format(time.RFC3339Nano),
	}
	var canonico strings.Builder
	for _, valor := range valores {
		canonico.WriteString(strconv.Itoa(len(valor)))
		canonico.WriteByte(':')
		canonico.WriteString(valor)
		canonico.WriteByte('\n')
	}
	suma := sha256.Sum256([]byte(canonico.String()))
	return hex.EncodeToString(suma[:])
}

// LigaduraExacta exige igual cardinalidad, orden y valor. Se usa con
// proyecciones de campos declaradas en el puerto para impedir coincidencias
// parciales o por subconjunto entre autorizacion, solicitud y evidencia.
func LigaduraExacta(observada, esperada []string) bool {
	if len(observada) != len(esperada) {
		return false
	}
	for indice := range observada {
		if observada[indice] != esperada[indice] {
			return false
		}
	}
	return true
}

func AccionOperacionValida(accion string) bool {
	switch accion {
	case AccionEscribir, AccionLeer, AccionPrepararCargaDirecta, AccionConfirmarCargaDirecta,
		AccionAbandonarCargaDirecta, AccionPromover, AccionAplicarRetencion, AccionInmovilizar,
		AccionLevantarInmovilizacion, AccionEliminar, AccionAnalizarContenido:
		return true
	default:
		return false
	}
}

func AccionCreaObjeto(accion string) bool {
	return accion == AccionEscribir || accion == AccionConfirmarCargaDirecta || accion == AccionPromover
}

func AccionIdempotente(accion string) bool {
	return AccionCreaObjeto(accion)
}

func AccionResultadoValida(accion string) bool {
	switch accion {
	case AccionEscribir, AccionConfirmarCargaDirecta, AccionPromover, AccionAplicarRetencion,
		AccionInmovilizar, AccionLevantarInmovilizacion:
		return true
	default:
		return false
	}
}
