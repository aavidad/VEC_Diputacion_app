package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionExportarInformePropioC12    = "cronos.informe.propio.exportar"
	AccionExportarInformeEquipoC12    = "cronos.informe.equipo.exportar"
	AudienciaExportarInformePropioC12 = "vec_cronos_v1.informe_propio.exportar.v1"
	AudienciaExportarInformeEquipoC12 = "vec_cronos_v1.informe_equipo.exportar.v1"
	FinalidadExportarInformePropioC12 = "exportar_informe_propio"
	FinalidadExportarInformeEquipoC12 = "exportar_informe_equipo"
)

func ContratoInformeC12(alcance ports.AlcanceInformeC12) (accion, finalidad, audiencia string, ok bool) {
	switch alcance {
	case ports.InformeC12Empleado:
		return AccionExportarInformePropioC12, FinalidadExportarInformePropioC12, AudienciaExportarInformePropioC12, true
	case ports.InformeC12Jefatura, ports.InformeC12RRHH:
		return AccionExportarInformeEquipoC12, FinalidadExportarInformeEquipoC12, AudienciaExportarInformeEquipoC12, true
	default:
		return "", "", "", false
	}
}

type ServicioInformeC12 struct {
	repo     ports.RepositorioInformeC12
	renderer ports.RenderizadorInformeC12
	reloj    ports.Reloj
	zona     *time.Location
}

func NuevoServicioInformeC12(repo ports.RepositorioInformeC12, renderer ports.RenderizadorInformeC12, reloj ports.Reloj, zona *time.Location) (*ServicioInformeC12, error) {
	if repo == nil || renderer == nil || reloj == nil || zona == nil || (zona.String() != domain.ZonaSaldoPeninsula && zona.String() != domain.ZonaSaldoCanarias) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ServicioInformeC12{repo, renderer, reloj, zona}, nil
}
func (s *ServicioInformeC12) ExportarInforme(ctx context.Context, o ports.OrdenInformeC12, p ports.PeticionInformeC12) (ports.ResultadoInformeC12, error) {
	cero := ports.ResultadoInformeC12{}
	if s == nil || s.repo == nil || s.renderer == nil || s.reloj == nil || s.zona == nil || ctx == nil || ctx.Err() != nil {
		return cero, ports.ErrDependenciaNoDisponible
	}
	actor, err := o.ContextoActor()
	if err != nil || o.Proveedor() == nil {
		return cero, ports.ErrDependenciaNoDisponible
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	ahora := s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if err != nil || len(empleados) != 1 || ahora.IsZero() || !actor.Instantanea.VigenteEn(ahora) {
		return cero, ports.ErrDependenciaNoDisponible
	}
	_, _, _, ok := ContratoInformeC12(o.Alcance())
	if !ok {
		return cero, ports.ErrInformeC12Invalido
	}
	if p.Tipo != domain.InformeSaldo && p.Tipo != domain.InformeMovimientos && p.Tipo != domain.InformePermisos {
		return cero, ports.ErrInformeC12Invalido
	}
	if p.Formato != domain.InformePDF && p.Formato != domain.InformeCSV {
		return cero, ports.ErrInformeC12Invalido
	}
	desde, e1 := time.Parse("2006-01-02", p.Desde)
	hasta, e2 := time.Parse("2006-01-02", p.Hasta)
	if e1 != nil || e2 != nil || desde.Format("2006-01-02") != p.Desde || hasta.Format("2006-01-02") != p.Hasta || hasta.Before(desde) || hasta.Sub(desde) > 31*24*time.Hour {
		return cero, ports.ErrInformeC12Invalido
	}
	if (o.Alcance() == ports.InformeC12Empleado && p.EquipoRef != "") || (o.Alcance() != ports.InformeC12Empleado && p.EquipoRef == "") {
		return cero, ports.ErrInformeC12Invalido
	}
	var foto *domain.FotoEquipoAutoritativa
	if p.EquipoRef != "" {
		f, err := s.repo.FotoEquipo(ctx, p.EquipoRef)
		if err != nil {
			return cero, err
		}
		if f.EquipoRef != p.EquipoRef || f.Desde > p.Desde || f.HastaExclusivo <= p.Hasta {
			return cero, ports.ErrDependenciaNoDisponible
		}
		foto = &f
	}
	politica, err := s.repo.PoliticaInforme(ctx, string(o.Alcance()), p.Tipo, p.Formato)
	if err != nil {
		return cero, err
	}
	if !referenciaExportacionValida(politica.Referencia) || politica.Version < 1 || !huellaExportacionValida(politica.SHA256) {
		return cero, ports.ErrDependenciaNoDisponible
	}
	material := domain.MaterialExportacionCronos{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleados[0], Foto: foto, Alcance: string(o.Alcance()), Tipo: p.Tipo, Formato: p.Formato, Desde: p.Desde, Hasta: p.Hasta, ZonaHoraria: s.zona.String(), Idioma: p.Idioma, PoliticaRef: politica.Referencia, PoliticaVersion: politica.Version, PoliticaSHA256: politica.SHA256, ClaveOperacion: p.ClaveOperacion, Fase: "preparacion"}
	if _, err := material.Canonico(); err != nil {
		return cero, ports.ErrInformeC12Invalido
	}
	preparacion, err := s.repo.PrepararInforme(ctx, o, material)
	if err != nil {
		return cero, err
	}
	if preparacion.PoliticaRef != politica.Referencia || preparacion.PoliticaVersion != politica.Version || preparacion.PoliticaSHA256 != politica.SHA256 || preparacion.Tipo != p.Tipo || preparacion.Formato != p.Formato || preparacion.EmpleadoRef != empleados[0] || preparacion.Desde != p.Desde || preparacion.Hasta != p.Hasta || preparacion.Idioma != p.Idioma || !huellaExportacionValida(preparacion.FuenteSHA256) || !referenciaExportacionValida(preparacion.PreparacionRef) || len(preparacion.Campos) == 0 || len(preparacion.Campos) > 20 || len(preparacion.Datos) > 10000 {
		return cero, ports.ErrDependenciaNoDisponible
	}
	if (foto == nil && preparacion.EquipoRef != "") || (foto != nil && preparacion.EquipoRef != foto.EquipoRef) || !camposInformeC12Validos(p.Tipo, foto != nil, preparacion.Campos) {
		return cero, ports.ErrDependenciaNoDisponible
	}
	for _, fila := range preparacion.Datos {
		if len(fila) != len(preparacion.Campos) {
			return cero, ports.ErrDependenciaNoDisponible
		}
	}
	bytes, err := s.renderer.RenderizarInforme(ctx, p.Formato, preparacion)
	if err != nil {
		return cero, err
	}
	if len(bytes) == 0 || len(bytes) > 2*1024*1024 {
		return cero, ports.ErrDependenciaNoDisponible
	}
	contenido := append([]byte(nil), bytes...)
	sha := sha256.Sum256(contenido)
	huella := hex.EncodeToString(sha[:])
	if ctx.Err() != nil || !actor.Instantanea.VigenteEn(s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)) {
		clear(contenido)
		return cero, ports.ErrDependenciaNoDisponible
	}
	confirmar := material
	confirmar.Fase = "confirmacion"
	confirmar.PreparacionRef = preparacion.PreparacionRef
	confirmar.FuenteSHA256 = preparacion.FuenteSHA256
	confirmar.DocumentoSHA256 = huella
	confirmar.Tamano = int64(len(contenido))
	if _, err := confirmar.Canonico(); err != nil {
		clear(contenido)
		return cero, ports.ErrDependenciaNoDisponible
	}
	recibo, err := s.repo.ConfirmarInforme(ctx, o, confirmar, huella, int64(len(contenido)))
	if err != nil {
		clear(contenido)
		return cero, err
	}
	if recibo.PreparacionRef != preparacion.PreparacionRef || recibo.DocumentoSHA256 != huella || recibo.Tamano != int64(len(contenido)) || !referenciaExportacionValida(recibo.ReciboRef) || !referenciaExportacionValida(recibo.AuditoriaRef) || recibo.ConfirmadaUTC.IsZero() || recibo.ConfirmadaUTC.Location() != time.UTC || recibo.ConfirmadaUTC.After(s.reloj.AhoraUTC().UTC().Add(time.Minute)) || !actor.Instantanea.VigenteEn(recibo.ConfirmadaUTC) {
		clear(contenido)
		return cero, ports.ErrDependenciaNoDisponible
	}
	return ports.ResultadoInformeC12{Contenido: contenido, Formato: p.Formato, Recibo: recibo}, nil
}

var _ ports.CasoUsoInformeC12 = (*ServicioInformeC12)(nil)

func camposInformeC12Validos(tipo domain.TipoInformeCronos, equipo bool, campos []string) bool {
	permitidos := map[string]bool{"empleado_etiqueta": equipo}
	switch tipo {
	case domain.InformeSaldo:
		for _, v := range []string{"fecha", "previstos_minutos", "trabajados_minutos", "saldo_minutos", "estado"} {
			permitidos[v] = true
		}
	case domain.InformeMovimientos:
		for _, v := range []string{"fecha", "movimiento", "instante_utc"} {
			permitidos[v] = true
		}
	case domain.InformePermisos:
		for _, v := range []string{"desde", "hasta", "estado"} {
			permitidos[v] = true
		}
	default:
		return false
	}
	vistos := map[string]bool{}
	for _, v := range campos {
		if !permitidos[v] || vistos[v] {
			return false
		}
		vistos[v] = true
	}
	return len(campos) > 0
}
