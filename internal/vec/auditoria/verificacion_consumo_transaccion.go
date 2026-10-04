package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

const TipoConsumoTransaccionV4 = "consumo_confirmado_v4"

// RegistroConsumoTransaccionV4 conserva los dos sellos xid8 como texto,
// sin convertirlos a enteros JSON ni inferir una transacción global.
type RegistroConsumoTransaccionV4 struct {
	RegistroConsumoFechaV3
	TransaccionOrigen        string `json:"transaccion_origen"`
	TransaccionConsumoOrigen string `json:"transaccion_consumo_origen"`
}

// CotejarConsumoTransaccionV4 comprueba la proyección AD193 y su preimagen
// de dieciséis campos. No verifica COSE, origen del archivo ni identidad.
func CotejarConsumoTransaccionV4(r RegistroConsumoTransaccionV4) *FalloVerificacion {
	fallar := func(codigo, clave string) *FalloVerificacion {
		return &FalloVerificacion{Codigo: codigo, Clave: clave,
			Esperado: "asiento_ad193_valido", Obtenido: "incompatible", Secuencia: r.Secuencia}
	}
	if r.TipoRegistro != TipoConsumoTransaccionV4 || r.VersionConsumo != 4 {
		return fallar("tipo_invalido", "tipo_version_consumo")
	}
	if fallo := cotejarCamposConsumoFecha(r.RegistroConsumoFechaV3); fallo != nil {
		fallo.Esperado = "asiento_ad193_valido"
		return fallo
	}
	for _, campo := range []struct{ clave, valor string }{
		{"transaccion_origen", r.TransaccionOrigen}, {"transaccion_consumo_origen", r.TransaccionConsumoOrigen},
	} {
		if !transaccionOrigenCanonica(campo.valor) {
			return fallar("transaccion_invalida", campo.clave)
		}
	}
	if r.TransaccionOrigen != r.TransaccionConsumoOrigen {
		return fallar("transaccion_distinta", "transaccion_origen")
	}
	if r.HuellaSHA256 != huellaConsumoTransaccionV4(r) {
		return fallar("huella_distinta", "huella_sha256")
	}
	return nil
}

func transaccionOrigenCanonica(valor string) bool {
	if len(valor) == 0 || len(valor) > 20 {
		return false
	}
	n, err := strconv.ParseUint(valor, 10, 64)
	return err == nil && n > 0 && strconv.FormatUint(n, 10) == valor
}

func huellaConsumoTransaccionV4(r RegistroConsumoTransaccionV4) string {
	h := sha256.New()
	for _, valor := range []string{TipoConsumoTransaccionV4, "4", strconv.FormatUint(r.Secuencia, 10),
		r.AnteriorSHA256, r.DecisionRef, r.EfectoRef, r.HuellaEfectoSHA256,
		r.ConsumoHuellaSHA256, r.Proceso, r.Canal, r.ConsumidaEn, r.RegistradaEn,
		r.ActorRef, r.PerfilActivoRef, r.FinalidadRef, r.TransaccionOrigen} {
		_, _ = h.Write(encuadrarVerificacion(valor))
	}
	return hex.EncodeToString(h.Sum(nil))
}
