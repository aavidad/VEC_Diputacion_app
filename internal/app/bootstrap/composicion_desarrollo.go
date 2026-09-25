package bootstrap

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	publicatransitoria "vec-diputacion-granada/internal/app/composicion/publicatransitoria"
	"vec-diputacion-granada/internal/app/server"
	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ComposicionSeguridadDesarrollo agrupa proveedores concretos ya validados.
// Los campos privados impiden extraer las claves locales; solo se entregan las
// interfaces existentes y la marca obligatoria para persistencia.
type ComposicionSeguridadDesarrollo struct {
	metadatos             MetadatosNoAutoritativos
	procedencia           gobiernoconvocatorias.ProcedenciaActoBorrador
	tls                   *tls.Config
	identidad             vechttp.DemoIdentityResolver
	emisorKMS             *emisorKMSDesarrollo
	revalidadorKMS        *revalidadorKMSDesarrollo
	verificadorFirmasKMS  *verificadorFirmasKMSDesarrollo
	tsa                   vecports.TimestampPort
	derivadorIdempotencia *derivadorIdentidadOperacionDesarrollo
}

func NuevaComposicionSeguridadDesarrollo(
	cfg config.Config,
	registro io.Writer,
) (*ComposicionSeguridadDesarrollo, error) {
	cfg = cfg.Normalize()
	if err := validarRedLocalDesarrollo(cfg); err != nil {
		return nil, err
	}
	material, err := cargarMaterialSeguridadDesarrollo(cfg)
	if err != nil {
		return nil, err
	}
	defer borrarBytes(material.firmaAtestacionKMS)
	defer borrarBytes(material.firmaRevalidacionKMS)
	defer borrarBytes(material.claveKMS[:])
	defer borrarBytes(material.claveTSA[:])
	defer material.idempotencia.borrar()
	derivadorIdempotencia, err := nuevoDerivadorIdentidadOperacionDesarrollo(&material.idempotencia)
	if err != nil {
		return nil, err
	}
	derivadorEntregado := false
	defer func() {
		if !derivadorEntregado {
			derivadorIdempotencia.borrar()
		}
	}()
	emisorKMS, revalidadorKMS, verificadorFirmasKMS, err := nuevosProveedoresKMSDesarrollo(
		material.claveKMS,
		material.firmaAtestacionKMS, material.verificadorAtestacionKMS,
		material.huellaPublicaAtestacionKMS,
		material.firmaRevalidacionKMS, material.verificadorRevalidacionKMS,
		material.huellaPublicaRevalidacionKMS,
	)
	if err != nil {
		return nil, err
	}
	selladorTSA := nuevoSelladorTiempoDesarrollo(material.claveTSA)
	proveedores, err := descriptoresProveedoresDesarrollo(
		material.identidad, emisorKMS, revalidadorKMS, verificadorFirmasKMS,
		selladorTSA, derivadorIdempotencia, material.configuracionTLS,
	)
	if err != nil {
		return nil, err
	}
	metadatos, err := PrepararPerfilEjecucion(cfg, proveedores, registro)
	if err != nil {
		return nil, err
	}
	procedencia, err := gobiernoconvocatorias.NuevaProcedenciaActoBorrador(
		config.ExecutionProfileDevelopment,
		gobiernoconvocatorias.AutoridadActoNoAutoritativa,
		proveedorSeguridadDesarrolloRef,
		false,
	)
	if err != nil {
		return nil, err
	}
	resultado := &ComposicionSeguridadDesarrollo{
		metadatos:             metadatos,
		procedencia:           procedencia,
		tls:                   material.configuracionTLS.Clone(),
		identidad:             material.identidad,
		emisorKMS:             emisorKMS,
		revalidadorKMS:        revalidadorKMS,
		verificadorFirmasKMS:  verificadorFirmasKMS,
		tsa:                   selladorTSA,
		derivadorIdempotencia: derivadorIdempotencia,
	}
	derivadorEntregado = true
	return resultado, nil
}

func (c *ComposicionSeguridadDesarrollo) MetadatosComposicion() (MetadatosNoAutoritativos, error) {
	if c == nil || c.tls == nil || c.identidad == nil || c.emisorKMS == nil ||
		c.revalidadorKMS == nil || c.verificadorFirmasKMS == nil || c.tsa == nil ||
		c.derivadorIdempotencia == nil || !c.derivadorIdempotencia.valido() {
		return MetadatosNoAutoritativos{}, ErrComposicionDesarrolloIncompleta
	}
	return c.metadatos, nil
}

// ProcedenciaActosBorrador es la unica marca durable. T20 debe inyectarla en
// ServicioBorradores para que viaje en AAD, agregado, recibo, auditoria y outbox.
func (c *ComposicionSeguridadDesarrollo) ProcedenciaActosBorrador() (gobiernoconvocatorias.ProcedenciaActoBorrador, error) {
	if c == nil || c.procedencia.Esquema == "" {
		return gobiernoconvocatorias.ProcedenciaActoBorrador{}, ErrComposicionDesarrolloIncompleta
	}
	return c.procedencia, nil
}

func (c *ComposicionSeguridadDesarrollo) CifradorBorradores() (gobiernoconvocatorias.CifradorAEADKMSBorrador, error) {
	if c == nil || c.emisorKMS == nil {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	return c.emisorKMS, nil
}

func (c *ComposicionSeguridadDesarrollo) DescifradorBorradores() (
	gobiernoconvocatorias.DescifradorBorradorDurable,
	error,
) {
	if c == nil || c.emisorKMS == nil {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	return c.emisorKMS, nil
}

func (c *ComposicionSeguridadDesarrollo) RevalidadorKMSBorradores() (gobiernoconvocatorias.RevalidadorAtestacionKMSBorrador, error) {
	if c == nil || c.revalidadorKMS == nil {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	return c.revalidadorKMS, nil
}

// VerificadorFirmasKMSBorradores devuelve la responsabilidad de lectura que
// solo posee las dos claves publicas. T20 la envuelve al releer el recibo
// durable; no permite cifrar ni emitir pruebas.
func (c *ComposicionSeguridadDesarrollo) VerificadorFirmasKMSBorradores() (*verificadorFirmasKMSDesarrollo, error) {
	if c == nil || c.verificadorFirmasKMS == nil {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	return c.verificadorFirmasKMS, nil
}

func (c *ComposicionSeguridadDesarrollo) ConfiguracionTLS() (*tls.Config, error) {
	if c == nil || c.tls == nil {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	return c.tls.Clone(), nil
}

func (c *ComposicionSeguridadDesarrollo) ResolvedorIdentidad() (vechttp.DemoIdentityResolver, error) {
	if c == nil || c.identidad == nil {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	return c.identidad, nil
}

func (c *ComposicionSeguridadDesarrollo) SelladorTiempo() (vecports.TimestampPort, error) {
	if c == nil || c.tsa == nil {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	return c.tsa, nil
}

// DerivadorIdentidadesBorrador entrega el puerto HMAC ya compuesto. Las claves
// siguen encapsuladas en el adaptador privado y no tienen getters.
func (c *ComposicionSeguridadDesarrollo) DerivadorIdentidadesBorrador() (
	gobiernoconvocatorias.DerivadorIdentidadOperacion,
	error,
) {
	if c == nil || c.derivadorIdempotencia == nil || !c.derivadorIdempotencia.valido() {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	return c.derivadorIdempotencia, nil
}

// NewHTTPServerDesarrolloWithConfig arranca una vertical web real sobre mTLS.
// Devuelve tambien la composicion: T20 debe consumir ProcedenciaActosBorrador
// y su KMS en la misma raiz antes de admitir escrituras durables.
func NewHTTPServerDesarrolloWithConfig(
	cfg config.Config,
	registro io.Writer,
	incorporacion ...ConfiguracionIncorporacionDesarrollo,
) (*http.Server, *ComposicionSeguridadDesarrollo, error) {
	return nuevoServidorDesarrollo(cfg, registro, vecports.EmisorIncidenciasTecnicasNulo{}, incorporacion...)
}

// nuevoServidorDesarrollo es NewHTTPServerDesarrolloWithConfig con el emisor
// de incidencias técnicas de la composición raíz.
func nuevoServidorDesarrollo(
	cfg config.Config,
	registro io.Writer,
	emisor vecports.EmisorIncidenciasTecnicas,
	incorporacion ...ConfiguracionIncorporacionDesarrollo,
) (*http.Server, *ComposicionSeguridadDesarrollo, error) {
	if len(incorporacion) > 1 {
		return nil, nil, ErrComposicionDesarrolloIncompleta
	}
	cfg = cfg.Normalize()
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, registro)
	if err != nil {
		return nil, nil, err
	}
	resolvedor, err := composicion.ResolvedorIdentidad()
	if err != nil {
		return nil, nil, err
	}
	consultaCategorias, categoriasPersonal, err := nuevasDependenciasCategoriasProfesionales(cfg)
	if err != nil {
		return nil, nil, err
	}
	consultasPersonal := nuevasConsultasPublicasPersonal(cfg, categoriasPersonal, registro)
	rutasContratacion, autoridadContratacion, cerrarContratacion, err := nuevasRutasContratacionTemporalDesarrollo(
		cfg, resolvedor, composicion.derivadorIdempotencia, composicion.emisorKMS, registro, incorporacion...,
	)
	if err != nil {
		return nil, nil, err
	}
	completa := false
	defer func() {
		if !completa {
			cerrarContratacion()
		}
	}()
	ctxBolsas, cancelarBolsas := context.WithTimeout(context.Background(), 15*time.Second)
	fuenteConstituida := nuevaFuenteConstituidaRRHHDesarrollo(ctxBolsas, cfg)
	cancelarBolsas()
	autoridadContratacion.personalizacionB7.fijar(fuenteConstituida)
	rutasBolsasRRHH, coleccionesBolsasRRHH, err := nuevasRutasBolsasRRHHDesarrolloConFuente(cfg, fuenteConstituida, autoridadContratacion.manejadorSituacionParticipacion)
	if err != nil {
		return nil, nil, err
	}
	rutasContratacion = append(rutasContratacion, rutasBolsasRRHH...)
	coleccionesBolsasRRHH = append(coleccionesBolsasRRHH, autoridadContratacion.coleccionesAdicionales...)
	rutasCalendarios, cerrarCalendarios, err := nuevasRutasCalendariosDesarrollo(cfg)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if !completa {
			cerrarCalendarios()
		}
	}()
	rutasContratacion = append(rutasContratacion, rutasCalendarios...)
	autoridadDietas, cerrarDietas, err := nuevasRutasDietasDesarrollo(cfg, resolvedor, composicion.derivadorIdempotencia)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if !completa {
			cerrarDietas()
		}
	}()
	comisionesDietas, err := nuevasComisionesDietasDesarrollo(cfg, resolvedor, composicion.derivadorIdempotencia, autoridadContratacion.materialDietas)
	if err != nil {
		return nil, nil, err
	}
	if comisionesDietas != nil {
		defer func() {
			if !completa {
				comisionesDietas.cerrar()
			}
		}()
		rutasContratacion = append(rutasContratacion, comisionesDietas.rutas...)
		coleccionesBolsasRRHH = append(coleccionesBolsasRRHH, comisionesDietas.colecciones...)
	}
	cronosEmpleado, err := nuevasRutasCronosEmpleadoDesarrollo(cfg, resolvedor, composicion.derivadorIdempotencia, autoridadContratacion.materialCronos)
	if err != nil {
		return nil, nil, err
	}
	if cronosEmpleado != nil {
		defer func() {
			if !completa {
				cronosEmpleado.cerrar()
			}
		}()
	}
	documentos, err := nuevosDocumentosDesarrollo(cfg, resolvedor, composicion.derivadorIdempotencia, autoridadContratacion.materialDocumentos, registro)
	if err != nil {
		return nil, nil, err
	}
	if documentos != nil {
		defer func() {
			if !completa {
				documentos.cerrar()
			}
		}()
		rutasContratacion = append(rutasContratacion, documentos.rutas...)
	}
	personalEmpleado, err := nuevasRutasPersonalEmpleadoDesarrollo(cfg, resolvedor, composicion.derivadorIdempotencia, autoridadContratacion.materialPersonalFichaPropia)
	if err != nil {
		return nil, nil, err
	}
	if personalEmpleado != nil {
		defer func() {
			if !completa {
				personalEmpleado.cerrar()
			}
		}()
	}
	autoridadExactas := vechttp.AutoridadRutasExactas(autoridadContratacion)
	if comisionesDietas != nil {
		autoridadExactas = autoridadExactasConDietas{delegada: autoridadContratacion, dietas: comisionesDietas}
	}
	if documentos != nil {
		autoridadExactas = autoridadExactasConDocumentos{delegada: autoridadExactas, documentos: documentos}
	}
	vecAPI, err := newVECShellAPICompuestaConIdentidadYRutas(
		cfg, emisor, resolvedor, categoriasPersonal, rutasContratacion, autoridadExactas,
		autoridadContratacion.registradorAuditoriaFronteraRutasExactas, autoridadDietas, coleccionesBolsasRRHH...,
	)
	if err != nil {
		return nil, nil, err
	}
	vecAPI = autoridadContratacion.proteger(vecAPI)
	if comisionesDietas != nil {
		vecAPI = comisionesDietas.proteger(vecAPI)
	}
	if documentos != nil {
		vecAPI = documentos.proteger(vecAPI)
	}
	cfgPublica := cfg
	cfgPublica.AuthMode = config.AuthModeDisabled
	publicaBolsaAPI, err := publicatransitoria.NuevaAPIConCatalogos(cfgPublica, consultaCategorias)
	if err != nil {
		return nil, nil, err
	}
	bolsasPublicas, err := nuevoManejadorBolsasPublicasDesarrollo(fuenteConstituida)
	if err != nil {
		return nil, nil, err
	}
	servidor, err := server.NewHTTPServer(cfg, componerRaizConPersonalPublico(componerRaizConPersonalEmpleado(componerRaizConCronosEmpleado(composeVECShellAPIConBolsasPublicas(vecAPI, publicaBolsaAPI, bolsasPublicas), cronosEmpleado), personalEmpleado), consultasPersonal))
	if err != nil {
		return nil, nil, err
	}
	servidor.TLSConfig, err = composicion.ConfiguracionTLS()
	if err != nil {
		return nil, nil, err
	}
	servidor.RegisterOnShutdown(cerrarContratacion)
	servidor.RegisterOnShutdown(cerrarDietas)
	servidor.RegisterOnShutdown(cerrarCalendarios)
	if comisionesDietas != nil {
		servidor.RegisterOnShutdown(comisionesDietas.cerrar)
	}
	if cronosEmpleado != nil {
		servidor.RegisterOnShutdown(cronosEmpleado.cerrar)
	}
	if documentos != nil {
		servidor.RegisterOnShutdown(documentos.cerrar)
	}
	if personalEmpleado != nil {
		servidor.RegisterOnShutdown(personalEmpleado.cerrar)
	}
	completa = true
	return servidor, composicion, nil
}

func descriptoresProveedoresDesarrollo(
	identidad *resolvedorIdentidadDesarrollo,
	emisor *emisorKMSDesarrollo,
	revalidador *revalidadorKMSDesarrollo,
	verificador *verificadorFirmasKMSDesarrollo,
	tsa *selladorTiempoDesarrollo,
	derivadorIdempotencia *derivadorIdentidadOperacionDesarrollo,
	configuracionTLS *tls.Config,
) ([]DescriptorProveedorSeguridad, error) {
	// Los descriptores sólo se emiten después de comprobar los adaptadores
	// concretos que realmente formarán la composición. Así la guarda no puede
	// aprobar un inventario declarativo desconectado del cableado efectivo.
	if identidad == nil || emisor == nil || revalidador == nil || verificador == nil ||
		tsa == nil || derivadorIdempotencia == nil || !derivadorIdempotencia.valido() ||
		configuracionTLS == nil ||
		configuracionTLS.ClientAuth != tls.RequireAndVerifyClientCert ||
		configuracionTLS.MinVersion != tls.VersionTLS13 ||
		configuracionTLS.MaxVersion != tls.VersionTLS13 {
		return nil, ErrComposicionDesarrolloIncompleta
	}
	datos := []struct {
		tipo       TipoProveedorSeguridad
		referencia string
	}{
		{ProveedorIdentidad, "identidad-mtls-local-v1"},
		{ProveedorIdempotencia, referenciaProveedorIdempotenciaDesarrollo},
		{ProveedorKMS, "kms-fichero-local-v2"},
		{ProveedorTSA, "tsa-determinista-local-v1"},
		{ProveedorTLS, "tls-ca-local-v1"},
	}
	resultado := make([]DescriptorProveedorSeguridad, 0, len(datos))
	for _, dato := range datos {
		descriptor, err := NuevoDescriptorProveedorDesarrollo(dato.tipo, dato.referencia)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, descriptor)
	}
	return resultado, nil
}

func validarRedLocalDesarrollo(cfg config.Config) error {
	host, puerto, err := net.SplitHostPort(strings.TrimSpace(cfg.Address))
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if err != nil || ip == nil || !ip.IsLoopback() || puerto == "" {
		return errors.Join(ErrActivacionDesarrolloInvalida, errors.New("listener desarrollo debe ser loopback literal"))
	}
	for _, cidr := range cfg.HTTPAllowedCIDRs {
		ipRed, red, err := net.ParseCIDR(cidr)
		if err != nil || ipRed == nil || !ipRed.IsLoopback() || red == nil ||
			(ipRed.To4() != nil && strings.TrimSpace(cidr) != "127.0.0.1/32") ||
			(ipRed.To4() == nil && strings.TrimSpace(cidr) != "::1/128") {
			return errors.Join(ErrActivacionDesarrolloInvalida, errors.New("red desarrollo debe limitarse a loopback"))
		}
	}
	return nil
}
