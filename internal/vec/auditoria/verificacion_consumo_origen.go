package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

const TipoConsumoOrigenV2 = "consumo_confirmado_v2"

// RegistroConsumoOrigenV2 proyecta el eslabón AD172. El proceso y el canal
// pertenecen al eslabón; no forman parte del material V3 firmado original.
// El cotejo de esta proyección no autentica el checkpoint ni la configuración DBA.
type RegistroConsumoOrigenV2 struct {
	RegistroCadenaV3
	TipoRegistro   string `json:"tipo_registro"`
	VersionConsumo uint8  `json:"version_consumo"`
	Proceso        string `json:"proceso"`
	Canal          string `json:"canal"`
}

// CotejarConsumoOrigenV2 comprueba la forma y la huella de un asiento AD172.
// La cadena completa debe verificar además orden, unicidad y checkpoint.
func CotejarConsumoOrigenV2(r RegistroConsumoOrigenV2) *FalloVerificacion {
	fallar := func(codigo, clave string) *FalloVerificacion {
		return &FalloVerificacion{Codigo: codigo, Clave: clave,
			Esperado: "asiento_ad172_valido", Obtenido: "incompatible", Secuencia: r.Secuencia}
	}
	if r.TipoRegistro != TipoConsumoOrigenV2 || r.VersionConsumo != 2 {
		return fallar("tipo_invalido", "tipo_version_consumo")
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
	h := sha256.New()
	for _, valor := range []string{TipoConsumoOrigenV2, "2", strconv.FormatUint(r.Secuencia, 10),
		r.AnteriorSHA256, r.DecisionRef, r.EfectoRef, r.HuellaEfectoSHA256,
		r.ConsumoHuellaSHA256, r.Proceso, r.Canal} {
		_, _ = h.Write([]byte(strconv.Itoa(len(valor)) + ":" + valor + "\n"))
	}
	if r.HuellaSHA256 != hex.EncodeToString(h.Sum(nil)) {
		return fallar("huella_distinta", "huella_sha256")
	}
	return nil
}
