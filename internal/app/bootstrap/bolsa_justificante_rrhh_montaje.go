package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	Version           int                                 `json:"version"`
	Superficie        core.SuperficieAutenticacionActorV1 `json:"superficie"`
	DSNDocumentos     string                              `json:"dsn_documentos"`
	CertificadoSHA256 string                              `json:"certificado_sha256"`
	Sujeto            string                              `json:"sujeto"`
	CuentaRef         string                              `json:"cuenta_ref"`
	PerfilRef         string                              `json:"perfil_ref"`
	PersonaRef        string                              `json:"persona_ref"`
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
		c.DSNDocumentos == "" || strings.TrimSpace(c.DSNDocumentos) != c.DSNDocumentos ||
		!huellaPreimagenMiBolsaPortalExterno.MatchString(c.CertificadoSHA256) ||
		c.Sujeto == "" || !referenciaCuentaInscripcionInterna.MatchString(c.CuentaRef) ||
		!referenciaCuentaInscripcionInterna.MatchString(c.PerfilRef) || c.PersonaRef == "" {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	return c, nil
}

type proveedorIdentidadJustificanteRenunciaRRHH struct {
	identidad *resolvedorIdentidadDesarrollo
	autoridad *autoridadPreferenciasUsuariosDesarrollo
	cuenta    cuentaUsuariosPreferenciasDesarrollo
	persona   string
}

func (p proveedorIdentidadJustificanteRenunciaRRHH) PrepararIdentidadJustificanteRenunciaRRHH(r *http.Request) (
	bolsahttp.IdentidadJustificanteRenunciaRRHH, error) {
	var vacia bolsahttp.IdentidadJustificanteRenunciaRRHH
	if p.identidad == nil || p.autoridad == nil || p.autoridad.base == nil || r == nil || r.URL == nil ||
		r.Context().Err() != nil || r.TLS == nil || len(r.TLS.VerifiedChains) != 1 ||
		len(r.TLS.VerifiedChains[0]) == 0 || r.TLS.VerifiedChains[0][0] == nil {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	peticion := peticionIdentidadConsultasContratacionTemporalDesarrollo(r)
	cert := peticion.TLS.VerifiedChains[0][0]
	if len(cert.Raw) == 0 {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	huella := sha256.Sum256(cert.Raw)
	ahora := p.autoridad.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if hex.EncodeToString(huella[:]) != p.cuenta.CertificadoSHA256 ||
		ahora.Before(cert.NotBefore) || !ahora.Before(cert.NotAfter) {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	principal, err := p.identidad.ResolveDemoIdentity(peticion.Context(), peticion)
	if err != nil || principal.ID != p.cuenta.Sujeto || principal.AuthMethod != core.AuthMethodCertificate ||
		principal.AuthAssurance != core.AuthAssuranceHigh ||
		principal.Attributes["certificate_sha256"] != p.cuenta.CertificadoSHA256 {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	vinculo, actor, err := p.autoridad.resolverSesion(peticion, p.cuenta, ahora)
	if err != nil {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	datos, err := vinculo.Datos()
	if err != nil || actor.Validar() != nil || vinculo.ValidarPara(actor) != nil ||
		!vinculo.VigenteEn(p.autoridad.reloj.Ahora(), actor) || datos.CuentaPrivilegiada ||
		datos.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 ||
		datos.CuentaRef != p.cuenta.CuentaRef || datos.PerfilActivoRef != p.cuenta.PerfilRef ||
		actor.Contexto.Instantanea.CuentaRef != p.cuenta.CuentaRef ||
		actor.Contexto.PerfilActivoRef != p.cuenta.PerfilRef || actor.Contexto.PersonaRef != p.persona {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	corr, err := core.GenerarReferenciaCorrelacionAutorizacionV2(peticion.Context(),
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
) (*vechttp.RutaExacta, func()) {
	ruta := &vechttp.RutaExacta{Ruta: bolsahttp.RutaJustificanteRenunciaRRHH,
		Manejador: bolsahttp.NuevoHandlerJustificanteRenunciaRRHHNoDisponible()}
	nada := func() {}
	etapa := "dependencias"
	fallo := func() (*vechttp.RutaExacta, func()) {
		slog.Warn("justificante_renuncia_bolsa_rrhh_no_disponible", "etapa", etapa)
		return ruta, nada
	}
	if !cfg.DevelopmentEnabledByDoubleKey() || identidad == nil || usuarios == nil ||
		usuarios.interna == nil || usuarios.interna.base == nil || usuarios.interna.fuente == nil ||
		usuarios.interna.base.resolvedor != identidad ||
		usuarios.interna.registroAutorizacion == nil || usuarios.interna.motivosPool == nil ||
		usuarios.interna.superficie != core.SuperficieAutenticacionInternaCorporativaV1 ||
		derivador == nil || !derivador.valido() {
		return fallo()
	}
	etapa = "configuracion_rrhh"
	conf, err := leerConfiguracionJustificanteRenunciaRRHH(cfg)
	if err != nil {
		return fallo()
	}
	etapa = "configuracion_almacen"
	privada, err := leerConfiguracionJustificantesBolsa(cfg.ExternoMaterialDir)
	if err != nil {
		return fallo()
	}
	etapa = "perfil_propio"
	if previa, ok := usuarios.interna.cuentas[conf.CertificadoSHA256]; ok && previa.PerfilRef == conf.PerfilRef {
		return fallo()
	}
	etapa = "certificado"
	var digest [sha256.Size]byte
	b, err := hex.DecodeString(conf.CertificadoSHA256)
	if err != nil || len(b) != sha256.Size {
		return fallo()
	}
	copy(digest[:], b)
	principal, ok := identidad.porHuella[digest]
	if !ok || principal.ID != conf.Sujeto || principal.AuthMethod != core.AuthMethodCertificate ||
		principal.AuthAssurance != core.AuthAssuranceHigh || principal.Attributes["certificate_sha256"] != conf.CertificadoSHA256 {
		return fallo()
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	etapa = "pool_nominal"
	pool, login, err := abrirPoolNominalPortalExterno(ctx, conf.DSNDocumentos,
		rolJustificanteRenunciaRRHH, "vec-bolsa-justificante-rrhh")
	if err != nil {
		return fallo()
	}
	completo := false
	defer func() {
		if !completo {
			pool.Close()
		}
	}()
	etapa = "topologia_funciones"
	topologia, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, usuarios.interna.motivosPool)
	if err != nil || login != loginJustificanteRenunciaRRHH || usuarios.interna.logins[login] ||
		cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) != nil ||
		acreditarFuncionesCarteraExterna(ctx, pool, []string{
			"vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1(text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
			"vec_documentos.consultar_justificante_renuncia_rrhh_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
		}) != nil {
		return fallo()
	}
	etapa = "claves"
	claves, err := leerClavesSubidaDocumentosExterna(cfg.ExternoMaterialDir, privada.Datos)
	if err != nil {
		return fallo()
	}
	_, seudonimos, _, err := nuevasClavesAplicacionSubidaLocal(&claves)
	if err != nil {
		return fallo()
	}
	reloj := relojContratacionTemporalDesarrollo{}
	etapa = "almacen"
	almacen, cerrarAlmacen, err := nuevoAlmacenDocumentos(ctx, almacenDocumentosDesarrollo{
		Tipo: "ficheros", Directorio: privada.Datos.Directorio, TamanoMaximo: 16 << 20,
		RetencionMinimaDias: privada.Datos.RetencionMinimaDias}, reloj)
	if err != nil {
		return fallo()
	}
	defer func() {
		if !completo {
			cerrarAlmacen()
		}
	}()
	etapa = "conservacion"
	if admitirCatalogoConservacion(cfg, true, "ficheros", almacen) != nil {
		return fallo()
	}
	etapa = "motivo"
	motivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(usuarios.interna.motivosPool, privada.Motivo.CatalogoID)
	if err != nil {
		return fallo()
	}
	etapa = "autorizacion"
	autorizador, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(usuarios.interna.fuente,
		usuarios.interna.registroAutorizacion, usuarios.interna.registroAutorizacion, motivos,
		reloj, seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
	if err != nil {
		return fallo()
	}
	etapa = "emisores"
	emisores, err := emisoresJustificanteRenunciaRRHH(derivador, reloj, autorizador)
	if err != nil {
		return fallo()
	}
	etapa = "resolutor"
	resolutor, err := bolsapg.NuevoResolutorConsultaJustificanteRenunciaRRHHConPool(pool,
		autoridadPreparacionJustificanteBolsaRRHH{emisores[audienciaJustificanteBolsaPrepararRRHH], privada.Motivo})
	if err != nil {
		return fallo()
	}
	etapa = "registro"
	registro, err := docpg.NuevoRegistroOriginalBolsaRRHHConPool(pool)
	if err != nil {
		return fallo()
	}
	cuenta := cuentaUsuariosPreferenciasDesarrollo{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{
		CertificadoSHA256: conf.CertificadoSHA256, Sujeto: conf.Sujeto,
		CuentaRef: conf.CuentaRef, PerfilRef: conf.PerfilRef}}
	etapa = "handler"
	h, err := bolsahttp.NuevoHandlerJustificanteRenunciaRRHH(
		proveedorIdentidadJustificanteRenunciaRRHH{identidad, usuarios.interna, cuenta, conf.PersonaRef},
		&consultorJustificanteRenunciaRRHH{resolutor: resolutor, registro: registro,
			emisor: emisorJustificanteRenunciaRRHH{emisores[audienciaJustificanteBolsaRRHH]},
			motivo: privada.Motivo, almacen: almacen, seudonimos: seudonimos, reloj: reloj})
	if err != nil {
		return fallo()
	}
	ruta.Manejador = h
	var unaVez sync.Once
	completo = true
	return ruta, func() { unaVez.Do(func() { cerrarAlmacen(); pool.Close() }) }
}
