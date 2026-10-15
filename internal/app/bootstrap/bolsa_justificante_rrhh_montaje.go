package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	bolsapg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	bolsapuertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridad "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	docaut "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docpg "vec-diputacion-granada/internal/vec/documentos/adapters/postgres"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const loginJustificanteRenunciaRRHH = "vec_interno_documentos_bolsa_rrhh"
const rolJustificanteRenunciaRRHH = "vec_documentos_bolsa_rrhh"
const rutaConfiguracionJustificanteRenunciaRRHH = "identidad/bolsa-justificantes-rrhh.json"

type configuracionJustificanteRenunciaRRHH struct {
	Version       int                                 `json:"version"`
	Superficie    core.SuperficieAutenticacionActorV1 `json:"superficie"`
	DSNDocumentos string                              `json:"dsn_documentos"`
}

func (configuracionJustificanteRenunciaRRHH) String() string {
	return "[CONFIGURACION PRIVADA BOLSA RRHH]"
}
func (configuracionJustificanteRenunciaRRHH) GoString() string {
	return "[CONFIGURACION PRIVADA BOLSA RRHH]"
}
func (configuracionJustificanteRenunciaRRHH) MarshalJSON() ([]byte, error) {
	return []byte(`{"configuracion":"privada"}`), nil
}
func (configuracionJustificanteRenunciaRRHH) LogValue() slog.Value {
	return slog.StringValue("[CONFIGURACION PRIVADA BOLSA RRHH]")
}

func leerConfiguracionJustificanteRenunciaRRHH(cfg config.Config) (configuracionJustificanteRenunciaRRHH, error) {
	var vacia configuracionJustificanteRenunciaRRHH
	b, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir,
		rutaConfiguracionJustificanteRenunciaRRHH), 16<<10)
	if err != nil || validarClavesJSONUnicas(b) != nil {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	defer borrarBytes(b)
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var c configuracionJustificanteRenunciaRRHH
	var extra any
	if d.Decode(&c) != nil || !errors.Is(d.Decode(&extra), io.EOF) ||
		c.Version != 1 || c.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 ||
		c.DSNDocumentos == "" || strings.TrimSpace(c.DSNDocumentos) != c.DSNDocumentos {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	return c, nil
}

type proveedorIdentidadJustificanteRenunciaRRHH struct {
	sesion resolutorContextoSeguridadComunDesarrollo
}

func (p proveedorIdentidadJustificanteRenunciaRRHH) PrepararIdentidadJustificanteRenunciaRRHH(r *http.Request) (
	bolsahttp.IdentidadJustificanteRenunciaRRHH, error) {
	var vacia bolsahttp.IdentidadJustificanteRenunciaRRHH
	if r == nil || r.URL == nil || r.Context().Err() != nil {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	if p.sesion == nil {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	sesion, err := p.sesion.ResolverContexto(r.Context())
	if err != nil {
		return vacia, err
	}
	vinculo, actor := sesion.Vinculo, sesion.Resultado
	datos, err := vinculo.Datos()
	if err != nil || actor.Validar() != nil || vinculo.ValidarPara(actor) != nil ||
		!vinculo.VigenteEn(relojContratacionTemporalDesarrollo{}.Ahora(), actor) || datos.CuentaPrivilegiada ||
		datos.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 ||
		datos.CuentaRef != actor.Contexto.Instantanea.CuentaRef ||
		datos.PerfilActivoRef != actor.Contexto.PerfilActivoRef ||
		actor.Contexto.PersonaRef == "" || actor.Contexto.PersonaRef != actor.Contexto.Instantanea.PersonaRef {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	corr, err := core.GenerarReferenciaCorrelacionAutorizacionV2(r.Context(),
		seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	valor, err := corr.ValorCanonico()
	if err != nil {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	return bolsahttp.IdentidadJustificanteRenunciaRRHH{Actor: actor, Vinculo: vinculo,
		CorrelacionRef: valor}, nil
}

// Esta sesión solo acepta la capacidad mTLS que selló la frontera interna de
// CT. El soporte nominal selecciona su propia sesión y perfil activo; la
// configuración documental no elige ni persona ni perfil.
type sesionTecnicaJustificanteRenunciaRRHH struct {
	autoridad *autoridadConsultasContratacionTemporalDesarrollo
}

func rutaJustificanteBolsaRRHHDesarrollo(ruta string) bool {
	return ruta == bolsahttp.RutaJustificanteRenunciaRRHH
}

func (s sesionTecnicaJustificanteRenunciaRRHH) ResolverContexto(ctx context.Context) (contextoSeguridadComunDesarrollo, error) {
	var vacia contextoSeguridadComunDesarrollo
	a := s.autoridad
	if a == nil || a.sello == nil || ctx == nil || ctx.Err() != nil {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	c, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	frontera, fronteraValida := fronteraSeguridadComunDesdeContexto(ctx)
	if !ok || c.sello != a.sello || c.metodo != http.MethodGet ||
		!rutaJustificanteBolsaRRHHDesarrollo(c.ruta) || c.contextoOperacion == nil ||
		!fronteraValida || frontera.metodo != http.MethodGet || frontera.ruta != c.ruta ||
		!frontera.catalogo.mismaInstancia(a.fronterasSeguridadComun) ||
		!principalContratacionTemporalDesarrolloValido(c.principal) ||
		c.certificadoVerificadoEn.IsZero() || c.certificadoValidoHasta.IsZero() {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	ahora := relojContratacionTemporalDesarrollo{}.Ahora()
	if c.certificadoVerificadoEn.After(ahora) || !ahora.Before(c.certificadoValidoHasta) {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	soporte := a.soporteDocumentosBolsa
	if !soporteDocumentoCoincideConCapacidad(soporte, c) {
		soporte = a.soportesLectoresDocumentosBolsa[c.principal.ID]
		if soporte == nil || !soporte.lectorConsultasRRHH || !soporte.tecnicoConsultaRRHH ||
			!soporteDocumentoCoincideConCapacidad(soporte, c) {
			return vacia, docports.ErrSubidaLocalDenegada
		}
	}
	resultado, err := soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		if errors.Is(err, ctports.ErrConsultaRRHHNoDisponible) ||
			errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return vacia, docports.ErrSubidaLocalNoDisponible
		}
		return vacia, docports.ErrSubidaLocalDenegada
	}
	if resultado.Resultado.Validar() != nil || resultado.Vinculo.ValidarPara(resultado.Resultado) != nil ||
		!frontera.descriptor.admitePerfil(resultado.Resultado.Contexto.PerfilActivoRef) {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	return contextoSeguridadComunDesarrollo{Vinculo: resultado.Vinculo, Resultado: resultado.Resultado}, nil
}

type vinculoJustificanteRenunciaRRHHFijado struct {
	identidad bolsahttp.IdentidadJustificanteRenunciaRRHH
}

func (v vinculoJustificanteRenunciaRRHHFijado) VinculoCargaLocalCertificado(ctx context.Context,
	a core.ResultadoContextoActorRegistradoV2) (core.VinculoAutenticacionActorV2, error) {
	if ctx == nil || ctx.Err() != nil || a.Validar() != nil ||
		v.identidad.Actor.RegistroContextoRef != a.RegistroContextoRef ||
		v.identidad.Actor.HuellaSHA256 != a.HuellaSHA256 || v.identidad.Vinculo.ValidarPara(a) != nil {
		return core.VinculoAutenticacionActorV2{}, docports.ErrSubidaLocalDenegada
	}
	return v.identidad.Vinculo, nil
}

type emisorJustificanteRenunciaRRHH struct {
	emisor *emisorMaterialRenovableCTDesarrollo
}

func (e emisorJustificanteRenunciaRRHH) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context,
	s core.SolicitudAutorizacionLigadaV3, a core.ResultadoContextoActorRegistradoV2) (
	core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	d, err := s.Datos()
	if err != nil || e.emisor == nil || d.Accion != docports.AccionBolsaJustificanteRenunciaDescargarRRHH {
		return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			nil, docports.ErrSubidaLocalDenegada
	}
	return e.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, a)
}

type consultorJustificanteRenunciaRRHH struct {
	resolutor  bolsapuertos.ResolutorConsultaJustificanteRenunciaRRHH
	registro   docports.RegistroOriginalSubidaLocalRRHH
	emisor     emisorJustificanteRenunciaRRHH
	motivo     core.ReferenciaEntradaCatalogo
	almacen    vecports.AlmacenObjetos
	seudonimos vecports.SeudonimizadorSujetoAlmacen
	reloj      vecports.Reloj
}

func (c *consultorJustificanteRenunciaRRHH) AbrirJustificanteRenunciaRRHH(ctx context.Context,
	identidad bolsahttp.IdentidadJustificanteRenunciaRRHH, bolsa, participacion string) (io.ReadCloser, error) {
	if c == nil || c.resolutor == nil || c.registro == nil || ctx == nil || ctx.Err() != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	proyeccion, err := c.resolutor.ResolverConsultaJustificanteRenunciaRRHH(ctx,
		bolsapuertos.SolicitudConsultaJustificanteRenunciaRRHH{ActorRegistradoV2: identidad.Actor,
			VinculoV2: identidad.Vinculo, BolsaRef: bolsa, ParticipacionRef: participacion})
	if err != nil {
		return nil, err
	}
	autoridad, err := docaut.NuevaAutoridadSubidaLocalBolsa(c.emisor,
		vinculoJustificanteRenunciaRRHHFijado{identidad}, c.motivo, c.reloj)
	if err != nil {
		return nil, err
	}
	servicio, err := docapp.NuevoServicioOriginalSubidaLocalRRHH(docapp.DependenciasOriginalSubidaLocalRRHH{
		Registro: c.registro, Autoridad: autoridad, Almacen: c.almacen,
		Seudonimizador: c.seudonimos, Reloj: c.reloj})
	if err != nil {
		return nil, err
	}
	return servicio.AbrirOriginalRRHH(ctx, docports.SolicitudOriginalSubidaLocalRRHH{
		ActorRegistradoV2: identidad.Actor, VinculoV2: identidad.Vinculo,
		DocumentoID: proyeccion.DocumentoRef, HuellaPDFSHA256: proyeccion.DocumentoSHA256,
		RecursoRef: proyeccion.RecursoRef, PersonaTitularRef: proyeccion.PersonaTitularRef,
		BolsaRef: bolsa, ParticipacionRef: participacion, CorrelacionRef: identidad.CorrelacionRef})
}

func descriptorMaterialJustificanteRenunciaRRHH(audiencia string) descriptorMaterialConsumidorV3Desarrollo {
	sufijo := "descargar"
	if audiencia == audienciaJustificanteBolsaPrepararRRHH {
		sufijo = "preparar"
	}
	return descriptorMaterialConsumidorV3Desarrollo{Audiencia: audiencia,
		Dominio:          "vec.bolsa.justificante-renuncia.rrhh." + sufijo + ".desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:bolsa-justificante-renuncia-rrhh-" + sufijo + ":",
		ProveedorNominal: "proveedor-material-bolsa-justificante-renuncia-rrhh-" + sufijo}
}

func emisoresJustificanteRenunciaRRHH(derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo, autorizador vecports.AutorizadorSolicitudLigadaV3) (map[string]*emisorMaterialRenovableCTDesarrollo, error) {
	base, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return nil, err
	}
	defer borrarBytes(base.claveHMAC)
	defer borrarBytes(base.privada)
	resultado := make(map[string]*emisorMaterialRenovableCTDesarrollo, 2)
	for _, audiencia := range []string{audienciaJustificanteBolsaRRHH, audienciaJustificanteBolsaPrepararRRHH} {
		derivada, err := derivarMaterialConsumidorV3Desarrollo(base, descriptorMaterialJustificanteRenunciaRRHH(audiencia))
		if err != nil {
			return nil, err
		}
		proveedor, err := nuevoProveedorMaterialAutorizacionBaseDesarrollo(derivada, reloj)
		borrarBytes(derivada.claveHMAC)
		if err != nil {
			return nil, err
		}
		emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, proveedor)
		if err != nil {
			return nil, err
		}
		resultado[audiencia] = emisor
	}
	return resultado, nil
}

func nuevaRutaJustificanteRenunciaRRHH(cfg config.Config, identidad *resolvedorIdentidadDesarrollo,
	usuarios *composicionPreferenciasUsuarios, derivador *derivadorIdentidadOperacionDesarrollo,
	autoridad *autoridadConsultasContratacionTemporalDesarrollo,
) (*vechttp.RutaExacta, func()) {
	ruta := &vechttp.RutaExacta{Ruta: bolsahttp.RutaJustificanteRenunciaRRHH,
		Manejador: bolsahttp.NuevoHandlerJustificanteRenunciaRRHHNoDisponible()}
	nada := func() {}
	etapa := "dependencias"
	fallo := func(causa error) (*vechttp.RutaExacta, func()) {
		// El error original puede contener el DSN o datos de una persona.
		// La etapa y esta clase cerrada bastan para el diagnóstico técnico.
		clase := "dependencia_no_disponible"
		if causa == nil || errors.Is(causa, errComposicionUsuariosPreferencias) {
			clase = "contrato_invalido"
		} else if etapa == "configuracion_rrhh" || etapa == "configuracion_almacen" {
			clase = "configuracion_invalida"
		} else if errors.Is(causa, context.DeadlineExceeded) {
			clase = "plazo_agotado"
		}
		slog.Warn("justificante_renuncia_bolsa_rrhh_no_disponible", "etapa", etapa, "causa", clase)
		return ruta, nada
	}
	if !cfg.DevelopmentEnabledByDoubleKey() || identidad == nil || usuarios == nil ||
		autoridad == nil || autoridad.sello == nil || autoridad.soporteDocumentosBolsa == nil ||
		autoridad.fronterasSeguridadComun.identidad == nil ||
		usuarios.interna == nil || usuarios.interna.base == nil ||
		usuarios.interna.base.registro == nil || usuarios.interna.base.revalidador == nil ||
		usuarios.interna.base.contextos == nil || usuarios.interna.fuente == nil ||
		usuarios.interna.base.resolvedor != identidad ||
		usuarios.interna.registroAutorizacion == nil || usuarios.interna.motivosPool == nil ||
		usuarios.interna.superficie != core.SuperficieAutenticacionInternaCorporativaV1 ||
		derivador == nil || !derivador.valido() {
		return fallo(errComposicionUsuariosPreferencias)
	}
	etapa = "configuracion_rrhh"
	conf, err := leerConfiguracionJustificanteRenunciaRRHH(cfg)
	if err != nil {
		return fallo(err)
	}
	etapa = "configuracion_almacen"
	privada, err := leerConfiguracionJustificantesBolsa(cfg.ExternoMaterialDir)
	if err != nil {
		return fallo(err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	etapa = "pool_nominal"
	pool, login, err := abrirPoolNominalPortalExterno(ctx, conf.DSNDocumentos,
		rolJustificanteRenunciaRRHH, "vec-bolsa-justificante-rrhh")
	if err != nil {
		return fallo(err)
	}
	completo := false
	defer func() {
		if !completo {
			pool.Close()
		}
	}()
	etapa = "topologia_funciones"
	topologia, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, usuarios.interna.motivosPool)
	if err != nil {
		return fallo(err)
	}
	if login != loginJustificanteRenunciaRRHH || usuarios.interna.logins[login] {
		return fallo(errComposicionUsuariosPreferencias)
	}
	if err = cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia); err != nil {
		return fallo(err)
	}
	if err = acreditarFuncionesCarteraExterna(ctx, pool, []string{
		"vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1(text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
		"vec_documentos.consultar_justificante_renuncia_rrhh_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	}); err != nil {
		return fallo(err)
	}
	etapa = "claves"
	claves, err := leerClavesSubidaDocumentosExterna(cfg.ExternoMaterialDir, privada.Datos)
	if err != nil {
		return fallo(err)
	}
	_, seudonimos, _, err := nuevasClavesAplicacionSubidaLocal(&claves)
	if err != nil {
		return fallo(err)
	}
	reloj := relojContratacionTemporalDesarrollo{}
	etapa = "almacen"
	almacen, cerrarAlmacen, err := nuevoAlmacenDocumentos(ctx, almacenDocumentosDesarrollo{
		Tipo: "ficheros", Directorio: privada.Datos.Directorio, TamanoMaximo: 16 << 20,
		RetencionMinimaDias: privada.Datos.RetencionMinimaDias}, reloj)
	if err != nil {
		return fallo(err)
	}
	defer func() {
		if !completo {
			cerrarAlmacen()
		}
	}()
	etapa = "conservacion"
	if err = admitirCatalogoConservacion(cfg, true, "ficheros", almacen); err != nil {
		return fallo(err)
	}
	etapa = "motivo"
	motivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(usuarios.interna.motivosPool, privada.Motivo.CatalogoID)
	if err != nil {
		return fallo(err)
	}
	etapa = "autorizacion"
	autorizador, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(usuarios.interna.fuente,
		usuarios.interna.registroAutorizacion, usuarios.interna.registroAutorizacion, motivos,
		reloj, seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
	if err != nil {
		return fallo(err)
	}
	etapa = "emisores"
	emisores, err := emisoresJustificanteRenunciaRRHH(derivador, reloj, autorizador)
	if err != nil {
		return fallo(err)
	}
	etapa = "resolutor"
	resolutor, err := bolsapg.NuevoResolutorConsultaJustificanteRenunciaRRHHConPool(pool,
		autoridadPreparacionJustificanteBolsaRRHH{emisores[audienciaJustificanteBolsaPrepararRRHH], privada.Motivo})
	if err != nil {
		return fallo(err)
	}
	etapa = "registro"
	registro, err := docpg.NuevoRegistroOriginalBolsaRRHHConPool(pool)
	if err != nil {
		return fallo(err)
	}
	etapa = "handler"
	h, err := bolsahttp.NuevoHandlerJustificanteRenunciaRRHH(
		proveedorIdentidadJustificanteRenunciaRRHH{sesion: sesionTecnicaJustificanteRenunciaRRHH{autoridad: autoridad}},
		&consultorJustificanteRenunciaRRHH{resolutor: resolutor, registro: registro,
			emisor: emisorJustificanteRenunciaRRHH{emisores[audienciaJustificanteBolsaRRHH]},
			motivo: privada.Motivo, almacen: almacen, seudonimos: seudonimos, reloj: reloj})
	if err != nil {
		return fallo(err)
	}
	ruta.Manejador = h
	var unaVez sync.Once
	completo = true
	return ruta, func() { unaVez.Do(func() { cerrarAlmacen(); pool.Close() }) }
}
