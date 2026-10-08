package ajustesreglas

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

type FuenteReglas interface {
	CatalogoVigente(context.Context) (vecdomain.CatalogoConfigurable, string, time.Time, error)
}

type CambioSolicitado struct {
	ReglaClave string `json:"regla_clave"`
	Campo      string `json:"campo"`
	Nuevo      string `json:"nuevo"`
}

type Solicitud struct {
	ClaveIdempotencia string             `json:"clave_idempotencia"`
	VersionEsperada   *int               `json:"version_esperada"`
	Cambios           []CambioSolicitado `json:"cambios"`
	MotivoClave       string             `json:"motivo_clave"`
	Referencia        *string            `json:"referencia,omitempty"`
	Nota              *string            `json:"nota,omitempty"`
}

type Servicio struct {
	repo    Repositorio
	reglas  FuenteReglas
	motivos CatalogoMotivos
	reloj   reglas.Reloj
}

func NuevoServicio(repo Repositorio, fuente FuenteReglas, motivos CatalogoMotivos, reloj reglas.Reloj) (*Servicio, error) {
	if repo == nil || fuente == nil || reloj == nil {
		return nil, ErrNoDisponible
	}
	contenido, err := json.Marshal(motivos)
	if err != nil {
		return nil, ErrNoDisponible
	}
	canonico, err := LeerCatalogoMotivos(contenido)
	if err != nil {
		return nil, ErrNoDisponible
	}
	return &Servicio{repo: repo, reglas: fuente, motivos: canonico, reloj: reloj}, nil
}

func (s *Servicio) Motivos() []Motivo {
	if s == nil {
		return nil
	}
	return append([]Motivo(nil), s.motivos.Motivos...)
}

func (s *Servicio) Consultar(ctx context.Context, actor vecdomain.ContextoActor, limite int, antes *int64) (Lectura, error) {
	if s == nil || ctx == nil || actor.Validar() != nil || limite < 1 || limite > 50 ||
		(antes != nil && (*antes < 2 || *antes > 10_000_000)) {
		return Lectura{}, ErrEntradaInvalida
	}
	lectura, err := s.repo.Consultar(ctx, actor, limite, antes)
	if err != nil {
		return Lectura{}, err
	}
	if err := validarCabeza(lectura); err != nil {
		return Lectura{}, err
	}
	// El repositorio entrega la activación y la cabeza comprobadas. CT191
	// repite la guarda de activación dentro de cualquier escritura posterior.
	activacion := lectura.Activacion
	if activacion.Estado != "activa" && activacion.Estado != "inactiva" && activacion.Estado != "sin_publicar" {
		return Lectura{}, ErrNoDisponible
	}
	if activacion.Estado == "sin_publicar" || activacion.Estado == "inactiva" {
		// La historia sigue siendo consultable, pero el catálogo local no se
		// presenta como regla vigente hasta que CT191 confirme su activación.
		lectura.PuedeAjustar = false
		lectura.Reglas = []reglas.Regla{}
		return lectura, nil
	}
	base, huella, instante, err := s.reglas.CatalogoVigente(ctx)
	if err != nil {
		return Lectura{}, ErrNoDisponible
	}
	if comprobarBaseActiva(base, huella, activacion) != nil {
		return Lectura{}, ErrNoDisponible
	}
	lectura.Reglas, err = reglas.ProyectarReglasConAjustes(base, instante, lectura.Vigente)
	if err != nil {
		return Lectura{}, ErrNoDisponible
	}
	return lectura, nil
}

func (s *Servicio) Publicar(ctx context.Context, actor vecdomain.ContextoActor, solicitud Solicitud) (Resultado, error) {
	if s == nil || ctx == nil || actor.Validar() != nil || !solicitud.valida(s.motivos) {
		return Resultado{}, ErrEntradaInvalida
	}
	if err := ctx.Err(); err != nil {
		return Resultado{}, err
	}
	// La lectura de cabeza reutiliza la consulta V3 existente de CT148; la
	// escritura pide otra decisión ligada al material y CT la consume al guardar.
	lectura, err := s.repo.Consultar(ctx, actor, 1, nil)
	if err != nil {
		return Resultado{}, err
	}
	if err := validarCabeza(lectura); err != nil {
		return Resultado{}, err
	}
	version := 0
	if lectura.Vigente != nil {
		version = lectura.Vigente.Version
	}
	if version < *solicitud.VersionEsperada {
		return Resultado{}, ErrConflicto
	}
	ahora := s.reloj.Ahora().UTC()
	if ahora.IsZero() {
		return Resultado{}, ErrNoDisponible
	}
	cambios := make([]reglas.SolicitudCambioAjuste, len(solicitud.Cambios))
	for i, c := range solicitud.Cambios {
		cambios[i] = reglas.SolicitudCambioAjuste{ReglaClave: c.ReglaClave, Campo: c.Campo, Nuevo: c.Nuevo}
	}
	var preparada reglas.PreparacionAjustes
	if version == *solicitud.VersionEsperada {
		base, huella, _, err := s.reglas.CatalogoVigente(ctx)
		if err != nil {
			return Resultado{}, ErrNoDisponible
		}
		activacion, err := s.repo.LeerActivacion(ctx)
		if err != nil || comprobarBaseActiva(base, huella, activacion) != nil {
			return Resultado{}, ErrNoDisponible
		}
		previa := reglas.VersionAjustes{}
		if lectura.Vigente != nil {
			previa = *lectura.Vigente
		}
		preparada, err = reglas.PrepararCambioAjustes(base, ahora, version, previa, lectura.Vigente != nil, cambios)
	} else {
		// Solo un replay puede superar esta rama: el CAS original queda
		// obsoleto, de modo que una clave nueva no puede crear otra versión.
		preparada, err = reglas.PrepararRepeticionAjustes(ahora, *solicitud.VersionEsperada,
			*lectura.Vigente, lectura.VigenteBaseVersion, lectura.VigenteBaseHuella, cambios)
	}
	if err != nil {
		if errors.Is(err, reglas.ErrAjustesConflicto) {
			return Resultado{}, ErrConflicto
		}
		if errors.Is(err, reglas.ErrAjusteInvalido) {
			return Resultado{}, ErrEntradaInvalida
		}
		return Resultado{}, ErrNoDisponible
	}
	d := preparada.Datos()
	material := Material{
		Operacion: "ajustar", CatalogoID: d.CatalogoAjustesID,
		ClaveIdempotencia: solicitud.ClaveIdempotencia, VersionEsperada: *solicitud.VersionEsperada,
		BaseVersion: d.BaseVersion, BaseHuellaSHA256: d.BaseHuellaSHA256,
		AjustesCanonico: string(d.Canonico), AjustesHuellaSHA256: d.HuellaSHA256,
		MotivoClave: solicitud.MotivoClave, Referencia: solicitud.Referencia, Nota: solicitud.Nota,
		Cambios: make([]Cambio, len(d.Cambios)),
	}
	for i, c := range d.Cambios {
		material.Cambios[i] = Cambio(c)
	}
	resultado, err := s.repo.Operar(ctx, actor, material)
	if err != nil {
		return Resultado{}, err
	}
	if resultado.Recibo.ReciboRef == "" || resultado.Recibo.ClaveIdempotencia != solicitud.ClaveIdempotencia ||
		resultado.Recibo.Version < 1 || resultado.Recibo.VigenteDesde.IsZero() ||
		resultado.Recibo.HuellaSHA256 == "" || resultado.Recibo.DecisionRef == "" ||
		resultado.Recibo.AuditoriaRef == "" || resultado.Recibo.ConsumoHuellaSHA256 == "" ||
		(!resultado.Replay && (resultado.Recibo.Version != version+1 || resultado.Recibo.HuellaSHA256 != d.HuellaSHA256)) ||
		(resultado.Replay && resultado.Recibo.Version > version) {
		return Resultado{}, ErrNoDisponible
	}
	return resultado, nil
}

// Solo la huella canónica del catálogo cargado por Go se compara con CT191.
// La huella del fichero fuente pertenece al proceso de publicación y no
// identifica la definición que resuelve el caso de uso.
func comprobarBaseActiva(base vecdomain.CatalogoConfigurable, huella string, activacion ActivacionBase) error {
	calculada, err := base.HuellaSHA256()
	if err != nil || calculada != huella || activacion.Estado != "activa" || activacion.Secuencia < 1 ||
		base.Estado != vecdomain.EstadoCatalogoPublicado || base.AprobacionRef == "" ||
		activacion.CatalogoID != base.ID || activacion.Version != base.Version ||
		activacion.HuellaSHA256 != huella || activacion.AprobacionRef != base.AprobacionRef {
		return ErrNoDisponible
	}
	return nil
}

func validarCabeza(l Lectura) error {
	if l.Vigente == nil {
		if l.VigenteBaseVersion != 0 || l.VigenteBaseHuella != "" {
			return ErrNoDisponible
		}
		return nil
	}
	if l.Vigente.CatalogoID != reglas.CatalogoAjustesDe(reglas.CatalogoContratacionTemporal) ||
		l.Vigente.Version < 1 || l.VigenteBaseVersion < 1 || l.VigenteBaseHuella == "" ||
		l.Vigente.VigenteDesde.IsZero() {
		return ErrNoDisponible
	}
	huella, err := reglas.HuellaAjustes(l.Vigente.Ajustes)
	if err != nil || huella != l.Vigente.HuellaSHA256 {
		return ErrNoDisponible
	}
	return nil
}

func (s Solicitud) valida(motivos CatalogoMotivos) bool {
	if s.VersionEsperada == nil || *s.VersionEsperada < 0 || *s.VersionEsperada > 9_999_998 ||
		!ports.ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		len(s.Cambios) == 0 || len(s.Cambios) > 256 || !motivos.Admite(s.MotivoClave) ||
		!textoOpcionalValido(s.Referencia, 120) || !textoOpcionalValido(s.Nota, 500) {
		return false
	}
	return true
}

func textoOpcionalValido(v *string, limite int) bool {
	if v == nil {
		return true
	}
	if *v == "" || len([]rune(*v)) > limite || strings.TrimSpace(*v) != *v {
		return false
	}
	for _, c := range *v {
		if c < 32 || c == 127 {
			return false
		}
	}
	return true
}
