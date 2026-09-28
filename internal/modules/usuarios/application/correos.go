package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const VigenciaDesafioCorreo = 24 * time.Hour

type ServicioCorreos struct {
	registro  ports.RegistroCorreos
	protector ports.ProtectorDireccionCorreo
	desafios  ports.PreparadorDesafioCorreo
	validador ports.ValidadorCodigoCorreo
	ahoraUTC  func() time.Time
}

func NuevoServicioCorreos(registro ports.RegistroCorreos, protector ports.ProtectorDireccionCorreo, desafios ports.PreparadorDesafioCorreo, validador ports.ValidadorCodigoCorreo, ahoraUTC func() time.Time) (*ServicioCorreos, error) {
	if registro == nil || protector == nil || desafios == nil || validador == nil || ahoraUTC == nil {
		return nil, ports.ErrCorreosNoDisponible
	}
	return &ServicioCorreos{registro: registro, protector: protector, desafios: desafios, validador: validador, ahoraUTC: ahoraUTC}, nil
}

func (s *ServicioCorreos) actor(ctx context.Context, orden ports.OrdenCorreos) (vecdomain.ContextoActor, error) {
	if s == nil || s.registro == nil || s.ahoraUTC == nil || ctx == nil || ctx.Err() != nil {
		return vecdomain.ContextoActor{}, ports.ErrCorreosNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return vecdomain.ContextoActor{}, ports.ErrCorreosNoAutenticado
	}
	if !actor.Instantanea.VigenteEn(s.ahoraUTC().UTC().Truncate(time.Microsecond)) {
		return vecdomain.ContextoActor{}, ports.ErrCorreosNoAutenticado
	}
	return actor, nil
}

func materialCorreos(actor vecdomain.ContextoActor, accion string, p ports.PeticionCorreo, huella string) ports.MaterialCorreos {
	return ports.MaterialCorreos{PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Accion: accion, FinalidadRef: ports.FinalidadCorreosPropios, VersionEsperada: p.VersionEsperada, ClaveOperacion: p.ClaveOperacion, HuellaPeticion: huella, CorreoRef: p.CorreoRef, SustitutoRef: p.SustitutoRef}
}

func autorizarCorreos(ctx context.Context, orden ports.OrdenCorreos, m ports.MaterialCorreos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	proveedor := orden.Proveedor()
	if proveedor == nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrCorreosNoAutenticado
	}
	v3, err := proveedor.ProveerMaterialCorreos(ctx, m)
	if err != nil {
		if errors.Is(err, ports.ErrCorreosNoAutenticado) || errors.Is(err, ports.ErrCorreosProhibido) {
			return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
		}
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrCorreosNoDisponible
	}
	if v3.ValidarEstructura() != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrCorreosNoDisponible
	}
	return v3, nil
}

func (s *ServicioCorreos) Consultar(ctx context.Context, orden ports.OrdenCorreos) (ports.VistaCorreos, error) {
	actor, err := s.actor(ctx, orden)
	if err != nil {
		return ports.VistaCorreos{}, err
	}
	m := materialCorreos(actor, ports.AccionConsultarCorreos, ports.PeticionCorreo{}, "")
	v3, err := autorizarCorreos(ctx, orden, m)
	if err != nil {
		return ports.VistaCorreos{}, err
	}
	vista, err := s.registro.ConsultarPropios(ctx, orden, m, v3)
	if err != nil {
		return ports.VistaCorreos{}, err
	}
	if vista.PersonaRef != actor.PersonaRef || vista.Version > math.MaxInt64 || domain.ValidarConjuntoCorreos(vista.Correos) != nil {
		return ports.VistaCorreos{}, ports.ErrCorreosNoDisponible
	}
	return vista, nil
}

func claveCorreosValida(clave string) bool {
	if len(clave) < 16 || len(clave) > 128 {
		return false
	}
	for _, r := range clave {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != ':' && r != '.' {
			return false
		}
	}
	return true
}

func referenciaCorreoValida(ref string) bool {
	if len(ref) < 12 || len(ref) > 128 {
		return false
	}
	for _, r := range ref {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != ':' {
			return false
		}
	}
	return true
}

func peticionCorreoValida(accion string, p ports.PeticionCorreo) bool {
	if !claveCorreosValida(p.ClaveOperacion) || p.VersionEsperada >= math.MaxInt64 {
		return false
	}
	switch accion {
	case ports.AccionAnadirCorreo:
		return p.CorreoRef == "" && domain.DireccionCorreoValida(p.Direccion) && p.Codigo == "" && p.SustitutoRef == ""
	case ports.AccionReenviarCorreo, ports.AccionActivarCorreo:
		return referenciaCorreoValida(p.CorreoRef) && p.Direccion == "" && p.Codigo == "" && p.SustitutoRef == ""
	case ports.AccionVerificarCorreo:
		return referenciaCorreoValida(p.CorreoRef) && p.Direccion == "" && len(p.Codigo) >= 16 && len(p.Codigo) <= 128 && p.SustitutoRef == ""
	case ports.AccionRetirarCorreo:
		return referenciaCorreoValida(p.CorreoRef) && p.Direccion == "" && p.Codigo == "" && (p.SustitutoRef == "" || (referenciaCorreoValida(p.SustitutoRef) && p.SustitutoRef != p.CorreoRef))
	default:
		return false
	}
}

func huellaCorreo(persona, accion string, p ports.PeticionCorreo) (string, error) {
	// El código no viaja al material V3; su digest sólo ata el replay a los
	// mismos bytes. El código tiene al menos 128 bits y el digest no se publica.
	codigoHuella := ""
	if p.Codigo != "" {
		h := sha256.Sum256([]byte(p.Codigo))
		codigoHuella = hex.EncodeToString(h[:])
	}
	b, err := json.Marshal(struct {
		Esquema, Persona, Accion, CorreoRef, Direccion, CodigoHuella, SustitutoRef string
		Version                                                                    uint64
	}{"usuarios.correos.peticion.v1", persona, accion, p.CorreoRef, p.Direccion, codigoHuella, p.SustitutoRef, p.VersionEsperada})
	if err != nil {
		return "", ports.ErrCorreosInvalidos
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func nuevaReferenciaCorreo() (string, error) {
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return "", ports.ErrCorreosNoDisponible
	}
	return "correo:" + hex.EncodeToString(aleatorio[:]), nil
}

type comprobadorCorreo struct {
	validador ports.ValidadorCodigoCorreo
	codigo    string
}

func (c comprobadorCorreo) Comprobar(ctx context.Context, meta ports.MetadatosDesafioCorreo) (bool, error) {
	if c.validador == nil || c.codigo == "" {
		return false, ports.ErrCorreosNoDisponible
	}
	return c.validador.ComprobarCodigoCorreo(ctx, meta, c.codigo)
}

func (s *ServicioCorreos) mutar(ctx context.Context, orden ports.OrdenCorreos, accion string, p ports.PeticionCorreo) (ports.ReciboCorreos, error) {
	actor, err := s.actor(ctx, orden)
	if err != nil {
		return ports.ReciboCorreos{}, err
	}
	if !peticionCorreoValida(accion, p) {
		return ports.ReciboCorreos{}, ports.ErrCorreosInvalidos
	}
	huella, err := huellaCorreo(actor.PersonaRef, accion, p)
	if err != nil {
		return ports.ReciboCorreos{}, err
	}
	m := materialCorreos(actor, accion, p, huella)
	v3, err := autorizarCorreos(ctx, orden, m)
	if err != nil {
		return ports.ReciboCorreos{}, err
	}
	recibo, existe, err := s.registro.RecuperarOperacion(ctx, orden, m, v3)
	if err != nil {
		return ports.ReciboCorreos{}, err
	}
	if existe {
		if !reciboCorreoValido(recibo, actor.PersonaRef, accion) {
			return ports.ReciboCorreos{}, ports.ErrCorreosNoDisponible
		}
		recibo.Replay = true
		return recibo, nil
	}
	var reserva ports.ReservaDesafio
	var sobre ports.SobreDireccionCorreo
	if accion == ports.AccionAnadirCorreo {
		p.CorreoRef, err = nuevaReferenciaCorreo()
		if err != nil {
			return ports.ReciboCorreos{}, err
		}
		claro := []byte(p.Direccion)
		sobre, err = s.protector.CifrarDireccionCorreo(ctx, actor.PersonaRef, p.CorreoRef, p.VersionEsperada+1, claro)
		for i := range claro {
			claro[i] = 0
		}
		if err != nil || sobre.CorreoRef != p.CorreoRef || sobre.Version != p.VersionEsperada+1 || sobre.ClaveRef == "" || len(sobre.Nonce) < 12 || len(sobre.Cifrado) == 0 || len(sobre.HuellaIgualdad) < 16 {
			return ports.ReciboCorreos{}, ports.ErrCorreosNoDisponible
		}
		p.Direccion = "" // El registro recibe sólo el sobre, nunca el claro.
	}
	if accion == ports.AccionAnadirCorreo || accion == ports.AccionReenviarCorreo {
		reserva, err = s.desafios.PrepararDesafioCorreo(ctx, actor.PersonaRef, p.CorreoRef, s.ahoraUTC().UTC().Add(VigenciaDesafioCorreo))
		if err != nil || len(reserva.Desafio) < 16 || len(reserva.HuellaCodigo) < 16 || reserva.ClaveRef == "" || reserva.DesafioRef == "" || reserva.VenceUTC.IsZero() {
			return ports.ReciboCorreos{}, ports.ErrCorreosNoDisponible
		}
	}
	comprobador := comprobadorCorreo{}
	if accion == ports.AccionVerificarCorreo {
		comprobador = comprobadorCorreo{validador: s.validador, codigo: p.Codigo}
		p.Codigo = "" // El registro sólo recibe el callback, nunca el código.
	}
	recibo, err = s.registro.Aplicar(ctx, orden, p, m, v3, sobre, reserva, comprobador)
	if err != nil {
		return ports.ReciboCorreos{}, err
	}
	if !reciboCorreoValido(recibo, actor.PersonaRef, accion) || recibo.Version != m.VersionEsperada+1 || recibo.Replay || (accion != ports.AccionAnadirCorreo && recibo.CorreoRef != m.CorreoRef) || (accion == ports.AccionAnadirCorreo && recibo.CorreoRef != p.CorreoRef) {
		return ports.ReciboCorreos{}, ports.ErrCorreosNoDisponible
	}
	return recibo, nil
}

func reciboCorreoValido(r ports.ReciboCorreos, persona, accion string) bool {
	return r.ReciboRef != "" && r.PersonaRef == persona && r.Accion == accion && referenciaCorreoValida(r.CorreoRef) && r.Version > 0 && !r.FechaUTC.IsZero()
}

func (s *ServicioCorreos) Anadir(ctx context.Context, o ports.OrdenCorreos, p ports.PeticionCorreo) (ports.ReciboCorreos, error) {
	return s.mutar(ctx, o, ports.AccionAnadirCorreo, p)
}
func (s *ServicioCorreos) Reenviar(ctx context.Context, o ports.OrdenCorreos, p ports.PeticionCorreo) (ports.ReciboCorreos, error) {
	return s.mutar(ctx, o, ports.AccionReenviarCorreo, p)
}
func (s *ServicioCorreos) Verificar(ctx context.Context, o ports.OrdenCorreos, p ports.PeticionCorreo) (ports.ReciboCorreos, error) {
	return s.mutar(ctx, o, ports.AccionVerificarCorreo, p)
}
func (s *ServicioCorreos) Activar(ctx context.Context, o ports.OrdenCorreos, p ports.PeticionCorreo) (ports.ReciboCorreos, error) {
	return s.mutar(ctx, o, ports.AccionActivarCorreo, p)
}
func (s *ServicioCorreos) Retirar(ctx context.Context, o ports.OrdenCorreos, p ports.PeticionCorreo) (ports.ReciboCorreos, error) {
	return s.mutar(ctx, o, ports.AccionRetirarCorreo, p)
}
