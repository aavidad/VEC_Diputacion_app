package domain

import (
	"encoding/hex"
	"errors"
	"strings"
)

var ErrEfectoPermisoSaldoInvalido = errors.New("cronos_efecto_permiso_saldo_invalido")

type ReglaEfectoPermisoSaldo struct {
	Referencia         string
	PermisoRef         string
	CatalogoVersionRef string
	ColectivoRef       string
	Efecto             string
	FuenteRef          string
	FuenteURL          string
	VigenteDesde       string
	HastaExclusivo     string
	FuenteVersionRef   string
	FuentePublicadaEn  string
	FuenteConsultadaEn string
}

type PoliticaEfectosPermisoSaldo struct {
	Referencia string
	Version    int64
	SHA256     string
	Reglas     []ReglaEfectoPermisoSaldo
}

func HuellaEfectosValida(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32 && hex.EncodeToString(b) == s
}

func ReferenciaEfectosValida(s string) bool { return referenciaMarcaje(s) }

func (p PoliticaEfectosPermisoSaldo) Validar() error {
	if !referenciaMarcaje(p.Referencia) || p.Version < 1 || !HuellaEfectosValida(p.SHA256) || p.Reglas == nil || len(p.Reglas) > 100 {
		return ErrEfectoPermisoSaldoInvalido
	}
	refs, claves := map[string]bool{}, map[[3]string]bool{}
	for _, r := range p.Reglas {
		desde, e1 := fechaPermiso(r.VigenteDesde)
		hasta, e2 := fechaPermiso(r.HastaExclusivo)
		publicada, e3 := fechaPermiso(r.FuentePublicadaEn)
		consultada, e4 := fechaPermiso(r.FuenteConsultadaEn)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || !hasta.After(desde) || consultada.Before(publicada) || !referenciaMarcaje(r.FuenteVersionRef) {
			return ErrEfectoPermisoSaldoInvalido
		}
		clave := [3]string{r.PermisoRef, r.CatalogoVersionRef, r.ColectivoRef}
		if !referenciaMarcaje(r.Referencia) || !referenciaMarcaje(r.PermisoRef) || !referenciaMarcaje(r.CatalogoVersionRef) || !referenciaMarcaje(r.ColectivoRef) || !referenciaMarcaje(r.FuenteRef) || len(r.FuenteURL) > 512 || !strings.HasPrefix(r.FuenteURL, "https://") || r.Efecto != "credito_jornada_completa" || refs[r.Referencia] || claves[clave] {
			return ErrEfectoPermisoSaldoInvalido
		}
		refs[r.Referencia], claves[clave] = true, true
	}
	return nil
}

type ProgramacionDiaPermisoSaldo struct {
	Fecha              string
	Referencia         string
	PoliticaVersionRef string
	FuenteRef          string
	MinutosPrevistos   *int64
	Acreditada         bool
}

// Son hechos de la misma lectura. El ensayo no acredita concesiones ni programación.
type ConcesionPermisoSaldo struct {
	SolicitudRef       string
	ResolucionRef      string
	PermisoRef         string
	CatalogoVersionRef string
	ColectivoRef       string
	Desde              string
	Hasta              string
	Concedido          bool
	JornadaCompleta    *bool
}

type DiaSaldoPermisos struct {
	// SinTrabajo es un hecho explícito; cero minutos no demuestra ausencia de trabajo.
	SinTrabajo        *bool
	Fecha             string
	Programacion      *ProgramacionDiaPermisoSaldo
	TrabajadosMinutos *int64
	Completo          *bool
	SinAnomalias      *bool
	Permisos          []ConcesionPermisoSaldo
}

type EfectoPermisoSaldo struct {
	VersionProgramacionRef  string
	FuenteProgramacionRef   string
	Fecha                   string
	TrabajadosMinutos       *int64
	SaldoBaseMinutos        *int64
	PermisoComputadoMinutos *int64
	SaldoAjustadoMinutos    *int64
	Causa                   string
	ProgramacionRef         string
	SolicitudRef            string
	ResolucionRef           string
	ReglaRef                string
	FuenteRef               string
}

// CalcularEfectoPermisoSaldo conserva el trabajo registrado. Un crédito conocido
// se representa aparte; una incompatibilidad deja el ajuste ausente, nunca a cero.
func CalcularEfectoPermisoSaldo(d DiaSaldoPermisos, p PoliticaEfectosPermisoSaldo) (EfectoPermisoSaldo, error) {
	r := EfectoPermisoSaldo{Fecha: d.Fecha, TrabajadosMinutos: d.TrabajadosMinutos, Causa: "programacion_ausente"}
	if _, err := fechaPermiso(d.Fecha); err != nil || p.Validar() != nil || len(d.Permisos) > 10 || (d.TrabajadosMinutos != nil && (*d.TrabajadosMinutos < 0 || *d.TrabajadosMinutos > 48*60)) {
		return EfectoPermisoSaldo{}, ErrEfectoPermisoSaldoInvalido
	}
	for _, c := range d.Permisos {
		desde, e1 := fechaPermiso(c.Desde)
		hasta, e2 := fechaPermiso(c.Hasta)
		if e1 != nil || e2 != nil || hasta.Before(desde) || d.Fecha < c.Desde || d.Fecha > c.Hasta {
			return EfectoPermisoSaldo{}, ErrEfectoPermisoSaldoInvalido
		}
		for _, ref := range []string{c.SolicitudRef, c.ResolucionRef, c.PermisoRef, c.CatalogoVersionRef, c.ColectivoRef} {
			if ref != "" && !referenciaMarcaje(ref) {
				return EfectoPermisoSaldo{}, ErrEfectoPermisoSaldoInvalido
			}
		}
	}
	if d.Programacion == nil || !d.Programacion.Acreditada || d.Programacion.MinutosPrevistos == nil {
		return r, nil
	}
	j := d.Programacion
	if j.Fecha != d.Fecha || !referenciaMarcaje(j.Referencia) || !referenciaMarcaje(j.PoliticaVersionRef) || !referenciaMarcaje(j.FuenteRef) || *j.MinutosPrevistos < 0 || *j.MinutosPrevistos > 24*60 {
		return EfectoPermisoSaldo{}, ErrEfectoPermisoSaldoInvalido
	}
	r.ProgramacionRef, r.VersionProgramacionRef, r.FuenteProgramacionRef = j.Referencia, j.PoliticaVersionRef, j.FuenteRef
	if d.Completo == nil || !*d.Completo || d.SinAnomalias == nil || !*d.SinAnomalias || d.SinTrabajo == nil || d.TrabajadosMinutos == nil || d.Permisos == nil {
		r.Causa = "hechos_incompletos"
		return r, nil
	}
	base := *d.TrabajadosMinutos - *j.MinutosPrevistos
	r.SaldoBaseMinutos = &base
	if len(d.Permisos) == 0 {
		cero := int64(0)
		r.PermisoComputadoMinutos, r.SaldoAjustadoMinutos, r.Causa = &cero, &base, "sin_permisos"
		return r, nil
	}
	if len(d.Permisos) != 1 {
		r.Causa = "permisos_coincidentes"
		return r, nil
	}
	c := d.Permisos[0]
	r.SolicitudRef, r.ResolucionRef = c.SolicitudRef, c.ResolucionRef
	if c.PermisoRef == "" || c.CatalogoVersionRef == "" || c.ColectivoRef == "" {
		r.Causa = "concesion_incompleta"
		return r, nil
	}
	if !c.Concedido || c.SolicitudRef == "" || c.ResolucionRef == "" || c.JornadaCompleta == nil {
		r.Causa = "concesion_no_acreditada"
		return r, nil
	}
	if !*c.JornadaCompleta {
		r.Causa = "permiso_parcial"
		return r, nil
	}
	if !*d.SinTrabajo || *d.TrabajadosMinutos != 0 {
		r.Causa = "trabajo_concurrente"
		return r, nil
	}
	for _, regla := range p.Reglas {
		if regla.PermisoRef == c.PermisoRef && regla.CatalogoVersionRef == c.CatalogoVersionRef && regla.ColectivoRef == c.ColectivoRef {
			if d.Fecha < regla.VigenteDesde || d.Fecha >= regla.HastaExclusivo {
				r.Causa = "fuera_vigencia"
				return r, nil
			}
			credito, ajustado := *j.MinutosPrevistos, base+*j.MinutosPrevistos
			r.PermisoComputadoMinutos, r.SaldoAjustadoMinutos, r.Causa = &credito, &ajustado, "permiso_computado"
			r.ReglaRef, r.FuenteRef = regla.Referencia, regla.FuenteRef
			return r, nil
		}
	}
	r.Causa = "regla_no_disponible"
	return r, nil
}
