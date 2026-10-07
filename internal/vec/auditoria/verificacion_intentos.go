package auditoria

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	PerfilesAsignables        *RegistroPerfilesAsignablesV1           `json:"perfiles_asignables,omitempty"`
	IntentoPerfilesAsignables *RegistroIntentoPerfilesAsignablesV1    `json:"intento_perfiles_asignables,omitempty"`
	ContextoAdminPreV2        *RegistroContextoAdminPreV2             `json:"contexto_admin_pre_v2,omitempty"`
	FronteraAdminTecnica      *RegistroFronteraAdminTecnicaV1         `json:"frontera_admin_tecnica,omitempty"`
	GobiernoUsuarios          *RegistroGobiernoUsuariosV1             `json:"gobierno_usuarios,omitempty"`
	IntentoGobiernoUsuarios   *RegistroIntentoGobiernoUsuariosV1      `json:"intento_gobierno_usuarios,omitempty"`
	Preservacion              *RegistroPreservacionAuditoriaV1        `json:"preservacion,omitempty"`
	Periodica                 *RegistroOperacionPeriodicaV1           `json:"periodica,omitempty"`
	MantenimientoFijo         *RegistroMantenimientoFijoV1            `json:"mantenimiento_fijo,omitempty"`
	IntentoMantenimientoFijo  *RegistroIntentoMantenimientoFijoV1     `json:"intento_mantenimiento_fijo,omitempty"`
	IntentoBootstrapCentral   *RegistroIntentoBootstrapCentralV1      `json:"intento_bootstrap_central,omitempty"`
	UnidadInicial             *RegistroUnidadInicialPersonalV1        `json:"unidad_inicial,omitempty"`
	IntentoUnidadInicial      *RegistroIntentoUnidadInicialPersonalV1 `json:"intento_unidad_inicial,omitempty"`
	FuentesIniciales          *RegistroFuentesInicialesV1             `json:"fuentes_iniciales,omitempty"`
	IntentoFuentesIniciales   *RegistroIntentoFuentesInicialesV1      `json:"intento_fuentes_iniciales,omitempty"`
	TipoRegistro              string                                  `json:"tipo_registro"`
	Consumo                   *RegistroCadenaV3                       `json:"-"`
	ConsumoOrigen             *RegistroConsumoOrigenV2                `json:"-"`
	ConsumoTransaccion        *RegistroConsumoTransaccionV4           `json:"-"`
	ConsumoFecha              *RegistroConsumoFechaV3                 `json:"-"`
	Intento                   *RegistroIntentoV2                      `json:"intento,omitempty"`
	Preperfil                 *RegistroPreperfilV3                    `json:"preperfil,omitempty"`
	Bootstrap                 *RegistroBootstrapV3                    `json:"bootstrap,omitempty"`
	// Eslabon acompaña a los asientos posteriores al corte de AD207.
	Eslabon *EslabonCadenaV5 `json:"eslabon,omitempty"`
}

var errRegistroMixtoJSON = errors.New("vec auditoria: registro mixto invalido")

// Las versiones conservan el mismo objeto JSON consumo. La proyección
// histórica mantiene su tipo Go; el discriminador elige una sola versión.
func (r RegistroMixtoV2) MarshalJSON() ([]byte, error) {
	type alias RegistroMixtoV2
	var consumo any
	for _, presente := range []bool{r.Consumo != nil, r.ConsumoOrigen != nil, r.ConsumoFecha != nil, r.ConsumoTransaccion != nil} {
		if presente && consumo != nil {
			return nil, errRegistroMixtoJSON
		}
		if presente {
			switch {
			case r.Consumo != nil:
				consumo = r.Consumo
			case r.ConsumoOrigen != nil:
				consumo = r.ConsumoOrigen
			case r.ConsumoFecha != nil:
				consumo = r.ConsumoFecha
			case r.ConsumoTransaccion != nil:
				consumo = r.ConsumoTransaccion
			}
		}
	}
	return json.Marshal(struct {
		alias
		Consumo any `json:"consumo,omitempty"`
	}{alias(r), consumo})
}

func (r *RegistroMixtoV2) UnmarshalJSON(b []byte) error {
	type alias RegistroMixtoV2
	var registro RegistroMixtoV2
	objeto := struct {
		*alias
		Consumo json.RawMessage `json:"consumo"`
	}{alias: (*alias)(&registro)}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&objeto) != nil {
		return errRegistroMixtoJSON
	}
	if len(objeto.Consumo) > 0 {
		var destino any
		switch registro.TipoRegistro {
		case "consumo_confirmado":
			destino = &registro.Consumo
		case TipoConsumoOrigenV2:
			destino = &registro.ConsumoOrigen
		case TipoConsumoFechaV3:
			destino = &registro.ConsumoFecha
		case TipoConsumoTransaccionV4:
			destino = &registro.ConsumoTransaccion
		default:
			return errRegistroMixtoJSON
		}
		decoder = json.NewDecoder(bytes.NewReader(objeto.Consumo))
		decoder.DisallowUnknownFields()
		if decoder.Decode(destino) != nil {
			return errRegistroMixtoJSON
		}
	}
	*r = registro
	return nil
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
	numerosV5 := map[uint64]bool{}
	// enlaces[posición] = huella (antes del corte) o eslabón (después): sirve
	// para cotejar la cabeza sellada que declaran las capturas periódicas.
	enlaces := map[uint64]string{}
	if checkpoint.PrimeraSecuencia > 0 {
		enlaces[checkpoint.PrimeraSecuencia-1] = checkpoint.AnteriorSHA256
	}
	type previaDeclarada struct {
		posicion, previa uint64
		cabeza           string
	}
	var previas []previaDeclarada
	var ultimaAntesDelCorte uint64
	var historicosSinFecha, fechaLigada, trasCorte bool
	for i, r := range d.Registros {
		// La cobertura cuenta posiciones en la cadena. Antes del corte de AD207
		// coinciden con el número del asiento; después, el eslabón da el número.
		posicion := checkpoint.PrimeraSecuencia + uint64(i)
		secuencia, enlace := posicion, anterior
		if r.Eslabon != nil {
			// Un número posterior al corte no puede repetirse ni quedar por
			// debajo de un asiento anterior al corte del mismo documento.
			if r.Eslabon.Posicion != posicion || r.Eslabon.Secuencia == 0 || r.Eslabon.Secuencia > maxSecuenciaVerificacion ||
				r.Eslabon.Secuencia <= ultimaAntesDelCorte || numerosV5[r.Eslabon.Secuencia] ||
				!huellaCadenaValida(r.Eslabon.AnteriorSHA256) || !huellaCadenaValida(r.Eslabon.EslabonSHA256) ||
				!instanteEslabonValido(r.Eslabon.RegistradaEn) || !instanteEslabonValido(r.Eslabon.SelladoEn) {
				return fallar("eslabon_invalido", "eslabon", "posicion_y_numero_unicos", "invalido", posicion)
			}
			numerosV5[r.Eslabon.Secuencia] = true
			secuencia, enlace = r.Eslabon.Secuencia, MarcadorSinAnteriorV5
		} else if trasCorte {
			return fallar("eslabon_ausente", "eslabon", "presente_tras_el_corte", "ausente", posicion)
		}
		var referencia, previo, huella string
		if r.ContextoAdminPreV2 != nil && r.TipoRegistro != "contexto_admin_pre_v2" {
			return fallar("tipo_invalido", "tipo_registro", "familia_exclusiva", "invalido", secuencia)
		}
		if r.ConsumoTransaccion != nil && r.TipoRegistro != TipoConsumoTransaccionV4 {
			return fallar("tipo_invalido", "tipo_registro", "consumo_exclusivo", "invalido", secuencia)
		}
		if (r.PerfilesAsignables != nil || r.IntentoPerfilesAsignables != nil) && r.TipoRegistro != "perfiles_asignables_admin" && r.TipoRegistro != "intento_perfiles_asignables_admin" {
			return fallar("tipo_invalido", "tipo_registro", "familia_exclusiva", "invalido", secuencia)
		}
		if r.FronteraAdminTecnica != nil && r.TipoRegistro != "frontera_admin_tecnica" {
			return fallar("tipo_invalido", "tipo_registro", "familia_exclusiva", "invalido", secuencia)
		}
		if (r.GobiernoUsuarios != nil || r.IntentoGobiernoUsuarios != nil) && r.TipoRegistro != "gobierno_usuarios_admin" && r.TipoRegistro != "intento_gobierno_usuarios_admin" {
			return fallar("tipo_invalido", "tipo_registro", "familia_exclusiva", "invalido", secuencia)
		}
		if r.Preservacion != nil && r.TipoRegistro != TipoOperacionPreservacionAuditoria {
			return fallar("tipo_invalido", "tipo_registro", "familia_exclusiva", "invalido", secuencia)
		}
		if r.Periodica != nil && r.TipoRegistro != TipoOperacionPeriodica {
			return fallar("tipo_invalido", "tipo_registro", "familia_exclusiva", "invalido", secuencia)
		}
		if (r.MantenimientoFijo != nil || r.IntentoMantenimientoFijo != nil) && r.TipoRegistro != "mantenimiento_perfil_fijo_admin" && r.TipoRegistro != "intento_mantenimiento_perfil_fijo_admin" {
			return fallar("tipo_invalido", "tipo_registro", "familia_exclusiva", "invalido", secuencia)
		}

		if r.IntentoBootstrapCentral != nil && r.TipoRegistro != "intento_bootstrap_central_admin" {
			return fallar("tipo_invalido", "tipo_registro", "familia_exclusiva", "invalido", secuencia)
		}

		if (r.UnidadInicial != nil || r.IntentoUnidadInicial != nil) && r.TipoRegistro != "unidad_inicial_personal" && r.TipoRegistro != "intento_unidad_inicial_personal" {
			return fallar("tipo_invalido", "tipo_registro", "familia_exclusiva", "invalido", secuencia)
		}

		if (r.FuentesIniciales != nil || r.IntentoFuentesIniciales != nil) && r.TipoRegistro != "provision_fuentes_iniciales_admin" && r.TipoRegistro != "intento_fuentes_iniciales_admin" {
			return fallar("tipo_invalido", "tipo_registro", "familia_exclusiva", "invalido", secuencia)
		}
		if (r.ConsumoOrigen != nil || r.ConsumoFecha != nil) && r.TipoRegistro != TipoConsumoOrigenV2 && r.TipoRegistro != TipoConsumoFechaV3 {
			return fallar("tipo_invalido", "tipo_registro", "consumo_exclusivo", "invalido", secuencia)
		}

		switch r.TipoRegistro {
		case "consumo_confirmado":
			if r.Consumo == nil || r.ConsumoOrigen != nil || r.ConsumoFecha != nil || r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil {
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
			if c.AnteriorSHA256 != enlace {
				return fallar("enlace_distinto", "anterior_sha256", enlace, c.AnteriorSHA256, secuencia)
			}
			if decisiones[c.DecisionRef] || consumos[c.ConsumoHuellaSHA256] {
				return fallar("consumo_duplicado", "decision_o_consumo", "unico", "duplicado", secuencia)
			}
			decisiones[c.DecisionRef], consumos[c.ConsumoHuellaSHA256] = true, true
			referencia, previo, huella = c.AuditoriaRef, c.AnteriorSHA256, huellaRegistroCadena(c)
			if c.HuellaSHA256 != huella {
				return fallar("huella_distinta", "huella_sha256", huella, c.HuellaSHA256, secuencia)
			}
			historicosSinFecha = true
		case TipoConsumoOrigenV2, TipoConsumoFechaV3, TipoConsumoTransaccionV4:
			if r.Consumo != nil || r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil {
				return fallar("tipo_invalido", "tipo_registro", "consumo_exclusivo", "invalido", secuencia)
			}
			var c RegistroCadenaV3
			var fallo *FalloVerificacion
			switch r.TipoRegistro {
			case TipoConsumoOrigenV2:
				if r.ConsumoOrigen == nil || r.ConsumoFecha != nil || r.ConsumoTransaccion != nil {
					return fallar("tipo_invalido", "tipo_registro", "consumo_exclusivo", "invalido", secuencia)
				}
				c = r.ConsumoOrigen.RegistroCadenaV3
				fallo = CotejarConsumoOrigenV2(*r.ConsumoOrigen)
				historicosSinFecha = true
			case TipoConsumoFechaV3:
				if r.ConsumoFecha == nil || r.ConsumoOrigen != nil || r.ConsumoTransaccion != nil {
					return fallar("tipo_invalido", "tipo_registro", "consumo_exclusivo", "invalido", secuencia)
				}
				c = r.ConsumoFecha.RegistroCadenaV3
				fallo = CotejarConsumoFechaV3(*r.ConsumoFecha)
				fechaLigada = true
			case TipoConsumoTransaccionV4:
				if r.ConsumoTransaccion == nil || r.ConsumoOrigen != nil || r.ConsumoFecha != nil {
					return fallar("tipo_invalido", "tipo_registro", "consumo_exclusivo", "invalido", secuencia)
				}
				c = r.ConsumoTransaccion.RegistroCadenaV3
				fallo = CotejarConsumoTransaccionV4(*r.ConsumoTransaccion)
				fechaLigada = true
			}
			if c.Secuencia != secuencia {
				return fallar("secuencia_distinta", "secuencia", strconv.FormatUint(secuencia, 10), strconv.FormatUint(c.Secuencia, 10), secuencia)
			}
			if fallo != nil {
				informe.Fallo = fallo
				return informe
			}
			if decisiones[c.DecisionRef] || consumos[c.ConsumoHuellaSHA256] {
				return fallar("consumo_duplicado", "decision_o_consumo", "unico", "duplicado", secuencia)
			}
			decisiones[c.DecisionRef], consumos[c.ConsumoHuellaSHA256] = true, true
			referencia, previo, huella = c.AuditoriaRef, c.AnteriorSHA256, c.HuellaSHA256
		case "intento_nominal":
			if r.Intento == nil || r.Consumo != nil || r.ConsumoOrigen != nil || r.ConsumoFecha != nil || r.Preperfil != nil || r.Bootstrap != nil {
				return fallar("tipo_invalido", "tipo_registro", "intento_exclusivo", "invalido", secuencia)
			}
			a := *r.Intento
			if a.Secuencia != secuencia {
				return fallar("secuencia_distinta", "secuencia", strconv.FormatUint(secuencia, 10), strconv.FormatUint(a.Secuencia, 10), secuencia)
			}
			if !huellaCadenaValida(a.AnteriorSHA256) {
				return fallar("registro_invalido", "anterior_sha256", "sha256", "no_admitido", secuencia)
			}
			if a.AnteriorSHA256 != enlace {
				return fallar("enlace_distinto", "anterior_sha256", enlace, "distinto", secuencia)
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
			if (!esquemaAdmiteFamiliasAdmin(esquema) && esquema != EsquemaVerificacionFronteraAdminTecnicaV1 && esquema != EsquemaVerificacionGobiernoUsuarios && esquema != EsquemaVerificacionPreperfil && esquema != EsquemaVerificacionFuentesIniciales && esquema != EsquemaVerificacionUnidadInicial && esquema != EsquemaVerificacionBootstrapCentral && esquema != EsquemaVerificacionMantenimientoFijo && esquema != EsquemaVerificacionPeriodica && esquema != EsquemaVerificacionPreservacionAuditoria) || r.ConsumoOrigen != nil || r.ConsumoFecha != nil {
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
			if !esquemaAdmiteFamiliasAdmin(esquema) && esquema != EsquemaVerificacionFronteraAdminTecnicaV1 && esquema != EsquemaVerificacionGobiernoUsuarios && esquema != EsquemaVerificacionFuentesIniciales && esquema != EsquemaVerificacionUnidadInicial && esquema != EsquemaVerificacionBootstrapCentral && esquema != EsquemaVerificacionMantenimientoFijo && esquema != EsquemaVerificacionPeriodica && esquema != EsquemaVerificacionPreservacionAuditoria {
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
			if !esquemaAdmiteFamiliasAdmin(esquema) && esquema != EsquemaVerificacionFronteraAdminTecnicaV1 && esquema != EsquemaVerificacionGobiernoUsuarios && esquema != EsquemaVerificacionUnidadInicial && esquema != EsquemaVerificacionBootstrapCentral && esquema != EsquemaVerificacionMantenimientoFijo && esquema != EsquemaVerificacionPeriodica && esquema != EsquemaVerificacionPreservacionAuditoria {
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
		case "intento_bootstrap_central_admin":
			if !esquemaAdmiteFamiliasAdmin(esquema) && esquema != EsquemaVerificacionFronteraAdminTecnicaV1 && esquema != EsquemaVerificacionGobiernoUsuarios && esquema != EsquemaVerificacionBootstrapCentral && esquema != EsquemaVerificacionMantenimientoFijo && esquema != EsquemaVerificacionPeriodica && esquema != EsquemaVerificacionPreservacionAuditoria {
				return fallar("tipo_invalido", "tipo_registro", "tipo_admitido", "invalido", secuencia)
			}
			evento, codigo, clave := cotejarIntentoBootstrapCentralV1(r, secuencia)
			if codigo != "" {
				return fallar(codigo, clave, "registro_ad179_valido", "no_admitido", secuencia)
			}
			if eventos[evento.EventoRef] {
				return fallar("evento_duplicado", "evento_ref", "unico", "duplicado", secuencia)
			}
			eventos[evento.EventoRef] = true
			referencia, previo, huella = evento.AuditoriaRef, evento.AnteriorSHA256, evento.HuellaSHA256
		case "contexto_admin_pre_v2":
			if !esquemaAdmiteFamiliasAdmin(esquema) {
				return fallar("tipo_invalido", "tipo_registro", "tipo_admitido", "invalido", secuencia)
			}
			e, codigo, clave := cotejarContextoAdminPreV2(r, secuencia)
			if codigo != "" {
				return fallar(codigo, clave, "registro_ad192_valido", "no_admitido", secuencia)
			}
			if eventos[e.EventoRef] {
				return fallar("evento_duplicado", "evento_ref", "unico", "duplicado", secuencia)
			}
			eventos[e.EventoRef] = true
			referencia, previo, huella = e.AuditoriaRef, e.AnteriorSHA256, e.HuellaSHA256
		case "frontera_admin_tecnica":
			if !esquemaAdmiteFamiliasAdmin(esquema) && esquema != EsquemaVerificacionFronteraAdminTecnicaV1 || r.FronteraAdminTecnica == nil || r.GobiernoUsuarios != nil || r.IntentoGobiernoUsuarios != nil || r.Preservacion != nil || r.Periodica != nil || r.MantenimientoFijo != nil || r.IntentoMantenimientoFijo != nil || r.IntentoBootstrapCentral != nil || r.UnidadInicial != nil || r.IntentoUnidadInicial != nil || r.FuentesIniciales != nil || r.IntentoFuentesIniciales != nil || r.Consumo != nil || r.ConsumoOrigen != nil || r.ConsumoFecha != nil || r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil {
				return fallar("tipo_invalido", "tipo_registro", "frontera_exclusiva", "invalido", secuencia)
			}
			evento, codigo, clave := CotejarRegistroFronteraAdminTecnicaV1(*r.FronteraAdminTecnica, secuencia)
			if codigo != "" {
				return fallar(codigo, clave, "registro_ad189_valido", "no_admitido", secuencia)
			}
			if eventos[evento.EventoRef] {
				return fallar("evento_duplicado", "evento_ref", "unico", "duplicado", secuencia)
			}
			eventos[evento.EventoRef] = true
			referencia, previo, huella = evento.AuditoriaRef, evento.AnteriorSHA256, evento.HuellaSHA256
		case "perfiles_asignables_admin", "intento_perfiles_asignables_admin":
			if esquema != EsquemaVerificacionPerfilesAsignables {
				return fallar("tipo_invalido", "tipo_registro", "tipo_admitido", "invalido", secuencia)
			}
			evento, codigo, clave := cotejarPerfilesAsignablesV1(r, secuencia)
			if codigo != "" {
				return fallar(codigo, clave, "registro_ad196_valido", "no_admitido", secuencia)
			}
			if eventos[evento.EventoRef] {
				return fallar("evento_duplicado", "evento_ref", "unico", "duplicado", secuencia)
			}
			eventos[evento.EventoRef] = true
			referencia, previo, huella = evento.AuditoriaRef, evento.AnteriorSHA256, evento.HuellaSHA256
		case "gobierno_usuarios_admin", "intento_gobierno_usuarios_admin":
			if !esquemaAdmiteFamiliasAdmin(esquema) && esquema != EsquemaVerificacionFronteraAdminTecnicaV1 && esquema != EsquemaVerificacionGobiernoUsuarios {
				return fallar("tipo_invalido", "tipo_registro", "tipo_admitido", "invalido", secuencia)
			}
			evento, codigo, clave := cotejarGobiernoUsuariosV1(r, secuencia)
			if codigo != "" {
				return fallar(codigo, clave, "registro_ad188_valido", "no_admitido", secuencia)
			}
			if eventos[evento.EventoRef] {
				return fallar("evento_duplicado", "evento_ref", "unico", "duplicado", secuencia)
			}
			eventos[evento.EventoRef] = true
			referencia, previo, huella = evento.AuditoriaRef, evento.AnteriorSHA256, evento.HuellaSHA256
		case "mantenimiento_perfil_fijo_admin", "intento_mantenimiento_perfil_fijo_admin":
			if !esquemaAdmiteFamiliasAdmin(esquema) && esquema != EsquemaVerificacionFronteraAdminTecnicaV1 && esquema != EsquemaVerificacionGobiernoUsuarios && esquema != EsquemaVerificacionMantenimientoFijo && esquema != EsquemaVerificacionPeriodica && esquema != EsquemaVerificacionPreservacionAuditoria {
				return fallar("tipo_invalido", "tipo_registro", "tipo_admitido", "invalido", secuencia)
			}
			evento, codigo, clave := cotejarMantenimientoFijoV1(r, secuencia)
			if codigo != "" {
				return fallar(codigo, clave, "registro_ad183_valido", "no_admitido", secuencia)
			}
			if eventos[evento.EventoRef] {
				return fallar("evento_duplicado", "evento_ref", "unico", "duplicado", secuencia)
			}
			eventos[evento.EventoRef] = true
			referencia, previo, huella = evento.AuditoriaRef, evento.AnteriorSHA256, evento.HuellaSHA256
		case TipoOperacionPreservacionAuditoria:
			if !esquemaAdmiteFamiliasAdmin(esquema) && esquema != EsquemaVerificacionFronteraAdminTecnicaV1 && esquema != EsquemaVerificacionGobiernoUsuarios && esquema != EsquemaVerificacionPreservacionAuditoria {
				return fallar("tipo_invalido", "tipo_registro", "tipo_admitido", "invalido", secuencia)
			}
			evento, codigo, clave := cotejarPreservacionAuditoriaV1(r, secuencia)
			if codigo != "" {
				return fallar(codigo, clave, "registro_ad187_valido", "no_admitido", secuencia)
			}
			if eventos[evento.EventoRef] {
				return fallar("evento_duplicado", "evento_ref", "unico", "duplicado", secuencia)
			}
			eventos[evento.EventoRef] = true
			referencia, previo, huella = evento.AuditoriaRef, evento.AnteriorSHA256, evento.HuellaSHA256
		case TipoOperacionPeriodica:
			if !esquemaAdmiteFamiliasAdmin(esquema) && esquema != EsquemaVerificacionFronteraAdminTecnicaV1 && esquema != EsquemaVerificacionGobiernoUsuarios && esquema != EsquemaVerificacionPeriodica && esquema != EsquemaVerificacionPreservacionAuditoria {
				return fallar("tipo_invalido", "tipo_registro", "tipo_admitido", "invalido", secuencia)
			}
			evento, codigo, clave := cotejarOperacionPeriodicaV1(r, secuencia)
			if codigo != "" {
				return fallar(codigo, clave, "registro_ad186_valido", "no_admitido", secuencia)
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
		if previo != enlace {
			return fallar("enlace_distinto", "anterior_sha256", enlace, previo, secuencia)
		}
		if r.Eslabon == nil {
			anterior, ultimaAntesDelCorte = huella, posicion
			enlaces[posicion] = anterior
			continue
		}
		fecha, presente, err := registradaEnDelAsiento(r)
		if err != nil {
			return fallar("registro_invalido", "registrada_en", "legible", "ilegible", posicion)
		}
		if presente && fecha != r.Eslabon.RegistradaEn {
			return fallar("fecha_distinta", "eslabon.registrada_en", "la_del_asiento", "distinta", posicion)
		}
		if r.Eslabon.AnteriorSHA256 != anterior {
			return fallar("enlace_distinto", "eslabon.anterior_sha256", anterior, r.Eslabon.AnteriorSHA256, posicion)
		}
		eslabon := HuellaEslabonV5("interna", posicion, anterior, secuencia, referencia, r.TipoRegistro, huella,
			r.Eslabon.RegistradaEn, r.Eslabon.SelladoEn)
		if eslabon != r.Eslabon.EslabonSHA256 {
			return fallar("eslabon_distinto", "eslabon_sha256", eslabon, r.Eslabon.EslabonSHA256, posicion)
		}
		anterior, trasCorte = eslabon, true
		enlaces[posicion] = anterior
		previa, cabeza, esCaptura, err := previaCapturaPeriodicaV5(r)
		if err != nil {
			return fallar("registro_invalido", "detalle_canonico_base64", "legible", "ilegible", posicion)
		}
		if esCaptura {
			previas = append(previas, previaDeclarada{posicion: posicion, previa: previa, cabeza: cabeza})
		}
	}
	// La captura fijó una cabeza sellada anterior a su propio asiento; si esa
	// posición está en el documento, su enlace tiene que coincidir.
	for _, p := range previas {
		if p.previa >= p.posicion {
			return fallar("captura_incoherente", "previa_secuencia", "anterior_a_su_posicion", "posterior", p.posicion)
		}
		if enlace, existe := enlaces[p.previa]; existe && enlace != p.cabeza {
			return fallar("captura_incoherente", "previa_cabeza_sha256", enlace, p.cabeza, p.posicion)
		}
	}
	if anterior != checkpoint.CabezaSHA256 {
		return fallar("cabeza_distinta", "cabeza_sha256", checkpoint.CabezaSHA256, anterior, checkpoint.UltimaSecuencia)
	}
	informe.Estado = "verificada"
	informe.MaterialIntentoRecalculado = len(intentos) > 0
	informe.ActorPerfilContextoCotejados = len(intentos) > 0
	informe.ConsumosHistoricosSinFechaLigada = historicosSinFecha
	informe.FechaConsumoLigadaCotejada = fechaLigada
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
