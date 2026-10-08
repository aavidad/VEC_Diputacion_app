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
	VigenteDesde      *string            `json:"vigente_desde,omitempty"`
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
	activacion, err := s.repo.LeerActivacion(ctx)
	if err != nil {
		return Lectura{}, ErrNoDisponible
	}
	lectura.Activacion = activacion
	if activacion.Estado == "sin_publicar" || activacion.Estado == "inactiva" {
		// La historia sigue siendo consultable, pero el catálogo local no se
		// presenta como regla vigente hasta que CT158 confirme su activación.
		lectura.PuedeAjustar = false
		lectura.Reglas = []reglas.Regla{}
		return lectura, nil
	}
	base, huella, _, err := s.reglas.CatalogoVigente(ctx)
	if err != nil {
		return Lectura{}, ErrNoDisponible
	}
	if comprobarBaseActiva(base, huella, activacion) != nil {
		return Lectura{}, ErrNoDisponible
	}
	lectura.Reglas, err = reglas.ProyectarReglasConAjustes(base, lectura.CorteEn, lectura.VigenteHoy)
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
	// La lectura exige sesión certificada, permiso exacto y auditoría común
	// en el consumidor owner-only. Operar pide una decisión V3 nueva y CT195
	// la consume en la misma transacción que el efecto, también en replay.
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
	efectoDesde, efectoCanonico, err := fechaEfectoSolicitada(solicitud.VigenteDesde)
	if err != nil {
		return Resultado{}, ErrEntradaInvalida
	}
	var preparada reglas.PreparacionAjustes
	if version == *solicitud.VersionEsperada {
		base, huella, _, fuenteErr := s.reglas.CatalogoVigente(ctx)
		if fuenteErr != nil {
			return Resultado{}, ErrNoDisponible
		}
		activacion, activacionErr := s.repo.LeerActivacion(ctx)
		if activacionErr != nil || comprobarBaseActiva(base, huella, activacion) != nil {
			return Resultado{}, ErrNoDisponible
		}
		previa := reglas.VersionAjustes{}
		if lectura.Vigente != nil {
			previa = *lectura.Vigente
		}
		preparada, err = reglas.PrepararAjustesSobreVersion(base, ahora, efectoDesde, version, previa, lectura.Vigente != nil, cambios)
	} else {
		// La preimagen almacenada es la única fuente de material de replay:
		// recalcular desde la cabeza actual alteraría la huella tras otras versiones.
		original, encontrada, leerErr := s.repo.LeerPreimagen(ctx, actor, solicitud.ClaveIdempotencia)
		if leerErr != nil {
			return Resultado{}, leerErr
		}
		if !encontrada || !mismaIntencion(original, solicitud, efectoCanonico) {
			return Resultado{}, ErrConflicto
		}
		return s.operarYValidar(ctx, actor, original, solicitud.ClaveIdempotencia, version, true)
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
		VigenteDesde: efectoCanonico,
		BaseVersion:  d.BaseVersion, BaseHuellaSHA256: d.BaseHuellaSHA256,
		AjustesCanonico: string(d.Canonico), AjustesHuellaSHA256: d.HuellaSHA256,
		MotivoClave: solicitud.MotivoClave, Referencia: solicitud.Referencia, Nota: solicitud.Nota,
		Cambios: make([]Cambio, len(d.Cambios)),
	}
	for i, c := range d.Cambios {
		material.Cambios[i] = Cambio(c)
	}
	return s.operarYValidar(ctx, actor, material, solicitud.ClaveIdempotencia, version, false)
}

func (s *Servicio) operarYValidar(ctx context.Context, actor vecdomain.ContextoActor, material Material, clave string, version int, replayEsperado bool) (Resultado, error) {
	resultado, err := s.repo.Operar(ctx, actor, material)
	if err != nil {
		return Resultado{}, err
	}
	if resultado.Recibo.ReciboRef == "" || resultado.Recibo.ClaveIdempotencia != clave ||
		resultado.Recibo.Version < 1 || resultado.Recibo.VigenteDesde.IsZero() || resultado.Recibo.PublicadaEn.IsZero() ||
		resultado.Recibo.VigenteDesde.Before(resultado.Recibo.PublicadaEn) ||
		resultado.Recibo.HuellaSHA256 == "" || resultado.Recibo.DecisionRef == "" ||
		resultado.Recibo.AuditoriaRef == "" || resultado.Recibo.ConsumoHuellaSHA256 == "" ||
		(resultado.Replay != replayEsperado) ||
		(!resultado.Replay && (resultado.Recibo.Version != version+1 || resultado.Recibo.HuellaSHA256 != material.AjustesHuellaSHA256)) ||
		(resultado.Replay && resultado.Recibo.Version > version) {
		return Resultado{}, ErrNoDisponible
	}
	if material.VigenteDesde != nil {
		efecto, err := time.Parse("2006-01-02T15:04:05.000000Z", *material.VigenteDesde)
		if err != nil || !resultado.Recibo.VigenteDesde.Equal(efecto) {
			return Resultado{}, ErrNoDisponible
		}
	}
	return resultado, nil
}

func fechaEfectoSolicitada(valor *string) (*time.Time, *string, error) {
	if valor == nil {
		return nil, nil, nil
	}
	texto := *valor
	if !strings.HasSuffix(texto, "Z") || len(texto) < len("2006-01-02T15:04:05Z") ||
		len(texto) > len("2006-01-02T15:04:05.000000Z") {
		return nil, nil, ErrEntradaInvalida
	}
	fecha, err := time.Parse(time.RFC3339Nano, texto)
	if err != nil || fecha.IsZero() || fecha.Location() != time.UTC || fecha.Nanosecond()%1000 != 0 {
		return nil, nil, ErrEntradaInvalida
	}
	canonico := fecha.UTC().Format("2006-01-02T15:04:05.000000Z")
	return &fecha, &canonico, nil
}

func mismaIntencion(original Material, solicitud Solicitud, efecto *string) bool {
	if original.Operacion != "ajustar" || original.ClaveIdempotencia != solicitud.ClaveIdempotencia ||
		original.VersionEsperada != *solicitud.VersionEsperada || !igualOpcional(original.VigenteDesde, efecto) ||
		original.MotivoClave != solicitud.MotivoClave || !igualOpcional(original.Referencia, solicitud.Referencia) ||
		!igualOpcional(original.Nota, solicitud.Nota) || len(original.Cambios) != len(solicitud.Cambios) {
		return false
	}
	pares := make(map[string]string, len(original.Cambios))
	for _, c := range original.Cambios {
		par := c.ReglaClave + "\x00" + c.Campo
		if _, existe := pares[par]; existe {
			return false
		}
		pares[par] = c.Nuevo
	}
	vistos := make(map[string]bool, len(solicitud.Cambios))
	for _, c := range solicitud.Cambios {
		par := c.ReglaClave + "\x00" + c.Campo
		if nuevo, existe := pares[par]; !existe || nuevo != c.Nuevo || vistos[par] {
			return false
		}
		vistos[par] = true
	}
	return len(pares) == len(solicitud.Cambios)
}

func igualOpcional(a, b *string) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

// Solo la huella canónica del catálogo cargado por Go se compara con CT158.
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
	if l.CorteEn.IsZero() {
		return ErrNoDisponible
	}
	if l.Vigente == nil {
		if l.VigenteBaseVersion != 0 || l.VigenteBaseHuella != "" || l.VigenteHoy != nil || len(l.Programados) != 0 {
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
	if !l.Vigente.VigenteDesde.After(l.CorteEn) &&
		(l.VigenteHoy == nil || l.VigenteHoy.Version != l.Vigente.Version) {
		return ErrNoDisponible
	}
	if l.VigenteHoy != nil {
		if l.VigenteHoy.Version > l.Vigente.Version || l.VigenteHoy.VigenteDesde.After(l.Vigente.VigenteDesde) ||
			l.VigenteHoy.VigenteDesde.After(l.CorteEn) {
			return ErrNoDisponible
		}
		vigenteHuella, err := reglas.HuellaAjustes(l.VigenteHoy.Ajustes)
		if err != nil || vigenteHuella != l.VigenteHoy.HuellaSHA256 {
			return ErrNoDisponible
		}
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
