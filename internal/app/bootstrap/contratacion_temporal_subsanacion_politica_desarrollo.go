package bootstrap

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type fuentePoliticaSubsanacionReparosDesarrollo struct {
	soporte       *soporteAltaContratacionTemporalDesarrollo
	configuracion configuracionPoliticaSubsanacionReparosDesarrollo
}

func (f fuentePoliticaSubsanacionReparosDesarrollo) ResolverPoliticaSubsanacionReparo(ctx context.Context, s ports.SolicitudResolverPoliticaSubsanacionReparo) (ports.PoliticaSubsanacionReparo, error) {
	return f.configuracion.resolver(ctx, s)
}
func (f fuentePoliticaSubsanacionReparosDesarrollo) ResolverContextoCanalSubsanacionReparos(ctx context.Context) (httpinterno.ContextoCanalSubsanacionReparos, error) {
	if f.soporte == nil {
		return httpinterno.ContextoCanalSubsanacionReparos{}, errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	capacidad, valida := f.soporte.capacidadValida(ctx)
	ahora := f.soporte.reloj.Ahora()
	if !valida || capacidad.ruta != httpinterno.RutaSubsanacionReparos ||
		!domain.InstanteUTCCanonico(ahora) ||
		!domain.InstanteUTCCanonico(capacidad.certificadoVerificadoEn) ||
		!domain.InstanteUTCCanonico(capacidad.certificadoValidoHasta) ||
		capacidad.certificadoVerificadoEn.After(ahora) ||
		!ahora.Before(capacidad.certificadoValidoHasta) {
		return httpinterno.ContextoCanalSubsanacionReparos{}, ports.ErrAutorizacionDenegada
	}
	v, err := f.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return httpinterno.ContextoCanalSubsanacionReparos{}, ports.ErrAutorizacionDenegada
	}
	return httpinterno.ContextoCanalSubsanacionReparos{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef, OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo}, nil
}

func (f fuentePoliticaSubsanacionReparosDesarrollo) configurar(alta *dependenciasAltaContratacionTemporalDesarrollo) error {
	if f.soporte == nil || alta == nil || alta.postgresql.gobierno == nil || f.configuracion.MotivoAutorizacion.Validar() != nil {
		return errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	v, err := f.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	i, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, f.soporte.reloj.Ahora(), "tecnico_rrhh_subsanacion_desarrollo", "Tecnico RRHH de subsanacion de desarrollo", "asignacion-rrhh-subsanacion-desarrollo-no-autoritativa", []vecdomain.ConcesionRol{{Accion: string(domain.AccionRegistrarSubsanacionReparo), ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoSubsanacionReparo, Finalidades: []string{ports.FinalidadRegistrarSubsanacionReparo}, GarantiaMinima: vecdomain.AuthAssuranceHigh}}, []vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		return errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(f.soporte.reloj.Ahora())
	if !vigente || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []vecdomain.ReferenciaEntradaCatalogo{f.configuracion.MotivoAutorizacion}, desde) != nil {
		return errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	f.soporte.mu.Lock()
	f.soporte.instantaneaSubsanacion = i
	f.soporte.motivoSubsanacion = f.configuracion.MotivoAutorizacion
	f.soporte.mu.Unlock()
	return nil
}

var errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible = errors.New("contratacion temporal: fuente de politica de subsanacion no disponible")

// Archivo privado canónico (sin actor, perfil ni secreto):
// {"definicion_ref":"...","definicion_version":1,"definicion_huella_sha256":"...",
//
//	"motivo_autorizacion":{"catalogo_id":"...","catalogo_version":1,"catalogo_huella_sha256":"...","entrada_clave":"..."}}
//
// La identidad procede exclusivamente del contexto autenticado del canal.
type configuracionPoliticaSubsanacionReparosDesarrollo struct {
	DefinicionRef          string                              `json:"definicion_ref"`
	DefinicionVersion      uint64                              `json:"definicion_version"`
	DefinicionHuellaSHA256 string                              `json:"definicion_huella_sha256"`
	MotivoAutorizacion     vecdomain.ReferenciaEntradaCatalogo `json:"motivo_autorizacion"`
}

func cargarConfiguracionPoliticaSubsanacionReparosDesarrollo(cfg config.Config) (configuracionPoliticaSubsanacionReparosDesarrollo, error) {
	var vacia configuracionPoliticaSubsanacionReparosDesarrollo
	ruta := strings.TrimSpace(cfg.ContratacionTemporalSubsanacionPoliticaFile)
	if ruta == "" {
		return vacia, errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	archivo, err := os.Open(ruta)
	if err != nil {
		return vacia, errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	defer archivo.Close()
	contenido, err := io.ReadAll(io.LimitReader(archivo, 16*1024+1))
	if err != nil || len(contenido) == 0 || len(contenido) > 16*1024 {
		return vacia, errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	defer borrarBytes(contenido)
	var politica configuracionPoliticaSubsanacionReparosDesarrollo
	dec := json.NewDecoder(strings.NewReader(string(contenido)))
	dec.DisallowUnknownFields()
	if dec.Decode(&politica) != nil || dec.Decode(&struct{}{}) != io.EOF || !domain.ReferenciaOpacaValida(politica.DefinicionRef) || !ports.VersionOperacionAnalisisValida(politica.DefinicionVersion) || politica.MotivoAutorizacion.Validar() != nil || len(politica.DefinicionHuellaSHA256) != 64 || politica.DefinicionHuellaSHA256 != strings.ToLower(politica.DefinicionHuellaSHA256) || politica.DefinicionHuellaSHA256 == strings.Repeat("0", 64) {
		return vacia, errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	if _, err := hex.DecodeString(politica.DefinicionHuellaSHA256); err != nil {
		return vacia, errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	return politica, nil
}

func (c configuracionPoliticaSubsanacionReparosDesarrollo) resolver(ctx context.Context, s ports.SolicitudResolverPoliticaSubsanacionReparo) (ports.PoliticaSubsanacionReparo, error) {
	if ctx == nil || !domain.ReferenciaOpacaValida(s.OrganizacionRef) || !domain.ReferenciaOpacaValida(s.ExpedienteRef) || !domain.ReferenciaOpacaValida(s.ActorRef) || !domain.ReferenciaOpacaValida(s.PerfilRef) || !domain.ReferenciaOpacaValida(s.RetornoRef) || s.VersionEsperada == 0 || !domain.InstanteUTCCanonico(s.Instante) {
		return ports.PoliticaSubsanacionReparo{}, errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	p := ports.PoliticaSubsanacionReparo{DefinicionRef: c.DefinicionRef, DefinicionVersion: c.DefinicionVersion, DefinicionHuellaSHA256: c.DefinicionHuellaSHA256, MotivoAutorizacion: c.MotivoAutorizacion, Accion: domain.AccionRegistrarSubsanacionReparo, Finalidad: domain.ClaveCatalogo(ports.FinalidadRegistrarSubsanacionReparo), EvaluadaEn: s.Instante, ValidaHasta: s.Instante.Add(5 * time.Minute)}
	if !p.ValidaPara(s, s.Instante) {
		return ports.PoliticaSubsanacionReparo{}, errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	return p, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) solicitudAutorizacionSubsanacionReparosValida(datos vecdomain.DatosSolicitudAutorizacionLigadaV3) bool {
	return s != nil && datos.Accion == string(domain.AccionRegistrarSubsanacionReparo) && datos.ReferenciaMotivo == s.motivoSubsanacion && datos.Recurso.ModuloID == ports.ModuloContratacion && datos.Recurso.Tipo == ports.TipoRecursoSubsanacionReparo && datos.Finalidad == ports.FinalidadRegistrarSubsanacionReparo && len(datos.Recurso.Ambitos) == 4 && datos.Recurso.Ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo && datos.Recurso.Ambitos["expediente_ref"] == datos.Recurso.Referencia && datos.Recurso.Ambitos["fase_previa"] == string(domain.FaseSubsanacionUnidad) && datos.Recurso.Ambitos["estado_previo"] == string(domain.EstadoIncidencia)
}
