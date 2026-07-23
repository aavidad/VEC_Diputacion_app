package ports

import (
	"crypto/ed25519"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// IdentidadAutoridadFuenteAnalisis es la proyección mínima que aplicación
// necesita tras autenticar una autoridad. Crear el valor solo valida su forma:
// la confianza procede exclusivamente del Autenticador inyectado al servicio.
type IdentidadAutoridadFuenteAnalisis struct {
	autoridadRef string
	backendRef   string
	clavePrueba  ed25519.PublicKey
	rol          RolAutoridadFuenteAnalisis
}

// EvidenciaPublicaAutoridadFuenteAnalisis conserva el desafío, la credencial
// institucional y la prueba de posesión que sustentaron una identidad en un
// instante concreto. No contiene claves privadas ni secretos y permite
// revalidar un recibo histórico tras rotar la clave de la operación actual.
type EvidenciaPublicaAutoridadFuenteAnalisis struct {
	desafio      DesafioAutoridadFuenteAnalisis
	presentacion PresentacionAutoridadFuenteAnalisis
	rol          RolAutoridadFuenteAnalisis
	comprobadaEn time.Time
}

// DatosEvidenciaPublicaAutoridadFuenteAnalisis es la representación durable y
// neutral de la evidencia. Solo contiene material público: credencial,
// firmas, desafío, rol e instante de comprobación.
type DatosEvidenciaPublicaAutoridadFuenteAnalisis struct {
	CredencialDatos    DatosCredencialAutoridadFuenteAnalisis
	FirmaInstitucional []byte
	PruebaPosesion     []byte
	Desafio            []byte
	Rol                RolAutoridadFuenteAnalisis
	ComprobadaEn       time.Time
}

func NuevaEvidenciaPublicaAutoridadFuenteAnalisis(
	desafio DesafioAutoridadFuenteAnalisis,
	presentacion PresentacionAutoridadFuenteAnalisis,
	rol RolAutoridadFuenteAnalisis,
	comprobadaEn time.Time,
) (EvidenciaPublicaAutoridadFuenteAnalisis, error) {
	copiaDesafio, errDesafio := copiarDesafioAutoridadFuenteAnalisis(desafio)
	copiaPresentacion, errPresentacion :=
		copiarPresentacionAutoridadFuenteAnalisis(presentacion)
	if errDesafio != nil || errPresentacion != nil || !rol.valida() ||
		!instanteFuenteAnalisisCanonico(comprobadaEn) {
		return EvidenciaPublicaAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return EvidenciaPublicaAutoridadFuenteAnalisis{
		desafio:      copiaDesafio,
		presentacion: copiaPresentacion,
		rol:          rol,
		comprobadaEn: comprobadaEn,
	}, nil
}

func (e EvidenciaPublicaAutoridadFuenteAnalisis) Datos() (
	PresentacionAutoridadFuenteAnalisis,
	DesafioAutoridadFuenteAnalisis,
	RolAutoridadFuenteAnalisis,
	time.Time,
	error,
) {
	presentacion, errPresentacion :=
		copiarPresentacionAutoridadFuenteAnalisis(e.presentacion)
	desafio, errDesafio := copiarDesafioAutoridadFuenteAnalisis(e.desafio)
	if errPresentacion != nil || errDesafio != nil || !e.rol.valida() ||
		!instanteFuenteAnalisisCanonico(e.comprobadaEn) {
		return PresentacionAutoridadFuenteAnalisis{},
			DesafioAutoridadFuenteAnalisis{},
			"",
			time.Time{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return presentacion, desafio, e.rol, e.comprobadaEn, nil
}

func (e EvidenciaPublicaAutoridadFuenteAnalisis) DatosPublicos() (
	DatosEvidenciaPublicaAutoridadFuenteAnalisis,
	error,
) {
	presentacion, desafio, rol, comprobadaEn, err := e.Datos()
	if err != nil || presentacion.credencial.datos == nil {
		return DatosEvidenciaPublicaAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	contenidoDesafio, err := desafio.Bytes()
	if err != nil {
		return DatosEvidenciaPublicaAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	datosCredencial := *presentacion.credencial.datos
	datosCredencial.ClavePruebaEd25519 = append(
		[]byte(nil),
		presentacion.credencial.datos.ClavePruebaEd25519...,
	)
	return DatosEvidenciaPublicaAutoridadFuenteAnalisis{
		CredencialDatos:    datosCredencial,
		FirmaInstitucional: append([]byte(nil), presentacion.credencial.firma...),
		PruebaPosesion:     append([]byte(nil), presentacion.prueba...),
		Desafio:            contenidoDesafio,
		Rol:                rol,
		ComprobadaEn:       comprobadaEn,
	}, nil
}

func RestaurarEvidenciaPublicaAutoridadFuenteAnalisis(
	datos DatosEvidenciaPublicaAutoridadFuenteAnalisis,
) (EvidenciaPublicaAutoridadFuenteAnalisis, error) {
	credencial, err := NuevaCredencialAutoridadFuenteAnalisis(
		datos.CredencialDatos,
		append([]byte(nil), datos.FirmaInstitucional...),
	)
	if err != nil {
		return EvidenciaPublicaAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	presentacion, err := NuevaPresentacionAutoridadFuenteAnalisis(
		credencial,
		append([]byte(nil), datos.PruebaPosesion...),
	)
	if err != nil {
		return EvidenciaPublicaAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	desafio := DesafioAutoridadFuenteAnalisis{
		contenido: append([]byte(nil), datos.Desafio...),
	}
	return NuevaEvidenciaPublicaAutoridadFuenteAnalisis(
		desafio,
		presentacion,
		datos.Rol,
		datos.ComprobadaEn,
	)
}

func (EvidenciaPublicaAutoridadFuenteAnalisis) String() string {
	return "[EVIDENCIA-PUBLICA-AUTORIDAD-FUENTE-ANALISIS-REDACTADA]"
}

func (e EvidenciaPublicaAutoridadFuenteAnalisis) GoString() string {
	return e.String()
}
func (e EvidenciaPublicaAutoridadFuenteAnalisis) Format(
	s fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(s, e.String())
}
func (e EvidenciaPublicaAutoridadFuenteAnalisis) LogValue() slog.Value {
	return slog.StringValue(e.String())
}

func NuevaIdentidadAutoridadFuenteAnalisis(
	autoridadRef string,
	backendRef string,
	clavePruebaEd25519 []byte,
	rol RolAutoridadFuenteAnalisis,
) (IdentidadAutoridadFuenteAnalisis, error) {
	if !domain.ReferenciaOpacaValida(autoridadRef) ||
		!domain.ReferenciaOpacaValida(backendRef) ||
		len(clavePruebaEd25519) != ed25519.PublicKeySize ||
		!rol.valida() {
		return IdentidadAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return IdentidadAutoridadFuenteAnalisis{
		autoridadRef: autoridadRef,
		backendRef:   backendRef,
		clavePrueba: append(
			ed25519.PublicKey(nil),
			clavePruebaEd25519...,
		),
		rol: rol,
	}, nil
}

func (i IdentidadAutoridadFuenteAnalisis) AutoridadRef() string {
	return i.autoridadRef
}

func (i IdentidadAutoridadFuenteAnalisis) BackendRef() string {
	return i.backendRef
}

func (i IdentidadAutoridadFuenteAnalisis) ClavePruebaEd25519() ed25519.PublicKey {
	return append(ed25519.PublicKey(nil), i.clavePrueba...)
}

func (i IdentidadAutoridadFuenteAnalisis) Rol() RolAutoridadFuenteAnalisis {
	return i.rol
}

func (IdentidadAutoridadFuenteAnalisis) String() string {
	return "[IDENTIDAD-AUTORIDAD-FUENTE-ANALISIS-REDACTADA]"
}

func (i IdentidadAutoridadFuenteAnalisis) GoString() string {
	return i.String()
}
func (i IdentidadAutoridadFuenteAnalisis) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, i.String())
}
func (i IdentidadAutoridadFuenteAnalisis) LogValue() slog.Value {
	return slog.StringValue(i.String())
}

// VerificadorPresentacionesAutoridadFuenteAnalisis es la frontera local de
// confianza que valida una presentación ya obtenida por application. No crea
// desafíos ni invoca presentadores; su implementación se fija en composición
// y nunca llega desde una petición de usuario.
type VerificadorPresentacionesAutoridadFuenteAnalisis interface {
	OrganizacionAutoridadFuenteAnalisis() string
	AudienciaAutoridadFuenteAnalisis() string
	VerificarEvidenciaPublicaAutoridadFuenteAnalisis(
		EvidenciaPublicaAutoridadFuenteAnalisis,
	) (IdentidadAutoridadFuenteAnalisis, error)
}

func AutoridadesFuenteAnalisisSeparadas(
	identidades ...IdentidadAutoridadFuenteAnalisis,
) bool {
	internas := make(
		[]identidadAutoridadFuenteAnalisis,
		0,
		len(identidades),
	)
	for _, identidad := range identidades {
		internas = append(internas, identidadAutoridadFuenteAnalisis{
			autoridadRef: identidad.autoridadRef,
			backendRef:   identidad.backendRef,
			clavePrueba: append(
				ed25519.PublicKey(nil),
				identidad.clavePrueba...,
			),
			rol: identidad.rol,
		})
	}
	return autoridadesFuenteAnalisisSeparadas(internas...)
}

func IdentidadesAutoridadFuenteAnalisisIguales(
	primera IdentidadAutoridadFuenteAnalisis,
	segunda IdentidadAutoridadFuenteAnalisis,
) bool {
	return primera.autoridadRef == segunda.autoridadRef &&
		primera.backendRef == segunda.backendRef &&
		primera.rol == segunda.rol &&
		ed25519.PublicKey(primera.clavePrueba).Equal(
			ed25519.PublicKey(segunda.clavePrueba),
		)
}
