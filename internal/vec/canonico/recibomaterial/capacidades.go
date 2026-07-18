package recibomaterial

import (
	"crypto/sha256"
	"crypto/subtle"
)

// DatosSolicitudAtestacion conserva el mensaje exacto y su compromiso.
type DatosSolicitudAtestacion struct {
	Dominio string
	Mensaje []byte
	Huella  [sha256.Size]byte
}

// PrepararSolicitudAtestacion copia el mensaje antes de calcular su huella.
func PrepararSolicitudAtestacion(dominio string, mensaje []byte) (DatosSolicitudAtestacion, error) {
	if !DominioAtestacionValido(dominio) || len(mensaje) == 0 {
		return DatosSolicitudAtestacion{}, ErrAtestacionNoValida
	}
	copia := append([]byte(nil), mensaje...)
	return DatosSolicitudAtestacion{Dominio: dominio, Mensaje: copia, Huella: sha256.Sum256(copia)}, nil
}

// RevelarSolicitudAtestacion valida y devuelve siempre una copia defensiva.
func RevelarSolicitudAtestacion(s DatosSolicitudAtestacion) (DatosSolicitudAtestacion, error) {
	if !SolicitudAtestacionValida(s.Dominio, s.Mensaje, s.Huella) {
		return DatosSolicitudAtestacion{}, ErrAtestacionNoValida
	}
	s.Mensaje = append([]byte(nil), s.Mensaje...)
	return s, nil
}

// DatosAtestacion representa el resultado criptografico ligado al mensaje.
type DatosAtestacion struct {
	Algoritmo    string
	ClaveRef     string
	ClaveVersion uint32
	Dominio      string
	Huella       [sha256.Size]byte
	Codigo       []byte
}

func NuevaAtestacion(s DatosSolicitudAtestacion, a DatosAtestacion) (DatosAtestacion, error) {
	if !AtestacionValida(s.Dominio, s.Mensaje, s.Huella, a.Algoritmo, a.ClaveRef,
		a.ClaveVersion, a.Dominio, a.Huella, a.Codigo) {
		return DatosAtestacion{}, ErrAtestacionNoValida
	}
	a.Codigo = append([]byte(nil), a.Codigo...)
	return a, nil
}

// RevelarVerificacionAtestacion valida ambas capacidades y copia sus bytes.
func RevelarVerificacionAtestacion(
	s DatosSolicitudAtestacion,
	a DatosAtestacion,
) (DatosSolicitudAtestacion, DatosAtestacion, error) {
	if !AtestacionValida(s.Dominio, s.Mensaje, s.Huella, a.Algoritmo, a.ClaveRef,
		a.ClaveVersion, a.Dominio, a.Huella, a.Codigo) {
		return DatosSolicitudAtestacion{}, DatosAtestacion{}, ErrAtestacionNoValida
	}
	s.Mensaje = append([]byte(nil), s.Mensaje...)
	a.Codigo = append([]byte(nil), a.Codigo...)
	return s, a, nil
}

// SolicitudPlanValida coteja el compromiso almacenado con el vinculo exacto.
func SolicitudPlanValida(v VinculoPlan, huella [sha256.Size]byte) bool {
	esperada, err := HuellaVinculoPlan(v)
	return err == nil && huella != ([sha256.Size]byte{}) &&
		subtle.ConstantTimeCompare(huella[:], esperada[:]) == 1
}

// ResultadoPlanValido impide sustituir o reutilizar el vinculo como plan.
func ResultadoPlanValido(
	v VinculoPlan,
	huellaVinculoSolicitud, huellaPlan, huellaVinculoResultado [sha256.Size]byte,
) bool {
	return SolicitudPlanValida(v, huellaVinculoSolicitud) &&
		ResultadoLigado(huellaVinculoSolicitud, huellaPlan, huellaVinculoResultado)
}

// DatosPerfilPublicado es la apertura minima hacia el catalogo homologado.
type DatosPerfilPublicado struct {
	Referencia       string
	Version          uint32
	ConectorLogicoID string
	Huella           [sha256.Size]byte
	Canonico         []byte
}

func PerfilPublicadoValido(p DatosPerfilPublicado) bool {
	if !AliasLogicoValido(p.Referencia, 512) || p.Version == 0 ||
		!AliasLogicoValido(p.ConectorLogicoID, 128) ||
		p.Huella == ([sha256.Size]byte{}) || len(p.Canonico) == 0 {
		return false
	}
	esperada := sha256.Sum256(p.Canonico)
	return subtle.ConstantTimeCompare(p.Huella[:], esperada[:]) == 1
}

func RevelarPerfilPublicado(p DatosPerfilPublicado) (DatosPerfilPublicado, error) {
	if !PerfilPublicadoValido(p) {
		return DatosPerfilPublicado{}, ErrReciboNoValido
	}
	p.Canonico = append([]byte(nil), p.Canonico...)
	return p, nil
}

// PerfilSelladoValido coteja hechos, huella y atestacion en una sola regla.
func PerfilSelladoValido(p Perfil, huella [sha256.Size]byte, a DatosAtestacion) bool {
	canonico, err := CanonicoPerfil(p)
	if err != nil || huella == ([sha256.Size]byte{}) {
		return false
	}
	esperada := sha256.Sum256(canonico)
	solicitud := DatosSolicitudAtestacion{Dominio: DominioPerfil, Mensaje: canonico, Huella: esperada}
	return subtle.ConstantTimeCompare(huella[:], esperada[:]) == 1 &&
		AtestacionValida(solicitud.Dominio, solicitud.Mensaje, solicitud.Huella,
			a.Algoritmo, a.ClaveRef, a.ClaveVersion, a.Dominio, a.Huella, a.Codigo)
}

// DatosResultadoReferencia liga la referencia durable a la identidad exacta.
type DatosResultadoReferencia struct {
	Referencia      string
	HuellaIdentidad [sha256.Size]byte
}

func NuevaHuellaIdentidad(canonico []byte) ([sha256.Size]byte, error) {
	if len(canonico) == 0 {
		return [sha256.Size]byte{}, ErrReciboNoValido
	}
	huella := sha256.Sum256(canonico)
	if huella == ([sha256.Size]byte{}) {
		return [sha256.Size]byte{}, ErrReciboNoValido
	}
	return huella, nil
}

func ResultadoReferenciaValido(huella [sha256.Size]byte, r DatosResultadoReferencia) bool {
	return huella != ([sha256.Size]byte{}) && AliasLogicoValido(r.Referencia, 512) &&
		r.HuellaIdentidad != ([sha256.Size]byte{}) &&
		subtle.ConstantTimeCompare(r.HuellaIdentidad[:], huella[:]) == 1
}

// ReciboSelladoValido coteja el recibo, sus dominios y huellas independientes.
func ReciboSelladoValido(r Recibo, huella [sha256.Size]byte, a DatosAtestacion) bool {
	canonico, err := CanonicoRecibo(r)
	if err != nil || huella == ([sha256.Size]byte{}) {
		return false
	}
	esperada := sha256.Sum256(canonico)
	if subtle.ConstantTimeCompare(huella[:], esperada[:]) != 1 ||
		HuellasIguales(huella, r.Instantanea.HuellaContenido) ||
		HuellasIguales(huella, r.HuellaPerfil) || HuellasIguales(huella, r.HuellaPlan.Suma) {
		return false
	}
	return AtestacionValida(DominioRecibo, canonico, esperada, a.Algoritmo, a.ClaveRef,
		a.ClaveVersion, a.Dominio, a.Huella, a.Codigo)
}
