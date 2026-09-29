package domain

import (
	"errors"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrCampoInvalido  = errors.New("aspirantes: campo o valor invalido")
	ErrCambioInvalido = errors.New("aspirantes: cambio de ficha invalido")
)

// CampoFicha es un dato de la ficha. El vocabulario es cerrado y no incluye
// datos de categoría especial (salud, discapacidad, víctimas): esos irán en
// un almacén aparte, con su propia clave, en la fase 2 del plan.
type CampoFicha string

const (
	CampoNombre          CampoFicha = "nombre"
	CampoPrimerApellido  CampoFicha = "primer_apellido"
	CampoSegundoApellido CampoFicha = "segundo_apellido"
	CampoTelefono        CampoFicha = "telefono"
	CampoMovil           CampoFicha = "movil"
	CampoDomicilio       CampoFicha = "domicilio"
	CampoCodigoPostal    CampoFicha = "codigo_postal"
)

// CamposIdentidad salen del certificado; la persona no los escribe.
var CamposIdentidad = []CampoFicha{CampoNombre, CampoPrimerApellido, CampoSegundoApellido}

// CamposContacto los declara la persona y solo si el catálogo los pide.
var CamposContacto = []CampoFicha{CampoTelefono, CampoMovil, CampoDomicilio, CampoCodigoPostal}

func (c CampoFicha) EsIdentidad() bool {
	return c == CampoNombre || c == CampoPrimerApellido || c == CampoSegundoApellido
}

func (c CampoFicha) EsContacto() bool {
	return c == CampoTelefono || c == CampoMovil || c == CampoDomicilio || c == CampoCodigoPostal
}

func (c CampoFicha) Valido() bool { return c.EsIdentidad() || c.EsContacto() }

func condicionValida(s string) bool {
	if len(s) > 64 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

// OrigenValor dice de dónde procede un valor guardado.
type OrigenValor string

const (
	OrigenCertificado OrigenValor = "certificado"
	OrigenTitular     OrigenValor = "titular"
)

func (o OrigenValor) Valido() bool { return o == OrigenCertificado || o == OrigenTitular }

// OrigenEsperado devuelve el único origen admitido para cada campo.
func (c CampoFicha) OrigenEsperado() OrigenValor {
	if c.EsIdentidad() {
		return OrigenCertificado
	}
	return OrigenTitular
}

// NormalizarValor valida y normaliza el valor de un campo. Una cadena vacía
// no es un valor: retirar un dato se expresa aparte.
func NormalizarValor(campo CampoFicha, valor string) (string, error) {
	switch campo {
	case CampoNombre, CampoPrimerApellido, CampoSegundoApellido:
		return normalizarNombre(valor)
	case CampoTelefono:
		return normalizarTelefono(valor, false)
	case CampoMovil:
		return normalizarTelefono(valor, true)
	case CampoDomicilio:
		return normalizarDomicilio(valor)
	case CampoCodigoPostal:
		return normalizarCodigoPostal(valor)
	}
	return "", ErrCampoInvalido
}

func colapsarEspacios(s string) string { return strings.Join(strings.Fields(s), " ") }

func normalizarNombre(valor string) (string, error) {
	v := colapsarEspacios(valor)
	if v == "" || utf8.RuneCountInString(v) > 100 || !utf8.ValidString(v) {
		return "", ErrCampoInvalido
	}
	letras := 0
	for _, r := range v {
		switch {
		case unicode.IsLetter(r) || unicode.Is(unicode.Mn, r):
			letras++
		case r == ' ' || r == '\'' || r == '’' || r == '-' || r == '.' || r == 'ª' || r == 'º':
		default:
			return "", ErrCampoInvalido
		}
	}
	if letras == 0 {
		return "", ErrCampoInvalido
	}
	return v, nil
}

// normalizarTelefono admite 9 cifras nacionales (fijo o móvil) o un número
// internacional con «+» o «00» y entre 8 y 15 cifras. Se guarda sin
// separadores; los internacionales con «+».
func normalizarTelefono(valor string, soloMovil bool) (string, error) {
	var b strings.Builder
	for i, r := range strings.TrimSpace(valor) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' && i == 0:
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '.' || r == '(' || r == ')':
		default:
			return "", ErrCampoInvalido
		}
	}
	v := b.String()
	if strings.HasPrefix(v, "00") {
		v = "+" + v[2:]
	}
	if strings.HasPrefix(v, "+34") {
		// Un número español siempre tiene 9 cifras tras el prefijo.
		if len(v) != 12 {
			return "", ErrCampoInvalido
		}
		v = v[3:]
	}
	if strings.HasPrefix(v, "+") {
		cifras := v[1:]
		if len(cifras) < 8 || len(cifras) > 15 || cifras[0] == '0' {
			return "", ErrCampoInvalido
		}
		return v, nil
	}
	if len(v) != 9 {
		return "", ErrCampoInvalido
	}
	iniciales := "6789"
	if soloMovil {
		iniciales = "67"
	}
	if !strings.ContainsRune(iniciales, rune(v[0])) {
		return "", ErrCampoInvalido
	}
	return v, nil
}

func normalizarDomicilio(valor string) (string, error) {
	v := colapsarEspacios(valor)
	n := utf8.RuneCountInString(v)
	if !utf8.ValidString(v) || n < 5 || n > 200 {
		return "", ErrCampoInvalido
	}
	for _, r := range v {
		if !unicode.IsPrint(r) {
			return "", ErrCampoInvalido
		}
	}
	return v, nil
}

// normalizarCodigoPostal admite solo códigos postales españoles (01000–52999).
// Un domicilio en otro país exigirá antes un campo de país.
func normalizarCodigoPostal(valor string) (string, error) {
	v := strings.TrimSpace(valor)
	n, ok := digitos(v)
	if !ok || len(v) != 5 || n/1000 < 1 || n/1000 > 52 {
		return "", ErrCampoInvalido
	}
	return v, nil
}

// MotivoCambio explica por qué cambia un dato de contacto. El alta tiene su
// propio motivo y no lo elige la persona.
type MotivoCambio string

const (
	MotivoAltaTitular       MotivoCambio = "alta_titular"
	MotivoDatoNuevo         MotivoCambio = "dato_nuevo"
	MotivoCambioDeDato      MotivoCambio = "cambio_de_dato"
	MotivoCorreccionDeError MotivoCambio = "correccion_de_error"
)

// ValidoParaRectificar dice si la persona puede elegir este motivo.
func (m MotivoCambio) ValidoParaRectificar() bool {
	return m == MotivoDatoNuevo || m == MotivoCambioDeDato || m == MotivoCorreccionDeError
}

// ExigenciaCampo dice que el catálogo pide un dato de contacto y si es
// obligatorio para las finalidades vigentes de la persona. Condicion es el
// código del catálogo que explica cuándo se pide (vacío si siempre).
type ExigenciaCampo struct {
	Campo       CampoFicha
	Obligatorio bool
	Condicion   string
}

// ValidarExigencias rechaza campos repetidos o que no sean de contacto.
func ValidarExigencias(exigencias []ExigenciaCampo) error {
	vistos := map[CampoFicha]bool{}
	for _, e := range exigencias {
		if !e.Campo.EsContacto() || vistos[e.Campo] || !condicionValida(e.Condicion) {
			return ErrCampoInvalido
		}
		vistos[e.Campo] = true
	}
	return nil
}

// ContactoPedido normaliza los datos de contacto de una petición contra lo
// que pide el catálogo. En un alta solo caben campos pedidos, deben venir
// todos los obligatorios y un opcional vacío se ignora. En una rectificación, un valor vacío
// significa retirar el dato: solo se admite si el campo no es obligatorio;
// retirar un dato que el catálogo ya no pide también se admite, porque
// guardar menos siempre es posible.
func ContactoPedido(campos map[CampoFicha]string, exigencias []ExigenciaCampo, alta bool) (map[CampoFicha]string, error) {
	if ValidarExigencias(exigencias) != nil {
		return nil, ErrCampoInvalido
	}
	pedido := map[CampoFicha]ExigenciaCampo{}
	for _, e := range exigencias {
		pedido[e.Campo] = e
	}
	resultado := make(map[CampoFicha]string, len(campos))
	for campo, valor := range campos {
		if !campo.EsContacto() {
			return nil, ErrCampoInvalido
		}
		exigencia, esPedido := pedido[campo]
		if strings.TrimSpace(valor) == "" {
			if exigencia.Obligatorio {
				return nil, ErrCambioInvalido
			}
			if !alta {
				resultado[campo] = ""
			}
			continue
		}
		if !esPedido {
			return nil, ErrCambioInvalido
		}
		normal, err := NormalizarValor(campo, valor)
		if err != nil {
			return nil, err
		}
		resultado[campo] = normal
	}
	if alta {
		for _, e := range exigencias {
			if e.Obligatorio && resultado[e.Campo] == "" {
				return nil, ErrCambioInvalido
			}
		}
	} else if len(resultado) == 0 {
		return nil, ErrCambioInvalido
	}
	return resultado, nil
}

// ValidarMotivo comprueba que el motivo encaja con lo que había. Si ninguno
// de los campos cambiados tenía valor, el motivo es «dato nuevo»; si alguno
// lo tenía, la persona dice si ha cambiado o estaba mal, y ese motivo vale
// para toda la petición (la pantalla envía un cambio por formulario). Retirar
// un dato que no existe no es un cambio. Reescribir el mismo valor no se
// detecta: compararlo exigiría descifrar dentro de la transacción.
func ValidarMotivo(motivo MotivoCambio, cambios map[CampoFicha]string, presentes []CampoFicha) error {
	if !motivo.ValidoParaRectificar() || len(cambios) == 0 {
		return ErrCambioInvalido
	}
	habia := map[CampoFicha]bool{}
	for _, c := range presentes {
		habia[c] = true
	}
	algunoPresente := false
	for campo, valor := range cambios {
		if habia[campo] {
			algunoPresente = true
		} else if valor == "" {
			return ErrCambioInvalido
		}
	}
	if algunoPresente == (motivo == MotivoDatoNuevo) {
		return ErrCambioInvalido
	}
	return nil
}

// CamposOrdenados devuelve las claves en orden estable para serializar.
func CamposOrdenados(m map[CampoFicha]string) []CampoFicha {
	r := make([]CampoFicha, 0, len(m))
	for c := range m {
		r = append(r, c)
	}
	sort.Slice(r, func(i, j int) bool { return r[i] < r[j] })
	return r
}
