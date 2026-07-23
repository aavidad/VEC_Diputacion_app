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
	VerificarPresentacionAutoridadFuenteAnalisis(
		PresentacionAutoridadFuenteAnalisis,
		DesafioAutoridadFuenteAnalisis,
		RolAutoridadFuenteAnalisis,
		time.Time,
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
