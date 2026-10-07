package bootstrap

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
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
	operativo, err := f.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return httpinterno.ContextoCanalSubsanacionReparos{}, ports.ErrAutorizacionDenegada
	}
	v, err := operativo.Vinculo.Datos()
	if err != nil {
		return httpinterno.ContextoCanalSubsanacionReparos{}, ports.ErrAutorizacionDenegada
	}
	return httpinterno.ContextoCanalSubsanacionReparos{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef, OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo}, nil
}

// configurar publica el motivo de la política y compone el perfil fijo de la
// subsanación: la organización y el par fase/estado del catálogo (c23), sin
// expediente. Se publica una vez si falta y después solo se consume.
func (f fuentePoliticaSubsanacionReparosDesarrollo) configurar(alta *dependenciasAltaContratacionTemporalDesarrollo,
	aprobacion aprobacionProvisionPerfilesRRHHDesarrollo) error {
	if f.soporte == nil || alta == nil || alta.postgresql.gobierno == nil || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(f.configuracion.MotivoAutorizacion) {
		log.Print("contratacion temporal: subsanacion no disponible; etapa=fuente.dependencias")
		return errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	v, err := f.soporte.contexto.Vinculo.Datos()
	if err != nil {
		log.Print("contratacion temporal: subsanacion no disponible; etapa=fuente.vinculo")
		return errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	fase, ok := f.soporte.opcionesCatalogo.faseOperacionVigente(operacionFaseSubsanacionCT)
	plantilla := func(principalID, perfilRef string) (vecdomain.InstantaneaAutorizacion, error) {
		return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principalID, perfilRef, f.soporte.reloj.Ahora(),
			"tecnico_rrhh_subsanacion_desarrollo", "Tecnico RRHH de subsanacion de desarrollo",
			"asignacion-rrhh-subsanacion-desarrollo-no-autoritativa",
			[]vecdomain.ConcesionRol{{Accion: string(domain.AccionRegistrarSubsanacionReparo), ModuloID: ports.ModuloContratacion,
				TipoRecurso: ports.TipoRecursoSubsanacionReparo, Finalidades: []string{ports.FinalidadRegistrarSubsanacionReparo},
				GarantiaMinima: vecdomain.AuthAssuranceHigh}},
			fase.ambitosPerfil(organizacionAltaContratacionTemporalDesarrollo))
	}
	i, err := plantilla(v.PrincipalID, v.PerfilActivoRef)
	if err != nil || !ok {
		log.Print("contratacion temporal: subsanacion no disponible; etapa=fuente.instantanea")
		return errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
	defer cancelar()
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(f.soporte.reloj.Ahora())
	if !vigente || publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []vecdomain.ReferenciaEntradaCatalogo{f.configuracion.MotivoAutorizacion}, desde) != nil {
		log.Print("contratacion temporal: subsanacion no disponible; etapa=fuente.catalogo")
		return errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	principal := vecdomain.Principal{ID: f.soporte.principalID, Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": f.soporte.certificadoSHA256}}
	fijo, err := nuevoPerfilFijoCTDesarrollo(principal, f.soporte.contexto, f.soporte.reloj.Ahora(), clavePerfilFijoSubsanacionCTDesarrollo,
		[]string{httpinterno.RutaSubsanacionReparos}, plantilla)
	if err != nil {
		log.Print("contratacion temporal: subsanacion no disponible; etapa=fuente.perfil_fijo")
		return errFuentePoliticaSubsanacionReparosDesarrolloNoDisponible
	}
	f.soporte.mu.Lock()
	f.soporte.instantaneaSubsanacion = i
	f.soporte.motivoSubsanacion = f.configuracion.MotivoAutorizacion
	f.soporte.mu.Unlock()
	if err := f.soporte.registrarPerfilFijoCTDesarrollo(fijo); err != nil {
		return err
	}
	return asegurarPerfilesFijosCTDesarrollo(ctx, alta.postgresql.gobierno, f.soporte, aprobacion, fijo)
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
	if dec.Decode(&politica) != nil || dec.Decode(&struct{}{}) != io.EOF || !domain.ReferenciaOpacaValida(politica.DefinicionRef) || !ports.VersionOperacionAnalisisValida(politica.DefinicionVersion) || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(politica.MotivoAutorizacion) || len(politica.DefinicionHuellaSHA256) != 64 || politica.DefinicionHuellaSHA256 != strings.ToLower(politica.DefinicionHuellaSHA256) || politica.DefinicionHuellaSHA256 == strings.Repeat("0", 64) {
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

// solicitudAutorizacionSubsanacionReparosValida: el expediente va en la
// referencia del recurso; los ámbitos son la organización y el par
// fase/estado previo que admite el catálogo (c23).
func (s *soporteAltaContratacionTemporalDesarrollo) solicitudAutorizacionSubsanacionReparosValida(datos vecdomain.DatosSolicitudAutorizacionLigadaV3) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	motivo := s.motivoSubsanacion
	s.mu.Unlock()
	fase, ok := s.opcionesCatalogo.faseOperacionVigente(operacionFaseSubsanacionCT)
	return ok && datos.Accion == string(domain.AccionRegistrarSubsanacionReparo) && datos.ReferenciaMotivo == motivo &&
		datos.Recurso.ModuloID == ports.ModuloContratacion && datos.Recurso.Tipo == ports.TipoRecursoSubsanacionReparo &&
		datos.Finalidad == ports.FinalidadRegistrarSubsanacionReparo && datos.Recurso.Referencia != "" &&
		len(datos.Recurso.Ambitos) == 3 && datos.Recurso.Ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		fase.admite(domain.ClaveFase(datos.Recurso.Ambitos["fase_previa"]), domain.EstadoOperativo(datos.Recurso.Ambitos["estado_previo"]))
}
