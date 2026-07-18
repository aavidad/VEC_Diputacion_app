package pagos

import (
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// MetodoHandoffCobro es una lista cerrada de mecanismos de entrega al cliente.
type MetodoHandoffCobro string

const MetodoHandoffCobroPOSTFormulario MetodoHandoffCobro = "post_formulario"

// Validar comprueba tambien que la huella corresponda a la configuracion.
func (o OrigenPasarelaCobroPublicado) Validar() error {
	huella, err := HuellaConfiguracionOrigen(o)
	if err != nil || !HuellaSHA256Valida(o.HuellaConfiguracionSHA256) ||
		o.HuellaConfiguracionSHA256 != huella {
		return ErrInicioOperacionCobroInvalido
	}
	return nil
}

// CampoHandoffCobro es un par permitido de la carga opaca de handoff.
type CampoHandoffCobro struct {
	Nombre string
	Valor  string
}

// CargaHandoffCobro oculta la carga para impedir serializarla o construirla
// sin pasar por NuevaCargaHandoffCobro.
type CargaHandoffCobro struct{ campos []CampoHandoffCobro }

// NuevaCargaHandoffCobro copia y valida una carga contra una lista cerrada.
func NuevaCargaHandoffCobro(campos []CampoHandoffCobro, permitidos []string) (CargaHandoffCobro, error) {
	if len(campos) == 0 || len(campos) > 32 || !ListaCerradaValida(permitidos, false) {
		return CargaHandoffCobro{}, ErrInicioOperacionCobroInvalido
	}
	permitido := make(map[string]struct{}, len(permitidos))
	for _, campo := range permitidos {
		permitido[campo] = struct{}{}
	}
	vistos := make(map[string]struct{}, len(campos))
	copia := make([]CampoHandoffCobro, len(campos))
	for indice, campo := range campos {
		if !ClaveValida(campo.Nombre) || !TextoValido(campo.Valor, 4096) || ContieneDatoTarjeta(campo.Valor) {
			return CargaHandoffCobro{}, ErrInicioOperacionCobroInvalido
		}
		if _, existe := permitido[campo.Nombre]; !existe {
			return CargaHandoffCobro{}, ErrInicioOperacionCobroInvalido
		}
		if _, repetido := vistos[campo.Nombre]; repetido {
			return CargaHandoffCobro{}, ErrInicioOperacionCobroInvalido
		}
		vistos[campo.Nombre] = struct{}{}
		copia[indice] = campo
	}
	return CargaHandoffCobro{campos: copia}, nil
}

func (c CargaHandoffCobro) copiarCampos() ([]CampoHandoffCobro, error) {
	if len(c.campos) == 0 {
		return nil, ErrInicioOperacionCobroInvalido
	}
	return append([]CampoHandoffCobro(nil), c.campos...), nil
}

func (CargaHandoffCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrInicioOperacionCobroInvalido
}
func (CargaHandoffCobro) String() string     { return "[CARGA-HANDOFF-OCULTA]" }
func (c CargaHandoffCobro) GoString() string { return c.String() }

// InicioOperacionCobro separa el origen publicado de la carga de handoff. No
// acepta una URL devuelta libremente por el proveedor ni secretos en query.
type InicioOperacionCobro struct {
	Evidencia                 domain.EvidenciaInicioOperacionCobro
	Origen                    OrigenPasarelaCobroPublicado
	VersionOrden              int
	HuellaOrdenSHA256         string
	HuellaConfiguracionSHA256 string
	Ruta                      string
	Metodo                    MetodoHandoffCobro
	Carga                     CargaHandoffCobro
	GeneradaEn                time.Time
	ExpiraEn                  time.Time
}

// Validar comprueba la estructura. La entrega exige ademas ValidarContra.
func (i InicioOperacionCobro) Validar() error {
	control, errControl := i.Evidencia.Control()
	if i.Origen.Validar() != nil || i.VersionOrden < 1 ||
		!HuellaSHA256Valida(i.HuellaOrdenSHA256) ||
		!HuellaSHA256Valida(i.HuellaConfiguracionSHA256) ||
		i.HuellaConfiguracionSHA256 != i.Origen.HuellaConfiguracionSHA256 ||
		i.Metodo != MetodoHandoffCobroPOSTFormulario ||
		i.GeneradaEn.IsZero() || !i.ExpiraEn.After(i.GeneradaEn) ||
		i.ExpiraEn.Sub(i.GeneradaEn) > 15*time.Minute || i.GeneradaEn.Before(i.Origen.PublicadaEn) ||
		errControl != nil || control.ConectorID != i.Origen.ID || control.VersionConector != i.Origen.Version ||
		i.GeneradaEn.Before(control.RecibidaEn) || i.GeneradaEn.Sub(control.RecibidaEn) > 2*time.Minute ||
		len(i.Ruta) > 1024 || !ContieneCadenaExacta(i.Origen.RutasPermitidas, i.Ruta) ||
		!RutaHandoffValida(i.Ruta) {
		return ErrInicioOperacionCobroInvalido
	}
	campos, err := i.Carga.copiarCampos()
	if err != nil || !camposHandoffCoinciden(campos, i.Origen.CamposHandoffPermitidos) {
		return ErrInicioOperacionCobroInvalido
	}
	return nil
}

// ValidarContra liga la respuesta al comando sellado, al origen publicado y
// al reloj confiable. Es la unica validacion suficiente para entregar handoff.
func (i InicioOperacionCobro) ValidarContra(
	comando domain.ComandoInicioOperacionCobro,
	origen OrigenPasarelaCobroPublicado,
	ahora time.Time,
) error {
	datos, errComando := comando.Datos()
	control, errControl := i.Evidencia.Control()
	if i.Validar() != nil || errComando != nil || errControl != nil || origen.Validar() != nil || ahora.IsZero() ||
		!ConfiguracionesOrigenIguales(i.Origen, origen) ||
		i.VersionOrden != datos.VersionOrden || i.HuellaOrdenSHA256 != datos.HuellaOrdenSHA256 ||
		i.HuellaConfiguracionSHA256 != origen.HuellaConfiguracionSHA256 ||
		control.OrdenRef != datos.OrdenRef || control.LiquidacionRef != datos.LiquidacionRef ||
		!control.Importe.Igual(datos.Importe) || control.Concepto != datos.Concepto ||
		ahora.UTC().Before(i.GeneradaEn.UTC()) || !ahora.UTC().Before(i.ExpiraEn.UTC()) ||
		i.ExpiraEn.After(datos.CaducaEn) {
		return ErrInicioOperacionCobroInvalido
	}
	return nil
}

// CamposRespuestaPOSTContra devuelve una copia solo tras la validacion
// completa. El consumo unico corresponde a una custodia durable externa.
func (i InicioOperacionCobro) CamposRespuestaPOSTContra(
	comando domain.ComandoInicioOperacionCobro,
	origen OrigenPasarelaCobroPublicado,
	ahora time.Time,
) ([]CampoHandoffCobro, error) {
	if i.ValidarContra(comando, origen, ahora) != nil {
		return nil, ErrInicioOperacionCobroInvalido
	}
	return i.Carga.copiarCampos()
}

func camposHandoffCoinciden(campos []CampoHandoffCobro, permitidos []string) bool {
	for _, campo := range campos {
		if !ContieneCadenaExacta(permitidos, campo.Nombre) {
			return false
		}
	}
	return len(campos) > 0
}

var _ json.Marshaler = CargaHandoffCobro{}
