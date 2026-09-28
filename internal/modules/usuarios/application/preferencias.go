package application

import (
	"context"
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

type ServicioPreferencias struct {
	registro ports.RegistroPreferencias
	ahoraUTC func() time.Time
}

func NuevoServicioPreferencias(registro ports.RegistroPreferencias, ahoraUTC func() time.Time) (*ServicioPreferencias, error) {
	if registro == nil || ahoraUTC == nil {
		return nil, ports.ErrNoDisponible
	}
	return &ServicioPreferencias{registro: registro, ahoraUTC: ahoraUTC}, nil
}

func (s *ServicioPreferencias) actor(ctx context.Context, orden ports.OrdenPreferencias) (vecdomain.ContextoActor, error) {
	if s == nil || s.registro == nil || s.ahoraUTC == nil || ctx == nil {
		return vecdomain.ContextoActor{}, ports.ErrNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return vecdomain.ContextoActor{}, ports.ErrNoAutenticado
	}
	ahora := s.ahoraUTC().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() || !actor.Instantanea.VigenteEn(ahora) {
		return vecdomain.ContextoActor{}, ports.ErrNoAutenticado
	}
	return actor, nil
}

func material(actor vecdomain.ContextoActor, accion string, p ports.PeticionGuardarPreferencias, huella string) ports.MaterialPreferencias {
	return ports.MaterialPreferencias{PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Accion: accion, FinalidadRef: ports.FinalidadPreferenciasPropias, CatalogoVersionRef: p.CatalogoVersionRef, VersionEsperada: p.VersionEsperada, ClaveOperacion: p.ClaveOperacion, HuellaPeticion: huella, Valores: p.Valores}
}

func proveer(ctx context.Context, orden ports.OrdenPreferencias, m ports.MaterialPreferencias) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	proveedor := orden.Proveedor()
	if proveedor == nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrNoAutenticado
	}
	v3, err := proveedor.ProveerMaterialPreferencias(ctx, m)
	if err != nil {
		if errors.Is(err, ports.ErrNoAutenticado) || errors.Is(err, ports.ErrProhibido) {
			return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
		}
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrNoDisponible
	}
	if v3.ValidarEstructura() != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrNoDisponible
	}
	return v3, nil
}

func (s *ServicioPreferencias) Consultar(ctx context.Context, orden ports.OrdenPreferencias) (ports.VistaPreferencias, error) {
	actor, err := s.actor(ctx, orden)
	if err != nil {
		return ports.VistaPreferencias{}, err
	}
	catalogo, err := s.registro.CatalogoVigente(ctx)
	if err != nil || catalogo.Validar() != nil {
		return ports.VistaPreferencias{}, ports.ErrNoDisponible
	}
	m := material(actor, ports.AccionConsultarPreferencias, ports.PeticionGuardarPreferencias{CatalogoVersionRef: catalogo.VersionRef}, "")
	v3, err := proveer(ctx, orden, m)
	if err != nil {
		return ports.VistaPreferencias{}, err
	}
	estado, existe, err := s.registro.ConsultarPropias(ctx, orden, m, v3)
	if err != nil {
		return ports.VistaPreferencias{}, err
	}
	if !existe {
		estado = ports.EstadoPreferencias{PersonaRef: actor.PersonaRef, Version: 0, CatalogoVersionRef: catalogo.VersionRef, Valores: catalogo.Predeterminados}
	}
	if estado.PersonaRef != actor.PersonaRef || estado.Valores.ValidarCodigos() != nil || (existe && (estado.Version == 0 || estado.CatalogoVersionRef == "")) {
		return ports.VistaPreferencias{}, ports.ErrNoDisponible
	}
	return ports.VistaPreferencias{Catalogo: catalogo.Clonar(), Estado: estado}, nil
}

func huellaPeticion(persona string, p ports.PeticionGuardarPreferencias) (string, error) {
	// La clave queda fuera: identifica esta huella semántica por persona.
	b, err := json.Marshal(struct {
		Persona  string                     `json:"persona_ref"`
		Version  uint64                     `json:"version_esperada"`
		Catalogo string                     `json:"catalogo_version_ref"`
		Valores  domain.ValoresPreferencias `json:"valores"`
	}{persona, p.VersionEsperada, p.CatalogoVersionRef, p.Valores})
	if err != nil {
		return "", ports.ErrPeticionInvalida
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func claveValida(clave string) bool {
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

func (s *ServicioPreferencias) Guardar(ctx context.Context, orden ports.OrdenPreferencias, p ports.PeticionGuardarPreferencias) (ports.ReciboPreferencias, error) {
	actor, err := s.actor(ctx, orden)
	if err != nil {
		return ports.ReciboPreferencias{}, err
	}
	if !claveValida(p.ClaveOperacion) || p.VersionEsperada >= math.MaxInt64 || p.CatalogoVersionRef == "" || len(p.CatalogoVersionRef) > 96 || p.Valores.ValidarCodigos() != nil {
		return ports.ReciboPreferencias{}, ports.ErrPeticionInvalida
	}
	huella, err := huellaPeticion(actor.PersonaRef, p)
	if err != nil {
		return ports.ReciboPreferencias{}, err
	}
	m := material(actor, ports.AccionActualizarPreferencias, p, huella)
	v3, err := proveer(ctx, orden, m)
	if err != nil {
		return ports.ReciboPreferencias{}, err
	}
	// La recuperación consume V3 incluso cuando la clave no existe; así la
	// existencia de una operación no se observa antes de autorizarla.
	recibo, existe, err := s.registro.RecuperarOperacion(ctx, orden, m, v3)
	if err != nil {
		return ports.ReciboPreferencias{}, err
	}
	if existe {
		if recibo.PersonaRef != actor.PersonaRef || recibo.ReciboRef == "" || recibo.Version == 0 || recibo.FechaUTC.IsZero() {
			return ports.ReciboPreferencias{}, ports.ErrNoDisponible
		}
		recibo.Replay = true
		return recibo, nil
	}
	catalogo, err := s.registro.CatalogoVigente(ctx)
	if err != nil || catalogo.Validar() != nil {
		return ports.ReciboPreferencias{}, ports.ErrNoDisponible
	}
	if catalogo.VersionRef != p.CatalogoVersionRef {
		return ports.ReciboPreferencias{}, ports.ErrConflicto
	}
	if catalogo.ValidarValores(p.Valores) != nil {
		return ports.ReciboPreferencias{}, ports.ErrPeticionInvalida
	}
	v3Guardado, err := proveer(ctx, orden, m)
	if err != nil {
		return ports.ReciboPreferencias{}, err
	}
	huellaRecuperacion, errRecuperacion := v3.HuellaConjuntoSHA256()
	huellaGuardado, errGuardado := v3Guardado.HuellaConjuntoSHA256()
	if errRecuperacion != nil || errGuardado != nil || huellaRecuperacion == huellaGuardado {
		return ports.ReciboPreferencias{}, ports.ErrNoDisponible
	}
	recibo, err = s.registro.Guardar(ctx, orden, p, m, v3Guardado)
	if err != nil {
		return ports.ReciboPreferencias{}, err
	}
	if recibo.PersonaRef != actor.PersonaRef || recibo.Version != p.VersionEsperada+1 || recibo.CatalogoVersionRef != p.CatalogoVersionRef || recibo.Valores != p.Valores || recibo.ReciboRef == "" || recibo.FechaUTC.IsZero() {
		return ports.ReciboPreferencias{}, ports.ErrNoDisponible
	}
	return recibo, nil
}
