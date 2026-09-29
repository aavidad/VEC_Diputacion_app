// Package datospersonales resuelve el catálogo de datos personales por
// finalidad y momento: qué datos pide cada tipo de convocatoria (bolsa,
// proceso selectivo libre, promoción interna y provisión de puestos), si son
// obligatorios, condicionales o voluntarios, en qué momento del recorrido se
// piden, con qué finalidad y base del RGPD, de dónde se obtienen (identificación
// electrónica, la propia persona, otra Administración o Personal) y si son de
// categoría especial.
//
// Es la fuente única para que el módulo Aspirantes pida solo lo necesario en
// cada momento (RGPD art. 5.1.c): un dato que no figura en el catálogo vigente
// para ese tipo y momento no se pide ni se admite (denegación por defecto).
//
// El paquete no contiene textos visibles: el catálogo lleva códigos y la
// interfaz los traduce con web/static/textos/<idioma>/datos-personales.json.
// En este corte solo existe el paquete de ejemplo
// (data/demo/reglas/aspirantes_datos_personales.ejemplo.demo.json), pendiente de
// RRHH y del DPD; se retira antes de producción como los demás paquetes de
// ejemplo. Una versión aprobada se publica como nueva versión del catálogo.
package datospersonales

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Identificador del catálogo y del módulo que lo posee.
const (
	CatalogoDatosPersonales = "vec.aspirantes.datos_personales"
	ModuloAspirantes        = "aspirantes"
)

var (
	// ErrCatalogoNoConfigurado: no hay catálogo compuesto. El consumidor no
	// debe pedir ningún dato opcional ni suponer una lista por defecto.
	ErrCatalogoNoConfigurado = errors.New("datospersonales: catalogo no configurado")
	// ErrCatalogoNoDisponible: el catálogo no existe, no está vigente o no se
	// puede leer. Nunca se interpreta como «se puede pedir cualquier dato».
	ErrCatalogoNoDisponible = errors.New("datospersonales: catalogo no disponible")
	// ErrCatalogoInvalido: alguna entrada vigente no cumple el contrato; el
	// catálogo entero queda invalidado.
	ErrCatalogoInvalido = errors.New("datospersonales: catalogo no valido")
	ErrConfiguracion    = errors.New("datospersonales: configuracion no valida")
)

// TipoConvocatoria es el tipo de procedimiento que pide los datos.
type TipoConvocatoria string

const (
	TipoBolsa            TipoConvocatoria = "bolsa"
	TipoSelectivoLibre   TipoConvocatoria = "selectivo_libre"
	TipoPromocionInterna TipoConvocatoria = "promocion_interna"
	TipoProvision        TipoConvocatoria = "provision"
)

// TiposConvocatoria devuelve los tipos en su orden de presentación.
func TiposConvocatoria() []TipoConvocatoria {
	return []TipoConvocatoria{TipoBolsa, TipoSelectivoLibre, TipoPromocionInterna, TipoProvision}
}

func (t TipoConvocatoria) Valido() bool {
	switch t {
	case TipoBolsa, TipoSelectivoLibre, TipoPromocionInterna, TipoProvision:
		return true
	}
	return false
}

// Momento del recorrido en que se pide el dato, en orden cronológico.
type Momento string

const (
	MomentoInscripcion  Momento = "inscripcion"
	MomentoPagoTasa     Momento = "pago_tasa"
	MomentoAdmision     Momento = "admision"
	MomentoBaremacion   Momento = "baremacion"
	MomentoPruebas      Momento = "pruebas"
	MomentoLlamamiento  Momento = "llamamiento"
	MomentoContratacion Momento = "contratacion"
)

// Momentos devuelve los momentos en orden cronológico.
func Momentos() []Momento {
	return []Momento{MomentoInscripcion, MomentoPagoTasa, MomentoAdmision, MomentoBaremacion,
		MomentoPruebas, MomentoLlamamiento, MomentoContratacion}
}

// Posicion es el orden cronológico del momento (1..7) o 0 si no es válido.
func (m Momento) Posicion() int {
	for indice, momento := range Momentos() {
		if momento == m {
			return indice + 1
		}
	}
	return 0
}

// Obligatoriedad indica si el dato se exige siempre, solo si se da una
// condición o solo si la persona quiere aportarlo.
type Obligatoriedad string

const (
	Obligatorio Obligatoriedad = "obligatorio"
	Condicional Obligatoriedad = "condicional"
	Voluntario  Obligatoriedad = "voluntario"
)

// Fuente es de dónde se obtiene el dato.
type Fuente string

const (
	// FuenteIdentificacionElectronica: certificado, DNIe o Cl@ve. La persona
	// lo ve, no lo escribe.
	FuenteIdentificacionElectronica Fuente = "identificacion_electronica"
	FuentePersona                   Fuente = "persona"
	// FuenteConsultaAdministracion: se consulta a otra Administración en
	// lugar de pedirlo (Ley 39/2015, art. 28.2).
	FuenteConsultaAdministracion Fuente = "consulta_administracion"
	// FuentePersonalDiputacion: la Diputación ya lo tiene en Personal.
	FuentePersonalDiputacion Fuente = "personal_diputacion"
	// FuenteVigilanciaSalud: RRHH solo recibe «apto» o «no apto».
	FuenteVigilanciaSalud Fuente = "vigilancia_salud"
)

func (f Fuente) valida() bool {
	switch f {
	case FuenteIdentificacionElectronica, FuentePersona, FuenteConsultaAdministracion,
		FuentePersonalDiputacion, FuenteVigilanciaSalud:
		return true
	}
	return false
}

// RegimenConsulta distingue la consulta que se presume autorizada salvo
// oposición expresa (Ley 39/2015, art. 28.2) de la que exige autorización
// previa (datos tributarios, Ley 58/2003, art. 95.1.k).
type RegimenConsulta string

const (
	RegimenOposicion    RegimenConsulta = "oposicion"
	RegimenAutorizacion RegimenConsulta = "autorizacion"
)

// Categoria del dato a efectos del RGPD.
type Categoria string

const (
	CategoriaOrdinaria Categoria = "ordinaria"
	// CategoriaEspecial: art. 9 RGPD (salud, discapacidad) o datos que lo
	// revelan, como un turno de reserva.
	CategoriaEspecial Categoria = "especial"
	// CategoriaPenal: art. 10 RGPD; solo cuando una norma lo exige.
	CategoriaPenal Categoria = "penal"
	// CategoriaProtegida: no es del art. 9, pero exige protección reforzada
	// (víctimas de violencia, LOPDGDD disposición adicional 7.ª).
	CategoriaProtegida Categoria = "protegida"
)

func (c Categoria) valida() bool {
	switch c {
	case CategoriaOrdinaria, CategoriaEspecial, CategoriaPenal, CategoriaProtegida:
		return true
	}
	return false
}

// EsSensible es cierto para todo lo que no es ordinario.
func (c Categoria) EsSensible() bool { return c != CategoriaOrdinaria }

// Custodia es el módulo que pide y guarda el dato. Los datos del
// nombramiento o del contrato nunca se guardan en Aspirantes, y los trámites
// de empleado (promoción interna y provisión) se hacen desde el portal
// interno sin pasar por Aspirantes (estudio, apartado 4.8).
type Custodia string

const (
	CustodiaAspirantes       Custodia = "aspirantes"
	CustodiaPersonal         Custodia = "personal"
	CustodiaProcesosEmpleado Custodia = "procesos_empleado"
)

// EsTramiteEmpleado es cierto para los tipos que solo usa el personal de la
// Diputación desde el portal interno.
func (t TipoConvocatoria) EsTramiteEmpleado() bool {
	return t == TipoPromocionInterna || t == TipoProvision
}

// Origen distingue el valor provisional de ejemplo del aprobado.
type Origen string

const (
	OrigenEjemplo  Origen = "ejemplo"
	OrigenAprobado Origen = "aprobado"
)

// PendienteDe indica quién debe confirmar un valor de ejemplo.
type PendienteDe string

const (
	PendienteRRHH    PendienteDe = "rrhh"
	PendienteDPD     PendienteDe = "dpd"
	PendienteRRHHDPD PendienteDe = "rrhh_dpd"
)

var (
	patronCodigo    = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	patronBaseRGPD  = regexp.MustCompile(`^[0-9]{1,2}\.[0-9]\.[a-j](,[0-9]{1,2}\.[0-9]\.[a-j]){0,3}$`)
	atributosPermit = map[string]bool{
		"tipo_convocatoria": true, "dato": true, "momento": true, "momento_comprobacion": true,
		"obligatoriedad": true, "condicion": true, "finalidad": true, "base_rgpd": true,
		"fuente": true, "fuente_alternativa": true, "consulta_servicio": true, "consulta_regimen": true,
		"categoria": true, "custodia": true, "origen": true, "pendiente_de": true,
		"aprobacion_ref": true, "norma": true, "duda": true,
	}
)

// maximoEntradas acota el catálogo vigente antes de construir la lista.
const maximoEntradas = 512

// maximoCaracteresCita acota norma y duda de cada entrada.
const maximoCaracteresCita = 512

// DatoRequerido es una entrada resuelta: un dato que un tipo de convocatoria
// pide en un momento. Los campos de texto son códigos, no textos visibles.
type DatoRequerido struct {
	Clave               string
	Tipo                TipoConvocatoria
	Dato                string
	Momento             Momento
	MomentoComprobacion Momento
	Obligatoriedad      Obligatoriedad
	Condicion           string
	Finalidad           string
	BaseRGPD            []string
	Fuente              Fuente
	// FuenteAlternativa es a quién se pide el dato si la persona se opone a
	// la consulta o si la fuente preferente no lo aporta.
	FuenteAlternativa Fuente
	ConsultaServicio  string
	ConsultaRegimen   RegimenConsulta
	Categoria         Categoria
	Custodia          Custodia
	Origen            Origen
	PendienteDe       PendienteDe
	AprobacionRef     string
	// Norma y Duda son citas del catálogo (datos, no textos de interfaz).
	Norma string
	Duda  string
	// Referencia es catalogo:version:entrada; ReferenciaEntrada añade la
	// huella del catálogo para dejar constancia de qué versión se aplicó.
	Referencia        string
	ReferenciaEntrada domain.ReferenciaEntradaCatalogo
}

// EsEjemplo es cierto si el valor está pendiente de RRHH o del DPD. La
// interfaz debe rotularlo («pendiente de RRHH/DPD»).
func (d DatoRequerido) EsEjemplo() bool { return d.Origen == OrigenEjemplo }

// SeConsulta es cierto si el dato se obtiene de otra Administración.
func (d DatoRequerido) SeConsulta() bool { return d.Fuente == FuenteConsultaAdministracion }

// Catalogo es la versión vigente resuelta con su procedencia.
type Catalogo struct {
	Datos          []DatoRequerido
	CatalogoID     string
	Version        int
	HuellaCatalogo string
	PaqueteEjemplo bool
}

// Para devuelve, en el orden del catálogo, los datos que ese tipo pide en ese
// momento. Una lista vacía significa «no pedir nada».
func (c Catalogo) Para(tipo TipoConvocatoria, momento Momento) []DatoRequerido {
	var salida []DatoRequerido
	for _, dato := range c.Datos {
		if dato.Tipo == tipo && dato.Momento == momento {
			salida = append(salida, copiaDato(dato))
		}
	}
	return salida
}

// Permitido indica si ese tipo puede pedir ese dato en ese momento y devuelve
// su entrada. Sirve para rechazar en la frontera cualquier campo que el
// catálogo no prevea.
func (c Catalogo) Permitido(tipo TipoConvocatoria, momento Momento, dato string) (DatoRequerido, bool) {
	for _, entrada := range c.Datos {
		if entrada.Tipo == tipo && entrada.Momento == momento && entrada.Dato == dato {
			return copiaDato(entrada), true
		}
	}
	return DatoRequerido{}, false
}

func copiaDato(dato DatoRequerido) DatoRequerido {
	dato.BaseRGPD = append([]string(nil), dato.BaseRGPD...)
	return dato
}

// Resolutor lee el catálogo en cada consulta, sin estado mutable, para que
// una versión nueva se aplique sin reiniciar. Un puntero nulo es válido y
// responde ErrCatalogoNoConfigurado.
type Resolutor struct {
	base      *reglas.Resolutor
	metadatos ports.ConsultaMetadatosFuenteCatalogos
}

// NuevoResolutor compone el catálogo sobre el mismo puerto de catálogos que
// los demás catálogos gobernados de VEC. Metadatos es opcional.
func NuevoResolutor(
	consulta ports.ConsultaCatalogosConfigurablesAcotada,
	metadatos ports.ConsultaMetadatosFuenteCatalogos,
	reloj reglas.Reloj,
) (*Resolutor, error) {
	if esNulo(metadatos) {
		metadatos = nil
	}
	base, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: metadatos, Reloj: reloj,
		CatalogoID: CatalogoDatosPersonales, ModuloID: ModuloAspirantes,
	})
	if err != nil {
		return nil, ErrConfiguracion
	}
	return &Resolutor{base: base, metadatos: metadatos}, nil
}

// Catalogo resuelve la versión vigente completa. Una entrada vigente no válida
// invalida el catálogo entero.
func (r *Resolutor) Catalogo(ctx context.Context) (Catalogo, error) {
	if r == nil {
		return Catalogo{}, ErrCatalogoNoConfigurado
	}
	if ctx == nil {
		return Catalogo{}, ErrCatalogoNoDisponible
	}
	catalogo, huella, instante, err := r.base.CatalogoVigente(ctx)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Catalogo{}, ctxErr
		}
		return Catalogo{}, ErrCatalogoNoDisponible
	}
	ejemplo := catalogo.FuenteRef == reglas.MarcaPaqueteEjemplo
	if r.metadatos != nil {
		fuente, err := r.metadatos.ObtenerMetadatosFuenteCatalogos(ctx)
		if err != nil {
			return Catalogo{}, ErrCatalogoNoDisponible
		}
		ejemplo = ejemplo || fuente.Demostracion
	}
	resultado := Catalogo{
		CatalogoID: catalogo.ID, Version: catalogo.Version, HuellaCatalogo: huella, PaqueteEjemplo: ejemplo,
	}
	vistos := map[string]bool{}
	for _, entrada := range catalogo.Entradas {
		if !entrada.VigenteEn(instante) {
			continue
		}
		dato, err := datoDesdeEntrada(catalogo, huella, ejemplo, entrada)
		if err != nil {
			return Catalogo{}, err
		}
		unico := string(dato.Tipo) + "|" + string(dato.Momento) + "|" + dato.Dato
		if vistos[unico] || len(resultado.Datos) >= maximoEntradas {
			return Catalogo{}, ErrCatalogoInvalido
		}
		vistos[unico] = true
		resultado.Datos = append(resultado.Datos, dato)
	}
	if len(resultado.Datos) == 0 {
		return Catalogo{}, ErrCatalogoNoDisponible
	}
	return resultado, nil
}

// Para resuelve el catálogo vigente y devuelve los datos de ese tipo y
// momento junto con la procedencia (sin Datos) para anotarla en el recibo.
func (r *Resolutor) Para(ctx context.Context, tipo TipoConvocatoria, momento Momento) ([]DatoRequerido, Catalogo, error) {
	if !tipo.Valido() || momento.Posicion() == 0 {
		return nil, Catalogo{}, ErrConfiguracion
	}
	catalogo, err := r.Catalogo(ctx)
	if err != nil {
		return nil, Catalogo{}, err
	}
	datos := catalogo.Para(tipo, momento)
	catalogo.Datos = nil
	return datos, catalogo, nil
}

// datoDesdeEntrada aplica el contrato de atributos y las invariantes que el
// código sabe interpretar. Los valores concretos viven en el catálogo.
func datoDesdeEntrada(catalogo domain.CatalogoConfigurable, huella string, ejemplo bool, entrada domain.EntradaCatalogoConfigurable) (DatoRequerido, error) {
	a := entrada.Atributos
	for clave := range a {
		if !atributosPermit[clave] {
			return DatoRequerido{}, ErrCatalogoInvalido
		}
	}
	dato := DatoRequerido{
		Clave: entrada.Clave, Tipo: TipoConvocatoria(a["tipo_convocatoria"]), Dato: a["dato"],
		Momento: Momento(a["momento"]), MomentoComprobacion: Momento(a["momento_comprobacion"]),
		Obligatoriedad: Obligatoriedad(a["obligatoriedad"]), Condicion: a["condicion"],
		Finalidad: a["finalidad"], Fuente: Fuente(a["fuente"]), FuenteAlternativa: Fuente(a["fuente_alternativa"]),
		ConsultaServicio: a["consulta_servicio"], ConsultaRegimen: RegimenConsulta(a["consulta_regimen"]),
		Categoria: Categoria(a["categoria"]), Custodia: Custodia(a["custodia"]), Origen: Origen(a["origen"]),
		PendienteDe: PendienteDe(a["pendiente_de"]), AprobacionRef: a["aprobacion_ref"],
		Norma: a["norma"], Duda: a["duda"],
		ReferenciaEntrada: domain.ReferenciaEntradaCatalogo{
			CatalogoID: catalogo.ID, CatalogoVersion: catalogo.Version,
			CatalogoHuellaSHA256: huella, EntradaClave: entrada.Clave,
		},
	}
	if a["base_rgpd"] != "" {
		dato.BaseRGPD = strings.Split(a["base_rgpd"], ",")
	}
	dato.Referencia = dato.ReferenciaEntrada.Referencia()
	if dato.ReferenciaEntrada.Validar() != nil || !contratoValido(dato, a["base_rgpd"], ejemplo) {
		return DatoRequerido{}, ErrCatalogoInvalido
	}
	return dato, nil
}

func contratoValido(d DatoRequerido, baseRGPD string, ejemplo bool) bool {
	return d.Tipo.Valido() && patronCodigo.MatchString(d.Dato) && d.Momento.Posicion() > 0 &&
		comprobacionValida(d) && obligatoriedadValida(d) && patronCodigo.MatchString(d.Finalidad) &&
		patronBaseRGPD.MatchString(baseRGPD) && fuentesValidas(d) && d.Categoria.valida() &&
		custodiaValida(d) && categoriaCoherente(d) && origenValido(d, ejemplo) && citaValida(d.Norma) && citaValida(d.Duda)
}

// comprobacionValida: la comprobación, si existe, es posterior a la petición.
func comprobacionValida(d DatoRequerido) bool {
	return d.MomentoComprobacion == "" || d.MomentoComprobacion.Posicion() > d.Momento.Posicion()
}

// obligatoriedadValida: un dato condicional nombra su condición y solo él.
func obligatoriedadValida(d DatoRequerido) bool {
	switch d.Obligatoriedad {
	case Condicional:
		return patronCodigo.MatchString(d.Condicion)
	case Obligatorio, Voluntario:
		return d.Condicion == ""
	}
	return false
}

// fuentesValidas: la consulta a otra Administración nombra servicio y
// régimen y tiene alternativa (el documento que aporta la persona si se
// opone); ninguna otra fuente lleva servicio ni régimen.
func fuentesValidas(d DatoRequerido) bool {
	if !d.Fuente.valida() || (d.FuenteAlternativa != "" && (!d.FuenteAlternativa.valida() || d.FuenteAlternativa == d.Fuente)) {
		return false
	}
	if d.Fuente == FuenteConsultaAdministracion {
		return patronCodigo.MatchString(d.ConsultaServicio) &&
			(d.ConsultaRegimen == RegimenOposicion || d.ConsultaRegimen == RegimenAutorizacion) &&
			d.FuenteAlternativa == FuentePersona
	}
	return d.ConsultaServicio == "" && d.ConsultaRegimen == ""
}

// custodiaValida: los datos del nombramiento o contrato los pide y guarda
// Personal, nunca Aspirantes; y solo esos. Un dato penal (art. 10 RGPD) solo
// cabe ahí.
func custodiaValida(d DatoRequerido) bool {
	switch d.Custodia {
	case CustodiaPersonal:
		return d.Momento == MomentoContratacion
	case CustodiaAspirantes:
		return !d.Tipo.EsTramiteEmpleado() && d.Momento != MomentoContratacion && d.Categoria != CategoriaPenal
	case CustodiaProcesosEmpleado:
		return d.Tipo.EsTramiteEmpleado() && d.Momento != MomentoContratacion && d.Categoria != CategoriaPenal
	}
	return false
}

// categoriaCoherente: un dato sensible que se guarda fuera de Personal nunca
// es obligatorio, y un dato especial cita una letra del art. 9.2 del RGPD.
func categoriaCoherente(d DatoRequerido) bool {
	if d.Categoria.EsSensible() && d.Custodia != CustodiaPersonal && d.Obligatoriedad == Obligatorio {
		return false
	}
	if d.Categoria != CategoriaEspecial {
		return true
	}
	for _, base := range d.BaseRGPD {
		if strings.HasPrefix(base, "9.2.") {
			return true
		}
	}
	return false
}

// citaValida acota las citas de norma y duda, que son datos del catálogo.
func citaValida(texto string) bool {
	return texto != "" && len(texto) <= maximoCaracteresCita && texto == strings.TrimSpace(texto)
}

// origenValido: un paquete de ejemplo solo contiene valores de ejemplo con
// quién debe confirmarlos, y un valor de ejemplo no cabe fuera de él. Un valor
// aprobado cita su aprobación.
func origenValido(d DatoRequerido, ejemplo bool) bool {
	switch d.Origen {
	case OrigenEjemplo:
		return ejemplo && d.AprobacionRef == "" &&
			(d.PendienteDe == PendienteRRHH || d.PendienteDe == PendienteDPD || d.PendienteDe == PendienteRRHHDPD)
	case OrigenAprobado:
		return !ejemplo && d.PendienteDe == "" && d.AprobacionRef != "" && d.AprobacionRef == strings.TrimSpace(d.AprobacionRef)
	}
	return false
}

func esNulo(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	}
	return false
}
