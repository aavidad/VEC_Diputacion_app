package domain

import (
	"errors"
	"regexp"
	"time"
)

var ErrSaldoEfectosC3Invalido = errors.New("cronos: efecto de saldo inválido")

const EfectoCreditoTiempoAutorizadoC3 = "credito_tiempo_autorizado"

type ReglaSaldoEfectosC3 struct {
	Referencia         string `json:"referencia"`
	PermisoRef         string `json:"permiso_ref"`
	CatalogoVersionRef string `json:"catalogo_version_ref"`
	CentroRef          string `json:"centro_ref"`
	ColectivoRef       string `json:"colectivo_ref"`
	Efecto             string `json:"efecto"`
	FuenteRef          string `json:"fuente_ref"`
	FuenteURL          string `json:"fuente_url"`
	FuenteVersionRef   string `json:"fuente_version_ref"`
	VigenteDesde       string `json:"vigente_desde"`
	HastaExclusivo     string `json:"hasta_exclusivo"`
	FuentePublicadaEn  string `json:"fuente_publicada_en"`
	FuenteConsultadaEn string `json:"fuente_consultada_en"`
}

type PoliticaSaldoEfectosC3 struct {
	Referencia  string                `json:"referencia"`
	Version     int64                 `json:"version"`
	SHA256      string                `json:"sha256"`
	ActoRef     string                `json:"acto_ref"`
	PublicadaEn time.Time             `json:"publicada_en"`
	Reglas      []ReglaSaldoEfectosC3 `json:"reglas"`
}

var horaSaldoC3 = regexp.MustCompile(`^(?:[01][0-9]|2[0-3]):[0-5][0-9]$`)

func (p PoliticaSaldoEfectosC3) Validar() error {
	if !referenciaMarcaje(p.Referencia) || p.Version < 1 || !HuellaEfectosValida(p.SHA256) || !referenciaMarcaje(p.ActoRef) ||
		p.PublicadaEn.IsZero() || p.PublicadaEn.Location() != time.UTC || p.Reglas == nil || len(p.Reglas) > 100 {
		return ErrSaldoEfectosC3Invalido
	}
	claves := map[[3]string]bool{}
	for _, r := range p.Reglas {
		desde, e1 := fechaPermiso(r.VigenteDesde)
		hasta, e2 := fechaPermiso(r.HastaExclusivo)
		publicada, e3 := fechaPermiso(r.FuentePublicadaEn)
		consultada, e4 := fechaPermiso(r.FuenteConsultadaEn)
		clave := [3]string{r.PermisoRef, r.CatalogoVersionRef, r.CentroRef + ":" + r.ColectivoRef}
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || consultada.Before(publicada) || consultada.After(p.PublicadaEn) || !hasta.After(desde) || claves[clave] ||
			!referenciaMarcaje(r.Referencia) || !referenciaMarcaje(r.PermisoRef) || !referenciaMarcaje(r.CatalogoVersionRef) ||
			!referenciaMarcaje(r.CentroRef) || !referenciaMarcaje(r.ColectivoRef) || !referenciaMarcaje(r.FuenteRef) || !referenciaMarcaje(r.FuenteVersionRef) ||
			r.Efecto != EfectoCreditoTiempoAutorizadoC3 || len(r.FuenteURL) > 512 || len(r.FuenteURL) < 9 || r.FuenteURL[:8] != "https://" {
			return ErrSaldoEfectosC3Invalido
		}
		claves[clave] = true
	}
	return nil
}

type ProgramacionSaldoEfectosC3 struct {
	Referencia   string    `json:"referencia"`
	FuenteRef    string    `json:"fuente_ref"`
	Version      int64     `json:"version"`
	Minutos      int64     `json:"minutos"`
	InicioUTC    time.Time `json:"inicio_utc"`
	FinUTC       time.Time `json:"fin_utc"`
	Fecha        string    `json:"fecha"`
	ColectivoRef string    `json:"colectivo_ref"`
	CentroRef    string    `json:"centro_ref"`
}

type PermisoConcedidoSaldoEfectosC3 struct {
	SolicitudRef           string    `json:"solicitud_ref"`
	ResolucionRef          string    `json:"resolucion_ref"`
	PermisoRef             string    `json:"permiso_ref"`
	CatalogoVersionRef     string    `json:"catalogo_version_ref"`
	ColectivoRef           string    `json:"colectivo_ref"`
	CentroRef              string    `json:"centro_ref"`
	Desde                  string    `json:"desde"`
	Hasta                  string    `json:"hasta"`
	HoraInicio             string    `json:"hora_inicio"`
	HoraFin                string    `json:"hora_fin"`
	CantidadMinutos        int64     `json:"cantidad_minutos"`
	ResolucionVersion      int64     `json:"resolucion_version"`
	ResolucionDecisionRef  string    `json:"resolucion_decision_ref"`
	ResolucionAuditoriaRef string    `json:"resolucion_auditoria_ref"`
	ResueltaEn             time.Time `json:"resuelta_en"`
	CatalogoFuenteRef      string    `json:"catalogo_fuente_ref"`
	CatalogoPublicadaEn    time.Time `json:"catalogo_publicada_en"`
	PendienteJustificar    bool      `json:"pendiente_justificar"`
}

type DiaSaldoEfectosC3 struct {
	Fecha            string
	ConocidoEn       time.Time
	Programacion     *ProgramacionSaldoEfectosC3
	TrabajadoMinutos int64
	SinTrabajo       bool
	Completo         bool
	Permisos         []PermisoConcedidoSaldoEfectosC3
}

type ResultadoSaldoEfectosC3 struct {
	Fecha, Causa, SolicitudRef, ResolucionRef, ReglaRef, FuenteRef string
	PoliticaRef, PoliticaSHA256                                    string
	PoliticaVersion, ResolucionVersion                             int64
	SaldoBaseMinutos, CreditoPermisoMinutos, SaldoAjustadoMinutos  *int64
}

// El crédito es un movimiento separado: jamás altera la programación ni los fichajes.
func CalcularSaldoEfectosC3(d DiaSaldoEfectosC3, p PoliticaSaldoEfectosC3, zona *time.Location) (ResultadoSaldoEfectosC3, error) {
	r := ResultadoSaldoEfectosC3{Fecha: d.Fecha, Causa: "fuente_no_determinable", PoliticaRef: p.Referencia, PoliticaVersion: p.Version, PoliticaSHA256: p.SHA256}
	if p.Validar() != nil || zona == nil || d.ConocidoEn.IsZero() || d.ConocidoEn.Location() != time.UTC ||
		len(d.Permisos) > 10 || d.TrabajadoMinutos < 0 || d.TrabajadoMinutos > 48*60 {
		return ResultadoSaldoEfectosC3{}, ErrSaldoEfectosC3Invalido
	}
	fecha, err := fechaPermiso(d.Fecha)
	if err != nil {
		return ResultadoSaldoEfectosC3{}, ErrSaldoEfectosC3Invalido
	}
	if p.PublicadaEn.After(d.ConocidoEn) {
		return r, nil
	}
	if !d.Completo || d.Programacion == nil || d.Permisos == nil {
		return r, nil
	}
	j := d.Programacion
	if j.Fecha != d.Fecha || j.ColectivoRef == "" || j.CentroRef == "" || j.Minutos < 0 || j.Minutos > 1440 || j.Version < 1 || !referenciaMarcaje(j.Referencia) || !referenciaMarcaje(j.FuenteRef) ||
		j.InicioUTC.IsZero() || j.FinUTC.IsZero() || j.InicioUTC.Location() != time.UTC || j.FinUTC.Location() != time.UTC ||
		!j.FinUTC.After(j.InicioUTC) || j.FinUTC.After(j.InicioUTC.Add(24*time.Hour)) || j.InicioUTC.In(zona).Format("2006-01-02") != d.Fecha {
		return ResultadoSaldoEfectosC3{}, ErrSaldoEfectosC3Invalido
	}
	base := d.TrabajadoMinutos - j.Minutos
	r.SaldoBaseMinutos = &base
	if len(d.Permisos) == 0 {
		cero := int64(0)
		r.Causa = "sin_permisos"
		r.CreditoPermisoMinutos = &cero
		r.SaldoAjustadoMinutos = &base
		return r, nil
	}
	if len(d.Permisos) != 1 {
		r.Causa = "permisos_coincidentes"
		return r, nil
	}
	c := d.Permisos[0]
	r.SolicitudRef, r.ResolucionRef, r.ResolucionVersion = c.SolicitudRef, c.ResolucionRef, c.ResolucionVersion
	if c.SolicitudRef == "" || c.ResolucionRef == "" || c.ResolucionVersion < 2 || c.ColectivoRef != j.ColectivoRef || c.CentroRef != j.CentroRef ||
		!referenciaMarcaje(c.ResolucionDecisionRef) || !referenciaMarcaje(c.ResolucionAuditoriaRef) ||
		c.ResueltaEn.IsZero() || c.ResueltaEn.Location() != time.UTC || c.ResueltaEn.After(d.ConocidoEn) ||
		!referenciaMarcaje(c.CatalogoFuenteRef) || c.CatalogoPublicadaEn.IsZero() || c.CatalogoPublicadaEn.Location() != time.UTC ||
		c.CatalogoPublicadaEn.After(c.ResueltaEn) || c.CatalogoPublicadaEn.After(d.ConocidoEn) ||
		!referenciaMarcaje(c.SolicitudRef) || !referenciaMarcaje(c.ResolucionRef) ||
		!referenciaMarcaje(c.PermisoRef) || !referenciaMarcaje(c.CatalogoVersionRef) || !referenciaMarcaje(c.ColectivoRef) ||
		c.Desde != d.Fecha || c.Hasta != d.Fecha || !horaSaldoC3.MatchString(c.HoraInicio) || !horaSaldoC3.MatchString(c.HoraFin) || c.CantidadMinutos < 1 || c.CantidadMinutos > 1440 {
		r.Causa = "concesion_incompleta"
		return r, nil
	}
	var regla *ReglaSaldoEfectosC3
	for i := range p.Reglas {
		x := &p.Reglas[i]
		if x.PermisoRef == c.PermisoRef && x.CatalogoVersionRef == c.CatalogoVersionRef && x.CentroRef == j.CentroRef && x.ColectivoRef == c.ColectivoRef && d.Fecha >= x.VigenteDesde && d.Fecha < x.HastaExclusivo {
			regla = x
			break
		}
	}
	if regla == nil {
		r.Causa = "regla_no_disponible"
		return r, nil
	}
	r.ReglaRef, r.FuenteRef = regla.Referencia, regla.FuenteRef
	if !d.SinTrabajo || d.TrabajadoMinutos != 0 {
		r.Causa = "trabajo_concurrente"
		return r, nil
	}
	// Un cambio de hora exige una regla explícita de tramos para evitar asumir duración civil.
	diaLocal := time.Date(fecha.Year(), fecha.Month(), fecha.Day(), 0, 0, 0, 0, zona)
	if diaLocal.AddDate(0, 0, 1).Sub(diaLocal) != 24*time.Hour {
		r.Causa = "tramo_no_determinable"
		return r, nil
	}
	inicio, e1 := time.ParseInLocation("2006-01-02 15:04", d.Fecha+" "+c.HoraInicio, zona)
	fin, e2 := time.ParseInLocation("2006-01-02 15:04", d.Fecha+" "+c.HoraFin, zona)
	if e1 != nil || e2 != nil || !fin.After(inicio) || inicio.Format("15:04") != c.HoraInicio || fin.Format("15:04") != c.HoraFin ||
		int64(fin.Sub(inicio)/time.Minute) != c.CantidadMinutos || inicio.Before(j.InicioUTC) || fin.After(j.FinUTC) || c.CantidadMinutos > j.Minutos {
		r.Causa = "tramo_no_determinable"
		return r, nil
	}
	credito := c.CantidadMinutos
	ajustado := base + credito
	r.Causa = "permiso_computado"
	r.CreditoPermisoMinutos = &credito
	r.SaldoAjustadoMinutos = &ajustado
	return r, nil
}
