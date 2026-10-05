package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// EsquemaVerificacion identifies the offline projection of the existing AD3
// chain. It does not introduce a store or an append authority.
const EsquemaVerificacion = "vec.auditoria.verificacion.v1"

const maxSecuenciaVerificacion uint64 = 9007199254740991

// RegistroCadenaV3 contains only the coordinates needed by AD3-002's hash.
// It deliberately excludes canonical decisions, actor contexts and keys.
type RegistroCadenaV3 struct {
	AuditoriaRef        string `json:"auditoria_ref"`
	Secuencia           uint64 `json:"secuencia"`
	DecisionRef         string `json:"decision_ref"`
	EfectoRef           string `json:"efecto_ref"`
	HuellaEfectoSHA256  string `json:"huella_efecto_sha256"`
	AnteriorSHA256      string `json:"anterior_sha256"`
	HuellaSHA256        string `json:"huella_sha256"`
	ConsumoHuellaSHA256 string `json:"consumo_huella_sha256"`
}

// CoberturaCadena must describe one contiguous range in one database chain.
// Different databases have separate chains and cannot share a checkpoint.
type CoberturaCadena struct {
	CadenaID         string `json:"cadena_id"`
	PrimeraSecuencia uint64 `json:"primera_secuencia"`
	UltimaSecuencia  uint64 `json:"ultima_secuencia"`
	AnteriorSHA256   string `json:"anterior_sha256"`
	CabezaSHA256     string `json:"cabeza_sha256"`
	Registros        uint64 `json:"registros"`
}

type DocumentoVerificacion struct {
	Esquema    string             `json:"esquema"`
	Manifiesto CoberturaCadena    `json:"manifiesto"`
	Registros  []RegistroCadenaV3 `json:"registros"`
}

// FalloVerificacion exposes only structural keys and safe comparison values.
// It never returns a supplied reference or raw input as an error.
type FalloVerificacion struct {
	Codigo    string `json:"codigo"`
	Secuencia uint64 `json:"secuencia,omitempty"`
	Clave     string `json:"clave"`
	Esperado  string `json:"esperado"`
	Obtenido  string `json:"obtenido"`
}

type InformeVerificacion struct {
	Esquema                          string             `json:"esquema"`
	Estado                           string             `json:"estado"`
	Cobertura                        CoberturaCadena    `json:"cobertura"`
	CheckpointCotejado               bool               `json:"checkpoint_cotejado"`
	AutenticidadCheckpoint           string             `json:"autenticidad_checkpoint"`
	ContenidoConsumoRecalculado      bool               `json:"contenido_consumo_recalculado"`
	CamposFueraHuellaVerificados     bool               `json:"campos_fuera_huella_verificados"`
	MaterialIntentoRecalculado       bool               `json:"material_intento_recalculado,omitempty"`
	ActorPerfilContextoCotejados     bool               `json:"actor_perfil_contexto_cotejados,omitempty"`
	AutenticidadFuentesHistoricas    string             `json:"autenticidad_fuentes_historicas,omitempty"`
	ConsumosHistoricosSinFechaLigada bool               `json:"consumos_historicos_sin_fecha_ligada"`
	FechaConsumoLigadaCotejada       bool               `json:"fecha_consumo_ligada_cotejada"`
	Fallo                            *FalloVerificacion `json:"fallo,omitempty"`
}

// VerificarCadenaV3 reconstructs exactly the hash in AD3-002. The checkpoint
// is supplied through a separate trusted channel by the operator; this method
// does not authenticate its provenance. It cannot verify absent future rows.
func VerificarCadenaV3(d DocumentoVerificacion, checkpoint CoberturaCadena, maxRegistros uint64) InformeVerificacion {
	informe := InformeVerificacion{Esquema: EsquemaVerificacion, Estado: "rechazada", AutenticidadCheckpoint: "no_comprobada"}
	fallar := func(codigo, clave, esperado, obtenido string, secuencia uint64) InformeVerificacion {
		informe.Fallo = &FalloVerificacion{Codigo: codigo, Clave: clave, Esperado: esperado, Obtenido: obtenido, Secuencia: secuencia}
		return informe
	}
	if d.Esquema != EsquemaVerificacion {
		return fallar("esquema_invalido", "esquema", EsquemaVerificacion, "no_admitido", 0)
	}
	if maxRegistros == 0 || !coberturaValida(checkpoint) || !coberturaValida(d.Manifiesto) {
		return fallar("cobertura_invalida", "manifiesto_checkpoint_limite", "rango_valido_y_limite_positivo", "invalido", 0)
	}
	if d.Manifiesto != checkpoint {
		return fallar("checkpoint_distinto", "manifiesto", "checkpoint_separado", "distinto", 0)
	}
	informe.Cobertura = checkpoint
	informe.CheckpointCotejado = true
	if uint64(len(d.Registros)) > maxRegistros {
		return fallar("limite_registros", "max_registros", strconv.FormatUint(maxRegistros, 10), strconv.Itoa(len(d.Registros)), 0)
	}
	if uint64(len(d.Registros)) != checkpoint.Registros {
		return fallar("cantidad_distinta", "registros", strconv.FormatUint(checkpoint.Registros, 10), strconv.Itoa(len(d.Registros)), 0)
	}
	anterior := checkpoint.AnteriorSHA256
	decisiones := make(map[string]struct{}, len(d.Registros))
	consumos := make(map[string]struct{}, len(d.Registros))
	for i, r := range d.Registros {
		secuencia := checkpoint.PrimeraSecuencia + uint64(i)
		if r.Secuencia != secuencia {
			return fallar("secuencia_distinta", "secuencia", strconv.FormatUint(secuencia, 10), strconv.FormatUint(r.Secuencia, 10), secuencia)
		}
		if !referenciaCadenaValida(r.DecisionRef) || !referenciaCadenaValida(r.EfectoRef) ||
			!huellaCadenaValida(r.HuellaEfectoSHA256) || !huellaCadenaValida(r.AnteriorSHA256) ||
			!huellaCadenaValida(r.HuellaSHA256) || !huellaCadenaValida(r.ConsumoHuellaSHA256) {
			return fallar("registro_invalido", "coordenadas", "referencias_opacas_y_sha256", "invalido", secuencia)
		}
		if r.AuditoriaRef != "aud_v3_"+r.ConsumoHuellaSHA256[:32] {
			return fallar("referencia_distinta", "auditoria_ref", "derivada_del_consumo", "distinta", secuencia)
		}
		if _, existe := decisiones[r.DecisionRef]; existe {
			return fallar("decision_duplicada", "decision_ref", "unica", "duplicada", secuencia)
		}
		if _, existe := consumos[r.ConsumoHuellaSHA256]; existe {
			return fallar("consumo_duplicado", "consumo_huella_sha256", "unica", "duplicada", secuencia)
		}
		decisiones[r.DecisionRef], consumos[r.ConsumoHuellaSHA256] = struct{}{}, struct{}{}
		if r.AnteriorSHA256 != anterior {
			return fallar("enlace_distinto", "anterior_sha256", anterior, r.AnteriorSHA256, secuencia)
		}
		huella := huellaRegistroCadena(r)
		if r.HuellaSHA256 != huella {
			return fallar("huella_distinta", "huella_sha256", huella, r.HuellaSHA256, secuencia)
		}
		anterior = huella
	}
	if anterior != checkpoint.CabezaSHA256 {
		return fallar("cabeza_distinta", "cabeza_sha256", checkpoint.CabezaSHA256, anterior, checkpoint.UltimaSecuencia)
	}
	informe.Estado = "verificada"
	informe.ConsumosHistoricosSinFechaLigada = len(d.Registros) > 0
	return informe
}

func coberturaValida(c CoberturaCadena) bool {
	if !referenciaCadenaValida(c.CadenaID) || !huellaCadenaValida(c.AnteriorSHA256) ||
		!huellaCadenaValida(c.CabezaSHA256) || c.UltimaSecuencia > maxSecuenciaVerificacion {
		return false
	}
	if c.Registros == 0 {
		return c.PrimeraSecuencia == 0 && c.UltimaSecuencia == 0 &&
			c.AnteriorSHA256 == strings.Repeat("0", 64) && c.CabezaSHA256 == c.AnteriorSHA256
	}
	return c.PrimeraSecuencia >= 1 && c.UltimaSecuencia >= c.PrimeraSecuencia &&
		c.Registros == c.UltimaSecuencia-c.PrimeraSecuencia+1 &&
		(c.PrimeraSecuencia != 1 || c.AnteriorSHA256 == strings.Repeat("0", 64))
}

func referenciaCadenaValida(v string) bool {
	if len(v) == 0 || len(v) > 512 || !utf8.ValidString(v) || strings.ContainsAny(v, "*?") {
		return false
	}
	return !strings.ContainsFunc(v, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) })
}

func huellaCadenaValida(v string) bool {
	if len(v) != sha256.Size*2 {
		return false
	}
	for _, c := range v {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func huellaRegistroCadena(r RegistroCadenaV3) string {
	h := sha256.New()
	for _, v := range []string{strconv.FormatUint(r.Secuencia, 10), r.AnteriorSHA256,
		r.DecisionRef, r.EfectoRef, r.HuellaEfectoSHA256, r.ConsumoHuellaSHA256} {
		// encuadrar_mac uses UTF-8 octet_length, not the number of runes.
		_, _ = h.Write([]byte(strconv.Itoa(len(v)) + ":" + v + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}
