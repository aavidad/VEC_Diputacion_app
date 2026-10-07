package gobiernoreglasbaremo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestGobiernoV3ConsultaCruceDeAmbitoYRecuperacionNoEsAlta(t *testing.T) {
	s, b, r, c, p, _ := servicioGobiernoPrueba(t)
	ctx := context.Background()
	alta, err := s.GuardarAltaBorrador(ctx, c, p)
	debeSinError(t, err)
	id := p.Conjunto.Identidad()
	otra, err := reglas.NuevaIdentidadConjuntoReglasBaremo(id.Referencia(), id.Version(), id.ConvocatoriaRef(), "exp_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	debeSinError(t, err)
	motivo, err := motivoGobiernoV3(p.Motivo)
	debeSinError(t, err)
	selector := ports.SelectorGobiernoReglasV3{Identidad: otra, Estado: alta.Recibo.Estado}
	if _, err := s.ConsultarExacta(ctx, c, PeticionConsultaExactaV3{Selector: selector, Motivo: motivo}); !errors.Is(err, ports.ErrConfirmacionReglasBaremoInvalida) {
		t.Fatalf("fila de otro ámbito aceptada: %v", err)
	}
	selector.Identidad = id
	recuperacion := PeticionRecuperarReciboV3{Selector: selector, Motivo: motivo, ClaveOperacion: strings.Repeat("b", 32), HuellaSolicitudSHA256: alta.Recibo.HuellaSolicitudSHA256}
	ausente, err := s.RecuperarRecibo(ctx, c, recuperacion)
	debeSinError(t, err)
	if ausente.Existe || r.guardados != 1 || r.recuperaciones != 1 || ausente.Acceso.DecisionRef == "" {
		t.Fatal("ausencia sin auditoría o con alta")
	}
	recuperacion.ClaveOperacion = p.ClaveOperacion
	recuperacion.HuellaSolicitudSHA256 = strings.Repeat("f", 64)
	if _, err := s.RecuperarRecibo(ctx, c, recuperacion); !errors.Is(err, ports.ErrConfirmacionReglasBaremoInvalida) {
		t.Fatalf("intención de recuperación divergente: %v", err)
	}
	b.denegar = true
	consultas, recuperaciones := r.consultas, r.recuperaciones
	if _, err := s.ConsultarExacta(ctx, c, PeticionConsultaExactaV3{Selector: selector, Motivo: motivo}); !errors.Is(err, ErrGobiernoV3Prohibido) {
		t.Fatal("lectura sin concesión")
	}
	if _, err := s.RecuperarRecibo(ctx, c, recuperacion); !errors.Is(err, ErrGobiernoV3Prohibido) {
		t.Fatal("recuperación con autorización histórica")
	}
	if r.consultas != consultas || r.recuperaciones != recuperaciones || r.guardados != 1 {
		t.Fatal("denegación llegó al repositorio")
	}
}

func TestMaterialGobiernoV3RecursoIntencionYVersion(t *testing.T) {
	s, b, r, c, p, ahora := servicioGobiernoPrueba(t)
	ctx := context.Background()
	// Se pierde la primera respuesta: el cliente conserva solo clave y contenido.
	_, err := s.GuardarAltaBorrador(ctx, c, p)
	debeSinError(t, err)
	original := r.recibos[p.ClaveOperacion]
	*ahora = ahora.Add(time.Second)
	recuperada, err := s.GuardarAltaBorrador(ctx, c, p)
	debeSinError(t, err)
	alta, retry := b.solicitudes[0], b.solicitudes[1]
	if alta.Recurso.Tipo != "intencion_gobierno_reglas_baremo" || alta.Recurso.Referencia != "intencion-reglas-baremo:"+original.HuellaSolicitudSHA256 || retry.Recurso.Tipo != alta.Recurso.Tipo || retry.Recurso.Referencia != alta.Recurso.Referencia {
		t.Fatal("alta no autoriza la intención estable")
	}
	if !recuperada.Replay || !recuperada.Recibo.ConfirmadaEn.Equal(original.ConfirmadaEn) || !bytes.Equal(recuperada.Recibo.VersionCanonica, original.VersionCanonica) || alta.Recurso.Atributos["material_sha256"] == retry.Recurso.Atributos["material_sha256"] {
		t.Fatal("respuesta perdida no conserva original y material fresco")
	}
	selector := ports.SelectorGobiernoReglasV3{Identidad: p.Conjunto.Identidad(), Estado: original.Estado}
	motivo, _ := motivoGobiernoV3(p.Motivo)
	_, err = s.ConsultarExacta(ctx, c, PeticionConsultaExactaV3{Selector: selector, Motivo: motivo})
	debeSinError(t, err)
	consulta := b.solicitudes[2]
	if consulta.Recurso.Tipo != "version_reglas_baremo_gobernada" || consulta.Recurso.Referencia != "reglas-baremo:"+original.Estado.HuellaEstadoSHA256() {
		t.Fatal("lectura dejó de seleccionar la versión exacta")
	}
	_, err = s.RecuperarRecibo(ctx, c, PeticionRecuperarReciboV3{Selector: selector, Motivo: motivo,
		ClaveOperacion: p.ClaveOperacion, HuellaSolicitudSHA256: original.HuellaSolicitudSHA256})
	debeSinError(t, err)
	recibo := b.solicitudes[3]
	if consulta.Accion != "bolsa.reglas_baremo.version.consultar" || !slices.Equal(consulta.Campos, []string{"estado_reglas_baremo"}) ||
		recibo.Accion != "bolsa.reglas_baremo.recibo.consultar" || !slices.Equal(recibo.Campos, []string{"estado_reglas_baremo", "recibo"}) ||
		recibo.Recurso.Referencia != consulta.Recurso.Referencia || recibo.Finalidad != consulta.Finalidad {
		t.Fatal("consulta y recuperación no conservaron capacidades nominales separadas")
	}
}

func TestMaterialGobiernoV3ContratoJSONAlta(t *testing.T) {
	s, b, _, c, p, _ := servicioGobiernoPrueba(t)
	_, err := s.GuardarAltaBorrador(context.Background(), c, p)
	debeSinError(t, err)
	solicitud := b.solicitudes[0]
	esperado := []byte(`{"esquema":"vec.bolsa.gobierno-borrador.material.v3","operacion":"alta_borrador","accion":"bolsa.reglas_baremo.borrador.crear","modulo_id":"bolsa","tipo_recurso":"intencion_gobierno_reglas_baremo","finalidad":"gobierno_reglas_baremo",`)
	if !bytes.HasPrefix(solicitud.MaterialCanonico, esperado) || solicitud.Recurso.Atributos["material_sha256"] != shaGobiernoV3(solicitud.MaterialCanonico) {
		t.Fatal("JSON y recurso discrepan del contrato nominal")
	}
	var material MaterialGobiernoV3
	debeSinError(t, json.Unmarshal(solicitud.MaterialCanonico, &material))
	if material.TipoRecurso != solicitud.Recurso.Tipo || material.EstadoEsperado != nil || material.HuellaSolicitudSHA256 == "" || material.SolicitadaEn == "" || len(material.VersionCanonica) == 0 {
		t.Fatal("proyección JSON incompleta")
	}
}
