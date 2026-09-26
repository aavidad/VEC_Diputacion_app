package domain

import (
	"errors"
	"slices"
)

var ErrPoliticaAvisosInvalida = errors.New("bolsa: politica de avisos invalida")

// Modos de la regla «ya presta servicios» (b16).
const (
	ModoPrestaServiciosAviso   = "aviso"
	ModoPrestaServiciosExcluir = "excluir"
)

// Motivos por los que una participación está «en revisión» (duda 18): hay
// una exclusión o una renuncia propuesta que RRHH aún no ha confirmado.
const (
	RevisionRenunciaPendiente  = "renuncia_pendiente"
	RevisionSolicitudPendiente = "solicitud_pendiente"
	RevisionBajaPropuesta      = "baja_propuesta"
)

// PoliticaAvisosBolsa son los parámetros de los avisos y marcas de Bolsa que
// fija el catálogo de reglas (b16, b17 y b19) y que la base aplica (000041).
// Un campo a cero no está configurado: el plazo de trabajo continuado
// conserva el de la migración (la conducta anterior) y el encadenamiento y
// la marca de servicios quedan desactivados.
type PoliticaAvisosBolsa struct {
	ContinuadoMeses            int
	ContinuadoAntelacionDias   int
	ContinuadoConfigurado      bool
	EncadenamientoUmbralMeses  int
	EncadenamientoVentanaMeses int
	PrestaServiciosModo        string
	PrestaServiciosSituaciones []string
}

// Validar reproduce los límites de la base para fallar antes de publicar.
func (p PoliticaAvisosBolsa) Validar() error {
	if p.ContinuadoConfigurado && (p.ContinuadoMeses < 1 || p.ContinuadoMeses > 600 || p.ContinuadoAntelacionDias < 0 || p.ContinuadoAntelacionDias > 3650) {
		return ErrPoliticaAvisosInvalida
	}
	if !p.ContinuadoConfigurado && (p.ContinuadoMeses != 0 || p.ContinuadoAntelacionDias != 0) {
		return ErrPoliticaAvisosInvalida
	}
	if (p.EncadenamientoUmbralMeses == 0) != (p.EncadenamientoVentanaMeses == 0) ||
		p.EncadenamientoUmbralMeses < 0 || p.EncadenamientoUmbralMeses > 600 || p.EncadenamientoVentanaMeses > 1200 ||
		p.EncadenamientoUmbralMeses > p.EncadenamientoVentanaMeses {
		return ErrPoliticaAvisosInvalida
	}
	if (p.PrestaServiciosModo == "") != (len(p.PrestaServiciosSituaciones) == 0) {
		return ErrPoliticaAvisosInvalida
	}
	if p.PrestaServiciosModo != "" && p.PrestaServiciosModo != ModoPrestaServiciosAviso && p.PrestaServiciosModo != ModoPrestaServiciosExcluir {
		return ErrPoliticaAvisosInvalida
	}
	vistas := make(map[string]struct{}, len(p.PrestaServiciosSituaciones))
	for _, situacion := range p.PrestaServiciosSituaciones {
		if _, repetida := vistas[situacion]; repetida || situacion == SituacionDisponible || !slices.Contains(SituacionesParticipacion(), situacion) {
			return ErrPoliticaAvisosInvalida
		}
		vistas[situacion] = struct{}{}
	}
	return nil
}

// MarcasParticipacion son los rótulos que el cuadro y la ficha muestran junto
// a la situación. Ninguno cambia situación, orden ni llamamiento.
type MarcasParticipacion struct {
	ParticipacionRef string
	// PrestaServicios es el modo de b16 si la persona trabaja ya por otra
	// participación; vacío si no.
	PrestaServicios string
	// EnRevision es uno de los motivos Revision*; vacío si no.
	EnRevision string
	// Encadenamiento: días de contrato en la ventana, solo si superan el
	// umbral de b17.
	EncadenamientoDias         int
	EncadenamientoUmbralMeses  int
	EncadenamientoVentanaMeses int
}
