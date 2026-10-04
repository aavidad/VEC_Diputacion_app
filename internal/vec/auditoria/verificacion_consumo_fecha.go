package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

const TipoConsumoFechaV3 = "consumo_confirmado_v3"

// RegistroConsumoFechaV3 proyecta el asiento AD173. Actor, perfil y finalidad
// proceden de la decisión V3 validada por el registrador; las fechas proceden
// del mismo instante de consumo. El cotejo no autentica esa fuente ni el COSE.
type RegistroConsumoFechaV3 struct {
	RegistroConsumoOrigenV2
	RegistradaEn    string `json:"registrada_en"`
	ConsumidaEn     string `json:"consumida_en"`
	ActorRef        string `json:"actor_ref"`
	PerfilActivoRef string `json:"perfil_activo_ref"`
	FinalidadRef    string `json:"finalidad_ref"`
}

// CotejarConsumoFechaV3 comprueba forma y huella de un asiento AD173.
// No verifica la firma COSE, la identidad del actor ni la confianza del
// checkpoint. Orden, unicidad y enlaces pertenecen al cotejo de la cadena.
func CotejarConsumoFechaV3(r RegistroConsumoFechaV3) *FalloVerificacion {
	fallar := func(codigo, clave string) *FalloVerificacion {
		return &FalloVerificacion{Codigo: codigo, Clave: clave,
			Esperado: "asiento_ad173_valido", Obtenido: "incompatible", Secuencia: r.Secuencia}
	}
	if r.TipoRegistro != TipoConsumoFechaV3 || r.VersionConsumo != 3 {
		return fallar("tipo_invalido", "tipo_version_consumo")
	}
	if fallo := cotejarCamposConsumoFecha(r); fallo != nil {
		return fallo
	}
	if r.HuellaSHA256 != huellaConsumoFechaV3(r) {
		return fallar("huella_distinta", "huella_sha256")
	}
	return nil
}

// Las versiones v3 y v4 comparten coordenadas, origen, fechas y referencias.
// Cada versión comprueba por separado su discriminador y su preimagen.
func cotejarCamposConsumoFecha(r RegistroConsumoFechaV3) *FalloVerificacion {
	fallar := func(codigo, clave string) *FalloVerificacion {
		return &FalloVerificacion{Codigo: codigo, Clave: clave,
			Esperado: "asiento_ad173_valido", Obtenido: "incompatible", Secuencia: r.Secuencia}
	}
	if r.Secuencia == 0 || r.Secuencia > maxSecuenciaVerificacion ||
		!referenciaCadenaValida(r.DecisionRef) || !referenciaCadenaValida(r.EfectoRef) ||
		!huellaCadenaValida(r.HuellaEfectoSHA256) || !huellaCadenaValida(r.ConsumoHuellaSHA256) ||
		!huellaCadenaValida(r.AnteriorSHA256) || !huellaCadenaValida(r.HuellaSHA256) ||
		!procesoOrdenAD169.MatchString(r.Proceso) {
		return fallar("registro_invalido", "coordenadas_origen")
	}
	switch r.Canal {
	case "interna_corporativa", "administracion_privilegiada", "externa_personal":
	default:
		return fallar("registro_invalido", "canal")
	}
	if r.AuditoriaRef != "aud_v3_"+r.ConsumoHuellaSHA256[:32] {
		return fallar("referencia_distinta", "auditoria_ref")
	}
	for _, campo := range []struct{ clave, valor string }{
		{"registrada_en", r.RegistradaEn}, {"consumida_en", r.ConsumidaEn},
	} {
		if !instanteConsumoAD173Valido(campo.valor) {
			return fallar("instante_invalido", campo.clave)
		}
	}
	if r.RegistradaEn != r.ConsumidaEn {
		return fallar("instante_distinto", "registrada_consumida_en")
	}
	for _, campo := range []struct{ clave, valor string }{
		{"actor_ref", r.ActorRef}, {"perfil_activo_ref", r.PerfilActivoRef}, {"finalidad_ref", r.FinalidadRef},
	} {
		if !referenciaCadenaValida(campo.valor) {
			return fallar("registro_invalido", campo.clave)
		}
	}
	return nil
}

func instanteConsumoAD173Valido(valor string) bool {
	if len(valor) != len("2006-01-02T15:04:05.000000Z") {
		return false
	}
	instante, err := time.Parse(time.RFC3339Nano, valor)
	return err == nil && instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.UTC().Format("2006-01-02T15:04:05.000000Z") == valor
}

func huellaConsumoFechaV3(r RegistroConsumoFechaV3) string {
	h := sha256.New()
	for _, valor := range []string{TipoConsumoFechaV3, "3", strconv.FormatUint(r.Secuencia, 10),
		r.AnteriorSHA256, r.DecisionRef, r.EfectoRef, r.HuellaEfectoSHA256,
		r.ConsumoHuellaSHA256, r.Proceso, r.Canal, r.RegistradaEn, r.ConsumidaEn,
		r.ActorRef, r.PerfilActivoRef, r.FinalidadRef} {
		// El encuadre SQL cuenta octetos UTF-8, no caracteres.
		_, _ = h.Write(encuadrarVerificacion(valor))
	}
	return hex.EncodeToString(h.Sum(nil))
}
