package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrFuenteContextoActorNoDisponible             = errors.New("vec: fuente de contexto de actor no disponible")
	ErrResolutorRegistroContextoActorNoDisponible  = errors.New("vec: resolutor y registro de contexto de actor no disponible")
	ErrGeneradorOperacionContextoActorNoDisponible = errors.New("vec: generador de operacion de contexto de actor no disponible")
	ErrSolicitudRegistroContextoActorV2Invalida    = errors.New("vec: solicitud de registro de contexto de actor v2 invalida")
	ErrConfirmacionRegistroContextoActorV2Invalida = errors.New("vec: confirmacion de registro de contexto de actor v2 invalida")
)

// VentanaMaximaFrescuraContextoActorV2 acota el tiempo entre el instante
// solicitado y la confirmacion durable. No es una gracia de vigencia: tanto el
// adaptador como el servicio deben exigir ademas que perfil y referencias sigan
// activos en el instante autoritativo de su comprobacion.
const VentanaMaximaFrescuraContextoActorV2 = 5 * time.Second

const (
	longitudMinimaTokenContextoActorV2 = 24
	longitudMaximaTokenContextoActorV2 = 128
)

// GeneradorOperacionContextoActorV2 crea una referencia nueva antes de entrar
// en la operacion durable. Cada token debe proceder de un CSPRNG, aportar como
// minimo 144 bits de entropia y pertenecer al espacio oca_. La misma invocacion
// logica conserva el token al reconciliar un COMMIT ambiguo; una invocacion
// nueva nunca reutiliza el anterior.
type GeneradorOperacionContextoActorV2 interface {
	NuevaReferenciaOperacionContextoActorV2(context.Context) (string, error)
}

// SolicitudResolucionRegistroContextoActorV2 fija la identidad de una unica
// invocacion durable. SolicitadoEn es una observacion local para acotar
// frescura y deriva de reloj; no es el instante autoritativo que debe guardar
// el adaptador.
type SolicitudResolucionRegistroContextoActorV2 struct {
	OperacionRef string
	Contexto     domain.SolicitudContextoActor
	SolicitadoEn time.Time
}

func (s SolicitudResolucionRegistroContextoActorV2) Validar() error {
	if !referenciaRegistroContextoActorV2Valida(s.OperacionRef, "oca_") ||
		s.Contexto.Validar() != nil || !instanteRegistroContextoActorV2Canonico(s.SolicitadoEn) {
		return ErrSolicitudRegistroContextoActorV2Invalida
	}
	return nil
}

// ConfirmacionRegistroContextoActorV2 es el recibo exacto que el adaptador
// recupera de la persistencia y devuelve unicamente despues de un COMMIT
// confirmado o reconciliado. Las dos representaciones canonicas, sus huellas y
// la autoridad efectiva son los valores almacenados, no una reconstruccion
// oportunista posterior al commit.
type ConfirmacionRegistroContextoActorV2 struct {
	OperacionRef                      string
	RegistroContextoRef               string
	Contexto                          domain.ContextoActor
	RepresentacionCanonica            []byte
	HuellaSHA256                      string
	ManifiestoProcedenciaCanonico     []byte
	ManifiestoProcedenciaHuellaSHA256 string
	AutoridadEfectiva                 domain.AutoridadProcedenciaContextoActorV1
	ResueltoEnAutoritativo            time.Time
}

// ValidarPara comprueba el eco exacto de la solicitud y liga el recibo a los
// mismos bytes que comprometen Contexto y su procedencia. No demuestra por si
// solo que exista la fila: esa autoridad pertenece al puerto durable y a su
// reconciliacion.
func (c ConfirmacionRegistroContextoActorV2) ValidarPara(
	solicitud SolicitudResolucionRegistroContextoActorV2,
) error {
	if solicitud.Validar() != nil || c.OperacionRef != solicitud.OperacionRef ||
		!referenciaRegistroContextoActorV2Valida(c.RegistroContextoRef, "rca_") ||
		c.Contexto.Validar() != nil ||
		!instanteRegistroContextoActorV2Canonico(c.ResueltoEnAutoritativo) ||
		!c.Contexto.ResueltoEn.Equal(c.ResueltoEnAutoritativo) ||
		c.Contexto.Instantanea.CuentaRef != solicitud.Contexto.Cuenta.CuentaRef ||
		c.Contexto.PerfilActivoRef != solicitud.Contexto.PerfilActivoRef ||
		c.Contexto.Principal.AuthMethod != solicitud.Contexto.Cuenta.Metodo ||
		c.Contexto.Principal.AuthAssurance != solicitud.Contexto.Cuenta.Garantia {
		return ErrConfirmacionRegistroContextoActorV2Invalida
	}
	representacion, err := c.Contexto.RepresentacionCanonicaVinculadaV2()
	if err != nil || len(c.RepresentacionCanonica) == 0 ||
		!bytes.Equal(representacion, c.RepresentacionCanonica) {
		return ErrConfirmacionRegistroContextoActorV2Invalida
	}
	suma := sha256.Sum256(c.RepresentacionCanonica)
	if c.HuellaSHA256 != hex.EncodeToString(suma[:]) {
		return ErrConfirmacionRegistroContextoActorV2Invalida
	}
	manifiesto, err := domain.RehidratarManifiestoProcedenciaContextoActorV1(
		c.ManifiestoProcedenciaCanonico,
	)
	if err != nil || manifiesto.AutoridadEfectiva != c.AutoridadEfectiva ||
		manifiesto.ValidarParaContexto(c.Contexto) != nil {
		return ErrConfirmacionRegistroContextoActorV2Invalida
	}
	huellaManifiesto, err := domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
		c.ManifiestoProcedenciaCanonico,
	)
	if err != nil || huellaManifiesto != c.ManifiestoProcedenciaHuellaSHA256 {
		return ErrConfirmacionRegistroContextoActorV2Invalida
	}
	return nil
}

// ValidarParaProductiva exige ademas que todas las versiones procedan de la
// autoridad maestra acreditada. Una evidencia no autoritativa puede cotejarse
// estructuralmente, pero nunca convertirse en ContextoActor productivo.
func (c ConfirmacionRegistroContextoActorV2) ValidarParaProductiva(
	solicitud SolicitudResolucionRegistroContextoActorV2,
) error {
	if c.ValidarPara(solicitud) != nil ||
		c.AutoridadEfectiva != domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1 {
		return ErrConfirmacionRegistroContextoActorV2Invalida
	}
	return nil
}

// ResolutorRegistroContextoActorV2 es la unica frontera productiva para
// resolver y dejar registrada una capacidad de actor. La implementacion debe
// ejecutar ambas operaciones en una sola transaccion: no puede leer primero y
// registrar despues ni devolver una capacidad si el registro durable falla.
//
// La operacion, la cuenta, el perfil, el metodo y la garantia son entradas
// exactas. La implementacion no los normaliza, completa ni sustituye.
// SolicitadoEn no reemplaza al reloj autoritativo del adaptador. Tras
// adquirir todos los bloqueos, la implementacion debe releer cuenta, perfil,
// persona y vinculos; obtener clock_timestamp() despues de esos bloqueos;
// exigir que no sea anterior a SolicitadoEn ni lo supere en la ventana maxima;
// comprobar toda la vigencia en ese instante; y solo entonces registrar y
// confirmar en la misma transaccion. Antes del commit debe conservar los bytes
// de RepresentacionCanonicaVinculadaV2 y su huella SHA-256, asi como el
// manifiesto canonico de procedencia, su huella y autoridad efectiva, ligados a
// todas las versiones resueltas. Una espera que alcance caducidad o exceda la
// ventana debe abortar.
//
// operacion_ref es clave idempotente. Ante un resultado de COMMIT ambiguo, el
// adaptador consulta de nuevo por esa referencia y solo devuelve la fila si la
// solicitud, el recibo, los bytes y la huella coinciden exactamente. Una
// colision con otro contenido falla cerrada. Si la ausencia queda confirmada,
// la misma invocacion puede reintentarse con la misma referencia; nunca con una
// nueva. RegistroContextoRef pertenece a un espacio CSPRNG rca_ independiente.
type ResolutorRegistroContextoActorV2 interface {
	ResolverYRegistrarContextoActorV2(
		context.Context,
		SolicitudResolucionRegistroContextoActorV2,
	) (ConfirmacionRegistroContextoActorV2, error)
}

func referenciaRegistroContextoActorV2Valida(valor, prefijo string) bool {
	if len(valor) < len(prefijo)+longitudMinimaTokenContextoActorV2 ||
		len(valor) > len(prefijo)+longitudMaximaTokenContextoActorV2 ||
		len(prefijo) == 0 || len(valor) <= len(prefijo) || valor[:len(prefijo)] != prefijo {
		return false
	}
	for indice := len(prefijo); indice < len(valor); indice++ {
		caracter := valor[indice]
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= 'A' && caracter <= 'Z') ||
			(caracter >= '0' && caracter <= '9') || caracter == '_' || caracter == '-' {
			continue
		}
		return false
	}
	return true
}

func instanteRegistroContextoActorV2Canonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 && instante.Nanosecond()%1_000 == 0
}

// FuenteContextoActor es un puerto heredado para pruebas y migracion. No es una
// frontera productiva porque separa la lectura del registro durable.
//
// Devuelve todas las instantaneas que coincidan exactamente
// con cuenta y perfil. No debe usar LIMIT 1, precedencia ni perfil por defecto:
// el servicio de aplicacion es quien exige una coincidencia unica.
//
// La implementacion devuelve copias defensivas y nunca consulta por DNI, nombre,
// correo ni otro dato personal. Cuenta y referencias son identificadores opacos.
type FuenteContextoActor interface {
	BuscarInstantaneasContextoActor(
		context.Context,
		domain.SolicitudContextoActor,
	) ([]domain.InstantaneaContextoActor, error)
}
