package ports

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
)

var (
	// ErrResultadoTransaccionalBaremacionInvalido impide convertir un valor
	// incompleto o fabricado en una afirmacion sobre el resultado del COMMIT.
	ErrResultadoTransaccionalBaremacionInvalido = errors.New("bolsa: resultado transaccional invalido")

	// ErrTransaccionBaremacionNoAplicada solo identifica el desenlace negativo
	// que lleva una prueba autenticada de no aplicacion.
	ErrTransaccionBaremacionNoAplicada = errors.New("bolsa: transaccion no aplicada acreditada")

	// ErrResultadoTransaccionalBaremacionIndeterminado significa que el COMMIT
	// pudo haber surtido efecto. No equivale a rollback ni concede un reintento.
	ErrResultadoTransaccionalBaremacionIndeterminado = errors.New("bolsa: resultado transaccional indeterminado")

	// ErrReconciliacionTransaccionalBaremacionRequerida permite enrutar el fallo
	// a una cola de reconciliacion sin inspeccionar textos de error.
	ErrReconciliacionTransaccionalBaremacionRequerida = errors.New("bolsa: reconciliacion transaccional requerida")

	// ErrSerializacionResultadoTransaccionalBaremacionProhibida evita que las
	// referencias y sellos probatorios crucen accidentalmente logs o DTO.
	ErrSerializacionResultadoTransaccionalBaremacionProhibida = errors.New("bolsa: serializacion generica de resultado transaccional prohibida")

	// ErrVerificadorNoAplicacionBaremacionRequerido impide convertir una mera
	// declaracion con forma valida en una prueba de no aplicacion.
	ErrVerificadorNoAplicacionBaremacionRequerido = errors.New("bolsa: verificador de no aplicacion requerido")

	// ErrEvidenciaNoAplicacionBaremacionNoVerificada no envuelve el fallo
	// tecnico del verificador para evitar conservar datos o credenciales.
	ErrEvidenciaNoAplicacionBaremacionNoVerificada = errors.New("bolsa: evidencia de no aplicacion no verificada")

	ErrContextoVerificacionNoAplicacionBaremacionInvalido = errors.New("bolsa: contexto de verificacion de no aplicacion invalido")
)

const (
	prefijoReferenciaOperacionBaremacion = "brc1_"
	prefijoReferenciaEvidenciaBaremacion = "bre1_"
	longitudReferenciaOpacaBaremacion    = 32
)

const (
	mensajeIdentificadorOperacionBaremacion = "[IDENTIFICADOR-RECONCILIACION-OCULTO]"
	mensajeEvidenciaNoAplicacionBaremacion  = "[EVIDENCIA-NO-APLICACION-OCULTA]"
	mensajePruebaNoAplicacionBaremacion     = "[PRUEBA-VERIFICADA-NO-APLICACION-OCULTA]"
	mensajeResultadoNoAplicadoBaremacion    = "bolsa: transaccion no aplicada acreditada"
	mensajeResultadoIndeterminadoBaremacion = "bolsa: resultado transaccional indeterminado; reconciliacion requerida"
)

// IdentificadorOperacionTransaccionalBaremacion enlaza la referencia opaca
// creada antes de la transaccion con un indice HMAC estable. No es una
// capacidad de acceso y no debe derivarse de DNI, correo, expediente ni otras
// referencias de negocio. El adaptador debe generar la referencia con un CSPRNG.
//
// Sus campos son privados para impedir que un DTO generico los publique. Solo
// DatosReconciliacion los abre de forma explicita al adaptador reconciliador.
type IdentificadorOperacionTransaccionalBaremacion struct {
	referenciaOpaca     string
	indiceOperacionHMAC string
}

// NuevoIdentificadorOperacionTransaccionalBaremacion valida la forma opaca de
// la referencia y el indice autenticado. La validacion de forma no sustituye la
// obligacion del adaptador de usar aleatoriedad criptografica.
func NuevoIdentificadorOperacionTransaccionalBaremacion(
	referenciaOpaca string,
	indiceOperacionHMAC string,
) (IdentificadorOperacionTransaccionalBaremacion, error) {
	identificador := IdentificadorOperacionTransaccionalBaremacion{
		referenciaOpaca:     referenciaOpaca,
		indiceOperacionHMAC: indiceOperacionHMAC,
	}
	if err := identificador.Validar(); err != nil {
		return IdentificadorOperacionTransaccionalBaremacion{}, err
	}
	return identificador, nil
}

func (i IdentificadorOperacionTransaccionalBaremacion) Validar() error {
	if !referenciaOpacaBaremacionValida(i.referenciaOpaca, prefijoReferenciaOperacionBaremacion) ||
		!huellaHMACSHA256Valida(i.indiceOperacionHMAC) {
		return ErrResultadoTransaccionalBaremacionInvalido
	}
	return nil
}

func (i IdentificadorOperacionTransaccionalBaremacion) Clonar() (IdentificadorOperacionTransaccionalBaremacion, error) {
	if err := i.Validar(); err != nil {
		return IdentificadorOperacionTransaccionalBaremacion{}, err
	}
	return i, nil
}

// CoincideExactamenteCon comprueba las dos partes protegidas del identificador
// sin abrirlas al llamador. Un valor cero, alterado o de otro esquema nunca
// coincide, aunque comparta una de las dos partes.
func (i IdentificadorOperacionTransaccionalBaremacion) CoincideExactamenteCon(
	otro IdentificadorOperacionTransaccionalBaremacion,
) bool {
	if i.Validar() != nil || otro.Validar() != nil {
		return false
	}
	return textoIgualConstanteBaremacion(i.referenciaOpaca, otro.referenciaOpaca) &&
		textoIgualConstanteBaremacion(i.indiceOperacionHMAC, otro.indiceOperacionHMAC)
}

// DatosReconciliacion es la unica apertura deliberada del identificador. El
// llamador debe tratar los valores como material probatorio, no como texto de
// observabilidad.
func (i IdentificadorOperacionTransaccionalBaremacion) DatosReconciliacion() (
	referenciaOpaca string,
	indiceOperacionHMAC string,
	err error,
) {
	if err := i.Validar(); err != nil {
		return "", "", err
	}
	return i.referenciaOpaca, i.indiceOperacionHMAC, nil
}

func (IdentificadorOperacionTransaccionalBaremacion) String() string {
	return mensajeIdentificadorOperacionBaremacion
}
func (IdentificadorOperacionTransaccionalBaremacion) GoString() string {
	return "ports.IdentificadorOperacionTransaccionalBaremacion{[OCULTO]}"
}
func (i IdentificadorOperacionTransaccionalBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, i.String())
}
func (IdentificadorOperacionTransaccionalBaremacion) LogValue() slog.Value {
	return slog.StringValue(mensajeIdentificadorOperacionBaremacion)
}
func (IdentificadorOperacionTransaccionalBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*IdentificadorOperacionTransaccionalBaremacion) UnmarshalJSON([]byte) error {
	return ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (IdentificadorOperacionTransaccionalBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*IdentificadorOperacionTransaccionalBaremacion) UnmarshalText([]byte) error {
	return ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (IdentificadorOperacionTransaccionalBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*IdentificadorOperacionTransaccionalBaremacion) UnmarshalBinary([]byte) error {
	return ErrSerializacionResultadoTransaccionalBaremacionProhibida
}

// EvidenciaNoAplicacionBaremacion es material candidato: su forma valida no
// demuestra por si sola que la operacion no se aplico. El sello HMAC debe
// cubrir, mediante un esquema canonico versionado, el identificador, la
// consulta autoritativa y la conclusion "no aplicada".
type EvidenciaNoAplicacionBaremacion struct {
	identificador       IdentificadorOperacionTransaccionalBaremacion
	referenciaEvidencia string
	selloEvidenciaHMAC  string
}

func NuevaEvidenciaNoAplicacionBaremacion(
	identificador IdentificadorOperacionTransaccionalBaremacion,
	referenciaEvidencia string,
	selloEvidenciaHMAC string,
) (EvidenciaNoAplicacionBaremacion, error) {
	evidencia := EvidenciaNoAplicacionBaremacion{
		identificador:       identificador,
		referenciaEvidencia: referenciaEvidencia,
		selloEvidenciaHMAC:  selloEvidenciaHMAC,
	}
	if err := evidencia.Validar(); err != nil {
		return EvidenciaNoAplicacionBaremacion{}, err
	}
	return evidencia, nil
}

func (e EvidenciaNoAplicacionBaremacion) Validar() error {
	if e.identificador.Validar() != nil ||
		!referenciaOpacaBaremacionValida(e.referenciaEvidencia, prefijoReferenciaEvidenciaBaremacion) ||
		!huellaHMACSHA256Valida(e.selloEvidenciaHMAC) {
		return ErrResultadoTransaccionalBaremacionInvalido
	}
	return nil
}

func (e EvidenciaNoAplicacionBaremacion) Clonar() (EvidenciaNoAplicacionBaremacion, error) {
	if err := e.Validar(); err != nil {
		return EvidenciaNoAplicacionBaremacion{}, err
	}
	return e, nil
}

// DatosVerificacion abre la evidencia solo para su verificador explicito.
func (e EvidenciaNoAplicacionBaremacion) DatosVerificacion() (
	identificador IdentificadorOperacionTransaccionalBaremacion,
	referenciaEvidencia string,
	selloEvidenciaHMAC string,
	err error,
) {
	if err := e.Validar(); err != nil {
		return IdentificadorOperacionTransaccionalBaremacion{}, "", "", err
	}
	return e.identificador, e.referenciaEvidencia, e.selloEvidenciaHMAC, nil
}

func (EvidenciaNoAplicacionBaremacion) String() string { return mensajeEvidenciaNoAplicacionBaremacion }
func (EvidenciaNoAplicacionBaremacion) GoString() string {
	return "ports.EvidenciaNoAplicacionBaremacion{[OCULTA]}"
}
func (e EvidenciaNoAplicacionBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.String())
}
func (EvidenciaNoAplicacionBaremacion) LogValue() slog.Value {
	return slog.StringValue(mensajeEvidenciaNoAplicacionBaremacion)
}
func (EvidenciaNoAplicacionBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*EvidenciaNoAplicacionBaremacion) UnmarshalJSON([]byte) error {
	return ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (EvidenciaNoAplicacionBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*EvidenciaNoAplicacionBaremacion) UnmarshalText([]byte) error {
	return ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (EvidenciaNoAplicacionBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*EvidenciaNoAplicacionBaremacion) UnmarshalBinary([]byte) error {
	return ErrSerializacionResultadoTransaccionalBaremacionProhibida
}

// VerificadorNoAplicacionBaremacion es la frontera de confianza que comprueba
// el sello y contrasta la consulta con la fuente transaccional autoritativa. Un
// error, ausencia o implementacion nula nunca produce una conclusion negativa.
type VerificadorNoAplicacionBaremacion interface {
	VerificarNoAplicacionBaremacion(context.Context, EvidenciaNoAplicacionBaremacion) error
}

// PruebaNoAplicacionVerificadaBaremacion es una capacidad opaca. Sus campos
// privados impiden construirla fuera de este paquete sin pasar por Verificar.
type PruebaNoAplicacionVerificadaBaremacion struct {
	evidencia EvidenciaNoAplicacionBaremacion
}

// VerificarEvidenciaNoAplicacionBaremacion descarta deliberadamente la causa
// tecnica del verificador. La causa puede registrarse en su propio limite de
// seguridad, pero nunca queda incorporada al resultado transaccional.
func VerificarEvidenciaNoAplicacionBaremacion(
	ctx context.Context,
	verificador VerificadorNoAplicacionBaremacion,
	evidencia EvidenciaNoAplicacionBaremacion,
) (PruebaNoAplicacionVerificadaBaremacion, error) {
	if err := evidencia.Validar(); err != nil {
		return PruebaNoAplicacionVerificadaBaremacion{}, err
	}
	if interfazNulaResultadoTransaccional(ctx) {
		return PruebaNoAplicacionVerificadaBaremacion{}, ErrContextoVerificacionNoAplicacionBaremacionInvalido
	}
	if err := ctx.Err(); err != nil {
		return PruebaNoAplicacionVerificadaBaremacion{},
			errors.Join(ErrEvidenciaNoAplicacionBaremacionNoVerificada, err)
	}
	if interfazNulaResultadoTransaccional(verificador) {
		return PruebaNoAplicacionVerificadaBaremacion{}, ErrVerificadorNoAplicacionBaremacionRequerido
	}
	if err := verificador.VerificarNoAplicacionBaremacion(ctx, evidencia); err != nil {
		if errors.Is(err, context.Canceled) {
			return PruebaNoAplicacionVerificadaBaremacion{},
				errors.Join(ErrEvidenciaNoAplicacionBaremacionNoVerificada, context.Canceled)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return PruebaNoAplicacionVerificadaBaremacion{},
				errors.Join(ErrEvidenciaNoAplicacionBaremacionNoVerificada, context.DeadlineExceeded)
		}
		return PruebaNoAplicacionVerificadaBaremacion{}, ErrEvidenciaNoAplicacionBaremacionNoVerificada
	}
	return PruebaNoAplicacionVerificadaBaremacion{evidencia: evidencia}, nil
}

func (p PruebaNoAplicacionVerificadaBaremacion) Validar() error {
	if p.evidencia.Validar() != nil {
		return ErrResultadoTransaccionalBaremacionInvalido
	}
	return nil
}

func (p PruebaNoAplicacionVerificadaBaremacion) Clonar() (PruebaNoAplicacionVerificadaBaremacion, error) {
	if err := p.Validar(); err != nil {
		return PruebaNoAplicacionVerificadaBaremacion{}, err
	}
	return p, nil
}

func (p PruebaNoAplicacionVerificadaBaremacion) Evidencia() (EvidenciaNoAplicacionBaremacion, error) {
	if err := p.Validar(); err != nil {
		return EvidenciaNoAplicacionBaremacion{}, err
	}
	return p.evidencia.Clonar()
}

func (PruebaNoAplicacionVerificadaBaremacion) String() string {
	return mensajePruebaNoAplicacionBaremacion
}
func (PruebaNoAplicacionVerificadaBaremacion) GoString() string {
	return "ports.PruebaNoAplicacionVerificadaBaremacion{[OCULTA]}"
}
func (p PruebaNoAplicacionVerificadaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (PruebaNoAplicacionVerificadaBaremacion) LogValue() slog.Value {
	return slog.StringValue(mensajePruebaNoAplicacionBaremacion)
}
func (PruebaNoAplicacionVerificadaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*PruebaNoAplicacionVerificadaBaremacion) UnmarshalJSON([]byte) error {
	return ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (PruebaNoAplicacionVerificadaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*PruebaNoAplicacionVerificadaBaremacion) UnmarshalText([]byte) error {
	return ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (PruebaNoAplicacionVerificadaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*PruebaNoAplicacionVerificadaBaremacion) UnmarshalBinary([]byte) error {
	return ErrSerializacionResultadoTransaccionalBaremacionProhibida
}

// EstadoResultadoTransaccionalBaremacion es un catalogo cerrado. No existe un
// estado "aplicada": una operacion aplicada debe recuperarse como el resultado
// ordinario completo, no fabricarse desde este contrato de error.
type EstadoResultadoTransaccionalBaremacion string

const (
	EstadoResultadoTransaccionalNoAplicadaVerificada  EstadoResultadoTransaccionalBaremacion = "no_aplicada_verificada"
	EstadoResultadoTransaccionalPodriaHaberseAplicado EstadoResultadoTransaccionalBaremacion = "podria_haberse_aplicado"
)

func (e EstadoResultadoTransaccionalBaremacion) valido() bool {
	return e == EstadoResultadoTransaccionalNoAplicadaVerificada ||
		e == EstadoResultadoTransaccionalPodriaHaberseAplicado
}

// ErrorResultadoTransaccionalBaremacion representa exclusivamente un fracaso
// cuyo efecto es negativo acreditado o desconocido. No conserva causa tecnica,
// contexto, identidad, sesion, autorizacion ni capacidades temporales.
//
// Un valor cero o alterado se interpreta en cerrado: podria haberse aplicado y
// requiere reconciliacion. Este error nunca concede permiso para reintentar.
type ErrorResultadoTransaccionalBaremacion struct {
	// Los campos exportados son envoltorios protegidos, no cadenas. Asi incluso
	// una desreferenciacion deliberada usa sus formateadores seguros. Validar
	// comprueba siempre la coherencia entre los tres.
	IdentificadorOperacion       IdentificadorOperacionTransaccionalBaremacion
	EstadoAplicacion             EstadoResultadoTransaccionalBaremacion
	PruebaNoAplicacionVerificada PruebaNoAplicacionVerificadaBaremacion
}

func NuevoErrorTransaccionBaremacionNoAplicada(
	prueba PruebaNoAplicacionVerificadaBaremacion,
) (*ErrorResultadoTransaccionalBaremacion, error) {
	if err := prueba.Validar(); err != nil {
		return nil, err
	}
	resultado := &ErrorResultadoTransaccionalBaremacion{
		IdentificadorOperacion:       prueba.evidencia.identificador,
		EstadoAplicacion:             EstadoResultadoTransaccionalNoAplicadaVerificada,
		PruebaNoAplicacionVerificada: prueba,
	}
	if err := resultado.Validar(); err != nil {
		return nil, err
	}
	return resultado, nil
}

func NuevoErrorResultadoTransaccionalIndeterminadoBaremacion(
	identificador IdentificadorOperacionTransaccionalBaremacion,
) (*ErrorResultadoTransaccionalBaremacion, error) {
	if err := identificador.Validar(); err != nil {
		return nil, err
	}
	resultado := &ErrorResultadoTransaccionalBaremacion{
		IdentificadorOperacion: identificador,
		EstadoAplicacion:       EstadoResultadoTransaccionalPodriaHaberseAplicado,
	}
	if err := resultado.Validar(); err != nil {
		return nil, err
	}
	return resultado, nil
}

func (e *ErrorResultadoTransaccionalBaremacion) Validar() error {
	if e == nil || !e.EstadoAplicacion.valido() || e.IdentificadorOperacion.Validar() != nil {
		return ErrResultadoTransaccionalBaremacionInvalido
	}
	switch e.EstadoAplicacion {
	case EstadoResultadoTransaccionalNoAplicadaVerificada:
		if e.PruebaNoAplicacionVerificada.Validar() != nil ||
			e.PruebaNoAplicacionVerificada.evidencia.identificador != e.IdentificadorOperacion {
			return ErrResultadoTransaccionalBaremacionInvalido
		}
	case EstadoResultadoTransaccionalPodriaHaberseAplicado:
		if e.PruebaNoAplicacionVerificada != (PruebaNoAplicacionVerificadaBaremacion{}) {
			return ErrResultadoTransaccionalBaremacionInvalido
		}
	default:
		return ErrResultadoTransaccionalBaremacionInvalido
	}
	return nil
}

func (e *ErrorResultadoTransaccionalBaremacion) Clonar() (*ErrorResultadoTransaccionalBaremacion, error) {
	if err := e.Validar(); err != nil {
		return nil, err
	}
	clon := *e
	return &clon, nil
}

// Identificador devuelve una copia protegida para que el reconciliador pueda
// localizar la operacion sin acceder a ningun dato personal.
func (e *ErrorResultadoTransaccionalBaremacion) Identificador() (
	IdentificadorOperacionTransaccionalBaremacion,
	error,
) {
	if err := e.Validar(); err != nil {
		return IdentificadorOperacionTransaccionalBaremacion{}, err
	}
	return e.IdentificadorOperacion.Clonar()
}

func (e *ErrorResultadoTransaccionalBaremacion) Estado() EstadoResultadoTransaccionalBaremacion {
	if e.Validar() != nil {
		return EstadoResultadoTransaccionalPodriaHaberseAplicado
	}
	return e.EstadoAplicacion
}

// NoAplicadaVerificada solo devuelve true ante una prueba validada por la
// frontera de confianza y enlazada a
// la misma operacion. Ausencia, manipulacion o estado desconocido devuelven
// false.
func (e *ErrorResultadoTransaccionalBaremacion) NoAplicadaVerificada() bool {
	return e != nil && e.Validar() == nil && e.EstadoAplicacion == EstadoResultadoTransaccionalNoAplicadaVerificada
}

// RequiereReconciliacion falla en cerrado: cualquier valor invalido se trata
// como posiblemente aplicado.
func (e *ErrorResultadoTransaccionalBaremacion) RequiereReconciliacion() bool {
	return e == nil || e.Validar() != nil || e.EstadoAplicacion != EstadoResultadoTransaccionalNoAplicadaVerificada
}

func (e *ErrorResultadoTransaccionalBaremacion) PruebaNoAplicacion() (
	PruebaNoAplicacionVerificadaBaremacion,
	bool,
) {
	if !e.NoAplicadaVerificada() {
		return PruebaNoAplicacionVerificadaBaremacion{}, false
	}
	return e.PruebaNoAplicacionVerificada, true
}

func (e *ErrorResultadoTransaccionalBaremacion) Error() string {
	if e != nil && e.NoAplicadaVerificada() {
		return mensajeResultadoNoAplicadoBaremacion
	}
	return mensajeResultadoIndeterminadoBaremacion
}

func (e *ErrorResultadoTransaccionalBaremacion) String() string { return e.Error() }
func (e *ErrorResultadoTransaccionalBaremacion) GoString() string {
	if e != nil && e.NoAplicadaVerificada() {
		return "ports.ErrorResultadoTransaccionalBaremacion{[NO-APLICADA-VERIFICADA]}"
	}
	return "ports.ErrorResultadoTransaccionalBaremacion{[INDETERMINADO]}"
}
func (e *ErrorResultadoTransaccionalBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.Error())
}
func (e *ErrorResultadoTransaccionalBaremacion) LogValue() slog.Value {
	return slog.StringValue(e.Error())
}

// Is ofrece clasificacion estable sin desenvolver causas tecnicas que pudieran
// contener datos o credenciales. Un valor invalido conserva la clasificacion
// mas restrictiva: indeterminado y pendiente de reconciliacion.
func (e *ErrorResultadoTransaccionalBaremacion) Is(objetivo error) bool {
	valido := e != nil && e.Validar() == nil
	switch objetivo {
	case ErrTransaccionBaremacionNoAplicada:
		return valido && e.EstadoAplicacion == EstadoResultadoTransaccionalNoAplicadaVerificada
	case ErrResultadoTransaccionalBaremacionIndeterminado,
		ErrReconciliacionTransaccionalBaremacionRequerida:
		return !valido || e.EstadoAplicacion == EstadoResultadoTransaccionalPodriaHaberseAplicado
	case ErrResultadoTransaccionalBaremacionInvalido:
		return !valido
	default:
		return false
	}
}

func (*ErrorResultadoTransaccionalBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*ErrorResultadoTransaccionalBaremacion) UnmarshalJSON([]byte) error {
	return ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*ErrorResultadoTransaccionalBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*ErrorResultadoTransaccionalBaremacion) UnmarshalText([]byte) error {
	return ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*ErrorResultadoTransaccionalBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionResultadoTransaccionalBaremacionProhibida
}
func (*ErrorResultadoTransaccionalBaremacion) UnmarshalBinary([]byte) error {
	return ErrSerializacionResultadoTransaccionalBaremacionProhibida
}

func referenciaOpacaBaremacionValida(valor, prefijo string) bool {
	if !strings.HasPrefix(valor, prefijo) || valor != strings.TrimSpace(valor) {
		return false
	}
	codificada := strings.TrimPrefix(valor, prefijo)
	decodificada, err := base64.RawURLEncoding.DecodeString(codificada)
	return err == nil && len(decodificada) == longitudReferenciaOpacaBaremacion &&
		base64.RawURLEncoding.EncodeToString(decodificada) == codificada
}

func interfazNulaResultadoTransaccional(valor any) bool {
	if valor == nil {
		return true
	}
	reflejado := reflect.ValueOf(valor)
	switch reflejado.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejado.IsNil()
	default:
		return false
	}
}
