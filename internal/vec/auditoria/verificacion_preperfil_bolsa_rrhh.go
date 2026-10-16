package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const TipoPreperfilBolsaRRHHV1 = "preperfil_bolsa_rrhh"

// RegistroPreperfilBolsaRRHHV1 proyecta únicamente las columnas selladas por
// AD287; el contenido del justificante y la lista de perfiles no se exportan.
type RegistroPreperfilBolsaRRHHV1 struct {
	RegistroEventoAdminV3
	ActorRef string `json:"actor_ref"`
}

var recursoPreperfilBolsaRRHH = regexp.MustCompile(`\Acuenta:[0-9a-f]{64}\z`)

func cotejarPreperfilBolsaRRHHV1(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var vacio RegistroEventoAdminV3
	if r.PreperfilBolsaRRHH == nil || r.Consumo != nil || r.ConsumoOrigen != nil || r.ConsumoFecha != nil ||
		r.ConsumoTransaccion != nil || r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil ||
		r.ContextoAdminPreV2 != nil || r.FronteraAdminTecnica != nil || r.GobiernoUsuarios != nil ||
		r.IntentoGobiernoUsuarios != nil || r.Preservacion != nil || r.Periodica != nil ||
		r.MantenimientoFijo != nil || r.IntentoMantenimientoFijo != nil || r.IntentoBootstrapCentral != nil ||
		r.UnidadInicial != nil || r.IntentoUnidadInicial != nil || r.FuentesIniciales != nil ||
		r.IntentoFuentesIniciales != nil || r.ProvisionIdentidadInterna != nil ||
		r.IntentoIdentidadInterna != nil || r.PerfilesAsignables != nil ||
		r.IntentoPerfilesAsignables != nil || r.CatalogoAcciones != nil || r.IntentoCatalogoAcciones != nil {
		return vacio, "tipo_invalido", "tipo_registro"
	}
	p := r.PreperfilBolsaRRHH
	e := p.RegistroEventoAdminV3
	if e.Secuencia != secuencia {
		return e, "secuencia_distinta", "secuencia"
	}
	if len(e.EventoRef) != 39 || !strings.HasPrefix(e.EventoRef, "evento_") ||
		!hex32Valido(e.EventoRef[7:]) || e.AuditoriaRef != "aud_v3_b_"+e.EventoRef[7:] {
		return e, "referencia_distinta", "auditoria_ref"
	}
	if !actorAD171.MatchString(p.ActorRef) || !recursoPreperfilBolsaRRHH.MatchString(e.RecursoRef) ||
		!huellaCadenaValida(e.AnteriorSHA256) || !huellaCadenaValida(e.HuellaSHA256) ||
		!huellaCadenaValida(e.EventoMaterialSHA256) || !huellaCadenaValida(e.FuenteSHA256) ||
		e.FuenteRef != "identidad_sesiones_v1" || e.ModuloID != "bolsa" ||
		e.Accion != "bolsa.justificante_renuncia.perfil_propio.consultar" ||
		e.FinalidadRef != "seleccion_perfil_rrhh" || e.Proceso != "vec-server" ||
		e.Canal != "interna_corporativa" || len(e.CorrelacionRef) != 44 ||
		!strings.HasPrefix(e.CorrelacionRef, "correlacion_") || !hex32Valido(e.CorrelacionRef[12:]) ||
		!(e.Resultado == "permitido" && e.MotivoRef == "perfil_unico" ||
			e.Resultado == "denegado" && e.MotivoRef == "perfil_ausente_o_ambiguo") {
		return e, "registro_invalido", "coordenadas"
	}
	fecha, err := time.Parse(time.RFC3339Nano, e.RegistradaEn)
	if err != nil || fecha.Year() < 1 || fecha.Year() > 9999 ||
		fecha.UTC().Format("2006-01-02T15:04:05.000000Z") != e.RegistradaEn {
		return e, MotivoInstanteAD171Invalido, "registrada_en"
	}
	material := sha256.Sum256(huellaEncuadradaIntento(
		"vec.auditoria.preperfil-bolsa-rrhh.v1", e.EventoRef, p.ActorRef,
		e.RecursoRef, e.Resultado, e.MotivoRef, e.CorrelacionRef, e.FuenteSHA256))
	if hex.EncodeToString(material[:]) != e.EventoMaterialSHA256 {
		return e, "material_distinto", "evento_material_sha256"
	}
	eslabon := sha256.Sum256(huellaEncuadradaIntento(
		"vec.auditoria.eslabon.preperfil-bolsa-rrhh.v1", strconv.FormatUint(secuencia, 10),
		e.AnteriorSHA256, e.AuditoriaRef, e.EventoMaterialSHA256, e.RegistradaEn))
	if hex.EncodeToString(eslabon[:]) != e.HuellaSHA256 {
		return e, "huella_distinta", "huella_sha256"
	}
	return e, "", ""
}
