package bootstrap

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	personalhttp "vec-diputacion-granada/internal/modules/personal/adapters/httpinterno"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

const rutaClienteM10LlamamientoCT = "VEC_CT_BOLSA_ORGANIZACION_SOURCE_PATH"

type configuracionClienteM10LlamamientoCT struct {
	Version            int    `json:"version"`
	OrganismoRef       string `json:"organismo_ref"`
	Origen             string `json:"origen"`
	Directorio         string `json:"directorio"`
	AutoridadCA        string `json:"autoridad_ca"`
	CertificadoCliente string `json:"certificado_cliente"`
	ClaveCliente       string `json:"clave_cliente"`
	MaximoBytesPagina  int64  `json:"maximo_bytes_pagina"`
}

func cargarClienteM10LlamamientoCT() (*personalhttp.ClienteOrganizacionHistorica, string, error) {
	ruta := os.Getenv(rutaClienteM10LlamamientoCT)
	if ruta == "" || !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta || dentroDeRepositorioGit(ruta) {
		return nil, "", ports.ErrEmisionLlamamientoNoDisponible
	}
	contenido, err := leerFicheroMaterialSeguro(ruta, 16<<10)
	if err != nil {
		return nil, "", ports.ErrEmisionLlamamientoNoDisponible
	}
	defer borrarBytes(contenido)
	if validarClavesJSONUnicas(contenido) != nil {
		return nil, "", ports.ErrEmisionLlamamientoNoDisponible
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var c configuracionClienteM10LlamamientoCT
	if decodificador.Decode(&c) != nil || decodificador.Decode(new(any)) != io.EOF ||
		c.Version != 1 || c.OrganismoRef == "" || c.Origen == "" ||
		c.MaximoBytesPagina < 1 || c.MaximoBytesPagina > 8<<20 {
		return nil, "", ports.ErrEmisionLlamamientoNoDisponible
	}
	cliente, err := personalhttp.NuevoClienteOrganizacionHistorica(personalhttp.ConfiguracionOrganizacionHistorica{
		Origen: c.Origen, Directorio: c.Directorio, AutoridadCA: c.AutoridadCA,
		CertificadoCliente: c.CertificadoCliente, ClaveCliente: c.ClaveCliente,
		MaximoBytesPagina: c.MaximoBytesPagina,
	})
	if err != nil {
		return nil, "", ports.ErrEmisionLlamamientoNoDisponible
	}
	return cliente, c.OrganismoRef, nil
}

// La orden nace de la identidad mTLS revalidada y del recurso Bolsa leído en
// la misma transacción. El acuse lo produce exclusivamente el registrador SQL.
var camposLecturaOrigenCT = []string{
	"acta_ref", "bolsa_canonica", "bolsa_ref",
	"categoria_ref", "corte", "entradas[].candidato_ref",
	"entradas[].elegible", "entradas[].orden_acta", "entradas[].orden_vigente",
	"entradas[].participacion_ref", "entradas[].razon", "entradas[].situacion",
	"huella_bolsa_sha256", "huella_instantanea_sha256", "huella_politica_sha256",
	"instantanea_canonica", "instantanea_ref", "politica_provisional",
	"politica_ref", "reserva.candidato_ref", "reserva.clave_ct",
	"reserva.configuracion.asunto", "reserva.configuracion.categoria", "reserva.configuracion.centro",
	"reserva.configuracion.cuerpo", "reserva.configuracion.descripcion", "reserva.configuracion.fecha_inicio",
	"reserva.configuracion.modalidad", "reserva.configuracion.plantilla_version", "reserva.configuracion.plazo",
	"reserva.configuracion.referencia", "reserva.expediente_ref", "reserva.orden_seleccionado",
	"reserva.participacion_ref", "reserva.unidad_ref", "version_bolsa",
	"version_instantanea", "version_politica",
}

func nuevoConstructorOrdenOrigenCT(
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	preparador *preparadorBorradorLlamamientoDesarrollo,
	proceso, canal string,
) registradorLecturaOrdenCT {
	contrato := contratoLecturaBolsa{
		accion:    ports.AccionConsultarOrigenCT,
		finalidad: ports.FinalidadConsultarOrigenCT,
		tipo:      "bolsa_constituida",
		motivo:    "lectura_origen_ct_correcta",
		campos:    camposLecturaOrigenCT,
	}
	return func(ctx context.Context, d preparacionLlamamientoDesarrollo, bolsaRef string) (ports.CapturaOrigenCT, error) {
		vacia := ports.CapturaOrigenCT{}
		if ctx == nil || ctx.Err() != nil || alta == nil || alta.soporte == nil ||
			preparador == nil || preparador.soporte == nil || preparador.fuenteRespuestas == nil ||
			bolsaRef == "" || d.unidad == "" || d.unidad != preparador.soporte.unidadRef ||
			preparador.soporte.ambitoRef == "" {
			return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
		}
		operativo, err := alta.soporte.contextoOperativoDesarrollo(ctx)
		if err != nil || operativo.Resultado.Validar() != nil || operativo.Vinculo.ValidarPara(operativo.Resultado) != nil {
			return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
		}
		propio, err := preparador.contextoRevalidado(ctx)
		if err != nil || propio.Resultado.Validar() != nil || propio.Vinculo.ValidarPara(propio.Resultado) != nil {
			return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
		}
		a, ea := operativo.Vinculo.Datos()
		b, eb := propio.Vinculo.Datos()
		if ea != nil || eb != nil || a.PrincipalID != b.PrincipalID ||
			a.PerfilActivoRef != b.PerfilActivoRef || a.CuentaRef != b.CuentaRef ||
			a.AutenticacionRef != b.AutenticacionRef || a.SesionRef != b.SesionRef {
			return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
		}
		permiso, err := capturarLecturaBolsa(ctx, preparador.fuenteRespuestas,
			propio.Vinculo, propio.Resultado, preparador.soporte.unidadRef,
			preparador.soporte.ambitoRef, bolsaRef, bolsaRef, contrato)
		if err != nil {
			return vacia, err
		}
		intento, err := vecports.NuevaReferenciaIntentoAuditoria()
		if err != nil {
			return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
		}
		correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
		if err != nil {
			return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
		}
		correlacionRef, err := correlacion.ValorCanonico()
		if err != nil {
			return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
		}
		orden, err := vecports.NuevaOrdenIntentoAuditoria(intento, propio.Resultado, propio.Vinculo,
			domain.DatosIntentoAuditoria{
				Accion: ports.AccionConsultarOrigenCT, ModuloID: "bolsa", RecursoRef: bolsaRef,
				FinalidadRef: ports.FinalidadConsultarOrigenCT, Resultado: domain.ResultadoIntentoAuditoriaExito,
				Motivo: motivoLlamamientoDesarrollo(false), Proceso: proceso, Canal: canal,
				CorrelacionRef: correlacionRef,
			})
		if err != nil {
			return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
		}
		return ports.CapturaOrigenCT{Permiso: permiso, Auditoria: orden}, nil
	}
}

// El expediente aporta la necesidad exacta; Personal comprueba que su puesto
// sigue perteneciendo a la publicación RPT cuya huella quedó en la petición.
// Un expediente anterior sin ese vínculo no permite emitir en el circuito nativo.
type resolutorPuestoRPTConstituidoCT func(context.Context, string, string) (string, error)

var patronPuestoRPTConstituidoCT = regexp.MustCompile(`^puesto:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func nuevoProveedorDatosEmisionConstituidaCT(
	rpt personalports.ConsultaRPTPublica, resolutor *reglas.Resolutor, ahora func() time.Time,
	resolverPuesto resolutorPuestoRPTConstituidoCT,
) proveedorConfiguracionEmisionCT {
	return func(ctx context.Context, d preparacionLlamamientoDesarrollo) (ports.ConfiguracionLlamamiento, string, domain.ReferenciaEntradaCatalogo, error) {
		vacio := ports.ConfiguracionLlamamiento{}
		if ctx == nil || ctx.Err() != nil || rpt == nil || resolutor == nil || !resolutor.Disponible() || ahora == nil || resolverPuesto == nil {
			return vacio, "", domain.ReferenciaEntradaCatalogo{}, ports.ErrEmisionLlamamientoNoDisponible
		}
		e := d.expediente.Fiscalizado
		if e.Validar() != nil || e.Analisis == nil || e.Solicitud.Necesidad == nil ||
			e.Solicitud.Necesidad.ValidarInstantanea() != nil || e.Solicitud.Detalle == "" {
			return vacio, "", domain.ReferenciaEntradaCatalogo{}, ports.ErrEmisionLlamamientoNoDisponible
		}
		campos := e.Solicitud.Necesidad.Campos
		codigo := campos["puesto_codigo"]
		publicacion, err := rpt.ObtenerRPTPublica(ctx)
		if err != nil || publicacion.Validar() != nil ||
			publicacion.Fuente.Importacion != campos["rpt_catalogo_ref"] ||
			subtle.ConstantTimeCompare([]byte(publicacion.Fuente.HuellaSHA256), []byte(campos["rpt_catalogo_huella_sha256"])) != 1 {
			return vacio, "", domain.ReferenciaEntradaCatalogo{}, ports.ErrEmisionLlamamientoNoDisponible
		}
		puesto, categoria, ok := puestoPublicadoNecesidadCT(publicacion, codigo, d.categoria, e.Solicitud.CentroRef)
		if !ok {
			return vacio, "", domain.ReferenciaEntradaCatalogo{}, ports.ErrEmisionLlamamientoNoDisponible
		}
		puestoRef, err := resolverPuesto(ctx, codigo, d.unidad)
		if err != nil || !patronPuestoRPTConstituidoCT.MatchString(puestoRef) {
			return vacio, "", domain.ReferenciaEntradaCatalogo{}, ports.ErrEmisionLlamamientoNoDisponible
		}
		// b05 comienza con contacto efectivo. El aviso informativo no crea una
		// fecha de vencimiento: el texto de plazo procede de la regla publicada.
		regla, err := resolutor.Regla(ctx, reglas.BolsaPlazoRespuesta)
		if err != nil || regla.Descripcion == "" || regla.Atributos["inicio"] != inicioPlazoContactoEfectivo {
			return vacio, "", domain.ReferenciaEntradaCatalogo{}, ports.ErrEmisionLlamamientoNoDisponible
		}
		correo, err := cargarCatalogoCorreoLlamamientoBolsa()
		if err != nil || !correo.Valido() {
			return vacio, "", domain.ReferenciaEntradaCatalogo{}, ports.ErrEmisionLlamamientoNoDisponible
		}
		textos := correo.TextosDefecto(correo.IdiomaDefecto())
		configuracion := ports.ConfiguracionLlamamiento{
			Referencia: e.Referencia, Descripcion: e.Solicitud.Detalle,
			Categoria: categoria, Centro: puesto.Centro,
			Modalidad:   string(e.Analisis.ModalidadClave),
			FechaInicio: e.Analisis.Periodo.Inicio.Format("02/01/2006"),
			Plazo:       regla.Descripcion, PlantillaVersion: correo.PlantillaVigente(),
			Asunto: textos.Asunto, Cuerpo: textos.Cuerpo,
		}
		if ctx.Err() != nil || ahora().IsZero() ||
			correo.ValidarPlantilla(configuracion.PlantillaVersion, configuracion.Asunto) != nil ||
			correo.ValidarPlantilla(configuracion.PlantillaVersion, configuracion.Cuerpo) != nil {
			return vacio, "", domain.ReferenciaEntradaCatalogo{}, ports.ErrEmisionLlamamientoNoDisponible
		}
		return configuracion, puestoRef, motivoEmitirLlamamientoBolsaDesarrollo(), nil
	}
}

// El cliente HTTP existente de M10 entrega páginas con autorización V3 y
// auditoría SQL de lectura. La raíz aporta su origen mTLS y ámbito privado.
type consultaOrganizacionHistoricaPuestoCT interface {
	Consultar(context.Context, personaldomain.SelectorOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error)
}

func nuevoResolutorPuestoRPTConstituidoCT(
	cliente consultaOrganizacionHistoricaPuestoCT, organismoRef string, ahora func() time.Time,
) resolutorPuestoRPTConstituidoCT {
	return func(ctx context.Context, codigo, unidadRef string) (string, error) {
		if ctx == nil || ctx.Err() != nil || cliente == nil || ahora == nil || codigo == "" || unidadRef == "" || organismoRef == "" {
			return "", ports.ErrEmisionLlamamientoNoDisponible
		}
		instante := ahora().UTC().Truncate(time.Microsecond)
		selector := personaldomain.SelectorOrganizacionHistorica{
			OrganismoRef: organismoRef, UnidadClave: unidadRef,
			VigenteEn:  personaldomain.FechaCivil(instante.Format(time.DateOnly)),
			ConocidoEn: instante, Limite: personaldomain.LimiteMaximoOrganizacionHistorica,
		}
		if selector.Validar() != nil {
			return "", ports.ErrEmisionLlamamientoNoDisponible
		}
		var puestoRef, versionRPT string
		visitados := map[string]bool{}
		for pagina := 0; pagina < 64; pagina++ {
			resultado, err := cliente.Consultar(ctx, selector)
			if err != nil || ctx.Err() != nil || resultado.Evidencia.AuditoriaRef == "" ||
				resultado.Pagina.VersionRPTRef == "" {
				return "", ports.ErrEmisionLlamamientoNoDisponible
			}
			if versionRPT != "" && versionRPT != resultado.Pagina.VersionRPTRef {
				return "", ports.ErrEmisionLlamamientoNoDisponible
			}
			versionRPT = resultado.Pagina.VersionRPTRef
			for _, fila := range resultado.Pagina.PuestosIndividuales {
				if !codigoPuestoM10ConstituidoCoincide(fila.CodigoFuente, codigo) ||
					fila.UnidadID != unidadRef || fila.EstadoEstructural != "vigente" {
					continue
				}
				if !patronPuestoRPTConstituidoCT.MatchString(fila.Traza.ID) || puestoRef != "" && puestoRef != fila.Traza.ID {
					return "", ports.ErrEmisionLlamamientoNoDisponible
				}
				puestoRef = fila.Traza.ID
			}
			siguiente := resultado.Pagina.CursorSiguiente
			if siguiente == "" {
				if puestoRef == "" {
					return "", ports.ErrEmisionLlamamientoNoDisponible
				}
				return puestoRef, nil
			}
			if visitados[siguiente] {
				return "", ports.ErrEmisionLlamamientoNoDisponible
			}
			visitados[siguiente] = true
			selector.Cursor = siguiente
		}
		return "", ports.ErrEmisionLlamamientoNoDisponible
	}
}

// M10 identifica cada puesto individual con el código público y su ordinal
// de dos cifras; varios ordinales para la misma necesidad son ambiguos.
func codigoPuestoM10ConstituidoCoincide(codigoFuente, codigo string) bool {
	if len(codigoFuente) != len(codigo)+3 || !strings.HasPrefix(codigoFuente, codigo+"/") {
		return false
	}
	sufijo := codigoFuente[len(codigo)+1:]
	return sufijo[0] >= '0' && sufijo[0] <= '9' && sufijo[1] >= '0' && sufijo[1] <= '9' && sufijo != "00"
}

func puestoPublicadoNecesidadCT(c personaldomain.CatalogoRPTPublica, codigo, categoriaRef, centroRef string) (personaldomain.PuestoRPTPublico, string, bool) {
	if codigo == "" || !strings.HasPrefix(categoriaRef, "categoria:rpt:") {
		return personaldomain.PuestoRPTPublico{}, "", false
	}
	claveCategoria := strings.TrimPrefix(categoriaRef, "categoria:rpt:")
	var categoria string
	for _, fila := range c.Categorias {
		if fila.Clave == claveCategoria {
			categoria = fila.Denominacion
			break
		}
	}
	if categoria == "" {
		return personaldomain.PuestoRPTPublico{}, "", false
	}
	for _, puesto := range c.Puestos {
		if puesto.Codigo == codigo && puesto.CategoriaClave == claveCategoria &&
			(centroRef == "centro:rpt:"+puesto.CentroCodigo || centroRef == "centro-"+puesto.CentroCodigo ||
				centroRef == "centro:"+puesto.CentroCodigo) {
			return puesto, categoria, true
		}
	}
	return personaldomain.PuestoRPTPublico{}, "", false
}

// La primera generación es la activa; las demás permiten recuperar una
// reserva B7 anterior sin recrear el token ni exponer el secreto del fichero.
type derivadorTokenFinalizacionCTDesarrollo struct {
	material *derivadorIdentidadOperacionDesarrollo
}

var _ ports.DerivadorTokenFinalizacionCT = derivadorTokenFinalizacionCTDesarrollo{}

func nuevoDerivadorTokenFinalizacionCTDesarrollo(material *derivadorIdentidadOperacionDesarrollo) (ports.DerivadorTokenFinalizacionCT, error) {
	if material == nil || !material.valido() {
		return nil, ports.ErrEmisionLlamamientoNoDisponible
	}
	return derivadorTokenFinalizacionCTDesarrollo{material: material}, nil
}

func (d derivadorTokenFinalizacionCTDesarrollo) DerivarTokenFinalizacionCT(ctx context.Context, datos ports.DatosDerivacionTokenFinalizacionCT) ([]byte, string, error) {
	if ctx == nil || ctx.Err() != nil || d.material == nil || !d.material.valido() ||
		!ports.ReferenciaOpacaLlamamientoValida(datos.BolsaRef) || datos.ClaveB7 == "" ||
		!ports.ReferenciaOpacaLlamamientoValida(datos.ActorRef) ||
		!huellaSHA256ValidaContratacionTemporalDesarrollo(datos.HuellaOrigenSHA256) {
		return nil, "", ports.ErrEmisionLlamamientoNoDisponible
	}
	// JSON fija límites entre campos. El prefijo separa este uso de cualquier
	// otra identidad HMAC obtenida del mismo proveedor privado versionado.
	preimagen, err := json.Marshal([]string{ports.DominioFinalizacionCT, datos.BolsaRef, datos.ClaveB7, datos.ActorRef, datos.HuellaOrigenSHA256})
	if err != nil {
		return nil, "", ports.ErrEmisionLlamamientoNoDisponible
	}
	defer borrarBytes(preimagen)
	resultados, err := d.material.calcularHMAC(preimagen, preimagen)
	if err != nil {
		return nil, "", ports.ErrEmisionLlamamientoNoDisponible
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	for _, resultado := range resultados {
		version := fmt.Sprintf("bolsa.finalizacion.v1:g%d", resultado.generacion)
		if datos.VersionClave == "" || datos.VersionClave == version {
			token := append([]byte(nil), resultado.localizador[:]...)
			return token, version, nil
		}
	}
	return nil, "", ports.ErrEmisionLlamamientoNoDisponible
}
