package domain

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
)

const (
	EsquemaModeloCanonicoGINPIXV1 = "vec.dipgra.contratacion-temporal.ginpix.modelo.v1"
	EsquemaMapeoGINPIXV1          = "vec.dipgra.contratacion-temporal.ginpix.mapeo.v1"
	EsquemaCargaMapeadaGINPIXV1   = "vec.dipgra.contratacion-temporal.ginpix.carga.v1"

	maximoCamposGINPIX       = 128
	maximoBytesValorGINPIX   = 4_096
	maximoBytesTotalesGINPIX = 65_536
	maximoEnteroSeguroGINPIX = uint64(9_007_199_254_740_991)
)

var (
	ErrModeloGINPIXInvalido         = errors.New("contratacion temporal: modelo canonico ginpix invalido")
	ErrMapeoGINPIXInvalido          = errors.New("contratacion temporal: mapeo ginpix invalido")
	ErrCompatibilidadGINPIXDenegada = errors.New("contratacion temporal: compatibilidad ginpix denegada")
)

// EstadoCampoGINPIX distingue presencia, nulidad y valor sin depender de
// omitempty ni de la representación particular de un adaptador.
type EstadoCampoGINPIX uint8

const (
	EstadoCampoGINPIXAusente EstadoCampoGINPIX = iota + 1
	EstadoCampoGINPIXNulo
	EstadoCampoGINPIXValor
)

// CampoGINPIX conserva un valor canónico. Un valor presente y vacío no es ni
// un nulo ni un campo ausente.
type CampoGINPIX struct {
	Estado EstadoCampoGINPIX `json:"estado"`
	Valor  string            `json:"valor"`
}

func CampoAusenteGINPIX() CampoGINPIX { return CampoGINPIX{Estado: EstadoCampoGINPIXAusente} }

func CampoNuloGINPIX() CampoGINPIX { return CampoGINPIX{Estado: EstadoCampoGINPIXNulo} }

func CampoValorGINPIX(valor string) (CampoGINPIX, error) {
	campo := CampoGINPIX{Estado: EstadoCampoGINPIXValor, Valor: valor}
	if campo.Validar() != nil {
		return CampoGINPIX{}, ErrModeloGINPIXInvalido
	}
	return campo, nil
}

func (c CampoGINPIX) Validar() error {
	switch c.Estado {
	case EstadoCampoGINPIXAusente, EstadoCampoGINPIXNulo:
		if c.Valor != "" {
			return ErrModeloGINPIXInvalido
		}
	case EstadoCampoGINPIXValor:
		if len(c.Valor) > maximoBytesValorGINPIX ||
			!textoValido(c.Valor, maximoBytesValorGINPIX, true) {
			return ErrModeloGINPIXInvalido
		}
	default:
		return ErrModeloGINPIXInvalido
	}
	return nil
}

type DatoCanonicoGINPIX struct {
	Clave ClaveCatalogo `json:"clave"`
	Campo CampoGINPIX   `json:"campo"`
}

type BorradorModeloCanonicoGINPIX struct {
	Esquema           string               `json:"esquema"`
	VersionExpediente uint64               `json:"version_expediente"`
	ExpedienteRef     string               `json:"expediente_ref"`
	IncorporacionRef  string               `json:"incorporacion_ref"`
	ProcedenciaRef    string               `json:"procedencia_ref"`
	CorrelacionRef    string               `json:"correlacion_ref"`
	IdempotenciaRef   string               `json:"idempotencia_ref"`
	Datos             []DatoCanonicoGINPIX `json:"datos"`
}

type PublicacionModeloCanonicoGINPIX struct {
	BorradorModeloCanonicoGINPIX
	HuellaSHA256 string `json:"huella_sha256"`
}

// ModeloCanonicoGINPIX es inmutable desde fuera del paquete. Sus snapshots y
// bytes canónicos siempre son copias defensivas.
type ModeloCanonicoGINPIX struct {
	publicacion PublicacionModeloCanonicoGINPIX
}

func NuevoModeloCanonicoGINPIX(borrador BorradorModeloCanonicoGINPIX) (ModeloCanonicoGINPIX, error) {
	normalizado, err := normalizarModeloGINPIX(borrador)
	if err != nil {
		return ModeloCanonicoGINPIX{}, err
	}
	publicacion := PublicacionModeloCanonicoGINPIX{
		BorradorModeloCanonicoGINPIX: normalizado,
	}
	material := materialModeloGINPIX(publicacion.BorradorModeloCanonicoGINPIX)
	publicacion.HuellaSHA256 = huellaGINPIX(material)
	return ModeloCanonicoGINPIX{publicacion: publicacion}, nil
}

func RestaurarModeloCanonicoGINPIX(publicacion PublicacionModeloCanonicoGINPIX) (ModeloCanonicoGINPIX, error) {
	restaurado, err := NuevoModeloCanonicoGINPIX(
		publicacion.BorradorModeloCanonicoGINPIX,
	)
	if err != nil || !huellasGINPIXIguales(
		publicacion.HuellaSHA256,
		restaurado.publicacion.HuellaSHA256,
	) {
		return ModeloCanonicoGINPIX{}, ErrModeloGINPIXInvalido
	}
	return restaurado, nil
}

func (m ModeloCanonicoGINPIX) Validar() error {
	_, err := RestaurarModeloCanonicoGINPIX(m.Publicacion())
	return err
}

func (m ModeloCanonicoGINPIX) Publicacion() PublicacionModeloCanonicoGINPIX {
	publicacion := m.publicacion
	publicacion.Datos = clonarDatosGINPIX(publicacion.Datos)
	return publicacion
}

func (m ModeloCanonicoGINPIX) SerializarCanonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrModeloGINPIXInvalido
	}
	return append([]byte(nil), materialModeloGINPIX(m.publicacion.BorradorModeloCanonicoGINPIX)...), nil
}

func (m ModeloCanonicoGINPIX) HuellaSHA256() string {
	if m.Validar() != nil {
		return ""
	}
	return m.publicacion.HuellaSHA256
}

type ReglaMapeoGINPIX struct {
	CampoCanonico ClaveCatalogo `json:"campo_canonico"`
	CampoDestino  ClaveCatalogo `json:"campo_destino"`
	Obligatorio   bool          `json:"obligatorio"`
	PermiteNulo   bool          `json:"permite_nulo"`
	PermiteVacio  bool          `json:"permite_vacio"`
}

type BorradorMapeoVersionadoGINPIX struct {
	Esquema        string             `json:"esquema"`
	Referencia     string             `json:"referencia"`
	Version        uint64             `json:"version"`
	ProcedenciaRef string             `json:"procedencia_ref"`
	Reglas         []ReglaMapeoGINPIX `json:"reglas"`
}

type PublicacionMapeoVersionadoGINPIX struct {
	BorradorMapeoVersionadoGINPIX
	HuellaSHA256 string `json:"huella_sha256"`
}

type MapeoVersionadoGINPIX struct {
	publicacion PublicacionMapeoVersionadoGINPIX
}

func PublicarMapeoVersionadoGINPIX(borrador BorradorMapeoVersionadoGINPIX) (MapeoVersionadoGINPIX, error) {
	normalizado, err := normalizarMapeoGINPIX(borrador)
	if err != nil {
		return MapeoVersionadoGINPIX{}, err
	}
	publicacion := PublicacionMapeoVersionadoGINPIX{
		BorradorMapeoVersionadoGINPIX: normalizado,
	}
	publicacion.HuellaSHA256 = huellaGINPIX(materialMapeoGINPIX(normalizado))
	return MapeoVersionadoGINPIX{publicacion: publicacion}, nil
}

func RestaurarMapeoVersionadoGINPIX(publicacion PublicacionMapeoVersionadoGINPIX) (MapeoVersionadoGINPIX, error) {
	restaurado, err := PublicarMapeoVersionadoGINPIX(
		publicacion.BorradorMapeoVersionadoGINPIX,
	)
	if err != nil || !huellasGINPIXIguales(
		publicacion.HuellaSHA256,
		restaurado.publicacion.HuellaSHA256,
	) {
		return MapeoVersionadoGINPIX{}, ErrMapeoGINPIXInvalido
	}
	return restaurado, nil
}

func (m MapeoVersionadoGINPIX) Validar() error {
	_, err := RestaurarMapeoVersionadoGINPIX(m.Publicacion())
	return err
}

func (m MapeoVersionadoGINPIX) Publicacion() PublicacionMapeoVersionadoGINPIX {
	publicacion := m.publicacion
	publicacion.Reglas = append([]ReglaMapeoGINPIX(nil), publicacion.Reglas...)
	return publicacion
}

func (m MapeoVersionadoGINPIX) SerializarCanonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrMapeoGINPIXInvalido
	}
	return append([]byte(nil), materialMapeoGINPIX(m.publicacion.BorradorMapeoVersionadoGINPIX)...), nil
}

type CampoMapeadoGINPIX struct {
	Clave ClaveCatalogo `json:"clave"`
	Campo CampoGINPIX   `json:"campo"`
}

type DatosCargaMapeadaGINPIX struct {
	Esquema              string               `json:"esquema"`
	VersionExpediente    uint64               `json:"version_expediente"`
	ExpedienteRef        string               `json:"expediente_ref"`
	IncorporacionRef     string               `json:"incorporacion_ref"`
	ProcedenciaModeloRef string               `json:"procedencia_modelo_ref"`
	CorrelacionRef       string               `json:"correlacion_ref"`
	IdempotenciaRef      string               `json:"idempotencia_ref"`
	ModeloHuellaSHA256   string               `json:"modelo_huella_sha256"`
	MapeoRef             string               `json:"mapeo_ref"`
	MapeoVersion         uint64               `json:"mapeo_version"`
	ProcedenciaMapeoRef  string               `json:"procedencia_mapeo_ref"`
	MapeoHuellaSHA256    string               `json:"mapeo_huella_sha256"`
	Campos               []CampoMapeadoGINPIX `json:"campos"`
	HuellaSHA256         string               `json:"huella_sha256"`
}

type CargaMapeadaGINPIX struct{ datos DatosCargaMapeadaGINPIX }

// AplicarMapeoGINPIX es una transformación pura. Exige correspondencia uno a
// uno y nunca omite silenciosamente un campo canónico.
func AplicarMapeoGINPIX(modelo ModeloCanonicoGINPIX, mapeo MapeoVersionadoGINPIX) (CargaMapeadaGINPIX, error) {
	if modelo.Validar() != nil || mapeo.Validar() != nil ||
		len(modelo.publicacion.Datos) != len(mapeo.publicacion.Reglas) {
		return CargaMapeadaGINPIX{}, ErrCompatibilidadGINPIXDenegada
	}
	porClave := make(map[ClaveCatalogo]CampoGINPIX, len(modelo.publicacion.Datos))
	for _, dato := range modelo.publicacion.Datos {
		porClave[dato.Clave] = dato.Campo
	}
	campos := make([]CampoMapeadoGINPIX, 0, len(mapeo.publicacion.Reglas))
	for _, regla := range mapeo.publicacion.Reglas {
		campo, existe := porClave[regla.CampoCanonico]
		if !existe || !campoCompatibleGINPIX(campo, regla) {
			return CargaMapeadaGINPIX{}, ErrCompatibilidadGINPIXDenegada
		}
		campos = append(campos, CampoMapeadoGINPIX{
			Clave: regla.CampoDestino,
			Campo: campo,
		})
	}
	sort.Slice(campos, func(i, j int) bool { return campos[i].Clave < campos[j].Clave })
	modeloDatos := modelo.publicacion
	mapeoDatos := mapeo.publicacion
	datos := DatosCargaMapeadaGINPIX{
		Esquema:              EsquemaCargaMapeadaGINPIXV1,
		VersionExpediente:    modeloDatos.VersionExpediente,
		ExpedienteRef:        modeloDatos.ExpedienteRef,
		IncorporacionRef:     modeloDatos.IncorporacionRef,
		ProcedenciaModeloRef: modeloDatos.ProcedenciaRef,
		CorrelacionRef:       modeloDatos.CorrelacionRef,
		IdempotenciaRef:      modeloDatos.IdempotenciaRef,
		ModeloHuellaSHA256:   modeloDatos.HuellaSHA256,
		MapeoRef:             mapeoDatos.Referencia,
		MapeoVersion:         mapeoDatos.Version,
		ProcedenciaMapeoRef:  mapeoDatos.ProcedenciaRef,
		MapeoHuellaSHA256:    mapeoDatos.HuellaSHA256,
		Campos:               campos,
	}
	datos.HuellaSHA256 = huellaGINPIX(materialCargaGINPIX(datos))
	return CargaMapeadaGINPIX{datos: datos}, nil
}

func (c CargaMapeadaGINPIX) Validar() error {
	if c.datos.Esquema != EsquemaCargaMapeadaGINPIXV1 ||
		!huellasGINPIXIguales(c.datos.HuellaSHA256, huellaGINPIX(materialCargaGINPIX(c.datos))) {
		return ErrCompatibilidadGINPIXDenegada
	}
	return nil
}

func (c CargaMapeadaGINPIX) Datos() DatosCargaMapeadaGINPIX {
	datos := c.datos
	datos.Campos = append([]CampoMapeadoGINPIX(nil), datos.Campos...)
	return datos
}

func (c CargaMapeadaGINPIX) SerializarCanonico() ([]byte, error) {
	if c.Validar() != nil {
		return nil, ErrCompatibilidadGINPIXDenegada
	}
	return append([]byte(nil), materialCargaGINPIX(c.datos)...), nil
}

func normalizarModeloGINPIX(borrador BorradorModeloCanonicoGINPIX) (BorradorModeloCanonicoGINPIX, error) {
	if borrador.Esquema != EsquemaModeloCanonicoGINPIXV1 ||
		borrador.VersionExpediente == 0 ||
		borrador.VersionExpediente > maximoEnteroSeguroGINPIX ||
		!referenciaValida(borrador.ExpedienteRef) ||
		!referenciaValida(borrador.IncorporacionRef) ||
		!referenciaValida(borrador.ProcedenciaRef) ||
		!referenciaValida(borrador.CorrelacionRef) ||
		!referenciaValida(borrador.IdempotenciaRef) ||
		len(borrador.Datos) == 0 || len(borrador.Datos) > maximoCamposGINPIX {
		return BorradorModeloCanonicoGINPIX{}, ErrModeloGINPIXInvalido
	}
	claves := make(map[ClaveCatalogo]struct{}, len(borrador.Datos))
	total := 0
	for _, dato := range borrador.Datos {
		total += len(dato.Clave) + len(dato.Campo.Valor)
		if total > maximoBytesTotalesGINPIX || !dato.Clave.Valida() ||
			dato.Campo.Validar() != nil {
			return BorradorModeloCanonicoGINPIX{}, ErrModeloGINPIXInvalido
		}
		if _, repetida := claves[dato.Clave]; repetida {
			return BorradorModeloCanonicoGINPIX{}, ErrModeloGINPIXInvalido
		}
		claves[dato.Clave] = struct{}{}
	}
	normalizado := borrador
	normalizado.Datos = clonarDatosGINPIX(borrador.Datos)
	sort.Slice(normalizado.Datos, func(i, j int) bool {
		return normalizado.Datos[i].Clave < normalizado.Datos[j].Clave
	})
	return normalizado, nil
}

func normalizarMapeoGINPIX(borrador BorradorMapeoVersionadoGINPIX) (BorradorMapeoVersionadoGINPIX, error) {
	if borrador.Esquema != EsquemaMapeoGINPIXV1 ||
		!referenciaValida(borrador.Referencia) || borrador.Version == 0 ||
		borrador.Version > maximoEnteroSeguroGINPIX ||
		!referenciaValida(borrador.ProcedenciaRef) ||
		len(borrador.Reglas) == 0 || len(borrador.Reglas) > maximoCamposGINPIX {
		return BorradorMapeoVersionadoGINPIX{}, ErrMapeoGINPIXInvalido
	}
	origenes := make(map[ClaveCatalogo]struct{}, len(borrador.Reglas))
	destinos := make(map[ClaveCatalogo]struct{}, len(borrador.Reglas))
	for _, regla := range borrador.Reglas {
		if !regla.CampoCanonico.Valida() || !regla.CampoDestino.Valida() {
			return BorradorMapeoVersionadoGINPIX{}, ErrMapeoGINPIXInvalido
		}
		if _, repetida := origenes[regla.CampoCanonico]; repetida {
			return BorradorMapeoVersionadoGINPIX{}, ErrMapeoGINPIXInvalido
		}
		if _, repetida := destinos[regla.CampoDestino]; repetida {
			return BorradorMapeoVersionadoGINPIX{}, ErrMapeoGINPIXInvalido
		}
		origenes[regla.CampoCanonico] = struct{}{}
		destinos[regla.CampoDestino] = struct{}{}
	}
	normalizado := borrador
	normalizado.Reglas = append([]ReglaMapeoGINPIX(nil), borrador.Reglas...)
	sort.Slice(normalizado.Reglas, func(i, j int) bool {
		return normalizado.Reglas[i].CampoCanonico < normalizado.Reglas[j].CampoCanonico
	})
	return normalizado, nil
}

func campoCompatibleGINPIX(campo CampoGINPIX, regla ReglaMapeoGINPIX) bool {
	if campo.Validar() != nil {
		return false
	}
	switch campo.Estado {
	case EstadoCampoGINPIXAusente:
		return !regla.Obligatorio
	case EstadoCampoGINPIXNulo:
		return regla.PermiteNulo
	case EstadoCampoGINPIXValor:
		return campo.Valor != "" || regla.PermiteVacio
	default:
		return false
	}
}

func materialModeloGINPIX(b BorradorModeloCanonicoGINPIX) []byte {
	c := nuevoCanonGINPIX("modelo")
	c.campo("esquema_modelo", b.Esquema)
	c.entero("version_expediente", b.VersionExpediente)
	c.campo("expediente_ref", b.ExpedienteRef)
	c.campo("incorporacion_ref", b.IncorporacionRef)
	c.campo("procedencia_ref", b.ProcedenciaRef)
	c.campo("correlacion_ref", b.CorrelacionRef)
	c.campo("idempotencia_ref", b.IdempotenciaRef)
	c.entero("numero_datos", uint64(len(b.Datos)))
	for _, dato := range b.Datos {
		c.campo("clave", string(dato.Clave))
		c.campo("estado", strconv.Itoa(int(dato.Campo.Estado)))
		c.campo("valor", dato.Campo.Valor)
	}
	return c.bytes()
}

func materialMapeoGINPIX(b BorradorMapeoVersionadoGINPIX) []byte {
	c := nuevoCanonGINPIX("mapeo")
	c.campo("esquema_mapeo", b.Esquema)
	c.campo("referencia", b.Referencia)
	c.entero("version", b.Version)
	c.campo("procedencia_ref", b.ProcedenciaRef)
	c.entero("numero_reglas", uint64(len(b.Reglas)))
	for _, regla := range b.Reglas {
		c.campo("campo_canonico", string(regla.CampoCanonico))
		c.campo("campo_destino", string(regla.CampoDestino))
		c.booleano("obligatorio", regla.Obligatorio)
		c.booleano("permite_nulo", regla.PermiteNulo)
		c.booleano("permite_vacio", regla.PermiteVacio)
	}
	return c.bytes()
}

func materialCargaGINPIX(d DatosCargaMapeadaGINPIX) []byte {
	c := nuevoCanonGINPIX("carga")
	c.campo("esquema_carga", d.Esquema)
	c.entero("version_expediente", d.VersionExpediente)
	c.campo("expediente_ref", d.ExpedienteRef)
	c.campo("incorporacion_ref", d.IncorporacionRef)
	c.campo("procedencia_modelo_ref", d.ProcedenciaModeloRef)
	c.campo("correlacion_ref", d.CorrelacionRef)
	c.campo("idempotencia_ref", d.IdempotenciaRef)
	c.campo("modelo_huella_sha256", d.ModeloHuellaSHA256)
	c.campo("mapeo_ref", d.MapeoRef)
	c.entero("mapeo_version", d.MapeoVersion)
	c.campo("procedencia_mapeo_ref", d.ProcedenciaMapeoRef)
	c.campo("mapeo_huella_sha256", d.MapeoHuellaSHA256)
	c.entero("numero_campos", uint64(len(d.Campos)))
	for _, campo := range d.Campos {
		c.campo("clave", string(campo.Clave))
		c.campo("estado", strconv.Itoa(int(campo.Campo.Estado)))
		c.campo("valor", campo.Campo.Valor)
	}
	return c.bytes()
}

type constructorCanonGINPIX struct{ contenido bytes.Buffer }

func nuevoCanonGINPIX(tipoMaterial string) *constructorCanonGINPIX {
	c := &constructorCanonGINPIX{}
	c.campo("dominio", "vec.dipgra.contratacion-temporal.ginpix")
	c.campo("tipo", tipoMaterial)
	return c
}

func (c *constructorCanonGINPIX) campo(nombre, valor string) {
	c.contenido.WriteString(strconv.Itoa(len(nombre)))
	c.contenido.WriteByte(':')
	c.contenido.WriteString(nombre)
	c.contenido.WriteString(strconv.Itoa(len(valor)))
	c.contenido.WriteByte(':')
	c.contenido.WriteString(valor)
}

func (c *constructorCanonGINPIX) entero(nombre string, valor uint64) {
	c.campo(nombre, strconv.FormatUint(valor, 10))
}

func (c *constructorCanonGINPIX) booleano(nombre string, valor bool) {
	c.campo(nombre, strconv.FormatBool(valor))
}

func (c *constructorCanonGINPIX) bytes() []byte {
	return append([]byte(nil), c.contenido.Bytes()...)
}

func clonarDatosGINPIX(datos []DatoCanonicoGINPIX) []DatoCanonicoGINPIX {
	return append([]DatoCanonicoGINPIX(nil), datos...)
}

func huellaGINPIX(material []byte) string {
	suma := sha256.Sum256(material)
	return hex.EncodeToString(suma[:])
}

func huellasGINPIXIguales(primera, segunda string) bool {
	if len(primera) != sha256.Size*2 || len(segunda) != sha256.Size*2 {
		return false
	}
	primeraBytes, errPrimera := hex.DecodeString(primera)
	segundaBytes, errSegunda := hex.DecodeString(segunda)
	return errPrimera == nil && errSegunda == nil &&
		subtle.ConstantTimeCompare(primeraBytes, segundaBytes) == 1
}
