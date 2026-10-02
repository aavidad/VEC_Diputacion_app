package http

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type guardarPreparacionJSON struct {
	Esperada       prep.Esperada `json:"esperada"`
	Material       prep.Material `json:"material"`
	ClaveOperacion string        `json:"clave_operacion"`
}

type consultarPreparacionJSON struct {
	Modo       string `json:"modo"`
	Referencia string `json:"preparacion_ref"`
	Revision   int    `json:"revision"`
	Huella     string `json:"huella_material_sha256"`
}

func (q consultarPreparacionJSON) Exacta() prep.Esperada {
	return prep.Esperada{PreparacionRef: q.Referencia, Revision: q.Revision, HuellaMaterialSHA256: q.Huella}
}

func leerPreparacionJSON(lector io.Reader, destino any) error {
	b, err := io.ReadAll(lector)
	if err != nil || !utf8.Valid(b) {
		return ports.ErrPreparacionBasesInvalida
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if validarObjetoPreparacion(d, 0) != nil {
		return ports.ErrPreparacionBasesInvalida
	}
	if _, err := d.Token(); err != io.EOF {
		return ports.ErrPreparacionBasesInvalida
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil {
		return ports.ErrPreparacionBasesInvalida
	}
	return nil
}

// Validacion de transporte: duplicados, alias de mayusculas y profundidad.
// Los campos y reglas de preparacion pertenecen al dominio de Bolsa.
func validarObjetoPreparacion(d *json.Decoder, profundidad int) error {
	if profundidad > 20 {
		return ports.ErrPreparacionBasesInvalida
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	inicio, contenedor := t.(json.Delim)
	if !contenedor {
		return nil
	}
	if inicio != '{' && inicio != '[' {
		return ports.ErrPreparacionBasesInvalida
	}
	vistos := map[string]bool{}
	for d.More() {
		if inicio == '{' {
			k, err := d.Token()
			if err != nil {
				return err
			}
			clave, ok := k.(string)
			if !ok || vistos[clave] || clave != strings.ToLower(clave) {
				return ports.ErrPreparacionBasesInvalida
			}
			vistos[clave] = true
		}
		if err := validarObjetoPreparacion(d, profundidad+1); err != nil {
			return err
		}
	}
	fin, err := d.Token()
	if err != nil || (inicio == '{' && fin != json.Delim('}')) || (inicio == '[' && fin != json.Delim(']')) {
		return ports.ErrPreparacionBasesInvalida
	}
	return nil
}

type accesoPreparacionJSON struct {
	DecisionRef    string    `json:"decision_ref"`
	ConsumoHuella  string    `json:"consumo_huella_sha256"`
	AuditoriaRef   string    `json:"auditoria_ref"`
	ReciboRef      string    `json:"recibo_ref"`
	CorrelacionRef string    `json:"correlacion_ref"`
	AccedidaEn     time.Time `json:"accedida_en"`
}

func accesoPreparacion(a ports.EvidenciaAccesoPreparacionBasesV3) accesoPreparacionJSON {
	return accesoPreparacionJSON{a.DecisionRef, a.ConsumoHuellaSHA256, a.AuditoriaRef, a.ReciboRef, a.CorrelacionRef, a.AccedidaEn}
}

type reciboPreparacionJSON struct {
	Referencia      string    `json:"recibo_ref"`
	Historia        string    `json:"historia_ref"`
	Auditoria       string    `json:"auditoria_ref"`
	Evento          string    `json:"evento_ref"`
	HuellaIntencion string    `json:"huella_intencion_sha256"`
	ConfirmadaEn    time.Time `json:"confirmada_en"`
}

type respuestaPreparacionJSON struct {
	Estado     string                `json:"estado"`
	Version    prep.Version          `json:"preparacion"`
	Pendientes []prep.Pendiente      `json:"pendientes"`
	Recibo     reciboPreparacionJSON `json:"recibo"`
	Acceso     accesoPreparacionJSON `json:"acceso"`
}

func respuestaPreparacionBases(r ports.ResultadoPreparacionBasesV3) (respuestaPreparacionJSON, error) {
	var cero respuestaPreparacionJSON
	if r.Version.Validar() != nil || (r.Estado != "guardada" && r.Estado != "recuperada" && r.Estado != "obtenida") {
		return cero, ports.ErrResultadoPreparacionBasesInvalido
	}
	pendientes, err := r.Version.Material.Pendientes()
	if err != nil {
		return cero, err
	}
	x := r.Recibo
	return respuestaPreparacionJSON{r.Estado, r.Version, pendientes, reciboPreparacionJSON{x.ReciboRef, x.HistoriaRef, x.AuditoriaRef, x.EventoRef, x.HuellaIntencionSHA256, x.ConfirmadaEn}, accesoPreparacion(r.Acceso)}, nil
}
