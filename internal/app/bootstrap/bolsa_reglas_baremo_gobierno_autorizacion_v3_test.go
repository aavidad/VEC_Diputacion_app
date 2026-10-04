package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	reglasapp "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	bolsapuertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func pedidoGobiernoReglasV3Prueba(t *testing.T, p *PerfilGobiernoReglasBaremoV3, operacion string, ahora time.Time) bolsapuertos.SolicitudMaterialGobiernoReglasV3 {
	t.Helper()
	accion, finalidad, tipo, ref := "bolsa.reglas_baremo.version.consultar", "consulta_gobierno_reglas_baremo", "version_reglas_baremo_gobernada", "reglas-baremo:"+strings.Repeat("a", 64)
	campos := []string{"estado_reglas_baremo"}
	version := []byte(nil)
	huellaSolicitud := ""
	claveOperacion := ""
	if operacion == "alta_borrador" {
		accion, finalidad, tipo, ref = "bolsa.reglas_baremo.borrador.crear", "gobierno_reglas_baremo", "intencion_gobierno_reglas_baremo", "intencion-reglas-baremo:"+strings.Repeat("b", 64)
		campos, version, huellaSolicitud = []string{"auditoria", "estado_reglas_baremo", "salida_eventos"}, []byte("version-canonica-prueba"), strings.Repeat("b", 64)
		claveOperacion = strings.Repeat("a", 32)
	} else if operacion == "recuperar_recibo" {
		accion = "bolsa.reglas_baremo.recibo.consultar"
		campos = []string{"estado_reglas_baremo", "recibo"}
		claveOperacion, huellaSolicitud = strings.Repeat("a", 32), strings.Repeat("b", 64)
	}
	motivo := motivoCatalogoPlantillasCTDesarrollo()
	motivoCanon, err := vecdomain.RepresentacionCanonicaMotivoAutorizacionV2(motivo)
	if err != nil {
		t.Fatal(err)
	}
	actor := p.soporte.contexto.Resultado.Contexto
	material := reglasapp.MaterialGobiernoV3{
		Esquema: "vec.bolsa.gobierno-borrador.material.v3", Operacion: operacion,
		Accion: accion, ModuloID: "bolsa", TipoRecurso: tipo, Finalidad: finalidad,
		PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef,
		ConvocatoriaRef: p.convocatoriaRef, ExpedienteRef: p.expedienteRef,
		Estado: reglasapp.EstadoMaterialGobiernoV3{Referencia: "reglas:prueba", Version: 1,
			HuellaContenidoSHA256: strings.Repeat("c", 64), Revision: 1, HuellaEstadoSHA256: strings.Repeat("a", 64)},
		VersionCanonica: version, ClaveOperacion: claveOperacion, HuellaSolicitudSHA256: huellaSolicitud,
		MotivoCanonico: motivoCanon, SolicitadaEn: ahora.Format("2006-01-02T15:04:05.000000Z"),
	}
	canon, err := json.Marshal(material)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(canon)
	return bolsapuertos.SolicitudMaterialGobiernoReglasV3{
		Operacion: operacion, Accion: accion, Finalidad: finalidad, Campos: campos,
		Audiencia: reglasapp.AudienciaGobiernoBorradorReglasV3, Motivo: motivo, MaterialCanonico: canon,
		Recurso: vecdomain.RecursoAutorizable{Referencia: ref, ModuloID: "bolsa", Tipo: tipo,
			Ambitos:   map[string]string{"convocatoria_ref": p.convocatoriaRef, "expediente_ref": p.expedienteRef},
			Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}},
	}
}

// Evalúa el perfil con el motor V3 real: no basta que el pedido declare
// campos mínimos si la decisión del PDP sigue reuniendo dos lecturas.
func TestGobiernoReglasBaremoPDPEmiteCamposNominales(t *testing.T) {
	p := perfilGobiernoReglasBaremoPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	for _, op := range []string{"alta_borrador", "consultar_exacta", "recuperar_recibo"} {
		t.Run(op, func(t *testing.T) {
			pedido := pedidoGobiernoReglasV3Prueba(t, p, op, ahora)
			correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridad.GeneradorReferenciasCriptograficas{})
			if err != nil {
				t.Fatal(err)
			}
			solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
				VinculoAutenticacionActor: p.soporte.contexto.Vinculo, ReferenciaMotivo: pedido.Motivo,
				Accion: pedido.Accion, Recurso: pedido.Recurso, Finalidad: pedido.Finalidad, Correlacion: correlacion,
			})
			if err != nil {
				t.Fatal(err)
			}
			evidencia, err := vecdomain.NuevaEvidenciaEvaluacionAutorizacionV3(solicitud, p.plantilla, "decision:baremo-prueba", ahora, ahora.Add(time.Second))
			if err != nil {
				t.Fatal(err)
			}
			decision, err := vecdomain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
			if err != nil || decision.ExigirProyeccionPara(solicitud, pedido.Campos, nil) != nil {
				t.Fatalf("PDP no emitió mínimo exacto para %s: %v", op, err)
			}
			for _, campos := range [][]string{{"estado_reglas_baremo", "recibo"}, {"recibo"}, {"estado_reglas_baremo"}} {
				if !slices.Equal(campos, pedido.Campos) && decision.ExigirProyeccionPara(solicitud, campos, nil) == nil {
					t.Fatal("decisión admitió cruce o unión de campos")
				}
			}
		})
	}
}

func TestBrokerGobiernoReglasBaremoCamposYAlcancePorOperacion(t *testing.T) {
	p := perfilGobiernoReglasBaremoPrueba(t)
	b := &ProveedorGobiernoReglasBaremoV3{perfil: p}
	actor := p.soporte.contexto.Resultado.Contexto
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	for _, op := range []string{"alta_borrador", "consultar_exacta", "recuperar_recibo"} {
		s := pedidoGobiernoReglasV3Prueba(t, p, op, ahora)
		if b.validarPedido(s, actor, ahora) != nil {
			t.Fatalf("pedido legítimo rechazado: %s", op)
		}
		s.Campos = []string{"estado_reglas_baremo", "recibo"}
		if op == "recuperar_recibo" {
			s.Campos = []string{"recibo"}
		}
		if b.validarPedido(s, actor, ahora) == nil {
			t.Fatalf("campos ajenos al contrato admitidos: %s", op)
		}
		if op != "alta_borrador" {
			s = pedidoGobiernoReglasV3Prueba(t, p, op, ahora)
			s.Campos = []string{"estado_reglas_baremo", "recibo"}
			s.Accion = "bolsa.reglas_baremo.recibo.consultar"
			if op == "recuperar_recibo" {
				s.Campos = []string{"estado_reglas_baremo"}
				s.Accion = "bolsa.reglas_baremo.version.consultar"
			}
			var material reglasapp.MaterialGobiernoV3
			if err := json.Unmarshal(s.MaterialCanonico, &material); err != nil {
				t.Fatal(err)
			}
			material.Accion = s.Accion
			var err error
			s.MaterialCanonico, err = json.Marshal(material)
			if err != nil {
				t.Fatal(err)
			}
			h := sha256.Sum256(s.MaterialCanonico)
			s.Recurso.Atributos["material_sha256"] = hex.EncodeToString(h[:])
			if b.validarPedido(s, actor, ahora) == nil {
				t.Fatalf("acción y campos de otra lectura admitidos: %s", op)
			}
		}
	}
	s := pedidoGobiernoReglasV3Prueba(t, p, "alta_borrador", ahora)
	s.Recurso.Ambitos["expediente_ref"] = "expediente:ajeno"
	if b.validarPedido(s, actor, ahora) == nil {
		t.Fatal("ámbito ajeno admitido")
	}
	s = pedidoGobiernoReglasV3Prueba(t, p, "consultar_exacta", ahora)
	s.Audiencia = "vec_bolsa_llamamientos.borrador.v3"
	if b.validarPedido(s, actor, ahora) == nil {
		t.Fatal("audiencia de llamamientos admitida")
	}
	s = pedidoGobiernoReglasV3Prueba(t, p, "recuperar_recibo", ahora)
	s.MaterialCanonico = append([]byte(" "), s.MaterialCanonico...)
	if b.validarPedido(s, actor, ahora) == nil {
		t.Fatal("material no canónico admitido")
	}
}

func TestBrokerGobiernoReglasBaremoDeniegaSinCanalNiDependencias(t *testing.T) {
	if _, err := NuevoProveedorGobiernoReglasBaremoV3(nil, nil, nil, nil, nil,
		RutasGobiernoReglasBaremoV3{}, relojContratacionTemporalDesarrollo{}); err == nil {
		t.Fatal("broker sin fuentes confiables admitido")
	}
	r := RutasGobiernoReglasBaremoV3{"/reglas/alta", "/reglas/consulta", "/reglas/recuperar"}
	if !r.valida() || (RutasGobiernoReglasBaremoV3{"/reglas/alta", "/reglas/alta", "/reglas/recuperar"}).valida() {
		t.Fatal("rutas exactas no distinguen operaciones")
	}
	var b ProveedorGobiernoReglasBaremoV3
	if b.rutaOperacion(r.Alta, "alta_borrador") {
		t.Fatal("broker cero atendió una ruta")
	}
}
