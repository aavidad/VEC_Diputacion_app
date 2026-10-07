package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func vectorGobiernoUsuariosPrueba(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	b := RegistroOperacionMantenimientoV1{AuditoriaRef: "aud_v3_gu_" + strings.Repeat("1", 32), Secuencia: 1, AnteriorSHA256: strings.Repeat("0", 64), RegistradaEn: "2026-10-04T08:30:00.000000Z", EventoRef: "evento_" + strings.Repeat("1", 32), OperadorLogin: "vec_admin_gobierno_ensayo", Accion: "aprovisionar_gobierno_usuarios_admin_v1", ModuloID: "administracion", Proceso: "postgresql", Canal: "operacion_tecnica_privada", FinalidadRef: "gobierno_usuarios_admin", CorrelacionRef: "correlacion_" + strings.Repeat("2", 32), Resultado: "permitido", MotivoRef: "gobierno_usuarios_registrado", RecursoRef: "gobierno_usuarios:" + strings.Repeat("b", 32)}
	m := RegistroGobiernoUsuariosV1{RegistroOperacionMantenimientoV1: b, PlanSHA256: strings.Repeat("b", 64), PreimagenSHA256: strings.Repeat("c", 64), ConfiguracionOrigenRef: "confianza:atestacion:ct:desarrollo:2026-10-02", ConfiguracionDestinoRef: "confianza:atestacion:ct:desarrollo:2026-10-04", ClavesSHA256: strings.Repeat("d", 64)}
	fields := []string{"vec.auditoria.gobierno-usuarios-admin.v1", "gobierno_usuarios_admin", b.EventoRef, b.OperadorLogin, m.PlanSHA256, m.PreimagenSHA256, m.ConfiguracionOrigenRef, m.ConfiguracionDestinoRef, m.ClavesSHA256, b.Proceso, b.Canal, b.FinalidadRef, b.CorrelacionRef}
	h := sha256.Sum256(huellaEncuadradaIntento(fields...))
	m.EventoMaterialSHA256 = hex.EncodeToString(h[:])
	h = sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.eslabon.gobierno-usuarios-admin.v1", "1", b.AnteriorSHA256, b.AuditoriaRef, m.EventoMaterialSHA256, b.RegistradaEn))
	m.HuellaSHA256 = hex.EncodeToString(h[:])
	return DocumentoVerificacionMixta{Esquema: EsquemaVerificacionGobiernoUsuarios, Manifiesto: CoberturaCadena{CadenaID: "cadena:comun:interna", PrimeraSecuencia: 1, UltimaSecuencia: 1, AnteriorSHA256: b.AnteriorSHA256, CabezaSHA256: m.HuellaSHA256, Registros: 1}, Registros: []RegistroMixtoV2{{TipoRegistro: "gobierno_usuarios_admin", GobiernoUsuarios: &m}}}
}
func TestGobiernoUsuariosVerificaMaterialYRechazaCambios(t *testing.T) {
	for _, caso := range []string{"valido", "hash", "configuracion", "fecha", "cruce"} {
		t.Run(caso, func(t *testing.T) {
			d := vectorGobiernoUsuariosPrueba(t)
			switch caso {
			case "hash":
				d.Registros[0].GobiernoUsuarios.ClavesSHA256 = strings.Repeat("e", 64)
			case "configuracion":
				d.Registros[0].GobiernoUsuarios.ConfiguracionDestinoRef = "otra:configuracion"
			case "fecha":
				d.Registros[0].GobiernoUsuarios.RegistradaEn = "contenido_privado_no_fecha"
			case "cruce":
				d.Registros[0].MantenimientoFijo = &RegistroMantenimientoFijoV1{}
			}
			r := VerificarCadenaGobiernoUsuariosV1(d, d.Manifiesto, 2)
			raw, _ := json.Marshal(r)
			if caso == "valido" {
				if r.Estado != "verificada" || r.ActorPerfilContextoCotejados {
					t.Fatalf("estado=%s fallo=%+v", r.Estado, r.Fallo)
				}
			} else if r.Estado != "rechazada" || strings.Contains(string(raw), "contenido_privado") {
				t.Fatal("alteracion aceptada o exportada")
			}
		})
	}
}
func TestGobiernoUsuariosConservaFamiliasPrevias(t *testing.T) {
	ds := []DocumentoVerificacionMixta{vectorConsumosMixtosAD173(t), documentoFuentesPrueba(t), documentoUnidadInicialPrueba(t), documentoBootstrapIntentosPrueba(t), vectorMantenimientoFijo(t)}
	for _, d := range ds {
		before, _ := json.Marshal(d.Registros)
		d.Esquema = EsquemaVerificacionGobiernoUsuarios
		r := VerificarCadenaGobiernoUsuariosV1(d, d.Manifiesto, uint64(len(d.Registros)))
		after, _ := json.Marshal(d.Registros)
		if r.Estado != "verificada" || !strings.EqualFold(string(before), string(after)) {
			t.Fatalf("familia previa no conservada: %+v", r.Fallo)
		}
	}
}
