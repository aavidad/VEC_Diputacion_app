package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrManifiestoGeneracionDocumentalInvalido = errors.New("vec: manifiesto de generacion documental invalido")
	ErrSerializacionManifiestoProhibida       = errors.New("vec: serializacion de manifiesto documental prohibida")
)

const (
	EsquemaManifiestoGeneracionDocumentalV1 = "vec.documentos.manifiesto-generacion.v1"
	esquemaPasoGeneracionDocumentalV1       = "vec.documentos.manifiesto-generacion.paso.v1"
	maximoPasosManifiestoDocumental         = 256
)

// DeclaracionRepresentacionGeneracionDocumental contiene los metadatos de los
// bytes ya renderizados. No concede acceso. La fabrica los canoniza y los
// compromete antes de solicitar una decision al PDP.
type DeclaracionRepresentacionGeneracionDocumental struct {
	ReferenciaLogica  string
	ClaveIdempotencia string
	Formato           domain.FormatoDocumento
	Zona              ZonaAlmacen
	MIME              string
	Tamano            int64
	HuellaSHA256      string
}

type datosPasoGeneracionDocumental struct {
	pasoRef           PasoOperacionAlmacen
	referenciaLogica  string
	claveIdempotencia string
	formato           domain.FormatoDocumento
	zona              ZonaAlmacen
	mime              string
	tamano            int64
	huellaSHA256      string
	huellaPasoSHA256  string
}

func (p datosPasoGeneracionDocumental) validar() error {
	if p.pasoRef == "" || contieneComodinContextoAlmacen(string(p.pasoRef)) ||
		!referenciaOpacaAlmacenValida(p.referenciaLogica, 512) ||
		!referenciaOpacaAlmacenValida(p.claveIdempotencia, 512) ||
		!p.formato.Valido() || !p.zona.Valida() || p.mime != p.formato.MIME() ||
		!textoSeguroAlmacen(p.mime, 255) || p.tamano < 1 ||
		!esSHA256Hexadecimal(p.huellaSHA256) || !esSHA256Hexadecimal(p.huellaPasoSHA256) ||
		contieneComodinContextoAlmacen(p.referenciaLogica, p.claveIdempotencia, p.mime) {
		return ErrManifiestoGeneracionDocumentalInvalido
	}
	return nil
}

type datosManifiestoGeneracionDocumental struct {
	esquema                string
	plantillaID            string
	plantillaVersion       int
	moduloID               string
	tipoDocumental         string
	huellaPlantillaSHA256  string
	permisoGenerar         string
	pasos                  []datosPasoGeneracionDocumental
	huellaManifiestoSHA256 string
}

// ManifiestoGeneracionDocumental es un plan opaco e inmutable. Su valor cero
// y cualquier intento de reconstruirlo por serializacion son invalidos.
// Una generacion simple es exactamente el mismo contrato con un unico paso.
type ManifiestoGeneracionDocumental struct {
	datos *datosManifiestoGeneracionDocumental
}

type ProyeccionPasoGeneracionDocumental struct {
	PasoRef           PasoOperacionAlmacen
	ReferenciaLogica  string
	ClaveIdempotencia string
	Formato           domain.FormatoDocumento
	Zona              ZonaAlmacen
	MIME              string
	Tamano            int64
	HuellaSHA256      string
	HuellaPasoSHA256  string
}

// ProyeccionManifiestoGeneracionDocumental es una copia defensiva interna.
// PermisoGenerar permite formular la solicitud al PDP, pero no es autoridad:
// la fabrica volvera a cotejar la decision con el valor opaco.
type ProyeccionManifiestoGeneracionDocumental struct {
	Esquema                string
	PlantillaID            string
	PlantillaVersion       int
	ModuloID               string
	TipoDocumental         string
	HuellaPlantillaSHA256  string
	PermisoGenerar         string
	HuellaManifiestoSHA256 string
	Pasos                  []ProyeccionPasoGeneracionDocumental
}

func NuevoManifiestoGeneracionDocumental(
	plantilla domain.PlantillaDocumento,
	representaciones []DeclaracionRepresentacionGeneracionDocumental,
) (ManifiestoGeneracionDocumental, error) {
	if plantilla.Validar() != nil || plantilla.Estado != domain.EstadoPlantillaPublicada ||
		!referenciaOpacaAlmacenValida(plantilla.PermisoGenerar, 256) ||
		contieneComodinContextoAlmacen(plantilla.PermisoGenerar) ||
		len(representaciones) == 0 || len(representaciones) > maximoPasosManifiestoDocumental {
		return ManifiestoGeneracionDocumental{}, ErrManifiestoGeneracionDocumentalInvalido
	}
	huellaPlantilla, err := plantilla.HuellaSHA256()
	if err != nil || !esSHA256Hexadecimal(huellaPlantilla) {
		return ManifiestoGeneracionDocumental{}, ErrManifiestoGeneracionDocumentalInvalido
	}

	declaraciones := append([]DeclaracionRepresentacionGeneracionDocumental(nil), representaciones...)
	sort.Slice(declaraciones, func(i, j int) bool {
		return declaraciones[i].ReferenciaLogica < declaraciones[j].ReferenciaLogica
	})
	pasos := make([]datosPasoGeneracionDocumental, 0, len(declaraciones))
	referencias := make(map[string]struct{}, len(declaraciones))
	idempotencias := make(map[string]struct{}, len(declaraciones))
	pasosRef := make(map[PasoOperacionAlmacen]struct{}, len(declaraciones))
	for _, declaracion := range declaraciones {
		if !plantilla.AdmiteFormato(declaracion.Formato) ||
			!declaracionGeneracionDocumentalValida(declaracion) {
			return ManifiestoGeneracionDocumental{}, ErrManifiestoGeneracionDocumentalInvalido
		}
		if _, repetida := referencias[declaracion.ReferenciaLogica]; repetida {
			return ManifiestoGeneracionDocumental{}, ErrManifiestoGeneracionDocumentalInvalido
		}
		if _, repetida := idempotencias[declaracion.ClaveIdempotencia]; repetida {
			return ManifiestoGeneracionDocumental{}, ErrManifiestoGeneracionDocumentalInvalido
		}
		referencias[declaracion.ReferenciaLogica] = struct{}{}
		idempotencias[declaracion.ClaveIdempotencia] = struct{}{}

		huellaPaso := huellaPasoGeneracionDocumental(huellaPlantilla, plantilla.PermisoGenerar, declaracion)
		pasoRef := PasoOperacionAlmacen("generar_documento_" + huellaPaso)
		if !esSHA256Hexadecimal(huellaPaso) {
			return ManifiestoGeneracionDocumental{}, ErrManifiestoGeneracionDocumentalInvalido
		}
		if _, repetido := pasosRef[pasoRef]; repetido {
			return ManifiestoGeneracionDocumental{}, ErrManifiestoGeneracionDocumentalInvalido
		}
		pasosRef[pasoRef] = struct{}{}
		pasos = append(pasos, datosPasoGeneracionDocumental{
			pasoRef: pasoRef, referenciaLogica: declaracion.ReferenciaLogica,
			claveIdempotencia: declaracion.ClaveIdempotencia, formato: declaracion.Formato,
			zona: declaracion.Zona, mime: declaracion.MIME, tamano: declaracion.Tamano,
			huellaSHA256: declaracion.HuellaSHA256, huellaPasoSHA256: huellaPaso,
		})
	}

	datos := &datosManifiestoGeneracionDocumental{
		esquema:     EsquemaManifiestoGeneracionDocumentalV1,
		plantillaID: plantilla.ID, plantillaVersion: plantilla.Version,
		moduloID: plantilla.ModuloID, tipoDocumental: plantilla.TipoDocumental,
		huellaPlantillaSHA256: huellaPlantilla, permisoGenerar: plantilla.PermisoGenerar,
		pasos: clonarPasosGeneracionDocumental(pasos),
	}
	datos.huellaManifiestoSHA256 = huellaManifiestoGeneracionDocumental(*datos)
	manifiesto := ManifiestoGeneracionDocumental{datos: datos}
	if manifiesto.validarEstructura() != nil {
		return ManifiestoGeneracionDocumental{}, ErrManifiestoGeneracionDocumentalInvalido
	}
	return manifiesto, nil
}

func declaracionGeneracionDocumentalValida(d DeclaracionRepresentacionGeneracionDocumental) bool {
	return referenciaOpacaAlmacenValida(d.ReferenciaLogica, 512) &&
		referenciaOpacaAlmacenValida(d.ClaveIdempotencia, 512) &&
		d.Formato.Valido() && d.Zona.Valida() && d.MIME == d.Formato.MIME() &&
		textoSeguroAlmacen(d.MIME, 255) && d.Tamano > 0 && esSHA256Hexadecimal(d.HuellaSHA256) &&
		!contieneComodinContextoAlmacen(d.ReferenciaLogica, d.ClaveIdempotencia, d.MIME)
}

func (m ManifiestoGeneracionDocumental) Proyeccion() (ProyeccionManifiestoGeneracionDocumental, error) {
	if m.validarEstructura() != nil {
		return ProyeccionManifiestoGeneracionDocumental{}, ErrManifiestoGeneracionDocumentalInvalido
	}
	d := m.datos
	proyeccion := ProyeccionManifiestoGeneracionDocumental{
		Esquema: d.esquema, PlantillaID: d.plantillaID, PlantillaVersion: d.plantillaVersion,
		ModuloID: d.moduloID, TipoDocumental: d.tipoDocumental,
		HuellaPlantillaSHA256: d.huellaPlantillaSHA256, PermisoGenerar: d.permisoGenerar,
		HuellaManifiestoSHA256: d.huellaManifiestoSHA256,
		Pasos:                  make([]ProyeccionPasoGeneracionDocumental, 0, len(d.pasos)),
	}
	for _, paso := range d.pasos {
		proyeccion.Pasos = append(proyeccion.Pasos, ProyeccionPasoGeneracionDocumental{
			PasoRef: paso.pasoRef, ReferenciaLogica: paso.referenciaLogica,
			ClaveIdempotencia: paso.claveIdempotencia, Formato: paso.formato,
			Zona: paso.zona, MIME: paso.mime, Tamano: paso.tamano,
			HuellaSHA256: paso.huellaSHA256, HuellaPasoSHA256: paso.huellaPasoSHA256,
		})
	}
	return proyeccion, nil
}

// VincularRecursoGeneracionDocumental crea la copia exacta que debe enviarse
// al PDP. Rechaza atributos reservados preexistentes para impedir que el
// llamador o una entrada externa sombreen los vinculos calculados.
func VincularRecursoGeneracionDocumental(
	recurso domain.RecursoAutorizable,
	manifiesto ManifiestoGeneracionDocumental,
	vinculos VinculosOperacionAlmacen,
) (domain.RecursoAutorizable, error) {
	especificacion, err := manifiesto.especificacionAutorizacionAlmacen()
	if err != nil || recurso.Validar() != nil || contieneComodinRecursoAlmacen(recurso) ||
		recurso.ModuloID != manifiesto.datos.moduloID || !vinculos.validosPara(especificacion) {
		return domain.RecursoAutorizable{}, ErrManifiestoGeneracionDocumentalInvalido
	}
	reservados := []string{
		AtributoAlmacenOperacionRef, AtributoAlmacenCargaRef, AtributoAlmacenClasificacion,
		AtributoAlmacenSujetoSeudonimoHMAC, AtributoAlmacenHuellaSolicitudHMAC,
		AtributoAlmacenEfectoRef, AtributoAlmacenObjetoRef, AtributoAlmacenObjetoVersion,
		AtributoAlmacenHuellaManifiestoSHA256,
	}
	for _, clave := range reservados {
		if _, existe := recurso.Atributos[clave]; existe {
			return domain.RecursoAutorizable{}, ErrManifiestoGeneracionDocumentalInvalido
		}
	}
	resultado := recurso
	resultado.Ambitos = clonarMapaTexto(recurso.Ambitos)
	resultado.Atributos = clonarMapaTexto(recurso.Atributos)
	if resultado.Atributos == nil {
		resultado.Atributos = make(map[string]string, 7)
	}
	resultado.Atributos[AtributoAlmacenOperacionRef] = vinculos.OperacionRef
	resultado.Atributos[AtributoAlmacenCargaRef] = vinculos.CargaRef
	resultado.Atributos[AtributoAlmacenClasificacion] = vinculos.Clasificacion
	resultado.Atributos[AtributoAlmacenSujetoSeudonimoHMAC] = vinculos.SujetoSeudonimoHMAC
	resultado.Atributos[AtributoAlmacenHuellaSolicitudHMAC] = vinculos.HuellaSolicitudHMAC
	resultado.Atributos[AtributoAlmacenEfectoRef] = vinculos.EfectoRef
	resultado.Atributos[AtributoAlmacenHuellaManifiestoSHA256] = manifiesto.datos.huellaManifiestoSHA256
	if resultado.Validar() != nil || !recursoVinculaOperacionAlmacen(resultado, vinculos, especificacion) {
		return domain.RecursoAutorizable{}, ErrManifiestoGeneracionDocumentalInvalido
	}
	return resultado, nil
}

func (m ManifiestoGeneracionDocumental) especificacionAutorizacionAlmacen() (especificacionAutorizacionAlmacen, error) {
	if m.validarEstructura() != nil {
		return especificacionAutorizacionAlmacen{}, ErrManifiestoGeneracionDocumentalInvalido
	}
	pasos := make([]pasoPlanOperacionAlmacen, 0, len(m.datos.pasos))
	for _, paso := range m.datos.pasos {
		copia := paso
		pasos = append(pasos, pasoPlanOperacionAlmacen{
			referencia: paso.pasoRef, accion: AccionAlmacenEscribir,
			huellaPasoSHA256: paso.huellaPasoSHA256, pasoDocumental: &copia,
		})
	}
	especificacion := especificacionAutorizacionAlmacen{
		accionNegocio:          m.datos.permisoGenerar,
		camposExactos:          nil,
		pasos:                  pasos,
		huellaManifiestoSHA256: m.datos.huellaManifiestoSHA256,
	}
	if !especificacion.valida() {
		return especificacionAutorizacionAlmacen{}, ErrManifiestoGeneracionDocumentalInvalido
	}
	return especificacion, nil
}

func (m ManifiestoGeneracionDocumental) validarEstructura() error {
	if m.datos == nil {
		return ErrManifiestoGeneracionDocumentalInvalido
	}
	d := m.datos
	if d.esquema != EsquemaManifiestoGeneracionDocumentalV1 ||
		!referenciaOpacaAlmacenValida(d.plantillaID, 128) || d.plantillaVersion < 1 ||
		!referenciaOpacaAlmacenValida(d.moduloID, 128) ||
		!referenciaOpacaAlmacenValida(d.tipoDocumental, 128) ||
		!esSHA256Hexadecimal(d.huellaPlantillaSHA256) ||
		!referenciaOpacaAlmacenValida(d.permisoGenerar, 256) ||
		!esSHA256Hexadecimal(d.huellaManifiestoSHA256) ||
		contieneComodinContextoAlmacen(d.plantillaID, d.moduloID, d.tipoDocumental, d.permisoGenerar) ||
		len(d.pasos) == 0 || len(d.pasos) > maximoPasosManifiestoDocumental {
		return ErrManifiestoGeneracionDocumentalInvalido
	}
	referencias := make(map[string]struct{}, len(d.pasos))
	idempotencias := make(map[string]struct{}, len(d.pasos))
	pasosRef := make(map[PasoOperacionAlmacen]struct{}, len(d.pasos))
	anterior := ""
	for _, paso := range d.pasos {
		if paso.validar() != nil || (anterior != "" && paso.referenciaLogica <= anterior) {
			return ErrManifiestoGeneracionDocumentalInvalido
		}
		declaracion := DeclaracionRepresentacionGeneracionDocumental{
			ReferenciaLogica: paso.referenciaLogica, ClaveIdempotencia: paso.claveIdempotencia,
			Formato: paso.formato, Zona: paso.zona, MIME: paso.mime,
			Tamano: paso.tamano, HuellaSHA256: paso.huellaSHA256,
		}
		huellaEsperada := huellaPasoGeneracionDocumental(d.huellaPlantillaSHA256, d.permisoGenerar, declaracion)
		if paso.huellaPasoSHA256 != huellaEsperada ||
			paso.pasoRef != PasoOperacionAlmacen("generar_documento_"+huellaEsperada) {
			return ErrManifiestoGeneracionDocumentalInvalido
		}
		if _, existe := referencias[paso.referenciaLogica]; existe {
			return ErrManifiestoGeneracionDocumentalInvalido
		}
		if _, existe := idempotencias[paso.claveIdempotencia]; existe {
			return ErrManifiestoGeneracionDocumentalInvalido
		}
		if _, existe := pasosRef[paso.pasoRef]; existe {
			return ErrManifiestoGeneracionDocumentalInvalido
		}
		referencias[paso.referenciaLogica] = struct{}{}
		idempotencias[paso.claveIdempotencia] = struct{}{}
		pasosRef[paso.pasoRef] = struct{}{}
		anterior = paso.referenciaLogica
	}
	if huellaManifiestoGeneracionDocumental(*d) != d.huellaManifiestoSHA256 {
		return ErrManifiestoGeneracionDocumentalInvalido
	}
	return nil
}

func (c ContextoOperacionAlmacen) validarPasoGeneracionDocumental(
	referenciaLogica, claveIdempotencia string,
	zona ZonaAlmacen,
	mime string,
	tamano int64,
	huellaSHA256 string,
) error {
	if c.validarParaPaso(AccionAlmacenEscribir) != nil || c.datos.huellaManifiestoSHA256 == "" ||
		c.datos.huellaPasoSHA256 == "" {
		return errorAutorizacionAlmacen()
	}
	for _, paso := range c.datos.pasos {
		if paso.referencia == c.datos.pasoRef && paso.pasoDocumental != nil &&
			paso.huellaPasoSHA256 == c.datos.huellaPasoSHA256 {
			declarado := paso.pasoDocumental
			if declarado.referenciaLogica == referenciaLogica &&
				declarado.claveIdempotencia == claveIdempotencia && declarado.zona == zona &&
				declarado.mime == mime && declarado.tamano == tamano &&
				declarado.huellaSHA256 == huellaSHA256 {
				return nil
			}
		}
	}
	return errorAutorizacionAlmacen()
}

// validarEscrituraContraManifiesto no altera los siete planes preexistentes.
// Si el contexto pertenece a un manifiesto, exige todos los metadatos exactos.
func (c ContextoOperacionAlmacen) validarEscrituraContraManifiesto(
	claveIdempotencia string,
	zona ZonaAlmacen,
	mime string,
	tamano int64,
	huellaSHA256 string,
) error {
	if c.validarEstructura() != nil {
		return errorAutorizacionAlmacen()
	}
	if c.datos.huellaManifiestoSHA256 == "" {
		return nil
	}
	for _, paso := range c.datos.pasos {
		if paso.referencia == c.datos.pasoRef && paso.pasoDocumental != nil {
			declarado := paso.pasoDocumental
			if declarado.claveIdempotencia == claveIdempotencia && declarado.zona == zona &&
				declarado.mime == mime && declarado.tamano == tamano &&
				declarado.huellaSHA256 == huellaSHA256 {
				return nil
			}
		}
	}
	return errorAutorizacionAlmacen()
}

func (c ContextoOperacionAlmacen) coincideManifiestoGeneracionDocumental(
	manifiesto ManifiestoGeneracionDocumental,
) bool {
	if c.validarEstructura() != nil || manifiesto.validarEstructura() != nil ||
		c.datos.huellaManifiestoSHA256 != manifiesto.datos.huellaManifiestoSHA256 ||
		c.datos.accionNegocio != manifiesto.datos.permisoGenerar ||
		len(c.datos.pasos) != len(manifiesto.datos.pasos) {
		return false
	}
	for indice, paso := range c.datos.pasos {
		declarado := manifiesto.datos.pasos[indice]
		if paso.referencia != declarado.pasoRef || paso.accion != AccionAlmacenEscribir ||
			paso.huellaPasoSHA256 != declarado.huellaPasoSHA256 || paso.pasoDocumental == nil ||
			*paso.pasoDocumental != declarado {
			return false
		}
	}
	return true
}

func (c ContextoOperacionAlmacen) esPrimerPasoManifiesto() bool {
	return c.validarEstructura() == nil && c.datos.huellaManifiestoSHA256 != "" &&
		len(c.datos.pasos) > 0 && c.datos.pasoRef == c.datos.pasos[0].referencia
}

func (c ContextoOperacionAlmacen) validarResultadoPasoGeneracionDocumental(
	guardado ContenidoDocumentoGuardado,
) error {
	if c.validarParaPaso(AccionAlmacenEscribir) != nil ||
		c.datos.huellaManifiestoSHA256 == "" || guardado.EvidenciaOperacion.Validar() != nil ||
		!evidenciaAlmacenLigada(guardado.EvidenciaOperacion, c) {
		return ErrSolicitudAlmacenInvalida
	}
	objeto := ReferenciaObjetoAlmacen{Referencia: guardado.Referencia, Version: guardado.Version}
	if objeto.Validar() != nil || guardado.EvidenciaOperacion.Objeto != objeto ||
		guardado.EvidenciaOperacion.ConectorID != guardado.ConectorID ||
		guardado.EvidenciaOperacion.Accion != AccionAlmacenEscribir ||
		guardado.EvidenciaOperacion.FundamentoRef != "" {
		return ErrSolicitudAlmacenInvalida
	}
	for _, paso := range c.datos.pasos {
		if paso.referencia != c.datos.pasoRef || paso.pasoDocumental == nil {
			continue
		}
		declarado := paso.pasoDocumental
		if guardado.ReferenciaLogica == declarado.referenciaLogica &&
			guardado.Zona == declarado.zona && guardado.MIME == declarado.mime &&
			guardado.Tamano == declarado.tamano && guardado.HuellaSHA256 == declarado.huellaSHA256 &&
			referenciaOpacaAlmacenValida(guardado.ConectorID, 128) {
			return nil
		}
	}
	return ErrSolicitudAlmacenInvalida
}

func huellaPasoGeneracionDocumental(
	huellaPlantilla, permiso string,
	d DeclaracionRepresentacionGeneracionDocumental,
) string {
	return huellaCanonicaGeneracionDocumental([]string{
		esquemaPasoGeneracionDocumentalV1, huellaPlantilla, permiso,
		d.ReferenciaLogica, d.ClaveIdempotencia, string(d.Formato), string(d.Zona),
		d.MIME, strconv.FormatInt(d.Tamano, 10), d.HuellaSHA256,
	})
}

func huellaManifiestoGeneracionDocumental(d datosManifiestoGeneracionDocumental) string {
	valores := []string{
		d.esquema, d.plantillaID, strconv.Itoa(d.plantillaVersion), d.moduloID,
		d.tipoDocumental, d.huellaPlantillaSHA256, d.permisoGenerar,
		strconv.Itoa(len(d.pasos)),
	}
	for _, paso := range d.pasos {
		valores = append(valores, string(paso.pasoRef), paso.huellaPasoSHA256,
			paso.referenciaLogica, paso.claveIdempotencia, string(paso.formato),
			string(paso.zona), paso.mime, strconv.FormatInt(paso.tamano, 10), paso.huellaSHA256)
	}
	return huellaCanonicaGeneracionDocumental(valores)
}

func huellaCanonicaGeneracionDocumental(valores []string) string {
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

func clonarPasosGeneracionDocumental(pasos []datosPasoGeneracionDocumental) []datosPasoGeneracionDocumental {
	return append([]datosPasoGeneracionDocumental(nil), pasos...)
}

func clonarMapaTexto(origen map[string]string) map[string]string {
	if origen == nil {
		return nil
	}
	resultado := make(map[string]string, len(origen))
	for clave, valor := range origen {
		resultado[clave] = valor
	}
	return resultado
}

func (ManifiestoGeneracionDocumental) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionManifiestoProhibida
}

func (*ManifiestoGeneracionDocumental) UnmarshalJSON([]byte) error {
	return ErrSerializacionManifiestoProhibida
}

func (ManifiestoGeneracionDocumental) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionManifiestoProhibida
}

func (*ManifiestoGeneracionDocumental) UnmarshalText([]byte) error {
	return ErrSerializacionManifiestoProhibida
}

func (ManifiestoGeneracionDocumental) String() string {
	return "[MANIFIESTO-GENERACION-DOCUMENTAL-OPACO]"
}

func (m ManifiestoGeneracionDocumental) GoString() string { return m.String() }

func (m ManifiestoGeneracionDocumental) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, m.String())
}

func (m ManifiestoGeneracionDocumental) LogValue() slog.Value {
	return slog.StringValue(m.String())
}

func (ProyeccionManifiestoGeneracionDocumental) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionManifiestoProhibida
}

func (*ProyeccionManifiestoGeneracionDocumental) UnmarshalJSON([]byte) error {
	return ErrSerializacionManifiestoProhibida
}

func (ProyeccionManifiestoGeneracionDocumental) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionManifiestoProhibida
}

func (*ProyeccionManifiestoGeneracionDocumental) UnmarshalText([]byte) error {
	return ErrSerializacionManifiestoProhibida
}

func (ProyeccionManifiestoGeneracionDocumental) String() string {
	return "[PROYECCION-MANIFIESTO-GENERACION-DOCUMENTAL-INTERNA]"
}

func (p ProyeccionManifiestoGeneracionDocumental) GoString() string { return p.String() }

func (p ProyeccionManifiestoGeneracionDocumental) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}

func (p ProyeccionManifiestoGeneracionDocumental) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
