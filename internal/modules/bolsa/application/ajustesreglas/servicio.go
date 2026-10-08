package ajustesreglas

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"time"

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
	fuente  FuenteReglas
	motivos CatalogoMotivos
	reloj   reglas.Reloj
}

func NuevoServicio(repo Repositorio, fuente FuenteReglas, motivos CatalogoMotivos, reloj reglas.Reloj) (*Servicio, error) {
	if dependenciaNula(repo) || dependenciaNula(fuente) || dependenciaNula(reloj) {
		return nil, ErrNoDisponible
	}
	b, err := json.Marshal(motivos)
	if err != nil {
		return nil, ErrNoDisponible
	}
	canon, err := LeerCatalogoMotivos(b)
	if err != nil {
		return nil, ErrNoDisponible
	}
	return &Servicio{repo: repo, fuente: fuente, motivos: canon, reloj: reloj}, nil
}

func dependenciaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}

func (s *Servicio) Motivos() []Motivo {
	if s == nil {
		return nil
	}
	return s.motivos.Lista()
}

func (s *Servicio) Consultar(ctx context.Context, actor vecdomain.ContextoActor, limite int, antes *int64) (Lectura, error) {
	if s == nil || ctx == nil || actor.Validar() != nil || limite < 1 || limite > 50 || antes != nil && (*antes < 2 || *antes > 10_000_000) {
		return Lectura{}, ErrEntradaInvalida
	}
	lectura, err := s.repo.Consultar(ctx, actor, limite, antes)
	if err != nil {
		return Lectura{}, err
	}
	if validarLectura(lectura) != nil {
		return Lectura{}, ErrNoDisponible
	}
	base, huella, _, err := s.fuente.CatalogoVigente(ctx)
	if err != nil || base.ID != CatalogoBase {
		return Lectura{}, ErrNoDisponible
	}
	comprobada, err := base.HuellaSHA256()
	if err != nil || comprobada != huella {
		return Lectura{}, ErrNoDisponible
	}
	lectura.Reglas, err = reglas.ProyectarReglasConAjustes(base, lectura.ConsultadaEn, lectura.VigenteHoy)
	if err != nil {
		return Lectura{}, ErrNoDisponible
	}
	return lectura, nil
}

func (s *Servicio) Publicar(ctx context.Context, actor vecdomain.ContextoActor, p Solicitud) (Resultado, error) {
	if s == nil || ctx == nil || actor.Validar() != nil || !p.valida(s.motivos) {
		return Resultado{}, ErrEntradaInvalida
	}
	if err := ctx.Err(); err != nil {
		return Resultado{}, err
	}
	// B88/AD223 autoriza el acto en su transacción. Esta lectura sólo obtiene
	// la cabeza no personal para preparar el CAS; no pide READ V3 previo.
	versionCabeza, err := s.repo.LeerCabeza(ctx)
	if err != nil {
		return Resultado{}, err
	}
	cabeza := 0
	if versionCabeza != nil {
		if validarVersion(*versionCabeza) != nil {
			return Resultado{}, ErrNoDisponible
		}
		cabeza = versionCabeza.Version
	}
	if cabeza < *p.VersionEsperada {
		return Resultado{}, ErrConflicto
	}
	efecto, canonEfecto, err := fechaSolicitada(p.VigenteDesde)
	if err != nil {
		return Resultado{}, err
	}
	if cabeza > *p.VersionEsperada {
		original, existe, err := s.repo.LeerPreimagen(ctx, actor, p.ClaveIdempotencia)
		if err != nil {
			return Resultado{}, err
		}
		if !existe || !mismaSolicitud(original, p, canonEfecto) {
			return Resultado{}, ErrConflicto
		}
		return s.operar(ctx, actor, original, p.ClaveIdempotencia, cabeza, true)
	}
	base, huella, _, err := s.fuente.CatalogoVigente(ctx)
	if err != nil || base.ID != CatalogoBase {
		return Resultado{}, ErrNoDisponible
	}
	calculada, err := base.HuellaSHA256()
	if err != nil || calculada != huella {
		return Resultado{}, ErrNoDisponible
	}
	ahora := s.reloj.Ahora().UTC()
	if ahora.IsZero() {
		return Resultado{}, ErrNoDisponible
	}
	cambios := make([]reglas.SolicitudCambioAjuste, len(p.Cambios))
	for i, c := range p.Cambios {
		cambios[i] = reglas.SolicitudCambioAjuste{ReglaClave: c.ReglaClave, Campo: c.Campo, Nuevo: c.Nuevo}
	}
	previa := reglas.VersionAjustes{}
	if versionCabeza != nil {
		previa = *versionCabeza
	}
	preparada, err := reglas.PrepararAjustesSobreVersion(base, ahora, efecto, cabeza, previa, versionCabeza != nil, cambios)
	if err != nil {
		switch {
		case errors.Is(err, reglas.ErrAjustesConflicto):
			return Resultado{}, ErrConflicto
		case errors.Is(err, reglas.ErrAjusteInvalido):
			return Resultado{}, ErrEntradaInvalida
		default:
			return Resultado{}, ErrNoDisponible
		}
	}
	d := preparada.Datos()
	m := Material{Operacion: "ajustar", CatalogoID: d.CatalogoAjustesID, ClaveIdempotencia: p.ClaveIdempotencia,
		VersionEsperada: *p.VersionEsperada, VigenteDesde: canonEfecto, BaseVersion: d.BaseVersion,
		BaseHuellaSHA256: d.BaseHuellaSHA256, AjustesCanonico: string(d.Canonico), AjustesHuellaSHA256: d.HuellaSHA256,
		MotivoClave: p.MotivoClave, Referencia: p.Referencia, Nota: p.Nota, Cambios: make([]Cambio, len(d.Cambios))}
	for i, c := range d.Cambios {
		m.Cambios[i] = Cambio(c)
	}
	return s.operar(ctx, actor, m, p.ClaveIdempotencia, cabeza, false)
}

func (s *Servicio) operar(ctx context.Context, actor vecdomain.ContextoActor, m Material, clave string, cabeza int, replay bool) (Resultado, error) {
	r, err := s.repo.Operar(ctx, actor, m)
	if err != nil {
		return Resultado{}, err
	}
	if r.Replay != replay || r.Recibo.ClaveIdempotencia != clave || r.Recibo.ReciboRef == "" || r.Recibo.Version < 1 || r.Recibo.Version > cabeza+1 ||
		r.Recibo.HuellaSHA256 == "" || r.Recibo.DecisionRef == "" || r.Recibo.AuditoriaRef == "" || r.Recibo.ConsumoHuellaSHA256 == "" ||
		r.Recibo.PublicadaEn.IsZero() || r.Recibo.VigenteDesde.IsZero() ||
		!replay && (r.Recibo.Version != cabeza+1 || r.Recibo.HuellaSHA256 != m.AjustesHuellaSHA256) {
		return Resultado{}, ErrNoDisponible
	}
	return r, nil
}

func validarLectura(l Lectura) error {
	if l.ConsultadaEn.IsZero() {
		return ErrNoDisponible
	}
	if l.Cabeza == nil {
		if l.VigenteHoy != nil || len(l.Programados) != 0 || !l.CabezaPublicadaEn.IsZero() {
			return ErrNoDisponible
		}
		return nil
	}
	if validarVersion(*l.Cabeza) != nil || l.CabezaPublicadaEn.IsZero() || l.CabezaPublicadaEn.After(l.ConsultadaEn) || l.Cabeza.VigenteDesde.Before(l.CabezaPublicadaEn) {
		return ErrNoDisponible
	}
	if !l.Cabeza.VigenteDesde.After(l.ConsultadaEn) && (l.VigenteHoy == nil || l.VigenteHoy.Version != l.Cabeza.Version) {
		return ErrNoDisponible
	}
	if l.VigenteHoy != nil {
		if validarVersion(*l.VigenteHoy) != nil || l.VigenteHoy.Version > l.Cabeza.Version || l.VigenteHoy.VigenteDesde.After(l.ConsultadaEn) || l.VigentePublicadaEn.IsZero() || l.VigentePublicadaEn.After(l.ConsultadaEn) {
			return ErrNoDisponible
		}
	}
	for i, p := range l.Programados {
		h, err := reglas.HuellaAjustes(p.Ajustes)
		if err != nil || h != p.HuellaSHA256 || p.Version < 1 || p.Version > l.Cabeza.Version ||
			!p.VigenteDesde.After(l.ConsultadaEn) || p.PublicadaEn.IsZero() || p.PublicadaEn.After(l.ConsultadaEn) ||
			p.VigenteDesde.Before(p.PublicadaEn) ||
			(l.VigenteHoy != nil && p.Version <= l.VigenteHoy.Version) ||
			(i > 0 && p.VigenteDesde.Before(l.Programados[i-1].VigenteDesde)) {
			return ErrNoDisponible
		}
	}
	return nil
}

func validarVersion(v reglas.VersionAjustes) error {
	if v.CatalogoID != CatalogoAjustes || v.Version < 1 || v.Version > 9_999_999 || v.VigenteDesde.IsZero() {
		return ErrNoDisponible
	}
	h, err := reglas.HuellaAjustes(v.Ajustes)
	if err != nil || h != v.HuellaSHA256 {
		return ErrNoDisponible
	}
	return nil
}

var uuidCanonical = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func (p Solicitud) valida(m CatalogoMotivos) bool {
	if p.VersionEsperada == nil || *p.VersionEsperada < 0 || *p.VersionEsperada > 9_999_998 || !uuidCanonical.MatchString(p.ClaveIdempotencia) ||
		len(p.Cambios) < 1 || len(p.Cambios) > 256 || !m.Admite(p.MotivoClave) || !textoOpcional(p.Referencia, 120) || !textoOpcional(p.Nota, 500) {
		return false
	}
	return true
}
func textoOpcional(s *string, n int) bool {
	if s == nil {
		return true
	}
	if *s == "" || len([]rune(*s)) > n || strings.TrimSpace(*s) != *s {
		return false
	}
	for _, r := range *s {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}
func fechaSolicitada(raw *string) (*time.Time, *string, error) {
	if raw == nil {
		return nil, nil, nil
	}
	if len(*raw) < 20 || len(*raw) > 27 || !strings.HasSuffix(*raw, "Z") {
		return nil, nil, ErrEntradaInvalida
	}
	t, err := time.Parse(time.RFC3339Nano, *raw)
	if err != nil || t.IsZero() || t.Nanosecond()%1000 != 0 {
		return nil, nil, ErrEntradaInvalida
	}
	u := t.UTC()
	s := u.Format("2006-01-02T15:04:05.000000Z")
	return &u, &s, nil
}
func mismaSolicitud(m Material, p Solicitud, fecha *string) bool {
	if m.Operacion != "ajustar" || m.CatalogoID != CatalogoAjustes || m.ClaveIdempotencia != p.ClaveIdempotencia ||
		m.VersionEsperada != *p.VersionEsperada || m.MotivoClave != p.MotivoClave || !igualOpt(m.VigenteDesde, fecha) ||
		!igualOpt(m.Referencia, p.Referencia) || !igualOpt(m.Nota, p.Nota) || len(m.Cambios) != len(p.Cambios) {
		return false
	}
	pares := make(map[string]string, len(m.Cambios))
	for _, c := range m.Cambios {
		k := c.ReglaClave + "\x00" + c.Campo
		if _, existe := pares[k]; existe {
			return false
		}
		pares[k] = c.Nuevo
	}
	vistos := make(map[string]bool, len(p.Cambios))
	for _, c := range p.Cambios {
		k := c.ReglaClave + "\x00" + c.Campo
		if v, existe := pares[k]; !existe || v != c.Nuevo || vistos[k] {
			return false
		}
		vistos[k] = true
	}
	return true
}
func igualOpt(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
