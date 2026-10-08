package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/documentos/adapters/validadorautofirma"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const plazoObservacionPreparacionR5 = 30 * time.Second

func pasoPreparacionR5Coincide(q ports.SolicitudDisponibilidadFirmaR5, paso domain.CompetenciaPasoFirmaV2) bool {
	return paso.Validar() == nil && paso.Circuito.Referencia == q.CatalogoRef &&
		paso.Circuito.HuellaSHA256 == q.CatalogoHuella && paso.Documento == q.Preflight.Documento &&
		paso.PasoRef == q.PasoRef && paso.PasoOrden == uint64(q.PasoOrden) &&
		paso.OrganizacionRef == q.Preflight.Canal.OrganizacionRef
}

// El objeto Documentos ya está compuesto con repositorio, almacén y catálogo
// validados. Esta lectura no escribe ni convierte la política provisional en
// una aprobación: acredita que la cadena de custodia está preparada.
type comprobadorCustodiaPreparadaR5 struct {
	documentos *custodiaDocumentosDesarrollo
	reloj      relojContratacionTemporalDesarrollo
}

var _ comprobadorCustodiaPreparadaR5CTDesarrollo = (*comprobadorCustodiaPreparadaR5)(nil)

func nuevoComprobadorCustodiaPreparadaR5(documentos *custodiaDocumentosDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*comprobadorCustodiaPreparadaR5, error) {
	if documentos == nil || documentos.politicas == nil || documentos.reloj == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(documentos.almacen) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(documentos.repositorio) || len(documentos.documentos) == 0 {
		return nil, ports.ErrPreflightFirmaR5NoDisponible
	}
	return &comprobadorCustodiaPreparadaR5{documentos: documentos, reloj: reloj}, nil
}

func (c *comprobadorCustodiaPreparadaR5) ComprobarCustodiaPreparadaR5(ctx context.Context,
	q ports.SolicitudDisponibilidadFirmaR5, paso domain.CompetenciaPasoFirmaV2,
) (ports.EvidenciaPreparacionExternaR5, error) {
	var cero ports.EvidenciaPreparacionExternaR5
	if c == nil || c.documentos == nil || c.documentos.politicas == nil || c.documentos.reloj == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(c.documentos.almacen) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(c.documentos.repositorio) || ctx == nil ||
		!pasoPreparacionR5Coincide(q, paso) {
		return cero, ports.ErrPreparacionExternaR5NoAcreditada
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	ahora := c.reloj.Ahora()
	if !domain.InstanteUTCCanonico(ahora) {
		return cero, ports.ErrPreflightFirmaR5NoDisponible
	}
	tipo, ok := c.documentos.documentos[q.Preflight.Documento]
	if !ok {
		return cero, ports.ErrPreparacionExternaR5NoAcreditada
	}
	tipoRef, err := c.documentos.politicas.TipoDocumentalRef(tipo)
	if err != nil || !c.documentos.politicas.CustodiaFirmadoReservada(tipoRef) {
		return cero, ports.ErrPreparacionExternaR5NoAcreditada
	}
	expediente := ports.ExpedienteDocumentalRef(q.Preflight.Canal.OrganizacionRef, q.Preflight.Canal.ExpedienteRef)
	solicitud, err := c.documentos.politicas.SolicitudPara(tipo, expediente)
	if err != nil || solicitud.Validar() != nil || solicitud.TipoDocumentalRef() != tipoRef ||
		ahora.Before(solicitud.VigenteDesde()) || !ahora.Before(solicitud.VigenteHasta()) {
		return cero, ports.ErrPreparacionExternaR5NoAcreditada
	}
	politicas, err := c.documentos.politicas.BuscarPoliticasConservacionDocumental(ctx, solicitud)
	if err != nil {
		return cero, err
	}
	if len(politicas) != 1 || politicas[0].Validar() != nil || politicas[0].Solicitud() != solicitud {
		return cero, ports.ErrPreparacionExternaR5NoAcreditada
	}
	return ports.EvidenciaPreparacionExternaR5{Referencia: solicitud.PoliticaRef(), Version: solicitud.VersionPolitica(),
		HuellaSHA256: hex.EncodeToString(solicitud.HuellaPoliticaSHA256()), VigenteHasta: solicitud.VigenteHasta()}, nil
}

// Se usa con el pool del LOGIN CT ejecutor. La consulta lee solo pg_catalog:
// firma exacta y definición de CT185/v4 y AUT56/v1, pertenencia al grupo,
// privilegios de esquema y EXECUTE efectivos. No ejecuta ninguna fachada.
type lectorCatalogoRegistroR5 interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type comprobadorRegistroConPlanR5 struct {
	pool          lectorCatalogoRegistroR5
	loginEsperado string
	reloj         relojContratacionTemporalDesarrollo
}

var _ comprobadorRegistroConPlanR5CTDesarrollo = (*comprobadorRegistroConPlanR5)(nil)

func nuevoComprobadorRegistroConPlanR5(pool lectorCatalogoRegistroR5, loginEsperado string,
	reloj relojContratacionTemporalDesarrollo,
) (*comprobadorRegistroConPlanR5, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(pool) || loginEsperado == "" ||
		!domain.ReferenciaOpacaValida(loginEsperado) {
		return nil, ports.ErrPreflightFirmaR5NoDisponible
	}
	return &comprobadorRegistroConPlanR5{pool: pool, loginEsperado: loginEsperado, reloj: reloj}, nil
}

const consultarACLRegistroPlanR5 = `
SELECT session_user::text, current_user::text,
       pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER'),
       NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER'),
       NOT pg_catalog.pg_has_role(session_user,'vec_autorizacion_propietario','MEMBER'),
       pg_catalog.has_schema_privilege(session_user,'vec_contratacion_temporal','USAGE'),
       pg_catalog.has_schema_privilege(session_user,'vec_autorizacion','USAGE'),
       pg_catalog.has_function_privilege(session_user,
         pg_catalog.to_regprocedure('vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'), 'EXECUTE'),
       pg_catalog.has_function_privilege(session_user,
         pg_catalog.to_regprocedure('vec_autorizacion.seleccionar_firmante_plan_ct_v1(text,text,text,text,text,text,text,text)'), 'EXECUTE'),
       (SELECT p.proname FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure('vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),
       (SELECT p.proname FROM pg_catalog.pg_proc p WHERE p.oid=pg_catalog.to_regprocedure('vec_autorizacion.seleccionar_firmante_plan_ct_v1(text,text,text,text,text,text,text,text)')),
       pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure('vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),
       pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure('vec_autorizacion.seleccionar_firmante_plan_ct_v1(text,text,text,text,text,text,text,text)'))`

func (c *comprobadorRegistroConPlanR5) ComprobarRegistroConPlanR5(ctx context.Context,
	q ports.SolicitudDisponibilidadFirmaR5, paso domain.CompetenciaPasoFirmaV2,
) (ports.EvidenciaPreparacionExternaR5, error) {
	var cero ports.EvidenciaPreparacionExternaR5
	if c == nil || ctx == nil || dependenciaEsNulaContratacionTemporalDesarrollo(c.pool) ||
		!pasoPreparacionR5Coincide(q, paso) {
		return cero, ports.ErrPreparacionExternaR5NoAcreditada
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	var sesion, actual, nombreCT, nombreAUT, definicionCT, definicionAUT string
	var grupo, sinPropietarioCT, sinPropietarioAUT, esquemaCT, esquemaAUT, ejecutarCT, ejecutarAUT bool
	err := c.pool.QueryRow(ctx, consultarACLRegistroPlanR5).Scan(&sesion, &actual, &grupo,
		&sinPropietarioCT, &sinPropietarioAUT, &esquemaCT, &esquemaAUT,
		&ejecutarCT, &ejecutarAUT, &nombreCT, &nombreAUT, &definicionCT, &definicionAUT)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, fmt.Errorf("%w: %w", ports.ErrPreflightFirmaR5NoDisponible, err)
	}
	if sesion != c.loginEsperado || actual != sesion || !grupo || !sinPropietarioCT || !sinPropietarioAUT ||
		!esquemaCT || !esquemaAUT || !ejecutarCT || !ejecutarAUT ||
		nombreCT != "registrar_firma_con_plan_v4" || nombreAUT != "seleccionar_firmante_plan_ct_v1" ||
		definicionCT == "" || definicionAUT == "" ||
		!strings.Contains(definicionCT, "registrar_firma_con_plan_v4(") ||
		!strings.Contains(definicionAUT, "seleccionar_firmante_plan_ct_v1(") {
		return cero, ports.ErrPreparacionExternaR5NoAcreditada
	}
	ahora := c.reloj.Ahora()
	if !domain.InstanteUTCCanonico(ahora) {
		return cero, ports.ErrPreflightFirmaR5NoDisponible
	}
	h := sha256.Sum256([]byte(sesion + "\x00" + definicionCT + "\x00" + definicionAUT))
	version, err := strconv.ParseUint(strings.TrimPrefix(nombreCT, "registrar_firma_con_plan_v"), 10, 64)
	if err != nil || version == 0 {
		return cero, ports.ErrPreflightFirmaR5NoDisponible
	}
	return ports.EvidenciaPreparacionExternaR5{Referencia: "registro:ct:" + nombreCT + ":" + nombreAUT, Version: version,
		HuellaSHA256: hex.EncodeToString(h[:]), VigenteHasta: ahora.Add(plazoObservacionPreparacionR5)}, nil
}

// DescriptorConfiguracionVerificadorR5 procede de configuración privada.
// Su huella se coteja con el material exacto usado al construir el cliente;
// la revisión y caducidad no se deducen de una URL o de un token.
type DescriptorConfiguracionVerificadorR5 struct {
	Referencia   string
	Version      uint64
	HuellaSHA256 string
	VigenteHasta time.Time
}

type comprobadorConfiguracionVerificadorR5 struct {
	configuracion config.Config
	descriptor    DescriptorConfiguracionVerificadorR5
	cliente       *validadorautofirma.Cliente
	reloj         relojContratacionTemporalDesarrollo
}

var _ comprobadorConfiguracionVerificadorR5CTDesarrollo = (*comprobadorConfiguracionVerificadorR5)(nil)

// La raíz recibe el cliente y su comprobador como un par indivisible,
// construido de los mismos bytes privados leídos una sola vez. El descriptor
// fija revisión, huella y caducidad; nunca se deducen de URL o token.
func nuevosVerificadorYComprobadorPreparacionR5(cfg config.Config,
	descriptor DescriptorConfiguracionVerificadorR5, reloj relojContratacionTemporalDesarrollo,
	fuenteResultados ...func() vecports.EmisorResultadosTecnicosConContexto,
) (*validadorautofirma.Cliente, *comprobadorConfiguracionVerificadorR5, error) {
	ahora := reloj.Ahora()
	if len(fuenteResultados) > 1 || !domain.ReferenciaOpacaValida(descriptor.Referencia) || descriptor.Version == 0 ||
		!domain.HuellaSHA256FirmaValida(descriptor.HuellaSHA256) ||
		!domain.InstanteUTCCanonico(descriptor.VigenteHasta) || !ahora.Before(descriptor.VigenteHasta) {
		return nil, nil, ports.ErrPreflightFirmaR5NoDisponible
	}
	cfg = cfg.Normalize()
	material, err := cargarMaterialFirmaDocumentos(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ports.ErrPreflightFirmaR5NoDisponible, err)
	}
	defer material.borrar()
	if _, err := vigenciaMaterialVerificadorR5(material, cfg, descriptor, ahora); err != nil {
		if errors.Is(err, ports.ErrPreflightFirmaR5NoDisponible) {
			return nil, nil, err
		}
		return nil, nil, ports.ErrPreflightFirmaR5NoDisponible
	}
	material.configuracion.Disponibilidad = observadorFirmaDocumentos(fuenteResultados...)
	cliente, err := validadorautofirma.Nuevo(material.configuracion)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ports.ErrPreflightFirmaR5NoDisponible, err)
	}
	c := &comprobadorConfiguracionVerificadorR5{configuracion: cfg.Normalize(), descriptor: descriptor,
		cliente: cliente, reloj: reloj}
	return cliente, c, nil
}

func (c *comprobadorConfiguracionVerificadorR5) ComprobarConfiguracionVerificadorR5(ctx context.Context,
	q ports.SolicitudDisponibilidadFirmaR5, paso domain.CompetenciaPasoFirmaV2,
) (ports.EvidenciaPreparacionExternaR5, error) {
	var cero ports.EvidenciaPreparacionExternaR5
	if c == nil || c.cliente == nil || ctx == nil || !pasoPreparacionR5Coincide(q, paso) {
		return cero, ports.ErrPreparacionExternaR5NoAcreditada
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	ahora := c.reloj.Ahora()
	hasta, err := c.comprobarMaterial(ahora)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return ports.EvidenciaPreparacionExternaR5{Referencia: c.descriptor.Referencia,
		Version: c.descriptor.Version, HuellaSHA256: c.descriptor.HuellaSHA256, VigenteHasta: hasta}, nil
}

func (c *comprobadorConfiguracionVerificadorR5) comprobarMaterial(ahora time.Time) (time.Time, error) {
	if c == nil || !domain.InstanteUTCCanonico(ahora) ||
		!ahora.Before(c.descriptor.VigenteHasta) || c.configuracion.FirmaVerificacionEnabled != "true" {
		return time.Time{}, ports.ErrPreparacionExternaR5NoAcreditada
	}
	material, err := cargarMaterialFirmaDocumentos(c.configuracion)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %w", ports.ErrPreflightFirmaR5NoDisponible, err)
	}
	defer material.borrar()
	// Esta construcción valida de nuevo los bytes actuales sin tráfico remoto.
	if _, err := validadorautofirma.Nuevo(material.configuracion); err != nil {
		return time.Time{}, fmt.Errorf("%w: %w", ports.ErrPreflightFirmaR5NoDisponible, err)
	}
	return vigenciaMaterialVerificadorR5(material, c.configuracion, c.descriptor, ahora)
}

func vigenciaMaterialVerificadorR5(m materialFirmaDocumentos, cfg config.Config,
	descriptor DescriptorConfiguracionVerificadorR5, ahora time.Time,
) (time.Time, error) {
	if huellaMaterialVerificadorR5([]byte(cfg.FirmaVerificacionEnabled),
		[]byte(cfg.FirmaVerificacionURL), []byte(cfg.FirmaVerificacionNombreServidorTLS),
		[]byte(cfg.FirmaVerificacionTimeout), m.ca, m.token, m.certificado, m.clave) != descriptor.HuellaSHA256 {
		return time.Time{}, ports.ErrPreparacionExternaR5NoAcreditada
	}
	hasta := descriptor.VigenteHasta
	for _, certs := range [][]byte{m.ca, m.certificado} {
		if len(certs) == 0 {
			continue
		}
		vencimiento, err := menorVencimientoCertificadosR5(certs, ahora)
		if err != nil {
			return time.Time{}, ports.ErrPreparacionExternaR5NoAcreditada
		}
		if vencimiento.Before(hasta) {
			hasta = vencimiento
		}
	}
	if !ahora.Before(hasta) {
		return time.Time{}, ports.ErrPreparacionExternaR5NoAcreditada
	}
	return hasta, nil
}

func menorVencimientoCertificadosR5(material []byte, ahora time.Time) (time.Time, error) {
	var hasta time.Time
	encontrados := 0
	for len(material) > 0 {
		bloque, resto := pem.Decode(material)
		if bloque == nil {
			break
		}
		material = resto
		if bloque.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(bloque.Bytes)
		if err != nil {
			return time.Time{}, fmt.Errorf("%w: %w", ports.ErrPreflightFirmaR5NoDisponible, err)
		}
		if ahora.Before(cert.NotBefore) || !ahora.Before(cert.NotAfter) {
			return time.Time{}, ports.ErrPreparacionExternaR5NoAcreditada
		}
		if hasta.IsZero() || cert.NotAfter.Before(hasta) {
			hasta = cert.NotAfter.UTC().Truncate(time.Microsecond)
		}
		encontrados++
	}
	if encontrados == 0 {
		return time.Time{}, ports.ErrPreparacionExternaR5NoAcreditada
	}
	return hasta, nil
}

// huellaMaterialVerificadorR5 conserva el mismo formato de campo longitud
// (uint64 big endian) + bytes para preparar el descriptor privado sin revelar
// los bytes al llamador. El callback/observador no pertenece a la identidad
// criptográfica del cliente.
func huellaMaterialVerificadorR5(campos ...[]byte) string {
	h := sha256.New()
	for _, campo := range campos {
		var tamano [8]byte
		binary.BigEndian.PutUint64(tamano[:], uint64(len(campo)))
		_, _ = h.Write(tamano[:])
		_, _ = h.Write(campo)
	}
	return hex.EncodeToString(h.Sum(nil))
}
