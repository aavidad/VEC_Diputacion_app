package bootstrap

import (
	"errors"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// custodiaFirmadoConfigDesarrollo es la sección opcional del material de
// Documentos que decide qué documentos del circuito de firma de Contratación
// temporal se custodian y con qué tipo documental del catálogo de
// conservación. Es configuración de RRHH (duda 74), no una constante.
type custodiaFirmadoConfigDesarrollo struct {
	Documentos map[string]string `json:"documentos"`
}

var errCustodiaDocumentosConfiguracion = errors.New("bootstrap: custodia de documentos firmados mal configurada")

// custodiaDocumentosDesarrollo reúne lo que la custodia comparte con el resto
// de Documentos. El servicio de custodia se crea con la fábrica de concesiones
// que aporta quien custodia (Contratación temporal).
type custodiaDocumentosDesarrollo struct {
	repositorio    docports.RepositorioCustodiaFirmado
	almacen        vecports.AlmacenObjetos
	politicas      *conservacion.Catalogo
	reloj          vecports.Reloj
	seudonimizador *seudonimizadorAlmacenDesarrollo
	documentos     map[string]string
}

// nuevaCustodiaDocumentosDesarrollo devuelve nil sin sección. Con ella exige
// seudonimizador del almacén y que cada tipo esté reservado a la custodia de
// firmados en el catálogo de conservación; si no, falla cerrado.
func nuevaCustodiaDocumentosDesarrollo(c *custodiaFirmadoConfigDesarrollo, repositorio docports.RepositorioCustodiaFirmado,
	almacen vecports.AlmacenObjetos, politicas *conservacion.Catalogo, reloj vecports.Reloj, seudonimizador *seudonimizadorAlmacenDesarrollo,
) (*custodiaDocumentosDesarrollo, error) {
	if c == nil {
		return nil, nil
	}
	if len(c.Documentos) == 0 || len(c.Documentos) > 32 || !seudonimizador.valido() || politicas == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(repositorio) || dependenciaEsNulaContratacionTemporalDesarrollo(almacen) {
		return nil, errCustodiaDocumentosConfiguracion
	}
	documentos := make(map[string]string, len(c.Documentos))
	for documento, tipo := range c.Documentos {
		ref, err := politicas.TipoDocumentalRef(tipo)
		if !ctdomain.ClaveDocumentoFirmaValida(documento) || err != nil || !politicas.CustodiaFirmadoReservada(ref) {
			return nil, errCustodiaDocumentosConfiguracion
		}
		documentos[documento] = tipo
	}
	return &custodiaDocumentosDesarrollo{repositorio: repositorio, almacen: almacen, politicas: politicas, reloj: reloj,
		seudonimizador: seudonimizador, documentos: documentos}, nil
}

// servicio es un Servicio de Documentos que solo custodia firmados: comparte
// repositorio, almacén, catálogo y reloj con el resto de Documentos.
func (c *custodiaDocumentosDesarrollo) servicio(fabrica docports.FabricaContextoCustodia) *docapp.Servicio {
	return &docapp.Servicio{Almacen: c.almacen, Politicas: c.politicas, Reloj: c.reloj,
		RepositorioCustodia: c.repositorio, ContextosCustodia: fabrica}
}
