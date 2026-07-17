package gobiernoreglasbaremo

import reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"

// OperacionGobiernoReglasBaremoV2 es un catalogo compilado y cerrado. No es
// texto procedente de formularios, JSON ni configuracion.
type OperacionGobiernoReglasBaremoV2 uint8

const (
	OperacionNoDeclarada OperacionGobiernoReglasBaremoV2 = iota
	OperacionAltaBorrador
	OperacionPublicar
	OperacionActivar
	OperacionSustituir
	OperacionRetirar
	OperacionDescartar
	OperacionConsultaExacta
)

func (o OperacionGobiernoReglasBaremoV2) valida() bool {
	return o >= OperacionAltaBorrador && o <= OperacionConsultaExacta
}

func (o OperacionGobiernoReglasBaremoV2) esCambio() bool {
	return o >= OperacionAltaBorrador && o <= OperacionDescartar
}

func (o OperacionGobiernoReglasBaremoV2) nombreCanonico() (string, error) {
	switch o {
	case OperacionAltaBorrador:
		return "alta_borrador", nil
	case OperacionPublicar:
		return "publicar", nil
	case OperacionActivar:
		return "activar", nil
	case OperacionSustituir:
		return "sustituir", nil
	case OperacionRetirar:
		return "retirar", nil
	case OperacionDescartar:
		return "descartar", nil
	case OperacionConsultaExacta:
		return "consultar_version_exacta", nil
	default:
		return "", ErrOperacionInvalida
	}
}

func (o OperacionGobiernoReglasBaremoV2) estadoResultado() (
	reglas.EstadoGobiernoReglasBaremo,
	uint64,
	bool,
) {
	switch o {
	case OperacionAltaBorrador:
		return reglas.EstadoReglasBaremoBorrador, 1, true
	case OperacionPublicar:
		return reglas.EstadoReglasBaremoPublicada, 2, true
	case OperacionActivar:
		return reglas.EstadoReglasBaremoActiva, 3, true
	case OperacionSustituir:
		return reglas.EstadoReglasBaremoSustituida, 4, true
	case OperacionRetirar:
		return reglas.EstadoReglasBaremoRetirada, 4, true
	case OperacionDescartar:
		return reglas.EstadoReglasBaremoDescartada, 2, true
	default:
		return "", 0, false
	}
}

// ComponenteEscrituraReglasBaremoV2 enumera todas las partes del efecto. La
// lista es fija incluso cuando CAS o evidencia se materializan como ausencia
// explicita en un alta.
type ComponenteEscrituraReglasBaremoV2 uint8

const (
	ComponenteNoDeclarado ComponenteEscrituraReglasBaremoV2 = iota
	ComponenteContenido
	ComponenteVersion
	ComponentePunteroCAS
	ComponenteVinculoEvidencia
	ComponenteConsumoVEC
	ComponenteAuditoria
	ComponenteOutbox
	// ComponenteRecibo obliga al ejecutor a generar el recibo en el mismo
	// COMMIT. El plan no inventa ni anticipa su identidad.
	ComponenteRecibo
)

func componentesEscrituraFijos() []ComponenteEscrituraReglasBaremoV2 {
	return []ComponenteEscrituraReglasBaremoV2{
		ComponenteContenido,
		ComponenteVersion,
		ComponentePunteroCAS,
		ComponenteVinculoEvidencia,
		ComponenteConsumoVEC,
		ComponenteAuditoria,
		ComponenteOutbox,
		ComponenteRecibo,
	}
}

func (c ComponenteEscrituraReglasBaremoV2) nombreCanonico() (string, error) {
	switch c {
	case ComponenteContenido:
		return "contenido", nil
	case ComponenteVersion:
		return "version", nil
	case ComponentePunteroCAS:
		return "puntero_cas", nil
	case ComponenteVinculoEvidencia:
		return "vinculo_evidencia", nil
	case ComponenteConsumoVEC:
		return "vec", nil
	case ComponenteAuditoria:
		return "auditoria", nil
	case ComponenteOutbox:
		return "outbox", nil
	case ComponenteRecibo:
		return "recibo", nil
	default:
		return "", ErrPlanCambioInvalido
	}
}
