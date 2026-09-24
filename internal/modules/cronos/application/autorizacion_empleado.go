package application

import (
	"crypto/sha256"
	"encoding/hex"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Contrato V3 de las lecturas propias de la persona empleada. Cada acción
// tiene su audiencia nominal (AD3-53) y la función durable de cronos_v1
// 000007 comprueba exactamente estos valores antes de consumir.
const (
	AudienciaConsultaSaldoPropio  = "vec_cronos_v1.saldo_propio.consultar.v1"
	AccionConsultarSaldoPropio    = "cronos.saldo.propio.consultar"
	FinalidadConsultarSaldoPropio = "consultar_saldo_propio"

	AudienciaDisponibilidadMarcajeRemoto   = "vec_cronos_v1.marcaje_remoto_disponibilidad.v1"
	AccionConsultarDisponibilidadRemota    = "cronos.marcaje.remoto.disponibilidad.consultar"
	FinalidadConsultarDisponibilidadRemota = "consultar_disponibilidad_marcaje_remoto"

	FinalidadRecuperarMarcajeRemoto = "recuperar_recibo_marcaje_remoto"
)

func recursoEmpleado(referencia, tipo, empleado string, canonico []byte) (vecdomain.RecursoAutorizable, error) {
	h := sha256.Sum256(canonico)
	r := vecdomain.RecursoAutorizable{
		Referencia: referencia, ModuloID: "cronos", Tipo: tipo,
		Ambitos:   map[string]string{"empleado_ref": empleado},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}
	if r.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return r, nil
}

// RecursoConsultaSaldoPropio liga la lectura al periodo exacto por la huella
// del material; la referencia identifica el saldo de la persona.
func RecursoConsultaSaldoPropio(m domain.MaterialConsultaSaldoPropio) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoEmpleado("saldo:cronos:"+m.EmpleadoRef, "saldo_propio", m.EmpleadoRef, canonico)
}

func RecursoDisponibilidadMarcajeRemoto(m domain.MaterialDisponibilidadMarcajeRemoto) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoEmpleado("teletrabajo:cronos:"+m.EmpleadoRef, "marcaje_remoto_disponibilidad", m.EmpleadoRef, canonico)
}
