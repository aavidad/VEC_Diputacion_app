package bootstrap

import (
	"context"

	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	dochttp "vec-diputacion-granada/internal/vec/documentos/adapters/httpinterno"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const finalidadRegistrarDocumentoExterno = "registrar_documento_externo"

// registroExternoDocumentosDesarrollo es la parte opcional del material
// privado que publica POST /api/vec/documentos/externos/registros. Sin ella la
// ruta no existe. Qué tipos se admiten, para qué módulo y con qué custodio es
// configuración de RRHH, no una constante del código; el motivo V3 pertenece
// al mismo catálogo de motivos que la consulta.
type registroExternoDocumentosDesarrollo struct {
	Motivo    core.ReferenciaEntradaCatalogo      `json:"motivo"`
	Admitidos []admitidoRegistroExternoDesarrollo `json:"admitidos"`
}

type admitidoRegistroExternoDesarrollo struct {
	PrefijoTipo string `json:"prefijo_tipo"`
	ModuloID    string `json:"modulo_id"`
	CustodioID  string `json:"custodio_id"`
}

func (r *registroExternoDocumentosDesarrollo) valido(catalogoMotivos string) bool {
	return r == nil || (r.Motivo.Validar() == nil && r.Motivo.CatalogoID == catalogoMotivos &&
		len(r.Admitidos) > 0 && len(r.Admitidos) <= 32)
}

// autorizadorRegistroExternoDocumentos pide al PDP V3, con la identidad que
// la frontera registró para esta petición, la concesión ligada a la preimagen
// exacta del registro externo.
type autorizadorRegistroExternoDocumentos struct {
	consulta autoridadConsultaDocumentos
}

func (a autorizadorRegistroExternoDocumentos) AutorizarRegistroExterno(ctx context.Context, preimagen []byte, documentoID, expedienteRef string) (docports.AutorizacionV3, error) {
	return a.consulta.autorizar(ctx, docports.AccionRegistrarExterno, finalidadRegistrarDocumentoExterno, documentoID, expedienteRef, preimagen)
}

// rutaRegistroExternoDocumentos devuelve la ruta compuesta, o ninguna si el
// material no declara registro externo.
func rutaRegistroExternoDocumentos(
	c *registroExternoDocumentosDesarrollo,
	servicio dochttp.ServicioRegistroExterno,
	consulta autoridadConsultaDocumentos,
	politicas dochttp.PoliticasRegistroExterno,
	incidencias vecports.EmisorIncidenciasTecnicas,
) ([]vechttp.RutaExacta, error) {
	if c == nil {
		return nil, nil
	}
	admitidos := make([]dochttp.RegistroExternoAdmitido, 0, len(c.Admitidos))
	for _, a := range c.Admitidos {
		admitidos = append(admitidos, dochttp.RegistroExternoAdmitido{PrefijoTipo: a.PrefijoTipo, ModuloID: a.ModuloID, CustodioID: a.CustodioID})
	}
	consulta.motivo = c.Motivo
	ruta, err := dochttp.NuevaRutaRegistroExterno(dochttp.ConfiguracionRegistroExterno{
		Servicio: servicio, Autoridad: autorizadorRegistroExternoDocumentos{consulta: consulta},
		Politicas: politicas, Admitidos: admitidos, Incidencias: incidencias,
	})
	if err != nil {
		return nil, err
	}
	return []vechttp.RutaExacta{ruta}, nil
}
