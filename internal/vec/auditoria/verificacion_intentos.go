package auditoria

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

const EsquemaVerificacionMixta = "vec.auditoria.verificacion.v2"

// Formatos cerrados de AD169 y de su configuración de registradores.
var (
	codigoOrdenAD169  = regexp.MustCompile(`\A[a-z][a-z0-9._:-]{0,159}\z`)
	recursoOrdenAD169 = regexp.MustCompile(`\A[a-z0-9][a-z0-9._:-]{0,199}\z`)
	correlacionAD169  = regexp.MustCompile(`\A[a-zA-Z0-9][a-zA-Z0-9._:-]{0,127}\z`)
	procesoOrdenAD169 = regexp.MustCompile(`\A[a-z][a-z0-9._-]{1,79}\z`)
)

// RegistroIntentoV2 es una proyección autorizada de AD169. El contexto V2
// canónico permite cotejar actor y perfil; no acredita por sí solo su origen.
type RegistroIntentoV2 struct {
	AuditoriaRef           string `json:"auditoria_ref"`
	Secuencia              uint64 `json:"secuencia"`
	AnteriorSHA256         string `json:"anterior_sha256"`
	HuellaSHA256           string `json:"huella_sha256"`
	RegistradaEn           string `json:"registrada_en"`
	IntentoRef             string `json:"intento_ref"`
	IntentoMaterialSHA256  string `json:"intento_material_sha256"`
	ActorRef               string `json:"actor_ref"`
	PerfilActivoRef        string `json:"perfil_activo_ref"`
	RegistroContextoRef    string `json:"registro_contexto_ref"`
	ContextoSHA256         string `json:"contexto_sha256"`
	ProcedenciaSHA256      string `json:"procedencia_sha256"`
	AutenticacionRef       string `json:"autenticacion_ref"`
	SesionRef              string `json:"sesion_ref"`
	AutenticacionSHA256    string `json:"autenticacion_sha256"`
	Accion                 string `json:"accion"`
	ModuloID               string `json:"modulo_id"`
	RecursoRef             string `json:"recurso_ref"`
	FinalidadRef           string `json:"finalidad_ref"`
	Resultado              string `json:"resultado"`
	MotivoRef              string `json:"motivo_ref"`
	Proceso                string `json:"proceso"`
	Canal                  string `json:"canal"`
	CorrelacionRef         string `json:"correlacion_ref"`
	VinculoSHA256          string `json:"vinculo_sha256"`
	ContextoCanonicoBase64 string `json:"contexto_canonico_base64"`
}

type RegistroMixtoV2 struct {
	UnidadInicial           *RegistroUnidadInicialPersonalV1        `json:"unidad_inicial,omitempty"`
	IntentoUnidadInicial    *RegistroIntentoUnidadInicialPersonalV1 `json:"intento_unidad_inicial,omitempty"`
	TipoRegistro            string                                  `json:"tipo_registro"`
	Consumo                 *RegistroCadenaV3                       `json:"consumo,omitempty"`
	Intento                 *RegistroIntentoV2                      `json:"intento,omitempty"`
	Preperfil               *RegistroPreperfilV3                    `json:"preperfil,omitempty"`
	Bootstrap               *RegistroBootstrapV3                    `json:"bootstrap,omitempty"`
	FuentesIniciales        *RegistroFuentesInicialesV1             `json:"fuentes_iniciales,omitempty"`
	IntentoFuentesIniciales *RegistroIntentoFuentesInicialesV1      `json:"intento_fuentes_iniciales,omitempty"`
}

type DocumentoVerificacionMixta struct {
	Esquema    string            `json:"esquema"`
	Manifiesto CoberturaCadena   `json:"manifiesto"`
	Registros  []RegistroMixtoV2 `json:"registros"`
}

// VerificarCadenaMixtaV2 coteja el encadenado AD3-002/AD169 y las columnas
// que AD169 compromete. El checkpoint separado necesita un origen confiable.
func VerificarCadenaMixtaV2(d DocumentoVerificacionMixta, checkpoint CoberturaCadena, maxRegistros uint64) InformeVerificacion {
	return verificarCadenaMixta(d, checkpoint, maxRegistros, EsquemaVerificacionMixta)
}

func verificarCadenaMixta(d DocumentoVerificacionMixta, checkpoint CoberturaCadena, maxRegistros uint64, esquema string) InformeVerificacion {
	informe := InformeVerificacion{
		Esquema: esquema, Estado: "rechazada",
		AutenticidadCheckpoint: "no_comprobada", AutenticidadFuentesHistoricas: "no_comprobada",
	}
	fallar := func(codigo, clave, esperado, obtenido string, secuencia uint64) InformeVerificacion {
		informe.Fallo = &FalloVerificacion{Codigo: codigo, Clave: clave, Esperado: esperado, Obtenido: obtenido, Secuencia: secuencia}
		return informe
	}
	if d.Esquema != esquema {
		return fallar("esquema_invalido", "esquema", esquema, "no_admitido", 0)
	}
	if maxRegistros == 0 || !coberturaValida(checkpoint) || !coberturaValida(d.Manifiesto) {
		return fallar("cobertura_invalida", "manifiesto_checkpoint_limite", "rango_valido_y_limite_positivo", "invalido", 0)
	}
	if d.Manifiesto != checkpoint {
		return fallar("checkpoint_distinto", "manifiesto", "checkpoint_separado", "distinto", 0)
	}
	informe.Cobertura, informe.CheckpointCotejado = checkpoint, true
	if uint64(len(d.Registros)) > maxRegistros {
		return fallar("limite_registros", "max_registros", strconv.FormatUint(maxRegistros, 10), strconv.Itoa(len(d.Registros)), 0)
	}
	if uint64(len(d.Registros)) != checkpoint.Registros {
		return fallar("cantidad_distinta", "registros", strconv.FormatUint(checkpoint.Registros, 10), strconv.Itoa(len(d.Registros)), 0)
	}
	anterior := checkpoint.AnteriorSHA256
	auditorias, decisiones, consumos, intentos := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	eventos := map[string]bool{}
	for i, r := range d.Registros {
		secuencia := checkpoint.PrimeraSecuencia + uint64(i)
		var referencia, previo, huella string
		switch r.TipoRegistro {
		case "consumo_confirmado":
			if r.Consumo == nil || r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil || (r.FuentesIniciales != nil || r.IntentoFuentesIniciales != nil || r.UnidadInicial != nil || r.IntentoUnidadInicial != nil) {
				return fallar("tipo_invalido", "tipo_registro", "consumo_exclusivo", "invalido", secuencia)
			}
			c := *r.Consumo
			if c.Secuencia != secuencia {
				return fallar("secuencia_distinta", "secuencia", strconv.FormatUint(secuencia, 10), strconv.FormatUint(c.Secuencia, 10), secuencia)
			}
			if !referenciaCadenaValida(c.DecisionRef) || !referenciaCadenaValida(c.EfectoRef) ||
				!huellaCadenaValida(c.HuellaEfectoSHA256) || !huellaCadenaValida(c.ConsumoHuellaSHA256) ||
				!huellaCadenaValida(c.AnteriorSHA256) || !huellaCadenaValida(c.HuellaSHA256) {
				return fallar("registro_invalido", "coordenadas", "referencias_opacas_y_sha256", "invalido", secuencia)
			}
			if c.AuditoriaRef != "aud_v3_"+c.ConsumoHuellaSHA256[:32] {
				return fallar("referencia_distinta", "auditoria_ref", "derivada_del_consumo", "distinta", secuencia)
			}
			if c.AnteriorSHA256 != anterior {
				return fallar("enlace_distinto", "anterior_sha256", anterior, c.AnteriorSHA256, secuencia)
			}
			if decisiones[c.DecisionRef] || consumos[c.ConsumoHuellaSHA256] {
				return fallar("consumo_duplicado", "decision_o_consumo", "unico", "duplicado", secuencia)
			}
			decisiones[c.DecisionRef], consumos[c.ConsumoHuellaSHA256] = true, true
			referencia, previo, huella = c.AuditoriaRef, c.AnteriorSHA256, huellaRegistroCadena(c)
			if c.HuellaSHA256 != huella {
				return fallar("huella_distinta", "huella_sha256", huella, c.HuellaSHA256, secuencia)
			}
		case "intento_nominal":
			if r.Intento == nil || r.Consumo != nil || r.Preperfil != nil || r.Bootstrap != nil || (r.FuentesIniciales != nil || r.IntentoFuentesIniciales != nil || r.UnidadInicial != nil || r.IntentoUnidadInicial != nil) {
				return fallar("tipo_invalido", "tipo_registro", "intento_exclusivo", "invalido", secuencia)
			}
			a := *r.Intento
			if a.Secuencia != secuencia {
				return fallar("secuencia_distinta", "secuencia", strconv.FormatUint(secuencia, 10), strconv.FormatUint(a.Secuencia, 10), secuencia)
			}
			if !huellaCadenaValida(a.AnteriorSHA256) {
				return fallar("registro_invalido", "anterior_sha256", "sha256", "no_admitido", secuencia)
			}
			if a.AnteriorSHA256 != anterior {
				return fallar("enlace_distinto", "anterior_sha256", anterior, "distinto", secuencia)
			}
			if intentos[a.IntentoRef] {
				return fallar("intento_duplicado", "intento_ref", "unico", "duplicado", secuencia)
			}
			intentos[a.IntentoRef] = true
			codigo, clave, esperado, obtenido := cotejarIntentoV2(a)
			if codigo != "" {
				return fallar(codigo, clave, esperado, obtenido, secuencia)
			}
			referencia, previo, huella = a.AuditoriaRef, a.AnteriorSHA256, a.HuellaSHA256
		case "preperfil_autenticado", "bootstrap_operador":
			if (esquema != EsquemaVerificacionPreperfil && esquema != EsquemaVerificacionFuentesIniciales && esquema != EsquemaVerificacionUnidadInicial) || (r.FuentesIniciales != nil || r.IntentoFuentesIniciales != nil || r.UnidadInicial != nil || r.IntentoUnidadInicial != nil) {
				return fallar("tipo_invalido", "tipo_registro", "tipo_admitido", "invalido", secuencia)
			}
			evento, codigo, clave := cotejarRegistroAdminV3(r, secuencia)
			if codigo != "" {
				return fallar(codigo, clave, "registro_ad171_valido", "no_admitido", secuencia)
			}
			if eventos[evento.EventoRef] {
				return fallar("evento_duplicado", "evento_ref", "unico", "duplicado", secuencia)
			}
			eventos[evento.EventoRef] = true
			referencia, previo, huella = evento.AuditoriaRef, evento.AnteriorSHA256, evento.HuellaSHA256
		case "provision_fuentes_iniciales_admin", "intento_fuentes_iniciales_admin":
			if (esquema != EsquemaVerificacionFuentesIniciales && esquema != EsquemaVerificacionUnidadInicial) || r.UnidadInicial != nil || r.IntentoUnidadInicial != nil {
				return fallar("tipo_invalido", "tipo_registro", "tipo_admitido", "invalido", secuencia)
			}
			evento, codigo, clave := cotejarRegistroFuentesInicialesV1(r, secuencia)
			if r.TipoRegistro == "intento_fuentes_iniciales_admin" {
				evento, codigo, clave = cotejarIntentoFuentesInicialesV1(r, secuencia)
			}
			if codigo != "" {
				return fallar(codigo, clave, "registro_ad174_valido", "no_admitido", secuencia)
			}
			if eventos[evento.EventoRef] {
				return fallar("evento_duplicado", "evento_ref", "unico", "duplicado", secuencia)
			}
			eventos[evento.EventoRef] = true
			referencia, previo, huella = evento.AuditoriaRef, evento.AnteriorSHA256, evento.HuellaSHA256
		case "unidad_inicial_personal", "intento_unidad_inicial_personal":
			if esquema != EsquemaVerificacionUnidadInicial {
				return fallar("tipo_invalido", "tipo_registro", "tipo_admitido", "invalido", secuencia)
			}
			evento, codigo, clave := cotejarRegistroUnidadInicialV1(r, secuencia)
			if r.TipoRegistro == "intento_unidad_inicial_personal" {
				evento, codigo, clave = cotejarIntentoUnidadInicialV1(r, secuencia)
			}
			if codigo != "" {
				return fallar(codigo, clave, "registro_ad176_valido", "no_admitido", secuencia)
			}
			if eventos[evento.EventoRef] {
				return fallar("evento_duplicado", "evento_ref", "unico", "duplicado", secuencia)
			}
			eventos[evento.EventoRef] = true
			referencia, previo, huella = evento.AuditoriaRef, evento.AnteriorSHA256, evento.HuellaSHA256
		default:
			return fallar("tipo_invalido", "tipo_registro", "tipo_admitido", "invalido", secuencia)
		}
		if auditorias[referencia] {
			return fallar("auditoria_duplicada", "auditoria_ref", "unica", "duplicada", secuencia)
		}
		auditorias[referencia] = true
		if previo != anterior {
			return fallar("enlace_distinto", "anterior_sha256", anterior, previo, secuencia)
		}
		anterior = huella
	}
	if anterior != checkpoint.CabezaSHA256 {
		return fallar("cabeza_distinta", "cabeza_sha256", checkpoint.CabezaSHA256, anterior, checkpoint.UltimaSecuencia)
	}
	informe.Estado = "verificada"
	informe.MaterialIntentoRecalculado = len(intentos) > 0
	informe.ActorPerfilContextoCotejados = len(intentos) > 0
	return informe
}

func cotejarIntentoV2(a RegistroIntentoV2) (codigo, clave, esperado, obtenido string) {
	fallar := func(c, k, e string) (string, string, string, string) { return c, k, e, "invalido" }
	if len(a.IntentoRef) != 40 || !strings.HasPrefix(a.IntentoRef, "intento_") ||
		!hex32Valido(a.IntentoRef[8:]) || a.AuditoriaRef != "aud_v3_i_"+a.IntentoRef[8:] {
		return fallar("referencia_distinta", "auditoria_ref", "derivada_del_intento")
	}
	if !huellaCadenaValida(a.AnteriorSHA256) || !huellaCadenaValida(a.HuellaSHA256) ||
		!huellaCadenaValida(a.IntentoMaterialSHA256) || !huellaCadenaValida(a.ContextoSHA256) ||
		!huellaCadenaValida(a.ProcedenciaSHA256) || !huellaCadenaValida(a.AutenticacionSHA256) ||
		!huellaCadenaValida(a.VinculoSHA256) || a.Resultado != "denegado" && a.Resultado != "error" {
		return fallar("registro_invalido", "coordenadas", "sha256_y_resultado_admitido")
	}
	for _, v := range ordenIntentoV2(a) {
		if !referenciaCadenaValida(v) || len(v) > 200 {
			return fallar("registro_invalido", "material", "texto_opaco_acotado")
		}
	}
	if len(a.RegistroContextoRef) > 128 || len(a.AutenticacionRef) > 128 ||
		len(a.SesionRef) > 128 || !codigoOrdenAD169.MatchString(a.Accion) ||
		!codigoOrdenAD169.MatchString(a.ModuloID) || !recursoOrdenAD169.MatchString(a.RecursoRef) ||
		!codigoOrdenAD169.MatchString(a.FinalidadRef) || !codigoOrdenAD169.MatchString(a.MotivoRef) ||
		!correlacionAD169.MatchString(a.CorrelacionRef) || !procesoOrdenAD169.MatchString(a.Proceso) ||
		(a.Canal != "administracion_privilegiada" && a.Canal != "interna_corporativa" &&
			a.Canal != "externa_personal") {
		return fallar("registro_invalido", "orden", "formato_ad169")
	}
	if !referenciaCadenaValida(a.ActorRef) || !referenciaCadenaValida(a.PerfilActivoRef) {
		return fallar("registro_invalido", "actor_perfil", "referencias_opacas")
	}
	if len(a.ContextoCanonicoBase64) > base64.StdEncoding.EncodedLen(domain.TamanoMaximoRepresentacionContextoActorV2) {
		return fallar("limite_contexto", "contexto_canonico_base64", "maximo_v2")
	}
	contexto, err := base64.StdEncoding.Strict().DecodeString(a.ContextoCanonicoBase64)
	if err != nil || len(contexto) == 0 || base64.StdEncoding.EncodeToString(contexto) != a.ContextoCanonicoBase64 {
		return fallar("contexto_invalido", "contexto_canonico_base64", "base64_canonico")
	}
	suma := sha256.Sum256(contexto)
	if hex.EncodeToString(suma[:]) != a.ContextoSHA256 {
		return fallar("contexto_distinto", "contexto_sha256", "huella_preimagen")
	}
	actor, err := domain.RehidratarContextoActorVinculadoV2(contexto)
	if err != nil || actor.Principal.ID != a.ActorRef || actor.PerfilActivoRef != a.PerfilActivoRef {
		return fallar("actor_perfil_distinto", "actor_perfil", "contexto_v2")
	}
	instante, err := time.Parse(time.RFC3339Nano, a.RegistradaEn)
	if err != nil || instante.Year() < 1 || instante.Year() > 9999 ||
		instante.UTC().Format("2006-01-02T15:04:05.000000Z") != a.RegistradaEn {
		return fallar("instante_invalido", "registrada_en", "utc_microsegundos")
	}
	material := huellaEncuadradaIntento("vec.auditoria.intento.v1", a.ContextoSHA256, a.VinculoSHA256)
	for _, v := range ordenIntentoV2(a) {
		material = append(material, encuadrarVerificacion(v)...)
	}
	materialSuma := sha256.Sum256(material)
	materialSHA := hex.EncodeToString(materialSuma[:])
	if materialSHA != a.IntentoMaterialSHA256 {
		return "material_distinto", "intento_material_sha256", materialSHA, a.IntentoMaterialSHA256
	}
	eslabon := huellaEncuadradaIntento("vec.auditoria.eslabon.intento.v1", strconv.FormatUint(a.Secuencia, 10),
		a.AnteriorSHA256, a.AuditoriaRef, materialSHA, a.RegistradaEn)
	eslabonSuma := sha256.Sum256(eslabon)
	eslabonSHA := hex.EncodeToString(eslabonSuma[:])
	if eslabonSHA != a.HuellaSHA256 {
		return "huella_distinta", "huella_sha256", eslabonSHA, a.HuellaSHA256
	}
	return "", "", "", ""
}

func ordenIntentoV2(a RegistroIntentoV2) []string {
	return []string{a.IntentoRef, a.RegistroContextoRef, a.ContextoSHA256,
		a.ProcedenciaSHA256, a.AutenticacionRef, a.SesionRef, a.AutenticacionSHA256,
		a.Accion, a.ModuloID, a.RecursoRef, a.FinalidadRef, a.Resultado,
		a.MotivoRef, a.Proceso, a.Canal, a.CorrelacionRef}
}

func huellaEncuadradaIntento(v ...string) []byte {
	var b []byte
	for _, x := range v {
		b = append(b, encuadrarVerificacion(x)...)
	}
	return b
}

func encuadrarVerificacion(v string) []byte {
	return []byte(strconv.Itoa(len(v)) + ":" + v + "\n")
}

func hex32Valido(v string) bool {
	if len(v) != 32 {
		return false
	}
	for _, c := range v {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
