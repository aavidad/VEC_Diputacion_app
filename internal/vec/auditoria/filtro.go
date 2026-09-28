package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const formatoInstante = "2006-01-02T15:04:05.000000Z"

var camposPermitidos = []string{
	"accion", "actor_ref", "antes", "antes_sha256", "datos_disponibles",
	"despues", "despues_sha256", "expediente_ref", "fuente", "id",
	"modulo_id", "motivo", "ocurrido_en", "recibo_ref", "resultado",
}

func CamposPermitidos() []string { return append([]string(nil), camposPermitidos...) }

func referenciaExacta(valor string, maximo int) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximo ||
		!utf8.ValidString(valor) || strings.ContainsAny(valor, "*?") {
		return false
	}
	for _, r := range valor {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func instanteValido(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC &&
		t.Nanosecond()%1000 == 0 && t.Year() >= 1 && t.Year() <= 9999
}

func (p Posicion) vacia() bool { return p.OcurridoEn.IsZero() && p.Fuente == "" && p.ID == "" }

func (p Posicion) validar() bool {
	return p.vacia() || (instanteValido(p.OcurridoEn) &&
		(p.Fuente == "ct" || p.Fuente == "bolsa") &&
		referenciaExacta(p.ID, 512))
}

func (f Filtro) Validar() error {
	if (f.Fuente != "ct" && f.Fuente != "bolsa") || !referenciaExacta(f.ExpedienteRef, 512) ||
		(f.ActorRef != "" && !referenciaExacta(f.ActorRef, 512)) ||
		!instanteValido(f.Desde) || !instanteValido(f.Hasta) ||
		!f.Hasta.After(f.Desde) || f.Hasta.Sub(f.Desde) > MaximoIntervalo ||
		f.Limite == 0 || f.Limite > MaximoRegistros || !f.Antes.validar() ||
		(!f.Antes.vacia() && (f.Antes.OcurridoEn.Before(f.Desde) || !f.Antes.OcurridoEn.Before(f.Hasta))) ||
		!referenciaExacta(f.FinalidadRef, 128) || !referenciaExacta(f.MotivoRef, 128) {
		return ErrDenegada
	}
	return nil
}

// HuellaFiltro es la ligadura reproducible para los consumidores SQL CT y
// Bolsa. El separador LF es inequívoco: ninguna referencia admite controles.
func HuellaFiltro(f Filtro) (string, error) {
	if f.Validar() != nil {
		return "", ErrDenegada
	}
	antes := ""
	if !f.Antes.vacia() {
		antes = f.Antes.OcurridoEn.Format(formatoInstante)
	}
	canon := strings.Join([]string{
		"vec.auditoria.filtro.v1", f.Fuente, f.ExpedienteRef, f.ActorRef,
		f.Desde.Format(formatoInstante), f.Hasta.Format(formatoInstante),
		strconv.FormatUint(uint64(f.Limite), 10), antes, f.Antes.Fuente,
		f.Antes.ID, f.FinalidadRef, f.MotivoRef,
	}, "\n")
	h := sha256.Sum256([]byte(canon))
	return hex.EncodeToString(h[:]), nil
}

// RecursoFiltro mantiene el expediente exacto como ámbito positivo y liga
// todos los filtros, incluida la página, a la decisión V3.
func RecursoFiltro(f Filtro) (vecdomain.RecursoAutorizable, error) {
	huella, err := HuellaFiltro(f)
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrDenegada
	}
	r := vecdomain.RecursoAutorizable{
		Referencia: f.ExpedienteRef, ModuloID: ModuloAutorizacion, Tipo: TipoRecurso,
		Ambitos:   map[string]string{"expediente_ref": f.ExpedienteRef, "fuente": f.Fuente},
		Atributos: map[string]string{"filtro_sha256": huella},
	}
	if r.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ErrDenegada
	}
	return r, nil
}
